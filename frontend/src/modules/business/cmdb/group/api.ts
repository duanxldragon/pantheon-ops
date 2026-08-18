import { apiRequest } from '../../../../api/request';
import { downloadFile, uploadImportFile } from '../../../../api/importExport';

export interface ConditionRule {
  key: string;
  op: 'eq' | 'neq' | 'in' | 'notIn';
  val: string;
}

export interface ConditionExpression {
  operator: 'AND' | 'OR';
  rules: ConditionRule[];
}

export interface GroupRow {
  id: number;
  parentId: number;
  name: string;
  description: string;
  conditions: ConditionExpression;
  memberCount: number;
  aggregateMemberCount: number;
  childCount: number;
  descendantGroupCount: number;
  children?: GroupRow[];
  createdAt: string;
  updatedAt: string;
}

export interface CreateGroupPayload {
  name: string;
  parentId?: number;
  description?: string;
  conditions: ConditionExpression;
}

export interface UpdateGroupPayload {
  name?: string;
  parentId?: number;
  description?: string;
  conditions?: ConditionExpression;
}

export interface GroupMemberResp {
  groupId: number;
  groupName: string;
  members: GroupMemberRow[];
}

export interface GroupMemberRow {
  id: number;
  hostname: string;
  ip: string;
  status: string;
}

export function getGroupList() {
  return apiRequest<GroupRow[]>({
    url: '/business/cmdb/groups',
    method: 'get',
  });
}

export function getGroupDetail(id: number) {
  return apiRequest<GroupRow>({
    url: `/business/cmdb/groups/${id}`,
    method: 'get',
  });
}

export function getGroupMembers(id: number) {
  return apiRequest<GroupMemberResp>({
    url: `/business/cmdb/groups/${id}/members`,
    method: 'get',
  });
}

export function createGroup(data: CreateGroupPayload) {
  return apiRequest<GroupRow>({
    url: '/business/cmdb/groups',
    method: 'post',
    data,
  });
}

export function updateGroup(id: number, data: UpdateGroupPayload) {
  return apiRequest<GroupRow>({
    url: `/business/cmdb/groups/${id}`,
    method: 'put',
    data,
  });
}

export function deleteGroup(id: number) {
  return apiRequest<void>({
    url: `/business/cmdb/groups/${id}`,
    method: 'delete',
  });
}

export interface GroupListQuery {
  keyword?: string;
}

export function exportGroups(params?: GroupListQuery) {
  return downloadFile({
    url: '/business/cmdb/groups/export',
    method: 'get',
    params: params as Record<string, unknown> | undefined,
    filename: 'cmdb-group-export.csv',
  });
}

export function downloadGroupImportTemplate() {
  return downloadFile({
    url: '/business/cmdb/groups/import-template',
    method: 'get',
    filename: 'cmdb-group-import-template.csv',
  });
}

export function importGroups(file: File) {
  return uploadImportFile('/business/cmdb/groups/import', file);
}
