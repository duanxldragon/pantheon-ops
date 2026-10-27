import { useMemo, type ReactNode } from 'react';
import { Menu } from '@arco-design/web-react';
import { type MenuNode, findFirstNavigableMenuPath } from '../../modules/system/iam/menu/api';
import { renderMenuIcon } from '../menu/icon';
import { findRouteByPath } from '../router/modules';
import { preloadRouteComponent } from '../router/prefetch';

export interface LayoutBreadcrumbItem {
  path: string;
  label: string;
}

type TranslateLabel = (key: string, options?: Record<string, unknown>) => string;
type NavigateToPath = (path: string) => void;

interface MenuRenderOptions {
  handleMenuNavigation: NavigateToPath;
  isPathEntitled: (path: string) => boolean;
  t: TranslateLabel;
}

function findMenuTitleKey(nodes: MenuNode[], path: string): string | undefined {
  for (const item of nodes) {
    if (item.path === path || item.activeMenu === path) {
      return item.titleKey;
    }
    if (item.children?.length) {
      const childTitleKey = findMenuTitleKey(item.children, path);
      if (childTitleKey) {
        return childTitleKey;
      }
    }
  }
  return undefined;
}

function findSelectedMenuPath(nodes: MenuNode[], path: string): string {
  for (const item of nodes) {
    if (item.path === path || item.activeMenu === path) {
      return item.path;
    }
    if (item.children?.length) {
      const childPath = findSelectedMenuPath(item.children, path);
      if (childPath) {
        return childPath;
      }
    }
  }
  return path;
}

function findMenuTrail(nodes: MenuNode[], path: string, ancestors: MenuNode[] = []): MenuNode[] {
  for (const item of nodes) {
    const trail = [...ancestors, item];
    if (item.path === path || item.activeMenu === path) {
      return trail;
    }
    if (item.children?.length) {
      const childTrail = findMenuTrail(item.children, path, trail);
      if (childTrail.length > 0) {
        return childTrail;
      }
    }
  }
  return [];
}

