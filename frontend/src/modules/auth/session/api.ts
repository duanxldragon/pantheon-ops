import { apiRequest } from '../../../api/request';

export interface AuthSessionPayload {
  tokenType?: string;
  accessExpiresAt?: string;
  refreshExpiresAt?: string;
  sessionId?: string;
}

export interface AuthSession {
  sessionId: string;
  isCurrent: boolean;
  lastIp: string;
  browser: string;
  os: string;
  device: string;
  userAgent: string;
  refreshExpiresAt: string;
  lastRefreshAt?: string;
  lastActivityAt?: string;
  revokedAt?: string;
  createdAt: string;
}

export interface AdminSessionRow {
  sessionId: string;
  userId: number;
  username: string;
  nickname: string;
  lastIp: string;
  browser: string;
  os: string;
  device: string;
  userAgent: string;
  refreshExpiresAt: string;
  lastRefreshAt?: string;
  lastActivityAt?: string;
  revokedAt?: string;
  createdAt: string;
}

export interface AdminSessionQuery {
  username?: string;
  lastIp?: string;
  browser?: string;
  os?: string;
  device?: string;
  status?: number;
  page?: number;
  pageSize?: number;
}

export interface AdminSessionPageResp {
  items: AdminSessionRow[];
  total: number;
  activeCount: number;
  revokedCount: number;
  page: number;
  pageSize: number;
}

export interface SessionCleanupPayload {
  retentionDays?: number;
  startedAt?: string;
  endedAt?: string;
}

export interface SessionBatchRevokePayload {
  sessionIds: string[];
}

export function getSessions() {
  return apiRequest<AuthSession[]>({
    url: '/auth/sessions',
    method: 'get',
  });
}

export function revokeSession(sessionId: string) {
  return apiRequest<{ revoked: boolean }>({
    url: `/auth/sessions/${sessionId}`,
    method: 'delete',
  });
}

export function logout() {
  return apiRequest<{ loggedOut: boolean }>({
    url: '/auth/logout',
    method: 'post',
  });
}

export function reportActivity() {
  return apiRequest<{ touched: boolean }>({
    url: '/auth/activity',
    method: 'post',
    skipErrorMessage: true,
  });
}

export function getAdminSessionList(params?: AdminSessionQuery) {
  return apiRequest<AdminSessionPageResp>({
    url: '/system/session/list',
    method: 'get',
    params,
  });
}

export function revokeAdminSession(sessionId: string) {
  return apiRequest<{ revoked: boolean }>({
    url: `/system/session/${sessionId}`,
    method: 'delete',
  });
}

export function cleanupAdminSessions(data: SessionCleanupPayload) {
  return apiRequest<{ clearedCount: number }>({
    url: '/system/session/cleanup',
    method: 'post',
    data,
  });
}

export function batchRevokeAdminSessions(data: SessionBatchRevokePayload) {
  return apiRequest<{ revokedCount: number }>({
    url: '/system/session/batch-revoke',
    method: 'post',
    data,
  });
}
