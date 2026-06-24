import { defineModule } from '../../../../core/router/types';

export const DeptModule = defineModule({
  name: 'dept',
  scope: 'system',
  routes: [
    {
      path: 'system/dept',
      routeName: 'system-dept',
      titleKey: 'system.menu.dept',
      icon: 'branch',
      pagePermission: 'system:dept:list',
      componentKey: 'system/org/dept/DeptList',
    },
  ],
  menus: [
    {
      path: '/system/dept',
      titleKey: 'system.menu.dept',
      icon: 'branch',
      routeName: 'system-dept',
      module: 'system.org',
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.domain.org',
      slot: 'domain-overview',
      sourceDomain: 'system/org',
      titleKey: 'dashboard.domain.org',
      descriptionKey: 'dashboard.domain.orgDesc',
      path: '/system/dept',
      permission: 'system:dept:list',
      cleanupPolicy: 'hide_when_forbidden',
      summary: (summary, t) =>
        t('dashboard.deptsAndPosts', {
          depts: summary?.totalDepts ?? 0,
          posts: summary?.totalPosts ?? 0,
        }),
    },
  ],
  permissions: [
    'system:dept:list',
    'system:dept:create',
    'system:dept:update',
    'system:dept:delete',
    'system:dept:export',
    'system:dept:import',
    'system:dept:batch-update',
    'system:dept:batch-delete',
  ],
  i18nNamespaces: ['system.dept', 'system.menu', 'system.permission'],
});
