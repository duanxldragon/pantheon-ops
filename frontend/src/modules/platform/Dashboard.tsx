import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Grid, Space, Statistic, Tag, Typography } from '@arco-design/web-react';
import {
  IconArrowRight,
  IconClockCircle,
  IconExclamationCircle,
  IconSafe,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  AppTable,
  DateTimeMeta,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
} from '../../components';
import { renderMenuIcon } from '../../core/menu/icon';
import { resolveRouteWarmData } from '../../core/router/prefetch';
import { usePermission } from '../../hooks/usePermission';
import { useMenuStore } from '../../store/useMenuStore';
import {
  getDashboardSummary,
  type DashboardRecentLogin,
  type DashboardSummary,
  type DashboardTodoItem,
} from './api';
import {
  dashboardDomainOverviewWidgets,
  dashboardQuickActionWidgets,
  isDashboardWidgetVisible,
} from './widgets';
import './dashboard.css';

const Row = Grid.Row;
const Col = Grid.Col;
const QUICK_ACTION_LIMIT = 6;
const QUICK_ACTION_PRIORITY = [
  'platform.users',
  'platform.roles',
  'platform.permission',
  'platform.menus',
  'platform.setting',
  'platform.security',
  'platform.dict',
  'platform.post',
  'platform.login-log',
  'platform.session',
  'platform.security-event',
  'platform.audit',
  'platform.i18n',
  'platform.module-manager',
  'platform.generator',
];
const DOMAIN_CARD_PRIORITY = [
  'platform.domain.access',
  'platform.domain.org',
  'platform.domain.config',
  'platform.domain.security',
  'platform.domain.lowcode',
  'platform.domain.governance',
];
const TODO_SCOPE_ALIASES: Record<string, 'dept' | 'post'> = {
  dept: 'dept',
  department: 'dept',
  部门: 'dept',
  post: 'post',
  岗位: 'post',
};
const TODO_ISSUE_ALIASES: Record<string, string> = {
  'clean': 'clean',
  'healthy': 'clean',
  '治理正常': 'clean',
  'leaderless': 'leaderless',
  'leader missing': 'leaderless',
  '缺负责人部门': 'leaderless',
  '缺负责人': 'leaderless',
  'no-post': 'no-post',
  'no posts': 'no-post',
  '无岗位部门': 'no-post',
  '无岗位': 'no-post',
  'empty': 'empty',
  'empty department': 'empty',
  '空部门': 'empty',
  'in-use': 'in-use',
  'assigned members': 'in-use',
  '在用岗位': 'in-use',
  'disabled': 'disabled',
  'disabled posts': 'disabled',
  '已禁用岗位': 'disabled',
};
const TODO_ACTION_ALIASES: Record<string, string> = {
  'assign-leader': 'assign-leader',
  'assign leader': 'assign-leader',
  '补负责人': 'assign-leader',
  'create-post': 'create-post',
  'create post': 'create-post',
  '补岗位': 'create-post',
  'review-merge-or-delete': 'review-merge-or-delete',
  'review merge or delete': 'review-merge-or-delete',
  '评估合并或删除': 'review-merge-or-delete',
  'clear-child-depts': 'clear-child-depts',
  'clear child departments': 'clear-child-depts',
  '清理下级部门': 'clear-child-depts',
  'clear-posts': 'clear-posts',
  'clear posts': 'clear-posts',
  '清理岗位': 'clear-posts',
  'clear-users': 'clear-users',
  'clear users': 'clear-users',
  '迁移成员': 'clear-users',
  'reassign-users': 'reassign-users',
  'reassign users': 'reassign-users',
  '调整岗位成员': 'reassign-users',
  'review-status': 'review-status',
  'review status': 'review-status',
  '复核岗位状态': 'review-status',
  'delete-or-keep-disabled': 'delete-or-keep-disabled',
  'delete or keep disabled': 'delete-or-keep-disabled',
  '删除或保留停用岗位': 'delete-or-keep-disabled',
};

function compareByPriority(leftKey: string, rightKey: string, priority: string[]) {
  const leftIndex = priority.indexOf(leftKey);
  const rightIndex = priority.indexOf(rightKey);

  if (leftIndex === -1 && rightIndex === -1) {
    return leftKey.localeCompare(rightKey);
  }
  if (leftIndex === -1) {
    return 1;
  }
  if (rightIndex === -1) {
    return -1;
  }
  return leftIndex - rightIndex;
}

function normalizeTodoAlias(value?: string | null) {
  return value?.trim().toLowerCase().replace(/\s+/g, ' ') ?? '';
}

