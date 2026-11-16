import { defineModule } from '@duanxldragon/pantheon-base-ui';

export const ServiceModule = defineModule({
  name: 'service',
  scope: 'business',
  routes: [
    {
      path: 'business/service-catalog',
      routeName: 'service-list',
      titleKey: 'operations.service.menu',
      icon: 'apps',
      pagePermission: 'business:service:list',
      componentKey: 'business/service/ServiceCatalog',
    },
    {
      path: 'business/service-catalog/applications',
      routeName: 'service-applications',
      titleKey: 'operations.service.application.menu',
      icon: 'apps',
      // Tab anchor of ServiceCatalog; the parent route owns business:service:list.
      activeMenu: '/business/service-catalog',
      componentKey: 'business/service/ServiceCatalog',
    },
    {
      path: 'business/service-catalog/services',
      routeName: 'service-services',
      titleKey: 'operations.service.service.menu',
      icon: 'apps',
      activeMenu: '/business/service-catalog',
      componentKey: 'business/service/ServiceCatalog',
    },
    {
      path: 'business/service-catalog/instances',
      routeName: 'service-instances',
      titleKey: 'operations.service.instance.menu',
      icon: 'apps',
      activeMenu: '/business/service-catalog',
      componentKey: 'business/service/ServiceCatalog',
    },
  ],
  menus: [
    {
      path: '/business/service-catalog',
      titleKey: 'operations.service.menu',
      icon: 'apps',
      routeName: 'service-list',
      module: 'business.service',
    },
    {
      path: '/business/service-catalog/applications',
      titleKey: 'operations.service.application.menu',
      icon: 'apps',
      routeName: 'service-applications',
      module: 'business.service',
    },
    {
      path: '/business/service-catalog/services',
      titleKey: 'operations.service.service.menu',
      icon: 'apps',
      routeName: 'service-services',
      module: 'business.service',
    },
    {
      path: '/business/service-catalog/instances',
      titleKey: 'operations.service.instance.menu',
      icon: 'apps',
      routeName: 'service-instances',
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
