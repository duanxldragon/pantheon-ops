import { defineModule } from '../../../core/router/types';
import { getMenuTree } from './api';

export const MenuModule = defineModule({
  name: 'menu',
  scope: 'system',
  routes: [
    {
      path: 'system/menu',
      routeName: 'system-menu',
      titleKey: 'system.menu.menu',
      icon: 'menu',
      pagePermission: 'system:menu:list',
      componentKey: 'system/menu/MenuList',
    },
  ],
  menus: [
    {
      path: '/system/menu',
      titleKey: 'system.menu.menu',
      icon: 'menu',
      routeName: 'system-menu',
      module: 'system.iam',
    },
  ],
  routeDataWarmers: [
    { path: '/system/menu', key: 'tree:manage', load: () => getMenuTree({ scope: 'manage' }) },
  ],
  dashboardWidgets: [
    {
      key: 'platform.menus',
      slot: 'quick-action',
      sourceDomain: 'system/iam',
      titleKey: 'system.menu.menu',
      descriptionKey: 'dashboard.quickAction.menu',
      path: '/system/menu',
      permission: 'system:menu:list',
      icon: 'menu',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:menu:list',
    'system:menu:create',
    'system:menu:update',
    'system:menu:delete',
  ],
  i18nNamespaces: ['system.menu', 'system.permission'],
});
