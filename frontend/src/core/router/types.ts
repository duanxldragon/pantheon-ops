import type { LazyExoticComponent, ComponentType } from 'react';
import type { RegisteredComponentKey } from './componentRegistry';
import type { MenuIconKey } from '../menu/icon';
import type { TFunction } from 'i18next';

export type ModuleScope = 'platform' | 'system' | 'business' | 'lowcode';

export type DashboardWidgetSourceDomain =
  | 'platform'
  | 'system/auth'
  | 'system/iam'
  | 'system/org'
  | 'system/config'
  | 'system/lowcode'
  | 'system/audit'
  | `business/${string}`;

export type DashboardWidgetCleanupPolicy =
  'platform_owned' | 'hide_when_forbidden' | 'remove_with_source_module';

export type DashboardWidgetSlot =
  | 'quick-action'
  | 'domain-overview'
  | 'status-summary'
  | 'attention-queue'
  | 'trend-snapshot'
  | 'recent-activity';
export type DashboardWidgetNavigationSource = 'menu' | 'direct';
export type DashboardWidgetFreshnessPolicy = 'static' | 'snapshot' | 'near_realtime';

export interface DashboardWidgetQueryBudget {
  maxRequestsPerMinute: number;
  maxItems: number;
  maxRenderItems?: number;
}

export interface DashboardWidgetFreshness {
  policy: DashboardWidgetFreshnessPolicy;
  maxAgeMs?: number;
  refreshIntervalMs?: number;
  timeRangeKey?: string;
}

export interface DashboardWidgetOperationalMetadata {
  owner: string;
  cleanupPolicy: DashboardWidgetCleanupPolicy;
  freshness: DashboardWidgetFreshness;
  queryBudget: DashboardWidgetQueryBudget;
  emptyStateKey: string;
  errorStateKey: string;
  errorIsolation: 'widget';
}

export interface DashboardSummarySnapshot {
  totalUsers?: number;
  totalRoles?: number;
  totalDepts?: number;
  totalPosts?: number;
  totalDictTypes?: number;
  totalSettings?: number;
  totalI18nEntries?: number;
  activeModuleCount?: number;
  pendingSecurityEventCount?: number;
  totalSecurityEventCount?: number;
  todayOperationCount?: number;
  loginFailureCount?: number;
}

interface DashboardWidgetBase {
  key: string;
  slot: DashboardWidgetSlot;
  sourceDomain: DashboardWidgetSourceDomain;
  titleKey: string;
  descriptionKey: string;
  path: string;
  permission?: string;
  cleanupPolicy: DashboardWidgetCleanupPolicy;
  navigationSource?: DashboardWidgetNavigationSource;
  registrationOwner?: string;
  metadata?: DashboardWidgetOperationalMetadata;
}

export interface DashboardQuickActionWidget extends DashboardWidgetBase {
  slot: 'quick-action';
  icon: string;
}

export interface DashboardDomainOverviewWidget extends DashboardWidgetBase {
  slot: 'domain-overview';
  summary: (summary: DashboardSummarySnapshot | null, t: TFunction) => string;
}

export interface DashboardStatusSummaryWidget extends DashboardWidgetBase {
  slot: 'status-summary';
}

export interface DashboardAttentionQueueWidget extends DashboardWidgetBase {
  slot: 'attention-queue';
}

export interface DashboardTrendSnapshotWidget extends DashboardWidgetBase {
  slot: 'trend-snapshot';
}

export interface DashboardRecentActivityWidget extends DashboardWidgetBase {
  slot: 'recent-activity';
}

export type DashboardOperationalWidget =
  | DashboardStatusSummaryWidget
  | DashboardAttentionQueueWidget
  | DashboardTrendSnapshotWidget
  | DashboardRecentActivityWidget;

export type DashboardWidgetDefinition =
  DashboardQuickActionWidget | DashboardDomainOverviewWidget | DashboardOperationalWidget;

export interface RouteDataWarmer {
  path: string;
  key: string;
  load: () => Promise<unknown>;
  ttlMs?: number;
}

interface ModuleRouteConfigBase {
  path: string;
  routeName?: string;
  titleKey: string;
  resolveTitleKey?: (path: string) => string | undefined;
  icon?: MenuIconKey;
  isCache?: boolean;
  activeMenu?: string;
  pagePermission?: string;
}

export type ModuleRouteConfig = ModuleRouteConfigBase &
  (
    | {
        component: LazyExoticComponent<ComponentType>;
        componentKey?: RegisteredComponentKey;
      }
    | {
        component?: undefined;
        componentKey: RegisteredComponentKey;
      }
  );

export interface ModuleMenuMeta {
  titleKey: string;
  path: string;
  icon?: MenuIconKey;
  routeName?: string;
  module?: string;
  isCache?: boolean;
  isExternal?: boolean;
  activeMenu?: string;
}

export interface ModuleConfig {
  name: string;
  scope: ModuleScope;
  routes: ModuleRouteConfig[];
  menus?: ModuleMenuMeta[];
  routeDataWarmers?: RouteDataWarmer[];
  dashboardWidgets?: DashboardWidgetDefinition[];
  permissions?: string[];
  i18nNamespaces?: string[];
  featureFlags?: string[];
}

export function defineModule(config: ModuleConfig): ModuleConfig {
  return config;
}