function resolveTodoScopeCode(...values: Array<string | null | undefined>) {
  for (const value of values) {
    const normalized = normalizeTodoAlias(value);
    if (normalized && TODO_SCOPE_ALIASES[normalized]) {
      return TODO_SCOPE_ALIASES[normalized];
    }
  }
  return values.find((value): value is string => Boolean(value?.trim())) ?? '';
}

function resolveTodoIssueCode(...values: Array<string | null | undefined>) {
  for (const value of values) {
    const normalized = normalizeTodoAlias(value);
    if (normalized && TODO_ISSUE_ALIASES[normalized]) {
      return TODO_ISSUE_ALIASES[normalized];
    }
  }
  return values.find((value): value is string => Boolean(value?.trim())) ?? '';
}

function resolveTodoActionCode(...values: Array<string | null | undefined>) {
  for (const value of values) {
    const normalized = normalizeTodoAlias(value);
    if (normalized && TODO_ACTION_ALIASES[normalized]) {
      return TODO_ACTION_ALIASES[normalized];
    }
  }
  return values.find((value): value is string => Boolean(value?.trim())) ?? '';
}

function buildTodoRouteState(item: DashboardTodoItem) {
  return {
    deptId: item.routeStateDeptId,
    taskKey: item.taskKey,
  };
}

