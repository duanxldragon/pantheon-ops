import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Avatar,
  Breadcrumb,
  Button,
  Dropdown,
  Empty,
  Input,
  Layout,
  Menu,
  Message,
  Space,
  Spin,
  Tooltip,
  Typography,
} from '@arco-design/web-react';
import {
  IconCheck,
  IconClose,
  IconLanguage,
  IconLayout,
  IconLock,
  IconMenuFold,
  IconMenuUnfold,
  IconNotification,
  IconPoweroff,
  IconPushpin,
  IconSafe,
  IconSearch,
  IconSettings,
  IconUser,
} from '@arco-design/web-react/icon';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { beginLogoutTransition, endLogoutTransition } from '../../api/request';
import { findFirstNavigableMenuPath, type MenuNode } from '../../modules/system/menu/api';
import {
  logout as logoutApi,
  reportActivity,
  updateCurrentUserPreferences,
  verifyOperationPassword,
  type UserPlatformPreferences,
} from '../../modules/auth/api';
import { ensureAuthUserInfo } from '../auth/bootstrap';
import { useMenuStore } from '../../store/useMenuStore';
import { useAuthStore } from '../../store/useAuthStore';
import { usePermission } from '../../hooks/usePermission';
import { useRefreshPolling, useRefreshSubscription } from '../refresh/refreshBus';
import { findRouteByPath, systemRouteTitleMap } from '../router/modules';
import { formatDateTime } from '../format/dateTime';
import { renderMenuIcon } from '../menu/icon';
import { clearPantheonThemePreference, usePantheonTheme } from '../theme/theme';
import { AppModal } from '../../components';
import { getDashboardSummary, type DashboardSummary } from '../../modules/dashboard/api';
import {
  getBrandInitial,
  hasExplicitLanguagePreference,
  refreshPublicSettings,
  setExplicitLanguagePreference,
  usePublicSettings,
} from '../settings/publicSettings';
import { clearExplicitLanguagePreference } from '../settings/languagePreference';
import { SUPPORTED_LOCALES, switchI18nLanguage, type SupportedLocale } from '../../i18n';
import { preloadRouteComponent } from '../router/prefetch';
import {
  OPENED_TABS_STORAGE_KEY,
  persistShellDensityMode,
  persistShellLastActivityAt,
  persistShellLockedState,
  persistLoginNotice,
  readShellDensityMode,
  readShellLastActivityAt,
  readShellLockedState,
  clearShellSessionState,
  persistShellLayoutMode,
  readShellLayoutMode,
  type ShellDensityMode,
  type ShellLayoutMode,
} from '../shellState';
import './index.css';

const { Header, Footer, Sider, Content } = Layout;
const MAX_OPENED_TABS = 8;
const MENU_COLLAPSE_PREFERRED_ROUTE_NAMES = new Set(['operations']);

interface OpenedPageTab {
  path: string;
  titleKey?: string;
  fallbackTitle: string;
  closable: boolean;
  pinned?: boolean;
}

type TabActionKey = 'close' | 'closeOthers' | 'closeRight' | 'closeAll' | 'togglePin';
type UserMenuActionKey = 'profile' | 'security' | 'lock' | 'logout';

interface CommandSearchItem {
  key: string;
  title: string;
  subtitle: string;
  section: string;
  searchText: string;
  icon?: React.ReactNode;
  run: () => void;
}

interface NoticeEntry {
  key: string;
  title: string;
  description: string;
  icon: React.ReactNode;
  run: () => void;
}

interface NoticeRiskItem {
  key: string;
  title: string;
  description: string;
  value: number;
  tone: 'danger' | 'warning' | 'neutral';
  run: () => void;
}

function orderOpenedTabs(tabs: OpenedPageTab[]): OpenedPageTab[] {
  const dashboardTabs = tabs.filter((item) => item.path === '/dashboard');
  const pinnedTabs = tabs.filter((item) => item.path !== '/dashboard' && item.pinned);
  const regularTabs = tabs.filter((item) => item.path !== '/dashboard' && !item.pinned);
  return [...dashboardTabs, ...pinnedTabs, ...regularTabs];
}

function limitOpenedTabs(tabs: OpenedPageTab[]): OpenedPageTab[] {
  const orderedTabs = orderOpenedTabs(tabs);
  const protectedTabs = orderedTabs.filter((item) => item.pinned);
  const regularTabs = orderedTabs.filter((item) => !item.pinned);
  const regularCapacity = MAX_OPENED_TABS - protectedTabs.length;
  if (regularCapacity <= 0) {
    return protectedTabs;
  }
  return [...protectedTabs, ...regularTabs.slice(-regularCapacity)];
}

