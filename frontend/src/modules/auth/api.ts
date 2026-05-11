import { apiRequest } from '../../api/request';
import { downloadFile } from '../../api/file';

export interface LoginPayload {
  username: string;
  password: string;
}

export interface MFAVerifyPayload {
  challengeId: string;
  code: string;
}

export interface UserInfo {
  id: number;
  username: string;
  nickname: string;
  avatar?: string;
  email?: string;
  phone?: string;
  roles?: string[];
  perms?: string[];
  preferences?: UserPlatformPreferences;
}

export interface UserPlatformPreferences {
  theme?: 'indigo' | 'emerald' | 'violet' | 'slate';
  language?: 'zh-CN' | 'en-US' | 'ja-JP' | 'ko-KR' | 'fr-FR';
  layoutMode?: 'vertical' | 'horizontal';
  densityMode?: 'comfortable' | 'compact';
}

export interface AuthTokens {
  token?: string;
  accessToken?: string;
  refreshToken?: string;
  tokenType?: string;
  accessExpiresAt?: string;
  refreshExpiresAt?: string;
  sessionId?: string;
}

export interface LoginResp extends AuthTokens {
  mfaRequired?: boolean;
  challengeId?: string;
  setupRequired?: boolean;
  totpSecret?: string;
  totpProvisionUri?: string;
  expiresAt?: string;
  user?: UserInfo;
}

export interface RefreshTokenPayload {
  refreshToken: string;
}

export interface UserPasswordUpdatePayload {
  oldPassword: string;
  newPassword: string;
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

export interface SecurityOverview {
  user: UserInfo;
  currentSession?: AuthSession;
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
  createdAt: string;
}

export interface SecurityEventQuery {
  username?: string;
  eventType?: string;
  severity?: string;
  page?: number;
  pageSize?: number;
}

export interface SecurityEventPageResp {
  items: SecurityEventRow[];
  total: number;
  page: number;
  pageSize: number;
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
  retentionDays: number;
}

export interface SessionCleanupPayload {
  retentionDays: number;
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

export function login(data: LoginPayload) {
  return apiRequest<LoginResp>({
    url: '/auth/login',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}

export function verifyMFA(data: MFAVerifyPayload) {
  return apiRequest<LoginResp>({
    url: '/auth/mfa/verify',
    method: 'post',
    data,
    skipErrorMessage: true,
  });
}

export function refreshToken(data: RefreshTokenPayload) {
  return apiRequest<AuthTokens>({
    url: '/auth/refresh',
    method: 'post',
    data,
    skipAuthRefresh: true,
    skipErrorMessage: true,
  });
}

export function logout() {
  return apiRequest<{ loggedOut: boolean }>({
    url: '/auth/logout',
    method: 'post',
    skipAuthRefresh: true,
    skipErrorMessage: true,
  });
}

export function reportActivity() {
  return apiRequest<{ touched: boolean }>({
    url: '/auth/activity',
    method: 'post',
    skipErrorMessage: true,
  });
}

export function getMe() {
  return apiRequest<UserInfo>({
    url: '/auth/me',
    method: 'get',
  });
}

export function updateCurrentUserPreferences(data: UserPlatformPreferences) {
  return apiRequest<UserInfo>({
    url: '/auth/me/preferences',
    method: 'put',
    data,
    skipErrorMessage: true,
  });
}

export function getSecurityOverview() {
  return apiRequest<SecurityOverview>({
    url: '/auth/security',
    method: 'get',
  });
}

export function updatePassword(data: UserPasswordUpdatePayload) {
  return apiRequest<{ passwordUpdated: boolean }>({
    url: '/auth/password',
    method: 'put',
    data,
  });
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

export function getAdminSessionList(params?: AdminSessionQuery) {
  return apiRequest<AdminSessionPageResp>({
    url: '/system/session/list',
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

export function exportAdminLoginLogs(data?: LoginLogQuery) {
  return downloadFile({
    url: '/system/login-log/export',
    method: 'post',
    data,
    filename: 'system-login-log-export.csv',
  });
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

export function verifyOperationPassword(password: string) {
  return apiRequest<{ operationToken: string }>({
    url: '/auth/operation-verify',
    method: 'post',
    data: { password },
  });
}
