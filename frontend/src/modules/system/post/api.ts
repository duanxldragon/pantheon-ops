import { apiRequest } from '../../../api/request';
import { downloadFile } from '../../../api/file';
import { uploadImportFile } from '../../../api/importExport';

export interface PostRow {
  id: number;
  deptId: number;
  deptName: string;
  postCode: string;
  postName: string;
  sort: number;
  status: number;
  remark: string;
  assignedUserCount: number;
  governanceTags: string[];
  governanceTagLabels: string[];
  governanceBlockedBy: string[];
  governanceBlockedDesc: string[];
  governanceActions: string[];
  governanceActionLabel: string[];
  createdAt: string;
}

export interface PostListQuery {
  postCode?: string;
  postName?: string;
  deptId?: number;
  status?: number;
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface PostListPageResp {
  items: PostRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PostPayload {
  deptId: number;
  postCode: string;
  postName: string;
  sort: number;
  status: number;
  remark?: string;
}

export interface PostBatchStatusPayload {
  postIds: number[];
  status: number;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchDeleteResp {
  deletedCount: number;
  failedCount: number;
  failures: Array<{ id: number; reason: string }>;
}

export function getPostList(params?: PostListQuery) {
  return apiRequest<PostListPageResp>({
    url: '/system/post/list',
    method: 'get',
    params,
  });
}

export function createPost(data: PostPayload) {
  return apiRequest<PostRow>({
    url: '/system/post',
    method: 'post',
    data,
  });
}

export function updatePost(id: number, data: PostPayload) {
  return apiRequest<PostRow>({
    url: `/system/post/${id}`,
    method: 'put',
    data,
  });
}

export function deletePost(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/post/${id}`,
    method: 'delete',
  });
}

export function batchUpdatePostStatus(data: PostBatchStatusPayload) {
  return apiRequest<{ updatedCount: number }>({
    url: '/system/post/batch-status',
    method: 'post',
    data,
  });
}

export function batchDeletePosts(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/post/batch-delete',
    method: 'post',
    data,
  });
}

export function exportPosts(data?: PostListQuery) {
  return downloadFile({
    url: '/system/post/export',
    method: 'post',
    data,
    filename: 'system-post-export.csv',
  });
}

export function downloadPostImportTemplate() {
  return downloadFile({
    url: '/system/post/import-template',
    method: 'get',
    filename: 'system-post-import-template.csv',
  });
}

export function importPosts(file: File) {
  return uploadImportFile('/system/post/import', file);
}
