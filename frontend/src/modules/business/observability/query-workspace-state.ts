import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useAuthStore } from '@duanxldragon/pantheon-base-ui';
import type { QueryCatalog, QueryDuration, QueryHistoryRow, QueryRecipe, QuerySourceKind, QueryTime, SavedViewRow } from './api';

const durations: QueryDuration[] = ['5m', '15m', '1h', '6h', '24h'];
const fields: Record<QuerySourceKind, { field: string; op: string; values?: string[] }[]> = {
  prometheus: [{ field: 'availability', op: 'instant' }],
  victoriametrics: [{ field: 'availability', op: 'instant' }],
  loki: [{ field: 'stream', op: 'exists' }, { field: 'severity', op: 'equals', values: ['info', 'warning', 'error'] }],
  tempo: [{ field: 'service', op: 'exists' }],
};

export function validQueryTime(value: QueryTime): boolean {
  if (!value || typeof value !== 'object') return false;
  if (value.mode === 'relative') return durations.includes(value.duration) && Object.keys(value).every((key) => key === 'mode' || key === 'duration');
  if (value.mode !== 'absolute' || typeof value.start !== 'string' || typeof value.end !== 'string' || !/Z$/.test(value.start) || !/Z$/.test(value.end)) return false;
  const start = Date.parse(value.start), end = Date.parse(value.end);
  return Number.isFinite(start) && Number.isFinite(end) && end > start && end - start <= 86400000 && Object.keys(value).every((key) => ['mode', 'start', 'end'].includes(key));
}

// This is a defensive DTO whitelist, not a query compiler or an authorization oracle.
export function copyQueryRecipe(value: QueryRecipe, kind: QuerySourceKind): QueryRecipe | null {
  if (!value || value.version !== 1 || value.sourceKind !== kind || !Object.hasOwn(fields, kind) || value.resultMode !== 'table' || !validQueryTime(value.time)) return null;
  const max = kind === 'loki' ? 5000 : 1000;
  if (!Number.isInteger(value.limit) || value.limit < 1 || value.limit > max || !Array.isArray(value.rules) || value.rules.length < 1 || value.rules.length > 16) return null;
  if (Object.keys(value).some((key) => !['version', 'sourceKind', 'rules', 'time', 'limit', 'resultMode'].includes(key))) return null;
  const rules: QueryRecipe['rules'] = [];
  for (const rule of value.rules) {
    if (!rule || Object.keys(rule).some((key) => !['field', 'op', 'value'].includes(key))) return null;
    const field = fields[kind].find((item) => item.field === rule.field && item.op === rule.op);
    if (!field || (field.values ? !field.values.includes(rule.value ?? '') : rule.value !== undefined && rule.value !== '')) return null;
    if (rules.some((item) => item.field === rule.field)) return null;
    rules.push({ field: field.field, op: field.op, ...(field.values ? { value: rule.value } : {}) });
  }
  if (kind !== 'loki' && rules.length !== 1) return null;
  if (kind === 'loki' && (rules.length > 2 || !rules.some((rule) => rule.field === 'stream'))) return null;
  return { version: 1, sourceKind: kind, rules, time: { ...value.time }, limit: value.limit, resultMode: 'table' };
}

export function queryTimeBounds(time: QueryTime): { start: string; end: string } {
  if (time.mode === 'absolute') return { start: time.start, end: time.end };
  const end = new Date();
  const minutes = { '5m': 5, '15m': 15, '1h': 60, '6h': 360, '24h': 1440 }[time.duration];
  return { start: new Date(end.getTime() - minutes * 60000).toISOString(), end: end.toISOString() };
}

export function utcInputTime(value: string): string {
  const date = new Date(value + 'Z');
  return Number.isFinite(date.getTime()) ? date.toISOString() : '';
}

