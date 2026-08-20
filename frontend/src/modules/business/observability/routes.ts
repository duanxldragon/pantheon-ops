import type { RouteRecordRaw } from 'vue-router';

const observabilityRoutes: RouteRecordRaw[] = [
  {
    path: '/business/observability',
    name: 'BusinessObservability',
    redirect: '/business/observability/metrics',
    meta: {
      title: '可观测性',
      icon: 'icon-eye',
      requiresAuth: true,
      order: 50,
    },
    children: [
      {
        path: 'metrics',
        name: 'BusinessObservabilityMetrics',
        redirect: '/business/observability/metrics/sources',
        meta: {
          title: '指标监控',
          icon: 'icon-line-chart',
          requiresAuth: true,
        },
        children: [
          {
            path: 'sources',
            name: 'BusinessObservabilityMetricSources',
            component: () => import('./views/MetricSourceList.vue'),
            meta: {
              title: '指标源管理',
              requiresAuth: true,
              permission: 'observability:metric_source:view',
            },
          },
        ],
      },
      {
        path: 'alerts',
        name: 'BusinessObservabilityAlerts',
        redirect: '/business/observability/alerts/rules',
        meta: {
          title: '告警管理',
          icon: 'icon-notification',
          requiresAuth: true,
        },
        children: [
          {
            path: 'rules',
            name: 'BusinessObservabilityAlertRules',
            component: () => import('./views/AlertRuleList.vue'),
            meta: {
              title: '告警规则',
              requiresAuth: true,
              permission: 'observability:alert_rule:view',
            },
          },
          {
            path: 'records',
            name: 'BusinessObservabilityAlertRecords',
            component: () => import('./views/AlertRecordList.vue'),
            meta: {
              title: '告警历史',
              requiresAuth: true,
              permission: 'observability:alert_record:view',
            },
          },
          {
            path: 'active',
            name: 'BusinessObservabilityActiveAlerts',
            component: () => import('./views/ActiveAlertList.vue'),
            meta: {
              title: '活跃告警',
              requiresAuth: true,
              permission: 'observability:alert_record:view',
            },
          },
          {
            path: 'channels',
            name: 'BusinessObservabilityNotificationChannels',
            component: () => import('./views/NotificationChannelList.vue'),
            meta: {
              title: '通知渠道',
              requiresAuth: true,
              permission: 'observability:notification_channel:view',
            },
          },
        ],
      },
    ],
  },
];

export default observabilityRoutes;
