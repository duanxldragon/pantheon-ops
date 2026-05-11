import { defineModule } from '../../../core/router/types';

export const UserModule = defineModule({
  name: 'user',
  scope: 'system',
  routes: [
    {
      path: 'system/user',
      routeName: 'system-user',
      titleKey: 'system.menu.user',
      icon: 'user',
      pagePermission: 'system:user:list',
      componentKey: 'system/user/UserList',
    },
    {
      path: 'system/user/:id',
      routeName: 'system-user-detail',
      titleKey: 'system.user.detail',
      pagePermission: 'system:user:view',
      activeMenu: '/system/user',
      componentKey: 'system/user/UserDetail',
    },
  ],
  menus: [
    {
      path: '/system/user',
      titleKey: 'system.menu.user',
      icon: 'user',
      routeName: 'system-user',
      module: 'system.iam',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.users',
      slot: 'quick-action',
      sourceDomain: 'system/iam',
      titleKey: 'system.menu.user',
      descriptionKey: 'dashboard.quickAction.user',
      path: '/system/user',
      permission: 'system:user:list',
      icon: 'user',
      cleanupPolicy: 'hide_when_forbidden',
    },
    {
      key: 'platform.domain.access',
      slot: 'domain-overview',
      sourceDomain: 'system/iam',
      titleKey: 'dashboard.domain.access',
      descriptionKey: 'dashboard.domain.accessDesc',
      path: '/system/user',
      permission: 'system:user:list',
      cleanupPolicy: 'hide_when_forbidden',
      summary: (summary, t) =>
        t('dashboard.usersAndRoles', {
          users: summary?.totalUsers ?? 0,
          roles: summary?.totalRoles ?? 0,
        }),
    },
  ],
  permissions: [
    'system:user:list',
    'system:user:view',
    'system:user:create',
    'system:user:update',
    'system:user:delete',
    'system:user:reset',
    'system:user:export',
    'system:user:import',
    'system:user:batch-update',
    'system:user:batch-delete',
  ],
  i18nNamespaces: ['system.user', 'system.menu', 'auth'],
});
