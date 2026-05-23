import { apiRequest } from '../../../api/request';
import { downloadFile } from '../../../api/file';
import { uploadImportFile } from '../../../api/importExport';

export interface PermissionPolicyRow {
  id: number;
  ptype: string;
  roleKey: string;
  path: string;
  method: string;
}

export interface PermissionPolicyQuery {
  roleKey?: string;
  path?: string;
  method?: string;
  page?: number;
  pageSize?: number;
}

export interface PermissionPolicyPageResp {
  items: PermissionPolicyRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface PermissionPolicyPayload {
  roleKey: string;
  path: string;
  method: string;
}

export interface BatchDeletePayload {
  ids: number[];
}

export interface BatchDeleteResp {
  deletedCount: number;
  failedCount: number;
  failures: Array<{ id: number; reason: string }>;
}

export interface PermissionWorkbenchQuery {
  roleKey?: string;
  status?: number;
  integrity?: string;
  coverage?: string;
}

export interface PermissionWorkbenchOverview {
  roleCount: number;
  enabledRoleCount: number;
  navigationAssignmentCount: number;
  pagePermissionAssignmentCount: number;
  actionPermissionAssignmentCount: number;
  apiActionCount: number;
  unknownPermissionAssignmentCount: number;
  pageGapRoleCount: number;
  apiGapRoleCount: number;
  pendingRemediationRoleCount: number;
  remediatedRoleCount: number;
  recentRemediationCount: number;
}

export interface PermissionWorkbenchMenu {
  id: number;
  titleKey: string;
  path: string;
  module: string;
}

export interface PermissionWorkbenchPermission {
  key: string;
  titleKey: string;
  path: string;
  module: string;
  kind: 'page' | 'action' | 'unknown';
}

export interface PermissionWorkbenchApiPolicy {
  id: number;
  path: string;
  method: string;
}

export interface PermissionWorkbenchRole {
  id: number;
  roleName: string;
  roleKey: string;
  status: number;
  menuCount: number;
  pagePermissionCount: number;
  actionPermissionCount: number;
  apiPolicyCount: number;
  requiredApiPolicyCount: number;
  missingApiPolicyCount: number;
  unknownPermissionCount: number;
  hasPageGap: boolean;
  hasApiGap: boolean;
  governanceStatus: 'pending' | 'remediated' | 'clean';
  lastRemediationAt?: string;
  lastRemediationAction?: string;
  menus: PermissionWorkbenchMenu[];
  pagePermissions: PermissionWorkbenchPermission[];
  actionPermissions: PermissionWorkbenchPermission[];
  unknownPermissions: PermissionWorkbenchPermission[];
  apiPolicies: PermissionWorkbenchApiPolicy[];
  missingApiPolicies: PermissionWorkbenchApiPolicy[];
}

export interface PermissionWorkbenchResp {
  overview: PermissionWorkbenchOverview;
  roles: PermissionWorkbenchRole[];
}

export interface PermissionWorkbenchRemediatePayload {
  roleKey: string;
}

export interface PermissionWorkbenchRemediateResp {
  roleKey: string;
  createdCount: number;
  skippedCount: number;
  createdPolicies: PermissionWorkbenchApiPolicy[];
}

export interface PermissionWorkbenchRemediationEvent {
  id: number;
  roleKey: string;
  issueType: string;
  issueKey: string;
  beforeState: string;
  afterState: string;
  action: string;
  createdCount: number;
  skippedCount: number;
  createdAt: string;
}

export type PermissionDataScopeMode = 'all' | 'self' | 'dept' | 'dept_and_children' | 'custom';

export interface PermissionDataScopeQuery {
  roleKey?: string;
  status?: number;
}

export interface PermissionDataScopePolicy {
  id: number;
  roleName: string;
  roleKey: string;
  status: number;
  mode: PermissionDataScopeMode;
  deptIds: number[];
  policyExists: boolean;
}

export interface PermissionDataScopePolicyListResp {
  items: PermissionDataScopePolicy[];
  total: number;
}

export interface PermissionDataScopePolicyPayload {
  mode: PermissionDataScopeMode;
  deptIds?: number[];
}

export function getPermissionWorkbench(params?: PermissionWorkbenchQuery) {
  return apiRequest<PermissionWorkbenchResp>({
    url: '/system/permission/workbench',
    method: 'get',
    params,
  });
}

export function getPermissionPolicyList(params?: PermissionPolicyQuery) {
  return apiRequest<PermissionPolicyPageResp>({
    url: '/system/permission/list',
    method: 'get',
    params,
  });
}

export function remediatePermissionWorkbenchRole(data: PermissionWorkbenchRemediatePayload) {
  return apiRequest<PermissionWorkbenchRemediateResp>({
    url: '/system/permission/workbench/remediate',
    method: 'post',
    data,
  });
}

export function getPermissionWorkbenchRemediationEvents(params?: {
  roleKey?: string;
  limit?: number;
}) {
  return apiRequest<PermissionWorkbenchRemediationEvent[]>({
    url: '/system/permission/workbench/remediation',
    method: 'get',
    params,
  });
}

export function getPermissionDataScopePolicies(params?: PermissionDataScopeQuery) {
  return apiRequest<PermissionDataScopePolicyListResp>({
    url: '/system/permission/data-scope',
    method: 'get',
    params,
  });
}

export function updatePermissionDataScopePolicy(
  roleKey: string,
  data: PermissionDataScopePolicyPayload,
) {
  return apiRequest<PermissionDataScopePolicy>({
    url: `/system/permission/data-scope/${roleKey}`,
    method: 'put',
    data,
  });
}

export function createPermissionPolicy(data: PermissionPolicyPayload) {
  return apiRequest<PermissionPolicyRow>({
    url: '/system/permission',
    method: 'post',
    data,
  });
}

export function updatePermissionPolicy(id: number, data: PermissionPolicyPayload) {
  return apiRequest<PermissionPolicyRow>({
    url: `/system/permission/${id}`,
    method: 'put',
    data,
  });
}

export function deletePermissionPolicy(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/permission/${id}`,
    method: 'delete',
  });
}

export function batchDeletePermissionPolicies(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/permission/batch-delete',
    method: 'post',
    data,
  });
}

export function exportPermissionPolicies(data?: PermissionPolicyQuery) {
  return downloadFile({
    url: '/system/permission/export',
    method: 'post',
    data,
    filename: 'system-permission-export.csv',
  });
}

export function exportPermissionWorkbench(params?: PermissionWorkbenchQuery) {
  return downloadFile({
    url: '/system/permission/workbench/export',
    method: 'get',
    params: params ? { ...params } : undefined,
    filename: 'system-permission-workbench-export.csv',
  });
}

export function downloadPermissionImportTemplate() {
  return downloadFile({
    url: '/system/permission/import-template',
    method: 'get',
    filename: 'system-permission-import-template.csv',
  });
}

export function importPermissionPolicies(file: File) {
  return uploadImportFile('/system/permission/import', file);
}
