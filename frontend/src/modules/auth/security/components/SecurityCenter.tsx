import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../../components/feedback/message';
import { IconEye, IconLock } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../../../core/format/dateTime';
import { resolveRouteWarmData } from '../../../../core/router/prefetch';
import {
  getOwnLoginLogs,
  getSecurityOverview,
  getSessions,
  revokeSession,
  updatePassword,
  type AuthSession,
  type LoginLogRow,
  type SecurityPolicy,
  type SecurityOverview,
  type UserPasswordUpdatePayload,
} from '../api';
import { formatClientSummary, renderClientInfo } from '../../session/clientInfo';
import { useAuthStore } from '../../../../store/useAuthStore';
import {
  AppTable,
  buildStandardPagination,
  DateTimeMeta,
  FormSection,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PageSplitLayout,
  StandardRailNotePanel,
  StandardRailSummary,
  SubmitBar,
  getPagedItems,
  TABLE_ACTION_COLUMN_WIDTH,
} from '../../../../components';
import SessionDetailModal from '../../session/components/SessionDetailModal';
import '../../../system/components/shared/list-page.css';
import '../../auth.css';

const FormItem = Form.Item;
const Row = Grid.Row;
const Col = Grid.Col;

type TranslateFn = (key: string, options?: Record<string, unknown>) => string;

const renderActivityTime = (value?: string | null) => {
  if (!value) {
    return '-';
  }
  return <DateTimeMeta value={value} />;
};

// 纯函数抽离：安全策略展示项构造不依赖组件状态，降低组件体认知复杂度。
function buildSecurityPolicyItems(
  t: TranslateFn,
  policy: SecurityPolicy | undefined,
): Array<{ label: string; value: string; hint: string }> {
  if (!policy) {
    return [];
  }
  const yesNo = (enabled: boolean) => (enabled ? t('common.yes') : t('common.no'));
  return [
    {
      label: t('system.setting.item.security.password_min_length'),
      value: t('auth.security.policy.passwordMinLength', { count: policy.passwordMinLength }),
      hint: t('system.setting.remark.security.password_min_length'),
    },
    {
      label: t('system.setting.item.security.password_require_digit'),
      value: yesNo(policy.passwordRequireDigit),
      hint: t('system.setting.remark.security.password_require_digit'),
    },
    {
      label: t('system.setting.item.security.password_require_uppercase'),
      value: yesNo(policy.passwordRequireUpper),
      hint: t('system.setting.remark.security.password_require_uppercase'),
    },
    {
      label: t('system.setting.item.security.password_history_limit'),
      value: String(policy.passwordHistoryLimit ?? 0),
      hint: t('system.setting.remark.security.password_history_limit'),
    },
    {
      label: t('system.setting.item.security.password_expire_days'),
      value: String(policy.passwordExpireDays ?? 0),
      hint: t('system.setting.remark.security.password_expire_days'),
    },
    {
      label: t('system.setting.item.login.max_failed_attempts'),
      value: t('auth.security.policy.maxFailedAttempts', { count: policy.maxFailedAttempts }),
      hint: t('system.setting.remark.login.max_failed_attempts'),
    },
    {
      label: t('system.setting.item.login.lock_minutes'),
      value: t('auth.security.policy.lockMinutes', { count: policy.lockMinutes }),
      hint: t('system.setting.remark.login.lock_minutes'),
    },
    {
      label: t('system.setting.item.login.source_max_failed_attempts'),
      value: t('auth.security.policy.sourceMaxFailedAttempts', {
        count: policy.sourceMaxFailedAttempts,
      }),
      hint: t('system.setting.remark.login.source_max_failed_attempts'),
    },
    {
      label: t('system.setting.item.login.source_window_minutes'),
      value: t('auth.security.policy.sourceWindowMinutes', { count: policy.sourceWindowMinutes }),
      hint: t('system.setting.remark.login.source_window_minutes'),
    },
    {
      label: t('system.setting.item.login.source_lock_minutes'),
      value: t('auth.security.policy.sourceLockMinutes', { count: policy.sourceLockMinutes }),
      hint: t('system.setting.remark.login.source_lock_minutes'),
    },
    {
      label: t('system.setting.item.login.session_idle_minutes'),
      value: t('auth.security.policy.sessionIdleMinutes', { count: policy.sessionIdleMinutes }),
      hint: t('system.setting.remark.login.session_idle_minutes'),
    },
    {
      label: t('system.setting.item.login.max_active_sessions_per_user'),
      value: t('auth.security.policy.maxActiveSessions', { count: policy.maxActiveSessions }),
      hint: t('system.setting.remark.login.max_active_sessions_per_user'),
    },
    {
      label: t('system.setting.item.audit.session_retention_days'),
      value: t('auth.security.policy.sessionRetentionDays', {
        count: policy.sessionRetentionDays,
      }),
      hint: t('system.setting.remark.audit.session_retention_days'),
    },
    {
      label: t('system.setting.item.login.captcha_enabled'),
      value: yesNo(policy.captchaEnabled),
      hint: t('system.setting.remark.login.captcha_enabled'),
    },
    {
      label: t('system.setting.item.login.mfa_enabled'),
      value: yesNo(policy.mfaEnabled),
      hint: t('system.setting.remark.login.mfa_enabled'),
    },
    {
      label: t('system.setting.item.login.sso_enabled'),
      value: yesNo(policy.ssoEnabled),
      hint: t('system.setting.remark.login.sso_enabled'),
    },
  ];
}

