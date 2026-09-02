import { apiRequest } from '../../../api/request';

export interface ApplicationRow {
  id: number;
  code: string;
  name: string;
  businessScopeId: number;
  businessScopeName: string;
  deptId: number;
  status: string;
  owner: string;
  remark: string;
  serviceCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface ApplicationListResponse {
  items: ApplicationRow[];
  total: number;
  page: number;
  pageSize: number;
}

export function getApplicationList(params?: { keyword?: string; status?: string; page?: number; pageSize?: number }) {
  return apiRequest<ApplicationListResponse>({ url: '/business/service/applications', method: 'get', params });
}

export function createApplication(data: { code: string; name: string; businessScopeId: number; status?: string; owner?: string; remark?: string }) {
  return apiRequest<ApplicationRow>({ url: '/business/service/applications', method: 'post', data });
}

export function updateApplication(id: number, data: { name?: string; status?: string; owner?: string; remark?: string }) {
  return apiRequest<ApplicationRow>({ url: `/business/service/applications/${id}`, method: 'put', data });
}

export function deleteApplication(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/service/applications/${id}`, method: 'delete' });
}

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

export function createService(data: { applicationId: number; code: string; name: string; runtimeType: string; description?: string; status?: string }) {
  return apiRequest<ServiceRow>({ url: '/business/service/services', method: 'post', data });
}

export function updateService(id: number, data: { name?: string; runtimeType?: string; description?: string; status?: string }) {
  return apiRequest<ServiceRow>({ url: `/business/service/services/${id}`, method: 'put', data });
}

export function deleteService(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/service/services/${id}`, method: 'delete' });
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

export function createServiceInstance(data: {
  serviceId: number;
  environment: string;
  targetType: 'vm' | 'k8s';
  hostId?: number;
  k8sClusterId?: number;
  namespace?: string;
  workloadKind?: string;
  workloadName?: string;
  desiredVersion?: string;
  status?: string;
}) {
  return apiRequest<ServiceInstanceRow>({ url: '/business/service/instances', method: 'post', data });
}

export function deleteServiceInstance(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/service/instances/${id}`, method: 'delete' });
}