// Lists may span source kinds. Keep only bounded metadata; restoration validates
// the recipe against the currently authorized datasource list separately.
export function copyWorkspaceRow(value: QueryHistoryRow | SavedViewRow, tab: 'history' | 'views'): QueryHistoryRow | SavedViewRow | null {
  if (!value || !Number.isSafeInteger(value.id) || value.id < 1 || !Number.isSafeInteger(value.datasourceId) || value.datasourceId < 1) return null;
  if (tab === 'views') {
    if (!('revision' in value) || !Number.isSafeInteger(value.revision) || value.revision < 1 || typeof value.updatedAt !== 'string' || value.updatedAt.length > 64) return null;
    const recipe = value.recipe && copyQueryRecipe(value.recipe, value.recipe.sourceKind);
    return recipe ? { id: value.id, datasourceId: value.datasourceId, revision: value.revision, updatedAt: value.updatedAt, recipe } : null;
  }
  if (!('sourceKind' in value) || !Object.hasOwn(fields, value.sourceKind) || !validQueryTime(value.time) || !Number.isInteger(value.limit) || value.limit < 1 || value.limit > (value.sourceKind === 'loki' ? 5000 : 1000)
    || value.resultMode !== 'table' || typeof value.createdAt !== 'string' || value.createdAt.length > 64 || typeof value.expressionUnavailable !== 'boolean'
    || typeof value.outcome !== 'string' || value.outcome.length > 64 || !Number.isFinite(value.durationMs) || value.durationMs < 0) return null;
  const recipe = value.recipe ? copyQueryRecipe(value.recipe, value.sourceKind) : null;
  if ((value.recipe && !recipe) || (!recipe && !value.expressionUnavailable) || (recipe && value.expressionUnavailable)) return null;
  return { id: value.id, datasourceId: value.datasourceId, sourceKind: value.sourceKind, time: { ...value.time }, limit: value.limit, resultMode: 'table', outcome: value.outcome, durationMs: value.durationMs, createdAt: value.createdAt, expressionUnavailable: value.expressionUnavailable, ...(recipe ? { recipe } : {}) };
}

export function supportedQueryCatalog(catalog: QueryCatalog, kind: QuerySourceKind): boolean {
  return Boolean(catalog && catalog.version === 1 && catalog.sourceKind === kind && fields[kind] && Array.isArray(catalog.fields)
    && catalog.fields.length === fields[kind].length && fields[kind].every((item) => catalog.fields.some((candidate) => candidate.field === item.field
      && Array.isArray(candidate.ops) && candidate.ops.length === 1 && candidate.ops[0] === item.op
      && (item.values ? candidate.values?.length === item.values.length && item.values.every((value) => candidate.values?.includes(value)) : !candidate.values?.length)))
    && catalog.maxLimit === (kind === 'loki' ? 5000 : 1000) && catalog.resultModes?.length === 1 && catalog.resultModes[0] === 'table'
    && catalog.durations?.length === durations.length && durations.every((duration) => catalog.durations.includes(duration)));
}

export function initialQueryRecipe(kind: QuerySourceKind, limit = kind === 'loki' ? 500 : 100): QueryRecipe {
  return { version: 1, sourceKind: kind, rules: [{ field: fields[kind][0].field, op: fields[kind][0].op }], time: { mode: 'relative', duration: '15m' }, limit, resultMode: 'table' };
}

// Auth updates, including a repeated cookie-session marker, invalidate in-memory
// work. Tenant switching is a Base-owned full shell reload; pagehide also aborts.
export function useQueryWorkspaceIdentity(onInvalidate: () => void) {
  const [epoch, setEpoch] = useState(0);
  const current = useRef(0);
  const invalidateRef = useRef(onInvalidate);
  useLayoutEffect(() => { invalidateRef.current = onInvalidate; }, [onInvalidate]);
  useEffect(() => {
    const invalidate = () => { current.current += 1; invalidateRef.current(); setEpoch(current.current); };
    const unsubscribe = useAuthStore.subscribe(invalidate);
    window.addEventListener('pagehide', invalidate);
    const storage = (event: StorageEvent) => {
      if (event.key === null || ['pantheon_session_hint', 'pantheon_csrf_token', 'pantheon_auth_user', 'pantheon_tenant_viewpoint'].includes(event.key)) invalidate();
    };
    window.addEventListener('storage', storage);
    return () => { unsubscribe(); window.removeEventListener('pagehide', invalidate); window.removeEventListener('storage', storage); current.current += 1; };
  }, []);
  const isCurrent = useCallback((captured: number) => captured === current.current, []);
  const snapshot = useCallback(() => current.current, []);
  return { epoch, snapshot, isCurrent };
}
