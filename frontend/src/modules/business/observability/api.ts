import { apiRequest } from '@duanxldragon/pantheon-base-ui';

export interface DatasourceRow {
  id: number;
  name: string;
  type: string;
  url: string;
  status: string;
  isDefault: boolean;
  lastChecked?: string;
  createdAt: string;
}

export interface DatasourcePayload {
  name: string;
  type: 'prometheus' | 'victoriametrics' | 'loki' | 'tempo';
  url: string;
  accessMode?: 'proxy' | 'direct';
  username?: string;
  password?: string;
  bearerToken?: string;
  timeout?: number;
  isDefault?: boolean;
}

export interface MonitoringTargetRow {
  id: number;
  businessScopeId: number;
  deptId: number;
  targetType: string;
  targetName: string;
  endpoint: string;
  metricsPath: string;
  enabled: boolean;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface AlertRuleRow {
  id: number;
  name: string;
  datasourceId: number;
  datasourceName?: string;
  enabled: boolean;
  query: string;
  interval: number;
  forDuration: number;
  severity: string;
  firedCount: number;
  lastFired?: string;
  createdAt: string;
}

export interface AlertEventRow {
  id: number;
  ruleId: number;
  ruleName: string;
  severity: string;
  status: string;
  value: number;
  firedAt: string;
  resolvedAt?: string;
}

export interface NotificationChannelRow {
  id: number;
  name: string;
  type: string;
  enabled: boolean;
  status: string;
  createdAt: string;
}

export const getDatasourceList = (signal?: AbortSignal) => {
  return apiRequest<{ items: DatasourceRow[]; total: number }>({
    url: '/business/observability/datasources',
    method: 'GET',
    skipErrorMessage: true,
    signal,
  });
};

export const createDatasource = (data: DatasourcePayload) => {
  return apiRequest<DatasourceRow>({
    url: '/business/observability/datasources',
    method: 'POST',
    data,
  });
};

export const getMonitoringTargetList = (signal?: AbortSignal) => {
  return apiRequest<{ items: MonitoringTargetRow[]; total: number; page: number; pageSize: number }>({
    url: '/business/observability/targets',
    method: 'GET',
    params: { page: 1, pageSize: 100 },
    signal,
  });
};

export const createMonitoringTarget = (data: {
  businessScopeId: number;
  targetType: 'host' | 'k8s_cluster' | 'other';
  targetName: string;
  endpoint?: string;
  metricsPath?: string;
  remark?: string;
}) => {
  return apiRequest<MonitoringTargetRow>({
    url: '/business/observability/targets',
    method: 'POST',
    data,
  });
};

export const deleteMonitoringTarget = (id: number) => {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/observability/targets/${id}`,
    method: 'DELETE',
  });
};

export interface LokiStreamResult {
  stream: Record<string, string>;
  values: [string, string][];
}

export const queryLogs = (datasourceId: number, data: RunQueryRequest, signal?: AbortSignal) => {
  return apiRequest<QueryRunResponse<LokiQueryResult>>({
    url: `/business/observability/datasources/${datasourceId}/logs`,
    method: 'POST',
    skipErrorMessage: true,
    data,
    timeout: 30000,
    signal,
  });
};

export interface TempoTraceSummary {
  traceID: string;
  rootServiceName?: string;
  rootTraceName?: string;
  startTimeUnixNano?: string;
  durationMs?: number;
  durationNano?: string;
}

export const searchTraces = (datasourceId: number, data: RecipeQueryRequest, signal?: AbortSignal) => {
  return apiRequest<QueryRunResponse<{ traces?: TempoTraceSummary[] }>>({
    url: `/business/observability/datasources/${datasourceId}/traces`,
    method: 'POST',
    skipErrorMessage: true,
    data,
    timeout: 30000,
    signal,
  });
};

export const getTrace = (datasourceId: number, traceId: string, signal?: AbortSignal) => {
  return apiRequest<Record<string, unknown>>({
    url: `/business/observability/datasources/${datasourceId}/traces/${encodeURIComponent(traceId)}`,
    method: 'GET',
    skipErrorMessage: true,
    signal,
  });
};

export const getAlertRuleList = (params: {
  keyword?: string;
  status?: string;
  datasourceId?: number;
  severity?: string;
  enabled?: boolean;
  page?: number;
  pageSize?: number;
}) => {
  return apiRequest<{ items: AlertRuleRow[]; total: number; page: number; pageSize: number }>({
    url: '/business/observability/alert-rules',
    method: 'GET',
    params,
  });
};

export const createAlertRule = (data: {
  name: string;
  datasourceId: number;
  enabled: boolean;
  query: string;
  interval: number;
  forDuration: number;
  severity: 'critical' | 'warning' | 'info';
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  notificationChannels?: number[];
}) => {
  return apiRequest<AlertRuleRow>({ url: '/business/observability/alert-rules', method: 'POST', data });
};

export const getAlertEventList = (params: {
  ruleId?: number;
  severity?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}) => {
  return apiRequest<{ items: AlertEventRow[]; total: number; page: number; pageSize: number }>({
    url: '/business/observability/alert-events',
    method: 'GET',
    params,
  });
};

export const getNotificationChannelList = () => {
  return apiRequest<{ items: NotificationChannelRow[]; total: number }>({
    url: '/business/observability/notification-channels',
    method: 'GET',
  });
};

export const createNotificationChannel = (data: {
  name: string;
  type: 'email' | 'webhook' | 'slack' | 'dingtalk' | 'feishu';
  enabled: boolean;
  config: Record<string, unknown>;
}) => {
  return apiRequest<NotificationChannelRow>({ url: '/business/observability/notification-channels', method: 'POST', data });
};

export const testDatasourceConnection = (id: number) => {
  return apiRequest({
    url: `/business/observability/datasources/${id}/test`,
    method: 'POST',
  });
};

export const deleteDatasource = (id: number) => {
  return apiRequest({
    url: `/business/observability/datasources/${id}`,
    method: 'DELETE',
  });
};

export const toggleAlertRule = (id: number, enabled: boolean) => {
  return apiRequest({
    url: `/business/observability/alert-rules/${id}/toggle`,
    method: 'PATCH',
    data: { enabled },
  });
};

export const deleteAlertRule = (id: number) => {
  return apiRequest({
    url: `/business/observability/alert-rules/${id}`,
    method: 'DELETE',
  });
};

export const deleteNotificationChannel = (id: number) => {
  return apiRequest({
    url: `/business/observability/notification-channels/${id}`,
    method: 'DELETE',
  });
};

export type QuerySourceKind = 'prometheus' | 'victoriametrics' | 'loki' | 'tempo';
export type QueryDuration = '5m' | '15m' | '1h' | '6h' | '24h';
export type QueryTime = { mode: 'relative'; duration: QueryDuration } | { mode: 'absolute'; start: string; end: string };
export interface QueryRecipe {
  version: 1;
  sourceKind: QuerySourceKind;
  rules: { field: string; op: string; value?: string }[];
  time: QueryTime;
  limit: number;
  resultMode: 'table';
}
export interface QueryCatalog {
  version: number;
  sourceKind: QuerySourceKind;
  fields: { field: string; ops: string[]; values?: string[] }[];
  durations: string[];
  maxLimit: number;
  resultModes: string[];
}
interface QueryHistoryMode { historyMode?: 'record' | 'coalesced'; historyId?: number }
export type RecipeQueryRequest = QueryHistoryMode & { recipe: QueryRecipe };
export type RunQueryRequest = RecipeQueryRequest | (QueryHistoryMode & { rawQuery: string; time: QueryTime; limit: number; resultMode: 'table' });
export interface QueryRunResponse<R> { result: R; historyUnavailable: boolean; historyId?: number }
export interface LokiQueryResult { status: string; warnings?: string[]; data: { resultType: string; result: LokiStreamResult[] } }
export interface QueryHistoryRow {
  id: number;
  datasourceId: number;
  sourceKind: QuerySourceKind;
  time: QueryTime;
  limit: number;
  resultMode: 'table';
  outcome: string;
  durationMs: number;
  createdAt: string;
  recipe?: QueryRecipe;
  expressionUnavailable: boolean;
}
export interface SavedViewRow { id: number; datasourceId: number; revision: number; recipe: QueryRecipe; updatedAt: string }
export interface QueryWorkspacePage<T> { items: T[]; total: number; catalogVersion: number }

const workspacePrefix = '/business/observability';
export const getQueryCatalog = (id: number, signal?: AbortSignal) => apiRequest<QueryCatalog>({ url: `${workspacePrefix}/datasources/${id}/query-fields`, method: 'GET', signal, skipErrorMessage: true });
export const queryMetrics = (id: number, data: RunQueryRequest, signal?: AbortSignal) => apiRequest<QueryRunResponse<{ status: string; warnings?: string[]; data: { resultType: string; result: { metric: Record<string, string>; value?: [number, string] }[] } }>>({ url: `${workspacePrefix}/datasources/${id}/query`, method: 'POST', data, signal, timeout: 30000, skipErrorMessage: true });
export const getQueryHistory = (page: number, signal?: AbortSignal) => apiRequest<QueryWorkspacePage<QueryHistoryRow>>({ url: `${workspacePrefix}/query-history`, method: 'GET', params: { page, pageSize: 20 }, signal, skipErrorMessage: true });
export const getSavedViews = (page: number, signal?: AbortSignal) => apiRequest<QueryWorkspacePage<SavedViewRow>>({ url: `${workspacePrefix}/saved-views`, method: 'GET', params: { page, pageSize: 20 }, signal, skipErrorMessage: true });
export const restoreQueryHistory = (id: number, signal?: AbortSignal) => apiRequest<QueryWorkspacePage<QueryHistoryRow>>({ url: `${workspacePrefix}/query-history/${id}/restore`, method: 'POST', data: {}, signal, skipErrorMessage: true });
export const restoreSavedView = (id: number, signal?: AbortSignal) => apiRequest<QueryWorkspacePage<SavedViewRow>>({ url: `${workspacePrefix}/saved-views/${id}/restore`, method: 'POST', data: {}, signal, skipErrorMessage: true });
export const deleteQueryHistory = (id?: number, signal?: AbortSignal) => apiRequest<{ deleted: boolean }>({ url: `${workspacePrefix}/query-history${id ? `/${id}` : ''}`, method: 'DELETE', signal, skipErrorMessage: true });
export const deleteSavedView = (id: number, signal?: AbortSignal) => apiRequest<{ deleted: boolean }>({ url: `${workspacePrefix}/saved-views/${id}`, method: 'DELETE', signal, skipErrorMessage: true });
export const saveQueryView = (datasourceId: number, recipe: QueryRecipe, signal?: AbortSignal, existing?: Pick<SavedViewRow, 'id' | 'revision'>) => apiRequest<SavedViewRow>({ url: `${workspacePrefix}/saved-views${existing ? `/${existing.id}` : ''}`, method: existing ? 'PUT' : 'POST', data: { datasourceId, recipe, ...(existing ? { expectedRevision: existing.revision } : {}) }, signal, skipErrorMessage: true });
