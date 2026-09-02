// Lowcode 模块定义 - 从 base 导入核心功能
import { defineModule } from '@duanxldragon/pantheon-base-ui';

export const LowcodeModule = defineModule({
  name: 'lowcode',
  scope: 'system',
  routes: [
    {
      path: 'system/modules',
      routeName: 'system-modules',
      titleKey: 'system.menu.modules',
      icon: 'code',
      pagePermission: 'system:module:list',
      componentKey: 'lowcode/dynamicmodule/ModuleManager',
    },
    {
      path: 'system/generator',
      routeName: 'system-generator',
      titleKey: 'system.menu.generator',
      icon: 'robot',
      pagePermission: 'system:generator:use',
      componentKey: 'lowcode/generator/ModuleWizard',
    },
  ],
  menus: [
    {
      path: '/system/modules',
      titleKey: 'system.menu.modules',
      icon: 'code',
      routeName: 'system-modules',
      module: 'system.lowcode',
    },
    {
      path: '/system/generator',
      titleKey: 'system.menu.generator',
      icon: 'robot',
      routeName: 'system-generator',
      module: 'system.lowcode',
    },
  ],
  permissions: [
    'system:module:list',
    'system:module:register',
    'system:module:unregister',
    'system:module:delete_record',
    'system:module:purge',
    'system:module:repair',
    'system:module:generate',
    'system:generator:use',
    'system:generator:datasource:manage',
  ],
  i18nNamespaces: ['lowcode', 'system.menu'],
});
