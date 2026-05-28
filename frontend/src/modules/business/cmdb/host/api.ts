import { apiRequest } from '../../../../api/request';

export interface LabelEntry {
  key: string;
  val: string;
}

export interface ComponentEntry {
  name: string;
  version: string;
  deployedAt?: string;
  deployTaskId?: number;
  deployTaskName?: string;
  executorType?: string;
}

export interface MatchedGroupEntry {
  id: number;
  parentId: number;
  name: string;
  fullPath: string;
}

export interface HostRow {
  id: number;
  hostname: string;
  ip: string;
  sshPort: number;
  os: string;
  osVersion: string;
  cpuCores: number;
  memoryGb: number;
  diskGb: number;
  labelValues: LabelEntry[];
  installedComponents: ComponentEntry[];
  matchedGroups: MatchedGroupEntry[];
  matchedGroupCount: number;
  status: string;
  businessScopeId: number;
  businessScopeCode: string;
  businessScopeName: string;
  deptId: number;
  owner: string;
  remark: string;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  updatedBy: string;
}

export interface HostListQuery {
  page?: number;
  pageSize?: number;
  keyword?: string;
  status?: string;
  os?: string;
  businessScopeId?: number;
}

export interface HostListResp {
  items: HostRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface CreateHostPayload {
  hostname: string;
  ip: string;
  sshPort?: number;
  os: string;
  osVersion?: string;
  cpuCores?: number;
  memoryGb?: number;
  diskGb?: number;
  labels?: LabelEntry[];
  businessScopeId?: number;
  deptId?: number;
  owner?: string;
  remark?: string;
}

export interface UpdateHostPayload {
  hostname?: string;
  ip?: string;
  sshPort?: number;
  os?: string;
  osVersion?: string;
  cpuCores?: number;
  memoryGb?: number;
  diskGb?: number;
  labels?: LabelEntry[];
  businessScopeId?: number;
  deptId?: number;
  owner?: string;
  remark?: string;
}

export interface CollectPayload {
  sshUser: string;
  sshPassword?: string;
  sshPrivateKey?: string;
  hostFingerprint: string;
  authMode: 'password' | 'private_key';
}

export interface UpdateStatusPayload {
  status: string;
}

export function getHostList(params?: HostListQuery) {
  return apiRequest<HostListResp>({
    url: '/business/cmdb/hosts',
    method: 'get',
    params,
  });
}

export function getHostDetail(id: number) {
  return apiRequest<HostRow>({
    url: `/business/cmdb/hosts/${id}`,
    method: 'get',
  });
}

export function createHost(data: CreateHostPayload) {
  return apiRequest<HostRow>({
    url: '/business/cmdb/hosts',
    method: 'post',
    data,
  });
}

export function updateHost(id: number, data: UpdateHostPayload) {
  return apiRequest<HostRow>({
    url: `/business/cmdb/hosts/${id}`,
    method: 'put',
    data,
  });
}

export function deleteHost(id: number) {
  return apiRequest<void>({
    url: `/business/cmdb/hosts/${id}`,
    method: 'delete',
  });
}

export function collectHostConfig(id: number, data: CollectPayload) {
  return apiRequest<HostRow>({
    url: `/business/cmdb/hosts/${id}/collect`,
    method: 'post',
    data,
  });
}
