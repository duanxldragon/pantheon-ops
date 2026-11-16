import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Pagination, Popconfirm, Select, Space, Tabs, Tag } from '@arco-design/web-react';
import { IconHistory, IconRefresh, IconSave } from '@arco-design/web-react/icon';
import { usePermission, isForbiddenRequestError } from '@duanxldragon/pantheon-base-ui';
import { AppDrawer, PageEmpty, PageError, PageLoading } from '../../../components';
import {
  deleteQueryHistory, deleteSavedView, getQueryCatalog, getQueryHistory, getSavedViews,
  restoreQueryHistory, restoreSavedView, saveQueryView,
  type DatasourceRow, type QueryCatalog, type QueryDuration, type QueryHistoryRow,
  type QueryRecipe, type QuerySourceKind, type SavedViewRow,
} from './api';
import { copyQueryRecipe, copyWorkspaceRow, initialQueryRecipe, supportedQueryCatalog, useQueryWorkspaceIdentity, validQueryTime } from './queryWorkspaceState';

export interface QueryRestoration {
  datasourceId: number;
  recipe: QueryRecipe | null;
  time: QueryHistoryRow['time'];
  limit: number;
  expressionUnavailable: boolean;
}
interface Props {
  sourceId?: number;
  kind: QuerySourceKind;
  sources: DatasourceRow[];
  recipe: QueryRecipe | null;
  onRecipeChange: (recipe: QueryRecipe | null) => void;
  onBeforeRestore: () => void;
  onRestore: (value: QueryRestoration) => void;
  onDenied: () => void;
  epoch: number;
  forbidden?: boolean;
  recipeOnly?: boolean;
  selectionKey?: string;
}

const prefix = 'business:observability:';
const key = (suffix: string) => `business.observability.workspace.${suffix}`;

