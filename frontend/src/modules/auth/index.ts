import { defineModule, type DashboardQuickActionWidget } from '../../core/router/types';
import { getOwnLoginLogs, getSecurityOverview } from './security/api';
import { getSessions } from './session/api';

function quickActionWidget(
  key: string,
  titleKey: string,
  descriptionKey: string,
  path: string,
  permission: string,
  icon: DashboardQuickActionWidget['icon'],
): DashboardQuickActionWidget {
  return {
    key,
    slot: 'quick-action',
    sourceDomain: 'system/auth',
    titleKey,
    descriptionKey,
    path,
    permission,
    icon,
    cleanupPolicy: 'hide_when_forbidden',
  };
}

export { LoginPageComponent as LoginPage } from './login/components/Login';
export { login } from './login/api';
export type { LoginPayload, LoginResp } from './login/api';

export { verifyMFA } from './mfa/api';
export type { MFAVerifyPayload } from './mfa/api';

export {
  getSessions,
  revokeSession,
  getAdminSessionList,
  revokeAdminSession,
  cleanupAdminSessions,
  batchRevokeAdminSessions,
  logout,
  reportActivity,
} from './session/api';
export type {
  AuthSession,
  AuthSessionPayload,
  AdminSessionRow,
  AdminSessionQuery,
  AdminSessionPageResp,
  SessionCleanupPayload,
  SessionBatchRevokePayload,
} from './session/api';

export {
  acknowledgeSecurityEvent,
  getSecurityOverview,
  getOwnLoginLogs,
  getAdminLoginLogList,
  getAdminSecurityEventList,
  exportAdminLoginLogs,
  exportSelectedAdminLoginLogs,
  cleanupAdminLoginLogs,
  batchDeleteAdminLoginLogs,
  updatePassword,
  getMe,
  verifyOperationPassword,
  updateCurrentUserPreferences,
} from './security/api';
export type {
  SecurityOverview,
  SecurityPolicy,
  SecurityEventRow,
  SecurityEventQuery,
  SecurityEventPageResp,
  SecurityEventAcknowledgePayload,
  LoginLogRow,
  LoginLogQuery,
  LoginLogCleanupPayload,
  LoginLogBatchDeletePayload,
  LoginLogPageResp,
  UserPasswordUpdatePayload,
  UserInfo,
  UserPlatformPreferences,
} from './security/api';

export { formatClientSummary, renderClientInfo } from './session/clientInfo';

export const AuthModule = defineModule({
  name: 'auth',
  scope: 'system',
  routes: [
    {
      path: 'auth/security',
      routeName: 'auth-security',
      titleKey: 'auth.security.title',
      icon: 'safe',
      componentKey: 'auth/SecurityCenter',
    },
    {
      path: 'system/login-log',
      routeName: 'system-login-log',
      titleKey: 'system.menu.loginLog',
      icon: 'clock',
      pagePermission: 'system:login-log:list',
      componentKey: 'auth/LoginLogList',
    },
    {
      path: 'system/session',
      routeName: 'system-session',
      titleKey: 'system.menu.session',
      icon: 'desktop',
      pagePermission: 'system:session:list',
      componentKey: 'auth/SessionList',
    },
    {
      path: 'system/security-event',
      routeName: 'system-security-event',
      titleKey: 'system.menu.securityEvent',
      icon: 'safe',
      pagePermission: 'system:security-event:list',
      componentKey: 'auth/SecurityEventList',
    },
  ],
  menus: [
    {
      path: '/system/login-log',
      titleKey: 'system.menu.loginLog',
      icon: 'clock',
      routeName: 'system-login-log',
      module: 'system.auth',
    },
    {
      path: '/system/session',
      titleKey: 'system.menu.session',
      icon: 'desktop',
      routeName: 'system-session',
      module: 'system.auth',
    },
    {
      path: '/system/security-event',
      titleKey: 'system.menu.securityEvent',
      icon: 'safe',
      routeName: 'system-security-event',
      module: 'system.auth',
    },
  ],
  routeDataWarmers: [
    { path: '/auth/security', key: 'overview', load: () => getSecurityOverview() },
    { path: '/auth/security', key: 'sessions', load: () => getSessions() },
    {
      path: '/auth/security',
      key: 'login-logs',
      load: () => getOwnLoginLogs({ page: 1, pageSize: 10 }),
    },
  ],
  dashboardWidgets: [
    quickActionWidget(
      'platform.login-log',
      'system.menu.loginLog',
      'dashboard.quickAction.loginLog',
      '/system/login-log',
      'system:login-log:list',
      'clock',
    ),
    quickActionWidget(
      'platform.session',
      'system.menu.session',
      'dashboard.quickAction.session',
      '/system/session',
      'system:session:list',
      'desktop',
    ),
    quickActionWidget(
      'platform.security-event',
      'system.menu.securityEvent',
      'dashboard.quickAction.securityEvent',
      '/system/security-event',
      'system:security-event:list',
      'safe',
    ),
    {
      key: 'platform.domain.security',
      slot: 'domain-overview',
      sourceDomain: 'system/auth',
      titleKey: 'dashboard.domain.security',
      descriptionKey: 'dashboard.domain.securityDesc',
      path: '/system/security-event',
      permission: 'system:security-event:list',
      cleanupPolicy: 'hide_when_forbidden',
      summary: (summary, t) =>
        t('dashboard.securityDomainSummary', {
          pending: summary?.pendingSecurityEventCount ?? 0,
          failed: summary?.loginFailureCount ?? 0,
        }),
    },
  ],
  permissions: [
    'system:login-log:list',
    'system:login-log:export',
    'system:login-log:clear',
    'system:login-log:delete',
    'system:session:list',
    'system:session:delete',
    'system:session:clear',
    'system:security-event:list',
    'system:security-event:acknowledge',
    'system:security-event:clear',
  ],
  i18nNamespaces: ['auth', 'system.menu'],
});
