import type { MenuNode } from '../system/menu/api';
import { registeredModules } from '../../core/router/modules';
import type {
  DashboardDomainOverviewWidget,
  DashboardOperationalWidget,
  DashboardWidgetCleanupPolicy,
  DashboardQuickActionWidget,
  DashboardWidgetDefinition,
  DashboardWidgetOperationalMetadata,
  DashboardWidgetSlot,
} from '../../core/router/types';

interface DashboardWidgetModuleLike {
  name: string;
  dashboardWidgets?: DashboardWidgetDefinition[];
}

interface DashboardWidgetVisibilityContext {
  menuTree: MenuNode[];
  hasPerm: (permission: string) => boolean;
  isAdmin: boolean;
}

function findMenuNodeByPath(nodes: MenuNode[], path: string): MenuNode | undefined {
  for (const item of nodes) {
    if (item.path === path || item.activeMenu === path) {
      return item;
    }
    if (item.children?.length) {
      const child = findMenuNodeByPath(item.children, path);
      if (child) {
        return child;
      }
    }
  }
  return undefined;
}

const LEGACY_DASHBOARD_SLOTS = new Set<DashboardWidgetSlot>(['quick-action', 'domain-overview']);
const OPERATIONAL_DASHBOARD_SLOTS = new Set<DashboardWidgetSlot>([
  'status-summary',
  'attention-queue',
  'trend-snapshot',
  'recent-activity',
]);
const MIN_REFRESH_INTERVAL_MS = 30_000;
const MAX_REQUESTS_PER_MINUTE = 12;
const MAX_WIDGET_ITEMS = 100;
const MAX_RENDER_ITEMS = 24;

function isOperationalDashboardSlot(slot: DashboardWidgetSlot) {
  return OPERATIONAL_DASHBOARD_SLOTS.has(slot);
}

function assertTrimmed(value: string | undefined, message: string) {
  if (!value?.trim()) {
    throw new Error(message);
  }
}

function assertCleanupPolicy(
  widget: DashboardWidgetDefinition,
  cleanupPolicy: DashboardWidgetCleanupPolicy,
) {
  if (widget.sourceDomain.startsWith('business/')) {
    if (cleanupPolicy !== 'remove_with_source_module') {
      throw new Error(
        `Business dashboard widget "${widget.key}" must declare remove_with_source_module cleanup.`,
      );
    }
    return;
  }

  if (cleanupPolicy === 'remove_with_source_module') {
    throw new Error(
      `Platform/system dashboard widget "${widget.key}" must not use business cleanup policy.`,
    );
  }
}

function assertOperationalMetadata(
  widget: DashboardWidgetDefinition,
): asserts widget is DashboardOperationalWidget {
  if (!isOperationalDashboardSlot(widget.slot)) {
    return;
  }
  if (!widget.permission) {
    throw new Error(`Operational dashboard widget "${widget.key}" must declare a permission.`);
  }
  const metadata = widget.metadata;
  if (!metadata) {
    throw new Error(`Operational dashboard widget "${widget.key}" must declare metadata.`);
  }

  assertTrimmed(metadata.owner, `Operational dashboard widget "${widget.key}" must declare owner.`);
  assertCleanupPolicy(widget, metadata.cleanupPolicy);
  assertTrimmed(
    metadata.emptyStateKey,
    `Operational dashboard widget "${widget.key}" must declare emptyStateKey.`,
  );
  assertTrimmed(
    metadata.errorStateKey,
    `Operational dashboard widget "${widget.key}" must declare errorStateKey.`,
  );

  if (metadata.errorIsolation !== 'widget') {
    throw new Error(
      `Operational dashboard widget "${widget.key}" must isolate errors at widget level.`,
    );
  }
  if (metadata.cleanupPolicy !== widget.cleanupPolicy) {
    throw new Error(
      `Operational dashboard widget "${widget.key}" cleanup policy must match metadata.`,
    );
  }
  if (metadata.freshness.refreshIntervalMs !== undefined) {
    if (
      !Number.isFinite(metadata.freshness.refreshIntervalMs) ||
      metadata.freshness.refreshIntervalMs < MIN_REFRESH_INTERVAL_MS
    ) {
      throw new Error(
        `Operational dashboard widget "${widget.key}" refresh interval is below query budget floor.`,
      );
    }
  }
  if (
    !Number.isFinite(metadata.queryBudget.maxRequestsPerMinute) ||
    metadata.queryBudget.maxRequestsPerMinute <= 0 ||
    metadata.queryBudget.maxRequestsPerMinute > MAX_REQUESTS_PER_MINUTE
  ) {
    throw new Error(`Operational dashboard widget "${widget.key}" exceeds request budget.`);
  }
  if (
    !Number.isFinite(metadata.queryBudget.maxItems) ||
    metadata.queryBudget.maxItems <= 0 ||
    metadata.queryBudget.maxItems > MAX_WIDGET_ITEMS
  ) {
    throw new Error(`Operational dashboard widget "${widget.key}" exceeds item budget.`);
  }
  const maxRenderItems = metadata.queryBudget.maxRenderItems ?? metadata.queryBudget.maxItems;
  if (!Number.isFinite(maxRenderItems) || maxRenderItems <= 0 || maxRenderItems > MAX_RENDER_ITEMS) {
    throw new Error(`Operational dashboard widget "${widget.key}" exceeds render budget.`);
  }
}

