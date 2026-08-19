import { defineModule } from '../../../core/router/types';
import { getPostList } from '../post/api';
import { getUserList } from '../user/api';
import { getDeptOverview, getDeptTree } from './api';

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
      componentKey: 'system/dept/DeptList',
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
  routeDataWarmers: [
    { path: '/system/dept', key: 'tree:default', load: () => getDeptTree({}) },
    { path: '/system/dept', key: 'overview', load: () => getDeptOverview() },
    {
      path: '/system/dept',
      key: 'tree:sorted',
      load: () => getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
    },
    {
      path: '/system/dept',
      key: 'posts:org-chart',
      load: () => getPostList({ page: 1, pageSize: 1000, sortField: 'sort', sortOrder: 'asc' }),
    },
    {
      path: '/system/dept',
      key: 'users:org-chart',
      load: () => getUserList({ page: 1, pageSize: 1000, sortField: 'username', sortOrder: 'asc' }),
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
