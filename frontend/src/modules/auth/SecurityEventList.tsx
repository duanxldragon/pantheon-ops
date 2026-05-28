import React, { useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconCheck, IconSearch } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { message } from '../../components/feedback/message';
import { formatDateTime } from '../../core/format/dateTime';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
} from '../../components';
import { usePermission } from '../../hooks/usePermission';
import {
  acknowledgeSecurityEvent,
  getAdminSecurityEventList,
  type SecurityEventPageResp,
  type SecurityEventQuery,
  type SecurityEventRow,
} from './api';
import './auth.css';
import '../system/list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const TextArea = Input.TextArea;

const emptyQuery: SecurityEventQuery = {
  username: '',
  eventType: '',
  severity: '',
  acknowledged: undefined,
  page: 1,
  pageSize: 10,
};

type SecurityEventFilterForm = Omit<SecurityEventQuery, 'acknowledged'> & {
  acknowledged?: 'acknowledged' | 'pending';
};

const emptyFilterForm: SecurityEventFilterForm = {
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
  const governanceRail = useGovernanceRail();
  const [form] = Form.useForm<SecurityEventFilterForm>();
  const [query, setQuery] = useState<SecurityEventQuery>(emptyQuery);
  const [data, setData] = useState<SecurityEventPageResp>({
    items: [],
    total: 0,
    page: 1,
    pageSize: 10,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [ackTarget, setAckTarget] = useState<SecurityEventRow | null>(null);
  const [ackNote, setAckNote] = useState('');
  const [ackSubmitting, setAckSubmitting] = useState(false);

  const fetchData = async (nextQuery: SecurityEventQuery) => {
    setLoading(true);
    setError(null);
    try {
      const result = await getAdminSecurityEventList(nextQuery);
      setData(result);
      setQuery(nextQuery);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'request.failed');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const timer = window.setTimeout(() => void fetchData(emptyQuery), 0);
    return () => window.clearTimeout(timer);
  }, []);

  const columns: ColumnProps<SecurityEventRow>[] = [
    {
      title: t('auth.securityEvent.createdAt'),
      dataIndex: 'createdAt',
      width: TABLE_COLUMN_WIDTH.datetime,
      render: (value) => formatDateTime(value as string),
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
        const color = severity === 'high' ? 'red' : severity === 'low' ? 'green' : 'orange';
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
              {record.acknowledgedByUser || '-'} · {formatDateTime(record.acknowledgedAt)}
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

  const handleSearch = () => {
    const values = form.getFieldsValue();
    void fetchData({
      ...emptyQuery,
      ...values,
      acknowledged:
        values.acknowledged === 'acknowledged'
          ? true
          : values.acknowledged === 'pending'
            ? false
            : undefined,
      page: 1,
    });
  };

  const handleReset = () => {
    form.resetFields();
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
        value: data.items.filter((item) => !item.acknowledgedAt).length,
      },
      {
        key: 'acknowledged',
        label: t('auth.securityEvent.status.acknowledged'),
        value: data.items.filter((item) => Boolean(item.acknowledgedAt)).length,
      },
      {
        key: 'high',
        label: t('auth.securityEvent.severity.high'),
        value: data.items.filter((item) => item.severity === 'high').length,
      },
    ],
    [data.items, data.total, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template auth-security-event-page">
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
        <FilterPanel>
          <Form form={form} layout="vertical" initialValues={emptyFilterForm}>
            <Row gutter={16} className="auth-filter-grid auth-security-event-page__filter-grid">
              <Col xs={24} md={12} lg={5}>
                <FormItem field="username" label={t('common.user')}>
                  <Input allowClear placeholder={t('auth.securityEvent.filter.usernamePlaceholder')} />
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={5}>
                <FormItem field="eventType" label={t('auth.securityEvent.eventType')}>
                  <Select allowClear placeholder={t('auth.securityEvent.filter.eventTypePlaceholder')}>
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
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={5}>
                <FormItem field="severity" label={t('auth.securityEvent.severity')}>
                  <Select allowClear placeholder={t('auth.securityEvent.filter.severityPlaceholder')}>
                    <Select.Option value="high">{t('auth.securityEvent.severity.high')}</Select.Option>
                    <Select.Option value="medium">{t('auth.securityEvent.severity.medium')}</Select.Option>
                    <Select.Option value="low">{t('auth.securityEvent.severity.low')}</Select.Option>
                  </Select>
                </FormItem>
              </Col>
              <Col xs={24} md={12} lg={5}>
                <FormItem field="acknowledged" label={t('auth.securityEvent.acknowledgement')}>
                  <Select
                    allowClear
                    placeholder={t('auth.securityEvent.filter.acknowledgedPlaceholder')}
                  >
                    <Select.Option value="acknowledged">
                      {t('auth.securityEvent.status.acknowledged')}
                    </Select.Option>
                    <Select.Option value="pending">
                      {t('auth.securityEvent.status.pending')}
                    </Select.Option>
                  </Select>
                </FormItem>
              </Col>
                <Col xs={24} md={12} lg={4}>
                  <FormItem className="filter-panel__action-item auth-security-event-page__filter-actions">
                    <Space>
                      <Button type="primary" icon={<IconSearch />} onClick={handleSearch}>
                        {t('common.search')}
                    </Button>
                    <Button onClick={handleReset}>{t('common.reset')}</Button>
                  </Space>
                </FormItem>
              </Col>
            </Row>
          </Form>
        </FilterPanel>
        <Card className="page-panel system-list__table-card auth-security-event-page__table-card">
          <Typography.Text type="secondary">{t('auth.securityEvent.hint')}</Typography.Text>
          {loading && data.items.length === 0 ? <PageLoading /> : null}
          {error ? <PageError description={error} onRetry={() => void fetchData(query)} /> : null}
          {!error && data.items.length === 0 && !loading ? (
            <PageEmpty description={t('auth.securityEvent.empty')} />
          ) : null}
          {!error && data.items.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="id"
              columns={columns}
              data={data.items}
              loading={loading}
              pagination={pagination}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('auth.securityEvent.subtitle')}
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
        title={t('auth.securityEvent.acknowledge')}
        visible={Boolean(ackTarget)}
        onCancel={() => {
          if (ackSubmitting) {
            return;
          }
          setAckTarget(null);
          setAckNote('');
        }}
        onOk={async () => {
          if (!ackTarget) {
            return;
          }
          if (!ackNote.trim()) {
            message.warning(t('auth.securityEvent.acknowledgeNoteRequired'));
            return;
          }
          setAckSubmitting(true);
          try {
            await acknowledgeSecurityEvent(ackTarget.id, { acknowledgementNote: ackNote.trim() });
            message.success(t('auth.securityEvent.acknowledgeSuccess'));
            setAckTarget(null);
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
          <Typography.Text>{t('auth.securityEvent.acknowledgeDialogHint')}</Typography.Text>
          <TextArea value={ackNote} onChange={setAckNote} maxLength={500} showWordLimit />
        </Space>
      </AppModal>
    </PageContainer>
  );
};

export default SecurityEventList;
