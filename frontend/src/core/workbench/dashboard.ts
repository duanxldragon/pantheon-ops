import type { TFunction } from 'i18next';

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

interface DashboardWidgetModuleLike {
  name: string;
  dashboardWidgets?: DashboardWidgetDefinition[];
}

function assertDashboardWidgetDefinition(widget: DashboardWidgetDefinition) {
  if (!widget.key.trim()) {
    throw new Error('Dashboard widget key is required.');
  }
  if (!widget.path.startsWith('/')) {
    throw new Error(`Dashboard widget "${widget.key}" must use an absolute route path.`);
  }
  if (widget.sourceDomain.startsWith('business/')) {
    if (!widget.permission) {
      throw new Error(`Business dashboard widget "${widget.key}" must declare a permission.`);
    }
    if (!widget.registrationOwner?.trim()) {
      throw new Error(
        `Business dashboard widget "${widget.key}" must declare a registration owner.`,
      );
    }
    if (widget.cleanupPolicy !== 'remove_with_source_module') {
      throw new Error(
        `Business dashboard widget "${widget.key}" must declare remove_with_source_module cleanup.`,
      );
    }
  }
}

export function buildDashboardWidgetRegistry(
  modules: DashboardWidgetModuleLike[],
): DashboardWidgetDefinition[] {
  const widgets: DashboardWidgetDefinition[] = [];
  const keys = new Set<string>();

  modules.forEach((module) => {
    module.dashboardWidgets?.forEach((widget) => {
      assertDashboardWidgetDefinition(widget);
      if (keys.has(widget.key)) {
        throw new Error(
          `Duplicate dashboard widget key "${widget.key}" declared by module "${module.name}".`,
        );
      }
      keys.add(widget.key);
      widgets.push(widget);
    });
  });

  return widgets;
}
