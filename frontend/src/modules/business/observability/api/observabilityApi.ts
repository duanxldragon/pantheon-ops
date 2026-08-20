import { api } from '@/shared/utils/apiClient';

export interface MetricSource {
  id: number;
  name: string;
  type: string;
  endpoint: string;
  credentialRef?: string;
  businessScopeId?: number;
  deptId?: number;
  status: string;
  config?: Record<string, any>;
  remark?: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  updatedBy?: string;
}

export interface AlertRule {
  id: number;
  metricSourceId: number;
  name: string;
  businessScopeId?: number;
  deptId?: number;
  environment?: string;
  promql: string;
  duration?: string;
  severity: string;
  labels?: Record<string, any>;
  annotations?: Record<string, any>;
  notificationChannelIds?: number[];
  status: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  updatedBy?: string;
}

export interface AlertRecord {
  id: number;
  alertRuleId: number;
  alertRuleName?: string;
  firedAt: string;
  resolvedAt?: string;
  severity: string;
  labels?: Record<string, any>;
  annotations?: Record<string, any>;
  notificationsSent?: Record<string, any>;
  createdAt: string;
  updatedAt: string;
}

export interface NotificationChannel {
  id: number;
  name: string;
  type: string;
  config: Record<string, any>;
  businessScopeId?: number;
  deptId?: number;
  status: string;
  remark?: string;
  createdAt: string;
  updatedAt: string;
  createdBy?: string;
  updatedBy?: string;
}

export interface PaginationParams {
  page: number;
  pageSize: number;
  businessScopeId?: number;
  status?: string;
  type?: string;
  environment?: string;
  severity?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
}

const observabilityApi = {
  // Metric Sources
  getMetricSources: (params: PaginationParams) =>
    api.get<PaginatedResponse<MetricSource>>('/v1/observability/metrics/sources', { params }),

  getMetricSource: (id: number) =>
    api.get<MetricSource>(`/v1/observability/metrics/sources/${id}`),

  createMetricSource: (data: Partial<MetricSource>) =>
    api.post<MetricSource>('/v1/observability/metrics/sources', data),

  updateMetricSource: (id: number, data: Partial<MetricSource>) =>
    api.put<MetricSource>(`/v1/observability/metrics/sources/${id}`, data),

  deleteMetricSource: (id: number) =>
    api.delete(`/v1/observability/metrics/sources/${id}`),

  // Alert Rules
  getAlertRules: (params: PaginationParams) =>
    api.get<PaginatedResponse<AlertRule>>('/v1/observability/alerts/rules', { params }),

  getAlertRule: (id: number) =>
    api.get<AlertRule>(`/v1/observability/alerts/rules/${id}`),

  createAlertRule: (data: Partial<AlertRule>) =>
    api.post<AlertRule>('/v1/observability/alerts/rules', data),

  updateAlertRule: (id: number, data: Partial<AlertRule>) =>
    api.put<AlertRule>(`/v1/observability/alerts/rules/${id}`, data),

  deleteAlertRule: (id: number) =>
    api.delete(`/v1/observability/alerts/rules/${id}`),

  validatePromQL: (promql: string) =>
    api.post<{ valid: boolean; message: string }>('/v1/observability/alerts/rules/validate', { promql }),

  // Alert Records
  getAlertRecords: (params: PaginationParams & { alertRuleId?: number; resolved?: boolean }) =>
    api.get<PaginatedResponse<AlertRecord>>('/v1/observability/alerts/records', { params }),

  getActiveAlerts: () =>
    api.get<PaginatedResponse<AlertRecord>>('/v1/observability/alerts/active'),

  // Notification Channels
  getNotificationChannels: (params: PaginationParams) =>
    api.get<PaginatedResponse<NotificationChannel>>('/v1/observability/alerts/channels', { params }),

  getNotificationChannel: (id: number) =>
    api.get<NotificationChannel>(`/v1/observability/alerts/channels/${id}`),

  createNotificationChannel: (data: Partial<NotificationChannel>) =>
    api.post<NotificationChannel>('/v1/observability/alerts/channels', data),

  updateNotificationChannel: (id: number, data: Partial<NotificationChannel>) =>
    api.put<NotificationChannel>(`/v1/observability/alerts/channels/${id}`, data),

  deleteNotificationChannel: (id: number) =>
    api.delete(`/v1/observability/alerts/channels/${id}`),

  testNotificationChannel: (id: number) =>
    api.post(`/v1/observability/alerts/channels/${id}/test`),
};

export default observabilityApi;
