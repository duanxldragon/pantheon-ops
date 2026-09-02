import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Select, Space } from '@arco-design/web-react';
import { IconEye, IconSearch } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppDrawer,
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { getDatasourceList, getTrace, searchTraces, type DatasourceRow, type TempoTraceSummary } from './api';
import '../../system/components/shared/list-page.css';

function traceStartTime(value?: string) {
  if (!value) return '-';
  const milliseconds = value.length > 13 ? Number(value.slice(0, 13)) : Number(value);
  return Number.isFinite(milliseconds) ? new Date(milliseconds).toLocaleString() : value;
}

function traceDuration(row: TempoTraceSummary) {
  if (typeof row.durationMs === 'number') return `${row.durationMs.toFixed(2)} ms`;
  if (row.durationNano) return `${(Number(row.durationNano) / 1_000_000).toFixed(2)} ms`;
  return '-';
}

export default function TraceExplorer() {
  const { t } = useTranslation();
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [datasourceId, setDatasourceId] = useState<number>();
  const [query, setQuery] = useState('{ resource.service.name != nil }');
  const [items, setItems] = useState<TempoTraceSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [hasQueried, setHasQueried] = useState(false);
  const [detail, setDetail] = useState<Record<string, unknown> | null>(null);
  const [detailTraceId, setDetailTraceId] = useState('');
  const [detailLoading, setDetailLoading] = useState(false);

  useEffect(() => {
    queueMicrotask(async () => {
      try {
        const response = await getDatasourceList();
        const tempoSources = response.items.filter((item) => item.type === 'tempo');
        setDatasources(tempoSources);
        setDatasourceId(tempoSources.find((item) => item.isDefault)?.id ?? tempoSources[0]?.id);
      } catch (requestError) {
        setError(requestError);
      }
    });
  }, []);

  const executeQuery = useCallback(async () => {
    if (!datasourceId || !query.trim()) {
      Message.warning(t('business.observability.traces.queryRequired'));
      return;
    }
    setLoading(true);
    setError(null);
    setHasQueried(true);
    try {
      const response = await searchTraces(datasourceId, { query: query.trim(), limit: 100 });
      setItems(response.traces ?? []);
    } catch (requestError) {
      setError(requestError);
      setItems([]);
      Message.error(t('business.observability.traces.queryFailed'));
    } finally {
      setLoading(false);
    }
  }, [datasourceId, query, t]);

  const openDetail = useCallback(async (traceId: string) => {
    if (!datasourceId) return;
    setDetailTraceId(traceId);
    setDetailLoading(true);
    try {
      setDetail(await getTrace(datasourceId, traceId));
    } catch {
      setDetail(null);
      Message.error(t('business.observability.traces.detailFailed'));
    } finally {
      setDetailLoading(false);
    }
  }, [datasourceId, t]);

  const columns = useMemo<ColumnProps<TempoTraceSummary>[]>(
    () => [
      { title: t('business.observability.traces.field.traceId'), dataIndex: 'traceID', width: 260, ellipsis: true },
      { title: t('business.observability.traces.field.service'), dataIndex: 'rootServiceName', width: 180 },
      { title: t('business.observability.traces.field.operation'), dataIndex: 'rootTraceName', width: 220 },
      { title: t('business.observability.traces.field.start'), dataIndex: 'startTimeUnixNano', width: 190, render: (value?: string) => traceStartTime(value) },
      { title: t('business.observability.traces.field.duration'), width: 120, render: (_: unknown, row) => traceDuration(row) },
      {
        title: t('common.action'),
        width: 90,
        fixed: 'right',
        render: (_: unknown, row) => <Button type="text" icon={<IconEye />} onClick={() => void openDetail(row.traceID)}>{t('common.view')}</Button>,
      },
    ],
    [openDetail, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.observability.traces.hero.eyebrow')}
          title={t('operations.observability.traces.menu')}
          description={t('business.observability.traces.hero.title')}
          metrics={[
            { key: 'sources', label: t('business.observability.traces.hero.sources'), value: datasources.length },
            { key: 'results', label: t('business.observability.traces.hero.results'), value: items.length },
          ]}
        />
        <SearchToolbar
          keyword={query}
          keywordPlaceholder={t('business.observability.traces.queryPlaceholder')}
          onKeywordChange={setQuery}
          inlineFilters={
            <Select value={datasourceId} placeholder={t('business.observability.common.datasource')} style={{ width: 240 }} onChange={setDatasourceId}>
              {datasources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}
            </Select>
          }
          hasActiveFilters={Boolean(query)}
          onClearAll={() => setQuery('')}
          extra={<Button type="primary" icon={<IconSearch />} loading={loading} onClick={() => void executeQuery()}>{t('common.search')}</Button>}
        />
        <Card className="page-panel system-list__table-card">
          {loading && items.length === 0 ? <PageLoading /> : null}
          {error && items.length === 0 ? <PageError onRetry={() => void executeQuery()} /> : null}
          {!loading && !error && datasources.length === 0 ? <PageEmpty description={t('business.observability.traces.noDatasource')} /> : null}
          {!loading && !error && datasources.length > 0 && hasQueried && items.length === 0 ? <PageEmpty description={t('business.observability.traces.empty')} /> : null}
          {!loading && !error && items.length > 0 ? <AppTable rowKey="traceID" columns={columns} data={items} pagination={{ pageSize: 20 }} scroll={{ x: 'max-content' }} /> : null}
        </Card>
      </Space>
      <AppDrawer
        visible={Boolean(detailTraceId)}
        title={t('business.observability.traces.detailTitle', { traceId: detailTraceId })}
        size="xl"
        footer={null}
        onCancel={() => { setDetailTraceId(''); setDetail(null); }}
      >
        {detailLoading ? <PageLoading /> : null}
        {!detailLoading && detail ? <pre style={{ margin: 0, overflow: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{JSON.stringify(detail, null, 2)}</pre> : null}
        {!detailLoading && !detail ? <PageEmpty description={t('business.observability.traces.detailEmpty')} /> : null}
      </AppDrawer>
    </PageContainer>
  );
}
