import { defineModule } from '../../../../core/router/types';

export const RoleModule = defineModule({
  name: 'role',
  scope: 'system',
  routes: [
    {
      path: 'system/role',
      routeName: 'system-role',
      titleKey: 'system.menu.role',
      icon: 'user-group',
      pagePermission: 'system:role:list',
      componentKey: 'system/iam/role/RoleList',
    },
  ],
  menus: [
    {
      path: '/system/role',
      titleKey: 'system.menu.role',
      icon: 'user-group',
      routeName: 'system-role',
      module: 'system.iam',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.roles',
      slot: 'quick-action',
      sourceDomain: 'system/iam',
      titleKey: 'system.menu.role',
      descriptionKey: 'dashboard.quickAction.role',
      path: '/system/role',
      permission: 'system:role:list',
      icon: 'safe',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:role:list',
    'system:role:create',
    'system:role:update',
    'system:role:delete',
    'system:role:batch-update',
    'system:role:batch-delete',
    'system:role:export',
  ],
  i18nNamespaces: ['system.role', 'system.menu'],
});
