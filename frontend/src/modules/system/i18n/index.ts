import { defineModule } from '../../../core/router/types';

export const I18nModule = defineModule({
  name: 'i18n',
  scope: 'system',
  routes: [
    {
      path: 'system/i18n',
      routeName: 'system-i18n',
      titleKey: 'system.menu.i18n',
      icon: 'language',
      pagePermission: 'system:i18n:list',
      componentKey: 'system/i18n/I18nList',
    },
  ],
  menus: [
    {
      path: '/system/i18n',
      titleKey: 'system.menu.i18n',
      icon: 'language',
      routeName: 'system-i18n',
      module: 'system.config',
    },
  ],
  permissions: [
    'system:i18n:list',
    'system:i18n:create',
    'system:i18n:update',
    'system:i18n:delete',
    'system:i18n:export',
    'system:i18n:import',
    'system:i18n:refresh',
  ],
  i18nNamespaces: ['system.i18n', 'system.menu', 'system.permission'],
});
