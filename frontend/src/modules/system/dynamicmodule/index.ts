/**
 * 动态模块管理 - 模块注册
 */

import { defineModule } from '../../../core/router/types';

export const DynamicModuleModule = defineModule({
  name: 'dynamic-module',
  scope: 'system',
  routes: [
    {
      path: 'system/modules',
      routeName: 'system-modules',
      titleKey: 'system.menu.modules',
      icon: 'apps',
      pagePermission: 'system:module:list',
      componentKey: 'system/dynamicmodule/ModuleManager',
    },
  ],
  menus: [
    {
      path: '/system/modules',
      titleKey: 'system.menu.modules',
      icon: 'apps',
      routeName: 'system-modules',
      module: 'system.dynamic-module',
    },
  ],
  permissions: [
    'system:module:list',
    'system:module:register',
    'system:module:unregister',
    'system:module:delete_record',
    'system:module:purge',
    'system:module:repair',
  ],
  i18nNamespaces: ['system.dynamic-module', 'system.menu'],
});
