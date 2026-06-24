import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Message,
  Popconfirm,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconLeft, IconLink } from '@arco-design/web-react/icon';
import {
  AppDrawer,
  AppTable,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
} from '../../../components';
import PageHeader from '../../../components/patterns/PageHeader';
import FormSection from '../../../components/patterns/FormSection';
import { usePermission } from '../../../hooks/usePermission';
import { formatDateTime } from '../../../core/format/dateTime';
import {
  bindBizScopeHosts,
  getBizScopeAvailableHosts,
  getBizScopeDetail,
  getBizScopeHosts,
  unbindBizScopeHost,
  type BizScopeDetail as BizScopeDetailType,
  type BizScopeHostRow,
} from './api';
import '../../system/list-page.css';
import '../cmdb/cmdb.css';

function bizScopeHostStatusColor(status: string) {
  switch (status) {
    case 'online':
      return 'green';
    case 'assigned':
      return 'arcoblue';
    case 'offline':
      return 'orangered';
    default:
      return 'gray';
  }
}

export default function BizScopeDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [detail, setDetail] = useState<BizScopeDetailType | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [hosts, setHosts] = useState<BizScopeHostRow[]>([]);
  const [hostsLoading, setHostsLoading] = useState(true);
  const [hostsError, setHostsError] = useState<unknown>(null);
  const [availableHosts, setAvailableHosts] = useState<BizScopeHostRow[]>([]);
  const [availableHostsLoading, setAvailableHostsLoading] = useState(false);
  const [availableHostsError, setAvailableHostsError] = useState<unknown>(null);
  const [bindDrawerVisible, setBindDrawerVisible] = useState(false);
  const [selectedAvailableHostIds, setSelectedAvailableHostIds] = useState<Array<string | number>>([]);
  const [binding, setBinding] = useState(false);
  const [unbindingHostId, setUnbindingHostId] = useState<number | null>(null);

  const scopeId = id ? Number(id) : 0;
  const canUpdate = hasPerm('business:bizscope:update');

  const loadHosts = useCallback(async () => {
    if (!scopeId) {
      return;
    }
    setHostsLoading(true);
    setHostsError(null);
    try {
      const result = await getBizScopeHosts(scopeId);
      setHosts(result.items);
    } catch (requestError) {
      setHostsError(requestError);
      setHosts([]);
    } finally {
      setHostsLoading(false);
    }
  }, [scopeId]);

  const loadAvailableHosts = useCallback(async () => {
    if (!scopeId) {
      return;
    }
    setAvailableHostsLoading(true);
    setAvailableHostsError(null);
    try {
      const result = await getBizScopeAvailableHosts(scopeId);
      setAvailableHosts(result.items);
    } catch (requestError) {
      setAvailableHostsError(requestError);
      setAvailableHosts([]);
    } finally {
      setAvailableHostsLoading(false);
    }
  }, [scopeId]);

  const loadDetail = useCallback(async () => {
    if (!scopeId) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getBizScopeDetail(scopeId);
      setDetail(result);
    } catch (requestError) {
      setError(requestError);
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, [scopeId]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadDetail();
      void loadHosts();
    });
  }, [loadDetail, loadHosts]);

  const openBindDrawer = useCallback(async () => {
    setBindDrawerVisible(true);
    setSelectedAvailableHostIds([]);
    await loadAvailableHosts();
  }, [loadAvailableHosts]);

  const handleBindHosts = useCallback(async () => {
    if (!scopeId || selectedAvailableHostIds.length === 0) {
      return;
    }
    setBinding(true);
    try {
      await bindBizScopeHosts(
        scopeId,
        selectedAvailableHostIds.map((item) => Number(item)),
      );
      Message.success(t('business.bizscope.bindSuccess'));
      setBindDrawerVisible(false);
      setSelectedAvailableHostIds([]);
      await Promise.all([loadDetail(), loadHosts()]);
    } finally {
      setBinding(false);
    }
  }, [loadDetail, loadHosts, scopeId, selectedAvailableHostIds, t]);

  const handleUnbindHost = useCallback(
    async (hostId: number) => {
      if (!scopeId) {
        return;
      }
      setUnbindingHostId(hostId);
      try {
        await unbindBizScopeHost(scopeId, hostId);
        Message.success(t('business.bizscope.unbindSuccess'));
        await Promise.all([loadDetail(), loadHosts()]);
      } finally {
        setUnbindingHostId(null);
      }
    },
    [loadDetail, loadHosts, scopeId, t],
  );

  const hostHeroStats = useMemo(
    () =>
      detail
        ? [
            {
              key: 'hostCount',
              label: t('business.bizscope.hero.boundHosts'),
              value: detail.hostCount,
              hint: t('business.bizscope.hero.boundHostsHint'),
            },
            {
              key: 'status',
              label: t('business.bizscope.field.status'),
              value: t(`business.bizscope.status.${detail.status}`),
              hint: t('business.bizscope.hero.statusHint'),
            },
            {
              key: 'environment',
              label: t('business.bizscope.field.environment'),
              value: t(`business.bizscope.environment.${detail.environment}`),
              hint: t('business.bizscope.hero.environmentHint'),
            },
            {
              key: 'updatedAt',
              label: t('common.updatedAt'),
              value: formatDateTime(detail.updatedAt),
              hint: t('business.bizscope.hero.updatedAtHint'),
            },
          ]
        : [],
    [detail, t],
  );

  const hostColumns = useMemo<ColumnProps<BizScopeHostRow>[]>(
    () => {
      const columns: ColumnProps<BizScopeHostRow>[] = [
        {
          title: t('business.cmdb.host.hostname'),
          dataIndex: 'hostname',
          width: 180,
        },
        {
          title: t('business.cmdb.host.ip'),
          dataIndex: 'ip',
          width: 160,
        },
        {
          title: t('business.cmdb.host.os'),
          dataIndex: 'os',
          width: 120,
          render: (_: unknown, row) => t(`business.cmdb.host.os.${row.os}`),
        },
        {
          title: t('business.cmdb.host.status'),
          dataIndex: 'status',
          width: 120,
          render: (_: unknown, row) => (
            <Tag color={bizScopeHostStatusColor(row.status)}>
              {t(`business.cmdb.host.status.${row.status}`)}
            </Tag>
          ),
        },
      ];
      if (canUpdate) {
        columns.push({
          title: t('common.action'),
          dataIndex: 'action',
          width: 120,
          fixed: 'right',
          render: (_: unknown, row) => (
            <Popconfirm
              title={t('business.bizscope.unbindConfirm', { hostname: row.hostname })}
              onOk={() => {
                void handleUnbindHost(row.id);
              }}
            >
              <Button
                type="text"
                size="small"
                status="danger"
                loading={unbindingHostId === row.id}
              >
                {t('business.bizscope.unbindAction')}
              </Button>
            </Popconfirm>
          ),
        });
      }
      return columns;
    },
    [canUpdate, handleUnbindHost, t, unbindingHostId],
  );

  const availableHostColumns = useMemo<ColumnProps<BizScopeHostRow>[]>(
    () => [
      {
        title: t('business.cmdb.host.hostname'),
        dataIndex: 'hostname',
        width: 180,
      },
      {
        title: t('business.cmdb.host.ip'),
        dataIndex: 'ip',
        width: 160,
      },
      {
        title: t('business.cmdb.host.os'),
        dataIndex: 'os',
        width: 120,
        render: (_: unknown, row) => t(`business.cmdb.host.os.${row.os}`),
      },
      {
        title: t('business.cmdb.host.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={bizScopeHostStatusColor(row.status)}>
            {t(`business.cmdb.host.status.${row.status}`)}
          </Tag>
        ),
      },
    ],
    [t],
  );

  if (loading) {
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  }

  if (error || !detail) {
    return (
      <PageContainer>
        <PageError description={t('common.loadFailedDesc')} onRetry={loadDetail} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <PageHeader
        title={detail.name}
        subtitle={t('operations.bizscope.detail')}
        extra={
          <Space>
            {canUpdate ? (
              <Button icon={<IconLink />} type="primary" onClick={() => void openBindDrawer()}>
                {t('business.bizscope.bindHosts')}
              </Button>
            ) : null}
            <Button icon={<IconLeft />} onClick={() => navigate('/operations/business-scope')}>
              {t('common.back')}
            </Button>
          </Space>
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero cmdb-page__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('business.bizscope.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title cmdb-page__hero-title">
                {detail.name}
              </Typography.Title>
              <Typography.Text type="secondary" className="system-page-hero__desc">
                {t('business.bizscope.detailLead')}
              </Typography.Text>
            </div>
          </div>
          <div className="cmdb-page__hero-grid">
            {hostHeroStats.map((item) => (
              <div key={item.key} className="cmdb-page__hero-metric">
                <span className="cmdb-page__hero-label">{item.label}</span>
                <span className="cmdb-page__hero-value">{item.value}</span>
                <span className="cmdb-page__hero-hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.bizscope.summaryTitle')}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.bizscope.field.code'), value: detail.code },
                { label: t('business.bizscope.field.name'), value: detail.name },
                { label: t('business.bizscope.field.owner'), value: detail.owner || '-' },
                {
                  label: t('business.bizscope.field.environment'),
                  value: t(`business.bizscope.environment.${detail.environment}`),
                },
                {
                  label: t('business.bizscope.field.status'),
                  value: t(`business.bizscope.status.${detail.status}`),
                },
                { label: t('business.bizscope.field.hostCount'), value: detail.hostCount },
                { label: t('common.updatedAt'), value: formatDateTime(detail.updatedAt) },
                { label: t('business.bizscope.field.remark'), value: detail.remark || '-' },
              ]}
            />
          </FormSection>
        </Card>
        <Card
          className="page-panel system-list__table-card"
          title={t('business.bizscope.boundHostsTitle')}
          extra={
            canUpdate ? (
              <Button size="small" onClick={() => void openBindDrawer()}>
                {t('business.bizscope.bindHosts')}
              </Button>
            ) : (
              <Button size="small" onClick={() => void loadHosts()}>
                {t('common.refresh')}
              </Button>
            )
          }
        >
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            {hostsLoading ? <PageLoading /> : null}
            {!hostsLoading && hostsError ? (
              <PageError
                description={t('business.bizscope.hostsLoadFailed')}
                onRetry={() => {
                  void loadHosts();
                }}
              />
            ) : null}
            {!hostsLoading && !hostsError && hosts.length === 0 ? (
              <PageEmpty description={t('business.bizscope.boundHostsEmpty')} />
            ) : null}
            {!hostsLoading && !hostsError && hosts.length > 0 ? (
              <AppTable
                rowKey="id"
                className="system-list__table"
                columns={hostColumns}
                data={hosts}
                pagination={false}
                scroll={{ x: 'max-content' }}
              />
            ) : null}
          </Space>
        </Card>
      </Space>
      <AppDrawer
        visible={bindDrawerVisible}
        onCancel={() => {
          setBindDrawerVisible(false);
          setSelectedAvailableHostIds([]);
        }}
        title={t('business.bizscope.bindDrawerTitle', { name: detail.name })}
        size="lg"
        footer={
          <Space>
            <Button
              onClick={() => {
                setBindDrawerVisible(false);
                setSelectedAvailableHostIds([]);
              }}
            >
              {t('common.close')}
            </Button>
            <Button
              type="primary"
              onClick={() => {
                void handleBindHosts();
              }}
              disabled={selectedAvailableHostIds.length === 0}
              loading={binding}
            >
              {t('business.bizscope.bindSubmit', { count: selectedAvailableHostIds.length })}
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <Typography.Text type="secondary">
            {t('business.bizscope.availableHostsLead')}
          </Typography.Text>
          {availableHostsLoading ? <PageLoading /> : null}
          {!availableHostsLoading && availableHostsError ? (
            <PageError
              description={t('business.bizscope.availableHostsLoadFailed')}
              onRetry={() => {
                void loadAvailableHosts();
              }}
            />
          ) : null}
          {!availableHostsLoading && !availableHostsError && availableHosts.length === 0 ? (
            <PageEmpty description={t('business.bizscope.availableHostsEmpty')} />
          ) : null}
          {!availableHostsLoading && !availableHostsError && availableHosts.length > 0 ? (
            <AppTable
              rowKey="id"
              className="system-list__table"
              columns={availableHostColumns}
              data={availableHosts}
              pagination={false}
              scroll={{ x: 'max-content' }}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys: selectedAvailableHostIds,
                onChange: (rowKeys) => setSelectedAvailableHostIds(rowKeys),
              }}
            />
          ) : null}
        </Space>
      </AppDrawer>
    </PageContainer>
  );
}
