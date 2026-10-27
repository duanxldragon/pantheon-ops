import { defineModule } from '../../../../core/router/types';

export const TenantModule = defineModule({
  name: 'tenant',
  scope: 'system',
  routes: [
    {
      path: 'system/tenant-ops/overview',
      routeName: 'system-tenant-overview',
      titleKey: 'system.menu.tenantOverview',
      icon: 'dashboard',
      pagePermission: 'system:tenant:overview',
      componentKey: 'system/tenant/TenantOverview',
    },
    {
      path: 'system/tenant',
      routeName: 'system-tenant',
      titleKey: 'system.menu.tenant',
      icon: 'storage',
      pagePermission: 'system:tenant:list',
      componentKey: 'system/tenant/TenantList',
    },
    {
      path: 'system/tenant-ops/member',
      routeName: 'system-tenant-member',
      titleKey: 'system.menu.tenantMember',
      icon: 'user-group',
      pagePermission: 'system:tenant:member-list',
      componentKey: 'system/tenant/TenantMemberList',
    },
  ],
  menus: [
    {
      path: '/system/tenant-ops/overview',
      titleKey: 'system.menu.tenantOverview',
      icon: 'dashboard',
      routeName: 'system-tenant-overview',
      module: 'system.tenant',
    },
    {
      path: '/system/tenant',
      titleKey: 'system.menu.tenant',
      icon: 'storage',
      routeName: 'system-tenant',
      module: 'system.tenant',
    },
    {
      path: '/system/tenant-ops/member',
      titleKey: 'system.menu.tenantMember',
      icon: 'user-group',
      routeName: 'system-tenant-member',
      module: 'system.tenant',
    },
  ],
  permissions: [
    'system:tenant:list',
    'system:tenant:create',
    'system:tenant:update',
    'system:tenant:delete',
    'system:tenant:member-add',
    'system:tenant:member-remove',
    'system:tenant:overview',
    'system:tenant:member-list',
    'system:tenant:plan:list',
    'system:tenant:plan:create',
    'system:tenant:plan:update',
  ],
  i18nNamespaces: ['system.tenant', 'system.menu'],
});
