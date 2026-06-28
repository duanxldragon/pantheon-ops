import { defineModule } from '../../../core/router/types';
import { getMenuTree } from '../menu/api';
import { getRoleList } from './api';

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
      componentKey: 'system/role/RoleList',
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
  routeDataWarmers: [
    {
      path: '/system/role',
      key: 'list:default',
      load: () => getRoleList({ roleName: '', roleKey: '', page: 1, pageSize: 10 }),
    },
    { path: '/system/role', key: 'menus:manage', load: () => getMenuTree({ scope: 'manage' }) },
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
