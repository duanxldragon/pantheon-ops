import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button, Card, Popconfirm, Select, Space, Tag, Typography } from '@arco-design/web-react';
import { IconDelete, IconDownload } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { message } from '../../../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../../components/table/crossPageSelection';
import { formatDateTime } from '../../../../core/format/dateTime';
import {
  batchDeleteAdminLoginLogs,
  cleanupAdminLoginLogs,
  exportAdminLoginLogs,
  exportSelectedAdminLoginLogs,
  getAdminLoginLogList,
  type LoginLogPageResp,
  type LoginLogQuery,
  type LoginLogRow,
} from '../api';
import { renderClientInfo } from '../../session/clientInfo';
import {
  AppTable,
  buildStandardPagination,
  GovernanceCleanupBar,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SearchToolbar,
  TABLE_COLUMN_WIDTH,
  TimeRangeFilter,
  type GovernanceCleanupPayload,
  type TimeRangeFilterValue,
  useGovernanceRail,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import { getSettingGroup, type SettingGroup } from '../../../system/setting/api';
import { loadRetentionSetting } from '../../../system/audit/retentionSetting';
import '../../../system/components/shared/list-page.css';
import '../../auth.css';
const defaultRetentionOptions = [1, 7, 30];

const emptyQuery: LoginLogQuery = {
  keyword: '',
  username: '',
  status: undefined,
  startedAt: '',
  endedAt: '',
  page: 1,
  pageSize: 10,
};

const LoginLogList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canExport = isAdmin || hasPerm('system:login-log:export');
  const canDelete = isAdmin || hasPerm('system:login-log:delete');
  const canClear = isAdmin || hasPerm('system:login-log:clear');
  const governanceRail = useGovernanceRail();
  const [data, setData] = useState<LoginLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [successCount, setSuccessCount] = useState(0);
  const [failedCount, setFailedCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadError, setLoadError] = useState<unknown>(null);
  const [query, setQuery] = useState<LoginLogQuery>(emptyQuery);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group: SettingGroup) =>
          loadRetentionSetting(
            group,
            'audit.login_log_retention_options',
            setRetentionOptions,
            setRetentionDays,
          ),
        )
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  const loadData = useCallback(
    async (nextQuery: LoginLogQuery = query) => {
      setLoading(true);
      setLoadError(null);
      try {
        const result: LoginLogPageResp = await getAdminLoginLogList(nextQuery);
        setData(result.items);
        setTotal(result.total);
        setSuccessCount(result.successCount ?? 0);
        setFailedCount(result.failedCount ?? 0);
      } catch (requestError) {
        setLoadError(requestError);
        message.error(t('common.loadFailed'));
      } finally {
        setLoading(false);
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

  const reset = () => {
    setSelectedRowKeys([]);
    setQuery({ ...emptyQuery });
  };

  const handleTimeRangeChange = useCallback((value: Required<TimeRangeFilterValue>) => {
    setSelectedRowKeys([]);
    setQuery((prev) => ({ ...prev, ...value, page: 1 }));
  }, []);

  const handleTableChange: TableProps<LoginLogRow>['onChange'] = (pagination) => {
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
    });
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    try {
      const resp = await batchDeleteAdminLoginLogs({ ids: selectedRowKeys });
      message.success(t('auth.loginLog.batchDeleteSuccess', { count: resp.deletedCount }));
      setSelectedRowKeys([]);
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const handleCleanup = async (payload: GovernanceCleanupPayload) => {
    try {
      const resp =
        payload.mode === 'range'
          ? await cleanupAdminLoginLogs({
              startedAt: payload.startedAt,
              endedAt: payload.endedAt,
            })
          : await cleanupAdminLoginLogs({ retentionDays: payload.retentionDays });
      message.success(t('auth.loginLog.cleanupSuccess', { count: resp.clearedCount }));
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const translateLogMessage = (value?: string | null) => {
    if (!value) {
      return '-';
    }
    return t(value, { defaultValue: value });
  };

  const visibleSelectedRowKeys = useMemo(
    () =>
      getVisibleSelectedRowKeys(
        selectedRowKeys,
        data.map((item) => item.id),
      ),
    [data, selectedRowKeys],
  );
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('auth.security.loginLogs'),
        value: total,
        hint: t('auth.loginLog.hero.totalHint'),
      },
      {
        key: 'success',
        label: t('auth.loginLog.status.success'),
        value: successCount,
        hint: t('auth.loginLog.hero.successHint'),
      },
      {
        key: 'failed',
        label: t('auth.loginLog.status.failed'),
        value: failedCount,
        hint: t('auth.loginLog.hero.failedHint'),
      },
    ],
    [failedCount, successCount, t, total],
  );

  const columns: ColumnProps<LoginLogRow>[] = [
    {
      title: t('system.user.username'),
      dataIndex: 'username',
      width: TABLE_COLUMN_WIDTH.identity,
    },
    { title: t('auth.loginLog.ip'), dataIndex: 'ipaddr', width: TABLE_COLUMN_WIDTH.identity },
    {
      title: t('auth.loginLog.location'),
      dataIndex: 'loginLocation',
      width: TABLE_COLUMN_WIDTH.location,
      ellipsis: true,
      // Backend stores a stable i18n key (location.*); legacy rows keep raw text,
      // which passes through via defaultValue.
      render: (value: string) => (value ? t(value, { defaultValue: value }) : '-'),
    },
    {
      title: t('auth.loginLog.browser'),
      dataIndex: 'browser',
      width: TABLE_COLUMN_WIDTH.diagnostics,
      render: (_: unknown, record: LoginLogRow) => renderClientInfo(record),
    },
    {
      title: t('auth.loginLog.status'),
      dataIndex: 'status',
      width: TABLE_COLUMN_WIDTH.status,
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
      width: TABLE_COLUMN_WIDTH.diagnostics,
      ellipsis: true,
      render: (value: string) => translateLogMessage(value),
    },
    {
      title: t('auth.loginLog.loginTime'),
      dataIndex: 'loginTime',
      width: TABLE_COLUMN_WIDTH.datetime,
      render: (value: string) => formatDateTime(value, { withSeconds: true }),
    },
  ];

  const handleExport = async () => {
    if (selectedRowKeys.length > 0) {
      const selectedRows = data.filter((item) => selectedRowKeys.includes(item.id));
      if (selectedRows.length !== selectedRowKeys.length) {
        message.warning(
          t('common.exportCurrentPageSelectionOnly', {
            defaultValue: '已选记录包含跨页项，请切回对应页面后再导出。',
          }),
        );
        return;
      }
      exportSelectedAdminLoginLogs(selectedRows);
      return;
    }
    await exportAdminLoginLogs(query);
  };

  const tableContent =
    data.length === 0 && !loading ? (
      <PageEmpty description={t('auth.loginLog.empty')} />
    ) : (
      <AppTable<LoginLogRow>
        className="system-list__table"
        rowKey="id"
        data={data}
        columns={columns}
        loading={loading}
        scroll={{ x: 'max-content' }}
        onChange={handleTableChange}
        emptyText={t('auth.loginLog.empty')}
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
                        keys as number[],
                        data.map((item) => item.id),
                      ) as number[],
                  ),
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
      <Space direction="vertical" size={16} className="system-page-template auth-login-log-page">
        <GovernanceSummaryBar
          className="auth-login-log-page__hero"
          eyebrow={t('auth.loginLog.hero.eyebrow')}
          title={t('auth.loginLog.hero.title')}
          description={t('auth.loginLog.hero.desc')}
          metrics={heroStats.slice(0, 3).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('auth.loginLog.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <SearchToolbar
            keyword={query.keyword ?? ''}
            keywordPlaceholder={t('auth.loginLog.search.placeholder')}
            onKeywordChange={(keyword) => {
              setSelectedRowKeys([]);
              setQuery((prev) => ({ ...prev, keyword, page: 1 }));
            }}
            inlineFilters={
              <>
                <Select
                  allowClear
                  placeholder={t('auth.loginLog.status')}
                  value={query.status}
                  onChange={(value) => {
                    setSelectedRowKeys([]);
                    setQuery((prev) => ({ ...prev, status: value, page: 1 }));
                  }}
                  options={[
                    { label: t('auth.loginLog.status.success'), value: 1 },
                    { label: t('auth.loginLog.status.failed'), value: 0 },
                  ]}
                />
                <TimeRangeFilter
                  value={{ startedAt: query.startedAt, endedAt: query.endedAt }}
                  onChange={handleTimeRangeChange}
                />
              </>
            }
            hasActiveFilters={Boolean(
              query.keyword || query.status !== undefined || query.startedAt,
            )}
            onClearAll={reset}
          />

          <Card className="page-panel system-list__table-card auth-login-log-page__table-card">
            <GovernanceCleanupBar
              showCleanup={canClear}
              retentionDays={retentionDays}
              retentionOptions={retentionOptions}
              onRetentionChange={setRetentionDays}
              retentionLabel={(option) => t('common.keepRecentDays', { count: option })}
              confirmTitle={t('common.cleanupIrreversibleWarning')}
              actionLabel={t('common.cleanupLogs')}
              confirmActionLabel={t('common.cleanup')}
              cleanupModeLabel={t('common.cleanupMode')}
              cleanupModeOptions={[
                { label: t('common.cleanupModeRetention'), value: 'retention' },
                { label: t('common.cleanupModeRange'), value: 'range' },
              ]}
              rangeStartLabel={t('common.cleanupRangeStart')}
              rangeEndLabel={t('common.cleanupRangeEnd')}
              rangeRequiredMessage={t('common.cleanupRangeRequired')}
              onConfirm={handleCleanup}
              hint={t('auth.loginLog.hero.cleanupHint')}
              trailing={
                <Button
                  icon={<IconDownload />}
                  onClick={() => {
                    void handleExport();
                  }}
                  disabled={!canExport}
                >
                  {t('common.export')}
                </Button>
              }
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
                    <PermissionAction allowed={canDelete} tooltip={t('common.noPermissionAction')}>
                      <Popconfirm
                        disabled={selectedRowKeys.length === 0 || !canDelete}
                        title={t('auth.loginLog.batchDeleteConfirm', {
                          count: selectedRowKeys.length,
                        })}
                        onOk={() => {
                          void handleBatchDelete();
                        }}
                      >
                        <Button
                          status="danger"
                          icon={<IconDelete />}
                          disabled={selectedRowKeys.length === 0 || !canDelete}
                        >
                          {t('common.deleteSelected')}
                        </Button>
                      </Popconfirm>
                    </PermissionAction>
                  </>
                ) : undefined
              }
            />
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
        title={t('auth.loginLog.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('auth.security.loginLogHint')}
        noteDescription={t('auth.loginLog.hero.sideDesc')}
        noteTone="warning"
      >
        <GovernanceRailSummary
          items={[
            {
              label: t('auth.loginLog.status.success'),
              value: successCount,
              description: t('auth.loginLog.hero.successHint'),
            },
            {
              tone: 'warning',
              label: t('auth.loginLog.status.failed'),
              value: failedCount,
              description: t('auth.loginLog.hero.failedHint'),
            },
            {
              label: t('auth.loginLog.hero.window'),
              value: t('auth.loginLog.hero.windowValue'),
              description: t('auth.security.recentWindow'),
            },
            {
              label: t('common.selected'),
              value: selectedRowKeys.length,
              description: t('auth.loginLog.hero.selectedHint'),
            },
          ]}
        />
      </GovernanceInsightDrawer>
    </PageContainer>
  );
};

export default LoginLogList;
