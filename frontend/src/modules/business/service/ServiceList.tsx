import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Input, Message, Select, Space, Tag, Tooltip } from '@arco-design/web-react';
import { IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
  buildStandardPagination,
} from '../../../components';
import { getServiceInstanceList, getServiceList, reconcileServiceInstance, type ServiceInstanceRow, type ServiceListQuery, type ServiceRow } from './api';
import '../../../modules/system/components/shared/list-page.css';

const emptyQuery: ServiceListQuery = { keyword: '', status: '', page: 1, pageSize: 20 };

export default function ServiceList() {
  const { t } = useTranslation();
  const [query, setQuery] = useState<ServiceListQuery>(emptyQuery);
  const [items, setItems] = useState<ServiceRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [instances, setInstances] = useState<ServiceInstanceRow[]>([]);
  const [instancesLoading, setInstancesLoading] = useState(false);

  const load = useCallback(async (next: ServiceListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const response = await getServiceList(next);
      setItems(response.items);
      setTotal(response.total);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadInstances = useCallback(async () => {
    setInstancesLoading(true);
    try {
      const response = await getServiceInstanceList({ page: 1, pageSize: 100 });
      setInstances(response.items);
    } catch {
      Message.error(t('business.service.instancesLoadFailed'));
    } finally {
      setInstancesLoading(false);
    }
  }, [t]);

  useEffect(() => {
    void loadInstances();
  }, [loadInstances]);

  const instanceColumns = useMemo<ColumnProps<ServiceInstanceRow>[]>(
    () => [
      { title: t('business.service.instance.service'), dataIndex: 'serviceName', width: 180 },
      { title: t('business.service.instance.environment'), dataIndex: 'environment', width: 110 },
      { title: t('business.service.instance.target'), dataIndex: 'targetType', width: 100 },
      { title: t('business.service.instance.desired'), dataIndex: 'desiredState', width: 110, render: (_: unknown, row) => <Tag color="arcoblue">{row.desiredState}</Tag> },
      { title: t('business.service.instance.observed'), dataIndex: 'observedState', width: 120, render: (_: unknown, row) => <Tag color={row.observedState === 'running' ? 'green' : row.observedState === 'failed' ? 'red' : 'orange'}>{row.observedState}</Tag> },
      { title: t('business.service.instance.health'), dataIndex: 'healthState', width: 110, render: (_: unknown, row) => <Tag color={row.healthState === 'healthy' ? 'green' : row.healthState === 'unhealthy' ? 'red' : 'gray'}>{row.healthState}</Tag> },
      { title: t('business.service.instance.version'), dataIndex: 'currentVersion', width: 120 },
      {
        title: t('business.service.instance.actions'),
        width: 72,
        render: (_: unknown, row) => (
          <Tooltip content={t('business.service.instance.reconcile')}>
            <Button
              type="text"
              size="small"
              icon={<IconRefresh />}
              loading={instancesLoading}
              aria-label={t('business.service.instance.reconcile')}
              onClick={async () => {
                try {
                  await reconcileServiceInstance(row.id);
                  Message.success(t('business.service.instance.reconcileSuccess'));
                  await loadInstances();
                } catch {
                  Message.error(t('business.service.instance.reconcileFailed'));
                }
              }}
            />
          </Tooltip>
        ),
      },
    ],
    [instancesLoading, loadInstances, t],
  );

  const updateQuery = (patch: Partial<ServiceListQuery>) => {
    const next = { ...query, ...patch, page: 1 };
    setQuery(next);
    void load(next);
  };

  const columns = useMemo<ColumnProps<ServiceRow>[]>(
    () => [
      { title: t('business.service.field.code'), dataIndex: 'code', width: 140, render: (_: unknown, row) => <Tag color="arcoblue">{row.code}</Tag> },
      { title: t('business.service.field.name'), dataIndex: 'name', width: 180 },
      { title: t('business.service.field.application'), dataIndex: 'applicationName', width: 180 },
      { title: t('business.service.field.runtimeType'), dataIndex: 'runtimeType', width: 120 },
      { title: t('business.service.field.instances'), dataIndex: 'instanceCount', width: 100 },
      {
        title: t('business.service.field.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => <Tag color={row.status === 'active' ? 'green' : 'gray'}>{t(`business.service.status.${row.status}`)}</Tag>,
      },
    ],
    [t],
  );

  const activeCount = items.filter((item) => item.status === 'active').length;
  const instanceCount = items.reduce((sum, item) => sum + item.instanceCount, 0);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.service.hero.eyebrow')}
          title={t('operations.service.menu')}
          description={t('business.service.hero.title')}
          metrics={[
            { key: 'total', label: t('business.service.hero.total'), value: total },
            { key: 'active', label: t('business.service.hero.active'), value: activeCount },
            { key: 'instances', label: t('business.service.hero.instances'), value: instanceCount },
          ]}
        />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('business.service.field.keywordPlaceholder')}
          onKeywordChange={(keyword) => updateQuery({ keyword })}
          inlineFilters={
            <Select
              allowClear
              placeholder={t('business.service.field.status')}
              value={query.status || undefined}
              onChange={(status) => updateQuery({ status: status || '' })}
            >
              <Select.Option value="active">{t('business.service.status.active')}</Select.Option>
              <Select.Option value="inactive">{t('business.service.status.inactive')}</Select.Option>
            </Select>
          }
          advancedFilters={<Input allowClear placeholder={t('business.service.field.name')} value={query.keyword} onChange={(keyword) => updateQuery({ keyword })} />}
          hasActiveFilters={Boolean(query.keyword || query.status)}
          onClearAll={() => {
            setQuery(emptyQuery);
            void load(emptyQuery);
          }}
        />
        <Card className="page-panel system-list__table-card">
          {loading && items.length === 0 ? <PageLoading /> : null}
          {error && items.length === 0 ? <PageError onRetry={() => void load(query)} /> : null}
          {!loading && !error && items.length === 0 ? <PageEmpty description={t('business.service.empty')} /> : null}
          {!loading && !error && items.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="id"
              data={items}
              columns={columns}
              scroll={{ x: 'max-content' }}
              pagination={buildStandardPagination(t, {
                current: query.page || 1,
                pageSize: query.pageSize || 20,
                total,
                onChange: (page, pageSize) => {
                  const next = { ...query, page, pageSize };
                  setQuery(next);
                  void load(next);
                },
              })}
            />
          ) : null}
        </Card>
        <Card className="page-panel system-list__table-card">
          <Space direction="vertical" size={10} style={{ width: '100%' }}>
            <strong>{t('business.service.instancesTitle')}</strong>
            <AppTable
              className="system-list__table"
              rowKey="id"
              data={instances}
              loading={instancesLoading}
              columns={instanceColumns}
              scroll={{ x: 'max-content' }}
              pagination={false}
            />
          </Space>
        </Card>
      </Space>
    </PageContainer>
  );
}
