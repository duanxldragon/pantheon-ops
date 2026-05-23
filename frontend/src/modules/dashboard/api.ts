import { apiRequest } from '../../api/request';

export interface DashboardRecentLogin {
  id: number;
  username: string;
  ipaddr: string;
  browser: string;
  os: string;
  status: number;
  msg: string;
  loginTime: string;
}

export interface DashboardTodoItem {
  taskKey: string;
  domain: string;
  scopeLabel: string;
  issueLabel: string;
  actionLabel: string;
  resourceLabel: string;
  relatedUserCount: number;
  routePath: string;
  routeStateDeptId: number;
}

export interface DashboardSummary {
  totalUsers: number;
  enabledUsers: number;
  totalRoles: number;
  totalDepts: number;
  totalPosts: number;
  totalDictTypes: number;
  totalSettings: number;
  totalI18nEntries: number;
  activeModuleCount: number;
  visibleMenuCount: number;
  activeSessionCount: number;
  loginSuccessCount: number;
  loginFailureCount: number;
  totalSecurityEventCount: number;
  pendingSecurityEventCount: number;
  todayOperationCount: number;
  lastSuccessfulLoginAt: string;
  periodDays: number;
  recentLogins: DashboardRecentLogin[];
  orgGovernanceTaskCount: number;
  orgGovernanceTasks: DashboardTodoItem[];
}

export function getDashboardSummary() {
  return apiRequest<DashboardSummary>({
    url: '/platform/dashboard/summary',
    method: 'get',
    skipErrorMessage: true,
  });
}