export default function QueryWorkspaceHistory({ sourceId, kind, sources, recipe, onRecipeChange, onBeforeRestore, onRestore, onDenied, epoch, forbidden, recipeOnly, selectionKey }: Props) {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [catalog, setCatalog] = useState<QueryCatalog | null>(null);
  const [catalogError, setCatalogError] = useState(false);
  const [visible, setVisible] = useState(false);
  const [tab, setTab] = useState<'history' | 'views'>('history');
  const [page, setPage] = useState(1);
  const [rows, setRows] = useState<(QueryHistoryRow | SavedViewRow)[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<'unavailable' | 'conflict' | 'denied' | null>(null);
  const [notice, setNotice] = useState<'saved' | null>(null);
  const listRequest = useRef<AbortController | null>(null);
  const actionRequest = useRef<AbortController | null>(null);
  const catalogRequest = useRef<AbortController | null>(null);
  const previousEpoch = useRef(epoch);
  const selectionGeneration = useRef(0);
  const canHistory = hasPerm(`${prefix}query-history:view`);
  const canViews = hasPerm(`${prefix}saved-view:view`);
  const safeRecipe = recipe ? copyQueryRecipe(recipe, kind) : null;
  const sourceKey = JSON.stringify(sources.map(({ id, type }) => [id, type]).sort());
  const sourceAvailable = sources.some((source) => source.id === sourceId && source.type === kind);
  const recipeKey = JSON.stringify(recipe);

  const cancel = useCallback(() => {
    listRequest.current?.abort(); listRequest.current = null;
    actionRequest.current?.abort(); actionRequest.current = null;
  }, []);
  const { epoch: identityEpoch, snapshot, isCurrent } = useQueryWorkspaceIdentity(useCallback(() => {
    cancel(); catalogRequest.current?.abort(); catalogRequest.current = null;
    setCatalog(null); setCatalogError(false); setRows([]); setTotal(0); setVisible(false);
    setBusy(false); setLoading(false); setError(null); setNotice(null);
  }, [cancel]));
  // Cancel at commit, before an ignored-abort reply can restore stale conditions.
  // Semantic keys avoid cancelling merely because a parent recreated an array.
  useLayoutEffect(() => {
    selectionGeneration.current += 1;
    cancel();
    return () => { selectionGeneration.current += 1; cancel(); };
  }, [sourceId, kind, sourceKey, recipeKey, selectionKey, cancel]);
  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (active) { setBusy(false); setLoading(false); setRows([]); setTotal(0); setError(null); setNotice(null); } });
    return () => { active = false; };
  }, [sourceId, kind, sourceKey, recipeKey, selectionKey]);
  useEffect(() => {
    if (previousEpoch.current === epoch) return cancel;
    previousEpoch.current = epoch;
    let active = true;
    cancel();
    queueMicrotask(() => { if (active) { setRows([]); setTotal(0); setVisible(false); setBusy(false); setLoading(false); setError(null); setNotice(null); } });
    return () => { active = false; cancel(); };
  }, [epoch, cancel]);
  useEffect(() => {
    if (!forbidden) return;
    let active = true;
    cancel();
    queueMicrotask(() => { if (active) { setRows([]); setTotal(0); setBusy(false); setLoading(false); setError('denied'); } });
    return () => { active = false; };
  }, [forbidden, cancel]);

  const requestFailed = useCallback((failure: unknown) => {
    if (isForbiddenRequestError(failure)) {
      cancel(); setRows([]); setTotal(0); setBusy(false); setLoading(false); setError('denied'); onDenied();
    } else setError(typeof failure === 'object' && failure !== null && 'status' in failure && failure.status === 409 ? 'conflict' : 'unavailable');
  }, [cancel, onDenied]);

  useEffect(() => {
    const controller = new AbortController(); catalogRequest.current = controller;
    const captured = snapshot();
    queueMicrotask(() => { if (!controller.signal.aborted) { setCatalog(null); setCatalogError(false); } });
    if (!sourceId || !sourceAvailable || forbidden) return () => controller.abort();
    void getQueryCatalog(sourceId, controller.signal).then((value) => {
      if (controller.signal.aborted || !isCurrent(captured)) return;
      if (!supportedQueryCatalog(value, kind)) { setCatalogError(true); return; }
      setCatalog(value);
    }).catch((failure) => {
      if (!controller.signal.aborted && isCurrent(captured)) { setCatalogError(true); if (isForbiddenRequestError(failure)) onDenied(); }
    });
    return () => controller.abort();
  }, [sourceId, kind, sourceKey, sourceAvailable, epoch, identityEpoch, snapshot, isCurrent, forbidden, onDenied]);

  const load = useCallback(async (nextTab: 'history' | 'views', nextPage: number) => {
    listRequest.current?.abort();
    const controller = new AbortController(); listRequest.current = controller;
    const captured = snapshot();
    setLoading(true); setRows([]); setTotal(0); setError(null); setNotice(null);
    try {
      const result = nextTab === 'history' ? await getQueryHistory(nextPage, controller.signal) : await getSavedViews(nextPage, controller.signal);
      if (listRequest.current !== controller || controller.signal.aborted || !isCurrent(captured)) return;
      if (result.catalogVersion !== 1 || !Array.isArray(result.items) || result.items.length > 20 || !Number.isSafeInteger(result.total) || result.total < 0) throw new Error('workspace-unavailable');
      const safeRows = result.items.map((row) => copyWorkspaceRow(row, nextTab));
      if (safeRows.some((row) => !row) || new Set(safeRows.map((row) => row!.id)).size !== safeRows.length) throw new Error('workspace-unavailable');
      setRows(safeRows as (QueryHistoryRow | SavedViewRow)[]); setTotal(result.total); setTab(nextTab); setPage(nextPage);
    } catch (failure) {
      if (listRequest.current === controller && !controller.signal.aborted && isCurrent(captured)) requestFailed(failure);
    } finally {
      if (listRequest.current === controller) { listRequest.current = null; setLoading(false); }
    }
  }, [requestFailed, snapshot, isCurrent]);

  const operate = useCallback(async (operation: (signal: AbortSignal) => Promise<void>) => {
    if (actionRequest.current) return;
    const controller = new AbortController(); actionRequest.current = controller;
    setBusy(true); setError(null);
    try { await operation(controller.signal); }
    catch (failure) { if (actionRequest.current === controller && !controller.signal.aborted) requestFailed(failure); }
    finally { if (actionRequest.current === controller) { actionRequest.current = null; setBusy(false); } }
  }, [requestFailed]);

  const restore = (row: QueryHistoryRow | SavedViewRow) => {
    cancel(); onBeforeRestore(); setRows([]); setTotal(0); setLoading(false);
    const captured = snapshot();
    const generation = selectionGeneration.current;
    void operate(async (signal) => {
      const result = tab === 'history' ? await restoreQueryHistory(row.id, signal) : await restoreSavedView(row.id, signal);
      if (signal.aborted || generation !== selectionGeneration.current || !isCurrent(captured)) return;
      const restored = result.items?.[0] && copyWorkspaceRow(result.items[0], tab);
      const source = sources.find((item) => item.id === restored?.datasourceId && item.type === kind);
      if (result.catalogVersion !== 1 || result.items?.length !== 1 || !restored || restored.id !== row.id || !source || ('sourceKind' in restored && restored.sourceKind !== kind)) throw new Error('workspace-unavailable');
      const nextRecipe = restored.recipe ? copyQueryRecipe(restored.recipe, kind) : null;
      if (restored.recipe && !nextRecipe) throw new Error('workspace-unavailable');
      const time = nextRecipe?.time ?? ('time' in restored ? restored.time : null);
      const limit = nextRecipe?.limit ?? ('limit' in restored ? restored.limit : 0);
      if (!time || !validQueryTime(time) || !Number.isInteger(limit) || limit < 1 || limit > (kind === 'loki' ? 5000 : 1000) || (!nextRecipe && (!('expressionUnavailable' in restored) || !restored.expressionUnavailable))) throw new Error('workspace-unavailable');
      onRestore({ datasourceId: restored.datasourceId, recipe: nextRecipe, time: { ...time }, limit, expressionUnavailable: !nextRecipe });
      setVisible(false);
    });
  };
  const selectedRule = safeRecipe?.rules.find((rule) => rule.field === 'severity');
  const show = () => { const next = canHistory ? 'history' : 'views'; setVisible(true); void load(next, 1); };
  const close = () => { cancel(); setVisible(false); setRows([]); setTotal(0); setBusy(false); setLoading(false); };

  return <>
    <div className="observability-workspace-controls">
      <Space wrap>
        <Select aria-label={t(key('editorMode'))} value={recipe ? 'recipe' : 'raw'} disabled={!catalog || forbidden} style={{ width: 180 }} onChange={(mode) => onRecipeChange(mode === 'recipe' ? initialQueryRecipe(kind) : null)}>
          <Select.Option value="recipe">{t(key('catalogRecipe'))}</Select.Option>
          {!recipeOnly && <Select.Option value="raw">{t(key('rawEditor'))}</Select.Option>}
        </Select>
        {recipe && catalog && <>
          <Tag>{t(key(`field.${kind === 'loki' ? 'stream' : kind === 'tempo' ? 'service' : 'availability'}`))}</Tag>
          {kind === 'loki' && <Select aria-label={t(key('severity'))} value={selectedRule?.value ?? 'all'} style={{ width: 160 }} onChange={(value) => onRecipeChange({ ...recipe, rules: [{ field: 'stream', op: 'exists' }, ...(value === 'all' ? [] : [{ field: 'severity', op: 'equals', value }])] })}>
            <Select.Option value="all">{t(key('allSeverity'))}</Select.Option>
            {['info', 'warning', 'error'].map((value) => <Select.Option key={value} value={value}>{t(key(`severity.${value}`))}</Select.Option>)}
          </Select>}
          <Select aria-label={t(key('duration'))} value={recipe.time.mode === 'relative' ? recipe.time.duration : 'absolute'} style={{ width: 180 }} onChange={(duration) => onRecipeChange({ ...recipe, time: { mode: 'relative', duration: duration as QueryDuration } })}>
            {recipe.time.mode === 'absolute' && <Select.Option value="absolute" disabled>{t(key('absoluteRestored'))}</Select.Option>}
            {catalog.durations.map((duration) => <Select.Option key={duration} value={duration}>{t(key(`duration.${duration}`))}</Select.Option>)}
          </Select>
        </>}
      </Space>
      <Space wrap>
        <Button icon={<IconHistory />} disabled={busy || forbidden || (!canHistory && !canViews)} onClick={show}>{t(key('historyViews'))}</Button>
        {hasPerm(`${prefix}saved-view:create`) && <Button icon={<IconSave />} disabled={!sourceId || !safeRecipe || !catalog || busy || forbidden} loading={busy && !visible} onClick={() => { if (sourceId && safeRecipe) void operate(async (signal) => { await saveQueryView(sourceId, safeRecipe, signal); if (!signal.aborted) setNotice('saved'); }); }}>{t(key('saveView'))}</Button>}
      </Space>
    </div>
    {catalogError && !forbidden && <Alert type="warning" content={t(key('catalogUnavailable'))} />}
    {notice && <Alert type="info" content={t(key('saved'))} />}
    {!visible && error && <Alert type="warning" content={t(key(error === 'denied' ? 'forbidden' : error === 'conflict' ? 'conflict' : 'actionUnavailable'))} />}
    <AppDrawer visible={visible} title={t(key('historyViews'))} size="lg" width="min(860px, 100vw)" footer={null} onCancel={close}>
      <Tabs activeTab={tab} onChange={(value) => { if (!busy) void load(value as 'history' | 'views', 1); }}>
        {canHistory && <Tabs.TabPane key="history" title={t(key('history'))} />}
        {canViews && <Tabs.TabPane key="views" title={t(key('views'))} />}
      </Tabs>
      <div className="observability-workspace-drawer-actions">
        <Button icon={<IconRefresh />} disabled={loading || busy} onClick={() => void load(tab, page)}>{t('common.refresh')}</Button>
        {tab === 'history' && hasPerm(`${prefix}query-history:delete`) && <Popconfirm title={t(key('clearConfirm'))} onOk={() => operate(async (signal) => { await deleteQueryHistory(undefined, signal); if (!signal.aborted) await load('history', 1); })}>
          <Button status="danger" disabled={loading || busy || !total}>{t(key('clearHistory'))}</Button>
        </Popconfirm>}
      </div>
      <p className="observability-workspace-note">{t(key(tab === 'history' ? 'retention' : 'privateViews'))}</p>
      {loading && <PageLoading />}
      {error === 'denied' && <PageEmpty description={t(key('forbidden'))} />}
      {error === 'conflict' && <Alert type="warning" content={t(key('conflict'))} />}
      {error === 'unavailable' && <PageError onRetry={() => void load(tab, page)} />}
      {!loading && !error && rows.length === 0 && <PageEmpty description={t(key(tab === 'history' ? 'emptyHistory' : 'emptyViews'))} />}
      {!forbidden && !loading && error !== 'denied' && rows.map((row) => {
        const source = sources.find((item) => item.id === row.datasourceId && item.type === kind);
        const rowRecipe = row.recipe ? copyQueryRecipe(row.recipe, kind) : null;
        const time = 'createdAt' in row ? row.createdAt : row.updatedAt;
        return <article className="observability-workspace-row" key={row.id}>
          <div><strong>{t(key(tab === 'history' ? 'queryLabel' : 'viewLabel'), { id: row.id })}</strong><span className="observability-workspace-note">{Number.isFinite(Date.parse(time)) ? new Date(time).toLocaleString() : t(key('unavailableTime'))}</span></div>
          {'expressionUnavailable' in row && row.expressionUnavailable && <Tag>{t(key('metadataOnly'))}</Tag>}
          <Space wrap>
            {hasPerm(`${prefix}${tab === 'history' ? 'query-history' : 'saved-view'}:restore`) && <Button disabled={busy || !source || (tab === 'views' && !rowRecipe)} onClick={() => restore(row)}>{t(key('restore'))}</Button>}
            {tab === 'views' && 'revision' in row && hasPerm(`${prefix}saved-view:update`) && <Button disabled={busy || !safeRecipe || sourceId !== row.datasourceId} onClick={() => { if (safeRecipe) void operate(async (signal) => { await saveQueryView(row.datasourceId, safeRecipe, signal, row); if (!signal.aborted) await load('views', page); }); }}>{t(key('replace'))}</Button>}
            {hasPerm(`${prefix}${tab === 'history' ? 'query-history' : 'saved-view'}:delete`) && <Popconfirm title={t(key('deleteConfirm'))} onOk={() => operate(async (signal) => { if (tab === 'history') await deleteQueryHistory(row.id, signal); else await deleteSavedView(row.id, signal); if (!signal.aborted) await load(tab, page); })}>
              <Button status="danger" disabled={busy}>{t('common.delete')}</Button>
            </Popconfirm>}
          </Space>
        </article>;
      })}
      {!error && total > 20 && <Pagination current={page} pageSize={20} total={total} disabled={loading || busy} onChange={(value) => void load(tab, value)} />}
    </AppDrawer>
  </>;
}
