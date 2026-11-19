import { apiRequest } from '@duanxldragon/pantheon-base-ui';
import type { DeployTargetSelection } from '../deploy/api';

export interface FileRecordRow {
  id: number;
  storageId: number;
  storageName?: string;
  name: string;
  fileName?: string;
  displayName?: string;
  path?: string;
  filePath?: string;
  size: number;
  fileSize?: number;
  md5?: string;
  sha256?: string;
  contentType?: string;
  storageType?: string;
  status?: string;
  uploadedBy?: string;
  uploadedAt?: string;
  createdAt?: string;
  createdBy?: string;
}

export interface FileListResponse {
  items: FileRecordRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DistributionTaskRow {
  id: number;
  name: string;
  fileId: number;
  fileName?: string;
  status: string;
  progress: number;
  totalTargets: number;
  successCount: number;
  failedCount: number;
  error?: string;
  scheduledAt?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt?: string;
  createdBy?: string;
}

export interface DistributionTargetRow {
  id: number;
  taskId: number;
  host: string;
  port: number;
  targetPath: string;
  username: string;
  status: string;
  progress: number;
  error?: string;
  retryCount: number;
  maxRetries: number;
  startedAt?: string;
  finishedAt?: string;
}

export interface DistributionEventRow {
  id: number;
  taskId: number;
  targetId?: number;
  type: string;
  message: string;
  createdAt?: string;
}
export type TaskDetailResponse = DistributionTaskRow;

/**
 * One resolved, authorized distribution target. `hostId` and `stateVersion` must
 * match the CMDB target-selection resolution; the backend provider canonicalizes
 * `host`/`port` on create and refuses conflicts. Authentication is write-only:
 * existing backend response DTOs omit password/privateKey and parent snapshots.
 */
export interface DistributionTargetConfig {
  hostId: number;
  stateVersion: number;
  host: string;
  port?: number;
  targetPath: string;
  username: string;
  password?: string;
  privateKey?: string;
  hostFingerprint: string;
  maxRetries?: number;
}

export interface CreateTaskPayload {
  name: string;
  fileId: number;
  targets: DistributionTargetConfig[];
  selection: DeployTargetSelection;
  scopeVersion: string;
  scheduledAt?: string;
}

export function getFileList(
  params?: { keyword?: string; storageId?: number; page?: number; pageSize?: number },
  signal?: AbortSignal,
) {
  return apiRequest<FileListResponse>({
    url: '/business/filedist/files',
    method: 'get',
    params,
    signal,
  });
}

export function uploadFile(file: File, storageId?: number) {
  const data = new FormData();
  data.append('file', file);
  if (storageId) data.append('storageId', String(storageId));
  return apiRequest<FileRecordRow>({
    url: '/business/filedist/files/upload',
    method: 'post',
    data,
  });
}

export function deleteFile(id: number) {
  return apiRequest({ url: `/business/filedist/files/${id}`, method: 'delete' });
}

export function getTaskList(
  params?: { status?: string; fileId?: number; keyword?: string; page?: number; pageSize?: number },
  signal?: AbortSignal,
) {
  return apiRequest<{
    items: DistributionTaskRow[];
    total: number;
    page: number;
    pageSize: number;
  }>({ url: '/business/filedist/tasks', method: 'get', params, signal });
}

export function getTaskDetail(id: number, signal?: AbortSignal) {
  return apiRequest<TaskDetailResponse>({
    url: `/business/filedist/tasks/${id}`,
    method: 'get',
    signal,
    skipErrorMessage: true,
  });
}
export function getTaskTargets(id: number, signal?: AbortSignal) {
  return apiRequest<{ items: DistributionTargetRow[] }>({
    url: `/business/filedist/tasks/${id}/targets`,
    method: 'get',
    signal,
    skipErrorMessage: true,
  });
}
export function getTaskEvents(id: number, signal?: AbortSignal) {
  return apiRequest<DistributionEventRow[]>({
    url: `/business/filedist/tasks/${id}/events`,
    method: 'get',
    signal,
    skipErrorMessage: true,
  });
}
export function startTask(id: number) {
  return apiRequest({
    url: `/business/filedist/tasks/${id}/start`,
    method: 'post',
    skipErrorMessage: true,
  });
}
export function cancelTask(id: number) {
  return apiRequest({
    url: `/business/filedist/tasks/${id}/cancel`,
    method: 'post',
    skipErrorMessage: true,
  });
}
export function retryTask(id: number) {
  return apiRequest({
    url: `/business/filedist/tasks/${id}/retry`,
    method: 'post',
    skipErrorMessage: true,
  });
}
export function deleteTask(id: number) {
  return apiRequest({ url: `/business/filedist/tasks/${id}`, method: 'delete' });
}

/**
 * Creates a distribution task with an authorized target selection. The response
 * is the created task row so the UI can navigate to its detail for live progress.
 * `skipErrorMessage` keeps the failure surface inside the create modal.
 */
export function createTask(data: CreateTaskPayload) {
  return apiRequest<DistributionTaskRow>({
    url: '/business/filedist/tasks',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}
