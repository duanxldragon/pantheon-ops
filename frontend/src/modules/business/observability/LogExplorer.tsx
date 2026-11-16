import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Card, Input, Message, Select, Space, Tag } from '@arco-design/web-react';
import { IconSearch } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppTable,
  buildStandardPagination,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { isForbiddenRequestError } from '../../../api/request';
import './log-explorer.css';
import LogResultPanels from './LogResultPanels';
import { readLokiRows, type LogRow } from './logResults';
import { getDatasourceList, queryLogs, type DatasourceRow, type QueryRecipe, type QueryTime, type RunQueryRequest } from './api';
import QueryWorkspaceHistory, { type QueryRestoration } from './QueryWorkspaceHistory';
import { copyQueryRecipe, queryTimeBounds, utcInputTime, useQueryWorkspaceIdentity } from './queryWorkspaceState';
import '../../../assets/list-page.css';

interface FollowIntent { datasourceId: number; query: string; limit: number; recipe: QueryRecipe | null }

function formatTimestamp(value: string) {
  const milliseconds = value.length > 13 ? Number(value.slice(0, 13)) : Number(value);
  return Number.isFinite(milliseconds) ? new Date(milliseconds).toLocaleString() : value;
}

export default function LogExplorer() {
  const { t } = useTranslation();
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [datasourceId, setDatasourceId] = useState<number>();
  const [query, setQuery] = useState('{app=~".+"}');
  const [recipe, setRecipe] = useState<QueryRecipe | null>(null);
  const [restoredRawTime, setRestoredRawTime] = useState<QueryTime | null>(null);
  const [restored, setRestored] = useState(false);
  const [historyUnavailable, setHistoryUnavailable] = useState(false);
  const [items, setItems] = useState<LogRow[]>([]);
  const [completed, setCompleted] = useState<{ key: string; datasourceId: number; start: string; end: string; limit: number; warningCount: number; omittedRows: number; atLimit: boolean; follow: boolean } | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [hasQueried, setHasQueried] = useState(false);
  const [start, setStart] = useState(() => new Date(Date.now() - 15 * 60_000).toISOString().slice(0, 16));
  const [end, setEnd] = useState(() => new Date().toISOString().slice(0, 16));
  const [limit, setLimit] = useState(500);
  const [stopped, setStopped] = useState(false);
  const [followIntent, setFollowIntent] = useState<FollowIntent | null>(null);
  const [forbidden, setForbidden] = useState(false);
  const request = useRef<AbortController | null>(null);
  const followHistory = useRef<number | undefined>(undefined);
  const clearWorkspace = useCallback(() => {
    request.current?.abort(); request.current = null; followHistory.current = undefined;
    setFollowIntent(null); setItems([]); setCompleted(null); setLoading(false); setError(null);
    setHasQueried(false); setStopped(false); setRestored(false); setHistoryUnavailable(false);
  }, []);
  const { epoch: epoch, snapshot, isCurrent } = useQueryWorkspaceIdentity(useCallback(() => {
    clearWorkspace(); setDatasources([]); setDatasourceId(undefined); setQuery(''); setRecipe(null); setRestoredRawTime(null); setForbidden(false);
  }, [clearWorkspace]));
  const denyWorkspace = useCallback(() => { clearWorkspace(); setForbidden(true); }, [clearWorkspace]);
  useEffect(() => () => { request.current?.abort(); }, []);
  const stopQuery = () => {
    followHistory.current = undefined;
    setFollowIntent(null);
    request.current?.abort(); request.current = null;
    setLoading(false); setStopped(true);
  };
  const stopFollowing = () => { if (followIntent) stopQuery(); };
  const chooseDatasource = (id: number) => {
    followHistory.current = undefined; setRecipe(null); setRestoredRawTime(null); setRestored(false); setHistoryUnavailable(false);
    setFollowIntent(null);
    request.current?.abort(); request.current = null;
    setDatasourceId(id); setLoading(false); setItems([]); setCompleted(null); setError(null);
    setStopped(false); setForbidden(false); setHasQueried(false);
  };

  useEffect(() => {
    const controller = new AbortController();
    const captured = snapshot();
    void (async () => {
      try {
        const response = await getDatasourceList(controller.signal);
        if (controller.signal.aborted || !isCurrent(captured)) return;
        const lokiSources = response.items.filter((item) => item.type === 'loki');
        setDatasources(lokiSources);
        setDatasourceId(lokiSources.find((item) => item.isDefault)?.id ?? lokiSources[0]?.id);
      } catch (requestError) {
        if (!controller.signal.aborted && isCurrent(captured)) { setError(requestError); setForbidden(isForbiddenRequestError(requestError)); }
      }
    })();
    return () => controller.abort();
  }, [epoch, snapshot, isCurrent]);

  const executeQuery = useCallback(async (follow?: FollowIntent) => {
    const source = follow?.datasourceId ?? datasourceId;
    const expression = follow?.query ?? query.trim();
    const intentRecipe = follow ? follow.recipe : recipe;
    const rowLimit = follow?.limit ?? intentRecipe?.limit ?? limit;
    if (!source || (!intentRecipe && !expression)) {
      Message.warning(t('business.observability.logs.queryRequired'));
      return false;
    }
    const bounds = follow ? queryTimeBounds({ mode: 'relative', duration: '15m' }) : intentRecipe ? queryTimeBounds(intentRecipe.time) : restoredRawTime ? queryTimeBounds(restoredRawTime) : { start: utcInputTime(start), end: utcInputTime(end) };
    const last = new Date(bounds.end), first = new Date(bounds.start);
    if (!Number.isFinite(first.getTime()) || !Number.isFinite(last.getTime()) || last <= first || last.getTime() - first.getTime() > 86_400_000) {
      Message.warning(t('business.observability.logs.invalidRange')); return false;
    }
    request.current?.abort();
    const controller = new AbortController(); request.current = controller;
    const captured = snapshot();
    setLoading(true); setStopped(false); setForbidden(false);
    setError(null);
    setHasQueried(true);
    setRestored(false); setHistoryUnavailable(false);
    try {
      const body: RunQueryRequest = intentRecipe
        ? { recipe: { ...intentRecipe, ...(follow ? { time: { mode: 'relative', duration: '15m' }, limit: rowLimit } : {}) } }
        : { rawQuery: expression, time: follow ? { mode: 'relative', duration: '15m' } : restoredRawTime ?? { mode: 'absolute', start: first.toISOString(), end: last.toISOString() }, limit: rowLimit, resultMode: 'table' };
      if ('recipe' in body && !copyQueryRecipe(body.recipe, 'loki')) throw new Error('invalid-recipe');
      if (follow) { body.historyMode = 'coalesced'; if (followHistory.current) body.historyId = followHistory.current; }
      const response = await queryLogs(source, body, controller.signal);
      if (request.current !== controller || controller.signal.aborted || !isCurrent(captured)) return false;
      const result = readLokiRows(response.result, rowLimit);
      setHistoryUnavailable(response.historyUnavailable);
      if (follow && !response.historyUnavailable) {
        if (!Number.isSafeInteger(response.historyId) || !response.historyId || response.historyId < 1) throw new Error('invalid-history');
        followHistory.current = response.historyId;
      }
      setItems(result.rows);
      setCompleted({ key: JSON.stringify([source, expression, intentRecipe, follow ? 'follow' : start, follow ? 'follow' : end, rowLimit]), datasourceId: source, start: first.toISOString(), end: last.toISOString(), limit: rowLimit, warningCount: result.warningCount, omittedRows: result.omittedRows, atLimit: result.atLimit, follow: Boolean(follow) });
      return true;
    } catch (requestError) {
      if (request.current !== controller || controller.signal.aborted || !isCurrent(captured)) return false;
      setError(requestError);
      setItems([]); setCompleted(null);
      if (isForbiddenRequestError(requestError)) denyWorkspace();
      return false;
    } finally {
      if (request.current === controller) { request.current = null; setLoading(false); }
    }
  }, [datasourceId, query, start, end, limit, recipe, restoredRawTime, t, snapshot, isCurrent, denyWorkspace]);

  const applyRestoration = useCallback((value: QueryRestoration) => {
    setDatasourceId(value.datasourceId); setRecipe(value.recipe); setQuery(''); setLimit(value.limit); setRestoredRawTime(value.recipe ? null : value.time);
    if (value.time.mode === 'absolute') { setStart(value.time.start.slice(0, -1)); setEnd(value.time.end.slice(0, -1)); }
    else { const last = new Date(); const minutes = { '5m': 5, '15m': 15, '1h': 60, '6h': 360, '24h': 1440 }[value.time.duration]; setStart(new Date(last.getTime() - minutes * 60000).toISOString().slice(0, 16)); setEnd(last.toISOString().slice(0, 16)); }
    setRestored(true); setForbidden(false);
  }, []);

  const executeRef = useRef(executeQuery);
  useEffect(() => { executeRef.current = executeQuery; }, [executeQuery]);

  useEffect(() => {
    if (!followIntent) return;
    let active = true;
    let timer: number | undefined;
    let ownedRequest: AbortController | null = null;
    const poll = async () => {
      const pending = executeRef.current(followIntent);
      ownedRequest = request.current;
      const success = await pending;
      if (!active) return;
      if (!success) { setFollowIntent(null); return; }
      timer = window.setTimeout(() => { void poll(); }, 5000);
    };
    void poll();
    return () => {
      active = false;
      window.clearTimeout(timer);
      if (ownedRequest && request.current === ownedRequest) ownedRequest.abort();
    };
  }, [followIntent]);

  const columns = useMemo<ColumnProps<LogRow>[]>(
    () => [
      { title: t('business.observability.logs.field.time'), dataIndex: 'timestamp', width: 190, render: (value: string) => formatTimestamp(value) },
      {
        title: t('business.observability.logs.field.labels'),
        dataIndex: 'labels',
        width: 300,
        render: (_: unknown, row) => {
          const labels = Object.entries(row.labels);
          const previews = labels.slice(0, 6).map(([key, value]) => ({ key, text: `${key.slice(0, 80)}=${value.slice(0, 80)}` }));
          return <div className="observability-log-labels">
            <Space size={4} wrap>
              {previews.map(({ key, text }) => <Tag key={key}><span className="observability-log-labels__value">{text.length > 80 ? `${text.slice(0, 80)}…` : text}</span></Tag>)}
            </Space>
            {labels.length > 6 && <div className="observability-log-labels__note">{t('business.observability.logs.labelsOmitted', { count: labels.length - 6 })}</div>}
            {previews.some(({ text }) => text.length > 80) && <div className="observability-log-labels__note">{t('business.observability.logs.labelsShortened')}</div>}
          </div>;
        },
      },
      { title: t('business.observability.logs.field.message'), dataIndex: 'message', ellipsis: true },
    ],
    [t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.observability.logs.hero.eyebrow')}
          title={t('operations.observability.logs.menu')}
          description={t('business.observability.logs.hero.title')}
          metrics={[
            { key: 'sources', label: t('business.observability.logs.hero.sources'), value: datasources.length },
            { key: 'results', label: t('business.observability.logs.hero.results'), value: items.length },
          ]}
        />
        <SearchToolbar
          keyword={recipe ? undefined : query}
          keywordPlaceholder={t('business.observability.logs.queryPlaceholder')}
          onKeywordChange={recipe ? undefined : (value) => { stopFollowing(); setQuery(value); setRestored(false); }}
          inlineFilters={
            <Select
              value={datasourceId}
              placeholder={t('business.observability.common.datasource')}
              style={{ width: 240 }}
              onChange={chooseDatasource}
            >
              {datasources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}
            </Select>
          }
          hasActiveFilters={Boolean(query)}
          onClearAll={() => { stopFollowing(); setQuery(''); }}
          extra={<Space wrap>
            <Button type="primary" icon={<IconSearch />} disabled={loading} onClick={() => { stopQuery(); void executeQuery(); }}>{t('common.search')}</Button>
            <Button disabled={loading || Boolean(followIntent) || !datasourceId || (!recipe && !query.trim())} onClick={() => { if (datasourceId && (recipe || query.trim())) { followHistory.current = undefined; setFollowIntent({ datasourceId, query: query.trim(), limit, recipe }); } }}>{t('business.observability.logs.follow')}</Button>
            <Button disabled={!loading && !followIntent} onClick={stopQuery}>{t('business.observability.logs.stop')}</Button>
          </Space>}
        />
        <QueryWorkspaceHistory selectionKey={JSON.stringify([query, start, end, limit])} sourceId={datasourceId} kind="loki" sources={datasources} recipe={recipe} onRecipeChange={(value) => { clearWorkspace(); setRecipe(value); setQuery(''); setRestoredRawTime(null); if (value) setLimit(value.limit); }} onBeforeRestore={clearWorkspace} onRestore={applyRestoration} onDenied={denyWorkspace} epoch={epoch} forbidden={forbidden} />
        {restored && <Alert type="info" content={t('business.observability.workspace.restoredRun')} />}
        {historyUnavailable && <Alert type="warning" content={t('business.observability.workspace.historyUnavailable')} />}
        <div className="observability-log-range">
          {!recipe && <><label>{t('business.observability.logs.startUTC')}<Input type="datetime-local" step="0.001" value={start} onChange={(value) => { stopFollowing(); setStart(value); setRestoredRawTime(null); }} /></label>
          <label>{t('business.observability.logs.endUTC')}<Input type="datetime-local" step="0.001" value={end} onChange={(value) => { stopFollowing(); setEnd(value); setRestoredRawTime(null); }} /></label></>}
          <label>{t('business.observability.logs.limit')}<Select aria-label={t('business.observability.logs.limit')} value={limit} onChange={(value) => { stopFollowing(); setLimit(value); if (recipe) { clearWorkspace(); setRecipe({ ...recipe, limit: value }); } }}
            options={[...new Set([100, 500, 1000, limit])].sort((a, b) => a - b).map((value) => ({ value, label: String(value) }))} /></label>
          <span>{t('business.observability.logs.budget')}</span>
        </div>
        {followIntent && <Alert type="info" content={t('business.observability.logs.following')} />}
        <Card className="page-panel system-list__table-card">
          {loading ? <Tag>{t('business.observability.logs.running')}</Tag> : null}
          {completed && <div className="observability-log-results__note">
            {t('business.observability.logs.resultContext', { datasource: datasources.find((source) => source.id === completed.datasourceId)?.name ?? completed.datasourceId, start: completed.start, end: completed.end, limit: completed.limit })}
          </div>}
          {completed?.follow && <Tag>{t('business.observability.logs.followSnapshot')}</Tag>}
          {completed && completed.key !== JSON.stringify([datasourceId, query.trim(), recipe, followIntent ? 'follow' : start, followIntent ? 'follow' : end, limit]) && <Alert type="warning" content={t('business.observability.logs.inputsChanged')} />}
          {completed && loading && <Tag>{t('business.observability.logs.previousWhileRunning')}</Tag>}
          {completed && completed.warningCount > 0 && <Alert type="warning" content={t('business.observability.logs.providerWarnings', { count: completed.warningCount })} />}
          {completed && completed.omittedRows > 0 && <Alert type="warning" content={t('business.observability.logs.rowsOmitted', { count: completed.omittedRows })} />}
          {completed && completed.atLimit && completed.omittedRows === 0 && <Tag>{t('business.observability.logs.rowLimitReached')}</Tag>}
          {stopped ? <Tag>{t('business.observability.logs.stopped')}</Tag> : null}
          {forbidden ? <PageEmpty description={t('business.observability.logs.forbidden')} /> : null}
          {loading && items.length === 0 ? <PageLoading /> : null}
          {error && !forbidden && items.length === 0 ? <PageError onRetry={() => void executeQuery()} /> : null}
          {!loading && !error && datasources.length === 0 ? <PageEmpty description={t('business.observability.logs.noDatasource')} /> : null}
          {!loading && !error && datasources.length > 0 && hasQueried && !stopped && items.length === 0 ? <PageEmpty description={t('business.observability.logs.empty')} /> : null}
          {!error && items.length > 0 ? <LogResultPanels rows={items} logs={<AppTable rowKey="id" columns={columns} data={items} pagination={buildStandardPagination(t, { pageSize: 50, total: items.length })} scroll={{ x: 'max-content' }} />} /> : null}
        </Card>
      </Space>
    </PageContainer>
  );
}
