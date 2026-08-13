import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, Input, Select, Space, Tag, Typography } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconCheck } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { message } from '../../../../components/feedback/message';
import { formatDateTime } from '../../../../core/format/dateTime';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../../components/table/crossPageSelection';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  SearchToolbar,
  GovernanceCleanupBar,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  TABLE_COLUMN_WIDTH,
  TimeRangeFilter,
  type GovernanceCleanupPayload,
  useGovernanceRail,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  acknowledgeSecurityEvent,
  batchAcknowledgeSecurityEvents,
  cleanupSecurityEvents,
  getAdminSecurityEventList,
  type SecurityEventPageResp,
  type SecurityEventQuery,
  type SecurityEventRow,
} from '../api';
import { getSettingGroup, type SettingGroup } from '../../../system/setting/api';
import { loadRetentionSetting } from '../../../system/audit/retentionSetting';
import '../../auth.css';
import '../../../system/components/shared/list-page.css';

const TextArea = Input.TextArea;

const defaultRetentionOptions = [1, 7, 30];

function getSeverityTagColor(severity: string) {
  if (severity === 'high') {
    return 'red';
  }
  if (severity === 'low') {
    return 'green';
  }
  return 'orange';
}

function getAcknowledgedFilterValue(acknowledged?: boolean) {
  if (acknowledged === undefined) {
    return undefined;
  }
  return acknowledged ? 'acknowledged' : 'pending';
}

function parseAcknowledgedFilterValue(value?: string) {
  if (value === 'acknowledged') {
    return true;
  }
  if (value === 'pending') {
    return false;
  }
  return undefined;
}

const emptyQuery: SecurityEventQuery = {
  keyword: '',
  username: '',
  eventType: '',
  severity: '',
  acknowledged: undefined,
  page: 1,
  pageSize: 10,
};

