import { useEffect, useMemo, useRef, useState } from 'react';
import { isForbiddenRequestError } from '../../../api/request';
import { getHostList, type HostRow } from '../cmdb/host/api';
import { resolveDeployTargets, type DeployTargetResolution, type DeployTargetSelection } from '../deploy/api';
import type { ContextSelectorOption } from '../../../components/operational/types';

// Shared business target-selection adapter. Reuses the existing CMDB host
// provider and the deploy resolve capability; it does not read across modules.
// Scope change invalidates the intent and aborts in-flight work intrinsically,
// so a consumer never must remember to key/remount by scope to avoid ghosts.

/** Default page size for the bounded CMDB candidate lookup. */
export const TARGET_CANDIDATE_PAGE_SIZE = 25;
/** Server-enforced explicit-ID budget; mirrors backend TargetSelectionBudget. */
export const TARGET_SELECTION_BUDGET = 10000;

/** Normalize manual input lines into a deduped, trimmed array. */
export function parseManualLines(text: string): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const raw of text.split(/\n/)) {
    const value = raw.trim();
    if (value && !seen.has(value)) {
      seen.add(value);
      out.push(value);
    }
  }
  return out;
}

/** An empty intent targeting `scopeId` (no host/group/manual/exclusion). */
export function emptySelection(scopeId: number): DeployTargetSelection {
  return { businessScopeId: scopeId, hostIds: [], groupIds: [], includeScope: false, manual: [], excludedIds: [] };
}

/** True when the explicit-ID intent stays within the bounded budget. */
export function withinSelectionBudget(selection: DeployTargetSelection, budget = TARGET_SELECTION_BUDGET): boolean {
  return selection.hostIds.length + selection.groupIds.length + selection.manual.length + selection.excludedIds.length <= budget;
}

/** Map a scoped CMDB host into a selector candidate option. */
export function hostToTargetOption(host: HostRow): ContextSelectorOption {
  return { id: String(host.id), source: 'host', label: `${host.hostname} · ${host.ip}`, disabled: !['assigned', 'online'].includes(host.status) };
}

/** Classify a lookup/resolve failure into a displayable state. */
export function classifyTargetFailure(error: unknown): 'error' | 'forbidden' {
  return isForbiddenRequestError(error) ? 'forbidden' : 'error';
}

/** Roll up a resolution into bounded summary counts for feedback/audit (no secrets). */
export interface TargetSelectionSummary {
  valid: number;
  duplicate: number;
  unavailable: number;
  ineligible: number;
  excluded: number;
  sources: Record<string, number>;
}
export function summarizeSelection(result: DeployTargetResolution): TargetSelectionSummary {
  return {
    valid: result.hosts.length,
    duplicate: result.duplicateCount,
    unavailable: result.unavailableCount,
    ineligible: result.ineligibleCount,
    excluded: result.excludedCount,
    sources: result.sources ?? {},
  };
}

export interface UseBusinessTargetSelectorArgs {
  scopeId: number;
  groups?: Array<{ id: number; label: string }>;
  initial?: DeployTargetSelection;
  initialHostIds?: number[];
  onChange: (resolution: DeployTargetResolution | null) => void;
}

export interface UseBusinessTargetSelector {
  intent: DeployTargetSelection;
  hosts: HostRow[];
  total: number;
  page: number;
  keyword: string;
  loading: boolean;
  lookupFailed: 'error' | 'forbidden' | null;
  resolving: boolean;
  resolution: DeployTargetResolution | null;
  failure: 'error' | 'forbidden' | null;
  onSearch: (keyword: string) => void;
  onPage: (page: number) => void;
  reload: () => void;
  /** Update the intent; invalidates the prior resolution and lets consumers drop it. */
  change: (next: DeployTargetSelection) => void;
  resolve: () => void;
  options: ContextSelectorOption[];
  selected: ContextSelectorOption[];
  budget: number;
  summary: TargetSelectionSummary | null;
}

/**
 * Shared composer for a business target selector: bounded CMDB host lookup +
 * provider revalidation of the intent. On scopeId change the prior intent is
 * cleared and any in-flight resolve/search is aborted, so stale or out-of-scope
 * (ghost) targets cannot survive a context switch without relying on a remount.
 */
