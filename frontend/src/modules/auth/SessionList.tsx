import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Button, Form, Grid, Input, Popconfirm, Select, Space, Tag } from '@arco-design/web-react';
import { message } from '../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { getSettingGroup } from '../system/setting/api';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../components/table/crossPageSelection';
import { formatDateTime } from '../../core/format/dateTime';
import { useAuthStore } from '../../store/useAuthStore';
import { usePermission } from '../../hooks/usePermission';
import {
  batchRevokeAdminSessions,
  cleanupAdminSessions,
  getAdminSessionList,
  revokeAdminSession,
  type AdminSessionPageResp,
  type AdminSessionQuery,
  type AdminSessionRow,
} from './api';
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
  TABLE_ACTION_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../components';
import { formatClientSummary } from './clientInfo';
import SessionDetailModal from './SessionDetailModal';
import '../system/list-page.css';
import './auth.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

const emptyQuery: AdminSessionQuery = {
  username: '',
  lastIp: '',
  browser: undefined,
  os: undefined,
  device: undefined,
  status: undefined,
  page: 1,
  pageSize: 10,
};
const defaultRetentionOptions = [1, 7, 30];

function toCleanupTimestamp(value: string) {
  const normalized = String(value || '').trim();
  const match = normalized.match(
    /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?$/,
  );
  if (!match) {
    return undefined;
  }
  const [, year, month, day, hour, minute, second = '00'] = match;
  const localDate = new Date(
    Number(year),
    Number(month) - 1,
    Number(day),
    Number(hour),
    Number(minute),
    Number(second),
  );
  if (Number.isNaN(localDate.getTime())) {
    return undefined;
  }
  const offsetMinutes = -localDate.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? '+' : '-';
  const offsetHours = `${Math.floor(Math.abs(offsetMinutes) / 60)}`.padStart(2, '0');
  const offsetRemainMinutes = `${Math.abs(offsetMinutes) % 60}`.padStart(2, '0');
  return `${year}-${month}-${day}T${hour}:${minute}:${second}${sign}${offsetHours}:${offsetRemainMinutes}`;
}

