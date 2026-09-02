import { apiRequest } from '../../../api/request';

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

export const getDatasourceList = () => {
  return apiRequest<{ items: DatasourceRow[]; total: number }>({
    url: '/business/observability/datasources',
    method: 'GET',
  });
};

export const createDatasource = (data: DatasourcePayload) => {
  return apiRequest<DatasourceRow>({
    url: '/business/observability/datasources',
    method: 'POST',
    data,
  });
};

export const getMonitoringTargetList = () => {
  return apiRequest<{ items: MonitoringTargetRow[]; total: number; page: number; pageSize: number }>({
    url: '/business/observability/targets',
    method: 'GET',
    params: { page: 1, pageSize: 100 },
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

export const queryLogs = (datasourceId: number, params: { query: string; start?: string; end?: string; limit?: number }) => {
  return apiRequest<{ status: string; data: { resultType: string; result: LokiStreamResult[] } }>({
    url: `/business/observability/datasources/${datasourceId}/logs`,
    method: 'GET',
    params,
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

export const searchTraces = (datasourceId: number, params: { query: string; start?: string; end?: string; limit?: number }) => {
  return apiRequest<{ traces?: TempoTraceSummary[] }>({
    url: `/business/observability/datasources/${datasourceId}/traces`,
    method: 'GET',
    params,
  });
};

export const getTrace = (datasourceId: number, traceId: string) => {
  return apiRequest<Record<string, unknown>>({
    url: `/business/observability/datasources/${datasourceId}/traces/${encodeURIComponent(traceId)}`,
    method: 'GET',
  });
};

export const getAlertRuleList = (params: {
  keyword?: string;
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
