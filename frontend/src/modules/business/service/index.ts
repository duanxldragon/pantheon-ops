import { defineModule } from '@duanxldragon/pantheon-base-ui';

export const ServiceModule = defineModule({
  name: 'service',
  scope: 'business',
  routes: [
    {
      path: 'business/service-catalog/applications',
      routeName: 'service-application-list',
      titleKey: 'operations.service.application.menu',
      icon: 'apps',
      pagePermission: 'business:service:list',
      componentKey: 'business/service/ApplicationList',
    },
    {
      path: 'business/service-catalog/services',
      routeName: 'service-service-list',
      titleKey: 'operations.service.service.menu',
      pagePermission: 'business:service:list',
      componentKey: 'business/service/ServiceList',
    },
    {
      path: 'business/service-catalog/instances',
      routeName: 'service-instance-list',
      titleKey: 'operations.service.instance.menu',
      pagePermission: 'business:service:list',
      componentKey: 'business/serviceinstance/ServiceInstanceList',
    },
  ],
  menus: [
    {
      path: '/business/service-catalog/applications',
      titleKey: 'operations.service.application.menu',
      icon: 'apps',
      routeName: 'service-application-list',
      module: 'business.service',
    },
    {
      path: '/business/service-catalog/services',
      titleKey: 'operations.service.service.menu',
      icon: 'apps',
      routeName: 'service-service-list',
      module: 'business.service',
    },
    {
      path: '/business/service-catalog/instances',
      titleKey: 'operations.service.instance.menu',
      icon: 'apps',
      routeName: 'service-instance-list',
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
