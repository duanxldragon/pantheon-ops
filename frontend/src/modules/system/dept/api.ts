import { apiRequest } from '../../../api/request';
import { downloadFile } from '../../../api/file';
import { uploadImportFile } from '../../../api/importExport';

export interface DeptNode {
  id: number;
  parentId: number;
  ancestors: string;
  isRoot: boolean;
  deptName: string;
  sort: number;
  leaderUserId: number;
  leader: string;
  phone: string;
  email: string;
  status: number;
  childDeptCount: number;
  postCount: number;
  isLeaderless: boolean;
  isNoPost: boolean;
  isEmpty: boolean;
  children?: DeptNode[];
}

export interface DeptListQuery {
  deptName?: string;
  status?: number;
  governance?: 'leaderless' | 'no-post' | 'empty';
  sortField?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface DeptPayload {
  parentId: number;
  deptName: string;
  sort: number;
  leaderUserId?: number;
  leader?: string;
  phone?: string;
  email?: string;
  status: number;
}

export interface DeptBatchStatusPayload {
  deptIds: number[];
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

export interface DeptBatchLeaderItemPayload {
  deptId: number;
  leaderUserId: number;
}

export interface DeptBatchLeaderPayload {
  items: DeptBatchLeaderItemPayload[];
}

export interface DeptOverviewResp {
  totalDeptCount: number;
  enabledDeptCount: number;
  disabledDeptCount: number;
  rootDeptCount: number;
  directChildDeptCount: number;
  totalPostCount: number;
  enabledPostCount: number;
  leaderlessDeptCount: number;
  noPostDeptCount: number;
  emptyDeptCount: number;
  healthIssueCount: number;
}

export interface DeptGovernanceTaskQuery {
  keyword?: string;
  scope?: 'all' | 'dept' | 'post';
  governance?: 'leaderless' | 'no-post' | 'empty' | 'in-use' | 'disabled';
  blockedBy?: 'children' | 'posts' | 'users' | 'none';
  action?: string;
}

export interface DeptGovernanceTask {
  taskKey: string;
  governanceScope: 'dept' | 'post';
  governanceScopeLabel: string;
  governanceTag: string;
  governanceTagLabel: string;
  governanceBlockedBy: string;
  governanceBlockedByLabel: string;
  governanceAction: string;
  governanceActionLabel: string;
  deptId: number;
  deptName: string;
  deptPath: string;
  postId: number;
  postName: string;
  relatedUserCount: number;
  resourceStatus: number;
}

export interface DeptLeaderCandidate {
  userId: number;
  username: string;
  nickname: string;
  displayName: string;
  deptId: number;
  deptName: string;
  postId: number;
  postName: string;
}

export function getDeptOverview() {
  return apiRequest<DeptOverviewResp>({
    url: '/system/dept/overview',
    method: 'get',
  });
}

export function getDeptGovernanceTasks(params?: DeptGovernanceTaskQuery) {
  return apiRequest<DeptGovernanceTask[]>({
    url: '/system/dept/governance/tasks',
    method: 'get',
    params,
  });
}

export function getDeptTree(params?: DeptListQuery) {
  return apiRequest<DeptNode[]>({
    url: '/system/dept/tree',
    method: 'get',
    params,
  });
}

export function getDeptLeaderCandidates(id: number) {
  return apiRequest<DeptLeaderCandidate[]>({
    url: `/system/dept/${id}/leader-candidates`,
    method: 'get',
  });
}

export function createDept(data: DeptPayload) {
  return apiRequest<DeptNode>({
    url: '/system/dept',
    method: 'post',
    data,
  });
}

export function updateDept(id: number, data: DeptPayload) {
  return apiRequest<DeptNode>({
    url: `/system/dept/${id}`,
    method: 'put',
    data,
  });
}

export function deleteDept(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/system/dept/${id}`,
    method: 'delete',
  });
}

export function batchUpdateDeptStatus(data: DeptBatchStatusPayload) {
  return apiRequest<{ updatedCount: number }>({
    url: '/system/dept/batch-status',
    method: 'post',
    data,
  });
}

export function batchDeleteDepts(data: BatchDeletePayload) {
  return apiRequest<BatchDeleteResp>({
    url: '/system/dept/batch-delete',
    method: 'post',
    data,
  });
}

export function batchUpdateDeptLeader(data: DeptBatchLeaderPayload) {
  return apiRequest<{ updatedCount: number }>({
    url: '/system/dept/batch-leader',
    method: 'post',
    data,
  });
}

export function exportDepts(data?: DeptListQuery) {
  return downloadFile({
    url: '/system/dept/export',
    method: 'post',
    data,
    filename: 'system-dept-export.csv',
  });
}

export function exportDeptGovernanceTasks(data?: DeptGovernanceTaskQuery) {
  return downloadFile({
    url: '/system/dept/governance/export',
    method: 'post',
    data,
    filename: 'system-org-governance-tasks.csv',
  });
}

export function downloadDeptImportTemplate() {
  return downloadFile({
    url: '/system/dept/import-template',
    method: 'get',
    filename: 'system-dept-import-template.csv',
  });
}

export function importDepts(file: File) {
  return uploadImportFile('/system/dept/import', file);
}
