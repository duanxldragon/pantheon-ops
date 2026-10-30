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
  getK8sBackupList,
  getK8sScheduleList,
  getK8sRestoreList,
  getK8sStorageLocationList,
  getK8sVeleroStatus,
  deleteK8sBackup,
  deleteK8sSchedule,
  deleteK8sRestore,
  type K8sClusterRow,
  type K8sBackupRow,
  type K8sScheduleRow,
  type K8sRestoreRow,
  type K8sStorageLocationRow,
  type K8sVeleroStatus,
} from '../api';
import '../../../../assets/list-page.css';

type BackupTab = 'backups' | 'schedules' | 'restores' | 'locations';

export default function BackupList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [tab, setTab] = useState<BackupTab>('backups');
  const [backupData, setBackupData] = useState<K8sBackupRow[]>([]);
  const [scheduleData, setScheduleData] = useState<K8sScheduleRow[]>([]);
  const [restoreData, setRestoreData] = useState<K8sRestoreRow[]>([]);
  const [locationData, setLocationData] = useState<K8sStorageLocationRow[]>([]);
  const [veleroStatus, setVeleroStatus] = useState<K8sVeleroStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const canDelete = hasPerm('business:k8s:backup:delete');

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
      setBackupData([]);
      setScheduleData([]);
      setRestoreData([]);
      setLocationData([]);
      setVeleroStatus(null);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const [backupResult, scheduleResult, restoreResult, locationResult, veleroResult] =
        await Promise.all([
          getK8sBackupList({ clusterId }),
          getK8sScheduleList({ clusterId }),
          getK8sRestoreList({ clusterId }),
          getK8sStorageLocationList(),
          getK8sVeleroStatus(clusterId),
        ]);
      setBackupData(backupResult.items);
      setScheduleData(scheduleResult.items);
      setRestoreData(restoreResult.items);
      setLocationData(locationResult);
      setVeleroStatus(veleroResult);
    } catch (requestError) {
      setError(requestError);
      setBackupData([]);
      setScheduleData([]);
      setRestoreData([]);
      setLocationData([]);
      setVeleroStatus(null);
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
      { key: 'backup', label: 'Backups', value: backupData.length },
      { key: 'schedule', label: 'Schedules', value: scheduleData.length },
      { key: 'restore', label: 'Restores', value: restoreData.length },
    ],
    [backupData, scheduleData, restoreData],
  );

  const handleDeleteBackup = useCallback(
    async (row: K8sBackupRow) => {
      try {
        await deleteK8sBackup(row.id);
        Message.success(t('business.k8s.backup.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const handleDeleteSchedule = useCallback(
    async (row: K8sScheduleRow) => {
      try {
        await deleteK8sSchedule(row.id);
        Message.success(t('business.k8s.backup.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const handleDeleteRestore = useCallback(
    async (row: K8sRestoreRow) => {
      try {
        await deleteK8sRestore(row.id);
        Message.success(t('business.k8s.backup.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const statusColor = useCallback((status: string) => {
    switch (status) {
      case 'Completed':
        return 'green';
      case 'InProgress':
        return 'blue';
      case 'Failed':
        return 'red';
      default:
        return 'gray';
    }
  }, []);

  const backupColumns = useMemo<ColumnProps<K8sBackupRow>[]>(
    () => [
      { title: t('business.k8s.backup.field.name'), dataIndex: 'name', width: 200 },
      { title: t('business.k8s.backup.field.type'), dataIndex: 'backupType', width: 120 },
      {
        title: t('business.k8s.backup.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.status)}>{row.status}</Tag>
        ),
      },
      { title: 'TTL', dataIndex: 'ttl', width: 80, render: (v: number) => `${v}h` },
      {
        title: 'PVs',
        dataIndex: 'includePvs',
        width: 60,
        render: (v: boolean) => (v ? t('common.yes') : t('common.no')),
      },
      { title: t('business.k8s.backup.field.createdAt'), dataIndex: 'createdAt', width: 180 },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.backup.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteBackup(row)}
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
    [canDelete, handleDeleteBackup, t, statusColor],
  );

  const scheduleColumns = useMemo<ColumnProps<K8sScheduleRow>[]>(
    () => [
      { title: t('business.k8s.backup.field.name'), dataIndex: 'name', width: 200 },
      { title: 'Cron', dataIndex: 'cronExpression', width: 140 },
      {
        title: t('common.status'),
        dataIndex: 'enabled',
        width: 100,
        render: (v: boolean) => (
          <Tag color={v ? 'green' : 'gray'}>{v ? 'Active' : 'Paused'}</Tag>
        ),
      },
      { title: t('business.k8s.backup.lastBackup'), dataIndex: 'lastBackupStatus', width: 120 },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.backup.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteSchedule(row)}
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
    [canDelete, handleDeleteSchedule, t],
  );

  const restoreColumns = useMemo<ColumnProps<K8sRestoreRow>[]>(
    () => [
      { title: t('business.k8s.backup.field.name'), dataIndex: 'name', width: 200 },
      { title: 'Backup', dataIndex: 'backupName', width: 200 },
      {
        title: t('business.k8s.backup.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.status)}>{row.status}</Tag>
        ),
      },
      {
        title: 'PVs',
        dataIndex: 'restorePvs',
        width: 60,
        render: (v: boolean) => (v ? t('common.yes') : t('common.no')),
      },
      { title: t('business.k8s.backup.field.createdAt'), dataIndex: 'createdAt', width: 180 },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.backup.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteRestore(row)}
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
    [canDelete, handleDeleteRestore, t, statusColor],
  );

  const locationColumns = useMemo<ColumnProps<K8sStorageLocationRow>[]>(
    () => [
      { title: t('business.k8s.backup.field.name'), dataIndex: 'name', width: 200 },
      { title: 'Provider', dataIndex: 'provider', width: 100 },
      { title: 'Bucket', dataIndex: 'bucket', width: 200 },
      { title: 'Region', dataIndex: 'region', width: 120 },
      {
        title: 'Default',
        dataIndex: 'isDefault',
        width: 80,
        render: (v: boolean) => (v ? <Tag color="blue">Default</Tag> : null),
      },
    ],
    [t],
  );

  const tabContent = useMemo(() => {
    if (!clusterId)
      return <PageEmpty description={t('business.k8s.backup.selectClusterHint')} />;
    if (loading) return <PageLoading />;
    if (error && !loading) return <PageError onRetry={loadData} />;

    if (tab === 'backups') {
      if (backupData.length === 0)
        return <PageEmpty description={t('business.k8s.backup.empty')} />;
      return (
        <AppTable
          rowKey="id"
          columns={backupColumns}
          data={backupData}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      );
    }
    if (tab === 'schedules') {
      if (scheduleData.length === 0)
        return <PageEmpty description={t('business.k8s.backup.empty')} />;
      return (
        <AppTable
          rowKey="id"
          columns={scheduleColumns}
          data={scheduleData}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      );
    }
    if (tab === 'restores') {
      if (restoreData.length === 0)
        return <PageEmpty description={t('business.k8s.backup.empty')} />;
      return (
        <AppTable
          rowKey="id"
          columns={restoreColumns}
          data={restoreData}
          pagination={false}
          scroll={{ x: 'max-content' }}
        />
      );
    }
    // locations
    if (locationData.length === 0)
      return <PageEmpty description={t('business.k8s.backup.empty')} />;
    return (
      <AppTable
        rowKey="id"
        columns={locationColumns}
        data={locationData}
        pagination={false}
        scroll={{ x: 'max-content' }}
      />
    );
  }, [
    clusterId, loading, error, tab, backupData, scheduleData, restoreData, locationData,
    backupColumns, scheduleColumns, restoreColumns, locationColumns, loadData, t,
  ]);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.backup.hero.eyebrow')}
          title={t('business.k8s.backup.hero.title')}
          description={
            veleroStatus
              ? `${t('business.k8s.backup.veleroStatus')}: ${veleroState(veleroStatus)}`
              : undefined
          }
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
                  setBackupData([]);
                  setScheduleData([]);
                  setRestoreData([]);
                  setLocationData([]);
                  setVeleroStatus(null);
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
                onClick={() => void loadData()}
                disabled={!clusterId || loading}
              >
                {t('business.k8s.storage.sync')}
              </Button>
            </>
          }
        />
        <Card className="page-panel system-list__table-card">
          <Tabs activeTab={tab} onChange={(key) => setTab(key as BackupTab)}>
            <Tabs.TabPane
              key="backups"
              title={`${t('business.k8s.backup.tab.backups')} (${backupData.length})`}
            />
            <Tabs.TabPane
              key="schedules"
              title={`${t('business.k8s.backup.tab.schedules')} (${scheduleData.length})`}
            />
            <Tabs.TabPane
              key="restores"
              title={`${t('business.k8s.backup.tab.restores')} (${restoreData.length})`}
            />
            <Tabs.TabPane
              key="locations"
              title={`${t('business.k8s.backup.tab.locations')} (${locationData.length})`}
            />
          </Tabs>
          <div style={{ marginTop: 16 }}>{tabContent}</div>
        </Card>
      </Space>
    </PageContainer>
  );
}

function veleroState(status: K8sVeleroStatus): string {
  const phase = status.installed ? 'Installed' : 'Not Installed';
  return status.version ? `${phase} (${veleroVersion(status.version)})` : phase;
}

function veleroVersion(image: string): string {
  // Extract version from image tag like velero/velero:v1.12.0
  const parts = image.split(':');
  return parts.length > 1 ? parts[parts.length - 1] : image;
}
