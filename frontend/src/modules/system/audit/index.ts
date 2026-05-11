import { defineModule } from '../../../core/router/types';

export const AuditModule = defineModule({
  name: 'audit',
  scope: 'system',
  routes: [
    {
      path: 'system/operation-log',
      routeName: 'system-operation-log',
      titleKey: 'system.menu.operationLog',
      icon: 'file',
      pagePermission: 'system:operation-log:list',
      componentKey: 'system/audit/OperationLogList',
    },
  ],
  menus: [
    {
      path: '/system/operation-log',
      titleKey: 'system.menu.operationLog',
      icon: 'file',
      routeName: 'system-operation-log',
      module: 'system.audit',
    },
  ],
  permissions: [
    'system:operation-log:list',
    'system:operation-log:delete',
    'system:operation-log:clear',
    'system:operation-log:export',
  ],
  i18nNamespaces: ['system.audit'],
});
