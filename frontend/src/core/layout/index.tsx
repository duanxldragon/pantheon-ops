import React, {
  forwardRef,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Avatar,
  Button,
  Dropdown,
  Empty,
  Input,
  Layout,
  Menu,
  Message,
  Space,
  Tooltip,
  Typography,
} from '@arco-design/web-react';
import {
  IconCheck,
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
import { type MenuNode } from '../../modules/system/menu/api';
import { logout as logoutApi, reportActivity } from '../../modules/auth/session/api';
import {
  updateCurrentUserPreferences,
  verifyOperationPassword,
  type UserPlatformPreferences,
} from '../../modules/auth/security/api';
import { ensureAuthUserInfo } from '../auth/bootstrap';
import { useMenuStore } from '../../store/useMenuStore';
import { useAuthStore } from '../../store/useAuthStore';
import { usePermission } from '../../hooks/usePermission';
import { useRefreshPolling, useRefreshSubscription } from '../refresh/refreshBus';
import { findRouteByPath, systemRouteTitleMap } from '../router/modules';
import { formatDateTime } from '../format/dateTime';
import { renderMenuIcon } from '../menu/icon';
import { useTheme } from '../../hooks';
import { clearPantheonThemePreference } from '../theme/theme';
import { usePantheonColorMode, type PantheonColorMode } from '../theme/colorMode';
import { AppModal, shouldShowIdentityLabel, UserAvatarContent } from '../../components';
import { getDashboardSummary, type DashboardSummary } from '../../modules/platform/api';
import { clearClientAuthSession } from '../auth/clientSession';
import {
  getBrandInitial,
  hasExplicitLanguagePreference,
  refreshPublicSettings,
  setExplicitLanguagePreference,
  usePublicSettings,
} from '../settings/publicSettings';
import { clearExplicitLanguagePreference } from '../settings/languagePreference';
import { SUPPORTED_LOCALES, switchI18nLanguage, type SupportedLocale } from '../../i18n';
import {
  shouldLoadShellNoticeSummary,
  shouldPollServerRefreshState,
  shouldReportShellActivity,
} from '../runtime/automationPolicy';
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
import LayoutBreadcrumb from './LayoutBreadcrumb';
import LayoutOpenedTabs from './LayoutOpenedTabs';
import LayoutSideMenu from './LayoutSideMenu';
import {
  buildOpenedPageTab,
  mergeOpenedTabsIntoState,
  orderOpenedTabs,
  readOpenedTabs,
  type OpenedPageTab,
  type TabActionKey,
} from './layoutTabs';
import { findMenuNodeByPath, useLayoutMenu } from './useLayoutMenu';
import './index.css';

const { Header, Footer, Content } = Layout;
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

interface NoticeStatItem {
  key: string;
  label: string;
  value: number;
  tone: 'danger' | 'warning' | 'neutral';
}

interface NoticeRecentItem {
  id: string | number;
  username: string;
  status: number;
  time: string;
  message: string;
}

type TranslateLabel = (key: string, options?: Record<string, unknown>) => string;
type NavigateToPath = (path: string) => void;

async function resolveSilently<T>(task: Promise<T>): Promise<T | undefined> {
  try {
    return await task;
  } catch {
    return undefined;
  }
}

function runSilently(task: Promise<unknown>) {
  task.catch(() => undefined);
}

function walkMenuNodes(
  nodes: MenuNode[],
  ancestors: MenuNode[],
  items: CommandSearchItem[],
  t: TranslateLabel,
  navigate: NavigateToPath,
) {
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
      walkMenuNodes(item.children, trail, items, t, navigate);
    }
  });
}

function buildCommandItems(options: {
  canAccessDashboard: boolean;
  visibleMenuTree: MenuNode[];
  t: TranslateLabel;
  navigate: NavigateToPath;
  handleGoProfile: () => void;
  handleGoSecurity: () => void;
}): CommandSearchItem[] {
  const { canAccessDashboard, visibleMenuTree, t, navigate, handleGoProfile, handleGoSecurity } =
    options;
  const items: CommandSearchItem[] = canAccessDashboard
    ? [
        {
          key: 'quick-dashboard',
          title: t('dashboard.title'),
          subtitle: t('app.command.section.quick'),
          section: t('app.command.section.quick'),
          searchText: `${t('dashboard.title')} dashboard /dashboard`,
          icon: renderMenuIcon('dashboard'),
          run: () => navigate('/dashboard'),
        },
      ]
    : [];

  items.push(
    {
      key: 'quick-profile',
      title: t('system.profile.title'),
      subtitle: t('app.command.section.quick'),
      section: t('app.command.section.quick'),
      searchText: `${t('system.profile.title')} profile /system/profile`,
      icon: <IconUser />,
      run: handleGoProfile,
    },
    {
      key: 'quick-security',
      title: t('auth.security.title'),
      subtitle: t('app.command.section.quick'),
      section: t('app.command.section.quick'),
      searchText: `${t('auth.security.title')} security /auth/security`,
      icon: <IconSafe />,
      run: handleGoSecurity,
    },
  );

  walkMenuNodes(visibleMenuTree, [], items, t, navigate);
  return items;
}

