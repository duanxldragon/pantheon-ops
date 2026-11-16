import { apiRequest, downloadFile, uploadImportFile } from '@duanxldragon/pantheon-base-ui';

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

export function getBizScopeOptions(signal?: AbortSignal) {
  return apiRequest<BizScopeOptionItem[]>({
    url: '/business/bizscope/options',
    method: 'get',
    signal,
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

export interface BizScopeQuery {
  keyword?: string;
  environment?: string;
  status?: string;
}

export function exportBizScopes(params?: BizScopeQuery) {
  return downloadFile({
    url: '/business/bizscope/export',
    method: 'get',
    params: params as Record<string, unknown> | undefined,
    filename: 'bizscope-export.csv',
  });
}

export function downloadBizScopeImportTemplate() {
  return downloadFile({
    url: '/business/bizscope/import-template',
    method: 'get',
    filename: 'bizscope-import-template.csv',
  });
}

export function importBizScopes(file: File) {
  return uploadImportFile('/business/bizscope/import', file);
}

export interface EnvironmentRow {
  id: number;
  businessScopeId: number;
  code: string;
  name: string;
  envType: string;
  status: string;
  remark?: string;
  createdAt: string;
}

export interface EnvironmentListResp {
  items: EnvironmentRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface EnvironmentListQuery {
  code?: string;
  name?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export interface EnvironmentPayload {
  code: string;
  name: string;
  envType: string;
  status: string;
  remark?: string;
}

export interface EnvironmentOption {
  label: string;
  value: string;
  code: string;
  name: string;
}

export function getEnvironmentList(scopeId: number, params?: EnvironmentListQuery) {
  return apiRequest<EnvironmentListResp>({
    url: `/business/bizscope/${scopeId}/environments`,
    method: 'get',
    params,
  });
}

export function getEnvironment(scopeId: number, id: number) {
  return apiRequest<EnvironmentRow>({
    url: `/business/bizscope/${scopeId}/environments/${id}`,
    method: 'get',
  });
}

export function getEnvironmentOptions(scopeId: number, activeOnly = false) {
  return apiRequest<EnvironmentOption[]>({
    url: `/business/bizscope/${scopeId}/environments/options`,
    method: 'get',
    params: activeOnly ? { activeOnly: true } : undefined,
  });
}

export function createEnvironment(scopeId: number, data: EnvironmentPayload) {
  return apiRequest<EnvironmentRow>({
    url: `/business/bizscope/${scopeId}/environments`,
    method: 'post',
    data,
  });
}

export function updateEnvironment(scopeId: number, id: number, data: Partial<EnvironmentPayload>) {
  return apiRequest<EnvironmentRow>({
    url: `/business/bizscope/${scopeId}/environments/${id}`,
    method: 'put',
    data,
  });
}

export function deleteEnvironment(scopeId: number, id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/bizscope/${scopeId}/environments/${id}`,
    method: 'delete',
  });
}
