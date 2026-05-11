import { apiRequest } from '../../../api/request';

export interface DeployPackageRow {
  id: number;
  name: string;
  version: string;
  description: string;
  installCommand: string;
  uninstallCommand: string;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface DeployPackageListResp {
  items: DeployPackageRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeployPackagePayload {
  name: string;
  version: string;
  description?: string;
  installCommand?: string;
  uninstallCommand?: string;
  status?: string;
}

export interface DeployTaskHostRow {
  id: number;
  taskId: number;
  hostId: number;
  hostname: string;
  hostIp: string;
  os: string;
  status: string;
  stdout: string;
  stderr: string;
  errorMessage: string;
  executorId: string;
  startedAt?: string;
  finishedAt?: string;
  reportedAt?: string;
  updatedAt: string;
}

export interface DeployTaskRow {
  id: number;
  name: string;
  packageId: number;
  packageName: string;
  packageVersion: string;
  targetType: 'host' | 'group';
  targetIds: number[];
  executorType: 'manual' | 'simulated' | 'agent' | 'ssh';
  status: string;
  remark: string;
  externalTaskId: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt: string;
  updatedAt: string;
  hosts: DeployTaskHostRow[];
}

export interface DeployTaskListResp {
  items: DeployTaskRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeployTaskPayload {
  name: string;
  packageId: number;
  targetType: 'host' | 'group';
  targetIds: number[];
  executorType: 'manual' | 'simulated' | 'agent' | 'ssh';
  remark?: string;
}

export interface MarkHostResultPayload {
  status: 'success' | 'failed' | 'skipped';
  stdout?: string;
  stderr?: string;
  errorMessage?: string;
  executorId?: string;
}

type ListQuery = Record<string, string | number | undefined>;

export function getDeployPackageList(params?: ListQuery) {
  return apiRequest<DeployPackageListResp>({
    url: '/business/deploy/packages',
    method: 'get',
    params,
  });
}

export function createDeployPackage(data: DeployPackagePayload) {
  return apiRequest<DeployPackageRow>({
    url: '/business/deploy/packages',
    method: 'post',
    data,
  });
}

export function updateDeployPackage(id: number, data: Partial<DeployPackagePayload>) {
  return apiRequest<DeployPackageRow>({
    url: `/business/deploy/packages/${id}`,
    method: 'put',
    data,
  });
}

export function deleteDeployPackage(id: number) {
  return apiRequest<void>({
    url: `/business/deploy/packages/${id}`,
    method: 'delete',
  });
}

export function getDeployTaskList(params?: ListQuery) {
  return apiRequest<DeployTaskListResp>({
    url: '/business/deploy/tasks',
    method: 'get',
    params,
  });
}

export function getDeployTaskDetail(id: number) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}`,
    method: 'get',
  });
}

export function createDeployTask(data: DeployTaskPayload) {
  return apiRequest<DeployTaskRow>({
    url: '/business/deploy/tasks',
    method: 'post',
    data,
  });
}

export function startDeployTask(id: number) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}/start`,
    method: 'post',
  });
}

export function cancelDeployTask(id: number) {
  return apiRequest<DeployTaskRow>({
    url: `/business/deploy/tasks/${id}/cancel`,
    method: 'post',
  });
}

export function markDeployHostResult(id: number, data: MarkHostResultPayload) {
  return apiRequest<DeployTaskHostRow>({
    url: `/business/deploy/task-hosts/${id}/result`,
    method: 'post',
    data,
  });
}
