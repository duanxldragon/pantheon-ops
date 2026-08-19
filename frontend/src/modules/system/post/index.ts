import { defineModule } from '../../../core/router/types';
import { getDeptTree } from '../dept/api';
import { getPostList } from './api';

export const PostModule = defineModule({
  name: 'post',
  scope: 'system',
  routes: [
    {
      path: 'system/post',
      routeName: 'system-post',
      titleKey: 'system.menu.post',
      icon: 'tags',
      pagePermission: 'system:post:list',
      componentKey: 'system/post/PostList',
    },
  ],
  menus: [
    {
      path: '/system/post',
      titleKey: 'system.menu.post',
      icon: 'tags',
      routeName: 'system-post',
      module: 'system.org',
    },
  ],
  routeDataWarmers: [
    {
      path: '/system/post',
      key: 'list:default',
      load: () => getPostList({ page: 1, pageSize: 10 }),
    },
    {
      path: '/system/post',
      key: 'depts:sorted',
      load: () => getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
    },
  ],
  dashboardWidgets: [
    {
      key: 'platform.post',
      slot: 'quick-action',
      sourceDomain: 'system/org',
      titleKey: 'system.menu.post',
      descriptionKey: 'dashboard.quickAction.post',
      path: '/system/post',
      permission: 'system:post:list',
      icon: 'tags',
      cleanupPolicy: 'hide_when_forbidden',
    },
  ],
  permissions: [
    'system:post:list',
    'system:post:create',
    'system:post:update',
    'system:post:delete',
    'system:post:export',
    'system:post:import',
    'system:post:batch-update',
    'system:post:batch-delete',
  ],
  i18nNamespaces: ['system.post', 'system.menu', 'system.permission'],
});
