import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Space, Tag } from '@arco-design/web-react';
import { IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { AppTable, PageContainer, PageEmpty, PageError, PageLoading } from '../../../components';
import { getServiceInstanceList, reconcileServiceInstance, type ServiceInstanceRow } from './api';
import '../../../assets/list-page.css';

export default function ServiceInstanceList() {
  const { t } = useTranslation();
  const [items, setItems] = useState<ServiceInstanceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getServiceInstanceList({ page: 1, pageSize: 100 });
      setItems(response.items);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => { void load(); }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [load]);

  const columns = useMemo<ColumnProps<ServiceInstanceRow>[]>(() => [
    { title: t('business.service.instance.service'), dataIndex: 'serviceName', width: 180 },
    { title: t('business.service.instance.environment'), dataIndex: 'environment', width: 120 },
    { title: t('business.service.instance.target'), dataIndex: 'targetType', width: 110 },
    { title: t('business.service.instance.desired'), dataIndex: 'desiredState', width: 120, render: (_: unknown, row) => <Tag>{row.desiredState}</Tag> },
    { title: t('business.service.instance.observed'), dataIndex: 'observedState', width: 120, render: (_: unknown, row) => <Tag color={row.observedState === 'running' ? 'green' : 'orange'}>{row.observedState}</Tag> },
    { title: t('business.service.instance.health'), dataIndex: 'healthState', width: 120, render: (_: unknown, row) => <Tag color={row.healthState === 'healthy' ? 'green' : row.healthState === 'unhealthy' ? 'red' : 'gray'}>{row.healthState}</Tag> },
    { title: t('business.service.instance.version'), dataIndex: 'currentVersion', width: 120 },
    {
      title: t('business.service.instance.actions'),
      width: 80,
      render: (_: unknown, row) => (
        <Button
          type="text"
          icon={<IconRefresh />}
          aria-label={t('business.service.instance.reconcile')}
          onClick={async () => {
            try {
              await reconcileServiceInstance(row.id);
              Message.success(t('business.serviceinstance.syncTriggered'));
              await load();
            } catch {
              Message.error(t('business.serviceinstance.syncFailed'));
            }
          }}
        />
      ),
    },
  ], [load, t]);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {error && !loading ? <PageError onRetry={() => void load()} /> : null}
          {!loading && !error && items.length === 0 ? <PageEmpty description={t('business.serviceinstance.error')} /> : null}
          {!loading && !error && items.length > 0 ? <AppTable rowKey="id" columns={columns} data={items} pagination={false} scroll={{ x: 'max-content' }} /> : null}
        </Card>
      </Space>
    </PageContainer>
  );
}
