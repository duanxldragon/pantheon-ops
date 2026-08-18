import { apiRequest } from '../../../api/request';

export interface ServiceRow {
  id: number;
  applicationId: number;
  applicationCode: string;
  applicationName: string;
  businessScopeId: number;
  code: string;
  name: string;
  runtimeType: string;
  description: string;
  status: string;
  instanceCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface ServiceInstanceRow {
  id: number;
  serviceId: number;
  serviceCode: string;
  serviceName: string;
  environment: string;
  targetType: string;
  hostId: number;
  k8sClusterId: number;
  namespace: string;
  workloadKind: string;
  workloadName: string;
  desiredVersion: string;
  currentVersion: string;
  rollbackVersion: string;
  desiredState: string;
  observedState: string;
  healthState: string;
  healthMessage: string;
  healthRevision: string;
  lifecycleVersion: number;
  status: string;
}

export interface ServiceInstanceListResponse {
  items: ServiceInstanceRow[];
  total: number;
  page: number;
  pageSize: number;
}

export function getServiceInstanceList(params?: { serviceId?: number; page?: number; pageSize?: number }) {
  return apiRequest<ServiceInstanceListResponse>({
    url: '/business/service/instances',
    method: 'get',
    params,
  });
}

export function reconcileServiceInstance(id: number, maxAgeSeconds = 900) {
  return apiRequest<ServiceInstanceRow>({
    url: `/business/service/instances/${id}/reconcile`,
    method: 'post',
    data: { maxAgeSeconds },
  });
}

export interface ServiceListQuery {
  keyword?: string;
  applicationId?: number;
  status?: string;
  page?: number;
  pageSize?: number;
}

export interface ServiceListResponse {
  items: ServiceRow[];
  total: number;
  page: number;
  pageSize: number;
}

export function getServiceList(params?: ServiceListQuery) {
  return apiRequest<ServiceListResponse>({
    url: '/business/service/services',
    method: 'get',
    params,
  });
}

export interface ServiceOption {
  id: number;
  code: string;
  name: string;
  label: string;
  value: number;
}

export function getServiceOptions(applicationId?: number) {
  return apiRequest<ServiceOption[]>({
    url: '/business/service/services/options',
    method: 'get',
    params: applicationId ? { applicationId } : undefined,
  });
}
