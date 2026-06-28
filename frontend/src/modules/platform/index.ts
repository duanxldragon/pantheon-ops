import { defineModule } from '../../core/router/types';
import { getDashboardSummary } from './api';

export const PlatformModule = defineModule({
  name: 'platform',
  scope: 'platform',
  routes: [
    {
      path: 'dashboard',
      routeName: 'dashboard',
      titleKey: 'system.menu.dashboard',
      icon: 'dashboard',
      pagePermission: 'platform:dashboard:view',
      componentKey: 'dashboard',
    },
  ],
  menus: [
    {
      path: '/dashboard',
      titleKey: 'system.menu.dashboard',
      icon: 'dashboard',
      routeName: 'dashboard',
      module: 'platform',
    },
  ],
  routeDataWarmers: [{ path: '/dashboard', key: 'summary', load: () => getDashboardSummary() }],
  dashboardWidgets: [
    {
      key: 'platform.security',
      slot: 'quick-action',
      sourceDomain: 'system/auth',
      titleKey: 'auth.security.title',
      descriptionKey: 'dashboard.quickAction.security',
      path: '/auth/security',
      icon: 'safe',
      cleanupPolicy: 'platform_owned',
      navigationSource: 'direct',
    },
  ],
  permissions: ['platform:dashboard:view'],
  i18nNamespaces: ['dashboard', 'system.menu'],
});
