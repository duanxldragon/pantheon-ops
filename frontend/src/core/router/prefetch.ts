import { findRouteByPath } from './modules';
import { preloadRegisteredComponent, type RegisteredComponentKey } from './componentRegistry';
import { getDashboardSummary } from '../../modules/dashboard/api';
import { getSecurityOverview, getSessions, getOwnLoginLogs } from '../../modules/auth/api';
import { getUserList } from '../../modules/system/user/api';
import { getRoleList } from '../../modules/system/role/api';
import { getDeptOverview, getDeptTree } from '../../modules/system/dept/api';
import { getPostList } from '../../modules/system/post/api';
import { getMenuTree } from '../../modules/system/menu/api';
import {
  getPermissionPolicyList,
  getPermissionWorkbench,
} from '../../modules/system/permission/api';
import { getDictTypeList } from '../../modules/system/dict/api';
import { getSettingList, getSettingOverview } from '../../modules/system/setting/api';
import type { MenuNode } from '../../modules/system/menu/api';
import type { UserInfo } from '../../store/useAuthStore';

const warmedRoutes = new Set<string>();
const warmedComponents = new Set<RegisteredComponentKey>();
const warmDataCache = new Map<
  string,
  { expiresAt: number; promise: Promise<unknown>; value?: unknown }
>();
const DEFAULT_WARM_TTL_MS = 30_000;

const HIGH_FREQUENCY_ROUTE_PATHS = [
  '/dashboard',
  '/auth/security',
  '/system/user',
  '/system/role',
  '/system/menu',
  '/system/permission',
  '/system/dept',
  '/system/post',
  '/system/dict',
  '/system/setting',
] as const;

const routeDataWarmers: Partial<
  Record<
    (typeof HIGH_FREQUENCY_ROUTE_PATHS)[number],
    Array<{ key: string; load: () => Promise<unknown> }>
  >
> = {
  '/dashboard': [{ key: 'summary', load: () => getDashboardSummary() }],
  '/auth/security': [
    { key: 'overview', load: () => getSecurityOverview() },
    { key: 'sessions', load: () => getSessions() },
    { key: 'login-logs', load: () => getOwnLoginLogs({ page: 1, pageSize: 10 }) },
  ],
  '/system/user': [
    {
      key: 'list:default',
      load: () => getUserList({ username: '', nickname: '', page: 1, pageSize: 10 }),
    },
    {
      key: 'roles:active',
      load: () =>
        getRoleList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc', status: 1 }),
    },
    { key: 'depts:default', load: () => getDeptTree({ sortField: 'sort', sortOrder: 'asc' }) },
    {
      key: 'posts:active',
      load: () =>
        getPostList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc', status: 1 }),
    },
  ],
  '/system/role': [
    {
      key: 'list:default',
      load: () => getRoleList({ roleName: '', roleKey: '', page: 1, pageSize: 10 }),
    },
    { key: 'menus:manage', load: () => getMenuTree({ scope: 'manage' }) },
  ],
  '/system/menu': [{ key: 'tree:manage', load: () => getMenuTree({ scope: 'manage' }) }],
  '/system/permission': [
    { key: 'workbench:default', load: () => getPermissionWorkbench({}) },
    { key: 'list:default', load: () => getPermissionPolicyList({ page: 1, pageSize: 10 }) },
    {
      key: 'roles:default',
      load: () => getRoleList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc' }),
    },
  ],
  '/system/dept': [
    { key: 'tree:default', load: () => getDeptTree({}) },
    { key: 'overview', load: () => getDeptOverview() },
    { key: 'tree:sorted', load: () => getDeptTree({ sortField: 'sort', sortOrder: 'asc' }) },
    {
      key: 'posts:org-chart',
      load: () => getPostList({ page: 1, pageSize: 1000, sortField: 'sort', sortOrder: 'asc' }),
    },
    {
      key: 'users:org-chart',
      load: () => getUserList({ page: 1, pageSize: 1000, sortField: 'username', sortOrder: 'asc' }),
    },
  ],
  '/system/post': [
    { key: 'list:default', load: () => getPostList({ page: 1, pageSize: 10 }) },
    { key: 'depts:sorted', load: () => getDeptTree({ sortField: 'sort', sortOrder: 'asc' }) },
  ],
  '/system/dict': [{ key: 'types:default', load: () => getDictTypeList({}) }],
  '/system/setting': [
    { key: 'list:default', load: () => getSettingList() },
    { key: 'overview', load: () => getSettingOverview() },
  ],
};

interface RouteWarmupContext {
  menuTree?: MenuNode[];
  userInfo?: UserInfo | null;
}

interface RouteWarmupScheduleOptions {
  excludePaths?: string[];
}

function normalizePath(path: string) {
  return path.startsWith('/') ? path : `/${path}`;
}

function collectNavigablePaths(nodes: MenuNode[], result: Set<string>) {
  nodes.forEach((item) => {
    if (item.path && item.type === 'C' && item.isExternal !== 1) {
      result.add(normalizePath(item.path));
    }
    if (item.children?.length) {
      collectNavigablePaths(item.children, result);
    }
  });
}