const SecurityEventList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canAcknowledge = isAdmin || hasPerm('system:security-event:acknowledge');
  const canClear = isAdmin || hasPerm('system:security-event:clear');
  const governanceRail = useGovernanceRail();
  const [retentionDays, setRetentionDays] = useState<number>(30);
  const [retentionOptions, setRetentionOptions] = useState<number[]>(() =>
    [...defaultRetentionOptions].sort((left, right) => right - left),
  );
  const [query, setQuery] = useState<SecurityEventQuery>(emptyQuery);
  const [data, setData] = useState<SecurityEventPageResp>({
    items: [],
    total: 0,
    pendingCount: 0,
    acknowledgedCount: 0,
    highSeverityCount: 0,
    page: 1,
    pageSize: 10,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [ackTarget, setAckTarget] = useState<SecurityEventRow | null>(null);
  const [batchAckVisible, setBatchAckVisible] = useState(false);
  const [ackNote, setAckNote] = useState('');
  const [ackSubmitting, setAckSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);

  const visibleSelectedRowKeys = useMemo(
    () =>
      getVisibleSelectedRowKeys(
        selectedRowKeys,
        data.items.map((item) => item.id),
      ),
    [data.items, selectedRowKeys],
  );

  const fetchData = async (nextQuery: SecurityEventQuery) => {
    // Commit the query synchronously (same as the other audit pages): a
    // deferred commit let a resolving response clobber in-flight keyword
    // drafts, and a failed request left pagination on the stale filter.
    setQuery(nextQuery);
    setLoading(true);
    setError(null);
    try {
      const result = await getAdminSecurityEventList(nextQuery);
      setData(result);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const timer = globalThis.setTimeout(() => void fetchData(emptyQuery), 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      getSettingGroup('audit')
        .then((group: SettingGroup) =>
          loadRetentionSetting(
            group,
            'audit.security_event_retention_options',
            setRetentionOptions,
            setRetentionDays,
          ),
        )
        .catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  const handleCleanup = async (payload: GovernanceCleanupPayload) => {
    try {
      const resp =
        payload.mode === 'range'
          ? await cleanupSecurityEvents({
              startedAt: payload.startedAt,
              endedAt: payload.endedAt,
            })
          : await cleanupSecurityEvents({ retentionDays: payload.retentionDays });
      message.success(t('auth.securityEvent.cleanupSuccess', { count: resp.clearedCount }));
      await fetchData(query);
    } catch {
      message.error(t('common.actionFailed'));
    }
  };

  const columns: ColumnProps<SecurityEventRow>[] = [
    {
      title: t('auth.securityEvent.createdAt'),
      dataIndex: 'createdAt',
      width: TABLE_COLUMN_WIDTH.datetime,
      render: (value) => (
        <Typography.Text className="system-list__datetime-text">
          {formatDateTime(value as string, { withSeconds: true })}
        </Typography.Text>
      ),
    },
    {
      title: t('common.username'),
      dataIndex: 'username',
      width: TABLE_COLUMN_WIDTH.identity,
      render: (value) => (value ? String(value) : '-'),
    },
    {
      title: t('auth.securityEvent.eventType'),
      dataIndex: 'eventType',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (value) =>
        t(`auth.securityEvent.type.${value}`, { defaultValue: String(value || '-') }),
    },
    {
      title: t('auth.securityEvent.severity'),
      dataIndex: 'severity',
      width: TABLE_COLUMN_WIDTH.status,
      render: (value) => {
        const severity = String(value || 'medium');
        const color = getSeverityTagColor(severity);
        return (
          <Tag color={color}>
            {t(`auth.securityEvent.severity.${severity}`, { defaultValue: severity })}
          </Tag>
        );
      },
    },
    {
      title: t('auth.securityEvent.sourceKey'),
      dataIndex: 'sourceKey',
      width: TABLE_COLUMN_WIDTH.identity,
      render: (value) => (value ? String(value) : '-'),
    },
    {
      title: t('auth.securityEvent.messageKey'),
      dataIndex: 'messageKey',
      width: TABLE_COLUMN_WIDTH.diagnostics,
      ellipsis: true,
      render: (value) => t(String(value || ''), { defaultValue: String(value || '-') }),
    },
    {
      title: t('auth.securityEvent.acknowledgement'),
      dataIndex: 'acknowledgedAt',
      width: TABLE_COLUMN_WIDTH.diagnostics,
      render: (_value, record) =>
        record.acknowledgedAt ? (
          <Space direction="vertical" size={2}>
            <Tag color="green">{t('auth.securityEvent.status.acknowledged')}</Tag>
            <Typography.Text type="secondary">
              {record.acknowledgedByUser || '-'} ·{' '}
              {formatDateTime(record.acknowledgedAt, { withSeconds: true })}
            </Typography.Text>
          </Space>
        ) : (
          <Tag color="orange">{t('auth.securityEvent.status.pending')}</Tag>
        ),
    },
    {
      title: t('common.action'),
      dataIndex: 'action',
      width: 120,
      render: (_value, record) =>
        record.acknowledgedAt ? (
          <Typography.Text type="secondary">
            {t('auth.securityEvent.status.acknowledged')}
          </Typography.Text>
        ) : (
          <Button
            size="small"
            type="text"
            icon={<IconCheck />}
            disabled={!canAcknowledge}
            onClick={() => {
              setAckTarget(record);
              setAckNote('');
            }}
          >
            {t('auth.securityEvent.acknowledge')}
          </Button>
        ),
    },
  ];

  const pagination = buildStandardPagination(t, {
    current: data.page,
    pageSize: data.pageSize,
    total: data.total,
    onChange: (page, pageSize) => {
      void fetchData({ ...query, page, pageSize });
    },
  });

  const handleSearch = (values: Partial<SecurityEventQuery>) => {
    setSelectedRowKeys([]);
    void fetchData({
      ...query,
      ...values,
      page: 1,
    });
  };

  const handleReset = () => {
    setSelectedRowKeys([]);
    void fetchData(emptyQuery);
  };

  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('common.total'),
        value: data.total,
      },
      {
        key: 'pending',
        label: t('auth.securityEvent.status.pending'),
        value: data.pendingCount ?? 0,
      },
      {
        key: 'acknowledged',
        label: t('auth.securityEvent.status.acknowledged'),
        value: data.acknowledgedCount ?? 0,
      },
      {
        key: 'high',
        label: t('auth.securityEvent.severity.high'),
        value: data.highSeverityCount ?? 0,
      },
    ],
    [data.acknowledgedCount, data.highSeverityCount, data.pendingCount, data.total, t],
  );

  return (
    <PageContainer>
      <Space
        direction="vertical"
        size={16}
        className="system-page-template auth-security-event-page"
      >
        <GovernanceSummaryBar
          eyebrow={t('auth.securityEvent.hero.eyebrow')}
          title={t('auth.securityEvent.hero.title')}
          description={t('auth.securityEvent.hero.desc')}
          metrics={heroStats.slice(0, 3)}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('auth.securityEvent.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('auth.securityEvent.search.placeholder')}
          onKeywordChange={(keyword) => handleSearch({ keyword })}
          inlineFilters={
            <>
              <Select
                allowClear
                placeholder={t('auth.securityEvent.filter.eventTypePlaceholder')}
                value={query.eventType || undefined}
                onChange={(value) => handleSearch({ eventType: value ?? '' })}
              >
                <Select.Option value="password_wrong">
                  {t('auth.securityEvent.type.password_wrong')}
                </Select.Option>
                <Select.Option value="source_blocked">
                  {t('auth.securityEvent.type.source_blocked')}
                </Select.Option>
                <Select.Option value="account_locked">
                  {t('auth.securityEvent.type.account_locked')}
                </Select.Option>
              </Select>
              <Select
                allowClear
                placeholder={t('auth.securityEvent.filter.severityPlaceholder')}
                value={query.severity || undefined}
                onChange={(value) => handleSearch({ severity: value ?? '' })}
              >
                <Select.Option value="high">{t('auth.securityEvent.severity.high')}</Select.Option>
                <Select.Option value="medium">
                  {t('auth.securityEvent.severity.medium')}
                </Select.Option>
                <Select.Option value="low">{t('auth.securityEvent.severity.low')}</Select.Option>
              </Select>
              <Select
                allowClear
                placeholder={t('auth.securityEvent.filter.acknowledgedPlaceholder')}
                value={getAcknowledgedFilterValue(query.acknowledged)}
                onChange={(value) =>
                  handleSearch({
                    acknowledged: parseAcknowledgedFilterValue(value),
                  })
                }
              >
                <Select.Option value="acknowledged">
                  {t('auth.securityEvent.status.acknowledged')}
                </Select.Option>
                <Select.Option value="pending">
                  {t('auth.securityEvent.status.pending')}
                </Select.Option>
              </Select>
              <TimeRangeFilter
                value={{ startedAt: query.startedAt, endedAt: query.endedAt }}
                onChange={(value) => handleSearch(value)}
              />
            </>
          }
          hasActiveFilters={Boolean(
            query.keyword ||
            query.eventType ||
            query.severity ||
            query.acknowledged !== undefined ||
            query.startedAt,
          )}
          onClearAll={handleReset}
        />
        <Card className="page-panel system-list__table-card auth-security-event-page__table-card">
          <Typography.Text type="secondary">{t('auth.securityEvent.hint')}</Typography.Text>
          {canClear || canAcknowledge ? (
            <GovernanceCleanupBar
              showCleanup={canClear}
              retentionDays={retentionDays}
              retentionOptions={retentionOptions}
              onRetentionChange={setRetentionDays}
              retentionLabel={(option) => t('common.keepRecentDays', { count: option })}
              confirmTitle={t('auth.securityEvent.cleanupWarning')}
              actionLabel={t('auth.securityEvent.cleanupAction')}
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
              hint={t('auth.securityEvent.cleanupHint')}
              extraActions={
                canAcknowledge ? (
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
                    <Button
                      type="primary"
                      icon={<IconCheck />}
                      disabled={selectedRowKeys.length === 0}
                      onClick={() => {
                        setAckNote('');
                        setBatchAckVisible(true);
                      }}
                    >
                      {t('auth.securityEvent.batchAcknowledge')}
                    </Button>
                  </>
                ) : undefined
              }
            />
          ) : null}
          {loading && data.items.length === 0 ? <PageLoading /> : null}
          {error ? <PageRequestError error={error} onRetry={() => void fetchData(query)} /> : null}
          {!error && data.items.length === 0 && !loading ? (
            <PageEmpty description={t('auth.securityEvent.empty')} />
          ) : null}
          {!error && data.items.length > 0 ? (
            <AppTable<SecurityEventRow>
              className="system-list__table"
              rowKey="id"
              columns={columns}
              data={data.items}
              loading={loading}
              pagination={pagination}
              scroll={{ x: 'max-content' }}
              rowSelection={
                canAcknowledge
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
                              data.items.map((item) => item.id),
                            ) as number[],
                        ),
                      checkboxProps: (record: SecurityEventRow) => ({
                        disabled: Boolean(record.acknowledgedAt),
                      }),
                    }
                  : undefined
              }
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('auth.securityEvent.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.menu.securityEvent')}
        noteDescription={t('auth.securityEvent.hint')}
        noteTone="warning"
      >
        <GovernanceRailSummary
          items={[
            {
              label: t('auth.securityEvent.status.pending'),
              value: heroStats[1]?.value ?? 0,
              description: t('auth.securityEvent.pendingHint'),
            },
            {
              label: t('auth.securityEvent.status.acknowledged'),
              value: heroStats[2]?.value ?? 0,
              description: t('auth.securityEvent.acknowledgedHint'),
            },
            {
              tone: 'warning',
              label: t('auth.securityEvent.severity.high'),
              value: heroStats[3]?.value ?? 0,
              description: t('auth.securityEvent.highSeverityHint'),
            },
          ]}
        />
      </GovernanceInsightDrawer>
      <AppModal
        title={
          batchAckVisible
            ? t('auth.securityEvent.batchAcknowledge')
            : t('auth.securityEvent.acknowledge')
        }
        visible={Boolean(ackTarget) || batchAckVisible}
        onCancel={() => {
          if (ackSubmitting) {
            return;
          }
          setAckTarget(null);
          setBatchAckVisible(false);
          setAckNote('');
        }}
        onOk={async () => {
          if (!ackTarget && !batchAckVisible) {
            return;
          }
          if (!ackNote.trim()) {
            message.warning(t('auth.securityEvent.acknowledgeNoteRequired'));
            return;
          }
          setAckSubmitting(true);
          try {
            if (batchAckVisible) {
              const resp = await batchAcknowledgeSecurityEvents({
                ids: selectedRowKeys,
                acknowledgementNote: ackNote.trim(),
              });
              message.success(
                t('auth.securityEvent.batchAcknowledgeSuccess', {
                  count: resp.acknowledgedCount,
                }),
              );
              setSelectedRowKeys([]);
            } else if (ackTarget) {
              await acknowledgeSecurityEvent(ackTarget.id, {
                acknowledgementNote: ackNote.trim(),
              });
              message.success(t('auth.securityEvent.acknowledgeSuccess'));
            }
            setAckTarget(null);
            setBatchAckVisible(false);
            setAckNote('');
            await fetchData(query);
          } catch {
            message.error(t('common.actionFailed'));
          } finally {
            setAckSubmitting(false);
          }
        }}
        confirmLoading={ackSubmitting}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Text>
            {batchAckVisible
              ? t('auth.securityEvent.batchAcknowledgeDialogHint')
              : t('auth.securityEvent.acknowledgeDialogHint')}
          </Typography.Text>
          <TextArea value={ackNote} onChange={setAckNote} maxLength={500} showWordLimit />
        </Space>
      </AppModal>
    </PageContainer>
  );
};

export default SecurityEventList;
