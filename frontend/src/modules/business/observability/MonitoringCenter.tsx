import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Card, Form, Input, InputNumber, Message, Popconfirm, Select, Space, Switch, Tabs, Tag } from '@arco-design/web-react';
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { usePermission, isForbiddenRequestError } from '@duanxldragon/pantheon-base-ui';
import { getBizScopeOptions, type BizScopeOptionItem } from '../bizscope/api';
import {
  createDatasource,
  createMonitoringTarget,
  deleteDatasource,
  deleteMonitoringTarget,
  getDatasourceList,
  getMonitoringTargetList,
  testDatasourceConnection,
  queryMetrics,
  type DatasourcePayload,
  type DatasourceRow,
  type MonitoringTargetRow,
  type QueryDuration,
  type QueryRecipe,
  type QueryTime,
} from './api';
import QueryWorkspaceHistory, { type QueryRestoration } from './QueryWorkspaceHistory';
import { copyQueryRecipe, initialQueryRecipe, queryTimeBounds, useQueryWorkspaceIdentity } from './queryWorkspaceState';
import PasswordInput from '../../../components/patterns/forms/PasswordInput';
import './log-explorer.css';
import '../../../assets/list-page.css';

const { TabPane } = Tabs;
const FormItem = Form.Item;

interface TargetFormValues {
  businessScopeId: number;
  targetType: 'host' | 'k8s_cluster' | 'other';
  targetName: string;
  endpoint?: string;
  metricsPath?: string;
  remark?: string;
}

function datasourceTypeColor(value: string): string {
  if (value === 'loki') {
    return 'cyan';
  }
  if (value === 'tempo') {
    return 'purple';
  }
  return 'arcoblue';
}

function datasourceStatusColor(value: string): string {
  if (value === 'healthy') {
    return 'green';
  }
  if (value === 'unhealthy') {
    return 'red';
  }
  return 'gray';
}

interface MetricResultRow { id: number; series: string; timestamp: string; value: string }

