import { defineModule } from '../../core/router/types';

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
  permissions: [
    'system:login-log:list',
    'system:login-log:export',
    'system:login-log:clear',
    'system:login-log:delete',
    'system:session:list',
    'system:session:delete',
    'system:session:clear',
    'system:security-event:list',
  ],
  i18nNamespaces: ['auth', 'system.menu'],
});
