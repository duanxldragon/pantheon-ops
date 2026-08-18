import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Input,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconEye, IconPlus, IconSync } from '@arco-design/web-react/icon';
import {
  AppModal,
  AppTable,
  SearchToolbar,
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  TableBatchActionBar,
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createK8sCluster,
  deleteK8sCluster,
  getK8sClusterList,
  syncK8sCluster,
  updateK8sCluster,
  type K8sClusterListQuery,
  type K8sClusterRow,
} from '../api';
import ClusterForm from './ClusterForm';
import '../../../system/components/shared/list-page.css';

const emptyQuery: K8sClusterListQuery = {
  keyword: '',
  environment: '',
  status: '',
  page: 1,
  pageSize: 20,
};

const statusColor: Record<string, string> = {
  healthy: 'green',
  unreachable: 'red',
  unknown: 'gray',
  degraded: 'orange',
};

export default function ClusterList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm } = usePermission();
  const [data, setData] = useState<K8sClusterRow[]>([]);
  const [query, setQuery] = useState<K8sClusterListQuery>(emptyQuery);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<K8sClusterRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [syncingId, setSyncingId] = useState<number | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);

  const canCreate = hasPerm('business:k8s:cluster:create');
  const canUpdate = hasPerm('business:k8s:cluster:update');
  const canDelete = hasPerm('business:k8s:cluster:delete');
  const canView = hasPerm('business:k8s:cluster:view');

  const loadData = useCallback(async (nextQuery: K8sClusterListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sClusterList(nextQuery);
      setData(result.items);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [query, t]);

  useEffect(() => {
    queueMicrotask(() => {
      loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'total', label: t('business.k8s.cluster.hero.total'), value: total },
      {
        key: 'healthy',
        label: t('business.k8s.cluster.hero.healthy'),
        value: data.filter((item) => item.status === 'healthy').length,
      },
      {
        key: 'nodes',
        label: t('business.k8s.cluster.hero.nodes'),
        value: data.reduce((sum, item) => sum + item.totalNodes, 0),
      },
    ],
    [data, t, total],
  );

  const updateQuery = (patch: Partial<K8sClusterListQuery>) => {
    setSelectedRowKeys([]);
    setQuery((current) => ({ ...current, ...patch, page: 1 }));
  };

  const handleSubmit = async (values: { code: string; name: string; environment: string; businessScopeId?: number; kubeconfig: string; remark?: string }) => {
    setSubmitting(true);
    try {
      if (editing) {
        const payload = {
          name: values.name,
          environment: values.environment,
          businessScopeId: values.businessScopeId,
          remark: values.remark,
        };
        await updateK8sCluster(editing.id, payload);
        Message.success(t('common.updateSuccess'));
      } else {
        await createK8sCluster(values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      await loadData(query);
    } finally {
      setSubmitting(false);
    }
  };

  const handleSync = useCallback(async (row: K8sClusterRow) => {
    setSyncingId(row.id);
    try {
      await syncK8sCluster(row.id);
      Message.success(t('business.k8s.cluster.syncSuccess'));
      await loadData(query);
    } catch {
      Message.error(t('business.k8s.cluster.syncFailed'));
    } finally {
      setSyncingId(null);
    }
  }, [loadData, query, t]);

  const columns = useMemo<ColumnProps<K8sClusterRow>[]>(
    () => [
      {
        title: t('business.k8s.cluster.field.code'),
        dataIndex: 'code',
        width: 140,
        render: (_: unknown, row) => <Tag color="arcoblue">{row.code}</Tag>,
      },
      {
        title: t('business.k8s.cluster.field.name'),
        dataIndex: 'name',
        width: 180,
      },
      {
        title: t('business.k8s.cluster.field.environment'),
        dataIndex: 'environment',
        width: 100,
        render: (_: unknown, row) => t(`business.k8s.cluster.environment.${row.environment}`),
      },
      {
        title: t('business.k8s.cluster.field.businessScope'),
        dataIndex: 'businessScopeName',
        width: 140,
        render: (_: unknown, row) => row.businessScopeName || '-',
      },
      {
        title: t('business.k8s.cluster.field.version'),
        dataIndex: 'version',
        width: 120,
        render: (_: unknown, row) => row.version || '-',
      },
      {
        title: t('business.k8s.cluster.field.nodes'),
        dataIndex: 'totalNodes',
        width: 110,
        render: (_: unknown, row) => `${row.readyNodes}/${row.totalNodes}`,
      },
      {
        title: t('business.k8s.cluster.field.pods'),
        dataIndex: 'totalPods',
        width: 110,
        render: (_: unknown, row) => `${row.runningPods}/${row.totalPods}`,
      },
      {
        title: t('business.k8s.cluster.field.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => (
          <Tag color={statusColor[row.status] || 'gray'}>
            {t(`business.k8s.cluster.status.${row.status}`)}
          </Tag>
        ),
      },
      {
        title: t('business.k8s.cluster.field.lastSyncedAt'),
        dataIndex: 'lastSyncedAt',
        width: 180,
        render: (_: unknown, row) => row.lastSyncedAt || '-',
      },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.wide,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canView ? (
              <Button
                type="text"
                size="small"
                icon={<IconEye />}
                onClick={() => navigate(`/business/k8s/cluster/${row.id}`)}
              >
                {t('common.detail')}
              </Button>
            ) : null}
            {canUpdate ? (
              <Button
                type="text"
                size="small"
                icon={<IconSync />}
                loading={syncingId === row.id}
                onClick={() => handleSync(row)}
              >
                {t('business.k8s.cluster.sync')}
              </Button>
            ) : null}
            {canUpdate ? (
              <Button
                type="text"
                size="small"
                icon={<IconEdit />}
                onClick={() => {
                  setEditing(row);
                  setVisible(true);
                }}
              >
                {t('common.edit')}
              </Button>
            ) : null}
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.cluster.deleteConfirm')}
                onOk={async () => {
                  await deleteK8sCluster(row.id);
                  Message.success(t('common.deleteSuccess'));
                  await loadData(query);
                }}
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
    [canDelete, canUpdate, canView, handleSync, loadData, navigate, query, syncingId, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.cluster.hero.eyebrow')}
          title={t('operations.k8s.cluster.menu')}
          description={t('business.k8s.cluster.hero.title')}
          metrics={heroMetrics}
        />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('business.k8s.cluster.field.keywordPlaceholder')}
          onKeywordChange={(keyword) => updateQuery({ keyword })}
          inlineFilters={
            <>
              <Select
                allowClear
                placeholder={t('business.k8s.cluster.field.environment')}
                value={query.environment || undefined}
                onChange={(environment) => updateQuery({ environment: environment || '' })}
              >
                {['dev', 'test', 'prod'].map((item) => (
                  <Select.Option key={item} value={item}>
                    {t(`business.k8s.cluster.environment.${item}`)}
                  </Select.Option>
                ))}
              </Select>
              <Select
                allowClear
                placeholder={t('business.k8s.cluster.field.status')}
                value={query.status || undefined}
                onChange={(status) => updateQuery({ status: status || '' })}
              >
                {['healthy', 'unreachable', 'unknown', 'degraded'].map((item) => (
                  <Select.Option key={item} value={item}>
                    {t(`business.k8s.cluster.status.${item}`)}
                  </Select.Option>
                ))}
              </Select>
            </>
          }
          advancedFilters={
            <Space direction="vertical" size={12} style={{ width: '100%' }}>
              <Input
                allowClear
                placeholder={t('business.k8s.cluster.field.name')}
                value={query.keyword}
                onChange={(keyword) => updateQuery({ keyword })}
              />
            </Space>
          }
          hasActiveFilters={Boolean(query.keyword || query.environment || query.status)}
          onClearAll={() => {
            setSelectedRowKeys([]);
            setQuery(emptyQuery);
          }}
        />
        <TableBatchActionBar
          selectedCount={selectedRowKeys.length}
          selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
          clearText={t('common.clearSelection')}
          clearSuccessText={t('common.clearSelectionSuccess')}
          onClear={() => setSelectedRowKeys([])}
          prefixActions={
            canCreate ? (
              <ListHeaderActions
                primary={
                  <Button
                    type="primary"
                    icon={<IconPlus />}
                    onClick={() => {
                      setEditing(null);
                      setVisible(true);
                    }}
                  >
                    {t('common.add')}
                  </Button>
                }
              />
            ) : undefined
          }
        />
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? (
            <PageError onRetry={() => loadData(query)} />
          ) : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.cluster.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="id"
              data={data}
              columns={columns}
              scroll={{ x: 'max-content' }}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
                fixed: true,
                onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
              }}
              pagination={buildStandardPagination(t, {
                current: query.page || 1,
                pageSize: query.pageSize || 20,
                total,
                onChange: (page, pageSize) => {
                  const nextQuery = { ...query, page, pageSize };
                  setQuery(nextQuery);
                  loadData(nextQuery);
                },
              })}
            />
          ) : null}
        </Card>
      </Space>
      <AppModal
        visible={visible}
        footer={null}
        title={editing ? t('business.k8s.cluster.editTitle') : t('business.k8s.cluster.createTitle')}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
      >
        <ClusterForm
          mode={editing ? 'update' : 'create'}
          initialValues={editing || undefined}
          submitting={submitting}
          onSubmit={handleSubmit}
          onCancel={() => {
            setVisible(false);
            setEditing(null);
          }}
        />
      </AppModal>
    </PageContainer>
  );
}