function MetricsQueryWorkspace({ sources }: { sources: DatasourceRow[] }) {
  const { t } = useTranslation();
  const key = (suffix: string) => `business.observability.workspace.${suffix}`;
  const metricSources = sources.filter((source) => source.type === 'prometheus' || source.type === 'victoriametrics');
  const [sourceId, setSourceId] = useState<number>();
  const source = metricSources.find((item) => item.id === sourceId);
  const kind = source?.type === 'victoriametrics' ? 'victoriametrics' : 'prometheus';
  const [recipe, setRecipe] = useState<QueryRecipe | null>(() => initialQueryRecipe('prometheus'));
  const [rawQuery, setRawQuery] = useState('');
  const [time, setTime] = useState<QueryTime>({ mode: 'relative', duration: '15m' });
  const [limit, setLimit] = useState(100);
  const [rows, setRows] = useState<MetricResultRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(false);
  const [forbidden, setForbidden] = useState(false);
  const [restored, setRestored] = useState(false);
  const [historyUnavailable, setHistoryUnavailable] = useState(false);
  const [completed, setCompleted] = useState<{ start: string; end: string; limit: number; warningCount: number } | null>(null);
  const request = useRef<AbortController | null>(null);
  const clear = useCallback(() => {
    request.current?.abort(); request.current = null;
    setRows([]); setLoading(false); setError(false); setCompleted(null); setRestored(false); setHistoryUnavailable(false);
  }, []);
  const { epoch, snapshot, isCurrent } = useQueryWorkspaceIdentity(useCallback(() => {
    clear(); setSourceId(undefined); setRecipe(null); setRawQuery(''); setForbidden(false);
  }, [clear, setRecipe, setRawQuery]));
  const deny = useCallback(() => { clear(); setRawQuery(''); setRecipe(null); setForbidden(true); }, [clear, setRecipe, setRawQuery]);
  useEffect(() => {
    if (sources.some((item) => item.id === sourceId)) return;
    let active = true;
    request.current?.abort(); request.current = null;
    queueMicrotask(() => {
      if (!active) return;
      clear();
      const next = sources.find((item) => (item.type === 'prometheus' || item.type === 'victoriametrics') && item.isDefault) ?? sources.find((item) => item.type === 'prometheus' || item.type === 'victoriametrics');
      setSourceId(next?.id); setRecipe(initialQueryRecipe(next?.type === 'victoriametrics' ? 'victoriametrics' : 'prometheus'));
    });
    return () => { active = false; };
  }, [sources, sourceId, clear]);
  useEffect(() => () => { request.current?.abort(); }, []);
  const restore = useCallback((value: QueryRestoration) => {
    setSourceId(value.datasourceId); setRecipe(value.recipe); setRawQuery(''); setTime(value.time); setLimit(value.limit); setRestored(true); setForbidden(false);
  }, [setRecipe, setRawQuery, setTime, setLimit]);
  const run = async () => {
    if (!source || forbidden || (!recipe && !rawQuery.trim())) return;
    clear();
    const controller = new AbortController(); request.current = controller;
    const captured = snapshot();
    setLoading(true);
    try {
      const safeRecipe = recipe && copyQueryRecipe(recipe, kind);
      if (recipe && !safeRecipe) throw new Error('invalid-recipe');
      const bounds = queryTimeBounds(safeRecipe?.time ?? time);
      const rowLimit = safeRecipe?.limit ?? limit;
      const response = await queryMetrics(source.id, safeRecipe ? { recipe: safeRecipe } : { rawQuery: rawQuery.trim(), time, limit, resultMode: 'table' }, controller.signal);
      if (controller.signal.aborted || request.current !== controller || !isCurrent(captured)) return;
      if (response.result?.status !== 'success' || response.result.data?.resultType !== 'vector' || !Array.isArray(response.result.data.result)) throw new Error('invalid-result');
      const warnings = response.result.warnings;
      if (warnings !== undefined && (!Array.isArray(warnings) || warnings.length > 1000 || warnings.some((value) => typeof value !== 'string'))) throw new Error('invalid-result');
      const values = response.result.data.result.slice(0, rowLimit).map((item, id) => {
        if (!item || !item.metric || typeof item.metric !== 'object' || Array.isArray(item.metric) || !Array.isArray(item.value) || item.value.length !== 2 || !Number.isFinite(item.value[0]) || typeof item.value[1] !== 'string') throw new Error('invalid-result');
        const labels = Object.entries(item.metric).slice(0, 16);
        if (labels.some(([, value]) => typeof value !== 'string')) throw new Error('invalid-result');
        const date = new Date(item.value[0] * 1000);
        if (!Number.isFinite(date.getTime())) throw new Error('invalid-result');
        return { id, series: labels.map(([field, value]) => `${field.slice(0, 80)}=${value.slice(0, 160)}`).join(', '), timestamp: date.toISOString(), value: item.value[1].slice(0, 80) };
      });
      setRows(values); setCompleted({ ...bounds, limit: rowLimit, warningCount: warnings?.length ?? 0 }); setHistoryUnavailable(response.historyUnavailable);
    } catch (failure) {
      if (controller.signal.aborted || request.current !== controller || !isCurrent(captured)) return;
      if (isForbiddenRequestError(failure)) deny(); else setError(true);
    } finally {
      if (request.current === controller) { request.current = null; setLoading(false); }
    }
  };
  const columns: ColumnProps<MetricResultRow>[] = [
    { title: t(key('series')), dataIndex: 'series', ellipsis: true },
    { title: t(key('sampleTime')), dataIndex: 'timestamp', width: 240 },
    { title: t(key('sampleValue')), dataIndex: 'value', width: 160 },
  ];
  return <Space direction="vertical" size={12} className="observability-metrics-workspace">
    <SearchToolbar keyword={recipe ? undefined : rawQuery} keywordPlaceholder={t(key('metricsPlaceholder'))} onKeywordChange={recipe ? undefined : (value) => { clear(); setRawQuery(value); }} onClearAll={() => { clear(); setRawQuery(''); }} hasActiveFilters={Boolean(rawQuery)}
      inlineFilters={<Select aria-label={t('business.observability.common.datasource')} value={sourceId} placeholder={t('business.observability.common.datasource')} style={{ width: 240 }} onChange={(id) => { clear(); setSourceId(id); setRawQuery(''); setForbidden(false); setRecipe(initialQueryRecipe(metricSources.find((item) => item.id === id)?.type === 'victoriametrics' ? 'victoriametrics' : 'prometheus')); }}>{metricSources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select>}
      extra={<Space wrap><Button type="primary" disabled={loading || forbidden || !source || (!recipe && !rawQuery.trim())} onClick={() => void run()}>{t(key('run'))}</Button><Button disabled={!loading} onClick={clear}>{t('business.observability.logs.stop')}</Button></Space>} />
    <QueryWorkspaceHistory selectionKey={JSON.stringify([rawQuery, time, limit])} sourceId={sourceId} kind={kind} sources={metricSources} recipe={recipe} onRecipeChange={(value) => { clear(); setRecipe(value); setRawQuery(''); if (value) setLimit(value.limit); }} onBeforeRestore={clear} onRestore={restore} onDenied={deny} epoch={epoch} forbidden={forbidden} />
    {!recipe && <div className="observability-workspace-controls"><Select aria-label={t(key('duration'))} value={time.mode === 'relative' ? time.duration : 'absolute'} style={{ width: 180 }} onChange={(duration) => { clear(); setTime({ mode: 'relative', duration: duration as QueryDuration }); }}>{time.mode === 'absolute' && <Select.Option value="absolute" disabled>{t(key('absoluteRestored'))}</Select.Option>}{['5m', '15m', '1h', '6h', '24h'].map((duration) => <Select.Option key={duration} value={duration}>{t(key(`duration.${duration}`))}</Select.Option>)}</Select></div>}
    <div className="observability-workspace-controls"><label>{t('business.observability.logs.limit')} <InputNumber aria-label={t('business.observability.logs.limit')} min={1} max={1000} value={limit} onChange={(value) => { if (Number.isInteger(value) && value >= 1 && value <= 1000) { clear(); setLimit(value); if (recipe) setRecipe({ ...recipe, limit: value }); } }} /></label><span className="observability-workspace-note">{t(key('metricsBudget'))}</span></div>
    {restored && <Alert type="info" content={t(key('restoredRun'))} />}
    {historyUnavailable && <Alert type="warning" content={t(key('historyUnavailable'))} />}
    {completed && completed.warningCount > 0 && <Alert type="warning" content={t(key('metricsPartial'), { count: completed.warningCount })} />}
    <Card className="page-panel system-list__table-card">
      {completed && <p className="observability-workspace-note">{t(key('metricsContext'), completed)}</p>}
      {loading && <PageLoading />}
      {forbidden && <PageEmpty description={t(key('forbidden'))} />}
      {error && !forbidden && <PageError onRetry={() => void run()} />}
      {!loading && !error && !forbidden && !rows.length && <PageEmpty description={t(key(!source ? 'metricsNoSource' : completed ? 'metricsEmpty' : 'metricsReady'))} />}
      {!loading && !error && !forbidden && rows.length > 0 && <AppTable rowKey="id" columns={columns} data={rows} pagination={buildStandardPagination(t, { pageSize: 20, total: rows.length })} scroll={{ x: 'max-content' }} />}
    </Card>
  </Space>;
}

export default function MonitoringCenter() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [targets, setTargets] = useState<MonitoringTargetRow[]>([]);
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [bizScopes, setBizScopes] = useState<BizScopeOptionItem[]>([]);
  const [targetsLoading, setTargetsLoading] = useState(false);
  const [datasourcesLoading, setDatasourcesLoading] = useState(false);
  const [scopesLoading, setScopesLoading] = useState(false);
  const [targetsError, setTargetsError] = useState<'denied' | 'unavailable' | null>(null);
  const [datasourcesError, setDatasourcesError] = useState<'denied' | 'unavailable' | null>(null);
  const [scopesError, setScopesError] = useState<'denied' | 'unavailable' | null>(null);
  const [targetVisible, setTargetVisible] = useState(false);
  const [datasourceVisible, setDatasourceVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [targetForm] = Form.useForm<TargetFormValues>();
  const [datasourceForm] = Form.useForm<DatasourcePayload>();
  const targetsRequest = useRef<AbortController | null>(null);
  const datasourcesRequest = useRef<AbortController | null>(null);
  const scopesRequest = useRef<AbortController | null>(null);
  const cancelLists = useCallback(() => {
    targetsRequest.current?.abort(); targetsRequest.current = null;
    datasourcesRequest.current?.abort(); datasourcesRequest.current = null;
    scopesRequest.current?.abort(); scopesRequest.current = null;
  }, []);
  const { epoch, snapshot, isCurrent } = useQueryWorkspaceIdentity(useCallback(() => {
    cancelLists(); setTargets([]); setDatasources([]); setBizScopes([]);
    setTargetsLoading(false); setDatasourcesLoading(false); setScopesLoading(false);
    setTargetsError(null); setDatasourcesError(null); setScopesError(null);
    setTargetVisible(false); setDatasourceVisible(false); setSubmitting(false); targetForm.resetFields(); datasourceForm.resetFields();
  }, [cancelLists, targetForm, datasourceForm]));

  const canCreate = hasPerm('business:observability:metric_source:create');
  const canDelete = hasPerm('business:observability:metric_source:delete');
  const canTest = hasPerm('business:observability:metric_source:update');

  const loadTargets = useCallback(async () => {
    targetsRequest.current?.abort();
    const controller = new AbortController(); targetsRequest.current = controller;
    const captured = snapshot();
    setTargetsLoading(true); setTargetsError(null); setTargets([]);
    try {
      const response = await getMonitoringTargetList(controller.signal);
      if (controller.signal.aborted || targetsRequest.current !== controller || !isCurrent(captured)) return;
      setTargets(response.items);
    } catch (requestError) {
      if (controller.signal.aborted || targetsRequest.current !== controller || !isCurrent(captured)) return;
      setTargetsError(isForbiddenRequestError(requestError) ? 'denied' : 'unavailable');
    } finally {
      if (targetsRequest.current === controller) { targetsRequest.current = null; setTargetsLoading(false); }
    }
  }, [snapshot, isCurrent]);
  const loadDatasources = useCallback(async () => {
    datasourcesRequest.current?.abort();
    const controller = new AbortController(); datasourcesRequest.current = controller;
    const captured = snapshot();
    setDatasourcesLoading(true); setDatasourcesError(null); setDatasources([]);
    try {
      const response = await getDatasourceList(controller.signal);
      if (controller.signal.aborted || datasourcesRequest.current !== controller || !isCurrent(captured)) return;
      setDatasources(response.items);
    } catch (requestError) {
      if (controller.signal.aborted || datasourcesRequest.current !== controller || !isCurrent(captured)) return;
      setDatasourcesError(isForbiddenRequestError(requestError) ? 'denied' : 'unavailable');
    } finally {
      if (datasourcesRequest.current === controller) { datasourcesRequest.current = null; setDatasourcesLoading(false); }
    }
  }, [snapshot, isCurrent]);
  const load = useCallback(async () => { await Promise.all([loadTargets(), loadDatasources()]); }, [loadTargets, loadDatasources]);
  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (active) void load(); });
    return () => { active = false; cancelLists(); };
  }, [load, epoch, cancelLists]);
  const loadScopes = useCallback(async () => {
    scopesRequest.current?.abort();
    const controller = new AbortController(); scopesRequest.current = controller;
    const captured = snapshot();
    setScopesLoading(true); setScopesError(null); setBizScopes([]);
    try {
      const response = await getBizScopeOptions(controller.signal);
      if (controller.signal.aborted || scopesRequest.current !== controller || !isCurrent(captured)) return;
      setBizScopes(response);
    } catch (requestError) {
      if (controller.signal.aborted || scopesRequest.current !== controller || !isCurrent(captured)) return;
      setScopesError(isForbiddenRequestError(requestError) ? 'denied' : 'unavailable');
    } finally {
      if (scopesRequest.current === controller) { scopesRequest.current = null; setScopesLoading(false); }
    }
  }, [snapshot, isCurrent]);
  const closeTarget = () => { scopesRequest.current?.abort(); scopesRequest.current = null; setTargetVisible(false); targetForm.resetFields(); setBizScopes([]); };

  const submitTarget = useCallback(async () => {
    if (scopesLoading || scopesError) return;
    const values = await targetForm.validate();
    setSubmitting(true);
    try {
      await createMonitoringTarget({ ...values, metricsPath: values.metricsPath || '/metrics' });
      Message.success(t('business.observability.monitoring.targetCreateSuccess'));
      setTargetVisible(false);
      targetForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.monitoring.targetCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [load, t, targetForm, scopesLoading, scopesError]);

  const submitDatasource = useCallback(async () => {
    const values = await datasourceForm.validate();
    setSubmitting(true);
    try {
      await createDatasource({ ...values, accessMode: 'proxy', timeout: values.timeout || 30 });
      Message.success(t('business.observability.monitoring.datasourceCreateSuccess'));
      setDatasourceVisible(false);
      datasourceForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.monitoring.datasourceCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [datasourceForm, load, t]);

  const targetColumns = useMemo<ColumnProps<MonitoringTargetRow>[]>(
    () => [
      { title: t('observability.metric_source.name'), dataIndex: 'targetName', width: 200 },
      { title: t('observability.metric_source.type'), dataIndex: 'targetType', width: 140 },
      { title: t('observability.metric_source.endpoint'), dataIndex: 'endpoint', width: 260, ellipsis: true },
      { title: t('business.observability.metricSource.metricsPath'), dataIndex: 'metricsPath', width: 150 },
      { title: t('observability.metric_source.status'), dataIndex: 'status', width: 110, render: (_: unknown, row) => <Tag color={row.status === 'up' ? 'green' : 'orange'}>{row.status}</Tag> },
      {
        title: t('common.action'),
        width: 90,
        fixed: 'right',
        render: (_: unknown, row) => canDelete ? (
          <Popconfirm title={t('observability.metric_source.delete_confirm')} onOk={async () => { await deleteMonitoringTarget(row.id); await load(); }}>
            <Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button>
          </Popconfirm>
        ) : null,
      },
    ],
    [canDelete, load, t],
  );

  const datasourceColumns = useMemo<ColumnProps<DatasourceRow>[]>(
    () => [
      { title: t('business.observability.monitoring.field.name'), dataIndex: 'name', width: 180 },
      { title: t('business.observability.monitoring.field.type'), dataIndex: 'type', width: 140, render: (value: string) => <Tag color={datasourceTypeColor(value)}>{value}</Tag> },
      { title: t('business.observability.monitoring.field.url'), dataIndex: 'url', width: 300, ellipsis: true },
      { title: t('business.observability.monitoring.field.status'), dataIndex: 'status', width: 120, render: (value: string) => <Tag color={datasourceStatusColor(value)}>{value}</Tag> },
      { title: t('business.observability.monitoring.field.default'), dataIndex: 'isDefault', width: 100, render: (value: boolean) => value ? <Tag color="arcoblue">{t('common.yes')}</Tag> : '-' },
      {
        title: t('common.action'),
        width: 180,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space>
            {canTest ? <Button type="text" icon={<IconRefresh />} onClick={async () => { await testDatasourceConnection(row.id); await load(); }}>{t('business.observability.monitoring.test')}</Button> : null}
            {canDelete ? (
              <Popconfirm title={t('business.observability.monitoring.datasourceDeleteConfirm')} onOk={async () => { await deleteDatasource(row.id); await load(); }}>
                <Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canTest, load, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.observability.monitoring.hero.eyebrow')}
          title={t('operations.observability.monitoring.menu')}
          description={t('business.observability.monitoring.hero.title')}
          metrics={[
            { key: 'targets', label: t('business.observability.monitoring.hero.targets'), value: targets.length },
            { key: 'healthy', label: t('business.observability.monitoring.hero.healthy'), value: targets.filter((item) => item.status === 'up').length },
            { key: 'datasources', label: t('business.observability.monitoring.hero.datasources'), value: datasources.length },
          ]}
        />
          <Tabs defaultActiveTab="targets">
            <TabPane key="queries" title={t('business.observability.workspace.metricsTab')}>
              {datasourcesLoading ? <PageLoading /> : datasourcesError === 'denied' ? <PageEmpty description={t('business.observability.workspace.forbidden')} /> : datasourcesError ? <PageError onRetry={() => void loadDatasources()} /> : <MetricsQueryWorkspace sources={datasources} />}
            </TabPane>
            <TabPane key="targets" title={t('business.observability.monitoring.targetsTab')}>
              <SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => { setTargetVisible(true); void loadScopes(); }}>{t('business.observability.monitoring.addTarget')}</Button> : undefined} />
              <Card className="page-panel system-list__table-card">
                {targetsLoading ? <PageLoading /> : targetsError === 'denied' ? <PageEmpty description={t('business.observability.workspace.forbidden')} /> : targetsError ? <PageError onRetry={() => void loadTargets()} /> : targets.length === 0 ? <PageEmpty description={t('business.observability.metricSource.empty')} /> : <AppTable rowKey="id" columns={targetColumns} data={targets} pagination={false} scroll={{ x: 'max-content' }} />}
              </Card>
            </TabPane>
            <TabPane key="datasources" title={t('business.observability.monitoring.datasourcesTab')}>
              <SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => setDatasourceVisible(true)}>{t('business.observability.monitoring.addDatasource')}</Button> : undefined} />
              <Card className="page-panel system-list__table-card">
                {datasourcesLoading ? <PageLoading /> : datasourcesError === 'denied' ? <PageEmpty description={t('business.observability.workspace.forbidden')} /> : datasourcesError ? <PageError onRetry={() => void loadDatasources()} /> : datasources.length === 0 ? <PageEmpty description={t('business.observability.monitoring.datasourceEmpty')} /> : <AppTable rowKey="id" columns={datasourceColumns} data={datasources} pagination={false} scroll={{ x: 'max-content' }} />}
              </Card>
            </TabPane>
          </Tabs>
      </Space>

      <AppModal visible={targetVisible} title={t('business.observability.monitoring.addTarget')} confirmLoading={submitting} okButtonProps={{ disabled: scopesLoading || Boolean(scopesError) }} onOk={() => void submitTarget()} onCancel={closeTarget}>
        {scopesLoading && <PageLoading />}
        {scopesError === 'denied' && <PageEmpty description={t('business.observability.workspace.forbidden')} />}
        {scopesError === 'unavailable' && <PageError onRetry={() => void loadScopes()} />}
        <Form form={targetForm} layout="vertical" initialValues={{ targetType: 'host', metricsPath: '/metrics' }}>
          <FormItem field="businessScopeId" label={t('business.observability.monitoring.field.scope')} rules={[{ required: true }]}><Select disabled={scopesLoading || Boolean(scopesError)}>{bizScopes.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem>
          <FormItem field="targetType" label={t('business.observability.monitoring.field.type')} rules={[{ required: true }]}><Select><Select.Option value="host">Host</Select.Option><Select.Option value="k8s_cluster">Kubernetes</Select.Option><Select.Option value="other">Other</Select.Option></Select></FormItem>
          <FormItem field="targetName" label={t('business.observability.monitoring.field.name')} rules={[{ required: true }]}><Input /></FormItem>
          <FormItem field="endpoint" label={t('business.observability.monitoring.field.endpoint')}><Input placeholder={t('business.observability.monitoring.field.endpointPlaceholder')} /></FormItem>
          <FormItem field="metricsPath" label={t('business.observability.metricSource.metricsPath')}><Input /></FormItem>
          <FormItem field="remark" label={t('common.remark')}><Input.TextArea /></FormItem>
        </Form>
      </AppModal>

      <AppModal visible={datasourceVisible} title={t('business.observability.monitoring.addDatasource')} confirmLoading={submitting} onOk={() => void submitDatasource()} onCancel={() => setDatasourceVisible(false)}>
        <Form form={datasourceForm} layout="vertical" initialValues={{ type: 'prometheus', timeout: 30, isDefault: false }}>
          <FormItem field="name" label={t('business.observability.monitoring.field.name')} rules={[{ required: true }]}><Input /></FormItem>
          <FormItem field="type" label={t('business.observability.monitoring.field.type')} rules={[{ required: true }]}><Select><Select.Option value="prometheus">Prometheus</Select.Option><Select.Option value="victoriametrics">VictoriaMetrics</Select.Option><Select.Option value="loki">Loki</Select.Option><Select.Option value="tempo">Tempo</Select.Option></Select></FormItem>
          <FormItem field="url" label={t('business.observability.monitoring.field.url')} rules={[{ required: true, type: 'url' }]}><Input placeholder={t('business.observability.monitoring.field.urlPlaceholder')} /></FormItem>
          <FormItem field="username" label={t('business.observability.monitoring.field.username')}><Input /></FormItem>
          <FormItem field="password" label={t('business.observability.monitoring.field.password')}><PasswordInput /></FormItem>
          <FormItem field="bearerToken" label={t('business.observability.monitoring.field.token')}><PasswordInput /></FormItem>
          <FormItem field="timeout" label={t('business.observability.monitoring.field.timeout')}><InputNumber min={1} max={120} /></FormItem>
          <FormItem field="isDefault" label={t('business.observability.monitoring.field.default')} triggerPropName="checked"><Switch /></FormItem>
        </Form>
      </AppModal>
    </PageContainer>
  );
}
