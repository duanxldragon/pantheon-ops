import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Message,
  Popconfirm,
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
  getArgoCDServerList,
  getArgoCDApplicationList,
  getArgoCDRepositoryList,
  deleteArgoCDServer,
  deleteArgoCDRepository,
  deleteArgoCDApplication,
  type ArgoCDServerRow,
  type ArgoCDApplicationRow,
  type ArgoCDRepositoryRow,
} from './api';
import ArgoCDPreview from './ArgoCDPreview';
import '../../../../assets/list-page.css';

type ArgoCDTab = 'servers' | 'repositories' | 'applications';

export default function ArgoCDList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [tab, setTab] = useState<ArgoCDTab>('applications');
  const [servers, setServers] = useState<ArgoCDServerRow[]>([]);
  const [applications, setApplications] = useState<ArgoCDApplicationRow[]>([]);
  const [repositories, setRepositories] = useState<ArgoCDRepositoryRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [reviewApplication, setReviewApplication] = useState<ArgoCDApplicationRow | null>(null);

  const canDelete = hasPerm('business:cicd:argocd:delete');
  const canCreate = hasPerm('business:cicd:argocd:create');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [serverResult, appResult, repoResult] = await Promise.all([
        getArgoCDServerList(),
        getArgoCDApplicationList(),
        getArgoCDRepositoryList(),
      ]);
      setServers(serverResult.items);
      setApplications(appResult.items);
      setRepositories(repoResult.items);
    } catch (requestError) {
      setError(requestError);
      setServers([]);
      setApplications([]);
      setRepositories([]);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'servers', label: t('business.cicd.argocd.tab.servers'), value: servers.length },
      { key: 'apps', label: t('business.cicd.argocd.tab.applications'), value: applications.length },
      { key: 'repos', label: t('business.cicd.argocd.tab.repositories'), value: repositories.length },
    ],
    [servers, applications, repositories, t],
  );

  const handleDeleteServer = useCallback(
    async (row: ArgoCDServerRow) => {
      try {
        await deleteArgoCDServer(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const handleDeleteRepo = useCallback(
    async (row: ArgoCDRepositoryRow) => {
      try {
        await deleteArgoCDRepository(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const handleDeleteApp = useCallback(
    async (row: ArgoCDApplicationRow) => {
      try {
        await deleteArgoCDApplication(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.deleteFailed'));
      }
    },
    [loadData, t],
  );

  const statusColor = useCallback((status: string) => {
    switch (status) {
      case 'Synced':
      case 'Healthy':
      case 'connected':
        return 'green';
      case 'OutOfSync':
      case 'Degraded':
        return 'orange';
      case 'Error':
      case 'disconnected':
        return 'red';
      case 'Progressing':
        return 'blue';
      default:
        return 'gray';
    }
  }, []);

  const serverColumns = useMemo<ColumnProps<ArgoCDServerRow>[]>(
    () => [
      { title: t('business.cicd.argocd.field.name'), dataIndex: 'name', width: 200 },
      { title: 'Endpoint', dataIndex: 'endpoint', width: 300, ellipsis: true },
      {
        title: t('business.cicd.argocd.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.status)}>{row.status}</Tag>
        ),
      },
      { title: 'Version', dataIndex: 'version', width: 140 },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.cicd.argocd.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteServer(row)}
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
    [canDelete, handleDeleteServer, t, statusColor],
  );

  const repoColumns = useMemo<ColumnProps<ArgoCDRepositoryRow>[]>(
    () => [
      { title: t('business.cicd.argocd.field.name'), dataIndex: 'name', width: 200 },
      { title: 'URL', dataIndex: 'repoUrl', width: 300, ellipsis: true },
      { title: 'Branch', dataIndex: 'branch', width: 120 },
      { title: 'Type', dataIndex: 'type', width: 80 },
      {
        title: t('business.cicd.argocd.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.status)}>{row.status}</Tag>
        ),
      },
      {
        title: t('common.action'),
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canDelete ? (
              <Popconfirm
                title={t('business.cicd.argocd.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteRepo(row)}
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
    [canDelete, handleDeleteRepo, t, statusColor],
  );

  const appColumns = useMemo<ColumnProps<ArgoCDApplicationRow>[]>(
    () => [
      { title: t('business.cicd.argocd.field.name'), dataIndex: 'name', width: 200 },
      { title: 'Project', dataIndex: 'project', width: 100 },
      { title: 'Source', dataIndex: 'sourceRepoUrl', width: 240, ellipsis: true },
      { title: 'Path', dataIndex: 'sourcePath', width: 140, ellipsis: true },
      { title: 'Rev', dataIndex: 'sourceTargetRev', width: 80 },
      { title: 'Namespace', dataIndex: 'destinationNamespace', width: 120 },
      {
        title: t('business.cicd.argocd.field.status'),
        dataIndex: 'status',
        width: 100,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.status)}>{row.status}</Tag>
        ),
      },
      {
        title: 'Health',
        dataIndex: 'healthStatus',
        width: 100,
        render: (_: unknown, row) => (
          <Tag color={statusColor(row.healthStatus)}>{row.healthStatus}</Tag>
        ),
      },
      {
        title: t('common.action'),
        width: 160,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canCreate && <Button
              type="text"
              size="small"
              icon={<IconSync />}
              onClick={() => setReviewApplication(row)}
            >
              {t('business.cicd.argocd.preview.review')}
            </Button>}
            {canDelete ? (
              <Popconfirm
                title={t('business.cicd.argocd.deleteConfirm', { name: row.name })}
                onOk={() => void handleDeleteApp(row)}
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
    [canCreate, canDelete, handleDeleteApp, t, statusColor],
  );

  const tabContent = useMemo(() => {
    if (loading) return <PageLoading />;
    if (error && !loading) return <PageError onRetry={loadData} />;

    if (tab === 'servers') {
      if (servers.length === 0) return <PageEmpty description={t('business.cicd.argocd.empty')} />;
      return <AppTable rowKey="id" columns={serverColumns} data={servers} pagination={false} scroll={{ x: 'max-content' }} />;
    }
    if (tab === 'repositories') {
      if (repositories.length === 0) return <PageEmpty description={t('business.cicd.argocd.empty')} />;
      return <AppTable rowKey="id" columns={repoColumns} data={repositories} pagination={false} scroll={{ x: 'max-content' }} />;
    }
    // applications
    if (applications.length === 0) return <PageEmpty description={t('business.cicd.argocd.empty')} />;
    return <AppTable rowKey="id" columns={appColumns} data={applications} pagination={false} scroll={{ x: 'max-content' }} />;
  }, [tab, loading, error, servers, repositories, applications, serverColumns, repoColumns, appColumns, loadData, t]);

  return (
    <PageContainer>
      {reviewApplication && <ArgoCDPreview application={reviewApplication} onClose={() => setReviewApplication(null)} onAccepted={loadData} />}
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.cicd.argocd.hero.eyebrow')}
          title={t('business.cicd.argocd.hero.title')}
          metrics={heroMetrics}
        />
        <SearchToolbar
          onClearAll={() => undefined}
          inlineFilters={
            <Button icon={<IconSync />} onClick={() => void loadData()} disabled={loading}>
              {t('business.k8s.storage.sync')}
            </Button>
          }
        />
        <Card className="page-panel system-list__table-card">
          <Tabs activeTab={tab} onChange={(key) => setTab(key as ArgoCDTab)}>
            <Tabs.TabPane
              key="applications"
              title={`${t('business.cicd.argocd.tab.applications')} (${applications.length})`}
            />
            <Tabs.TabPane
              key="servers"
              title={`${t('business.cicd.argocd.tab.servers')} (${servers.length})`}
            />
            <Tabs.TabPane
              key="repositories"
              title={`${t('business.cicd.argocd.tab.repositories')} (${repositories.length})`}
            />
          </Tabs>
          <div style={{ marginTop: 16 }}>{tabContent}</div>
        </Card>
      </Space>
    </PageContainer>
  );
}