function buildNoticeEntries(options: {
  canViewNoticeSummary: boolean;
  isAdmin: boolean;
  hasPerm: (permission: string) => boolean;
  t: TranslateLabel;
  navigate: NavigateToPath;
  handleGoSecurity: () => void;
}): NoticeEntry[] {
  const { canViewNoticeSummary, isAdmin, hasPerm, t, navigate, handleGoSecurity } = options;
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
}

function buildNoticeStatItems(
  summary: DashboardSummary | null,
  t: TranslateLabel,
): NoticeStatItem[] {
  if (!summary) {
    return [];
  }
  return [
    {
      key: 'failed-logins',
      label: t('app.notice.failedLogins'),
      value: summary.loginFailureCount,
      tone: summary.loginFailureCount > 0 ? 'danger' : 'neutral',
    },
    {
      key: 'sessions',
      label: t('app.notice.activeSessions'),
      value: summary.activeSessionCount,
      tone: 'neutral',
    },
    {
      key: 'security-events',
      label: t('app.notice.pendingSecurityEvents'),
      value: summary.pendingSecurityEventCount,
      tone: summary.pendingSecurityEventCount > 0 ? 'warning' : 'neutral',
    },
    {
      key: 'operations',
      label: t('app.notice.todayOperations'),
      value: summary.todayOperationCount,
      tone: 'neutral',
    },
  ];
}

function buildNoticeRecentItems(
  summary: DashboardSummary | null,
  t: TranslateLabel,
): NoticeRecentItem[] {
  if (!summary) {
    return [];
  }
  return summary.recentLogins.slice(0, 3).map((item) => ({
    id: item.id,
    username: item.username,
    status: item.status,
    time: formatDateTime(item.loginTime, { withSeconds: true }),
    message: t(item.msg || '', { defaultValue: item.msg || '-' }),
  }));
}

function buildNoticeRiskGroups(options: {
  summary: DashboardSummary | null;
  isAdmin: boolean;
  hasPerm: (permission: string) => boolean;
  t: TranslateLabel;
  navigate: NavigateToPath;
}): NoticeRiskItem[] {
  const { summary, isAdmin, hasPerm, t, navigate } = options;
  if (!summary) {
    return [];
  }
  const groups: NoticeRiskItem[] = [];

  if ((isAdmin || hasPerm('system:login-log:list')) && summary.loginFailureCount > 0) {
    groups.push({
      key: 'risk-login-failure',
      title: t('app.notice.risk.failedLogin'),
      description: t('app.notice.risk.failedLoginDesc'),
      value: summary.loginFailureCount,
      tone: 'danger',
      run: () => navigate('/system/login-log'),
    });
  }

  if ((isAdmin || hasPerm('system:security-event:list')) && summary.pendingSecurityEventCount > 0) {
    groups.push({
      key: 'risk-security-event',
      title: t('app.notice.risk.securityEvent'),
      description: t('app.notice.risk.securityEventDesc'),
      value: summary.pendingSecurityEventCount,
      tone: 'warning',
      run: () => navigate('/system/security-event'),
    });
  }

  if ((isAdmin || hasPerm('system:session:list')) && summary.activeSessionCount > 0) {
    groups.push({
      key: 'risk-active-session',
      title: t('app.notice.risk.activeSession'),
      description: t('app.notice.risk.activeSessionDesc'),
      value: summary.activeSessionCount,
      tone: 'neutral',
      run: () => navigate('/system/session'),
    });
  }

  if ((isAdmin || hasPerm('system:operation-log:list')) && summary.todayOperationCount > 0) {
    groups.push({
      key: 'risk-operation',
      title: t('app.notice.risk.operation'),
      description: t('app.notice.risk.operationDesc'),
      value: summary.todayOperationCount,
      tone: 'warning',
      run: () => navigate('/system/operation-log'),
    });
  }

  return groups;
}

// ── 页签操作纯函数 ──────────────────────────────────────────────
// 计算与副作用分离：以下 computeTab* 只算下一状态与跳转目标，
// 组件内统一 applyTabUpdate 落地，压 BaseLayout 认知复杂度。

type TabUpdate = { tabs: OpenedPageTab[]; navigateTo?: string } | null;

function dashboardFallbackTab(dashboardTitle: string): OpenedPageTab {
  return {
    path: '/dashboard',
    titleKey: 'dashboard.title',
    fallbackTitle: dashboardTitle,
    closable: false,
    pinned: true,
  };
}

function ensureTabsNotEmpty(tabs: OpenedPageTab[], dashboardTitle: string): OpenedPageTab[] {
  return tabs.length > 0 ? tabs : [dashboardFallbackTab(dashboardTitle)];
}

