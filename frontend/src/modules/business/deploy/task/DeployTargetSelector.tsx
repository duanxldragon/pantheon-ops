import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Checkbox, Input, Pagination, Select, Space, Typography } from '@arco-design/web-react';
import { ContextSelector, type ContextSelectorOption } from '../../../../components';
import { isForbiddenRequestError } from '../../../../api/request';
import { getHostList, type HostRow } from '../../cmdb/host/api';
import { resolveDeployTargets, type DeployTargetResolution, type DeployTargetSelection } from '../api';
interface Props {
  scopeId: number;
  groups: Array<{ id: number; label: string }>;
  initial?: DeployTargetSelection;
  initialHostIds?: number[];
  onChange: (resolution: DeployTargetResolution | null) => void;
}
export default function DeployTargetSelector({ scopeId, groups, initial, initialHostIds, onChange }: Readonly<Props>) {
  const { t } = useTranslation();
  const [intent, setIntent] = useState<DeployTargetSelection>(() => (initial?.businessScopeId === scopeId ? initial : undefined) || { businessScopeId: scopeId, hostIds: initialHostIds || [], groupIds: [], includeScope: false, manual: [], excludedIds: [] });
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState('');
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
      void getHostList({ page, pageSize: 25, keyword, businessScopeId: scopeId }, controller.signal).then((response) => {
        if (controller.signal.aborted) return;
        setHosts(response.items); setTotal(response.total); setLoading(false);
      }).catch((error: unknown) => { if (!controller.signal.aborted) { setHosts([]); setLookupFailed(isForbiddenRequestError(error) ? 'forbidden' : 'error'); setLoading(false); } });
    }, 250);
    return () => { clearTimeout(timer); controller.abort(); };
  }, [scopeId, page, keyword, retry]);
  useEffect(() => () => { generation.current += 1; resolver.current?.abort(); }, []);
  const change = (next: DeployTargetSelection) => {
    generation.current += 1; resolver.current?.abort();
    setIntent(next); setResolution(null); setFailure(null); setResolving(false); onChange(null);
  };
  const options = useMemo(() => hosts.map((host) => ({ id: String(host.id), source: 'host', label: `${host.hostname} · ${host.ip}`, disabled: !['assigned', 'online'].includes(host.status) })), [hosts]);
  const selected = intent.hostIds.map((id) => options.find((option) => option.id === String(id)) || { id: String(id), source: 'host', label: `#${id}` });
  const toggle = (option: ContextSelectorOption, remove: boolean) => change({ ...intent, hostIds: remove ? intent.hostIds.filter((id) => String(id) !== option.id) : [...intent.hostIds, Number(option.id)] });
  const resolve = async () => {
    const current = ++generation.current;
    resolver.current?.abort(); const controller = new AbortController(); resolver.current = controller;
    setResolving(true); setFailure(null); setResolution(null); onChange(null);
    try {
      const result = await resolveDeployTargets({ ...intent, businessScopeId: scopeId }, controller.signal);
      if (current !== generation.current || controller.signal.aborted) return;
      setResolution(result); onChange(result);
    } catch (error) { if (current === generation.current && !controller.signal.aborted) setFailure(isForbiddenRequestError(error) ? 'forbidden' : 'error'); }
    finally { if (current === generation.current) setResolving(false); }
  };
  const labels = {
    searchPlaceholder: t('business.deploy.selection.filterPage'), available: t('business.deploy.selection.available'), selected: t('business.deploy.selection.selected'),
    add: t('business.deploy.selection.add'), remove: t('business.deploy.selection.remove'), excluded: t('business.deploy.selection.excluded'), staleItem: t('business.deploy.selection.stale'),
    loading: t('common.loading'), empty: t('common.noData'), error: t('business.deploy.selection.loadFailed'), forbidden: t('business.deploy.preview.forbidden'), stale: t('business.deploy.selection.stale'), partial: t('business.deploy.preview.partial'), ready: t('business.deploy.preview.ready'),
  };
  const budget = intent.hostIds.length + intent.groupIds.length + intent.manual.length + intent.excludedIds.length;
  return <Space className="deploy-target-selector" direction="vertical" size={12} style={{ width: '100%' }}>
    <Input allowClear value={keyword} aria-label={t('business.deploy.selection.search')} placeholder={t('business.deploy.selection.search')} onChange={(value) => { setHosts([]); setLoading(true); setKeyword(value); setPage(1); }} />
    <ContextSelector title={t('business.deploy.selection.title')} options={options} selected={selected} labels={labels} state={loading ? 'loading' : lookupFailed || (hosts.length === 0 ? 'empty' : 'ready')} maxVisibleOptions={25} onSelect={(option) => toggle(option, false)} onRemove={(option) => toggle(option, true)} />
    {lookupFailed && <Button onClick={() => setRetry((value) => value + 1)}>{t('common.retry')}</Button>}
    <Pagination simple current={page} pageSize={25} total={total} onChange={(value) => { setHosts([]); setLoading(true); setPage(value); }} />
    <Checkbox checked={intent.includeScope} onChange={(value) => change({ ...intent, includeScope: value })}>{t('business.deploy.selection.includeScope')}</Checkbox>
    <Select mode="multiple" allowClear showSearch value={intent.groupIds} placeholder={t('business.deploy.selection.groups')} aria-label={t('business.deploy.selection.groups')} onChange={(ids: number[]) => change({ ...intent, groupIds: ids })}>{groups.map((group) => <Select.Option key={group.id} value={group.id}>{group.label}</Select.Option>)}</Select>
    <Input.TextArea value={intent.manual.join('\n')} aria-label={t('business.deploy.selection.manual')} placeholder={t('business.deploy.selection.manual')} maxLength={1280000} autoSize={{ minRows: 2, maxRows: 5 }} onChange={(value) => change({ ...intent, manual: value.split(/\n/).filter((item) => item.trim()) })} />
    <Select mode="multiple" allowClear value={intent.excludedIds} placeholder={t('business.deploy.selection.exclude')} aria-label={t('business.deploy.selection.exclude')} onChange={(ids: number[]) => change({ ...intent, excludedIds: ids })}>{(resolution?.hosts || hosts).map((host) => <Select.Option key={host.id} value={host.id}>{host.hostname} · {host.ip}</Select.Option>)}</Select>
    <Button loading={resolving} disabled={budget > 10000} onClick={() => void resolve()}>{t('business.deploy.selection.resolve')}</Button>
    {budget > 10000 && <Alert type="error" content={t('business.deploy.selection.budget')} />}
    {failure && <Alert type="error" content={t(failure === 'forbidden' ? 'business.deploy.preview.forbidden' : 'business.deploy.selection.loadFailed')} />}
    {resolution && <div role="status"><Typography.Text>{t('business.deploy.selection.summary', { count: resolution.hosts.length, duplicate: resolution.duplicateCount, unavailable: resolution.unavailableCount, ineligible: resolution.ineligibleCount, excluded: resolution.excludedCount })}</Typography.Text>{(resolution.unavailableCount > 0 || resolution.ineligibleCount > 0) && <Alert type="warning" content={t('business.deploy.selection.invalid')} />}</div>}
  </Space>;
}