function normalizeRetentionOptions(rawValue: string | undefined) {
  if (!rawValue) {
    return defaultRetentionOptions;
  }
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    if (!Array.isArray(parsed)) {
      return defaultRetentionOptions;
    }
    const normalized = Array.from(
      new Set(
        parsed.map((item) => Number(item)).filter((item) => Number.isInteger(item) && item > 0),
      ),
    ).sort((left, right) => right - left);
    return normalized.length > 0 ? normalized : defaultRetentionOptions;
  } catch {
    return defaultRetentionOptions;
  }
}

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
  const [data, setData] = useState<AdminSessionRow[]>([]);
  const [total, setTotal] = useState(0);
  const [activeCount, setActiveCount] = useState(0);
  const [revokedCount, setRevokedCount] = useState(0);
  const [loading, setLoading] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);
  const [query, setQuery] = useState<AdminSessionQuery>(emptyQuery);
  const [detailSession, setDetailSession] = useState<AdminSessionRow | null>(null);
  const [queryForm] = Form.useForm<AdminSessionQuery>();
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [cleanupMode, setCleanupMode] = useState<GovernanceCleanupMode>('retention');
  const [cleanupRangeStart, setCleanupRangeStart] = useState('');
  const [cleanupRangeEnd, setCleanupRangeEnd] = useState('');
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );

  const loadData = useCallback(
    async (nextQuery: AdminSessionQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setLoadFailed(false);
      }
      try {
        const result: AdminSessionPageResp = await getAdminSessionList(nextQuery);
        setData(result.items);
        setTotal(result.total);
        setActiveCount(result.activeCount);
        setRevokedCount(result.revokedCount);
      } catch {
        setLoadFailed(true);
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

  const isSessionRetentionSetting = (item) => item.settingKey === 'audit.session_cleanup_retention_options';

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group) => {
          const setting = group.items.find(
            isSessionRetentionSetting,
          );
          const nextOptions = normalizeRetentionOptions(setting?.settingValue);
          setRetentionOptions(nextOptions);
          setRetentionDays((current) => (nextOptions.includes(current) ? current : nextOptions[0]));
        })
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

  const clearHistoricSessions = async () => {
    if (cleanupMode === 'range' && (!cleanupRangeStart || !cleanupRangeEnd)) {
      message.warning(t('common.cleanupRangeRequired'));
      return;
    }
    try {
      const resp = await cleanupAdminSessions(
        cleanupMode === 'range'
          ? {
              startedAt: toCleanupTimestamp(cleanupRangeStart),
              endedAt: toCleanupTimestamp(cleanupRangeEnd),
            }
          : { retentionDays },
      );
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
    () => getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.sessionId)),
    [data, selectedRowKeys],
  );

  const columns: ColumnProps<AdminSessionRow>[] = [
    {
      title: t('system.user.username'),
      dataIndex: 'username',
      width: 168,
      render: (value: string) => (
        <Space direction="vertical" size={4}>
          <span style={{ whiteSpace: 'nowrap' }}>{value}</span>
          {value === currentUsername ? (
            <Tag color="arcoblue">{t('auth.session.currentUser')}</Tag>
          ) : null}
        </Space>
      ),
    },
    {
      title: t('system.profile.nickname'),
      dataIndex: 'nickname',
      width: 160,
      render: (value: string) => <span style={{ whiteSpace: 'nowrap' }}>{value || '-'}</span>,
    },
    {
      title: t('auth.session.ip'),
      dataIndex: 'lastIp',
      width: 128,
      render: (value: string) => <span style={{ whiteSpace: 'nowrap' }}>{value || '-'}</span>,
    },
    withTableColumnPriority(
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
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('auth.session.lastActive'),
        dataIndex: 'lastActivityAt',
        width: 150,
        render: (_: unknown, row: AdminSessionRow) =>
          formatDateTime(row.lastActivityAt || row.lastRefreshAt),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('auth.session.refreshExpiresAt'),
        dataIndex: 'refreshExpiresAt',
        width: 160,
        render: (value: string) => formatDateTime(value),
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

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('auth.session.hero.eyebrow')}
          title={t('auth.session.hero.title')}
          description={t('auth.session.hero.desc')}
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
              {t('auth.session.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16} className="auth-filter-grid">
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('system.user.username')} field="username">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('auth.session.ip')} field="lastIp">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('auth.session.filter.status')} field="status">
                    <Select allowClear>
                      <Select.Option value={1}>{t('auth.session.status.active')}</Select.Option>
                      <Select.Option value={2}>{t('auth.session.status.revoked')}</Select.Option>
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('auth.session.browserName')} field="browser">
                    <Select allowClear>
                      {browserOptions.map((item) => (
                        <Select.Option key={item} value={item}>
                          {item}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('auth.session.osName')} field="os">
                    <Select allowClear>
                      {osOptions.map((item) => (
                        <Select.Option key={item} value={item}>
                          {item}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('auth.session.deviceName')} field="device">
                    <Select allowClear>
                      {deviceOptions.map((item) => (
                        <Select.Option key={item} value={item}>
                          {item}
                        </Select.Option>
                      ))}
                    </Select>
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem className="filter-panel__action-item">
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

          <Card className="page-panel system-list__table-card">
            {(canClear || canDelete) ? (
              <div>
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
                      : t('auth.session.cleanupConfirm', { count: retentionDays })
                  }
                  actionLabel={t('auth.session.cleanupAction')}
                  onConfirm={() => {
                    void clearHistoricSessions();
                  }}
                  hint={t('auth.session.cleanupHint')}
                  extraActions={
                    <>
                      <span className="table-batch-action-bar__summary">
                        {t('common.selectedCount', { count: selectedRowKeys.length })}
                      </span>
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
                  }
                />
              </div>
            ) : null}
            {loading && data.length === 0 ? <PageLoading /> : null}
            {loadFailed && !loading ? (
              <PageError
                onRetry={() => {
                  void loadData(query);
                }}
              />
            ) : data.length === 0 && !loading ? (
              <PageEmpty description={t('auth.session.empty')} />
            ) : (
              <AppTable<AdminSessionRow>
                className="system-list__table"
                rowKey="sessionId"
                data={data}
                columns={columns}
                loading={loading}
                scroll={{ x: 1600 }}
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
                          setSelectedRowKeys((currentKeys) =>
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
              label: t('auth.session.currentUser'),
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
