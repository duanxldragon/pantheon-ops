import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  InputNumber,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconRefresh, IconEye } from '@arco-design/web-react/icon';
import {
  AppDrawer,
  AppModal,
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
  getK8sWorkloadList,
  getK8sWorkloadPods,
  restartK8sWorkload,
  scaleK8sWorkload,
  type K8sClusterRow,
  type K8sPodRow,
  type K8sWorkloadRow,
} from '../api';
import PodLogViewer from './PodLogViewer';
import '../../../system/components/shared/list-page.css';

const kindColor: Record<string, string> = {
  Deployment: 'arcoblue',
  StatefulSet: 'purple',
  DaemonSet: 'cyan',
};

export default function WorkloadList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [namespace, setNamespace] = useState('');
  const [kind, setKind] = useState('');
  const [data, setData] = useState<K8sWorkloadRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const [scaleTarget, setScaleTarget] = useState<K8sWorkloadRow | null>(null);
  const [scaleReplicas, setScaleReplicas] = useState<number>(1);
  const [scaling, setScaling] = useState(false);

  const [podsTarget, setPodsTarget] = useState<K8sWorkloadRow | null>(null);
  const [pods, setPods] = useState<K8sPodRow[]>([]);
  const [podsLoading, setPodsLoading] = useState(false);

  const [logTarget, setLogTarget] = useState<{ clusterId: number; namespace: string; podName: string } | null>(null);

  const canUpdate = hasPerm('business:k8s:workload:update');

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
      const result = await getK8sWorkloadList({
        clusterId,
        namespace: namespace || undefined,
        kind: kind || undefined,
      });
      setData(result.items);
    } catch (requestError) {
      setError(requestError);
      setData([]);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [clusterId, kind, namespace, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'total', label: t('business.k8s.workload.hero.total'), value: data.length },
      {
        key: 'ready',
        label: t('business.k8s.workload.hero.ready'),
        value: data.filter((item) => item.status === 'ready').length,
      },
      {
        key: 'progressing',
        label: t('business.k8s.workload.hero.progressing'),
        value: data.filter((item) => item.status === 'progressing').length,
      },
    ],
    [data, t],
  );

  const handleScale = useCallback(async () => {
    if (!scaleTarget) return;
    setScaling(true);
    try {
      await scaleK8sWorkload(clusterId!, scaleTarget.namespace, scaleTarget.kind.toLowerCase(), scaleTarget.name, scaleReplicas);
      Message.success(t('business.k8s.workload.scaleSuccess'));
      setScaleTarget(null);
      await loadData();
    } finally {
      setScaling(false);
    }
  }, [clusterId, loadData, scaleReplicas, scaleTarget, t]);

  const handleRestart = useCallback(
    async (row: K8sWorkloadRow) => {
      if (!clusterId) return;
      try {
        await restartK8sWorkload(clusterId, row.namespace, row.kind.toLowerCase(), row.name);
        Message.success(t('business.k8s.workload.restartSuccess'));
      } catch {
        Message.error(t('business.k8s.workload.restartFailed'));
      }
    },
    [clusterId, t],
  );

  const openPods = useCallback(async (row: K8sWorkloadRow) => {
    if (!clusterId) return;
    setPodsTarget(row);
    setPodsLoading(true);
    try {
      const result = await getK8sWorkloadPods(clusterId, row.namespace, row.kind.toLowerCase(), row.name);
      setPods(result.items);
    } catch {
      setPods([]);
    } finally {
      setPodsLoading(false);
    }
  }, [clusterId]);

  const columns = useMemo<ColumnProps<K8sWorkloadRow>[]>(
    () => [
      {
        title: t('business.k8s.workload.field.kind'),
        dataIndex: 'kind',
        width: 130,
        render: (_: unknown, row) => <Tag color={kindColor[row.kind] || 'gray'}>{row.kind}</Tag>,
      },
      { title: t('business.k8s.workload.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.workload.field.namespace'), dataIndex: 'namespace', width: 160 },
      {
        title: t('business.k8s.workload.field.replicas'),
        dataIndex: 'replicas',
        width: 110,
        render: (_: unknown, row) => `${row.readyReplicas}/${row.replicas}`,
      },
      {
        title: t('business.k8s.workload.field.images'),
        dataIndex: 'images',
        render: (_: unknown, row) => (
          <Space direction="vertical" size={2}>
            {row.images.map((image) => (
              <Tag key={image} color="gray">{image}</Tag>
            ))}
          </Space>
        ),
      },
      {
        title: t('business.k8s.workload.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'ready' ? 'green' : 'orange'}>
            {t(`business.k8s.workload.status.${row.status}`)}
          </Tag>
        ),
      },
      { title: t('business.k8s.workload.field.age'), dataIndex: 'age', width: 100 },
      {
        title: t('common.action'),
        width: 220,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            <Button type="text" size="small" icon={<IconEye />} onClick={() => void openPods(row)}>
              {t('business.k8s.workload.pods')}
            </Button>
            {canUpdate && row.kind !== 'DaemonSet' ? (
              <Button
                type="text"
                size="small"
                onClick={() => {
                  setScaleTarget(row);
                  setScaleReplicas(row.replicas || 1);
                }}
              >
                {t('business.k8s.workload.scale')}
              </Button>
            ) : null}
            {canUpdate && row.kind !== 'DaemonSet' ? (
              <Popconfirm title={t('business.k8s.workload.restartConfirm')} onOk={() => void handleRestart(row)}>
                <Button type="text" size="small" icon={<IconRefresh />}>
                  {t('business.k8s.workload.restart')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canUpdate, handleRestart, openPods, t],
  );

  const podColumns = useMemo<ColumnProps<K8sPodRow>[]>(
    () => [
      { title: t('business.k8s.workload.pod.name'), dataIndex: 'name', width: 220 },
      {
        title: t('business.k8s.workload.pod.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'Running' ? 'green' : 'orange'}>{row.status}</Tag>
        ),
      },
      { title: t('business.k8s.workload.pod.node'), dataIndex: 'nodeName', width: 170 },
      { title: t('business.k8s.workload.pod.restarts'), dataIndex: 'restarts', width: 90 },
      { title: t('business.k8s.workload.pod.createdAt'), dataIndex: 'createdAt', width: 180 },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Button
            type="text"
            size="small"
            icon={<IconEye />}
            onClick={() => {
              if (clusterId && podsTarget) {
                setLogTarget({ clusterId, namespace: podsTarget.namespace, podName: row.name });
              }
            }}
          >
            {t('business.k8s.workload.log.view')}
          </Button>
        ),
      },
    ],
    [clusterId, podsTarget, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.workload.hero.eyebrow')}
          title={t('operations.k8s.workload.menu')}
          description={t('business.k8s.workload.hero.title')}
          metrics={heroMetrics}
        />
        <SearchToolbar
          keyword={namespace}
          keywordPlaceholder={t('business.k8s.workload.field.namespacePlaceholder')}
          onKeywordChange={(value) => setNamespace(value)}
          inlineFilters={
            <>
              <Select
                placeholder={t('business.k8s.workload.field.cluster')}
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
              <Select
                allowClear
                placeholder={t('business.k8s.workload.field.kind')}
                value={kind || undefined}
                onChange={(value) => setKind(value || '')}
              >
                {['deployment', 'statefulset', 'daemonset'].map((item) => (
                  <Select.Option key={item} value={item}>
                    {t(`business.k8s.workload.kind.${item}`)}
                  </Select.Option>
                ))}
              </Select>
            </>
          }
          hasActiveFilters={Boolean(namespace || kind)}
          onClearAll={() => {
            setNamespace('');
            setKind('');
          }}
        />
        <Card className="page-panel system-list__table-card">
          {!clusterId ? (
            <PageEmpty description={t('business.k8s.workload.selectClusterHint')} />
          ) : null}
          {clusterId && loading ? <PageLoading /> : null}
          {clusterId && error && !loading ? (
            <PageError onRetry={loadData} />
          ) : null}
          {clusterId && !loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.workload.empty')} />
          ) : null}
          {clusterId && !loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              rowKey={(record) => `${record.kind}/${record.namespace}/${record.name}`}
              columns={columns}
              data={data}
              pagination={false}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
      <AppModal
        visible={Boolean(scaleTarget)}
        footer={null}
        title={t('business.k8s.workload.scaleTitle', { name: scaleTarget?.name ?? '' })}
        onCancel={() => setScaleTarget(null)}
      >
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          <InputNumber
            min={0}
            value={scaleReplicas}
            onChange={(value) => setScaleReplicas(Number(value) || 0)}
            style={{ width: '100%' }}
          />
          <Space>
            <Button onClick={() => setScaleTarget(null)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={scaling} onClick={() => void handleScale()}>
              {t('common.confirm')}
            </Button>
          </Space>
        </Space>
      </AppModal>
      <AppDrawer
        visible={Boolean(podsTarget)}
        title={t('business.k8s.workload.podsTitle', { name: podsTarget?.name ?? '' })}
        onCancel={() => setPodsTarget(null)}
        size="lg"
        footer={null}
      >
        {podsLoading ? <PageLoading /> : null}
        {!podsLoading && pods.length === 0 ? (
          <PageEmpty description={t('business.k8s.workload.podsEmpty')} />
        ) : null}
        {!podsLoading && pods.length > 0 ? (
          <AppTable rowKey="name" columns={podColumns} data={pods} pagination={false} scroll={{ x: 'max-content' }} />
        ) : null}
      </AppDrawer>
      <AppDrawer
        visible={Boolean(logTarget)}
        title={t('business.k8s.workload.log.title', { name: logTarget?.podName ?? '' })}
        onCancel={() => setLogTarget(null)}
        size="xl"
        footer={null}
      >
        {logTarget ? (
          <PodLogViewer
            clusterId={logTarget.clusterId}
            namespace={logTarget.namespace}
            podName={logTarget.podName}
          />
        ) : null}
      </AppDrawer>
    </PageContainer>
  );
}
