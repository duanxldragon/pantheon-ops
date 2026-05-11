import { defineModule } from '../../../core/router/types';

export const SettingModule = defineModule({
  name: 'setting',
  scope: 'system',
  routes: [
    {
      path: 'system/setting',
      routeName: 'system-setting',
      titleKey: 'system.menu.setting',
      icon: 'settings',
      pagePermission: 'system:setting:list',
      componentKey: 'system/setting/SettingPage',
    },
  ],
  menus: [
    {
      path: '/system/setting',
      titleKey: 'system.menu.setting',
      icon: 'settings',
      routeName: 'system-setting',
      module: 'system.config',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.setting',
      slot: 'quick-action',
      sourceDomain: 'system/config',
      titleKey: 'system.menu.setting',
      descriptionKey: 'dashboard.quickAction.setting',
      path: '/system/setting',
      permission: 'system:setting:list',
      icon: 'settings',
      cleanupPolicy: 'hide_when_forbidden',
    },
    {
      key: 'platform.domain.config',
      slot: 'domain-overview',
      sourceDomain: 'system/config',
      titleKey: 'dashboard.domain.config',
      descriptionKey: 'dashboard.domain.configDesc',
      path: '/system/setting',
      permission: 'system:setting:list',
      cleanupPolicy: 'hide_when_forbidden',
      summary: (summary, t) =>
        t('dashboard.dictAndSettings', {
          dicts: summary?.totalDictTypes ?? 0,
          settings: summary?.totalSettings ?? 0,
        }),
    },
  ],
  permissions: [
    'system:setting:list',
    'system:setting:update',
    'system:setting:refresh',
    'system:setting:export',
  ],
  i18nNamespaces: ['system.setting', 'system.menu'],
});
