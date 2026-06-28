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
  | 'platform_owned'
  | 'hide_when_forbidden'
  | 'remove_with_source_module';

export type DashboardWidgetSlot = 'quick-action' | 'domain-overview';
export type DashboardWidgetNavigationSource = 'menu' | 'direct';

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
}

export interface DashboardQuickActionWidget extends DashboardWidgetBase {
  slot: 'quick-action';
  icon: string;
}

export interface DashboardDomainOverviewWidget extends DashboardWidgetBase {
  slot: 'domain-overview';
  summary: (summary: DashboardSummarySnapshot | null, t: TFunction) => string;
}

export type DashboardWidgetDefinition = DashboardQuickActionWidget | DashboardDomainOverviewWidget;

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
