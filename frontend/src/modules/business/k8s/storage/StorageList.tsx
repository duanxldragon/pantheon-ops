import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Message,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
} from '@arco-design/web-react';
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
  getK8sPVList,
  getK8sPVCList,
  getK8sStorageClassList,
  syncK8sPVs,
  syncK8sPVCs,
  syncK8sStorageClasses,
  deleteK8sPV,
  deleteK8sPVC,
  type K8sClusterRow,
  type K8sPVRow,
  type K8sPVCRow,
  type K8sStorageClassRow,
} from '../api';
import '../../../../assets/list-page.css';

type StorageTab = 'pv' | 'pvc' | 'storageclass';

export default function StorageList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [tab, setTab] = useState<StorageTab>('pv');
  const [pvData, setPvData] = useState<K8sPVRow[]>([]);
  const [pvcData, setPvcData] = useState<K8sPVCRow[]>([]);
  const [scData, setScData] = useState<K8sStorageClassRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const canDelete = hasPerm('business:k8s:storage:delete');

  const loadClusters = useCallback(async () => {
    try {
      const result = await getK8sClusterList({ page: 1, pageSize: 100 });
      setClusters(result.items);
    } catch {
      // best-effort
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadClusters();
    });
  }, [loadClusters]);

  const loadData = useCallback(async () => {
    if (!clusterId) {
      setPvData([]);
      setPvcData([]);
      setScData([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [pvResult, pvcResult, scResult] = await Promise.all([
        getK8sPVList({ clusterId }),
        getK8sPVCList({ clusterId }),
        getK8sStorageClassList({ clusterId }),
      ]);
      setPvData(pvResult.items);
      setPvcData(pvcResult.items);
      setScData(scResult.items);
    } catch (requestError) {
      setError(requestError);
      setPvData([]);
      setPvcData([]);
      setScData([]);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [clusterId, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'pv', label: 'PV', value: pvData.length },
      { key: 'pvc', label: 'PVC', value: pvcData.length },
      { key: 'sc', label: 'StorageClass', value: scData.length },
    ],
    [pvData, pvcData, scData],
  );

  const handleSync = useCallback(
    async (target: StorageTab) => {
      if (!clusterId) return;
      try {
        if (target === 'pv') await syncK8sPVs(clusterId);
        else if (target === 'pvc') await syncK8sPVCs(clusterId);
        else await syncK8sStorageClasses(clusterId);
        Message.success(t('business.k8s.storage.syncSuccess'));
        await loadData();
      } catch {
        Message.error(t('business.k8s.storage.syncFailed'));
      }
    },
    [clusterId, loadData, t],
  );

  const handleDeletePV = useCallback(
    async (row: K8sPVRow) => {
      try {
        await deleteK8sPV(row.id);
        Message.success(t('business.k8s.storage.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('business.k8s.storage.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const handleDeletePVC = useCallback(
    async (row: K8sPVCRow) => {
      try {
        await deleteK8sPVC(row.id);
        Message.success(t('business.k8s.storage.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('business.k8s.storage.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const pvColumns = useMemo<ColumnProps<K8sPVRow>[]>(
    () => [
      { title: t('business.k8s.storage.pv.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.storage.pv.field.capacity'), dataIndex: 'capacity', width: 120 },
      { title: t('business.k8s.storage.pv.field.storageClass'), dataIndex: 'storageClass', width: 140 },
      {
        title: t('business.k8s.storage.pv.field.accessModes'),
        dataIndex: 'accessModes',
        render: (_: unknown, row) => (
          <Space size={4}>
            {(row.accessModes || []).map((m) => <Tag key={m} color="gray">{m}</Tag>)}
          </Space>
        ),
      },
      { title: t('business.k8s.storage.pv.field.reclaimPolicy'), dataIndex: 'reclaimPolicy', width: 130 },
      {
        title: t('business.k8s.storage.pv.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'Bound' ? 'green' : 'gray'}>{row.status}</Tag>
        ),
      },
      {
        title: t('common.action'),
        width: 140,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.storage.pv.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeletePV(row)}
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
    [canDelete, handleDeletePV, t],
  );

  const pvcColumns = useMemo<ColumnProps<K8sPVCRow>[]>(
    () => [
      { title: t('business.k8s.storage.pvc.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.storage.pvc.field.namespace'), dataIndex: 'namespace', width: 160 },
      { title: t('business.k8s.storage.pvc.field.pvName'), dataIndex: 'pvName', width: 160 },
      { title: t('business.k8s.storage.pvc.field.requestStorage'), dataIndex: 'requestStorage', width: 130 },
      { title: t('business.k8s.storage.pvc.field.storageClass'), dataIndex: 'storageClass', width: 140 },
      {
        title: t('business.k8s.storage.pvc.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'Bound' ? 'green' : 'gray'}>{row.status}</Tag>
        ),
      },
      {
        title: t('common.action'),
        width: 140,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.storage.pvc.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeletePVC(row)}
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
    [canDelete, handleDeletePVC, t],
  );

  const scColumns = useMemo<ColumnProps<K8sStorageClassRow>[]>(
    () => [
      { title: t('business.k8s.storage.sc.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.storage.sc.field.provisioner'), dataIndex: 'provisioner', width: 200 },
      { title: t('business.k8s.storage.sc.field.reclaimPolicy'), dataIndex: 'reclaimPolicy', width: 140 },
      { title: t('business.k8s.storage.sc.field.volumeBindingMode'), dataIndex: 'volumeBindingMode', width: 180 },
      {
        title: t('business.k8s.storage.sc.field.allowVolumeExpansion'),
        dataIndex: 'allowVolumeExpansion',
        width: 160,
        render: (_: unknown, row) => (
          <Tag color={row.allowVolumeExpansion ? 'green' : 'gray'}>
            {row.allowVolumeExpansion ? t('common.yes') : t('common.no')}
          </Tag>
        ),
      },
      {
        title: t('business.k8s.storage.sc.field.isDefault'),
        dataIndex: 'isDefault',
        width: 100,
        render: (_: unknown, row) => (
          <Tag color={row.isDefault ? 'blue' : undefined}>
            {row.isDefault ? t('common.yes') : t('common.no')}
          </Tag>
        ),
      },
    ],
    [t],
  );

  const tabContent = useMemo(() => {
    if (!clusterId) return <PageEmpty description={t('business.k8s.storage.selectClusterHint')} />;
    if (loading) return <PageLoading />;
    if (error && !loading) return <PageError onRetry={loadData} />;

    if (tab === 'pv') {
      if (pvData.length === 0) return <PageEmpty description={t('business.k8s.storage.pv.empty')} />;
      return <AppTable rowKey="id" columns={pvColumns} data={pvData} pagination={false} scroll={{ x: 'max-content' }} />;
    }
    if (tab === 'pvc') {
      if (pvcData.length === 0) return <PageEmpty description={t('business.k8s.storage.pvc.empty')} />;
      return <AppTable rowKey="id" columns={pvcColumns} data={pvcData} pagination={false} scroll={{ x: 'max-content' }} />;
    }
    // storageclass
    if (scData.length === 0) return <PageEmpty description={t('business.k8s.storage.sc.empty')} />;
    return <AppTable rowKey="id" columns={scColumns} data={scData} pagination={false} scroll={{ x: 'max-content' }} />;
  }, [clusterId, loading, error, tab, pvData, pvcData, scData, pvColumns, pvcColumns, scColumns, loadData, t]);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.storage.hero.eyebrow')}
          title={t('operations.k8s.storage.menu')}
          description={t('business.k8s.storage.hero.title')}
          metrics={heroMetrics}
        />
        <SearchToolbar
          onClearAll={() => setClusterId(undefined)}
          inlineFilters={
            <>
              <Select
                placeholder={t('business.k8s.storage.field.cluster')}
                value={clusterId}
                onChange={(value) => {
                  setClusterId(value);
                  setPvData([]);
                  setPvcData([]);
                  setScData([]);
                }}
                style={{ width: 220 }}
              >
                {clusters.map((cluster) => (
                  <Select.Option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </Select.Option>
                ))}
              </Select>
              <Button
                icon={<IconSync />}
                onClick={() => void handleSync(tab)}
                disabled={!clusterId || loading}
              >
                {t('business.k8s.storage.sync')}
              </Button>
            </>
          }
        />
        <Card className="page-panel system-list__table-card">
          <Tabs activeTab={tab} onChange={(key) => setTab(key as StorageTab)}>
            <Tabs.TabPane key="pv" title={`PV (${pvData.length})`} />
            <Tabs.TabPane key="pvc" title={`PVC (${pvcData.length})`} />
            <Tabs.TabPane key="storageclass" title={`StorageClass (${scData.length})`} />
          </Tabs>
          <div style={{ marginTop: 16 }}>{tabContent}</div>
        </Card>
      </Space>
    </PageContainer>
  );
}
