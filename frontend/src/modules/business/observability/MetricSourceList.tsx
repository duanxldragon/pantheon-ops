import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Popconfirm, Space, Tag } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete } from '@arco-design/web-react/icon';
import { AppTable, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { apiRequest, usePermission } from '@duanxldragon/pantheon-base-ui';
import '../../../assets/list-page.css';

interface MonitoringTargetRow {
  id: number;
  businessScopeId: number;
  targetType: string;
  targetName: string;
  endpoint: string;
  metricsPath: string;
  enabled: boolean;
  status: string;
  createdAt: string;
  updatedAt: string;
}

interface TargetListResp {
  items: MonitoringTargetRow[];
  total: number;
  page: number;
  pageSize: number;
}

function getTargets() {
  return apiRequest<TargetListResp>({
    url: '/business/observability/targets',
    method: 'get',
    params: { page: 1, pageSize: 100 },
  });
}

function deleteTarget(id: number) {
  return apiRequest<{ deleted: boolean }>({
    url: `/business/observability/targets/${id}`,
    method: 'delete',
  });
}

export default function MetricSourceList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [data, setData] = useState<MonitoringTargetRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const canDelete = hasPerm('business:observability:list');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getTargets();
      setData(result.items);
    } catch (requestError) {
      setError(requestError);
      setData([]);
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

  const handleDelete = useCallback(
    async (row: MonitoringTargetRow) => {
      try {
        await deleteTarget(row.id);
        Message.success(t('common.deleteSuccess'));
        await loadData();
      } catch {
        Message.error(t('common.failed'));
      }
    },
    [loadData, t],
  );

  const columns = useMemo<ColumnProps<MonitoringTargetRow>[]>(
    () => [
      { title: t('observability.metric_source.name'), dataIndex: 'targetName', width: 200 },
      { title: t('observability.metric_source.type'), dataIndex: 'targetType', width: 130 },
      { title: t('observability.metric_source.endpoint'), dataIndex: 'endpoint', width: 240 },
      { title: t('business.observability.metricSource.metricsPath'), dataIndex: 'metricsPath', width: 140 },
      {
        title: t('observability.metric_source.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'up' ? 'green' : 'orange'}>{row.status}</Tag>
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
                title={t('observability.metric_source.delete_confirm')}
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
    [canDelete, handleDelete, t],
  );

  return (
    <PageContainer>
      <Card className="page-panel system-list__table-card">
        {loading ? <PageLoading /> : null}
        {error && !loading ? <PageError onRetry={loadData} /> : null}
        {!loading && !error && data.length === 0 ? (
          <PageEmpty description={t('business.observability.metricSource.empty')} />
        ) : null}
        {!loading && !error && data.length > 0 ? (
          <AppTable rowKey="id" columns={columns} data={data} pagination={false} scroll={{ x: 'max-content' }} />
        ) : null}
      </Card>
    </PageContainer>
  );
}
