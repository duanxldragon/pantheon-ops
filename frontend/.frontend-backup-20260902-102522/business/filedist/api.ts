import { apiRequest } from '../../../api/request';

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

export interface FileListResponse { items: FileRecordRow[]; total: number; page: number; pageSize: number }

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

export interface DistributionEventRow { id: number; taskId: number; targetId?: number; type: string; message: string; createdAt?: string }
export type TaskDetailResponse = DistributionTaskRow;

export function getFileList(params?: { keyword?: string; storageId?: number; page?: number; pageSize?: number }) {
  return apiRequest<FileListResponse>({ url: '/business/filedist/files', method: 'get', params });
}

export function uploadFile(file: File, storageId?: number) {
  const data = new FormData();
  data.append('file', file);
  if (storageId) data.append('storageId', String(storageId));
  return apiRequest<FileRecordRow>({ url: '/business/filedist/files/upload', method: 'post', data });
}

export function deleteFile(id: number) { return apiRequest({ url: `/business/filedist/files/${id}`, method: 'delete' }); }

export function getTaskList(params?: { status?: string; fileId?: number; keyword?: string; page?: number; pageSize?: number }) {
  return apiRequest<{ items: DistributionTaskRow[]; total: number; page: number; pageSize: number }>({ url: '/business/filedist/tasks', method: 'get', params });
}

export function getTaskDetail(id: number) { return apiRequest<TaskDetailResponse>({ url: `/business/filedist/tasks/${id}`, method: 'get' }); }
export function getTaskTargets(id: number) { return apiRequest<{ items: DistributionTargetRow[] }>({ url: `/business/filedist/tasks/${id}/targets`, method: 'get' }); }
export function getTaskEvents(id: number) { return apiRequest<DistributionEventRow[]>({ url: `/business/filedist/tasks/${id}/events`, method: 'get' }); }
export function startTask(id: number) { return apiRequest({ url: `/business/filedist/tasks/${id}/start`, method: 'post' }); }
export function cancelTask(id: number) { return apiRequest({ url: `/business/filedist/tasks/${id}/cancel`, method: 'post' }); }
export function retryTask(id: number) { return apiRequest({ url: `/business/filedist/tasks/${id}/retry`, method: 'post' }); }
export function deleteTask(id: number) { return apiRequest({ url: `/business/filedist/tasks/${id}`, method: 'delete' }); }
