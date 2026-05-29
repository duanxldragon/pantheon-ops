import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Card,
  Button,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconDownload, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { getSettingGroup } from '../system/setting/api';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../components/table/crossPageSelection';
import { formatDateTime } from '../../core/format/dateTime';
import {
  batchDeleteAdminLoginLogs,
  cleanupAdminLoginLogs,
  exportAdminLoginLogs,
  exportSelectedAdminLoginLogs,
  getAdminLoginLogList,
  type LoginLogPageResp,
  type LoginLogQuery,
  type LoginLogRow,
} from './api';
import { renderClientInfo } from './clientInfo';
import {
  AppTable,
  buildStandardPagination,
  FilterPanel,
  type GovernanceCleanupMode,
  GovernanceCleanupBar,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PermissionAction,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
} from '../../components';
import { usePermission } from '../../hooks/usePermission';
import './auth.css';
import '../system/list-page.css';
import { toCleanupTimestamp, normalizeRetentionOptions, loadRetentionSetting } from '../system/audit/retentionSetting';
const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

const emptyQuery: LoginLogQuery = {
  username: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};
const defaultRetentionOptions = [1, 7, 30];

const LoginLogList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canExport = isAdmin || hasPerm('system:login-log:export');
  const canClear = isAdmin || hasPerm('system:login-log:clear');
  const canDelete = isAdmin || hasPerm('system:login-log:delete');
  const governanceRail = useGovernanceRail();
  const [data, setData] = useState<LoginLogRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [query, setQuery] = useState<LoginLogQuery>(emptyQuery);
  const [queryForm] = Form.useForm<LoginLogQuery>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [cleanupMode, setCleanupMode] = useState<GovernanceCleanupMode>('retention');
  const [cleanupRangeStart, setCleanupRangeStart] = useState('');
  const [cleanupRangeEnd, setCleanupRangeEnd] = useState('');
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );

  const loadData = useCallback(
    async (nextQuery: LoginLogQuery = query) => {
      setLoading(true);
      setLoadFailed(false);
      try {
        const result: LoginLogPageResp = await getAdminLoginLogList(nextQuery);
        setData(result.items);
        setTotal(result.total);
      } catch {
        setLoadFailed(true);
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


  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group: any) => loadRetentionSetting(group, 'audit.login_log_retention_options', setRetentionOptions, setRetentionDays))
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  const search = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const handleTableChange: TableProps<LoginLogRow>['onChange'] = (pagination) => {
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
    });
  };

  const handleCleanup = async () => {
    if (cleanupMode === 'range' && (!cleanupRangeStart || !cleanupRangeEnd)) {
      message.warning(t('common.cleanupRangeRequired'));
      return;
    }
    try {
      const resp = await cleanupAdminLoginLogs(
        cleanupMode === 'range'
          ? {
              startedAt: toCleanupTimestamp(cleanupRangeStart),
              endedAt: toCleanupTimestamp(cleanupRangeEnd),
            }
          : { retentionDays },
      );
      message.success(t('auth.loginLog.cleanupSuccess', { count: resp.clearedCount }));
      void loadData();
    } catch {
      message.error(t('common.actionFailed'));
    }
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

  const translateLogMessage = (value?: string | null) => {
    if (!value) {
      return '-';
    }
    return t(value, { defaultValue: value });
  };

  const successCount = data.filter((item) => item.status === 1).length;
  const failedCount = data.filter((item) => item.status !== 1).length;
  const visibleSelectedRowKeys = useMemo(
    () => getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.id)),
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
      {
        key: 'export',
        label: t('auth.loginLog.hero.exportReady'),
        value: canExport ? t('common.yes') : t('common.no'),
        hint: t('auth.loginLog.hero.exportHint'),
      },
      {
        key: 'cleanup',
        label: t('auth.loginLog.hero.cleanupReady'),
        value: canClear ? t('common.yes') : t('common.no'),
        hint: t('auth.loginLog.hero.cleanupHint'),
      },
    ],
    [canClear, canExport, failedCount, successCount, t, total],
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
      render: (value: string) => formatDateTime(value),
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
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16} className="auth-filter-grid auth-login-log-page__filter-grid">
                <Col xs={24} md={12} lg={6}>
                  <FormItem label={t('system.user.username')} field="username">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={4}>
                  <FormItem label={t('auth.loginLog.status')} field="status">
                    <Select
                      allowClear
                      options={[
                        { label: t('auth.loginLog.status.success'), value: 1 },
                        { label: t('auth.loginLog.status.failed'), value: 0 },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={24} lg={5}>
                  <FormItem className="filter-panel__action-item auth-login-log-page__filter-actions">
                    <Space>
                      <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                        {t('common.search')}
                      </Button>
                      <Button onClick={reset}>{t('common.reset')}</Button>
                    </Space>
                  </FormItem>
                </Col>
              </Row>
            </Form>
          </FilterPanel>

          <Card className="page-panel system-list__table-card auth-login-log-page__table-card">
            <GovernanceCleanupBar
              showCleanup={canClear}
              retentionDays={retentionDays}
              retentionOptions={retentionOptions}
              onRetentionChange={setRetentionDays}
              retentionLabel={(option) => t('common.keepRecentDays', { count: option })}
              cleanupMode={cleanupMode}
              onCleanupModeChange={setCleanupMode}
              cleanupModeLabel={t('common.cleanupMode')}
              cleanupModeOptions={[
                { label: t('common.cleanupModeRetention'), value: 'retention' },
                { label: t('common.cleanupModeRange'), value: 'range' },
              ]}
              rangeStart={cleanupRangeStart}
              rangeEnd={cleanupRangeEnd}
              onRangeStartChange={setCleanupRangeStart}
              onRangeEndChange={setCleanupRangeEnd}
              rangeStartLabel={t('common.cleanupRangeStart')}
              rangeEndLabel={t('common.cleanupRangeEnd')}
              confirmTitle={
                cleanupMode === 'range'
                  ? t('common.cleanupRangeConfirm')
                  : t('auth.loginLog.cleanupConfirm', { count: retentionDays })
              }
              actionLabel={t('common.cleanupLogs')}
              onConfirm={() => {
                void handleCleanup();
              }}
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
                    <PermissionAction
                      allowed={canDelete}
                      tooltip={t('common.noPermissionAction')}
                    >
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
            {loadFailed && !loading ? (
              <PageError
                onRetry={() => {
                  void loadData(query);
                }}
              />
            ) : data.length === 0 && !loading ? (
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
                          setSelectedRowKeys((currentKeys) =>
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
