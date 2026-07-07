import { apiRequest } from '../../../api/request';
import { downloadCsvFile, downloadFile } from '../../../api/file';
import type { UserInfo, UserPlatformPreferences } from './types';

export { getSessions, revokeSession } from '../session/api';
export type { AuthSession } from '../session/api';
export type { UserInfo, UserPlatformPreferences } from './types';

export interface SecurityOverview {
  user?: {
    id: number;
    username: string;
    nickname: string;
    avatar?: string;
    email?: string;
    phone?: string;
    roles?: string[];
    perms?: string[];
    preferences?: {
      theme?: 'indigo' | 'emerald' | 'violet' | 'slate';
      language?: 'zh-CN' | 'en-US' | 'ja-JP' | 'ko-KR' | 'fr-FR';
      layoutMode?: 'vertical' | 'horizontal';
      densityMode?: 'comfortable' | 'compact';
    };
  };
  currentSession?: {
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
  };
  activeSessionCount: number;
  lastLoginAt?: string;
  passwordExpired?: boolean;
  passwordExpiresAt?: string;
  recentSecurityEvents?: SecurityEventRow[];
  policy: SecurityPolicy;
}

export interface SecurityPolicy {
  passwordMinLength: number;
  passwordRequireDigit: boolean;
  passwordRequireUpper: boolean;
  passwordHistoryLimit?: number;
  passwordExpireDays?: number;
  maxFailedAttempts: number;
  lockMinutes: number;
  sourceMaxFailedAttempts: number;
  sourceWindowMinutes: number;
  sourceLockMinutes: number;
  sessionIdleMinutes: number;
  maxActiveSessions: number;
  sessionRetentionDays: number;
  captchaEnabled: boolean;
  mfaEnabled: boolean;
  ssoEnabled: boolean;
}

export interface SecurityEventRow {
  id: number;
  userId: number;
  username: string;
  eventType: string;
  severity: string;
  sourceKey: string;
  ip: string;
  userAgent: string;
  messageKey: string;
  metadata: string;
  acknowledgedAt?: string;
  acknowledgedBy: number;
  acknowledgedByUser: string;
  acknowledgementNote: string;
  createdAt: string;
}

export interface SecurityEventQuery {
  username?: string;
  eventType?: string;
  severity?: string;
  acknowledged?: boolean;
  page?: number;
  pageSize?: number;
}

export interface SecurityEventPageResp {
  items: SecurityEventRow[];
  total: number;
  page: number;
  pageSize: number;
}

export interface SecurityEventAcknowledgePayload {
  acknowledgementNote: string;
}

export interface LoginLogRow {
  id: number;
  username: string;
  ipaddr: string;
  loginLocation: string;
  browser: string;
  os: string;
  status: number;
  msg: string;
  loginTime: string;
}

export interface LoginLogQuery {
  username?: string;
  status?: number;
  page?: number;
  pageSize?: number;
}

export interface LoginLogCleanupPayload {
  retentionDays?: number;
  startedAt?: string;
  endedAt?: string;
}

export interface LoginLogBatchDeletePayload {
  ids: number[];
}

export interface LoginLogPageResp {
  items: LoginLogRow[];
  total: number;
  page: number;
  pageSize: number;
}

export function acknowledgeSecurityEvent(id: number, data: SecurityEventAcknowledgePayload) {
  return apiRequest<{ acknowledged: boolean }>({
    url: `/system/security-event/${id}/acknowledge`,
    method: 'post',
    data,
  });
}

export function getSecurityOverview() {
  return apiRequest<SecurityOverview>({
    url: '/auth/security',
    method: 'get',
  });
}

export function getMe() {
  return apiRequest<UserInfo>({
    url: '/auth/me',
    method: 'get',
  });
}

export function verifyOperationPassword(password: string) {
  return apiRequest<{ operationToken: string }>({
    url: '/auth/operation-verify',
    method: 'post',
    data: { password },
  });
}

export function updateCurrentUserPreferences(data: UserPlatformPreferences) {
  return apiRequest<UserInfo>({
    url: '/auth/me/preferences',
    method: 'put',
    data,
  });
}

export function getOwnLoginLogs(params?: LoginLogQuery) {
  return apiRequest<LoginLogPageResp>({
    url: '/auth/login-logs',
    method: 'get',
    params,
  });
}

export function getAdminLoginLogList(params?: LoginLogQuery) {
  return apiRequest<LoginLogPageResp>({
    url: '/system/login-log/list',
    method: 'get',
    params,
  });
}

export function getAdminSecurityEventList(params?: SecurityEventQuery) {
  return apiRequest<SecurityEventPageResp>({
    url: '/system/security-event/list',
    method: 'get',
    params,
  });
}

export function exportAdminLoginLogs(data?: LoginLogQuery) {
  return downloadFile({
    url: '/system/login-log/export',
    method: 'post',
    data,
    filename: 'system-login-log-export.csv',
  });
}

export function exportSelectedAdminLoginLogs(rows: LoginLogRow[]) {
  downloadCsvFile(
    'system-login-log-export.csv',
    ['username', 'ipaddr', 'loginLocation', 'browser', 'os', 'status', 'msg', 'loginTime'],
    rows.map((item) => [
      item.username || '',
      item.ipaddr || '',
      item.loginLocation || '',
      item.browser || '',
      item.os || '',
      String(item.status ?? ''),
      item.msg || '',
      item.loginTime || '',
    ]),
  );
}

export function cleanupAdminLoginLogs(data: LoginLogCleanupPayload) {
  return apiRequest<{ clearedCount: number }>({
    url: '/system/login-log/cleanup',
    method: 'post',
    data,
  });
}

export function batchDeleteAdminLoginLogs(data: LoginLogBatchDeletePayload) {
  return apiRequest<{ deletedCount: number }>({
    url: '/system/login-log/batch-delete',
    method: 'post',
    data,
  });
}

export interface UserPasswordUpdatePayload {
  oldPassword: string;
  newPassword: string;
}

export function updatePassword(data: UserPasswordUpdatePayload) {
  return apiRequest<{ passwordUpdated: boolean }>({
    url: '/auth/password',
    method: 'put',
    data,
  });
}