function readOpenedTabs(): OpenedPageTab[] {
  try {
    const rawValue = localStorage.getItem(OPENED_TABS_STORAGE_KEY);
    if (!rawValue) {
      return [];
    }
    const parsed = JSON.parse(rawValue) as OpenedPageTab[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return orderOpenedTabs(
      parsed
        .filter((item) => typeof item.path === 'string' && item.path.startsWith('/'))
        .map((item) => ({
          ...item,
          closable: item.path !== '/dashboard' && item.closable !== false,
          pinned: item.path === '/dashboard' || Boolean(item.pinned),
        })),
    );
  } catch {
    return [];
  }
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

function findMenuNodeByPath(nodes: MenuNode[], path: string): MenuNode | undefined {
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

const BaseLayout: React.FC = () => {
  const bootstrappedRef = useRef(false);
  const [collapsed, setCollapsed] = useState(false);
  const [layoutMode, setLayoutMode] = useState<ShellLayoutMode>(() => readShellLayoutMode());
  const [densityMode, setDensityMode] = useState<ShellDensityMode>(() => readShellDensityMode());
  const [openedTabs, setOpenedTabs] = useState<OpenedPageTab[]>(() => readOpenedTabs());
  const [draggingTabPath, setDraggingTabPath] = useState<string | null>(null);
  const [dragOverTabPath, setDragOverTabPath] = useState<string | null>(null);
  const [commandVisible, setCommandVisible] = useState(false);
  const [commandQuery, setCommandQuery] = useState('');
  const [noticeSummary, setNoticeSummary] = useState<DashboardSummary | null>(null);
  const [noticeLoading, setNoticeLoading] = useState(false);
  const [locked, setLocked] = useState(() => readShellLockedState());
  const [unlockPassword, setUnlockPassword] = useState('');
  const [unlockLoading, setUnlockLoading] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { t, i18n } = useTranslation();
  const publicSettings = usePublicSettings();
  const { menuTree, fetchMenuTree, resetMenuTree, loading } = useMenuStore();
  const { token, userInfo, clearAuth, setUserInfo } = useAuthStore();
  const { isAdmin, hasPerm } = usePermission();
  const lastActivityAtRef = useRef(readShellLastActivityAt() || 0);
  const lastSyncedActivityAtRef = useRef(0);
  const lastInteractionAtRef = useRef(0);
  const idleLogoutInFlightRef = useRef(false);
  const matchedRoute = useMemo(() => findRouteByPath(location.pathname), [location.pathname]);
  const currentRouteTitleKey =
    matchedRoute?.resolveTitleKey?.(location.pathname) ||
    matchedRoute?.titleKey ||
    systemRouteTitleMap[location.pathname];
  const activeMenuPath = matchedRoute?.activeMenu || location.pathname;
  const visibleMenuTree = useMemo(
    () => filterMenuTreeByCapabilities(menuTree, publicSettings.orgEnabled),
    [menuTree, publicSettings.orgEnabled],
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
      if (
        matchedRoute?.activeMenu &&
        currentRouteTitleKey &&
        currentRouteTitleKey !== currentMenuTitleKey
      ) {
        return [
          ...trailItems,
          {
            path: location.pathname,
            label: t(currentRouteTitleKey),
          },
        ];
      }
      return trailItems;
    }
    return [
      ...root,
      {
        path: activeMenuPath,
        label: currentMenuTitleKey
          ? t(currentMenuTitleKey)
          : currentRouteTitleKey
            ? t(currentRouteTitleKey)
            : location.pathname,
      },
    ];
  }, [
    activeMenuPath,
    currentMenuTitleKey,
    currentRouteTitleKey,
    location.pathname,
    matchedRoute?.activeMenu,
    menuTrail,
    t,
  ]);
  const menuOpenKeys = useMemo(
    () =>
      menuTrail
        .slice(0, -1)
        .filter((item) => !MENU_COLLAPSE_PREFERRED_ROUTE_NAMES.has(item.routeName || ''))
        .map((item) => item.id.toString()),
    [menuTrail],
  );
  const currentPageTitle = breadcrumbItems[breadcrumbItems.length - 1]?.label || t('app.workspace');
  const currentTabTitleKey = currentRouteTitleKey || currentMenuTitleKey;
  const userDisplayName = userInfo?.nickname || userInfo?.username || t('common.user');
  const roleLabel = userInfo?.roles?.[0] || '';
  const isHorizontalLayout = layoutMode === 'horizontal';
  const layoutModeLabel = t(
    isHorizontalLayout ? 'app.layoutMode.horizontal' : 'app.layoutMode.vertical',
  );
  const layoutModeActionLabel = t(
    isHorizontalLayout ? 'app.layoutMode.switchToVertical' : 'app.layoutMode.switchToHorizontal',
  );
  const densityModeLabel = t(
    densityMode === 'compact' ? 'app.density.compact' : 'app.density.comfortable',
  );
  const appName = publicSettings.siteName || t('app.name');
  const brandInitial = getBrandInitial(appName);
  const { theme, setTheme, options: themeOptions } = usePantheonTheme();
  const activeTheme = themeOptions.find((item) => item.key === theme) ?? themeOptions[0];
  const sessionIdleMinutes =
    publicSettings.sessionIdleMinutes > 0 ? publicSettings.sessionIdleMinutes : 30;
  const sessionIdleMs = sessionIdleMinutes * 60 * 1000;
  const hasDashboardEntry = useMemo(
    () => Boolean(findMenuNodeByPath(visibleMenuTree, '/dashboard')),
    [visibleMenuTree],
  );
  const canAccessDashboard = isAdmin || hasDashboardEntry || hasPerm('platform:dashboard:view');
  const canViewNoticeSummary =
    isAdmin ||
    hasDashboardEntry ||
    hasPerm('platform:dashboard:view') ||
    hasPerm('system:login-log:list') ||
    hasPerm('system:session:list') ||
    hasPerm('system:security-event:list') ||
    hasPerm('system:operation-log:list');

  const syncShellActivity = useCallback((value: number) => {
    lastActivityAtRef.current = value;
    persistShellLastActivityAt(value);
  }, []);

  const performLogout = useCallback(
    async (revokeSession: boolean, noticeKey?: string) => {
      if (idleLogoutInFlightRef.current) {
        return;
      }
      idleLogoutInFlightRef.current = true;
      beginLogoutTransition();
      if (noticeKey) {
        persistLoginNotice(noticeKey);
      }
      if (revokeSession) {
        await logoutApi().catch(() => undefined);
      }
      await refreshPublicSettings().catch(() => undefined);
      clearPantheonThemePreference();
      const nextLanguage = clearExplicitLanguagePreference();
      await switchI18nLanguage(nextLanguage).catch(() => undefined);
      clearShellSessionState();
      setOpenedTabs([]);
      resetMenuTree();
      clearAuth();
      setLocked(false);
      setUnlockPassword('');
      navigate('/login', { replace: true });
      globalThis.setTimeout(() => {
        endLogoutTransition();
        idleLogoutInFlightRef.current = false;
      }, 800);
    },
    [clearAuth, navigate, resetMenuTree],
  );

  const recordActivity = useCallback(
    (reason: 'interaction' | 'route' | 'unlock' | 'bootstrap', syncRemote = true) => {
      if (!token || (locked && reason !== 'unlock')) {
        return;
      }
      const now = Date.now();
      if (reason === 'interaction' && now - lastInteractionAtRef.current < 3000) {
        return;
      }
      if (reason === 'interaction') {
        lastInteractionAtRef.current = now;
      }
      syncShellActivity(now);
      if (!syncRemote || now - lastSyncedActivityAtRef.current < 60000) {
        return;
      }
      lastSyncedActivityAtRef.current = now;
      void reportActivity().catch(() => undefined);
    },
    [locked, syncShellActivity, token],
  );

  useEffect(() => {
    if (bootstrappedRef.current) {
      return;
    }
    bootstrappedRef.current = true;

    void fetchMenuTree();
    if (!useAuthStore.getState().userInfo) {
      void ensureAuthUserInfo().catch(() => undefined);
    }
    const initialActivityAt = readShellLastActivityAt();
    syncShellActivity(initialActivityAt && initialActivityAt > 0 ? initialActivityAt : Date.now());
  }, [fetchMenuTree, syncShellActivity]);

  useRefreshSubscription(
    [
      'system:menu:changed',
      'system:role:changed',
      'system:permission:changed',
      'system:user:changed',
      'system:setting:changed',
    ],
    () => {
      if (!token) {
        return;
      }
      if (!hasExplicitLanguagePreference()) {
        void refreshPublicSettings()
          .then((settings) => switchI18nLanguage(settings.defaultLanguage))
          .catch(() => undefined);
      } else {
        void refreshPublicSettings().catch(() => undefined);
      }
      void fetchMenuTree({ force: true });
    },
  );
  useRefreshPolling(token, [
    'system:user:changed',
    'system:role:changed',
    'system:menu:changed',
    'system:dept:changed',
    'system:post:changed',
    'system:permission:changed',
    'system:dict:changed',
    'system:setting:changed',
    'system:i18n:changed',
  ]);

  useEffect(() => {
    let active = true;
    const loadNoticeSummary = async () => {
      if (!token || !canViewNoticeSummary) {
        if (active) {
          setNoticeSummary(null);
          setNoticeLoading(false);
        }
        return;
      }
      setNoticeLoading(true);
      try {
        const data = await getDashboardSummary();
        if (active) {
          setNoticeSummary(data);
        }
      } catch {
        if (active) {
          setNoticeSummary(null);
        }
      } finally {
        if (active) {
          setNoticeLoading(false);
        }
      }
    };
    void loadNoticeSummary();
    return () => {
      active = false;
    };
  }, [canViewNoticeSummary, token]);

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (locked) {
        return;
      }
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setCommandVisible(true);
      }
    };
    globalThis.addEventListener('keydown', handleKeyDown);
    return () => globalThis.removeEventListener('keydown', handleKeyDown);
  }, [locked]);

  useEffect(() => {
    const nextTab: OpenedPageTab = {
      path: location.pathname,
      titleKey: currentTabTitleKey,
      fallbackTitle: currentPageTitle,
      closable: location.pathname !== '/dashboard',
      pinned: location.pathname === '/dashboard',
    };

    const mergeTabsIntoState = (
      currentTabs: OpenedPageTab[],
      currentNextTab: OpenedPageTab,
      dashboardTitle: string,
    ) => {
      const existingIndex = currentTabs.findIndex((item) => item.path === currentNextTab.path);
      const mergedTabs =
        existingIndex >= 0
          ? currentTabs.map((item, index) =>
              index === existingIndex
                ? { ...item, ...currentNextTab, pinned: item.pinned || currentNextTab.pinned }
                : item,
            )
          : [...currentTabs, currentNextTab];
      const normalizedTabs = mergedTabs.some((item) => item.path === '/dashboard')
        ? mergedTabs
        : [
            {
              path: '/dashboard',
              titleKey: 'dashboard.title',
              fallbackTitle: dashboardTitle,
              closable: false,
              pinned: true,
            },
            ...mergedTabs,
          ];
      const limitedTabs = limitOpenedTabs(
        normalizedTabs.map((item) => ({
          ...item,
          closable: item.path !== '/dashboard' && item.closable !== false,
          pinned: item.path === '/dashboard' || Boolean(item.pinned),
        })),
      );
      localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(limitedTabs));
      return limitedTabs;
    };

    const timer = globalThis.setTimeout(() => {
      setOpenedTabs((currentTabs) =>
        mergeTabsIntoState(currentTabs, nextTab, t('dashboard.title')),
      );
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [currentPageTitle, currentTabTitleKey, location.pathname, t]);

  useEffect(() => {
    if (!token || sessionIdleMs <= 0) {
      return;
    }
    const timer = globalThis.setInterval(() => {
      if (Date.now() - lastActivityAtRef.current >= sessionIdleMs) {
        void performLogout(true, 'session.idle_timeout');
      }
    }, 15000);
    return () => globalThis.clearInterval(timer);
  }, [locked, performLogout, sessionIdleMs, token]);

  useEffect(() => {
    if (!token || locked) {
      return;
    }

    const handleActivity = () => {
      recordActivity('interaction');
    };
    const handleVisible = () => {
      if (document.visibilityState === 'visible') {
        recordActivity('interaction');
      }
    };

    globalThis.addEventListener('pointerdown', handleActivity);
    globalThis.addEventListener('keydown', handleActivity);
    globalThis.addEventListener('scroll', handleActivity, true);
    globalThis.addEventListener('touchstart', handleActivity, true);
    document.addEventListener('visibilitychange', handleVisible);
    return () => {
      globalThis.removeEventListener('pointerdown', handleActivity);
      globalThis.removeEventListener('keydown', handleActivity);
      globalThis.removeEventListener('scroll', handleActivity, true);
      globalThis.removeEventListener('touchstart', handleActivity, true);
      document.removeEventListener('visibilitychange', handleVisible);
    };
  }, [locked, recordActivity, token]);

  useEffect(() => {
    document.documentElement.dataset.pantheonDensity = densityMode;
    persistShellDensityMode(densityMode);
  }, [densityMode]);

  useEffect(() => {
    recordActivity('route');
  }, [location.pathname, recordActivity]);

  const handleLogout = async () => {
    await performLogout(true);
  };

  const handleLockScreen = useCallback(() => {
    setCommandVisible(false);
    setCommandQuery('');
    persistShellLockedState(true);
    setLocked(true);
    setUnlockPassword('');
  }, []);

  const handleUnlock = useCallback(async () => {
    if (!unlockPassword.trim()) {
      return;
    }
    setUnlockLoading(true);
    try {
      await verifyOperationPassword(unlockPassword);
      persistShellLockedState(false);
      setLocked(false);
      setUnlockPassword('');
      recordActivity('unlock');
      Message.success(t('app.lock.unlockSuccess'));
    } finally {
      setUnlockLoading(false);
    }
  }, [recordActivity, t, unlockPassword]);

  const handleGoProfile = useCallback(() => {
    navigate('/system/profile');
  }, [navigate]);

  const handleGoSecurity = useCallback(() => {
    navigate('/auth/security');
  }, [navigate]);

  const handleUserMenuClick = (key: string) => {
    const actionKey = key as UserMenuActionKey;
    if (key === 'profile') {
      handleGoProfile();
      return;
    }
    if (key === 'security') {
      handleGoSecurity();
      return;
    }
    if (actionKey === 'lock') {
      handleLockScreen();
      return;
    }
    if (actionKey === 'logout') {
      void handleLogout();
    }
  };

  const walkMenuNodes = (
    nodes: MenuNode[],
    ancestors: MenuNode[],
    items: CommandSearchItem[],
    t: (key: string) => string,
  ) => {
    nodes.forEach((item) => {
        const trail = [...ancestors, item];
        if (item.path && item.type !== 'F') {
          const title = t(item.titleKey);
          const parentTrail = trail
            .slice(0, -1)
            .map((node) => t(node.titleKey))
            .join(' / ');
          items.push({
            key: `menu-${item.id}`,
            title,
            subtitle: parentTrail || item.path,
            section: t('app.command.section.menu'),
            searchText: [
              title,
              parentTrail,
              item.path,
              item.routeName,
              item.component,
              item.pagePerm,
              item.perms,
              item.module,
            ]
              .filter(Boolean)
              .join(' '),
            icon: renderMenuIcon(item.icon),
            run: () => {
              if (item.isExternal === 1) {
                globalThis.open(item.path, '_blank', 'noopener,noreferrer');
                return;
              }
              navigate(item.path);
            },
          });
        }
        if (item.children?.length) {
          walkMenuNodes(item.children, trail, items, t);
        }
      });
    }
  };

  const commandItems = useMemo<CommandSearchItem[]>(() => {
    const items: CommandSearchItem[] = [];

    if (canAccessDashboard) {
      items.push({
        key: 'quick-dashboard',
        title: t('dashboard.title'),
        subtitle: t('app.command.section.quick'),
        section: t('app.command.section.quick'),
        searchText: `${t('dashboard.title')} dashboard /dashboard`,
        icon: renderMenuIcon('dashboard'),
        run: () => navigate('/dashboard'),
      });
    }

    items.push({
      key: 'quick-profile',
      title: t('system.profile.title'),
      subtitle: t('app.command.section.quick'),
      section: t('app.command.section.quick'),
      searchText: `${t('system.profile.title')} profile /system/profile`,
      icon: <IconUser />,
      run: handleGoProfile,
    });
    items.push({
      key: 'quick-security',
      title: t('auth.security.title'),
      subtitle: t('app.command.section.quick'),
      section: t('app.command.section.quick'),
      searchText: `${t('auth.security.title')} security /auth/security`,
      icon: <IconSafe />,
      run: handleGoSecurity,
    });

    openedTabs.forEach((item) => {
      const title = item.titleKey ? t(item.titleKey) : item.fallbackTitle;
      items.push({
        key: `tab-${item.path}`,
        title,
        subtitle: item.path,
        section: t('app.command.section.opened'),
        searchText: `${title} ${item.path}`,
        icon: <IconPushpin />,
        run: () => navigate(item.path),
      });
    });

    walkMenuNodes(visibleMenuTree, [], items, t);
    return items;
  }, [
    canAccessDashboard,
    handleGoProfile,
    handleGoSecurity,
    navigate,
    openedTabs,
    t,
    visibleMenuTree,
  ]);

  const filteredCommandItems = useMemo(() => {
    const queryText = commandQuery.trim().toLowerCase();
    const filtered = queryText
      ? commandItems.filter((item) => item.searchText.toLowerCase().includes(queryText))
      : commandItems;
    return filtered.slice(0, 18);
  }, [commandItems, commandQuery]);

  const executeCommand = (item: CommandSearchItem) => {
    setCommandVisible(false);
    setCommandQuery('');
    item.run();
  };

  const noticeEntries = useMemo<NoticeEntry[]>(() => {
    const entries: NoticeEntry[] = [];

    if (canViewNoticeSummary) {
      entries.push({
        key: 'notice-security',
        title: t('auth.security.title'),
        description: t('app.notice.securityDesc'),
        icon: <IconSafe />,
        run: handleGoSecurity,
      });
    }

    if (isAdmin || hasPerm('system:session:list')) {
      entries.push({
        key: 'notice-session',
        title: t('system.menu.session'),
        description: t('app.notice.sessionDesc'),
        icon: renderMenuIcon('safe'),
        run: () => navigate('/system/session'),
      });
    }

    if (isAdmin || hasPerm('system:login-log:list')) {
      entries.push({
        key: 'notice-login-log',
        title: t('system.menu.loginLog'),
        description: t('app.notice.loginLogDesc'),
        icon: renderMenuIcon('safe'),
        run: () => navigate('/system/login-log'),
      });
    }

    if (isAdmin || hasPerm('system:security-event:list')) {
      entries.push({
        key: 'notice-security-event',
        title: t('system.menu.securityEvent'),
        description: t('app.notice.securityEventDesc'),
        icon: renderMenuIcon('safe'),
        run: () => navigate('/system/security-event'),
      });
    }

    if (isAdmin || hasPerm('system:operation-log:list')) {
      entries.push({
        key: 'notice-operation-log',
        title: t('system.menu.operationLog'),
        description: t('app.notice.operationLogDesc'),
        icon: renderMenuIcon('safe'),
        run: () => navigate('/system/operation-log'),
      });
    }

    return entries;
  }, [canViewNoticeSummary, handleGoSecurity, hasPerm, isAdmin, navigate, t]);

  const noticeBadgeCount = useMemo(() => {
    if (!noticeSummary) {
      return 0;
    }
    return Math.min(
      noticeSummary.loginFailureCount + noticeSummary.pendingSecurityEventCount,
      99,
    );
  }, [noticeSummary]);

  const noticeStatItems = useMemo(() => {
    if (!noticeSummary) {
      return [];
    }
    return [
      {
        key: 'failed-logins',
        label: t('app.notice.failedLogins'),
        value: noticeSummary.loginFailureCount,
        tone: noticeSummary.loginFailureCount > 0 ? 'danger' : 'neutral',
      },
      {
        key: 'sessions',
        label: t('app.notice.activeSessions'),
        value: noticeSummary.activeSessionCount,
        tone: 'neutral',
      },
      {
        key: 'security-events',
        label: t('app.notice.pendingSecurityEvents'),
        value: noticeSummary.pendingSecurityEventCount,
        tone: noticeSummary.pendingSecurityEventCount > 0 ? 'warning' : 'neutral',
      },
      {
        key: 'operations',
        label: t('app.notice.todayOperations'),
        value: noticeSummary.todayOperationCount,
        tone: 'neutral',
      },
    ];
  }, [noticeSummary, t]);

  const noticeRecentItems = useMemo(() => {
    if (!noticeSummary) {
      return [];
    }
    return noticeSummary.recentLogins.slice(0, 3).map((item) => ({
      id: item.id,
      username: item.username,
      status: item.status,
      time: formatDateTime(item.loginTime),
      message: t(item.msg || '', { defaultValue: item.msg || '-' }),
    }));
  }, [noticeSummary, t]);

  const noticeRiskGroups = useMemo<NoticeRiskItem[]>(() => {
    if (!noticeSummary) {
      return [];
    }
    const groups: NoticeRiskItem[] = [];

    if ((isAdmin || hasPerm('system:login-log:list')) && noticeSummary.loginFailureCount > 0) {
      groups.push({
        key: 'risk-login-failure',
        title: t('app.notice.risk.failedLogin'),
        description: t('app.notice.risk.failedLoginDesc'),
        value: noticeSummary.loginFailureCount,
        tone: 'danger',
        run: () => navigate('/system/login-log'),
      });
    }

    if (
      (isAdmin || hasPerm('system:security-event:list')) &&
      noticeSummary.pendingSecurityEventCount > 0
    ) {
      groups.push({
        key: 'risk-security-event',
        title: t('app.notice.risk.securityEvent'),
        description: t('app.notice.risk.securityEventDesc'),
        value: noticeSummary.pendingSecurityEventCount,
        tone: 'warning',
        run: () => navigate('/system/security-event'),
      });
    }

    if ((isAdmin || hasPerm('system:session:list')) && noticeSummary.activeSessionCount > 0) {
      groups.push({
        key: 'risk-active-session',
        title: t('app.notice.risk.activeSession'),
        description: t('app.notice.risk.activeSessionDesc'),
        value: noticeSummary.activeSessionCount,
        tone: 'neutral',
        run: () => navigate('/system/session'),
      });
    }

    if (
      (isAdmin || hasPerm('system:operation-log:list')) &&
      noticeSummary.todayOperationCount > 0
    ) {
      groups.push({
        key: 'risk-operation',
        title: t('app.notice.risk.operation'),
        description: t('app.notice.risk.operationDesc'),
        value: noticeSummary.todayOperationCount,
        tone: 'warning',
        run: () => navigate('/system/operation-log'),
      });
    }

    return groups;
  }, [hasPerm, isAdmin, navigate, noticeSummary, t]);
  const hasNoticeAttention = noticeBadgeCount > 0 || noticeRiskGroups.length > 0;
  const showNoticeCenter = canViewNoticeSummary;

  const closeTab = (targetPath: string) => {
    const targetTab = openedTabs.find((item) => item.path === targetPath);
    if (!targetTab?.closable || targetTab.pinned) {
      return;
    }
    const nextTabs = openedTabs.filter((item) => item.path !== targetPath);
    const safeTabs =
      nextTabs.length > 0
        ? nextTabs
        : [
            {
              path: '/dashboard',
              titleKey: 'dashboard.title',
              fallbackTitle: t('dashboard.title'),
              closable: false,
              pinned: true,
            },
          ];
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(safeTabs));
    setOpenedTabs(safeTabs);
    if (targetPath === location.pathname) {
      const fallbackTab = safeTabs[safeTabs.length - 1];
      navigate(fallbackTab.path);
    }
  };

  const closeOtherTabs = (targetPath: string) => {
    const targetTab = openedTabs.find((item) => item.path === targetPath);
    if (!targetTab) {
      return;
    }
    const nextTabs = orderOpenedTabs(
      openedTabs.filter((item) => item.pinned || item.path === targetPath),
    );
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(nextTabs));
    setOpenedTabs(nextTabs);
    if (location.pathname !== targetPath) {
      navigate(targetPath);
    }
  };

  const closeTabsToRight = (targetPath: string) => {
    const targetIndex = openedTabs.findIndex((item) => item.path === targetPath);
    if (targetIndex < 0) {
      return;
    }
    const nextTabs = orderOpenedTabs(
      openedTabs.filter((item, index) => item.pinned || index <= targetIndex),
    );
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(nextTabs));
    setOpenedTabs(nextTabs);
    if (!nextTabs.some((item) => item.path === location.pathname)) {
      navigate(targetPath);
    }
  };

  const closeAllTabs = () => {
    const nextTabs = orderOpenedTabs(openedTabs.filter((item) => item.pinned));
    const safeTabs =
      nextTabs.length > 0
        ? nextTabs
        : [
            {
              path: '/dashboard',
              titleKey: 'dashboard.title',
              fallbackTitle: t('dashboard.title'),
              closable: false,
              pinned: true,
            },
          ];
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(safeTabs));
    setOpenedTabs(safeTabs);
    if (!safeTabs.some((item) => item.path === location.pathname)) {
      navigate(safeTabs[safeTabs.length - 1].path);
    }
  };

  const togglePinTab = (targetPath: string) => {
    const nextTabs = orderOpenedTabs(
      openedTabs.map((item) => {
        if (item.path !== targetPath || item.path === '/dashboard') {
          return item;
        }
        return { ...item, pinned: !item.pinned };
      }),
    );
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(nextTabs));
    setOpenedTabs(nextTabs);
  };

  const moveTab = (dragPath: string, targetPath: string) => {
    if (dragPath === targetPath) {
      return;
    }
    const dragTab = openedTabs.find((item) => item.path === dragPath);
    const targetTab = openedTabs.find((item) => item.path === targetPath);
    if (
      !dragTab ||
      !targetTab ||
      dragTab.path === '/dashboard' ||
      targetTab.path === '/dashboard'
    ) {
      return;
    }
    if (Boolean(dragTab.pinned) !== Boolean(targetTab.pinned)) {
      return;
    }
    const nextTabs = [...openedTabs];
    const fromIndex = nextTabs.findIndex((item) => item.path === dragPath);
    const toIndex = nextTabs.findIndex((item) => item.path === targetPath);
    if (fromIndex < 0 || toIndex < 0) {
      return;
    }
    const [movedTab] = nextTabs.splice(fromIndex, 1);
    nextTabs.splice(toIndex, 0, movedTab);
    const orderedTabs = orderOpenedTabs(nextTabs);
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(orderedTabs));
    setOpenedTabs(orderedTabs);
  };

  const handleTabAction = (targetPath: string, action: TabActionKey) => {
    if (action === 'togglePin') {
      togglePinTab(targetPath);
      return;
    }
    if (action === 'close') {
      closeTab(targetPath);
      return;
    }
    if (action === 'closeOthers') {
      closeOtherTabs(targetPath);
      return;
    }
    if (action === 'closeRight') {
      closeTabsToRight(targetPath);
      return;
    }
    closeAllTabs();
  };

  const currentLanguage = (
    SUPPORTED_LOCALES.includes(i18n.language as SupportedLocale) ? i18n.language : 'zh-CN'
  ) as SupportedLocale;

  const persistPlatformPreferences = useCallback(
    (nextPreferences: Partial<UserPlatformPreferences>) => {
      const currentUserInfo = useAuthStore.getState().userInfo;
      if (!currentUserInfo) {
        return;
      }
      const mergedPreferences: UserPlatformPreferences = {
        ...currentUserInfo.preferences,
        ...nextPreferences,
      };

      setUserInfo({
        ...currentUserInfo,
        preferences: mergedPreferences,
      });

      void updateCurrentUserPreferences(mergedPreferences)
        .then((nextUserInfo) => {
          if (useAuthStore.getState().token === token) {
            setUserInfo(nextUserInfo);
          }
        })
        .catch(() => {
          if (useAuthStore.getState().token === token) {
            setUserInfo(currentUserInfo);
          }
          Message.error(t('app.preference.saveFailed'));
        });
    },
    [setUserInfo, t, token],
  );

  useEffect(() => {
    const preferences = userInfo?.preferences;
    if (!preferences) {
      return;
    }

    const timer = globalThis.setTimeout(() => {
      if (preferences.layoutMode && preferences.layoutMode !== layoutMode) {
        setLayoutMode(preferences.layoutMode);
        persistShellLayoutMode(preferences.layoutMode);
      }
      if (preferences.densityMode && preferences.densityMode !== densityMode) {
        setDensityMode(preferences.densityMode);
      }
      if (preferences.theme && preferences.theme !== theme) {
        setTheme(preferences.theme);
      }
      if (
        !hasExplicitLanguagePreference() &&
        preferences.language &&
        preferences.language !== currentLanguage
      ) {
        setExplicitLanguagePreference(preferences.language);
        void switchI18nLanguage(preferences.language);
      }
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [currentLanguage, densityMode, layoutMode, setTheme, theme, userInfo?.preferences]);

  const changeLanguage = (language: SupportedLocale) => {
    if (language === i18n.language) {
      return;
    }
    setExplicitLanguagePreference(language);
    void switchI18nLanguage(language);
  };

  const toggleLayoutMode = () => {
    setLayoutMode((currentMode) => {
      const nextMode: ShellLayoutMode = currentMode === 'vertical' ? 'horizontal' : 'vertical';
      persistShellLayoutMode(nextMode);
      persistPlatformPreferences({
        theme,
        layoutMode: nextMode,
        densityMode,
      });
      return nextMode;
    });
  };

  const changeDensityMode = (mode: ShellDensityMode) => {
    setDensityMode(mode);
    persistPlatformPreferences({
      theme,
      layoutMode,
      densityMode: mode,
    });
  };

  const preferencePanel = (
    <div className="app-shell__preference-panel">
      <div className="app-shell__preference-header">
        <span className="app-shell__preference-title">{t('app.preference.title')}</span>
        <span className="app-shell__preference-subtitle">{t('app.preference.subtitle')}</span>
      </div>

      <div className="app-shell__preference-section">
        <span className="app-shell__preference-section-title">
          {t('app.preference.navigation')}
        </span>
        <button type="button" className="app-shell__preference-item" onClick={toggleLayoutMode}>
          <span className="app-shell__preference-item-icon">
            <IconLayout />
          </span>
          <span className="app-shell__preference-item-copy">
            <span className="app-shell__preference-item-title">
              {t('app.preference.navigationMode')}
            </span>
            <span className="app-shell__preference-item-desc">{layoutModeActionLabel}</span>
          </span>
        </button>
      </div>

      <div className="app-shell__preference-section">
        <span className="app-shell__preference-section-title">{t('app.preference.density')}</span>
        <div className="app-shell__preference-pills">
          {(['comfortable', 'compact'] as ShellDensityMode[]).map((mode) => (
            <button
              key={mode}
              type="button"
              className={[
                'app-shell__preference-pill',
                densityMode === mode ? 'app-shell__preference-pill--active' : '',
              ]
                .join(' ')
                .trim()}
              onClick={() => changeDensityMode(mode)}
            >
              <span>{t(`app.density.${mode}`)}</span>
              <span>{t(`app.density.${mode}.description`)}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="app-shell__preference-section">
        <span className="app-shell__preference-section-title">{t('app.preference.language')}</span>
        <div className="app-shell__preference-pills">
          {SUPPORTED_LOCALES.map((language) => (
            <button
              key={language}
              type="button"
              className={[
                'app-shell__preference-pill',
                currentLanguage === language ? 'app-shell__preference-pill--active' : '',
              ]
                .join(' ')
                .trim()}
              onClick={() => changeLanguage(language)}
            >
              <IconLanguage />
              <span>{t(`app.language.${language}`)}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="app-shell__preference-section">
        <span className="app-shell__preference-section-title">{t('app.preference.theme')}</span>
        <div className="app-shell__preference-theme-list">
          {themeOptions.map((item) => (
            <button
              key={item.key}
              type="button"
              className={[
                'app-shell__preference-item',
                theme === item.key ? 'app-shell__preference-item--active' : '',
              ]
                .join(' ')
                .trim()}
              onClick={() => {
                setTheme(item.key);
                persistPlatformPreferences({
                  theme: item.key,
                  layoutMode,
                  densityMode,
                });
              }}
            >
              <span className="app-shell__preference-item-icon">
                <span
                  className="app-shell__preference-theme-swatch"
                  style={{ background: item.accent }}
                />
              </span>
              <span className="app-shell__preference-item-copy">
                <span className="app-shell__preference-item-title">{t(item.labelKey)}</span>
                <span className="app-shell__preference-item-desc">{t(item.descriptionKey)}</span>
              </span>
              {theme === item.key ? <IconCheck className="app-shell__preference-check" /> : null}
            </button>
          ))}
        </div>
      </div>
    </div>
  );

  const handleMenuNavigation = (key: string) => {
    const selected = findMenuNodeByPath(visibleMenuTree, key);
    if (selected?.isExternal === 1) {
      globalThis.open(selected.path, '_blank', 'noopener,noreferrer');
      return;
    }
    navigate(key);
  };

  const renderMenuItems = (nodes: MenuNode[], level = 0) =>
    nodes.map((item) => {
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
        return (
          <Menu.SubMenu
            key={item.id.toString()}
            title={
              <span
                className={entryClassName}
                role="button"
                tabIndex={0}
                onClick={() => {
                  if (navigationPath) {
                    handleMenuNavigation(navigationPath);
                  }
                }}
                onKeyDown={(event) => {
                  if ((event.key === 'Enter' || event.key === ' ') && navigationPath) {
                    event.preventDefault();
                    handleMenuNavigation(navigationPath);
                  }
                }}
              >
                <span className={iconClassName}>{renderMenuIcon(item.icon)}</span>
                <span className="app-shell__menu-entry-copy">
                  <span className="app-shell__menu-entry-label">{t(item.titleKey)}</span>
                </span>
              </span>
            }
          >
            {renderMenuItems(item.children, level + 1)}
          </Menu.SubMenu>
        );
      }
      return (
        <Menu.Item key={item.path}>
          <span
            className={entryClassName}
            onMouseEnter={() => {
              void preloadRouteComponent(item.path);
            }}
            onFocus={() => {
              void preloadRouteComponent(item.path);
            }}
          >
            <span className={iconClassName}>{renderMenuIcon(item.icon)}</span>
            <span className="app-shell__menu-entry-copy">
              <span className="app-shell__menu-entry-label">{t(item.titleKey)}</span>
            </span>
          </span>
        </Menu.Item>
      );
    });

  const openedTabsContent = publicSettings.enableTabBar ? (
    <div
      className={[
        'app-shell__tabs',
        isHorizontalLayout ? 'app-shell__tabs--horizontal' : 'app-shell__tabs--vertical',
      ].join(' ')}
      role="tablist"
      aria-label={t('app.openedTabs')}
    >
      {openedTabs.map((item) => {
        const active = item.path === location.pathname;
        const itemIndex = openedTabs.findIndex((tab) => tab.path === item.path);
        const canCloseCurrent = item.closable && !item.pinned;
        const canCloseOthers = openedTabs.some(
          (tab) => tab.path !== item.path && tab.closable && !tab.pinned,
        );
        const canCloseRight = openedTabs
          .slice(itemIndex + 1)
          .some((tab) => tab.closable && !tab.pinned);
        const canCloseAll = openedTabs.some((tab) => tab.closable && !tab.pinned);
        return (
          <Dropdown
            key={item.path}
            trigger="contextMenu"
            position="bl"
            droplist={
              <Menu
                onClickMenuItem={(key) => handleTabAction(item.path, key as TabActionKey)}
                className="app-shell__tab-menu"
              >
                <Menu.Item key="togglePin" disabled={item.path === '/dashboard'}>
                  {item.pinned ? t('app.tab.unpin') : t('app.tab.pin')}
                </Menu.Item>
                <Menu.Item key="close" disabled={!canCloseCurrent}>
                  {t('common.close')}
                </Menu.Item>
                <Menu.Item key="closeOthers" disabled={!canCloseOthers}>
                  {t('app.tab.closeOthers')}
                </Menu.Item>
                <Menu.Item key="closeRight" disabled={!canCloseRight}>
                  {t('app.tab.closeRight')}
                </Menu.Item>
                <Menu.Item key="closeAll" disabled={!canCloseAll}>
                  {t('app.tab.closeAll')}
                </Menu.Item>
              </Menu>
            }
          >
            <div
              role="tab"
              tabIndex={0}
              aria-selected={active}
              className={[
                'app-shell__tab',
                active ? 'app-shell__tab--active' : '',
                item.pinned ? 'app-shell__tab--pinned' : '',
                draggingTabPath === item.path ? 'app-shell__tab--dragging' : '',
                dragOverTabPath === item.path ? 'app-shell__tab--drag-over' : '',
              ]
                .filter(Boolean)
                .join(' ')}
              draggable={item.path !== '/dashboard'}
              onClick={() => navigate(item.path)}
              onMouseEnter={() => {
                void preloadRouteComponent(item.path);
              }}
              onFocus={() => {
                void preloadRouteComponent(item.path);
              }}
              onDoubleClick={() => closeTab(item.path)}
              onMouseDown={(event) => {
                if (event.button === 1) {
                  event.preventDefault();
                }
              }}
              onAuxClick={(event) => {
                if (event.button === 1) {
                  event.preventDefault();
                  closeTab(item.path);
                }
              }}
              onDragStart={(event) => {
                if (item.path === '/dashboard') {
                  event.preventDefault();
                  return;
                }
                setDraggingTabPath(item.path);
                event.dataTransfer.effectAllowed = 'move';
                event.dataTransfer.setData('text/plain', item.path);
              }}
              onDragOver={(event) => {
                if (!draggingTabPath || draggingTabPath === item.path) {
                  return;
                }
                const dragTab = openedTabs.find((tab) => tab.path === draggingTabPath);
                if (
                  !dragTab ||
                  dragTab.path === '/dashboard' ||
                  Boolean(dragTab.pinned) !== Boolean(item.pinned)
                ) {
                  return;
                }
                event.preventDefault();
                event.dataTransfer.dropEffect = 'move';
                if (dragOverTabPath !== item.path) {
                  setDragOverTabPath(item.path);
                }
              }}
              onDragLeave={() => {
                if (dragOverTabPath === item.path) {
                  setDragOverTabPath(null);
                }
              }}
              onDrop={(event) => {
                event.preventDefault();
                if (draggingTabPath) {
                  moveTab(draggingTabPath, item.path);
                }
                setDraggingTabPath(null);
                setDragOverTabPath(null);
              }}
              onDragEnd={() => {
                setDraggingTabPath(null);
                setDragOverTabPath(null);
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter' || event.key === ' ') {
                  event.preventDefault();
                  navigate(item.path);
                }
              }}
            >
              {item.pinned ? <IconPushpin className="app-shell__tab-pin" /> : null}
              <span className="app-shell__tab-label">
                {item.titleKey ? t(item.titleKey) : item.fallbackTitle}
              </span>
              {canCloseCurrent ? (
                <span
                  role="button"
                  tabIndex={0}
                  aria-label={t('common.close')}
                  className="app-shell__tab-close"
                  onClick={(event) => {
                    event.stopPropagation();
                    closeTab(item.path);
                  }}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      event.stopPropagation();
                      closeTab(item.path);
                    }
                  }}
                >
                  <IconClose />
                </span>
              ) : null}
            </div>
          </Dropdown>
        );
      })}
    </div>
  ) : null;

  return (
    <Layout
      className={[
        'app-shell',
        isHorizontalLayout ? 'app-shell--horizontal' : 'app-shell--vertical',
      ].join(' ')}
    >
      {!isHorizontalLayout ? (
        <Sider
          className="app-shell__sider"
          theme="light"
          trigger={null}
          width={248}
          collapsedWidth={76}
          collapsed={collapsed}
          collapsible
          breakpoint="xl"
          onCollapse={setCollapsed}
        >
          <div
            className={
              collapsed ? 'app-shell__brand app-shell__brand--collapsed' : 'app-shell__brand'
            }
          >
            <div className="app-shell__brand-mark">
              {publicSettings.siteLogo ? (
                <img src={publicSettings.siteLogo} alt={appName} />
              ) : (
                brandInitial
              )}
            </div>
            {!collapsed ? (
              <div className="app-shell__brand-text">
                <span className="app-shell__brand-title">{appName}</span>
              </div>
            ) : null}
          </div>
          <Spin loading={loading} className="app-shell__menu-loading">
            <Menu
              key={`${collapsed ? 'collapsed' : 'expanded'}-${menuOpenKeys.join(',')}`}
              className="app-shell__menu"
              theme="light"
              selectedKeys={[selectedMenuPath]}
              defaultOpenKeys={menuOpenKeys}
              onClickMenuItem={handleMenuNavigation}
            >
              {renderMenuItems(visibleMenuTree)}
            </Menu>
          </Spin>
        </Sider>
      ) : null}
      <Layout className="app-shell__main">
        <Header className="app-shell__header">
          <div className="app-shell__header-left">
            {!isHorizontalLayout ? (
              <Button
                type="text"
                className="app-shell__collapse-btn"
                icon={collapsed ? <IconMenuUnfold /> : <IconMenuFold />}
                onClick={() => setCollapsed((value) => !value)}
              />
            ) : (
              <div className="app-shell__header-brand" aria-label={appName}>
                <span className="app-shell__header-brand-mark">
                  {publicSettings.siteLogo ? (
                    <img src={publicSettings.siteLogo} alt={appName} />
                  ) : (
                    brandInitial
                  )}
                </span>
                <span className="app-shell__header-brand-text">{appName}</span>
              </div>
            )}
            <div className="app-shell__header-meta">
              <Breadcrumb className="app-shell__header-breadcrumb">
                {breadcrumbItems.map((item) => (
                  <Breadcrumb.Item key={`${item.path}-${item.label}`}>{item.label}</Breadcrumb.Item>
                ))}
              </Breadcrumb>
            </div>
          </div>
          <Space size={12} className="app-shell__header-actions">
            <button
              type="button"
              className="app-shell__search-trigger"
              onClick={() => setCommandVisible(true)}
              aria-label={t('app.command.title')}
            >
              <IconSearch />
              <span className="app-shell__search-placeholder">{t('app.command.placeholder')}</span>
              <kbd className="app-shell__search-shortcut">Ctrl K</kbd>
            </button>
            {showNoticeCenter ? (
              <Dropdown
                trigger="click"
                position="br"
                droplist={
                  <div className="app-shell__notice-panel">
                    <div className="app-shell__notice-header">
                      <span className="app-shell__notice-title">{t('app.notice.title')}</span>
                      <span className="app-shell__notice-subtitle">{t('app.notice.subtitle')}</span>
                    </div>
                    {noticeLoading ? (
                      <div className="app-shell__notice-empty">{t('common.loading')}</div>
                    ) : noticeSummary ? (
                      <>
                        <div className="app-shell__notice-stats">
                          {noticeStatItems.map((item) => (
                            <div
                              key={item.key}
                              className={[
                                'app-shell__notice-stat',
                                item.tone === 'danger' ? 'app-shell__notice-stat--danger' : '',
                              ]
                                .filter(Boolean)
                                .join(' ')}
                            >
                              <span className="app-shell__notice-stat-label">{item.label}</span>
                              <span className="app-shell__notice-stat-value">{item.value}</span>
                            </div>
                          ))}
                        </div>
                        <div className="app-shell__notice-summary">
                          <span className="app-shell__notice-summary-label">
                            {t('app.notice.lastSuccess')}
                          </span>
                          <span className="app-shell__notice-summary-value">
                            {noticeSummary.lastSuccessfulLoginAt
                              ? formatDateTime(noticeSummary.lastSuccessfulLoginAt)
                              : t('dashboard.lastSuccessfulLoginEmpty')}
                          </span>
                        </div>
                        {noticeRiskGroups.length > 0 ? (
                          <div className="app-shell__notice-list">
                            <div className="app-shell__notice-section">
                              {t('app.notice.section.risk')}
                            </div>
                            {noticeRiskGroups.map((item) => (
                              <button
                                key={item.key}
                                type="button"
                                className={[
                                  'app-shell__notice-risk',
                                  `app-shell__notice-risk--${item.tone}`,
                                ].join(' ')}
                                onClick={item.run}
                              >
                                <span className="app-shell__notice-risk-copy">
                                  <span className="app-shell__notice-risk-title">{item.title}</span>
                                  <span className="app-shell__notice-risk-desc">
                                    {item.description}
                                  </span>
                                </span>
                                <span className="app-shell__notice-risk-value">{item.value}</span>
                              </button>
                            ))}
                          </div>
                        ) : null}
                        <div className="app-shell__notice-list">
                          <div className="app-shell__notice-section">
                            {t('app.notice.section.recent')}
                          </div>
                          {noticeRecentItems.length > 0 ? (
                            noticeRecentItems.map((item) => (
                              <div key={item.id} className="app-shell__notice-log">
                                <span
                                  className={`app-shell__notice-log-dot ${item.status === 1 ? 'app-shell__notice-log-dot--success' : 'app-shell__notice-log-dot--danger'}`}
                                />
                                <span className="app-shell__notice-log-copy">
                                  <span className="app-shell__notice-log-title">
                                    {item.username}
                                  </span>
                                  <span className="app-shell__notice-log-desc">{item.message}</span>
                                </span>
                                <span className="app-shell__notice-log-time">{item.time}</span>
                              </div>
                            ))
                          ) : (
                            <div className="app-shell__notice-empty app-shell__notice-empty--compact">
                              {t('dashboard.recentLoginsEmpty')}
                            </div>
                          )}
                        </div>
                      </>
                    ) : (
                      <div className="app-shell__notice-empty">{t('app.notice.empty')}</div>
                    )}
                    {noticeEntries.length > 0 ? (
                      <div className="app-shell__notice-list">
                        <div className="app-shell__notice-section">
                          {t('app.notice.section.recommended')}
                        </div>
                        {noticeEntries.map((item) => (
                          <button
                            key={item.key}
                            type="button"
                            className="app-shell__notice-item"
                            onClick={item.run}
                          >
                            <span className="app-shell__notice-item-icon">{item.icon}</span>
                            <span className="app-shell__notice-item-copy">
                              <span className="app-shell__notice-item-title">{item.title}</span>
                              <span className="app-shell__notice-item-desc">
                                {item.description}
                              </span>
                            </span>
                          </button>
                        ))}
                      </div>
                    ) : null}
                  </div>
                }
              >
                <Tooltip
                  content={hasNoticeAttention ? t('app.notice.attention') : t('app.notice.title')}
                >
                  <Button
                    type="text"
                    className={[
                      'app-shell__icon-btn',
                      hasNoticeAttention ? 'app-shell__icon-btn--attention' : '',
                    ]
                      .join(' ')
                      .trim()}
                    icon={<IconNotification />}
                    aria-label={t('app.notice.title')}
                  >
                    {noticeBadgeCount > 0 ? (
                      <span className="app-shell__notice-badge">
                        {noticeBadgeCount > 99 ? '99+' : noticeBadgeCount}
                      </span>
                    ) : null}
                  </Button>
                </Tooltip>
              </Dropdown>
            ) : null}
            <Dropdown trigger="click" position="br" droplist={preferencePanel}>
              <Tooltip
                content={t('app.preference.tooltip', {
                  theme: t(activeTheme.labelKey),
                  layout: layoutModeLabel,
                  density: densityModeLabel,
                  language: t(`app.language.${currentLanguage}`),
                })}
              >
                <Button
                  type="text"
                  className="app-shell__icon-btn"
                  icon={<IconSettings />}
                  aria-label={t('app.preference.title')}
                />
              </Tooltip>
            </Dropdown>
            <Dropdown
              position="br"
              droplist={
                <Menu onClickMenuItem={handleUserMenuClick}>
                  <Menu.Item key="profile">
                    <IconUser />
                    {t('system.profile.title')}
                  </Menu.Item>
                  <Menu.Item key="security">
                    <IconSafe />
                    {t('auth.security.title')}
                  </Menu.Item>
                  <Menu.Item key="lock">
                    <IconLock />
                    {t('app.lock.action')}
                  </Menu.Item>
                  <Menu.Item key="logout">
                    <IconPoweroff />
                    {t('common.logout')}
                  </Menu.Item>
                </Menu>
              }
            >
              <Button type="text" className="app-shell__user-trigger">
                <Avatar size={28}>
                  {userInfo?.avatar ? (
                    <img src={userInfo.avatar} alt={userDisplayName} />
                  ) : (
                    userDisplayName.slice(0, 1).toUpperCase()
                  )}
                </Avatar>
                <div className="app-shell__user-meta">
                  <span className="app-shell__user-name">{userDisplayName}</span>
                  {roleLabel ? (
                    <span className="app-shell__user-subtitle">{roleLabel}</span>
                  ) : null}
                </div>
              </Button>
            </Dropdown>
          </Space>
        </Header>
        {isHorizontalLayout ? (
          <div className="app-shell__top-nav">
            <Spin loading={loading} className="app-shell__menu-loading">
              <Menu
                mode="horizontal"
                className="app-shell__top-menu"
                selectedKeys={[selectedMenuPath]}
                triggerProps={{ className: 'app-shell__top-menu-popup' }}
                onClickMenuItem={handleMenuNavigation}
              >
                {renderMenuItems(visibleMenuTree)}
              </Menu>
            </Spin>
          </div>
        ) : null}
        {isHorizontalLayout ? openedTabsContent : null}
        {!isHorizontalLayout ? openedTabsContent : null}
        <Content className="app-shell__content">
          <div className="app-shell__content-inner">
            <Outlet />
          </div>
        </Content>
        <Footer className="app-shell__footer">{t('app.footer')}</Footer>
      </Layout>
      <AppModal
        title={t('app.lock.title')}
        visible={locked}
        size="sm"
        footer={null}
        closable={false}
        maskClosable={false}
        className="app-shell__lock-modal"
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Typography.Text type="secondary">
            {t('app.lock.description', { minutes: sessionIdleMinutes })}
          </Typography.Text>
          <Input.Password
            autoFocus
            value={unlockPassword}
            placeholder={t('app.lock.passwordPlaceholder')}
            onChange={setUnlockPassword}
            onPressEnter={() => {
              void handleUnlock();
            }}
          />
          <Space>
            <Button
              type="primary"
              loading={unlockLoading}
              onClick={() => {
                void handleUnlock();
              }}
            >
              {t('app.lock.unlock')}
            </Button>
            <Button
              onClick={() => {
                void handleLogout();
              }}
            >
              {t('common.logout')}
            </Button>
          </Space>
        </Space>
      </AppModal>
      <AppModal
        title={t('app.command.title')}
        visible={commandVisible}
        size="md"
        footer={null}
        className="app-command"
        onCancel={() => {
          setCommandVisible(false);
          setCommandQuery('');
        }}
      >
        <Input
          autoFocus
          allowClear
          value={commandQuery}
          prefix={<IconSearch />}
          placeholder={t('app.command.placeholder')}
          className="app-command__input"
          onChange={setCommandQuery}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && filteredCommandItems[0]) {
              event.preventDefault();
              executeCommand(filteredCommandItems[0]);
            }
          }}
        />
        <div className="app-command__results">
          {filteredCommandItems.length > 0 ? (
            filteredCommandItems.map((item, index) => {
              const previousItem = filteredCommandItems[index - 1];
              const showSection = !previousItem || previousItem.section !== item.section;
              return (
                <React.Fragment key={item.key}>
                  {showSection ? <div className="app-command__section">{item.section}</div> : null}
                  <button
                    type="button"
                    className="app-command__item"
                    onClick={() => executeCommand(item)}
                  >
                    <span className="app-command__item-icon">{item.icon}</span>
                    <span className="app-command__item-copy">
                      <span className="app-command__item-title">{item.title}</span>
                      <span className="app-command__item-subtitle">{item.subtitle}</span>
                    </span>
                  </button>
                </React.Fragment>
              );
            })
          ) : (
            <Empty description={t('app.command.empty')} />
          )}
        </div>
      </AppModal>
    </Layout>
  );
};

export default BaseLayout;
