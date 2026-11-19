import { useTranslation } from 'react-i18next';
import { Alert, Button, Checkbox, Input, Pagination, Select, Space, Typography } from '@arco-design/web-react';
import { ContextSelector } from '../../../../components';
import type { DeployTargetResolution, DeployTargetSelection } from '../api';
import { parseManualLines, TARGET_CANDIDATE_PAGE_SIZE, TARGET_SELECTION_BUDGET, useBusinessTargetSelector } from '../../shared/targetSelector';
interface Props {
  scopeId: number;
  groups: Array<{ id: number; label: string }>;
  initial?: DeployTargetSelection;
  initialHostIds?: number[];
  onChange: (resolution: DeployTargetResolution | null) => void;
}
export default function DeployTargetSelector({ scopeId, groups, initial, initialHostIds, onChange }: Readonly<Props>) {
  const { t } = useTranslation();
  const { intent, hosts, total, page, keyword, loading, lookupFailed, resolving, resolution, failure, onSearch, onPage, reload, change, resolve, options, selected, budget, summary } =
    useBusinessTargetSelector({ scopeId, groups, initial, initialHostIds, onChange });
  const labels = {
    searchPlaceholder: t('business.deploy.selection.filterPage'), available: t('business.deploy.selection.available'), selected: t('business.deploy.selection.selected'),
    add: t('business.deploy.selection.add'), remove: t('business.deploy.selection.remove'), excluded: t('business.deploy.selection.excluded'), staleItem: t('business.deploy.selection.stale'),
    loading: t('common.loading'), empty: t('common.noData'), error: t('business.deploy.selection.loadFailed'), forbidden: t('business.deploy.preview.forbidden'), stale: t('business.deploy.selection.stale'), partial: t('business.deploy.preview.partial'), ready: t('business.deploy.preview.ready'),
  };
  return <Space className="deploy-target-selector" direction="vertical" size={12} style={{ width: '100%' }}>
    <Input allowClear value={keyword} aria-label={t('business.deploy.selection.search')} placeholder={t('business.deploy.selection.search')} onChange={onSearch} />
    <ContextSelector title={t('business.deploy.selection.title')} options={options} selected={selected} labels={labels} state={loading ? 'loading' : lookupFailed || (hosts.length === 0 ? 'empty' : 'ready')} maxVisibleOptions={25} onSelect={(option) => change({ ...intent, hostIds: [...intent.hostIds, Number(option.id)] })} onRemove={(option) => change({ ...intent, hostIds: intent.hostIds.filter((id) => String(id) !== option.id) })} />
    {lookupFailed && <Button onClick={reload}>{t('common.retry')}</Button>}
    <Pagination simple current={page} pageSize={TARGET_CANDIDATE_PAGE_SIZE} total={total} onChange={onPage} />
    <Checkbox checked={intent.includeScope} onChange={(value) => change({ ...intent, includeScope: value })}>{t('business.deploy.selection.includeScope')}</Checkbox>
    <Select mode="multiple" allowClear showSearch value={intent.groupIds} placeholder={t('business.deploy.selection.groups')} aria-label={t('business.deploy.selection.groups')} onChange={(ids: number[]) => change({ ...intent, groupIds: ids })}>{groups.map((group) => <Select.Option key={group.id} value={group.id}>{group.label}</Select.Option>)}</Select>
    <Input.TextArea value={intent.manual.join('\n')} aria-label={t('business.deploy.selection.manual')} placeholder={t('business.deploy.selection.manual')} maxLength={1280000} autoSize={{ minRows: 2, maxRows: 5 }} onChange={(value) => change({ ...intent, manual: parseManualLines(value) })} />
    <Select mode="multiple" allowClear value={intent.excludedIds} placeholder={t('business.deploy.selection.exclude')} aria-label={t('business.deploy.selection.exclude')} onChange={(ids: number[]) => change({ ...intent, excludedIds: ids })}>{(resolution?.hosts || hosts).map((host) => <Select.Option key={host.id} value={host.id}>{host.hostname} · {host.ip}</Select.Option>)}</Select>
    <Button loading={resolving} disabled={budget > TARGET_SELECTION_BUDGET} onClick={() => void resolve()}>{t('business.deploy.selection.resolve')}</Button>
    {budget > TARGET_SELECTION_BUDGET && <Alert type="error" content={t('business.deploy.selection.budget')} />}
    {failure && <Alert type="error" content={t(failure === 'forbidden' ? 'business.deploy.preview.forbidden' : 'business.deploy.selection.loadFailed')} />}
    {summary && <div role="status"><Typography.Text>{t('business.deploy.selection.summary', { count: summary.valid, duplicate: summary.duplicate, unavailable: summary.unavailable, ineligible: summary.ineligible, excluded: summary.excluded })}</Typography.Text>{(summary.unavailable > 0 || summary.ineligible > 0) && <Alert type="warning" content={t('business.deploy.selection.invalid')} />}</div>}
  </Space>;
}