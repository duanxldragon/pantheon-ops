import { OPENED_TABS_STORAGE_KEY } from '../shellState';

export interface OpenedPageTab {
  path: string;
  titleKey?: string;
  fallbackTitle: string;
  closable: boolean;
  pinned?: boolean;
}

export type TabActionKey = 'close' | 'closeOthers' | 'closeRight' | 'closeAll' | 'togglePin';

export function buildOpenedPageTab(
  path: string,
  fallbackTitle: string,
  options: Partial<OpenedPageTab> = {},
): OpenedPageTab {
  return {
    path,
    titleKey: options.titleKey,
    fallbackTitle,
    closable: options.closable ?? path !== '/dashboard',
    pinned: options.pinned ?? path === '/dashboard',
  };
}

export function normalizeOpenedPageTab(item: OpenedPageTab): OpenedPageTab {
  return {
    ...item,
    closable: item.path !== '/dashboard' && item.closable !== false,
    pinned: item.path === '/dashboard' || Boolean(item.pinned),
  };
}

export function normalizeOpenedTabs(tabs: OpenedPageTab[]): OpenedPageTab[] {
  return tabs.map((item) => normalizeOpenedPageTab(item));
}

export function orderOpenedTabs(tabs: OpenedPageTab[]): OpenedPageTab[] {
  const dashboardTabs = tabs.filter((item) => item.path === '/dashboard');
  const pinnedTabs = tabs.filter((item) => item.path !== '/dashboard' && item.pinned);
  const regularTabs = tabs.filter((item) => item.path !== '/dashboard' && !item.pinned);
  return [...dashboardTabs, ...pinnedTabs, ...regularTabs];
}

export function limitOpenedTabs(tabs: OpenedPageTab[]): OpenedPageTab[] {
  const orderedTabs = orderOpenedTabs(tabs);
  const protectedTabs = orderedTabs.filter((item) => item.pinned);
  const regularTabs = orderedTabs.filter((item) => !item.pinned);
  const regularCapacity = 8 - protectedTabs.length;
  if (regularCapacity <= 0) {
    return protectedTabs;
  }
  return [...protectedTabs, ...regularTabs.slice(-regularCapacity)];
}

export function readOpenedTabs(): OpenedPageTab[] {
  try {
    const rawValue = globalThis.localStorage.getItem(OPENED_TABS_STORAGE_KEY);
    if (!rawValue) {
      return [];
    }
    const parsed = JSON.parse(rawValue) as OpenedPageTab[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return orderOpenedTabs(
      normalizeOpenedTabs(
        parsed.filter((item) => typeof item.path === 'string' && item.path.startsWith('/')),
      ),
    );
  } catch {
    return [];
  }
}

export function mergeOpenedTabsIntoState(
  currentTabs: OpenedPageTab[],
  nextTab: OpenedPageTab,
  dashboardTitle: string,
): OpenedPageTab[] {
  const existingIndex = currentTabs.findIndex((item) => item.path === nextTab.path);
  const mergedTabs =
    existingIndex >= 0
      ? currentTabs.map((item, index) =>
          index === existingIndex
            ? { ...item, ...nextTab, pinned: item.pinned || nextTab.pinned }
            : item,
        )
      : [...currentTabs, nextTab];
  const dashboardTab = buildOpenedPageTab('/dashboard', dashboardTitle, {
    titleKey: 'dashboard.title',
    closable: false,
    pinned: true,
  });
  const normalizedTabs = mergedTabs.some((item) => item.path === '/dashboard')
    ? mergedTabs
    : [dashboardTab, ...mergedTabs];
  const limitedTabs = limitOpenedTabs(normalizeOpenedTabs(normalizedTabs));
  globalThis.localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(limitedTabs));
  return limitedTabs;
}
