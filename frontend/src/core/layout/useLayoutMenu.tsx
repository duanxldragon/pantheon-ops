import { useMemo, type ReactNode } from 'react';
import { Menu } from '@arco-design/web-react';
import { type MenuNode, findFirstNavigableMenuPath } from '../../modules/system/menu/api';
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

function findMenuNavigationPath(item: MenuNode): string | undefined {
  if (item.path && findRouteByPath(item.path)) {
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
      const navigationPath = findMenuNavigationPath(item);
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

function filterMenuTreeByCapabilities(nodes: MenuNode[], orgEnabled: boolean): MenuNode[] {
  return nodes
    .filter((item) => orgEnabled || item.module !== 'system.org')
    .map((item) => ({
      ...item,
      children: item.children?.length
        ? filterMenuTreeByCapabilities(item.children, orgEnabled)
        : [],
    }));
}

export interface UseLayoutMenuOptions {
  menuTree: MenuNode[];
  orgEnabled: boolean;
  currentPath: string;
  activeMenuPath: string;
  routeActiveMenu?: string;
  currentRouteTitleKey?: string;
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
    activeMenuPath,
    currentPath,
    routeActiveMenu,
    currentRouteTitleKey,
    t,
    handleMenuNavigation,
  } = options;

  const visibleMenuTree = useMemo(
    () => filterMenuTreeByCapabilities(menuTree, orgEnabled),
    [menuTree, orgEnabled],
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
        t,
      }),
    [handleMenuNavigation, t, visibleMenuTree],
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
