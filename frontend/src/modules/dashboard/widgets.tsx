import type { MenuNode } from '../system/menu/api';
import { registeredModules } from '../../core/router/modules';
import {
  buildDashboardWidgetRegistry,
  type DashboardDomainOverviewWidget,
  type DashboardQuickActionWidget,
  type DashboardWidgetDefinition,
} from '../../core/workbench/dashboard';

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
