import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Card, Input, Message, Select, Space, Tag } from '@arco-design/web-react';
import { IconEye, IconSearch } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppDrawer,
  AppTable,
  buildStandardPagination,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { getDatasourceList, getTrace, searchTraces, type DatasourceRow, type TempoTraceSummary, type QueryRecipe } from './api';
import QueryWorkspaceHistory, { type QueryRestoration } from './QueryWorkspaceHistory';
import { copyQueryRecipe, initialQueryRecipe, queryTimeBounds, utcInputTime, useQueryWorkspaceIdentity } from './queryWorkspaceState';
import { isForbiddenRequestError } from '../../../api/request';
import './log-explorer.css';
import '../../../assets/list-page.css';

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
  const [recipe, setRecipe] = useState<QueryRecipe>(() => initialQueryRecipe('tempo'));
  const [restored, setRestored] = useState(false);
  const [historyUnavailable, setHistoryUnavailable] = useState(false);
  const [items, setItems] = useState<TempoTraceSummary[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [hasQueried, setHasQueried] = useState(false);
  const [detail, setDetail] = useState<Record<string, unknown> | null>(null);
  const [detailTraceId, setDetailTraceId] = useState('');
  const [detailLoading, setDetailLoading] = useState(false);

  const [start, setStart] = useState(() => new Date(Date.now() - 15 * 60_000).toISOString().slice(0, 16));
  const [end, setEnd] = useState(() => new Date().toISOString().slice(0, 16));
  const [stopped, setStopped] = useState(false);
  const [forbidden, setForbidden] = useState(false);
  const [detailError, setDetailError] = useState(false);
  const [completed, setCompleted] = useState<{ key: string; datasourceId: number; start: string; end: string } | null>(null);
  const request = useRef<AbortController | null>(null);
  const detailRequest = useRef<AbortController | null>(null);

  const closeDetail = useCallback(() => {
    detailRequest.current?.abort(); detailRequest.current = null;
    setDetailTraceId(''); setDetail(null); setDetailLoading(false); setDetailError(false);
  }, []);
  const clearResults = useCallback(() => {
    request.current?.abort(); request.current = null;
    setItems([]); setCompleted(null); setLoading(false); closeDetail();
  }, [closeDetail]);
  const clearWorkspace = useCallback(() => {
    clearResults(); setError(null); setHasQueried(false); setStopped(false); setRestored(false); setHistoryUnavailable(false);
  }, [clearResults]);
  const { epoch: epoch, snapshot, isCurrent } = useQueryWorkspaceIdentity(useCallback(() => {
    clearWorkspace(); setDatasources([]); setDatasourceId(undefined); setRecipe(initialQueryRecipe('tempo')); setForbidden(false);
  }, [clearWorkspace]));
  const denyWorkspace = useCallback(() => { clearWorkspace(); setForbidden(true); }, [clearWorkspace]);
  const chooseDatasource = (id: number) => {
    clearWorkspace(); setRecipe(initialQueryRecipe('tempo')); setDatasourceId(id); setForbidden(false);
  };
  const stopQuery = () => {
    request.current?.abort(); request.current = null; setLoading(false); setStopped(true);
  };
  useEffect(() => () => { request.current?.abort(); detailRequest.current?.abort(); }, []);
  useEffect(() => {
    const controller = new AbortController();
    const captured = snapshot();
    void (async () => {
      try {
        const response = await getDatasourceList(controller.signal);
        if (controller.signal.aborted || !isCurrent(captured)) return;
        const tempoSources = response.items.filter((item) => item.type === 'tempo');
        setDatasources(tempoSources);
        setDatasourceId(tempoSources.find((item) => item.isDefault)?.id ?? tempoSources[0]?.id);
      } catch (requestError) {
        if (!controller.signal.aborted && isCurrent(captured)) { setError(requestError); setForbidden(isForbiddenRequestError(requestError)); }
      }
    })();
    return () => controller.abort();
  }, [epoch, snapshot, isCurrent]);

  const executeQuery = useCallback(async () => {
    if (!datasourceId) {
      Message.warning(t('business.observability.traces.queryRequired')); return;
    }
    const bounds = queryTimeBounds(recipe.time);
    const last = new Date(bounds.end), first = new Date(bounds.start);
    if (!Number.isFinite(first.getTime()) || !Number.isFinite(last.getTime()) || last <= first || last.getTime() - first.getTime() > 86_400_000) {
      Message.warning(t('business.observability.logs.invalidRange')); return;
    }
    request.current?.abort(); closeDetail();
    const controller = new AbortController(); request.current = controller;
    const captured = snapshot();
    setLoading(true); setStopped(false); setError(null); setForbidden(false); setHasQueried(true);
    setRestored(false); setHistoryUnavailable(false);
    try {
      const safe = copyQueryRecipe(recipe, 'tempo');
      if (!safe) throw new Error('invalid-recipe');
      const response = await searchTraces(datasourceId, { recipe: safe }, controller.signal);
      if (request.current !== controller || controller.signal.aborted || !isCurrent(captured)) return;
      setItems((response.result.traces ?? []).slice(0, 100)); setHistoryUnavailable(response.historyUnavailable);
      setCompleted({ key: JSON.stringify([datasourceId, recipe, start, end]), datasourceId, start: first.toISOString(), end: last.toISOString() });
    } catch (requestError) {
      if (request.current !== controller || controller.signal.aborted || !isCurrent(captured)) return;
      setError(requestError); clearResults(); if (isForbiddenRequestError(requestError)) denyWorkspace();
    } finally {
      if (request.current === controller) { request.current = null; setLoading(false); }
    }
  }, [datasourceId, recipe, start, end, t, closeDetail, clearResults, snapshot, isCurrent, denyWorkspace]);

  const applyRestoration = useCallback((value: QueryRestoration) => {
    if (!value.recipe) return;
    setDatasourceId(value.datasourceId); setRecipe(value.recipe); setRestored(true); setForbidden(false);
    if (value.time.mode === 'absolute') { setStart(value.time.start.slice(0, -1)); setEnd(value.time.end.slice(0, -1)); }
  }, []);

  const openDetail = useCallback(async (traceId: string) => {
    if (!datasourceId || forbidden || loading) return;
    detailRequest.current?.abort();
    const controller = new AbortController(); detailRequest.current = controller;
    const captured = snapshot();
    setDetailTraceId(traceId); setDetail(null); setDetailLoading(true); setDetailError(false);
    try {
      const response = await getTrace(datasourceId, traceId, controller.signal);
      if (detailRequest.current !== controller || controller.signal.aborted || !isCurrent(captured)) return;
      setDetail(response);
    } catch (requestError) {
      if (detailRequest.current !== controller || controller.signal.aborted || !isCurrent(captured)) return;
      if (isForbiddenRequestError(requestError)) {
        denyWorkspace(); setError(requestError);
      } else { setDetailError(true); }
    } finally {
      if (detailRequest.current === controller) { detailRequest.current = null; setDetailLoading(false); }
    }
  }, [datasourceId, forbidden, loading, denyWorkspace, snapshot, isCurrent]);

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
        render: (_: unknown, row) => <Button type="text" icon={<IconEye />} disabled={loading || forbidden} onClick={() => void openDetail(row.traceID)}>{t('common.detail')}</Button>,
      },
    ],
    [openDetail, t, loading, forbidden],
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
          inlineFilters={
            <Select aria-label={t('business.observability.common.datasource')} value={datasourceId} placeholder={t('business.observability.common.datasource')} style={{ width: 240 }} onChange={chooseDatasource}>
              {datasources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}
            </Select>
          }
          hasActiveFilters={false}
          onClearAll={() => undefined}
          extra={<Space wrap>
            <Button type="primary" icon={<IconSearch />} disabled={loading} onClick={() => void executeQuery()}>{t('common.search')}</Button>
            <Button disabled={!loading} onClick={stopQuery}>{t('business.observability.logs.stop')}</Button>
          </Space>}
        />
        <QueryWorkspaceHistory selectionKey={JSON.stringify([start, end])} sourceId={datasourceId} kind="tempo" sources={datasources} recipe={recipe} recipeOnly onRecipeChange={(value) => { clearWorkspace(); setRecipe(value ?? initialQueryRecipe('tempo')); }} onBeforeRestore={clearWorkspace} onRestore={applyRestoration} onDenied={denyWorkspace} epoch={epoch} forbidden={forbidden} />
        {restored && <Alert type="info" content={t('business.observability.workspace.restoredRun')} />}
        {historyUnavailable && <Alert type="warning" content={t('business.observability.workspace.historyUnavailable')} />}
        <div className="observability-log-range">
          <label>{t('business.observability.logs.startUTC')}<Input type="datetime-local" step="0.001" value={start} onChange={(value) => { setStart(value); setRecipe({ ...recipe, time: { mode: 'absolute', start: utcInputTime(value), end: utcInputTime(end) } }); }} /></label>
          <label>{t('business.observability.logs.endUTC')}<Input type="datetime-local" step="0.001" value={end} onChange={(value) => { setEnd(value); setRecipe({ ...recipe, time: { mode: 'absolute', start: utcInputTime(start), end: utcInputTime(value) } }); }} /></label>
          <span>{t('business.observability.traces.budget')}</span>
        </div>
        <Card className="page-panel system-list__table-card">
          {completed && <div className="observability-log-results__note">{t('business.observability.traces.resultContext', { datasource: datasources.find((source) => source.id === completed.datasourceId)?.name ?? completed.datasourceId, start: completed.start, end: completed.end })}</div>}
          {completed && completed.key !== JSON.stringify([datasourceId, recipe, start, end]) && <Alert type="warning" content={t('business.observability.traces.inputsChanged')} />}
          {loading && <Tag>{t('business.observability.traces.running')}</Tag>}
          {stopped && <Tag>{t('business.observability.traces.stopped')}</Tag>}
          {forbidden && <PageEmpty description={t('business.observability.traces.forbidden')} />}
          {loading && items.length === 0 ? <PageLoading /> : null}
          {error && !forbidden && items.length === 0 ? <PageError onRetry={() => void executeQuery()} /> : null}
          {!loading && !error && datasources.length === 0 ? <PageEmpty description={t('business.observability.traces.noDatasource')} /> : null}
          {!loading && !error && datasources.length > 0 && hasQueried && !stopped && items.length === 0 ? <PageEmpty description={t('business.observability.traces.empty')} /> : null}
          {!error && items.length > 0 ? <AppTable rowKey="traceID" columns={columns} data={items} pagination={buildStandardPagination(t, { pageSize: 20, total: items.length })} scroll={{ x: 'max-content' }} /> : null}
        </Card>
      </Space>
      <AppDrawer
        visible={Boolean(detailTraceId)}
        title={t('business.observability.traces.detailTitle', { traceId: detailTraceId })}
        size="xl"
        footer={null}
        onCancel={closeDetail}
      >
        {detailLoading ? <PageLoading /> : null}
        {detailError && <PageError onRetry={() => void openDetail(detailTraceId)} />}
        {!detailLoading && detail ? <pre tabIndex={0} role="region" aria-label={t('business.observability.traces.detailTitle', { traceId: detailTraceId })} style={{ margin: 0, overflow: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{JSON.stringify(detail, null, 2)}</pre> : null}
        {!detailLoading && !detail && !detailError ? <PageEmpty description={t('business.observability.traces.detailEmpty')} /> : null}
      </AppDrawer>
    </PageContainer>
  );
}
