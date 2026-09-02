import { apiRequest } from '../../../../api/request';

// --- Server ---

export interface ArgoCDServerRow {
  id: number;
  name: string;
  clusterId: number;
  endpoint: string;
  status: string;
  version?: string;
  installed: boolean;
  lastChecked?: string;
  createdAt: string;
  createdBy: string;
}

export interface ArgoCDServerListResp {
  items: ArgoCDServerRow[];
  total: number;
}

export function getArgoCDServerList() {
  return apiRequest<ArgoCDServerListResp>({ url: '/business/cicd/argocd/servers', method: 'get' });
}

export function addArgoCDServer(data: { name: string; clusterId?: number; endpoint: string; authToken: string }) {
  return apiRequest<ArgoCDServerRow>({ url: '/business/cicd/argocd/servers', method: 'post', data });
}

export function deleteArgoCDServer(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/cicd/argocd/servers/${id}`, method: 'delete' });
}

export function testArgoCDServer(id: number) {
  return apiRequest<{ connected: boolean }>({ url: `/business/cicd/argocd/servers/${id}/test`, method: 'post' });
}

// --- Repository ---

export interface ArgoCDRepositoryRow {
  id: number;
  serverId: number;
  repoUrl: string;
  name: string;
  branch: string;
  type: string;
  username?: string;
  status: string;
  createdAt: string;
  createdBy: string;
}

export interface ArgoCDRepositoryListResp {
  items: ArgoCDRepositoryRow[];
  total: number;
}

export function getArgoCDRepositoryList(serverId?: number) {
  return apiRequest<ArgoCDRepositoryListResp>({
    url: '/business/cicd/argocd/repositories',
    method: 'get',
    params: serverId ? { serverId } : undefined,
  });
}

export function addArgoCDRepository(data: { serverId: number; repoUrl: string; name?: string; branch?: string; type: string; username?: string; password?: string }) {
  return apiRequest<ArgoCDRepositoryRow>({ url: '/business/cicd/argocd/repositories', method: 'post', data });
}

export function deleteArgoCDRepository(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/cicd/argocd/repositories/${id}`, method: 'delete' });
}

// --- Application ---

export interface ArgoCDApplicationRow {
  id: number;
  serverId: number;
  name: string;
  project: string;
  sourceRepoUrl: string;
  sourcePath: string;
  sourceTargetRev: string;
  destinationServer: string;
  destinationNamespace: string;
  status: string;
  healthStatus: string;
  syncStatus: string;
  lastError?: string;
  lastSyncedAt?: string;
  createdAt: string;
  createdBy: string;
}

export interface ArgoCDApplicationListResp {
  items: ArgoCDApplicationRow[];
  total: number;
}

export function getArgoCDApplicationList(serverId?: number) {
  return apiRequest<ArgoCDApplicationListResp>({
    url: '/business/cicd/argocd/applications',
    method: 'get',
    params: serverId ? { serverId } : undefined,
  });
}

export function createArgoCDApplication(data: {
  serverId: number;
  name: string;
  project?: string;
  sourceRepoUrl: string;
  sourcePath?: string;
  sourceTargetRev?: string;
  destinationServer?: string;
  destinationNamespace?: string;
}) {
  return apiRequest<ArgoCDApplicationRow>({ url: '/business/cicd/argocd/applications', method: 'post', data });
}

export function deleteArgoCDApplication(id: number) {
  return apiRequest<{ deleted: boolean }>({ url: `/business/cicd/argocd/applications/${id}`, method: 'delete' });
}

export function syncArgoCDApplication(id: number, revision?: string) {
  return apiRequest<{ id: number; status: string }>({
    url: `/business/cicd/argocd/applications/${id}/sync`,
    method: 'post',
    data: { revision },
  });
}

export function refreshArgoCDApplication(id: number) {
  return apiRequest<{ refreshed: boolean }>({ url: `/business/cicd/argocd/applications/${id}/refresh`, method: 'post' });
}

export function rollbackArgoCDApplication(id: number, revision: string) {
  return apiRequest<{ id: number; status: string }>({
    url: `/business/cicd/argocd/applications/${id}/rollback`,
    method: 'post',
    data: { revision },
  });
}

export interface ArgoCDSyncHistoryRow {
  id: number;
  revision: string;
  status: string;
  message?: string;
  startedAt: string;
  finishedAt?: string;
}

export function getArgoCDSyncHistory(appId: number) {
  return apiRequest<ArgoCDSyncHistoryRow[]>({ url: `/business/cicd/argocd/applications/${appId}/history`, method: 'get' });
}