function assertDashboardWidgetDefinition(widget: DashboardWidgetDefinition) {
  if (!widget.key.trim()) {
    throw new Error('Dashboard widget key is required.');
  }
  if (!widget.path.startsWith('/')) {
    throw new Error(`Dashboard widget "${widget.key}" must use an absolute route path.`);
  }
  if (!LEGACY_DASHBOARD_SLOTS.has(widget.slot) && !OPERATIONAL_DASHBOARD_SLOTS.has(widget.slot)) {
    throw new Error(`Dashboard widget "${widget.key}" uses an unknown slot.`);
  }
  assertCleanupPolicy(widget, widget.cleanupPolicy);
  if (widget.sourceDomain.startsWith('business/')) {
    if (!widget.permission) {
      throw new Error(`Business dashboard widget "${widget.key}" must declare a permission.`);
    }
    if (!widget.registrationOwner?.trim()) {
      throw new Error(
        `Business dashboard widget "${widget.key}" must declare a registration owner.`,
      );
    }
  }
  assertOperationalMetadata(widget);
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

export function isDashboardWidgetVisible(
  widget: DashboardWidgetDefinition,
  context: DashboardWidgetVisibilityContext,
) {
  const hasAccess = !widget.permission || context.isAdmin || context.hasPerm(widget.permission);
  if (!hasAccess) {
    return false;
  }
  if (widget.navigationSource === 'direct') {
    return true;
  }
  return Boolean(findMenuNodeByPath(context.menuTree, widget.path));
}

export function getVisibleDashboardWidgets<T extends DashboardWidgetDefinition>(
  widgets: readonly T[],
  context: DashboardWidgetVisibilityContext,
) {
  return widgets.filter((widget) => isDashboardWidgetVisible(widget, context));
}

export function getDashboardWidgetMetadata(widget: DashboardOperationalWidget) {
  return widget.metadata as DashboardWidgetOperationalMetadata;
}

export const dashboardWidgetRegistry = buildDashboardWidgetRegistry(registeredModules);

export const dashboardQuickActionWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardQuickActionWidget => widget.slot === 'quick-action',
);

export const dashboardDomainOverviewWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardDomainOverviewWidget => widget.slot === 'domain-overview',
);

export const dashboardStatusSummaryWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardOperationalWidget => widget.slot === 'status-summary',
);

export const dashboardAttentionQueueWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardOperationalWidget => widget.slot === 'attention-queue',
);

export const dashboardTrendSnapshotWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardOperationalWidget => widget.slot === 'trend-snapshot',
);

export const dashboardRecentActivityWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardOperationalWidget => widget.slot === 'recent-activity',
);
