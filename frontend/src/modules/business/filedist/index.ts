import { defineModule } from '@duanxldragon/pantheon-base-ui';

export const FiledistModule = defineModule({
  name: 'filedist',
  scope: 'business',
  routes: [
    { path: 'business/filedist/files', routeName: 'filedist-file-list', titleKey: 'operations.filedist.menu', icon: 'storage', pagePermission: 'business:filedist:list', componentKey: 'business/filedist/FiledistList' },
    { path: 'business/filedist/tasks', routeName: 'filedist-task-list', titleKey: 'operations.filedist.menu', icon: 'storage', pagePermission: 'business:filedist:list', componentKey: 'business/filedist/FiledistList' },
    { path: 'business/filedist/tasks/:id', routeName: 'filedist-task-detail', titleKey: 'operations.filedist.menu', icon: 'storage', pagePermission: 'business:filedist:list', activeMenu: '/business/filedist/files', componentKey: 'business/filedist/FiledistDetail' },
  ],
  menus: [{ path: '/business/filedist/files', titleKey: 'operations.filedist.menu', icon: 'storage', routeName: 'filedist-file-list', module: 'business.filedist' }],
  permissions: ['business:filedist:list', 'business:filedist:create', 'business:filedist:update', 'business:filedist:delete'],
  i18nNamespaces: ['business.filedist'],
});
