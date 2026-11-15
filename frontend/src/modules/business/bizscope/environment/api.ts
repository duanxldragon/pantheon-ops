import { apiRequest } from '@duanxldragon/pantheon-base-ui';

export interface BusinessEnvironment {
  id: number;
  businessScopeId: number;
  businessScopeName: string;
  code: string;
  name: string;
  envType: 'dev' | 'test' | 'prod';
  status: 'active' | 'inactive';
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface EnvironmentPayload {
  businessScopeId: number;
  code: string;
  name: string;
  envType: BusinessEnvironment['envType'];
  status: BusinessEnvironment['status'];
  remark?: string;
}

export interface EnvironmentQuery {
  businessScopeId?: number;
  envType?: string;
  status?: string;
  keyword?: string;
  page?: number;
  pageSize?: number;
}

export function getEnvironmentList(params: EnvironmentQuery, signal?: AbortSignal) {
  return apiRequest<{ items: BusinessEnvironment[]; total: number; page: number; pageSize: number }>({
    url: '/business/bizscope/environments', method: 'get', params, signal,
  });
}

export function createEnvironment(data: EnvironmentPayload) {
  return apiRequest<BusinessEnvironment>({ url: '/business/bizscope/environments', method: 'post', data });
}

export function updateEnvironment(id: number, data: Partial<EnvironmentPayload>) {
  return apiRequest<BusinessEnvironment>({ url: `/business/bizscope/environments/${id}`, method: 'put', data });
}

export function deleteEnvironment(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/bizscope/environments/${id}`, method: 'delete' });
}
