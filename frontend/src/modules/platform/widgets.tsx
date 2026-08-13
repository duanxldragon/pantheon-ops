import type { MenuNode } from '../system/menu/api';
import { registeredModules } from '../../core/router/modules';
import type {
  DashboardDomainOverviewWidget,
  DashboardQuickActionWidget,
  DashboardWidgetDefinition,
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

function buildDashboardWidgetRegistry(
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

export const dashboardWidgetRegistry = buildDashboardWidgetRegistry(registeredModules);

export const dashboardQuickActionWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardQuickActionWidget => widget.slot === 'quick-action',
);

export const dashboardDomainOverviewWidgets = dashboardWidgetRegistry.filter(
  (widget): widget is DashboardDomainOverviewWidget => widget.slot === 'domain-overview',
);
