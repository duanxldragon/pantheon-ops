import { defineModule } from '../../../core/router/types';

export const ObservabilityModule = defineModule({
  name: 'observability',
  scope: 'business',
  routes: [
    {
      path: 'business/observability/metric-source',
      routeName: 'observability-metric-source-list',
      titleKey: 'operations.observability.metric_source.menu',
      icon: 'storage',
      pagePermission: 'business:observability:list',
      componentKey: 'business/observability/MetricSourceList',
    },
    {
      path: 'business/observability/alert-rule',
      routeName: 'observability-alert-rule-list',
      titleKey: 'operations.observability.alert_rule.menu',
      icon: 'book',
      pagePermission: 'business:observability:list',
      componentKey: 'business/observability/AlertRuleList',
    },
    {
      path: 'business/observability/alert-record',
      routeName: 'observability-alert-record-list',
      titleKey: 'operations.observability.alert_record.menu',
      icon: 'tags',
      pagePermission: 'business:observability:list',
      componentKey: 'business/observability/AlertRecordList',
    },
  ],
  menus: [
    {
      path: '/business/observability/metric-source',
      titleKey: 'operations.observability.metric_source.menu',
      icon: 'storage',
      routeName: 'observability-metric-source-list',
      module: 'business.observability',
    },
    {
      path: '/business/observability/alert-rule',
      titleKey: 'operations.observability.alert_rule.menu',
      icon: 'book',
      routeName: 'observability-alert-rule-list',
      module: 'business.observability',
    },
    {
      path: '/business/observability/alert-record',
      titleKey: 'operations.observability.alert_record.menu',
      icon: 'tags',
      routeName: 'observability-alert-record-list',
      module: 'business.observability',
    },
  ],
  permissions: ['business:observability:list'],
  i18nNamespaces: ['business.observability'],
});
