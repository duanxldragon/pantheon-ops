import { defineModule } from '../../../core/router/types';

export const BizScopeModule = defineModule({
  name: 'bizscope',
  scope: 'business',
  routes: [
    {
      path: 'business/business-scope',
      routeName: 'bizscope-list',
      titleKey: 'operations.bizscope.menu',
      icon: 'apps',
      pagePermission: 'business:bizscope:list',
      componentKey: 'business/bizscope/BizScopeList',
    },
    {
      path: 'business/business-scope/:id',
      routeName: 'bizscope-detail',
      titleKey: 'operations.bizscope.detail',
      pagePermission: 'business:bizscope:view',
      activeMenu: '/business/business-scope',
      componentKey: 'business/bizscope/BizScopeDetail',
    },
  ],
  menus: [
    {
      path: '/business/business-scope',
      titleKey: 'operations.bizscope.menu',
      icon: 'apps',
      routeName: 'bizscope-list',
      module: 'business.bizscope',
    },
  ],
  permissions: [
    'business:bizscope:list',
    'business:bizscope:view',
    'business:bizscope:create',
    'business:bizscope:update',
    'business:bizscope:delete',
    'business:bizscope:export',
    'business:bizscope:import',
  ],
  i18nNamespaces: ['business.bizscope'],
});
