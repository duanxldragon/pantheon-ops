import { defineModule } from '../../../core/router/types';

export const ServiceModule = defineModule({
  name: 'service',
  scope: 'business',
  routes: [
    {
      path: 'business/service',
      routeName: 'service-list',
      titleKey: 'operations.service.menu',
      icon: 'apps',
      pagePermission: 'business:service:list',
      componentKey: 'business/service/ServiceList',
    },
  ],
  menus: [
    {
      path: '/business/service',
      titleKey: 'operations.service.menu',
      icon: 'apps',
      routeName: 'service-list',
      module: 'business.service',
    },
  ],
  permissions: [
    'business:service:list',
    'business:service:view',
    'business:service:create',
    'business:service:update',
    'business:service:delete',
  ],
  i18nNamespaces: ['business.service'],
});
