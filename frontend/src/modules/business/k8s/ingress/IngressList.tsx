import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Input, Message, Popconfirm, Select, Space, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconSync, IconDelete } from '@arco-design/web-react/icon';
import {
  AppTable,
  SearchToolbar,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  getK8sClusterList,
  getK8sIngressList,
  syncK8sIngress,
  deleteK8sIngress,
  type K8sIngressRow,
  type K8sClusterRow,
} from '../api';
import '../../../../assets/list-page.css';

function ingressStatusColor(status: string): string {
  if (status === 'ready') {
    return 'green';
  }
  if (status === 'error') {
    return 'red';
  }
  return 'orange';
}

export default function IngressList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [namespace, setNamespace] = useState('');
  const [name, setName] = useState('');
  const [data, setData] = useState<K8sIngressRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const canDelete = hasPerm('business:k8s:ingress:delete');
  const canSync = hasPerm('business:k8s:ingress:sync');

  const loadClusters = useCallback(async () => {
    try {
      const result = await getK8sClusterList({ page: 1, pageSize: 100 });
      setClusters(result.items);
    } catch {
      // cluster options load is best-effort
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadClusters();
    });
  }, [loadClusters]);

  const loadData = useCallback(async () => {
    if (!clusterId) {
      setData([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sIngressList({
        clusterId,
        namespace: namespace || undefined,
        name: name || undefined,
      });
      setData(result.items);
    } catch (requestError) {
      setError(requestError);
      setData([]);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [clusterId, name, namespace, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'total', label: t('business.k8s.ingress.hero.total'), value: data.length },
      {
        key: 'active',
        label: t('business.k8s.ingress.hero.active'),
        value: data.filter((item) => item.status === 'ready').length,
      },
    ],
    [data, t],
  );

  const handleSync = useCallback(
    async (row: K8sIngressRow) => {
      try {
        await syncK8sIngress(row.id);
        Message.success(t('business.k8s.ingress.syncSuccess'));
        await loadData();
      } catch {
        Message.error(t('business.k8s.ingress.syncFailed'));
      }
    },
    [loadData, t],
  );

  const handleDelete = useCallback(
    async (row: K8sIngressRow) => {
      try {
        await deleteK8sIngress(row.id);
        Message.success(t('business.k8s.ingress.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('business.k8s.ingress.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const columns = useMemo<ColumnProps<K8sIngressRow>[]>(
    () => [
      { title: t('business.k8s.ingress.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.ingress.field.namespace'), dataIndex: 'namespace', width: 160 },
      { title: t('business.k8s.ingress.field.cluster'), dataIndex: 'clusterName', width: 140 },
      { title: t('business.k8s.ingress.field.ingressClass'), dataIndex: 'ingressClass', width: 140 },
      {
        title: t('business.k8s.ingress.field.hosts'),
        dataIndex: 'hosts',
        width: 260,
        render: (_: unknown, row) => (
          <Space size={4} wrap>
            {(row.hosts || []).map((host) => (
              <Tag key={host} color="arcoblue">{host}</Tag>
            ))}
          </Space>
        ),
      },
      {
        title: t('business.k8s.ingress.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={ingressStatusColor(row.status)}>
            {row.status}
          </Tag>
        ),
      },
      { title: t('business.k8s.ingress.field.createdAt'), dataIndex: 'createdAt', width: 180 },
      {
        title: t('common.action'),
        width: 160,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canSync ? (
              <Button type="text" size="small" icon={<IconSync />} onClick={() => void handleSync(row)}>
                {t('business.k8s.ingress.sync')}
              </Button>
            ) : null}
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.ingress.deleteConfirm', { name: row.name })}
                onOk={() => void handleDelete(row)}
              >
                <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canSync, handleDelete, handleSync, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.ingress.hero.eyebrow')}
          title={t('operations.k8s.ingress.menu')}
          description={t('business.k8s.ingress.hero.title')}
          metrics={heroMetrics}
        />
        <SearchToolbar
          keyword={name}
          keywordPlaceholder={t('business.k8s.ingress.field.name')}
          onKeywordChange={(value) => setName(value)}
          inlineFilters={
            <>
              <Select
                placeholder={t('business.k8s.ingress.field.cluster')}
                value={clusterId}
                onChange={(value) => {
                  setClusterId(value);
                  setData([]);
                }}
                style={{ width: 220 }}
              >
                {clusters.map((cluster) => (
                  <Select.Option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </Select.Option>
                ))}
              </Select>
              <Input
                allowClear
                placeholder={t('business.k8s.ingress.field.namespacePlaceholder')}
                value={namespace}
                onChange={(value) => setNamespace(value)}
                style={{ width: 200 }}
              />
            </>
          }
          hasActiveFilters={Boolean(namespace || name)}
          onClearAll={() => {
            setNamespace('');
            setName('');
          }}
        />
        <Card className="page-panel system-list__table-card">
          {!clusterId ? <PageEmpty description={t('business.k8s.ingress.selectClusterHint')} /> : null}
          {clusterId && loading ? <PageLoading /> : null}
          {clusterId && error && !loading ? <PageError onRetry={loadData} /> : null}
          {clusterId && !loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.ingress.empty')} />
          ) : null}
          {clusterId && !loading && !error && data.length > 0 ? (
            <AppTable rowKey="id" columns={columns} data={data} pagination={false} scroll={{ x: 'max-content' }} />
          ) : null}
        </Card>
      </Space>
    </PageContainer>
  );
}
