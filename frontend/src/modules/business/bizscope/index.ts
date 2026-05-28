import { defineModule } from '../../../core/router/types';

export const BizScopeModule = defineModule({
  name: 'bizscope',
  scope: 'business',
  routes: [
    {
      path: 'operations/business-scope',
      routeName: 'bizscope-list',
      titleKey: 'operations.bizscope.menu',
      icon: 'apps',
      pagePermission: 'business:bizscope:list',
      componentKey: 'business/bizscope/BizScopeList',
    },
    {
      path: 'operations/business-scope/:id',
      routeName: 'bizscope-detail',
      titleKey: 'operations.bizscope.detail',
      pagePermission: 'business:bizscope:view',
      activeMenu: '/operations/business-scope',
      componentKey: 'business/bizscope/BizScopeDetail',
    },
  ],
  menus: [
    {
      path: '/operations/business-scope',
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
  ],
  i18nNamespaces: ['business.bizscope'],
});
