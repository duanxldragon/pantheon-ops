import { defineModule } from '../../../../core/router/types';

export const PermissionModule = defineModule({
  name: 'permission',
  scope: 'system',
  routes: [
    {
      path: 'system/permission',
      routeName: 'system-permission',
      titleKey: 'system.menu.permission',
      icon: 'lock',
      pagePermission: 'system:permission:list',
      componentKey: 'system/iam/permission/PermissionList',
    },
  ],
  menus: [
    {
      path: '/system/permission',
      titleKey: 'system.menu.permission',
      icon: 'lock',
      routeName: 'system-permission',
      module: 'system.iam',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.permission',
      slot: 'quick-action',
      sourceDomain: 'system/iam',
      titleKey: 'system.menu.permission',
      descriptionKey: 'dashboard.quickAction.permission',
      path: '/system/permission',
      permission: 'system:permission:list',
      icon: 'lock',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:permission:list',
    'system:permission:create',
    'system:permission:update',
    'system:permission:delete',
    'system:permission:batch-delete',
    'system:permission:export',
    'system:permission:import',
  ],
  i18nNamespaces: ['system.permission', 'system.menu'],
});