function computeTabClose(
  tabs: OpenedPageTab[],
  targetPath: string,
  currentPath: string,
  dashboardTitle: string,
): TabUpdate {
  const targetTab = tabs.find((item) => item.path === targetPath);
  if (!targetTab?.closable || targetTab.pinned) {
    return null;
  }
  const safeTabs = ensureTabsNotEmpty(
    tabs.filter((item) => item.path !== targetPath),
    dashboardTitle,
  );
  const navigateTo = targetPath === currentPath ? safeTabs.at(-1)?.path || '/dashboard' : undefined;
  return { tabs: safeTabs, navigateTo };
}

function computeTabCloseOthers(
  tabs: OpenedPageTab[],
  targetPath: string,
  currentPath: string,
): TabUpdate {
  if (!tabs.some((item) => item.path === targetPath)) {
    return null;
  }
  const nextTabs = orderOpenedTabs(tabs.filter((item) => item.pinned || item.path === targetPath));
  return { tabs: nextTabs, navigateTo: currentPath !== targetPath ? targetPath : undefined };
}

function computeTabCloseToRight(
  tabs: OpenedPageTab[],
  targetPath: string,
  currentPath: string,
): TabUpdate {
  const targetIndex = tabs.findIndex((item) => item.path === targetPath);
  if (targetIndex < 0) {
    return null;
  }
  const nextTabs = orderOpenedTabs(
    tabs.filter((item, index) => item.pinned || index <= targetIndex),
  );
  const stillVisible = nextTabs.some((item) => item.path === currentPath);
  return { tabs: nextTabs, navigateTo: stillVisible ? undefined : targetPath };
}

function computeTabCloseAll(
  tabs: OpenedPageTab[],
  currentPath: string,
  dashboardTitle: string,
): TabUpdate {
  const safeTabs = ensureTabsNotEmpty(
    orderOpenedTabs(tabs.filter((item) => item.pinned)),
    dashboardTitle,
  );
  const stillVisible = safeTabs.some((item) => item.path === currentPath);
  return {
    tabs: safeTabs,
    navigateTo: stillVisible ? undefined : safeTabs.at(-1)?.path || '/dashboard',
  };
}

function computeTabTogglePin(tabs: OpenedPageTab[], targetPath: string): TabUpdate {
  const nextTabs = orderOpenedTabs(
    tabs.map((item) => {
      if (item.path !== targetPath || item.path === '/dashboard') {
        return item;
      }
      return { ...item, pinned: !item.pinned };
    }),
  );
  return { tabs: nextTabs };
}

function computeTabMove(tabs: OpenedPageTab[], dragPath: string, targetPath: string): TabUpdate {
  if (dragPath === targetPath) {
    return null;
  }
  const dragTab = tabs.find((item) => item.path === dragPath);
  const targetTab = tabs.find((item) => item.path === targetPath);
  if (!dragTab || !targetTab || dragTab.path === '/dashboard' || targetTab.path === '/dashboard') {
    return null;
  }
  if (Boolean(dragTab.pinned) !== Boolean(targetTab.pinned)) {
    return null;
  }
  const nextTabs = [...tabs];
  const fromIndex = nextTabs.findIndex((item) => item.path === dragPath);
  const toIndex = nextTabs.findIndex((item) => item.path === targetPath);
  if (fromIndex < 0 || toIndex < 0) {
    return null;
  }
  const [movedTab] = nextTabs.splice(fromIndex, 1);
  nextTabs.splice(toIndex, 0, movedTab);
  return { tabs: orderOpenedTabs(nextTabs) };
}

async function fetchNoticeSummarySafely(): Promise<DashboardSummary | null> {
  try {
    return await getDashboardSummary();
  } catch {
    return null;
  }
}

function pickText(...values: Array<string | null | undefined>) {
  return values.find(Boolean) || '';
}

function resolveRouteTitleKey(matchedRoute: ReturnType<typeof findRouteByPath>, pathname: string) {
  return pickText(
    matchedRoute?.resolveTitleKey?.(pathname),
    matchedRoute?.titleKey,
    systemRouteTitleMap[pathname],
  );
}

function resolveShellAccess(options: {
  hasDashboardEntry: boolean;
  hasPerm: (permission: string) => boolean;
  isAdmin: boolean;
}) {
  const { hasDashboardEntry, hasPerm, isAdmin } = options;
  return {
    canAccessDashboard: isAdmin || hasDashboardEntry || hasPerm('platform:dashboard:view'),
    canViewNoticeSummary:
      isAdmin ||
      hasDashboardEntry ||
      hasPerm('platform:dashboard:view') ||
      hasPerm('system:login-log:list') ||
      hasPerm('system:session:list') ||
      hasPerm('system:security-event:list') ||
      hasPerm('system:operation-log:list'),
  };
}

function resolveCurrentLanguage(language: string): SupportedLocale {
  return (
    SUPPORTED_LOCALES.includes(language as SupportedLocale) ? language : 'zh-CN'
  ) as SupportedLocale;
}

