import { defineModule } from '../../../core/router/types';
import { getSettingList, getSettingOverview } from './api';
import { isSettingGroupKey } from './settingGroups';

function resolveSettingRouteTitleKey(path: string) {
  const groupKey = path.split('/').filter(Boolean)[2];
  return isSettingGroupKey(groupKey) ? `system.setting.group.${groupKey}` : undefined;
}

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
      componentKey: 'system/setting/SettingOverviewPage',
    },
    {
      path: 'system/setting/:groupKey',
      routeName: 'system-setting-group',
      titleKey: 'system.menu.setting',
      resolveTitleKey: resolveSettingRouteTitleKey,
      pagePermission: 'system:setting:list',
      activeMenu: '/system/setting',
      componentKey: 'system/setting/SettingGroupPage',
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
  routeDataWarmers: [
    { path: '/system/setting', key: 'list:default', load: () => getSettingList() },
    { path: '/system/setting', key: 'overview', load: () => getSettingOverview() },
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
