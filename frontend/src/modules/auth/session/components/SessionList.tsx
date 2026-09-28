import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Button, Popconfirm, Select, Space, Tag, Typography } from '@arco-design/web-react';
import { message } from '../../../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../../components/table/crossPageSelection';
import { formatDateTime } from '../../../../core/format/dateTime';
import { useAuthStore } from '../../../../store/useAuthStore';
import { usePermission } from '../../../../hooks/usePermission';
import {
  batchRevokeAdminSessions,
  cleanupAdminSessions,
  getAdminSessionList,
  getSessions,
  revokeAdminSession,
  type AdminSessionPageResp,
  type AdminSessionQuery,
  type AdminSessionRow,
} from '../api';
import {
  AppTable,
  buildStandardPagination,
  SearchToolbar,
  GovernanceCleanupBar,
  buildGovernanceCleanupLabels,
  GovernanceListSummary,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  TABLE_ACTION_COLUMN_WIDTH,
  TimeRangeFilter,
  type GovernanceCleanupPayload,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../../components';
import { formatClientSummary } from '../clientInfo';
import SessionDetailModal from './SessionDetailModal';
import { getSettingGroup, type SettingGroup } from '../../../system/setting/api';
import { loadRetentionSetting } from '../../../system/audit/retentionSetting';
import '../../../system/components/shared/list-page.css';
import '../../auth.css';

const defaultRetentionOptions = [1, 7, 30];

const emptyQuery: AdminSessionQuery = {
  keyword: '',
  username: '',
  lastIp: '',
  browser: undefined,
  os: undefined,
  device: undefined,
  status: undefined,
  page: 1,
  pageSize: 10,
};

interface LoadDataOptions {
  silent?: boolean;
}

const SessionList: React.FC = () => {
  const { t } = useTranslation();
  const { userInfo } = useAuthStore();
  const { isAdmin, hasPerm } = usePermission();
  const canDelete = isAdmin || hasPerm('system:session:delete');
  const canClear = isAdmin || hasPerm('system:session:clear');
  const governanceRail = useGovernanceRail();
  const cleanupLabels = useMemo(() => buildGovernanceCleanupLabels(t), [t]);
  const [data, setData] = useState<AdminSessionRow[]>([]);
  const [total, setTotal] = useState(0);
  const [activeCount, setActiveCount] = useState(0);
  const [revokedCount, setRevokedCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [query, setQuery] = useState<AdminSessionQuery>(emptyQuery);
  const [detailSession, setDetailSession] = useState<AdminSessionRow | null>(null);
  const [currentSessionId, setCurrentSessionId] = useState<string | null>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );

  useEffect(() => {
    let active = true;
    const timer = globalThis.setTimeout(() => {
      getSessions()
        .then((sessions) => {
          if (active) {
            setCurrentSessionId(sessions.find((session) => session.isCurrent)?.sessionId ?? null);
          }
        })
        .catch(() => {
          if (active) {
            setCurrentSessionId(null);
          }
        });
    }, 0);
    return () => {
      active = false;
      globalThis.clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group: SettingGroup) =>
          loadRetentionSetting(
            group,
            'audit.session_cleanup_retention_options',
            setRetentionOptions,
            setRetentionDays,
          ),
        )
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  const loadData = useCallback(
    async (nextQuery: AdminSessionQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setLoadError(null);
      }
      try {
        const result: AdminSessionPageResp = await getAdminSessionList(nextQuery);
        setData(result.items);
        setTotal(result.total);
        setActiveCount(result.activeCount);
        setRevokedCount(result.revokedCount);
      } catch (requestError) {
        setLoadError(requestError);
        message.error(t('common.loadFailed'));
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [query, t],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  const search = (values: Partial<AdminSessionQuery>) => {
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const reset = () => {
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const handleTableChange: TableProps<AdminSessionRow>['onChange'] = (pagination) => {
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
    });
  };

  const removeSession = async (row: AdminSessionRow) => {
    try {
      await revokeAdminSession(row.sessionId);
      message.success(t('auth.session.revokeSuccess'));
      setSelectedRowKeys((current) => current.filter((item) => item !== row.sessionId));
      await loadData(query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const handleBatchRevoke = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    try {
      const resp = await batchRevokeAdminSessions({ sessionIds: selectedRowKeys });
      message.success(t('auth.session.batchRevokeSuccess', { count: resp.revokedCount }));
      setSelectedRowKeys([]);
      await loadData(query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const clearHistoricSessions = async (payload: GovernanceCleanupPayload) => {
    try {
      const resp =
        payload.mode === 'range'
          ? await cleanupAdminSessions({
              startedAt: payload.startedAt,
              endedAt: payload.endedAt,
            })
          : await cleanupAdminSessions({ retentionDays: payload.retentionDays });
      message.success(t('auth.session.cleanupSuccess', { count: resp.clearedCount }));
      await loadData(query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const currentUsername = userInfo?.username;
  const browserOptions = ['Chrome', 'Edge', 'Firefox', 'Safari', 'Opera', 'WeChat', 'Unknown'];
  const osOptions = ['Windows', 'macOS', 'Linux', 'Android', 'iOS', 'Unknown'];
  const deviceOptions = [
    'Desktop',
    'iPhone',
    'iPad',
    'Android Phone',
    'Android Tablet',
    'Mobile',
    'Unknown',
  ];
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('auth.session.hero.totalLabel'),
        value: total,
        hint: t('auth.session.hero.totalHint'),
      },
      {
        key: 'active',
        label: t('auth.session.status.active'),
        value: activeCount,
        hint: t('auth.session.hero.activeHint'),
      },
      {
        key: 'revoked',
        label: t('auth.session.status.revoked'),
        value: revokedCount,
        hint: t('auth.session.hero.revokedHint'),
      },
      {
        key: 'self',
        label: t('auth.session.hero.currentUser'),
        value: currentUsername || '-',
        hint: t('auth.session.selfProtected'),
      },
    ],
    [activeCount, currentUsername, revokedCount, t, total],
  );
  const visibleSelectedRowKeys = useMemo(
    () =>
      getVisibleSelectedRowKeys(
        selectedRowKeys,
        data.map((item) => item.sessionId),
      ),
    [data, selectedRowKeys],
  );

  const columns: ColumnProps<AdminSessionRow>[] = [
    {
      title: t('system.user.username'),
      dataIndex: 'username',
      width: 168,
      render: (value: string, row: AdminSessionRow) => {
        const isCurrentSession = currentSessionId
          ? row.sessionId === currentSessionId
          : currentSessionId === null && value === currentUsername && !row.revokedAt;
        return (
          <Space direction="vertical" size={4}>
            <span style={{ whiteSpace: 'nowrap' }}>{value}</span>
            {isCurrentSession ? <Tag color="arcoblue">{t('auth.session.currentUser')}</Tag> : null}
          </Space>
        );
      },
    },
    withTableColumnPriority(
      {
        title: t('system.profile.nickname'),
        dataIndex: 'nickname',
        width: 160,
        render: (value: string) => <span style={{ whiteSpace: 'nowrap' }}>{value || '-'}</span>,
      },
      'low',
    ),
    {
      title: t('auth.session.ip'),
      dataIndex: 'lastIp',
      width: 128,
      render: (value: string) => <span style={{ whiteSpace: 'nowrap' }}>{value || '-'}</span>,
    },
    {
      title: t('auth.session.userAgent'),
      dataIndex: 'device',
      width: 260,
      render: (_: unknown, row: AdminSessionRow) => (
        <Space direction="vertical" size={2}>
          <span className="auth-device-summary">{formatClientSummary(row)}</span>
          {row.userAgent ? (
            <span className="auth-device-summary__meta">{row.userAgent}</span>
          ) : null}
        </Space>
      ),
    },
    withTableColumnPriority(
      {
        title: t('auth.session.lastActive'),
        dataIndex: 'lastActivityAt',
        width: 150,
        render: (_: unknown, row: AdminSessionRow) => (
          <Typography.Text className="system-list__datetime-text">
            {formatDateTime(row.lastActivityAt || row.lastRefreshAt, { withSeconds: true })}
          </Typography.Text>
        ),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('auth.session.refreshExpiresAt'),
        dataIndex: 'refreshExpiresAt',
        width: 160,
        render: (value: string) => (
          <Typography.Text className="system-list__datetime-text">
            {formatDateTime(value, { withSeconds: true })}
          </Typography.Text>
        ),
      },
      'low',
    ),
    {
      title: t('auth.session.status'),
      dataIndex: 'revokedAt',
      width: 110,
      render: (value?: string) =>
        value ? (
          <Tag color="red">{t('auth.session.status.revoked')}</Tag>
        ) : (
          <Tag color="green">{t('auth.session.status.active')}</Tag>
        ),
    },
    {
      title: t('common.action'),
      dataIndex: 'action',
      width: TABLE_ACTION_COLUMN_WIDTH.medium,
      fixed: 'right',
      render: (_: unknown, row: AdminSessionRow) => {
        const isSelf = row.username === currentUsername;
        return (
          <Space size={4} className="system-list__actions">
            <Button size="small" onClick={() => setDetailSession(row)}>
              {t('common.detail')}
            </Button>
            <Popconfirm
              title={t('auth.session.revokeConfirm')}
              onOk={() => removeSession(row)}
              disabled={!canDelete || isSelf || !!row.revokedAt}
            >
              <Button
                size="small"
                status="danger"
                icon={<IconDelete />}
                disabled={!canDelete || isSelf || !!row.revokedAt}
              >
                {t('auth.session.revoke')}
              </Button>
            </Popconfirm>
          </Space>
        );
      },
    },
  ];

  const tableContent =
    data.length === 0 && !loading ? (
      <PageEmpty description={t('auth.session.empty')} />
    ) : (
      <AppTable<AdminSessionRow>
        className="system-list__table"
        rowKey="sessionId"
        data={data}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        onChange={handleTableChange}
        emptyText={t('auth.session.empty')}
        rowSelection={
          canDelete
            ? {
                type: 'checkbox',
                selectedRowKeys: visibleSelectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
                onChange: (keys) =>
                  setSelectedRowKeys(
                    (currentKeys) =>
                      mergeCrossPageSelection(
                        currentKeys,
                        keys as string[],
                        data.map((item) => item.sessionId),
                      ) as string[],
                  ),
                checkboxProps: (record: AdminSessionRow) => ({
                  disabled: record.username === currentUsername || Boolean(record.revokedAt),
                }),
              }
            : undefined
        }
        pagination={buildStandardPagination(t, {
          current: query.page || emptyQuery.page,
          pageSize: query.pageSize || emptyQuery.pageSize,
          total,
        })}
      />
    );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceListSummary
          eyebrow={t('auth.session.hero.eyebrow')}
          title={t('auth.session.hero.title')}
          description={t('auth.session.hero.desc')}
          metrics={heroStats.slice(0, 3).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          summaryTitle={t('auth.session.hero.summaryTitle')}
          rail={governanceRail}
          railToggle={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('auth.session.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <SearchToolbar
            keyword={query.keyword ?? ''}
            keywordPlaceholder={t('auth.session.search.placeholder')}
            onKeywordChange={(keyword) => search({ keyword })}
            inlineFilters={
              <>
                <Select
                  allowClear
                  placeholder={t('auth.session.filter.status')}
                  value={query.status}
                  onChange={(value) => search({ status: value })}
                >
                  <Select.Option value={1}>{t('auth.session.status.active')}</Select.Option>
                  <Select.Option value={2}>{t('auth.session.status.revoked')}</Select.Option>
                </Select>
                <TimeRangeFilter
                  value={{ startedAt: query.startedAt, endedAt: query.endedAt }}
                  onChange={(value) => search(value)}
                />
              </>
            }
            advancedFilters={
              <>
                <div className="search-toolbar__popover-field">
                  <label>{t('auth.session.browserName')}</label>
                  <Select
                    allowClear
                    placeholder={t('auth.session.browserName')}
                    value={query.browser}
                    onChange={(value) => search({ browser: value })}
                  >
                    {browserOptions.map((item) => (
                      <Select.Option key={item} value={item}>
                        {item}
                      </Select.Option>
                    ))}
                  </Select>
                </div>
                <div className="search-toolbar__popover-field">
                  <label>{t('auth.session.osName')}</label>
                  <Select
                    allowClear
                    placeholder={t('auth.session.osName')}
                    value={query.os}
                    onChange={(value) => search({ os: value })}
                  >
                    {osOptions.map((item) => (
                      <Select.Option key={item} value={item}>
                        {item}
                      </Select.Option>
                    ))}
                  </Select>
                </div>
                <div className="search-toolbar__popover-field">
                  <label>{t('auth.session.deviceName')}</label>
                  <Select
                    allowClear
                    placeholder={t('auth.session.deviceName')}
                    value={query.device}
                    onChange={(value) => search({ device: value })}
                  >
                    {deviceOptions.map((item) => (
                      <Select.Option key={item} value={item}>
                        {item}
                      </Select.Option>
                    ))}
                  </Select>
                </div>
              </>
            }
            advancedActiveCount={
              [query.browser, query.os, query.device].filter(
                (value) => value !== undefined && value !== '',
              ).length
            }
            hasActiveFilters={Boolean(
              query.keyword ||
              query.status !== undefined ||
              query.browser ||
              query.os ||
              query.device ||
              query.startedAt,
            )}
            onClearAll={reset}
          />

          <Card className="page-panel system-list__table-card">
            {canDelete || canClear ? (
              <GovernanceCleanupBar
                showCleanup={canClear}
                retentionDays={retentionDays}
                retentionOptions={retentionOptions}
                onRetentionChange={setRetentionDays}
                retentionLabel={cleanupLabels.retentionLabel}
                confirmTitle={cleanupLabels.confirmTitle}
                actionLabel={t('auth.session.cleanupAction')}
                confirmActionLabel={cleanupLabels.confirmActionLabel}
                cleanupModeLabel={cleanupLabels.cleanupModeLabel}
                cleanupModeOptions={cleanupLabels.cleanupModeOptions}
                rangeStartLabel={cleanupLabels.rangeStartLabel}
                rangeEndLabel={cleanupLabels.rangeEndLabel}
                rangeRequiredMessage={cleanupLabels.rangeRequiredMessage}
                onConfirm={clearHistoricSessions}
                hint={t('auth.session.cleanupHint')}
                extraActions={
                  canDelete ? (
                    <>
                      <Typography.Text type="secondary">
                        {t('common.selectedCount', { count: selectedRowKeys.length })}
                      </Typography.Text>
                      <Button
                        type="text"
                        size="small"
                        disabled={selectedRowKeys.length === 0}
                        onClick={() => {
                          if (selectedRowKeys.length === 0) {
                            return;
                          }
                          setSelectedRowKeys([]);
                          message.success(t('common.clearSelectionSuccess'));
                        }}
                      >
                        {t('common.clearSelection')}
                      </Button>
                      <Popconfirm
                        disabled={selectedRowKeys.length === 0 || !canDelete}
                        title={t('auth.session.batchRevokeConfirm', {
                          count: selectedRowKeys.length,
                        })}
                        onOk={() => {
                          void handleBatchRevoke();
                        }}
                      >
                        <Button
                          status="danger"
                          icon={<IconDelete />}
                          disabled={selectedRowKeys.length === 0 || !canDelete}
                        >
                          {t('auth.session.revokeSelected')}
                        </Button>
                      </Popconfirm>
                    </>
                  ) : undefined
                }
              />
            ) : null}
            {loading && data.length === 0 ? <PageLoading /> : null}
            {loadError && !loading ? (
              <PageRequestError
                error={loadError}
                onRetry={() => {
                  void loadData(query);
                }}
              />
            ) : (
              tableContent
            )}
          </Card>
        </>
      </Space>
      <GovernanceInsightDrawer
        title={t('auth.session.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('auth.security.sessionHint')}
        noteDescription={t('auth.session.hero.sideDesc')}
        noteTone="warning"
      >
        <GovernanceRailSummary
          items={[
            {
              label: t('auth.session.hero.currentUser'),
              value: currentUsername || '-',
              description: t('auth.session.selfProtected'),
            },
            {
              label: t('auth.session.status.active'),
              value: activeCount,
              description: t('auth.session.hero.activeHint'),
            },
            {
              tone: 'warning',
              label: t('auth.session.status.revoked'),
              value: revokedCount,
              description: t('auth.session.hero.revokedHint'),
            },
          ]}
        />
      </GovernanceInsightDrawer>
      <SessionDetailModal
        visible={Boolean(detailSession)}
        session={detailSession}
        onCancel={() => setDetailSession(null)}
      />
    </PageContainer>
  );
};

export default SessionList;