export function findMenuNodeByPath(nodes: MenuNode[], path: string): MenuNode | undefined {
  for (const item of nodes) {
    if (item.path === path) {
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

function findMenuNavigationPath(
  item: MenuNode,
  isPathEntitled: (path: string) => boolean,
): string | undefined {
  if (item.path && findRouteByPath(item.path) && isPathEntitled(item.path)) {
    return item.path;
  }
  return item.children?.length ? findFirstNavigableMenuPath(item.children) || undefined : undefined;
}

function preloadRouteByPath(path: string) {
  preloadRouteComponent(path).catch(() => undefined);
}

function renderMenuItems(nodes: MenuNode[], options: MenuRenderOptions, level = 0): ReactNode[] {
  return nodes.map((item) => {
    const entryClassName = [
      'app-shell__menu-entry',
      `app-shell__menu-entry--level-${Math.min(level, 2)}`,
      item.children && item.children.length > 0
        ? 'app-shell__menu-entry--group'
        : 'app-shell__menu-entry--leaf',
    ].join(' ');
    const iconClassName = [
      'app-shell__menu-entry-icon',
      `app-shell__menu-entry-icon--level-${Math.min(level, 2)}`,
    ].join(' ');

    if (item.children && item.children.length > 0) {
      const navigationPath = findMenuNavigationPath(item, options.isPathEntitled);
      const handleGroupNavigation = () => {
        if (navigationPath) {
          options.handleMenuNavigation(navigationPath);
        }
      };

      return (
        <Menu.SubMenu
          key={item.id.toString()}
          title={
            <button type="button" className={entryClassName} onClick={handleGroupNavigation}>
              <span className={iconClassName}>{renderMenuIcon(item.icon)}</span>
              <span className="app-shell__menu-entry-copy">
                <span className="app-shell__menu-entry-label">{options.t(item.titleKey)}</span>
              </span>
            </button>
          }
        >
          {renderMenuItems(item.children, options, level + 1)}
        </Menu.SubMenu>
      );
    }

    const handleLeafNavigation = () => {
      options.handleMenuNavigation(item.path);
    };
    const handleLeafPreload = () => {
      preloadRouteByPath(item.path);
    };

    return (
      <Menu.Item key={item.path}>
        <button
          type="button"
          className={entryClassName}
          onClick={handleLeafNavigation}
          onMouseEnter={handleLeafPreload}
          onFocus={handleLeafPreload}
        >
          <span className={iconClassName}>{renderMenuIcon(item.icon)}</span>
          <span className="app-shell__menu-entry-copy">
            <span className="app-shell__menu-entry-label">{options.t(item.titleKey)}</span>
          </span>
        </button>
      </Menu.Item>
    );
  });
}

interface CapabilityFilter {
  orgEnabled: boolean;
  tenantEnabled: boolean;
  /** True when the session is pinned to a tenant (multi mode, tenant id > 0). */
  tenantScoped: boolean;
}

// Mirrors backend pkg/tenant/platform_scope.go: these surfaces are gated by
// PlatformCapabilityMiddleware, so a tenant session that opens them gets
// 403 tenant.platformOnly. The backend already filters them out of the tree;
// repeating the rule here keeps a stale/gateway-cached tree from rendering a
// dead menu entry.
const PLATFORM_ONLY_MENU_MODULES = new Set(['system.tenant']);
const PLATFORM_ONLY_MENU_PATHS = new Set([
  '/system/tenant',
  '/system/tenant-ops/overview',
  '/system/tenant-ops/member',
  '/system/menu',
  '/system/i18n',
]);

function isPlatformOnlyMenu(item: MenuNode): boolean {
  return (
    PLATFORM_ONLY_MENU_MODULES.has(item.module) || PLATFORM_ONLY_MENU_PATHS.has(item.path)
  );
}

// isHiddenByCapabilities is the single place the capability rules apply:
// module disablement (org / tenant) and the platform-only surface a tenant
// session must never see.
function isHiddenByCapabilities(item: MenuNode, caps: CapabilityFilter): boolean {
  if (!caps.orgEnabled && item.module === 'system.org') {
    return true;
  }
  // 单租户（compat）下没有租户维度可管，租户管理菜单从访问控制移除。
  // 多租户（multi）下保留。与 org.enabled / system.org 同一套能力过滤机制。
  if (!caps.tenantEnabled && item.module === 'system.tenant') {
    return true;
  }
  return caps.tenantScoped && isPlatformOnlyMenu(item);
}

// isNavigableGroup keeps a group that still has a visible child, or that
// navigates somewhere the plan includes. Resolving the navigation target
// against the filtered children stops a hidden child from keeping its parent
// group alive as an empty expandable row.
function isNavigableGroup(
  item: MenuNode,
  children: MenuNode[],
  entitled: boolean,
  isPathEntitled: (path: string) => boolean,
): boolean {
  if (children.length > 0) {
    return true;
  }
  return entitled && Boolean(findMenuNavigationPath({ ...item, children }, isPathEntitled));
}

function filterMenuTreeByCapabilities(
  nodes: MenuNode[],
  caps: CapabilityFilter,
  isPathEntitled: (path: string) => boolean,
): MenuNode[] {
  const filtered: MenuNode[] = [];
  for (const item of nodes) {
    if (isHiddenByCapabilities(item, caps)) {
      continue;
    }
    const children = item.children?.length
      ? filterMenuTreeByCapabilities(item.children, caps, isPathEntitled)
      : [];
    const entitled = !item.path || isPathEntitled(item.path);
    const keep = item.children?.length
      ? isNavigableGroup(item, children, entitled, isPathEntitled)
      : entitled;
    if (!keep) {
      continue;
    }
    filtered.push({ ...item, children });
  }
  return filtered;
}

export interface UseLayoutMenuOptions {
  menuTree: MenuNode[];
  orgEnabled: boolean;
  tenantEnabled: boolean;
  tenantScoped: boolean;
  currentPath: string;
  activeMenuPath: string;
  routeActiveMenu?: string;
  currentRouteTitleKey?: string;
  /** Plan/feature filter resolved from the module route registry. */
  isPathEntitled: (path: string) => boolean;
  t: TranslateLabel;
  handleMenuNavigation: NavigateToPath;
}

export interface UseLayoutMenuResult {
  visibleMenuTree: MenuNode[];
  currentMenuTitleKey?: string;
  menuTrail: MenuNode[];
  selectedMenuPath: string;
  breadcrumbItems: LayoutBreadcrumbItem[];
  menuOpenKeys: string[];
  renderedMenuItems: ReactNode[];
}

export function useLayoutMenu(options: UseLayoutMenuOptions): UseLayoutMenuResult {
  const {
    menuTree,
    orgEnabled,
    tenantEnabled,
    tenantScoped,
    activeMenuPath,
    currentPath,
    routeActiveMenu,
    currentRouteTitleKey,
    isPathEntitled,
    t,
    handleMenuNavigation,
  } = options;

  const visibleMenuTree = useMemo(
    () =>
      filterMenuTreeByCapabilities(
        menuTree,
        { orgEnabled, tenantEnabled, tenantScoped },
        isPathEntitled,
      ),
    [menuTree, orgEnabled, tenantEnabled, tenantScoped, isPathEntitled],
  );

  const currentMenuTitleKey = useMemo(
    () => findMenuTitleKey(visibleMenuTree, activeMenuPath),
    [activeMenuPath, visibleMenuTree],
  );

  const menuTrail = useMemo(
    () => findMenuTrail(visibleMenuTree, activeMenuPath),
    [activeMenuPath, visibleMenuTree],
  );

  const selectedMenuPath = useMemo(
    () => findSelectedMenuPath(visibleMenuTree, activeMenuPath),
    [activeMenuPath, visibleMenuTree],
  );

  const breadcrumbItems = useMemo(() => {
    const root = [{ path: '/', label: t('common.home') }];
    if (menuTrail.length > 0) {
      const trailItems = [
        ...root,
        ...menuTrail.map((item) => ({
          path: item.path,
          label: t(item.titleKey),
        })),
      ];
      if (routeActiveMenu && currentRouteTitleKey && currentRouteTitleKey !== currentMenuTitleKey) {
        return [
          ...trailItems,
          {
            path: currentPath,
            label: t(currentRouteTitleKey),
          },
        ];
      }
      return trailItems;
    }

    let currentLabel = currentPath;
    if (currentMenuTitleKey) {
      currentLabel = t(currentMenuTitleKey);
    } else if (currentRouteTitleKey) {
      currentLabel = t(currentRouteTitleKey);
    }

    return [
      ...root,
      {
        path: activeMenuPath,
        label: currentLabel,
      },
    ];
  }, [
    currentMenuTitleKey,
    currentPath,
    currentRouteTitleKey,
    activeMenuPath,
    menuTrail,
    routeActiveMenu,
    t,
  ]);

  const menuOpenKeys = useMemo(
    () => menuTrail.slice(0, -1).map((item) => item.id.toString()),
    [menuTrail],
  );

  const renderedMenuItems = useMemo(
    () =>
      renderMenuItems(visibleMenuTree, {
        handleMenuNavigation,
        isPathEntitled,
        t,
      }),
    [handleMenuNavigation, isPathEntitled, t, visibleMenuTree],
  );

  return {
    visibleMenuTree,
    currentMenuTitleKey,
    menuTrail,
    selectedMenuPath,
    breadcrumbItems,
    menuOpenKeys,
    renderedMenuItems,
  };
}