export function useBusinessTargetSelector({ scopeId, initial, initialHostIds, onChange }: UseBusinessTargetSelectorArgs): UseBusinessTargetSelector {
  const [intent, setIntent] = useState<DeployTargetSelection>(() => {
    const inScope = initial?.businessScopeId === scopeId ? initial : undefined;
    // Seed the explicit-ID intent from initialHostIds (caller already scopes them to
    // this scope); a cross-domain initial is dropped, never carried into the scope.
    const hostIds = [...new Set([...(inScope?.hostIds ?? []), ...(initialHostIds ?? [])])];
    return inScope ? { ...inScope, hostIds } : { ...emptySelection(scopeId), hostIds };
  });
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPageState] = useState(1);
  const [keyword, setKeywordState] = useState('');
  const [loading, setLoading] = useState(true);
  const [lookupFailed, setLookupFailed] = useState<'error' | 'forbidden' | null>(null);
  const [retry, setRetry] = useState(0);
  const [resolving, setResolving] = useState(false);
  const [resolution, setResolution] = useState<DeployTargetResolution | null>(null);
  const [failure, setFailure] = useState<'error' | 'forbidden' | null>(null);
  const generation = useRef(0);
  const resolver = useRef<AbortController | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(() => {
      setLoading(true);
      setLookupFailed(null);
      void getHostList({ page, pageSize: TARGET_CANDIDATE_PAGE_SIZE, keyword, businessScopeId: scopeId }, controller.signal)
        .then((response) => {
          if (controller.signal.aborted) return;
          setHosts(response.items);
          setTotal(response.total);
          setLoading(false);
        })
        .catch((error: unknown) => {
          if (!controller.signal.aborted) {
            setHosts([]);
            setLookupFailed(classifyTargetFailure(error));
            setLoading(false);
          }
        });
    }, 250);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [scopeId, page, keyword, retry]);

  useEffect(() => () => { generation.current += 1; resolver.current?.abort(); }, []);

  // Invalidate the intent when the scope changes without relying on a consumer
  // remount: clear the prior (possibly out-of-scope) selection and abort work.
  const lastScope = useRef(scopeId);
  const change = (next: DeployTargetSelection) => {
    generation.current += 1;
    resolver.current?.abort();
    setIntent(next);
    setResolution(null);
    setFailure(null);
    setResolving(false);
    onChange(null);
  };
  useEffect(() => {
    if (lastScope.current !== scopeId) {
      lastScope.current = scopeId;
      change(emptySelection(scopeId));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- change() must run with the new scope; intent is cleared intentionally.
  }, [scopeId]);

  const onSearch = (value: string) => {
    setHosts([]);
    setLoading(true);
    setKeywordState(value);
    setPageState(1);
  };
  const onPage = (value: number) => {
    setHosts([]);
    setLoading(true);
    setPageState(value);
  };
  const reload = () => setRetry((value) => value + 1);

  const resolve = async () => {
    const current = ++generation.current;
    resolver.current?.abort();
    const controller = new AbortController();
    resolver.current = controller;
    setResolving(true);
    setResolution(null);
    setFailure(null);
    onChange(null);
    try {
      const result = await resolveDeployTargets({ ...intent, businessScopeId: scopeId }, controller.signal);
      if (current !== generation.current || controller.signal.aborted) return;
      setResolution(result);
      onChange(result);
    } catch (error) {
      if (current === generation.current && !controller.signal.aborted) setFailure(classifyTargetFailure(error));
    } finally {
      if (current === generation.current) setResolving(false);
    }
  };

  const options = useMemo(() => hosts.map(hostToTargetOption), [hosts]);
  const selected = useMemo(
    () => intent.hostIds.map((id) => options.find((option) => option.id === String(id)) || { id: String(id), source: 'host', label: `#${id}` }),
    [intent.hostIds, options],
  );
  const budget = intent.hostIds.length + intent.groupIds.length + intent.manual.length + intent.excludedIds.length;
  const summary = resolution ? summarizeSelection(resolution) : null;

  return { intent, hosts, total, page, keyword, loading, lookupFailed, resolving, resolution, failure, onSearch, onPage, reload, change, resolve, options, selected, budget, summary };
}