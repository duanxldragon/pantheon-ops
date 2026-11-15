import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Message, Popconfirm, Space, Tag } from '@arco-design/web-react';
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { isForbiddenRequestError } from '../../../api/request';
import {
  AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError,
  PageForbidden, PageLoading, SearchToolbar, buildStandardPagination, TABLE_ACTION_COLUMN_WIDTH,
} from '../../../components';
import { deleteServiceInstance, getServiceInstanceList, reconcileServiceInstance, type ServiceInstanceRow } from '../service/api';
import InstanceCreateModal from './InstanceCreateModal';
import '../../../assets/list-page.css';

const emptyQuery = { page: 1, pageSize: 20 };

export default function ServiceInstanceList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const canList = hasPerm('business:service:list');
  const canCreate = hasPerm('business:service:create');
  const canUpdate = hasPerm('business:service:update');
  const canDelete = hasPerm('business:service:delete');
  const [query, setQuery] = useState(emptyQuery);
  const [rows, setRows] = useState<ServiceInstanceRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [visible, setVisible] = useState(false);
  const [reconciling, setReconciling] = useState<number>();

  useEffect(() => {
    const controller = new AbortController();
    if (!canList) return () => controller.abort();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void getServiceInstanceList(query, controller.signal).then((response) => {
        if (controller.signal.aborted) return;
        setRows(response.items);
        setTotal(response.total);
      }).catch((requestError: unknown) => {
        if (!controller.signal.aborted) {
          setRows([]);
          setError(requestError);
        }
      }).finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    });
    return () => controller.abort();
  }, [canList, query, revision]);

  const reload = () => setRevision((value) => value + 1);
  const reconcile = async (row: ServiceInstanceRow) => {
    setReconciling(row.id);
    try {
      await reconcileServiceInstance(row.id);
      Message.success(t('business.service.instance.reconcileSuccess'));
      reload();
    } catch { Message.error(t('business.service.instance.reconcileFailed')); }
    finally { setReconciling(undefined); }
  };
  const columns: ColumnProps<ServiceInstanceRow>[] = [
    { title: t('business.service.instance.service'), dataIndex: 'serviceName', width: 180 },
    { title: t('business.service.instance.environment'), dataIndex: 'environment', width: 120 },
    { title: t('business.service.instance.target'), dataIndex: 'targetType', width: 140, render: (_: unknown, row) => t(`business.service.catalog.runtime.${row.targetType}`, { defaultValue: row.targetType }) },
    { title: t('business.service.catalog.field.binding'), width: 250, render: (_: unknown, row) => row.targetType === 'vm' ? t('business.service.catalog.hostBinding', { id: row.hostId }) : `${row.namespace}/${row.workloadKind}/${row.workloadName}` },
    { title: t('business.service.instance.desired'), dataIndex: 'desiredState', width: 120, render: (_: unknown, row) => <Tag>{row.desiredState}</Tag> },
    { title: t('business.service.instance.observed'), dataIndex: 'observedState', width: 120, render: (_: unknown, row) => <Tag color={row.observedState === 'running' ? 'green' : 'orange'}>{row.observedState}</Tag> },
    { title: t('business.service.instance.health'), dataIndex: 'healthState', width: 120, render: (_: unknown, row) => <Tag color={row.healthState === 'healthy' ? 'green' : 'gray'}>{row.healthState}</Tag> },
    { title: t('business.service.instance.version'), dataIndex: 'currentVersion', width: 140 },
    {
      title: t('common.action'), width: TABLE_ACTION_COLUMN_WIDTH.medium, fixed: 'right',
      render: (_: unknown, row) => <Space className="system-list__actions">
        {canUpdate ? <Button type="text" size="small" icon={<IconRefresh />} loading={reconciling === row.id} disabled={reconciling !== undefined} onClick={() => void reconcile(row)}>{t('business.service.instance.reconcile')}</Button> : null}
        {canDelete ? <Popconfirm title={t('business.service.catalog.instanceDeleteConfirm')} onOk={async () => {
          try { await deleteServiceInstance(row.id); reload(); }
          catch { Message.error(t('business.service.catalog.operationFailed')); }
        }}><Button type="text" size="small" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null}
      </Space>,
    },
  ];
  if (!canList || isForbiddenRequestError(error)) return <PageContainer><PageForbidden /></PageContainer>;

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.service.catalog.hero.eyebrow')} title={t('operations.service.instance.menu')} description={t('business.service.catalog.hero.title')} metrics={[
          { key: 'total', label: t('business.service.instance.total'), value: total },
        ]} />
        <SearchToolbar onClearAll={() => setQuery(emptyQuery)} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => setVisible(true)}>{t('business.service.catalog.addInstance')}</Button> : undefined} />
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {!loading && error ? <PageError onRetry={reload} /> : null}
          {!loading && !error && total === 0 ? <PageEmpty description={t('business.service.instance.empty')} /> : null}
          {!loading && !error && total > 0 ? <AppTable className="system-list__table" rowKey="id" data={rows} columns={columns} scroll={{ x: 'max-content' }} pagination={buildStandardPagination(t, {
            current: query.page, pageSize: query.pageSize, total, onChange: (page, pageSize) => setQuery({ page, pageSize }),
          })} /> : null}
        </Card>
      </Space>
      {visible ? <InstanceCreateModal onClose={() => setVisible(false)} onCreated={() => { setVisible(false); reload(); }} /> : null}
    </PageContainer>
  );
}