function canWarmRoute(path: string, context?: RouteWarmupContext) {
  if (!context) {
    return true;
  }

  const normalizedPath = normalizePath(path);
  if (normalizedPath === '/auth/security') {
    return true;
  }

  const route = findRouteByPath(normalizedPath);
  const userInfo = context.userInfo;
  const isAdmin = Boolean(userInfo?.roles?.includes('admin'));
  if (route?.pagePermission && !isAdmin && !userInfo?.perms?.includes(route.pagePermission)) {
    return false;
  }

  const menuTree = context.menuTree;
  if (!menuTree?.length) {
    return normalizedPath === '/dashboard'
      ? isAdmin || Boolean(userInfo?.perms?.includes('platform:dashboard:view'))
      : true;
  }

  const allowedPaths = new Set<string>();
  collectNavigablePaths(menuTree, allowedPaths);

  if (normalizedPath === '/dashboard') {
    return (
      allowedPaths.has('/dashboard') ||
      isAdmin ||
      Boolean(userInfo?.perms?.includes('platform:dashboard:view'))
    );
  }

  return allowedPaths.has(normalizedPath);
}

function buildWarmDataCacheKey(path: string, resourceKey: string) {
  return `${path}::${resourceKey}`;
}

function getFreshWarmDataEntry(cacheKey: string) {
  const entry = warmDataCache.get(cacheKey);
  if (!entry) {
    return undefined;
  }
  if (entry.expiresAt <= Date.now()) {
    warmDataCache.delete(cacheKey);
    return undefined;
  }
  return entry;
}

function cacheWarmData<T>(cacheKey: string, loader: () => Promise<T>, ttlMs = DEFAULT_WARM_TTL_MS) {
  const cached = getFreshWarmDataEntry(cacheKey);
  if (cached) {
    return cached.promise as Promise<T>;
  }
  const promise = loader()
    .then((value) => {
      const entry = warmDataCache.get(cacheKey);
      if (entry) {
        entry.value = value;
      }
      return value;
    })
    .catch((error) => {
      warmDataCache.delete(cacheKey);
      throw error;
    });
  warmDataCache.set(cacheKey, {
    expiresAt: Date.now() + ttlMs,
    promise,
  });
  return promise;
}

function preloadRouteData(path: string, context?: RouteWarmupContext) {
  if (!canWarmRoute(path, context)) {
    return Promise.resolve(undefined);
  }
  const warmers = routeDataWarmers[path as keyof typeof routeDataWarmers];
  if (!warmers?.length) {
    return Promise.resolve(undefined);
  }
  return Promise.allSettled(
    warmers.map((item) => cacheWarmData(buildWarmDataCacheKey(path, item.key), item.load)),
  ).then(() => undefined);
}

export function resolveRouteWarmData<T>(
  path: string,
  resourceKey: string,
  loader: () => Promise<T>,
  ttlMs = DEFAULT_WARM_TTL_MS,
) {
  return cacheWarmData(buildWarmDataCacheKey(normalizePath(path), resourceKey), loader, ttlMs);
}

export function invalidateRouteWarmData(path: string, resourceKeys?: string[]) {
  const normalizedPath = normalizePath(path);
  if (!resourceKeys?.length) {
    Array.from(warmDataCache.keys()).forEach((cacheKey) => {
      if (cacheKey.startsWith(`${normalizedPath}::`)) {
        warmDataCache.delete(cacheKey);
      }
    });
    return;
  }

  resourceKeys.forEach((resourceKey) => {
    warmDataCache.delete(buildWarmDataCacheKey(normalizedPath, resourceKey));
  });
}

export function invalidateRouteWarmDataMany(
  targets: Array<{ path: string; resourceKeys?: string[] }>,
) {
  targets.forEach((target) => invalidateRouteWarmData(target.path, target.resourceKeys));
}

export function preloadRouteComponent(path?: string, context?: RouteWarmupContext) {
  if (!path) {
    return Promise.resolve(undefined);
  }
  const normalizedPath = normalizePath(path);
  if (!canWarmRoute(normalizedPath, context)) {
    return Promise.resolve(undefined);
  }
  const route = findRouteByPath(normalizedPath);
  const componentKey = route?.componentKey;
  const componentTask = (() => {
    if (!componentKey) {
      warmedRoutes.add(normalizedPath);
      return Promise.resolve(undefined);
    }
    if (warmedRoutes.has(normalizedPath) || warmedComponents.has(componentKey)) {
      return Promise.resolve(undefined);
    }
    warmedRoutes.add(normalizedPath);
    warmedComponents.add(componentKey);
    return preloadRegisteredComponent(componentKey).catch(() => {
      warmedRoutes.delete(normalizedPath);
      warmedComponents.delete(componentKey);
      return undefined;
    });
  })();
  const dataTask = preloadRouteData(normalizedPath, context);
  return Promise.allSettled([componentTask, dataTask]).then(() => undefined);
}

export function scheduleHighFrequencyRouteWarmup(
  context?: RouteWarmupContext,
  options?: RouteWarmupScheduleOptions,
) {
  const excludedPaths = new Set((options?.excludePaths || []).map((path) => normalizePath(path)));
  const runWarmup = () => {
    HIGH_FREQUENCY_ROUTE_PATHS.forEach((path) => {
      if (excludedPaths.has(normalizePath(path))) {
        return;
      }
      void preloadRouteComponent(path, context);
    });
  };

  if (typeof window !== 'undefined' && 'requestIdleCallback' in window) {
    const callback = (
      window as Window & {
        requestIdleCallback: (cb: () => void, options?: { timeout: number }) => number;
      }
    ).requestIdleCallback;
    callback(runWarmup, { timeout: 1200 });
    return;
  }

  globalThis.setTimeout(runWarmup, 300);
}