const DashboardPage: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { menuTree } = useMenuStore();
  const { hasPerm, isAdmin } = usePermission();
  const [loading, setLoading] = useState(false);
  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [error, setError] = useState<unknown>(null);

  const loadSummary = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await resolveRouteWarmData('/dashboard', 'summary', () => getDashboardSummary());
      setSummary(data);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadSummary();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadSummary]);

  const translateLogMessage = useCallback(
    (value?: string | null) => {
      if (!value) {
        return '-';
      }
      return t(value, { defaultValue: value });
    },
    [t],
  );

  const successRate = useMemo(() => {
    if (!summary) {
      return 0;
    }
    const total = summary.loginSuccessCount + summary.loginFailureCount;
    if (total === 0) {
      return 100;
    }
    return Math.round((summary.loginSuccessCount / total) * 100);
  }, [summary]);

  const stats = useMemo(
    () => [
      {
        key: 'users',
        title: t('dashboard.users'),
        value: summary?.totalUsers ?? 0,
        extra: (
          <Tag color="green">
            {t('dashboard.enabledUsers')}: {summary?.enabledUsers ?? 0}
          </Tag>
        ),
        hint: t('dashboard.metric.usersHint'),
      },
      {
        key: 'menus',
        title: t('dashboard.menus'),
        value: summary?.visibleMenuCount ?? 0,
        extra: <Tag>{t('dashboard.platformOverview')}</Tag>,
        hint: t('dashboard.metric.menusHint'),
      },
      {
        key: 'sessions',
        title: t('dashboard.sessions'),
        value: summary?.activeSessionCount ?? 0,
        extra: <Tag color="arcoblue">{t('dashboard.securityOverview')}</Tag>,
        hint: t('dashboard.metric.sessionsHint'),
      },
      {
        key: 'operations',
        title: t('dashboard.todayOperations'),
        value: summary?.todayOperationCount ?? 0,
        extra: (
          <Tag color="purple">
            <IconClockCircle />
            {t('dashboard.platformActivity')}
          </Tag>
        ),
        hint: t('dashboard.metric.todayOpsHint'),
      },
    ],
    [summary, t],
  );

  const quickActions = useMemo(() => {
    return dashboardQuickActionWidgets
      .filter((widget) => isDashboardWidgetVisible(widget, { menuTree, hasPerm, isAdmin }))
      .map((widget) => ({
        key: widget.key,
        path: widget.path,
        title: t(widget.titleKey),
        description: t(widget.descriptionKey),
        icon: widget.icon,
      }))
      .sort((left, right) => compareByPriority(left.key, right.key, QUICK_ACTION_PRIORITY))
      .slice(0, QUICK_ACTION_LIMIT);
  }, [hasPerm, isAdmin, menuTree, t]);

  const domainCards = useMemo(
    () =>
      dashboardDomainOverviewWidgets
        .filter((widget) => isDashboardWidgetVisible(widget, { menuTree, hasPerm, isAdmin }))
        .map((widget) => ({
          key: widget.key,
          title: t(widget.titleKey),
          description: t(widget.descriptionKey),
          summary: widget.summary(summary, t),
          compactSummary: compactMetricText(widget.summary(summary, t)),
          path: widget.path,
        }))
        .sort((left, right) => compareByPriority(left.key, right.key, DOMAIN_CARD_PRIORITY)),
    [hasPerm, isAdmin, menuTree, summary, t],
  );

  const renderActivityTime = (value?: string | null) => {
    if (!value) {
      return '-';
    }
    return <DateTimeMeta value={value} />;
  };

  function compactMetricText(value: string) {
    const numbers = value.match(/\d+/g);
    if (!numbers?.length) {
      return value;
    }
    return numbers.join(' / ');
  }

  const recentLoginPreview = useMemo(() => summary?.recentLogins.slice(0, 6) ?? [], [summary]);

  const translateTodoScope = useCallback(
    (scope: string, fallback?: string) => {
      if (scope === 'dept') {
        return t('dashboard.todo.scope.dept');
      }
      if (scope === 'post') {
        return t('dashboard.todo.scope.post');
      }
      return fallback || scope || '-';
    },
    [t],
  );

  const translateTodoIssue = useCallback(
    (scope: string, issue: string, fallback?: string) => {
      if (scope === 'dept') {
        if (issue === 'leaderless') {
          return t('system.dept.governance.leaderless');
        }
        if (issue === 'no-post') {
          return t('system.dept.governance.noPost');
        }
        if (issue === 'empty') {
          return t('system.dept.governance.empty');
        }
        if (issue === 'clean') {
          return t('system.dept.governance.clean');
        }
      }
      if (scope === 'post') {
        if (issue === 'in-use') {
          return t('dashboard.todo.issue.inUse');
        }
        if (issue === 'disabled') {
          return t('dashboard.todo.issue.disabled');
        }
        if (issue === 'clean') {
          return t('system.dept.governance.clean');
        }
      }
      return fallback || issue || '-';
    },
    [t],
  );

  const translateTodoAction = useCallback(
    (action: string, fallback?: string) => {
      if (action === 'assign-leader') {
        return t('dashboard.todo.action.assignLeader');
      }
      if (action === 'create-post') {
        return t('system.dept.action.createPost');
      }
      if (action === 'review-merge-or-delete') {
        return t('dashboard.todo.action.reviewMergeOrDelete');
      }
      if (action === 'clear-child-depts') {
        return t('dashboard.todo.action.clearChildDepts');
      }
      if (action === 'clear-posts') {
        return t('dashboard.todo.action.clearPosts');
      }
      if (action === 'clear-users') {
        return t('dashboard.todo.action.clearUsers');
      }
      if (action === 'reassign-users') {
        return t('dashboard.todo.action.reassignUsers');
      }
      if (action === 'review-status') {
        return t('dashboard.todo.action.reviewStatus');
      }
      if (action === 'delete-or-keep-disabled') {
        return t('dashboard.todo.action.deleteOrKeepDisabled');
      }
      return fallback || action || '-';
    },
    [t],
  );

  const todoItems = useMemo(
    () =>
      (summary?.orgGovernanceTasks ?? []).map((item) => {
        const scopeCode = resolveTodoScopeCode(item.domain, item.scopeLabel);
        const issueCode = resolveTodoIssueCode(item.issue, item.issueLabel);
        const actionCode = resolveTodoActionCode(item.action, item.actionLabel);
        return {
          ...item,
          localizedScopeLabel: translateTodoScope(scopeCode, item.scopeLabel),
          localizedIssueLabel: translateTodoIssue(scopeCode, issueCode, item.issueLabel),
          localizedActionLabel: translateTodoAction(actionCode, item.actionLabel),
        };
      }),
    [summary, translateTodoAction, translateTodoIssue, translateTodoScope],
  );

  const openTodoTask = useCallback(
    (item: DashboardTodoItem) => {
      navigate(item.routePath || '/system/dept', {
        state: buildTodoRouteState(item),
      });
    },
    [navigate],
  );

  const loginColumns = [
    {
      title: t('auth.loginLog.loginTime'),
      dataIndex: 'loginTime',
      width: 176,
      render: (value: string) => renderActivityTime(value),
    },
    {
      title: t('auth.username'),
      dataIndex: 'username',
      width: 120,
      render: (value: string) => value || '-',
    },
    {
      title: t('auth.loginLog.ip'),
      dataIndex: 'ipaddr',
      width: 140,
      render: (value: string) => value || '-',
    },
    {
      title: t('auth.loginLog.browser'),
      dataIndex: 'browser',
      width: 220,
      ellipsis: true,
      render: (value: string, record: DashboardRecentLogin) =>
        `${value || '-'} / ${record.os || '-'}`,
    },
    {
      title: t('auth.loginLog.status'),
      dataIndex: 'status',
      width: 96,
      render: (value: number) =>
        value === 1 ? (
          <Tag color="green">{t('auth.loginLog.status.success')}</Tag>
        ) : (
          <Tag color="red">{t('auth.loginLog.status.failed')}</Tag>
        ),
    },
    {
      title: t('auth.loginLog.failureReason'),
      dataIndex: 'msg',
      width: 280,
      ellipsis: true,
      render: (value: string) => translateLogMessage(value),
    },
  ];

  const attentionItems = useMemo(() => {
    if (!summary) {
      return [];
    }
    return [
      {
        key: 'failed',
        tone: summary.loginFailureCount > 0 ? 'danger' : 'neutral',
        icon: <IconExclamationCircle />,
        label: t('dashboard.loginFailureTrend', { days: summary.periodDays }),
        value: summary.loginFailureCount,
        desc: t('dashboard.attention.failedLoginDesc'),
      },
      {
        key: 'success',
        tone: successRate >= 90 ? 'success' : 'warning',
        icon: <IconSafe />,
        label: t('dashboard.securitySuccessRate', { days: summary.periodDays }),
        value: `${successRate}%`,
        desc: t('dashboard.attention.successRateDesc'),
      },
      {
        key: 'security-events',
        tone: summary.pendingSecurityEventCount > 0 ? 'warning' : 'success',
        icon: <IconExclamationCircle />,
        label: t('app.notice.pendingSecurityEvents'),
        value: summary.pendingSecurityEventCount,
        desc: t('dashboard.domain.securityDesc'),
      },
      {
        key: 'org-tasks',
        tone: summary.orgGovernanceTaskCount > 0 ? 'warning' : 'success',
        icon: <IconExclamationCircle />,
        label: t('dashboard.orgGovernanceTasks'),
        value: summary.orgGovernanceTaskCount,
        desc: t('dashboard.orgGovernanceTasksDesc'),
      },
    ];
  }, [successRate, summary, t]);
  const primaryAttentionItems = attentionItems;
  return (
    <PageContainer className="dashboard-page">
      {loading && !summary ? (
        <Card className="page-panel dashboard-panel-card">
          <PageLoading />
        </Card>
      ) : null}
      {error && !summary ? (
        <Card className="page-panel dashboard-panel-card">
          <PageRequestError
            error={error}
            onRetry={() => {
              void loadSummary();
            }}
          />
        </Card>
      ) : null}
      {summary ? (
        <Space direction="vertical" size={20} className="dashboard-grid">
          <Card className="page-panel dashboard-panel-card dashboard-hero-card">
            <div className="dashboard-hero-card__top">
              <div className="dashboard-hero-card__copy">
                <span className="dashboard-hero-card__eyebrow">{t('dashboard.statusStrip')}</span>
                <Typography.Title heading={4} className="dashboard-hero-card__title">
                  {t('dashboard.title')}
                </Typography.Title>
              </div>
            </div>
            <Row gutter={[12, 12]}>
              {stats.map((item) => (
                <Col xs={24} sm={12} xl={6} key={item.key}>
                  <div className={`dashboard-stat-card dashboard-stat-card--${item.key}`}>
                    <div className="dashboard-stat-card__title">
                      <Typography.Text>{item.title}</Typography.Text>
                      {item.extra}
                    </div>
                    <Statistic className="dashboard-stat-card__value" value={item.value} />
                    <span className="dashboard-stat-card__hint">{item.hint}</span>
                  </div>
                </Col>
              ))}
            </Row>
          </Card>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card
                className="page-panel dashboard-panel-card dashboard-panel-card--todo"
                title={t('dashboard.todoCenter')}
              >
                {todoItems.length ? (
                  <div className="dashboard-task-grid">
                    {todoItems.map((item) => (
                      <div
                        key={item.taskKey}
                        className="dashboard-task-card"
                        onClick={() => openTodoTask(item)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            openTodoTask(item);
                          }
                        }}
                        role="button"
                        tabIndex={0}
                      >
                        <span className="dashboard-task-card__icon">
                          <IconExclamationCircle />
                        </span>
                        <div className="dashboard-task-card__body">
                          <div className="dashboard-task-card__head">
                            <span className="dashboard-task-card__title">
                              {item.localizedIssueLabel}
                              <Tag size="small" style={{ marginLeft: 8 }}>
                                {item.localizedScopeLabel}
                              </Tag>
                            </span>
                            <span className="dashboard-task-card__action">
                              <Button
                                type="text"
                                size="small"
                                icon={<IconArrowRight />}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  openTodoTask(item);
                                }}
                              >
                                {t('dashboard.openTask')}
                              </Button>
                            </span>
                          </div>
                          <span className="dashboard-task-card__desc">{item.resourceLabel}</span>
                          <span className="dashboard-task-card__desc">
                            {item.localizedActionLabel}
                            {item.relatedUserCount > 0
                              ? ` · ${t('dashboard.relatedUsers', { count: item.relatedUserCount })}`
                              : ''}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  <PageEmpty description={t('dashboard.todoEmpty')} />
                )}
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card
                className="page-panel dashboard-panel-card dashboard-panel-card--attention"
                title={t('dashboard.attentionPanel')}
              >
                <div className="dashboard-focus-stack">
                  {primaryAttentionItems.map((item) => (
                    <div
                      key={item.key}
                      className={`dashboard-focus-item dashboard-focus-item--${item.tone}`}
                    >
                      <span className="dashboard-focus-item__icon">{item.icon}</span>
                      <span className="dashboard-focus-item__body">
                        <span className="dashboard-focus-item__head">
                          <span className="dashboard-focus-item__label">{item.label}</span>
                          <span
                            className={`dashboard-focus-item__value${item.key === 'last-login' ? ' dashboard-focus-item__value--meta' : ''}`}
                          >
                            {item.value}
                          </span>
                        </span>
                        <span className="dashboard-focus-item__desc">{item.desc}</span>
                      </span>
                    </div>
                  ))}
                </div>
              </Card>
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24} lg={12}>
              <Card
                className="page-panel dashboard-panel-card dashboard-panel-card--actions"
                title={t('dashboard.primaryActions')}
              >
                {quickActions.length > 0 ? (
                  <div className="dashboard-quick-actions">
                    {quickActions.map((item) => (
                      <button
                        key={item.key}
                        type="button"
                        className="dashboard-quick-action"
                        title={item.title}
                        aria-label={item.title}
                        onClick={() => navigate(item.path)}
                      >
                        <span className="dashboard-quick-action__icon">
                          {renderMenuIcon(item.icon)}
                        </span>
                        <span className="dashboard-quick-action__body">
                          <span className="dashboard-quick-action__title">{item.title}</span>
                          <span className="dashboard-quick-action__desc">{item.description}</span>
                        </span>
                        <span className="dashboard-quick-action__arrow" aria-hidden="true">
                          <IconArrowRight />
                        </span>
                      </button>
                    ))}
                  </div>
                ) : (
                  <PageEmpty description={t('dashboard.emptyQuickActions')} />
                )}
              </Card>
            </Col>
            <Col xs={24} lg={12}>
              <Card
                className="page-panel dashboard-panel-card dashboard-panel-card--domains"
                title={t('dashboard.domainOverview')}
              >
                <div className="dashboard-domain-grid">
                  {domainCards.map((item) => (
                    <div
                      key={item.key}
                      className={`dashboard-domain-card dashboard-domain-card--${item.key}`}
                    >
                      <span className="dashboard-domain-card__summary" title={item.summary}>
                        {item.compactSummary}
                      </span>
                      <div className="dashboard-domain-card__body">
                        <span className="dashboard-domain-card__title">{item.title}</span>
                        <span className="dashboard-domain-card__desc">{item.description}</span>
                      </div>
                      <Button
                        type="text"
                        size="small"
                        icon={<IconArrowRight />}
                        aria-label={t('dashboard.openModule')}
                        title={t('dashboard.openModule')}
                        onClick={() => navigate(item.path)}
                      />
                    </div>
                  ))}
                </div>
              </Card>
            </Col>
          </Row>

          <Card
            className="page-panel dashboard-panel-card dashboard-login-table"
            title={t('dashboard.recentLogins')}
            extra={
              <Button
                type="text"
                size="small"
                icon={<IconArrowRight />}
                onClick={() => navigate('/system/login-log')}
              >
                {t('dashboard.viewAllActivity')}
              </Button>
            }
          >
            <AppTable<DashboardRecentLogin>
              rowKey="id"
              loading={loading}
              columns={loginColumns}
              data={recentLoginPreview}
              pagination={false}
              scroll={{ x: 1040 }}
              emptyText={t('dashboard.recentLoginsEmpty')}
            />
          </Card>
        </Space>
      ) : null}
    </PageContainer>
  );
};

export default DashboardPage;
