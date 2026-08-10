import { apiRequest } from '../../../api/request';

export interface BizScopeRow {
  id: number;
  code: string;
  name: string;
  owner?: string;
  environment: string;
  status: string;
  remark?: string;
  createdAt: string;
}

export interface BizScopeDetail extends BizScopeRow {
  hostCount: number;
  updatedAt: string;
}

export interface BizScopeHostRow {
  id: number;
  hostname: string;
  ip: string;
  os: string;
  status: string;
  businessScopeId: number;
  businessScopeName: string;
}

export interface BizScopeHostListResp {
  items: BizScopeHostRow[];
  total: number;
}

export interface BizScopeListQuery {
  code?: string;
  name?: string;
  owner?: string;
  environment?: string;
  status?: string;
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface BizScopeListResp {
  items: BizScopeRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface BizScopePayload {
  code: string;
  name: string;
  owner?: string;
  environment: string;
  status: string;
  remark?: string;
}

export interface BizScopeOptionItem {
  label: string;
  value: number;
  id: number;
  name: string;
}

export function getBizScopeList(params?: BizScopeListQuery) {
  return apiRequest<BizScopeListResp>({
    url: '/business/bizscope/list',
    method: 'get',
    params,
  });
}

export function getBizScopeDetail(id: number) {
  return apiRequest<BizScopeDetail>({
    url: `/business/bizscope/${id}`,
    method: 'get',
  });
}

export function getBizScopeHosts(id: number) {
  return apiRequest<BizScopeHostListResp>({
    url: `/business/bizscope/${id}/hosts`,
    method: 'get',
  });
}

export function getBizScopeAvailableHosts(id: number) {
  return apiRequest<BizScopeHostListResp>({
    url: `/business/bizscope/${id}/available-hosts`,
    method: 'get',
  });
}

export function bindBizScopeHosts(id: number, hostIds: number[]) {
  return apiRequest<{ bound: boolean }>({
    url: `/business/bizscope/${id}/hosts/bind`,
    method: 'post',
    data: { hostIds },
  });
}

export function unbindBizScopeHost(id: number, hostId: number) {
  return apiRequest<{ unbound: boolean }>({
    url: `/business/bizscope/${id}/hosts/${hostId}`,
    method: 'delete',
  });
}

export function getBizScopeOptions() {
  return apiRequest<BizScopeOptionItem[]>({
    url: '/business/bizscope/options',
    method: 'get',
  });
}

export function createBizScope(data: BizScopePayload) {
  return apiRequest<BizScopeRow>({
    url: '/business/bizscope',
    method: 'post',
    data,
  });
}

export function updateBizScope(id: number, data: Partial<BizScopePayload>) {
  return apiRequest<BizScopeRow>({
    url: `/business/bizscope/${id}`,
    method: 'put',
    data,
  });
}

export function deleteBizScope(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/bizscope/${id}`,
    method: 'delete',
  });
}
