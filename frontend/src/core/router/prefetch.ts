import { findRouteByPath, registeredModules } from './modules';
import { preloadRegisteredComponent, type RegisteredComponentKey } from './componentRegistry';
import type { RouteDataWarmer } from './types';
import type { UserInfo } from '../../store/authTypes';
import { shouldWarmHighFrequencyRouteData } from './warmupPolicy';

const warmedRoutes = new Set<string>();
const warmedComponents = new Set<RegisteredComponentKey>();
const warmDataCache = new Map<
  string,
  { expiresAt: number; promise: Promise<unknown>; value?: unknown }
>();
const DEFAULT_WARM_TTL_MS = 30_000;

interface WarmupMenuNode {
  path?: string;
  type?: string;
  isExternal?: number;
  children?: WarmupMenuNode[];
}

interface RouteWarmupContext {
  menuTree?: WarmupMenuNode[];
  userInfo?: UserInfo | null;
}

interface RouteWarmupScheduleOptions {
  excludePaths?: string[];
}

function normalizePath(path: string) {
  return path.startsWith('/') ? path : `/${path}`;
}

function collectNavigablePaths(nodes: WarmupMenuNode[], result: Set<string>) {
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

function buildRouteDataWarmers() {
  const warmersByPath = new Map<string, RouteDataWarmer[]>();
  registeredModules.forEach((module) => {
    module.routeDataWarmers?.forEach((warmer) => {
      const path = normalizePath(warmer.path);
      const pathWarmers = warmersByPath.get(path) ?? [];
      pathWarmers.push({ ...warmer, path });
      warmersByPath.set(path, pathWarmers);
    });
  });
  return warmersByPath;
}

const routeDataWarmers = buildRouteDataWarmers();

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
  if (!shouldWarmHighFrequencyRouteData()) {
    return Promise.resolve(undefined);
  }
  if (!canWarmRoute(path, context)) {
    return Promise.resolve(undefined);
  }
  const warmers = routeDataWarmers.get(path);
  if (!warmers?.length) {
    return Promise.resolve(undefined);
  }
  return Promise.allSettled(
    warmers.map((item) =>
      cacheWarmData(buildWarmDataCacheKey(path, item.key), item.load, item.ttlMs),
    ),
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
    routeDataWarmers.forEach((_warmers, path) => {
      if (excludedPaths.has(normalizePath(path))) {
        return;
      }
      void preloadRouteComponent(path, context);
    });
  };

  if (globalThis.document !== undefined && 'requestIdleCallback' in globalThis) {
    const callback = (
      globalThis as typeof globalThis & {
        requestIdleCallback: (cb: () => void, options?: { timeout: number }) => number;
      }
    ).requestIdleCallback;
    callback(runWarmup, { timeout: 1200 });
    return;
  }

  globalThis.setTimeout(runWarmup, 300);
}
