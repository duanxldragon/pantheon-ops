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
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../api/request';
import {
  AppTable,
  DateTimeMeta,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
} from '../../components';
import { renderMenuIcon } from '../../core/menu/icon';
import { resolveRouteWarmData } from '../../core/router/prefetch';
import { usePermission } from '../../hooks/usePermission';
import { useMenuStore } from '../../store/useMenuStore';
import { getDashboardSummary, type DashboardRecentLogin, type DashboardSummary } from './api';
import {
  dashboardDomainOverviewWidgets,
  dashboardQuickActionWidgets,
  isDashboardWidgetVisible,
} from './widgets';
import './dashboard.css';

const Row = Grid.Row;
const Col = Grid.Col;

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
        icon: widget.icon,
      }));
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
          path: widget.path,
        })),
    [hasPerm, isAdmin, menuTree, summary, t],
  );

  const renderActivityTime = (value?: string | null) => {
    if (!value) {
      return '-';
    }
    return <DateTimeMeta value={value} />;
  };

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
      {
        key: 'last-login',
        tone: 'neutral',
        icon: <IconClockCircle />,
        label: t('dashboard.lastSuccessfulLogin'),
        value: summary.lastSuccessfulLoginAt
          ? renderActivityTime(summary.lastSuccessfulLoginAt)
          : t('dashboard.lastSuccessfulLoginEmpty'),
        desc: t('dashboard.subtitle'),
      },
    ];
  }, [successRate, summary, t]);

  const primaryAttentionItems = useMemo(() => attentionItems.slice(0, 4), [attentionItems]);

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError
          timeout={isTimeoutRequestError(error)}
          onRetry={() => {
            void loadSummary();
          }}
        />
      );
    }
    if (isServerRequestError(error)) {
      return (
        <PageServerError
          onRetry={() => {
            void loadSummary();
          }}
        />
      );
    }
    return (
      <PageError
        onRetry={() => {
          void loadSummary();
        }}
      />
    );
  };

  return (
    <PageContainer className="dashboard-page">
      {loading && !summary ? (
        <Card className="page-panel dashboard-panel-card">
          <PageLoading />
        </Card>
      ) : null}
      {error && !summary ? (
        <Card className="page-panel dashboard-panel-card">{renderErrorState()}</Card>
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
                <Typography.Paragraph className="dashboard-hero-card__desc">
                  {t('dashboard.subtitle')}
                </Typography.Paragraph>
              </div>
            </div>
            <Row gutter={[12, 12]}>
              {stats.map((item) => (
                <Col xs={24} sm={12} xl={6} key={item.title}>
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
            <Col xs={24} lg={15}>
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
                      <span className="dashboard-focus-item__copy">
                        <span className="dashboard-focus-item__label">{item.label}</span>
                        <span
                          className={`dashboard-focus-item__value${item.key === 'last-login' ? ' dashboard-focus-item__value--meta' : ''}`}
                        >
                          {item.value}
                        </span>
                      </span>
                      <span className="dashboard-focus-item__desc">{item.desc}</span>
                    </div>
                  ))}
                </div>
              </Card>
            </Col>
            <Col xs={24} lg={9}>
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
                        <span className="dashboard-quick-action__title">{item.title}</span>
                      </button>
                    ))}
                  </div>
                ) : (
                  <PageEmpty description={t('dashboard.emptyQuickActions')} />
                )}
              </Card>
            </Col>
          </Row>

          <Row gutter={[16, 16]}>
            <Col xs={24}>
              <Card className="page-panel dashboard-panel-card" title={t('dashboard.todoCenter')}>
                {summary.orgGovernanceTasks?.length ? (
                  <div className="dashboard-task-grid">
                    {summary.orgGovernanceTasks.map((item) => (
                      <div key={item.taskKey} className="dashboard-task-card">
                        <span className="dashboard-task-card__icon">
                          <IconExclamationCircle />
                        </span>
                        <div className="dashboard-task-card__body">
                          <span className="dashboard-task-card__title">
                            {item.issueLabel}
                            <Tag size="small" style={{ marginLeft: 8 }}>
                              {item.scopeLabel}
                            </Tag>
                          </span>
                          <span className="dashboard-task-card__desc">{item.resourceLabel}</span>
                          <span className="dashboard-task-card__desc">
                            {item.actionLabel}
                            {item.relatedUserCount > 0
                              ? ` · ${t('dashboard.relatedUsers', { count: item.relatedUserCount })}`
                              : ''}
                          </span>
                        </div>
                        <Button
                          type="text"
                          size="small"
                          icon={<IconArrowRight />}
                          onClick={() =>
                            navigate(item.routePath, {
                              state: { deptId: item.routeStateDeptId, taskKey: item.taskKey },
                            })
                          }
                        >
                          {t('dashboard.openTask')}
                        </Button>
                      </div>
                    ))}
                  </div>
                ) : (
                  <PageEmpty description={t('dashboard.todoEmpty')} />
                )}
              </Card>
            </Col>
          </Row>

          <Card className="page-panel dashboard-panel-card" title={t('dashboard.domainOverview')}>
            <div className="dashboard-domain-grid">
              {domainCards.map((item) => (
                <div
                  key={item.key}
                  className={`dashboard-domain-card dashboard-domain-card--${item.key}`}
                >
                  <div className="dashboard-domain-card__head">
                    <span className="dashboard-domain-card__title">{item.title}</span>
                    <Button
                      type="text"
                      size="small"
                      icon={<IconArrowRight />}
                      onClick={() => navigate(item.path)}
                    >
                      {t('dashboard.openModule')}
                    </Button>
                  </div>
                  <span className="dashboard-domain-card__summary">{item.summary}</span>
                  <span className="dashboard-domain-card__desc">{item.description}</span>
                </div>
              ))}
            </div>
          </Card>

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
              data={summary.recentLogins}
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
