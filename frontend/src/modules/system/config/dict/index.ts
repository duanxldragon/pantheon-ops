import { defineModule } from '../../../../core/router/types';

export const DictModule = defineModule({
  name: 'dict',
  scope: 'system',
  routes: [
    {
      path: 'system/dict',
      routeName: 'system-dict',
      titleKey: 'system.menu.dict',
      icon: 'book',
      pagePermission: 'system:dict:list',
      componentKey: 'system/config/dict/DictPage',
    },
  ],
  menus: [
    {
      path: '/system/dict',
      titleKey: 'system.menu.dict',
      icon: 'book',
      routeName: 'system-dict',
      module: 'system.config',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.dict',
      slot: 'quick-action',
      sourceDomain: 'system/config',
      titleKey: 'system.menu.dict',
      descriptionKey: 'dashboard.quickAction.dict',
      path: '/system/dict',
      permission: 'system:dict:list',
      icon: 'storage',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:dict:list',
    'system:dict:create',
    'system:dict:update',
    'system:dict:delete',
    'system:dict:refresh',
    'system:dict:export',
    'system:dict:import',
    'system:dict:batch-update',
    'system:dict:batch-delete',
  ],
  i18nNamespaces: ['system.dict', 'system.menu'],
});