function resolveSessionIdleMinutes(value: number) {
  return value > 0 ? value : 30;
}

function resolveLayoutLabels(
  t: TranslateLabel,
  isHorizontalLayout: boolean,
  densityMode: ShellDensityMode,
) {
  return {
    layoutModeLabel: t(
      isHorizontalLayout ? 'app.layoutMode.horizontal' : 'app.layoutMode.vertical',
    ),
    layoutModeActionLabel: t(
      isHorizontalLayout ? 'app.layoutMode.switchToVertical' : 'app.layoutMode.switchToHorizontal',
    ),
    densityModeLabel: t(
      densityMode === 'compact' ? 'app.density.compact' : 'app.density.comfortable',
    ),
  };
}

function resolveActiveTheme<T>(items: T[], activeItem: T | undefined) {
  return activeItem ?? items[0];
}

function resolveRefreshPollingToken(enabled: boolean, token: string | null) {
  return enabled ? token : null;
}

function hasNoticeAttentionState(badgeCount: number, riskGroups: NoticeRiskItem[]) {
  return badgeCount > 0 || riskGroups.length > 0;
}

function NoticePanelBody({
  loading,
  recentItems,
  riskGroups,
  statItems,
  summary,
  t,
}: Readonly<{
  loading: boolean;
  recentItems: NoticeRecentItem[];
  riskGroups: NoticeRiskItem[];
  statItems: NoticeStatItem[];
  summary: DashboardSummary | null;
  t: TranslateLabel;
}>) {
  if (loading) {
    return <div className="app-shell__notice-empty">{t('common.loading')}</div>;
  }
  if (!summary) {
    return <div className="app-shell__notice-empty">{t('app.notice.empty')}</div>;
  }
  return (
    <>
      <div className="app-shell__notice-stats">
        {statItems.map((item) => (
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
        <span className="app-shell__notice-summary-label">{t('app.notice.lastSuccess')}</span>
        <span className="app-shell__notice-summary-value">
          {summary.lastSuccessfulLoginAt
            ? formatDateTime(summary.lastSuccessfulLoginAt, { withSeconds: true })
            : t('dashboard.lastSuccessfulLoginEmpty')}
        </span>
      </div>
      {riskGroups.length > 0 ? (
        <div className="app-shell__notice-list">
          <div className="app-shell__notice-section">{t('app.notice.section.risk')}</div>
          {riskGroups.map((item) => (
            <button
              key={item.key}
              type="button"
              className={['app-shell__notice-risk', `app-shell__notice-risk--${item.tone}`].join(
                ' ',
              )}
              onClick={item.run}
            >
              <span className="app-shell__notice-risk-copy">
                <span className="app-shell__notice-risk-title">{item.title}</span>
                <span className="app-shell__notice-risk-desc">{item.description}</span>
              </span>
              <span className="app-shell__notice-risk-value">{item.value}</span>
            </button>
          ))}
        </div>
      ) : null}
      <div className="app-shell__notice-list">
        <div className="app-shell__notice-section">{t('app.notice.section.recent')}</div>
        {recentItems.length > 0 ? (
          recentItems.map((item) => (
            <div key={item.id} className="app-shell__notice-log">
              <span
                className={`app-shell__notice-log-dot ${item.status === 1 ? 'app-shell__notice-log-dot--success' : 'app-shell__notice-log-dot--danger'}`}
              />
              <span className="app-shell__notice-log-copy">
                <span className="app-shell__notice-log-title">{item.username}</span>
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
  );
}

function ShellHeaderLeading({
  appName,
  brandInitial,
  collapsed,
  isMobile,
  isVerticalLayout,
  onToggleCollapsed,
  onToggleMobileNav,
  siteLogo,
  t,
}: Readonly<{
  appName: string;
  brandInitial: string;
  collapsed: boolean;
  isMobile: boolean;
  isVerticalLayout: boolean;
  onToggleCollapsed: () => void;
  onToggleMobileNav: () => void;
  siteLogo?: string;
  t: TranslateLabel;
}>) {
  if (isVerticalLayout) {
    return (
      <Button
        type="text"
        className="app-shell__collapse-btn"
        aria-label={t('app.nav.toggle')}
        icon={isMobile || collapsed ? <IconMenuUnfold /> : <IconMenuFold />}
        onClick={isMobile ? onToggleMobileNav : onToggleCollapsed}
      />
    );
  }
  return (
    <div className="app-shell__header-brand" aria-label={appName}>
      <span className="app-shell__header-brand-mark">
        {siteLogo ? <img src={siteLogo} alt={appName} /> : brandInitial}
      </span>
      <span className="app-shell__header-brand-text">{appName}</span>
    </div>
  );
}

type ShellUserTriggerHandle = {
  getRootDOMNode: () => HTMLElement | null;
};

type ShellUserTriggerProps = Readonly<
  Omit<React.ComponentProps<typeof Button>, 'children'> & {
    avatar?: string;
    expanded: boolean;
    label: string;
    roleLabel: string;
    showRoleLabel: boolean;
    userDisplayName: string;
  }
>;

const ShellUserTrigger = forwardRef<ShellUserTriggerHandle, ShellUserTriggerProps>(
  function ShellUserTrigger(
    {
      avatar,
      className,
      expanded,
      label,
      roleLabel,
      showRoleLabel,
      userDisplayName,
      ...buttonProps
    },
    ref,
  ) {
    const buttonRef = useRef<HTMLElement | { getRootDOMNode?: () => HTMLElement | null } | null>(
      null,
    );
    useImperativeHandle(
      ref,
      () => ({
        getRootDOMNode: () => {
          if (buttonRef.current instanceof HTMLElement) {
            return buttonRef.current;
          }
          return buttonRef.current?.getRootDOMNode?.() || null;
        },
      }),
      [],
    );

    return (
      <Button
        {...buttonProps}
        ref={buttonRef}
        type="text"
        className={['app-shell__user-trigger', className].filter(Boolean).join(' ')}
        aria-expanded={expanded}
        aria-haspopup="menu"
        aria-label={label}
      >
        <Avatar size={28}>
          <UserAvatarContent avatar={avatar} userDisplayName={userDisplayName} />
        </Avatar>
        <div className="app-shell__user-meta">
          <span className="app-shell__user-name">{userDisplayName}</span>
          {showRoleLabel ? <span className="app-shell__user-subtitle">{roleLabel}</span> : null}
        </div>
      </Button>
    );
  },
);

function ShellNoticeCenter({
  entries,
  hasAttention,
  loading,
  noticeBadgeCount,
  recentItems,
  riskGroups,
  show,
  statItems,
  summary,
  t,
}: Readonly<{
  entries: NoticeEntry[];
  hasAttention: boolean;
  loading: boolean;
  noticeBadgeCount: number;
  recentItems: NoticeRecentItem[];
  riskGroups: NoticeRiskItem[];
  show: boolean;
  statItems: NoticeStatItem[];
  summary: DashboardSummary | null;
  t: TranslateLabel;
}>) {
  if (!show) {
    return null;
  }
  return (
    <Dropdown
      trigger="click"
      position="br"
      triggerProps={{ autoFitPosition: true }}
      droplist={
        <div className="app-shell__notice-panel">
          <div className="app-shell__notice-header">
            <span className="app-shell__notice-title">{t('app.notice.title')}</span>
            <span className="app-shell__notice-subtitle">{t('app.notice.subtitle')}</span>
          </div>
          <NoticePanelBody
            loading={loading}
            recentItems={recentItems}
            riskGroups={riskGroups}
            statItems={statItems}
            summary={summary}
            t={t}
          />
          {entries.length > 0 ? (
            <div className="app-shell__notice-list">
              <div className="app-shell__notice-section">{t('app.notice.section.recommended')}</div>
              {entries.map((item) => (
                <button
                  key={item.key}
                  type="button"
                  className="app-shell__notice-item"
                  onClick={item.run}
                >
                  <span className="app-shell__notice-item-icon">{item.icon}</span>
                  <span className="app-shell__notice-item-copy">
                    <span className="app-shell__notice-item-title">{item.title}</span>
                    <span className="app-shell__notice-item-desc">{item.description}</span>
                  </span>
                </button>
              ))}
            </div>
          ) : null}
        </div>
      }
    >
      <Tooltip content={hasAttention ? t('app.notice.attention') : t('app.notice.title')}>
        <Button
          type="text"
          className={['app-shell__icon-btn', hasAttention ? 'app-shell__icon-btn--attention' : '']
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
  );
}

function CommandResults({
  items,
  onExecute,
  t,
}: Readonly<{
  items: CommandSearchItem[];
  onExecute: (item: CommandSearchItem) => void;
  t: TranslateLabel;
}>) {
  if (items.length === 0) {
    return <Empty description={t('app.command.empty')} />;
  }
  return items.map((item, index) => {
    const previousItem = items[index - 1];
    const showSection = previousItem?.section !== item.section;
    return (
      <React.Fragment key={item.key}>
        {showSection ? <div className="app-command__section">{item.section}</div> : null}
        <button type="button" className="app-command__item" onClick={() => onExecute(item)}>
          <span className="app-command__item-icon">{item.icon}</span>
          <span className="app-command__item-copy">
            <span className="app-command__item-title">{item.title}</span>
            <span className="app-command__item-subtitle">{item.subtitle}</span>
          </span>
        </button>
      </React.Fragment>
    );
  });
}

const BaseLayout: React.FC = () => {
  const bootstrappedRef = useRef(false);
  const [collapsed, setCollapsed] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [layoutMode, setLayoutMode] = useState<ShellLayoutMode>(() => readShellLayoutMode());
  const [densityMode, setDensityMode] = useState<ShellDensityMode>(() => readShellDensityMode());
  const [openedTabs, setOpenedTabs] = useState<OpenedPageTab[]>(() => readOpenedTabs());
  const [commandVisible, setCommandVisible] = useState(false);
  const [userMenuVisible, setUserMenuVisible] = useState(false);
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
  const handleMenuNavigation = useCallback(
    (key: string) => {
      const selected = findMenuNodeByPath(menuTree, key);
      if (selected?.isExternal === 1) {
        globalThis.open(selected.path, '_blank', 'noopener,noreferrer');
        return;
      }
      navigate(key);
      setMobileNavOpen(false);
    },
    [menuTree, navigate],
  );
  const currentRouteTitleKey = resolveRouteTitleKey(matchedRoute, location.pathname);
  const activeMenuPath = pickText(matchedRoute?.activeMenu, location.pathname);
  const {
    visibleMenuTree,
    currentMenuTitleKey,
    selectedMenuPath,
    breadcrumbItems,
    menuOpenKeys,
    renderedMenuItems,
  } = useLayoutMenu({
    menuTree,
    orgEnabled: publicSettings.orgEnabled,
    currentPath: location.pathname,
    activeMenuPath,
    routeActiveMenu: matchedRoute?.activeMenu,
    currentRouteTitleKey,
    t,
    handleMenuNavigation,
  });
  const currentPageTitle = pickText(breadcrumbItems.at(-1)?.label, t('app.workspace'));
  const currentTabTitleKey = pickText(currentRouteTitleKey, currentMenuTitleKey);
  const userDisplayName = pickText(userInfo?.nickname, userInfo?.username, t('common.user'));
  const roleLabel = pickText(userInfo?.roles?.[0]);
  const showRoleLabel = shouldShowIdentityLabel(userDisplayName, roleLabel);
  const isHorizontalLayout = layoutMode === 'horizontal';
  const isVerticalLayout = !isHorizontalLayout;
  const { layoutModeLabel, layoutModeActionLabel, densityModeLabel } = resolveLayoutLabels(
    t,
    isHorizontalLayout,
    densityMode,
  );
  const appName = pickText(publicSettings.siteName, t('app.name'));
  const brandInitial = getBrandInitial(appName);
  const showExpandedBrand = !collapsed;
  const { theme, setTheme, options: themeOptions } = useTheme();
  const { colorMode, setColorMode } = usePantheonColorMode();
  const activeTheme = resolveActiveTheme(
    themeOptions,
    themeOptions.find((item) => item.key === theme),
  );
  const sessionIdleMinutes = resolveSessionIdleMinutes(publicSettings.sessionIdleMinutes);
  const sessionIdleMs = sessionIdleMinutes * 60 * 1000;
  const backgroundNetworkEnabled = shouldPollServerRefreshState();
  const hasDashboardEntry = useMemo(
    () => Boolean(findMenuNodeByPath(visibleMenuTree, '/dashboard')),
    [visibleMenuTree],
  );
  const { canAccessDashboard, canViewNoticeSummary } = resolveShellAccess({
    hasDashboardEntry,
    hasPerm,
    isAdmin,
  });

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
        await resolveSilently(logoutApi());
      }
      await resolveSilently(refreshPublicSettings());
      clearPantheonThemePreference();
      const nextLanguage = clearExplicitLanguagePreference();
      await resolveSilently(switchI18nLanguage(nextLanguage));
      clearShellSessionState();
      clearClientAuthSession();
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
      if (
        !shouldReportShellActivity() ||
        !syncRemote ||
        now - lastSyncedActivityAtRef.current < 60000
      ) {
        return;
      }
      lastSyncedActivityAtRef.current = now;
      runSilently(reportActivity());
    },
    [locked, syncShellActivity, token],
  );

  useEffect(() => {
    if (bootstrappedRef.current) {
      return;
    }
    bootstrappedRef.current = true;

    runSilently(fetchMenuTree());
    if (!useAuthStore.getState().userInfo) {
      runSilently(ensureAuthUserInfo());
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
      if (hasExplicitLanguagePreference()) {
        runSilently(refreshPublicSettings());
      } else {
        runSilently(
          refreshPublicSettings().then((settings) => switchI18nLanguage(settings.defaultLanguage)),
        );
      }
      runSilently(fetchMenuTree({ force: true }));
    },
  );
  useRefreshPolling(resolveRefreshPollingToken(backgroundNetworkEnabled, token), [
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
    const applySummary = (data: DashboardSummary | null) => {
      if (active) {
        setNoticeSummary(data);
        setNoticeLoading(false);
      }
    };
    const loadNoticeSummary = async () => {
      const allowed = shouldLoadShellNoticeSummary() && Boolean(token) && canViewNoticeSummary;
      if (!allowed) {
        applySummary(null);
        return;
      }
      setNoticeLoading(true);
      applySummary(await fetchNoticeSummarySafely());
    };
    runSilently(loadNoticeSummary());
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
    const nextTab = buildOpenedPageTab(location.pathname, currentPageTitle, {
      titleKey: currentTabTitleKey,
    });

    const timer = globalThis.setTimeout(() => {
      const dashboardTitle = t('dashboard.title');
      setOpenedTabs((currentTabs) =>
        mergeOpenedTabsIntoState(currentTabs, nextTab, dashboardTitle),
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
        runSilently(performLogout(true, 'session.idle_timeout'));
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

  // Mobile detection drives the overlay drawer behaviour (≤768px). The desktop
  // shell is untouched; below the breakpoint the sider becomes a slide-in drawer.
  useEffect(() => {
    if (globalThis.matchMedia === undefined) {
      return undefined;
    }
    const query = globalThis.matchMedia('(max-width: 768px)');
    const apply = () => setIsMobile(query.matches);
    apply();
    const handleChange = (event: MediaQueryListEvent) => {
      setIsMobile(event.matches);
      if (!event.matches) {
        setMobileNavOpen(false);
      }
    };
    query.addEventListener('change', handleChange);
    return () => query.removeEventListener('change', handleChange);
  }, []);

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
    const userMenuActions: Record<UserMenuActionKey, () => void> = {
      profile: handleGoProfile,
      security: handleGoSecurity,
      lock: handleLockScreen,
      logout: () => runSilently(handleLogout()),
    };
    userMenuActions[key as UserMenuActionKey]?.();
  };

  const commandItems = useMemo<CommandSearchItem[]>(() => {
    const items = buildCommandItems({
      canAccessDashboard,
      visibleMenuTree,
      t,
      navigate,
      handleGoProfile,
      handleGoSecurity,
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
    return buildNoticeEntries({
      canViewNoticeSummary,
      isAdmin,
      hasPerm,
      t,
      navigate,
      handleGoSecurity,
    });
  }, [canViewNoticeSummary, handleGoSecurity, hasPerm, isAdmin, navigate, t]);

  const noticeBadgeCount = useMemo(() => {
    if (!noticeSummary) {
      return 0;
    }
    return Math.min(noticeSummary.loginFailureCount + noticeSummary.pendingSecurityEventCount, 99);
  }, [noticeSummary]);

  const noticeStatItems = useMemo(() => buildNoticeStatItems(noticeSummary, t), [noticeSummary, t]);

  const noticeRecentItems = useMemo(
    () => buildNoticeRecentItems(noticeSummary, t),
    [noticeSummary, t],
  );

  const noticeRiskGroups = useMemo<NoticeRiskItem[]>(
    () =>
      buildNoticeRiskGroups({
        summary: noticeSummary,
        isAdmin,
        hasPerm,
        t,
        navigate,
      }),
    [hasPerm, isAdmin, navigate, noticeSummary, t],
  );
  const hasNoticeAttention = hasNoticeAttentionState(noticeBadgeCount, noticeRiskGroups);
  const showNoticeCenter = canViewNoticeSummary;

  const applyTabUpdate = (update: TabUpdate) => {
    if (!update) {
      return;
    }
    localStorage.setItem(OPENED_TABS_STORAGE_KEY, JSON.stringify(update.tabs));
    setOpenedTabs(update.tabs);
    if (update.navigateTo) {
      navigate(update.navigateTo);
    }
  };

  const closeTab = (targetPath: string) =>
    applyTabUpdate(
      computeTabClose(openedTabs, targetPath, location.pathname, t('dashboard.title')),
    );

  const closeOtherTabs = (targetPath: string) =>
    applyTabUpdate(computeTabCloseOthers(openedTabs, targetPath, location.pathname));

  const closeTabsToRight = (targetPath: string) =>
    applyTabUpdate(computeTabCloseToRight(openedTabs, targetPath, location.pathname));

  const closeAllTabs = () =>
    applyTabUpdate(computeTabCloseAll(openedTabs, location.pathname, t('dashboard.title')));

  const togglePinTab = (targetPath: string) =>
    applyTabUpdate(computeTabTogglePin(openedTabs, targetPath));

  const moveTab = (dragPath: string, targetPath: string) =>
    applyTabUpdate(computeTabMove(openedTabs, dragPath, targetPath));

  const handleTabAction = (targetPath: string, action: TabActionKey) => {
    const tabActionHandlers: Record<TabActionKey, (path: string) => void> = {
      togglePin: togglePinTab,
      close: closeTab,
      closeOthers: closeOtherTabs,
      closeRight: closeTabsToRight,
      closeAll: closeAllTabs,
    };
    tabActionHandlers[action]?.(targetPath);
  };

  const openedTabsContent = (
    <LayoutOpenedTabs
      enabled={publicSettings.enableTabBar}
      layoutMode={layoutMode}
      tabs={openedTabs}
      currentPath={location.pathname}
      onNavigate={navigate}
      onMoveTab={moveTab}
      onTabAction={handleTabAction}
      t={t}
    />
  );

  const currentLanguage = resolveCurrentLanguage(i18n.language);

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

      updateCurrentUserPreferences(mergedPreferences)
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
        runSilently(switchI18nLanguage(preferences.language));
      }
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [currentLanguage, densityMode, layoutMode, setTheme, theme, userInfo?.preferences]);

  const changeLanguage = (language: SupportedLocale) => {
    if (language === i18n.language) {
      return;
    }
    setExplicitLanguagePreference(language);
    runSilently(switchI18nLanguage(language));
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
        <span className="app-shell__preference-section-title">
          {t('app.preference.appearance')}
        </span>
        <div className="app-shell__preference-pills">
          {(['light', 'dark'] as PantheonColorMode[]).map((mode) => (
            <button
              key={mode}
              type="button"
              className={[
                'app-shell__preference-pill',
                colorMode === mode ? 'app-shell__preference-pill--active' : '',
              ]
                .join(' ')
                .trim()}
              onClick={() => setColorMode(mode)}
            >
              <span>{t(`app.appearance.${mode}`)}</span>
              <span>{t(`app.appearance.${mode}.description`)}</span>
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

  return (
    <Layout
      className={[
        'app-shell',
        isHorizontalLayout ? 'app-shell--horizontal' : 'app-shell--vertical',
        isMobile && mobileNavOpen ? 'app-shell--mobile-open' : '',
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {isVerticalLayout ? (
        <LayoutSideMenu
          appName={appName}
          brandInitial={brandInitial}
          collapsed={isMobile ? false : collapsed}
          isHorizontalLayout={false}
          loading={loading}
          menuOpenKeys={menuOpenKeys}
          renderedMenuItems={renderedMenuItems}
          selectedMenuPath={selectedMenuPath}
          showExpandedBrand={showExpandedBrand}
          siteLogo={publicSettings.siteLogo}
          onCollapse={setCollapsed}
          onMenuItemClick={handleMenuNavigation}
        />
      ) : null}
      {isVerticalLayout && isMobile && mobileNavOpen ? (
        <button
          type="button"
          className="app-shell__mobile-backdrop"
          aria-label={t('common.close')}
          onClick={() => setMobileNavOpen(false)}
        />
      ) : null}
      <Layout className="app-shell__main">
        <Header className="app-shell__header">
          <div className="app-shell__header-left">
            <ShellHeaderLeading
              appName={appName}
              brandInitial={brandInitial}
              collapsed={collapsed}
              isMobile={isMobile}
              isVerticalLayout={isVerticalLayout}
              onToggleCollapsed={() => setCollapsed((value) => !value)}
              onToggleMobileNav={() => setMobileNavOpen((value) => !value)}
              siteLogo={publicSettings.siteLogo}
              t={t}
            />
            <div className="app-shell__header-meta">
              <LayoutBreadcrumb items={breadcrumbItems} />
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
            <ShellNoticeCenter
              entries={noticeEntries}
              hasAttention={hasNoticeAttention}
              loading={noticeLoading}
              noticeBadgeCount={noticeBadgeCount}
              recentItems={noticeRecentItems}
              riskGroups={noticeRiskGroups}
              show={showNoticeCenter}
              statItems={noticeStatItems}
              summary={noticeSummary}
              t={t}
            />
            <Dropdown
              trigger="click"
              position="br"
              triggerProps={{ autoFitPosition: true }}
              droplist={preferencePanel}
            >
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
              trigger={['hover', 'click']}
              position="br"
              popupVisible={userMenuVisible}
              onVisibleChange={setUserMenuVisible}
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
              <ShellUserTrigger
                avatar={userInfo?.avatar}
                expanded={userMenuVisible}
                label={`${t('common.user')}: ${userDisplayName}`}
                roleLabel={roleLabel}
                showRoleLabel={showRoleLabel}
                userDisplayName={userDisplayName}
              />
            </Dropdown>
          </Space>
        </Header>
        {isHorizontalLayout ? (
          <LayoutSideMenu
            appName={appName}
            brandInitial={brandInitial}
            collapsed={collapsed}
            isHorizontalLayout
            loading={loading}
            menuOpenKeys={menuOpenKeys}
            renderedMenuItems={renderedMenuItems}
            selectedMenuPath={selectedMenuPath}
            showExpandedBrand={showExpandedBrand}
            siteLogo={publicSettings.siteLogo}
            onCollapse={setCollapsed}
            onMenuItemClick={handleMenuNavigation}
          />
        ) : null}
        {openedTabsContent}
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
              runSilently(handleUnlock());
            }}
          />
          <Space>
            <Button
              type="primary"
              loading={unlockLoading}
              onClick={() => {
                runSilently(handleUnlock());
              }}
            >
              {t('app.lock.unlock')}
            </Button>
            <Button
              onClick={() => {
                runSilently(handleLogout());
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
          <CommandResults items={filteredCommandItems} onExecute={executeCommand} t={t} />
        </div>
      </AppModal>
    </Layout>
  );
};

export default BaseLayout;
