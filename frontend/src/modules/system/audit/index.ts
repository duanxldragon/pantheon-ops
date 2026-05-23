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
  dashboardWidgets: [
    {
      key: 'platform.audit',
      slot: 'quick-action',
      sourceDomain: 'system/audit',
      titleKey: 'system.menu.operationLog',
      descriptionKey: 'dashboard.quickAction.audit',
      path: '/system/operation-log',
      permission: 'system:operation-log:list',
      icon: 'file',
      cleanupPolicy: 'hide_when_forbidden',
    },
    {
      key: 'platform.domain.governance',
      slot: 'domain-overview',
      sourceDomain: 'system/audit',
      titleKey: 'dashboard.domain.governance',
      descriptionKey: 'dashboard.domain.governanceDesc',
      path: '/system/operation-log',
      permission: 'system:operation-log:list',
      cleanupPolicy: 'hide_when_forbidden',
      summary: (summary, t) =>
        t('dashboard.governanceSummary', {
          operations: summary?.todayOperationCount ?? 0,
          pending: summary?.pendingSecurityEventCount ?? 0,
        }),
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
