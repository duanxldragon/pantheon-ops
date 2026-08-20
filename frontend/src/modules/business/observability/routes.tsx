import type { RouteObject } from 'react-router-dom';
import {
  MetricSourceList,
  AlertRuleList,
  AlertRecordList,
  ActiveAlertList,
  NotificationChannelList,
} from './views';

const observabilityRoutes: RouteObject[] = [
  {
    path: 'observability',
    children: [
      {
        path: 'metrics/sources',
        element: <MetricSourceList />,
      },
      {
        path: 'alerts/rules',
        element: <AlertRuleList />,
      },
      {
        path: 'alerts/records',
        element: <AlertRecordList />,
      },
      {
        path: 'alerts/active',
        element: <ActiveAlertList />,
      },
      {
        path: 'alerts/channels',
        element: <NotificationChannelList />,
      },
    ],
  },
];

export default observabilityRoutes;
