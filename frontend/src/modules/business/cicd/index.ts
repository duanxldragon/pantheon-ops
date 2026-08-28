import { defineModule } from '../../../core/router/types';

export const CICDModule = defineModule({
  name: 'cicd',
  scope: 'business',
  routes: [
    {
      path: 'business/cicd/argocd',
      routeName: 'cicd-argocd-list',
      titleKey: 'operations.cicd.argocd.menu',
      icon: 'code',
      pagePermission: 'business:cicd:argocd:list',
      componentKey: 'business/cicd/argocd/ArgoCDList',
    },
  ],
  menus: [
    {
      path: '/business/cicd/argocd',
      titleKey: 'operations.cicd.argocd.menu',
      icon: 'code',
      routeName: 'cicd-argocd-list',
      module: 'business.cicd',
    },
  ],
  permissions: [
    'business:cicd:argocd:list',
    'business:cicd:argocd:view',
    'business:cicd:argocd:create',
    'business:cicd:argocd:delete',
  ],
  i18nNamespaces: ['business.cicd'],
});