function buildSessionColumns(options: {
  t: TranslateFn;
  revokingSessionId: string | null;
  onDetail: (session: AuthSession) => void;
  onRevoke: (sessionId: string) => Promise<void>;
}): ColumnProps<AuthSession>[] {
  const { t, revokingSessionId, onDetail, onRevoke } = options;
  return [
    {
      title: t('auth.session.current'),
      dataIndex: 'isCurrent',
      render: (_: unknown, record) =>
        record.isCurrent ? (
          <Tag color="arcoblue">{t('auth.session.currentDevice')}</Tag>
        ) : (
          <Tag>{t('auth.session.otherDevice')}</Tag>
        ),
    },
    {
      title: t('auth.session.ip'),
      dataIndex: 'lastIp',
      render: (value) => value || '-',
    },
    {
      title: t('auth.session.userAgent'),
      dataIndex: 'device',
      render: (_: unknown, record) => (
        <Typography.Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 320 }}>
          {formatClientSummary(record)}
        </Typography.Text>
      ),
    },
    {
      title: t('auth.session.lastActive'),
      dataIndex: 'lastRefreshAt',
      render: (value: string | undefined, record) => renderActivityTime(value || record.createdAt),
    },
    {
      title: t('auth.session.refreshExpiresAt'),
      dataIndex: 'refreshExpiresAt',
      render: (value) => (
        <Typography.Text className="system-list__datetime-text">
          {formatDateTime(value, { withSeconds: true })}
        </Typography.Text>
      ),
    },
    {
      title: t('system.profile.createdAt'),
      dataIndex: 'createdAt',
      render: (value) => (
        <Typography.Text className="system-list__datetime-text">
          {formatDateTime(value, { withSeconds: true })}
        </Typography.Text>
      ),
    },
    {
      title: t('common.action'),
      dataIndex: 'action',
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      render: (_: unknown, record) => (
        <Space size={4} className="system-list__actions">
          <Button type="text" icon={<IconEye />} onClick={() => onDetail(record)}>
            {t('common.detail')}
          </Button>
          <Popconfirm
            title={t('auth.session.revokeConfirm')}
            onOk={() => onRevoke(record.sessionId)}
            disabled={record.isCurrent}
          >
            <Button
              type="text"
              status="danger"
              disabled={record.isCurrent}
              loading={revokingSessionId === record.sessionId}
            >
              {record.isCurrent ? t('auth.session.current') : t('auth.session.revoke')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];
}

function buildLoginLogColumns(
  t: TranslateFn,
  translateLogMessage: (value?: string | null) => string,
): ColumnProps<LoginLogRow>[] {
  return [
    {
      title: t('auth.loginLog.loginTime'),
      dataIndex: 'loginTime',
      render: (value) => renderActivityTime(value),
    },
    {
      title: t('auth.loginLog.ip'),
      dataIndex: 'ipaddr',
      render: (value) => value || '-',
    },
    {
      title: t('auth.loginLog.browser'),
      dataIndex: 'browser',
      render: (_: unknown, record) => renderClientInfo(record),
    },
    {
      title: t('auth.loginLog.status'),
      dataIndex: 'status',
      render: (value) =>
        value === 1 ? (
          <Tag color="green">{t('auth.loginLog.status.success')}</Tag>
        ) : (
          <Tag color="red">{t('auth.loginLog.status.failed')}</Tag>
        ),
    },
    {
      title: t('auth.loginLog.failureReason'),
      dataIndex: 'msg',
      ellipsis: true,
      render: (value) => translateLogMessage(value),
    },
  ];
}

interface SecurityRailProps {
  currentSession: AuthSession | null;
  loading: boolean;
  overview: SecurityOverview | null;
  policyItems: Array<{ label: string; value: string; hint: string }>;
  t: TranslateFn;
  userInfo: ReturnType<typeof useAuthStore.getState>['userInfo'];
}

function SecurityRail({
  currentSession,
  loading,
  overview,
  policyItems,
  t,
  userInfo,
}: Readonly<SecurityRailProps>) {
  return (
    <>
      {loading && !overview ? (
        <PageLoading />
      ) : (
        <StandardRailSummary
          title={t('auth.security.overview')}
          items={[
            {
              label: t('system.profile.username'),
              value: overview?.user?.username || userInfo?.username || '-',
              description: overview?.user?.nickname || userInfo?.nickname || '-',
            },
            {
              label: t('auth.security.currentDevice'),
              value: formatClientSummary(currentSession),
              description: currentSession?.lastIp || '-',
            },
            {
              label: t('auth.session.lastActive'),
              value: currentSession
                ? renderActivityTime(currentSession.lastRefreshAt || currentSession.createdAt)
                : '-',
            },
          ]}
        />
      )}
      {policyItems.length === 0 && loading ? <PageLoading /> : null}
      {policyItems.length > 0 ? (
        <StandardRailSummary
          title={t('auth.security.policy')}
          items={policyItems.map((item) => ({
            label: item.label,
            value: item.value,
            description: item.hint,
          }))}
        />
      ) : null}
      {overview?.recentSecurityEvents?.length ? (
        <StandardRailSummary
          title={t('system.menu.securityEvent')}
          items={overview.recentSecurityEvents.slice(0, 4).map((item) => ({
            label: t(`auth.securityEvent.type.${item.eventType}`, {
              defaultValue: item.eventType,
            }),
            value: t(`auth.securityEvent.severity.${item.severity}`, {
              defaultValue: item.severity,
            }),
            description: formatDateTime(item.createdAt, { withSeconds: true }),
          }))}
        />
      ) : null}
      <StandardRailNotePanel
        title={t('auth.security.hero.sideTitle')}
        noteTitle={t('auth.security.currentSessionSummary')}
        noteDescription={t('auth.security.hero.sideDesc')}
      />
    </>
  );
}

const SecurityCenter: React.FC = () => {
  const { t } = useTranslation();
  const { userInfo, setUserInfo } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);
  const [revokingSessionId, setRevokingSessionId] = useState<string | null>(null);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [overview, setOverview] = useState<SecurityOverview | null>(null);
  const [sessions, setSessions] = useState<AuthSession[]>([]);
  const [loginLogs, setLoginLogs] = useState<LoginLogRow[]>([]);
  const [detailSession, setDetailSession] = useState<AuthSession | null>(null);
  const [sessionPagination, setSessionPagination] = useState({ current: 1, pageSize: 5 });
  const [passwordForm] = Form.useForm<UserPasswordUpdatePayload & { confirmPassword: string }>();

  const loadSecurityContext = useCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const [overviewResp, sessionsResp, loginLogsResp] = await Promise.all([
        resolveRouteWarmData('/auth/security', 'overview', () => getSecurityOverview()),
        resolveRouteWarmData('/auth/security', 'sessions', () => getSessions()),
        resolveRouteWarmData('/auth/security', 'login-logs', () =>
          getOwnLoginLogs({ page: 1, pageSize: 10 }),
        ),
      ]);
      setOverview(overviewResp);
      if (overviewResp.user) {
        setUserInfo(overviewResp.user);
      }
      setSessions(sessionsResp);
      setLoginLogs(loginLogsResp.items);
    } catch (requestError) {
      setLoadError(requestError);
      message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [setUserInfo, t]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadSecurityContext();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadSecurityContext]);

  const currentSession = useMemo(
    () => overview?.currentSession ?? sessions.find((item) => item.isCurrent) ?? null,
    [overview, sessions],
  );
  const { currentPage: sessionCurrentPage, pageItems: pagedSessions } = getPagedItems(
    sessions,
    sessionPagination.current,
    sessionPagination.pageSize,
  );
  const sessionPageSize = sessionPagination.pageSize;

  const translateLogMessage = useCallback(
    (value?: string | null) => {
      if (!value) {
        return '-';
      }
      return t(value, { defaultValue: value });
    },
    [t],
  );

  const handleChangePassword = async () => {
    const values = await passwordForm.validate();
    setSavingPassword(true);
    try {
      await updatePassword({
        oldPassword: values.oldPassword,
        newPassword: values.newPassword,
      });
      passwordForm.resetFields();
      message.success(t('system.profile.passwordSuccess'));
      await loadSecurityContext();
    } finally {
      setSavingPassword(false);
    }
  };

  const handleRevokeSession = async (sessionId: string) => {
    setRevokingSessionId(sessionId);
    try {
      await revokeSession(sessionId);
      message.success(t('auth.session.revokeSuccess'));
      await loadSecurityContext();
    } finally {
      setRevokingSessionId(null);
    }
  };

  const successCount = loginLogs.filter((item) => item.status === 1).length;
  const failedCount = loginLogs.filter((item) => item.status !== 1).length;
  const statItems = [
    {
      label: t('auth.security.activeSessionCount'),
      value: String(overview?.activeSessionCount ?? sessions.length),
      hint: t('auth.security.sessionHint'),
    },
    {
      label: t('auth.security.lastLoginAt'),
      value: <DateTimeMeta value={overview?.lastLoginAt} className="auth-page-stat-card__time" />,
      hint: t('auth.security.loginLogHint'),
    },
    {
      label: t('auth.security.currentIp'),
      value: currentSession?.lastIp || '-',
      hint: t('auth.security.currentSessionSummary'),
    },
    {
      label: t('auth.security.recentSuccessfulLoginCount'),
      value: String(successCount),
      hint: t('auth.security.recentWindow'),
    },
  ];

  const policyItems = useMemo<Array<{ label: string; value: string; hint: string }>>(
    () => buildSecurityPolicyItems(t, overview?.policy),
    [overview?.policy, t],
  );

  const sessionColumns = buildSessionColumns({
    t,
    revokingSessionId,
    onDetail: setDetailSession,
    onRevoke: handleRevokeSession,
  });
  const loginLogColumns = buildLoginLogColumns(t, translateLogMessage);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        {loadError && !loading && !overview ? (
          <Card className="page-panel">
            <PageRequestError
              error={loadError}
              onRetry={() => {
                void loadSecurityContext();
              }}
            />
          </Card>
        ) : null}

        <Card className="page-panel system-page-hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('auth.security.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title">
                {t('auth.security.hero.title')}
              </Typography.Title>
            </div>
          </div>
          <div className="system-page-kpi-grid">
            {statItems.map((item) => (
              <div key={item.label} className="system-page-kpi">
                <span className="system-page-kpi__label">{item.label}</span>
                <div className="system-page-kpi__value">{item.value}</div>
                <span className="system-page-kpi__hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>

        <PageSplitLayout
          rail={
            <SecurityRail
              currentSession={currentSession}
              loading={loading}
              overview={overview}
              policyItems={policyItems}
              t={t}
              userInfo={userInfo}
            />
          }
        >
          <Card
            className="page-panel"
            title={t('auth.security.password')}
            extra={
              <Tag color={overview?.passwordExpired ? 'red' : undefined}>
                {overview?.passwordExpired
                  ? t('auth.security.passwordExpired')
                  : t('auth.security.passwordTip')}
              </Tag>
            }
          >
            <Form
              form={passwordForm}
              layout="vertical"
              onSubmit={() => {
                void handleChangePassword();
              }}
            >
              <Space direction="vertical" size={20} className="auth-section-stack">
                <FormSection
                  title={t('system.profile.passwordTitle')}
                  description={t('system.profile.passwordHint')}
                >
                  <Row gutter={16} className="auth-form-grid">
                    <Col xs={24} md={12}>
                      <FormItem
                        label={t('system.profile.oldPassword')}
                        field="oldPassword"
                        rules={[
                          { required: true, message: t('system.profile.oldPasswordRequired') },
                        ]}
                      >
                        <Input.Password
                          prefix={<IconLock />}
                          onPressEnter={() => passwordForm.submit()}
                        />
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12}>
                      <FormItem
                        label={t('system.profile.newPassword')}
                        field="newPassword"
                        rules={[{ required: true, message: t('auth.passwordRequired') }]}
                      >
                        <Input.Password
                          prefix={<IconLock />}
                          onPressEnter={() => passwordForm.submit()}
                        />
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12}>
                      <FormItem
                        label={t('system.profile.confirmPassword')}
                        field="confirmPassword"
                        rules={[
                          { required: true, message: t('system.profile.confirmPasswordRequired') },
                          {
                            validator: (value, callback) => {
                              const nextPassword = passwordForm.getFieldValue('newPassword');
                              if (value && nextPassword && value !== nextPassword) {
                                callback(t('system.profile.confirmPasswordMismatch'));
                                return;
                              }
                              callback();
                            },
                          },
                        ]}
                      >
                        <Input.Password
                          prefix={<IconLock />}
                          onPressEnter={() => passwordForm.submit()}
                        />
                      </FormItem>
                    </Col>
                  </Row>
                </FormSection>
              </Space>
              <SubmitBar
                loading={savingPassword}
                onSubmit={() => {
                  void handleChangePassword();
                }}
                submitText={t('system.profile.savePassword')}
              />
            </Form>
          </Card>

          <Card
            className="page-panel"
            title={t('auth.security.sessions')}
            extra={
              <Tag color="arcoblue">{t('common.totalWithCount', { count: sessions.length })}</Tag>
            }
          >
            <Space direction="vertical" size={16} className="auth-table-stack">
              <div className="auth-inline-note">
                <div className="auth-inline-note__copy">
                  <span className="auth-inline-note__title">{t('auth.session.currentDevice')}</span>
                  <span className="auth-inline-note__desc">
                    {formatClientSummary(currentSession)}
                  </span>
                </div>
                <Tag>{t('auth.security.sessionHint')}</Tag>
              </div>
              {sessions.length === 0 && !loadError ? (
                <PageEmpty description={t('auth.session.empty')} />
              ) : (
                <AppTable<AuthSession>
                  rowKey="sessionId"
                  columns={sessionColumns}
                  data={pagedSessions}
                  loading={loading && Boolean(overview)}
                  pagination={buildStandardPagination(t, {
                    current: sessionCurrentPage,
                    pageSize: sessionPageSize,
                    total: sessions.length,
                    sizeCanChange: false,
                    sizeOptions: [5],
                    onChange: (page, pageSize) => {
                      setSessionPagination({
                        current: page,
                        pageSize: pageSize || sessionPageSize,
                      });
                    },
                  })}
                  scroll={{ x: 'max-content' }}
                  emptyText={t('auth.session.empty')}
                />
              )}
            </Space>
          </Card>

          <Card
            className="page-panel"
            title={t('auth.security.loginLogs')}
            extra={
              <Tag color={failedCount > 0 ? 'orange' : 'green'}>
                {t('common.totalWithCount', { count: loginLogs.length })}
              </Tag>
            }
          >
            <Space direction="vertical" size={16} className="auth-table-stack">
              <div className="auth-inline-note">
                <div className="auth-inline-note__copy">
                  <span className="auth-inline-note__title">{t('auth.security.loginLogHint')}</span>
                  <span className="auth-inline-note__desc">{t('auth.security.recentWindow')}</span>
                </div>
                <Space wrap>
                  <Tag color="green">
                    {t('auth.loginLog.status.success')}: {successCount}
                  </Tag>
                  <Tag color="red">
                    {t('auth.loginLog.status.failed')}: {failedCount}
                  </Tag>
                </Space>
              </div>
              {loginLogs.length === 0 && !loadError ? (
                <PageEmpty description={t('auth.loginLog.empty')} />
              ) : (
                <AppTable<LoginLogRow>
                  rowKey="id"
                  columns={loginLogColumns}
                  data={loginLogs}
                  loading={loading && Boolean(overview)}
                  pagination={false}
                  emptyText={t('auth.loginLog.empty')}
                />
              )}
            </Space>
          </Card>
        </PageSplitLayout>
      </Space>
      <SessionDetailModal
        visible={Boolean(detailSession)}
        session={detailSession}
        onCancel={() => setDetailSession(null)}
      />
    </PageContainer>
  );
};

export default SecurityCenter;
