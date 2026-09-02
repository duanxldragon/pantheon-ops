// Platform 模块定义 - 从 base 导入
import { defineModule } from '@duanxldragon/pantheon-base-ui';

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
  permissions: ['platform:dashboard:view'],
  i18nNamespaces: ['dashboard', 'system.menu'],
});
