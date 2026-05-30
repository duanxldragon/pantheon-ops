import { apiRequest } from '../../../api/request';
import { downloadFile } from '../../../api/file';

export interface RoleRow {
  id: number;
  roleName: string;
  roleKey: string;
  sort: number;
  status: number;
  createdAt: string;
  menuIds: number[];
  permissionKeys: string[];
}

export interface RoleListQuery {
  roleName?: string;
  roleKey?: string;
  status?: number;
  page?: number;
  pageSize?: number;
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface RoleListPageResp {
  items: RoleRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface RolePayload {
  roleName: string;
  roleKey: string;
  sort: number;
  status: number;
  menuIds: number[];
  permissionKeys: string[];
}

export interface RoleBatchStatusPayload {
  roleIds: number[];
  status: number;
}

export interface RoleMemberRow {
  id: number;
  username: string;
  nickname: string;
  deptId: number;
  deptName: string;
  postId: number;
  postName: string;
  status: number;
  createdAt: string;
  lastLoginAt?: string;
}

export interface RoleMemberQuery {
  keyword?: string;
  status?: number;
  page?: number;
  pageSize?: number;
}

export interface RoleMemberPageResp {
  items: RoleMemberRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchDeleteResp {
  deletedCount: number;
  failedCount: number;
  failures: Array<{ id: number; reason: string }>;
}

export function getRoleList(params?: RoleListQuery) {
  return apiRequest<RoleListPageResp>({
    url: '/system/role/list',
    method: 'get',
    params,
  });
}

export function createRole(data: RolePayload) {
  return apiRequest<RoleRow>({
    url: '/system/role',
    method: 'post',
    data,
  });
}

export function updateRole(id: number, data: RolePayload) {
  return apiRequest<RoleRow>({
    url: `/system/role/${id}`,
    method: 'put',
    data,
  });
}

export function deleteRole(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/role/${id}`,
    method: 'delete',
  });
}

export function exportRoles(data?: RoleListQuery) {
  return downloadFile({
    url: '/system/role/export',
    method: 'post',
    data,
    filename: 'system-role-export.csv',
  });
}

export function batchUpdateRoleStatus(data: RoleBatchStatusPayload) {
  return apiRequest<{ updatedCount: number }>({
    url: '/system/role/batch-status',
    method: 'post',
    data,
  });
}

export function batchDeleteRoles(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/role/batch-delete',
    method: 'post',
    data,
  });
}

export function getRoleMembers(id: number, params?: RoleMemberQuery) {
  return apiRequest<RoleMemberPageResp>({
    url: `/system/role/${id}/users`,
    method: 'get',
    params,
  });
}

export function getRoleMemberCandidates(id: number, params?: RoleMemberQuery) {
  return apiRequest<RoleMemberPageResp>({
    url: `/system/role/${id}/user-candidates`,
    method: 'get',
    params,
  });
}

export function addRoleMembers(id: number, data: { userIds: number[] }) {
  return apiRequest<{ addedCount: number }>({
    url: `/system/role/${id}/users`,
    method: 'post',
    data,
  });
}

export function removeRoleMembers(id: number, data: { userIds: number[] }) {
  return apiRequest<{ removedCount: number }>({
    url: `/system/role/${id}/users/remove`,
    method: 'post',
    data,
  });
}
