import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Input,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconPlus, IconUndo } from '@arco-design/web-react/icon';
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
  buildStandardPagination,
} from '../../../../components';
import { SubmitBar } from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createK8sRelease,
  getK8sClusterList,
  getK8sReleaseList,
  rollbackK8sRelease,
  type K8sClusterRow,
  type K8sReleaseListQuery,
  type K8sReleaseRow,
} from '../api';
import '../../../system/components/shared/list-page.css';

const FormItem = Form.Item;

const statusColor: Record<string, string> = {
  success: 'green',
  failed: 'red',
  pending: 'orange',
  rollback_success: 'cyan',
};

export default function ReleaseList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [data, setData] = useState<K8sReleaseRow[]>([]);
  const [query, setQuery] = useState<K8sReleaseListQuery>({ page: 1, pageSize: 20 });
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [visible, setVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm();

  const canCreate = hasPerm('business:k8s:release:create');
  const canRollback = hasPerm('business:k8s:release:rollback');

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

  const loadData = useCallback(async (nextQuery: K8sReleaseListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sReleaseList(nextQuery);
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
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      { key: 'total', label: t('business.k8s.release.hero.total'), value: total },
      {
        key: 'success',
        label: t('business.k8s.release.hero.success'),
        value: data.filter((item) => item.status === 'success').length,
      },
      {
        key: 'failed',
        label: t('business.k8s.release.hero.failed'),
        value: data.filter((item) => item.status === 'failed').length,
      },
    ],
    [data, t, total],
  );

  const handleSubmit = useCallback(async () => {
    setSubmitting(true);
    try {
      const values = await form.validate();
      await createK8sRelease(values as {
        name: string;
        clusterId: number;
        namespace: string;
        workloadType: string;
        workloadName: string;
        containerName?: string;
        image: string;
        strategy?: string;
      });
      Message.success(t('common.createSuccess'));
      setVisible(false);
      form.resetFields();
      await loadData(query);
    } finally {
      setSubmitting(false);
    }
  }, [form, loadData, query, t]);

  const columns = useMemo<ColumnProps<K8sReleaseRow>[]>(
    () => [
      { title: t('business.k8s.release.field.name'), dataIndex: 'name', width: 180 },
      { title: t('business.k8s.release.field.namespace'), dataIndex: 'namespace', width: 140 },
      { title: t('business.k8s.release.field.workload'), dataIndex: 'workloadName', width: 180 },
      {
        title: t('business.k8s.release.field.imageBefore'),
        dataIndex: 'imageBefore',
        width: 220,
        render: (_: unknown, row) => row.imageBefore || '-',
      },
      {
        title: t('business.k8s.release.field.imageAfter'),
        dataIndex: 'imageAfter',
        width: 220,
        render: (_: unknown, row) => <Tag color="arcoblue">{row.imageAfter}</Tag>,
      },
      {
        title: t('business.k8s.release.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={statusColor[row.status] || 'gray'}>
            {t(`business.k8s.release.status.${row.status}`)}
          </Tag>
        ),
      },
      { title: t('business.k8s.release.field.createdAt'), dataIndex: 'createdAt', width: 180 },
      {
        title: t('common.action'),
        width: 180,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canRollback && row.status === 'success' && row.imageBefore ? (
              <Popconfirm
                title={t('business.k8s.release.rollbackConfirm')}
                onOk={async () => {
                  await rollbackK8sRelease(row.id);
                  Message.success(t('business.k8s.release.rollbackSuccess'));
                  await loadData(query);
                }}
              >
                <Button type="text" size="small" icon={<IconUndo />}>
                  {t('business.k8s.release.rollback')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canRollback, loadData, query, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.release.hero.eyebrow')}
          title={t('operations.k8s.release.menu')}
          description={t('business.k8s.release.hero.title')}
          metrics={heroMetrics}
        />
        {canCreate ? (
          <ListHeaderActions
            primary={
              <Button type="primary" icon={<IconPlus />} onClick={() => setVisible(true)}>
                {t('business.k8s.release.create')}
              </Button>
            }
          />
        ) : null}
        <SearchToolbar
          keyword={query.namespace ?? ''}
          keywordPlaceholder={t('business.k8s.release.field.namespacePlaceholder')}
          onKeywordChange={(namespace) => setQuery((current) => ({ ...current, namespace: namespace || undefined, page: 1 }))}
          inlineFilters={
            <Select
              allowClear
              placeholder={t('business.k8s.release.field.cluster')}
              value={query.clusterId}
              onChange={(clusterId) => setQuery((current) => ({ ...current, clusterId: clusterId || undefined, page: 1 }))}
              style={{ width: 200 }}
            >
              {clusters.map((cluster) => (
                <Select.Option key={cluster.id} value={cluster.id}>
                  {cluster.name}
                </Select.Option>
              ))}
            </Select>
          }
          hasActiveFilters={Boolean(query.namespace || query.clusterId)}
          onClearAll={() => setQuery({ page: 1, pageSize: 20 })}
        />
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? <PageError onRetry={() => loadData(query)} /> : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.release.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              rowKey="id"
              columns={columns}
              data={data}
              scroll={{ x: 'max-content' }}
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
        title={t('business.k8s.release.createTitle')}
        onCancel={() => {
          setVisible(false);
          form.resetFields();
        }}
      >
        <Form form={form} layout="vertical">
          <FormItem field="name" label={t('business.k8s.release.field.name')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.release.field.namePlaceholder')} />
          </FormItem>
          <FormItem field="clusterId" label={t('business.k8s.release.field.cluster')} rules={[{ required: true, message: t('common.required') }]}>
            <Select>
              {clusters.map((cluster) => (
                <Select.Option key={cluster.id} value={cluster.id}>{cluster.name}</Select.Option>
              ))}
            </Select>
          </FormItem>
          <FormItem field="namespace" label={t('business.k8s.release.field.namespace')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.release.field.namespacePlaceholder')} />
          </FormItem>
          <FormItem field="workloadType" label={t('business.k8s.release.field.workloadType')} rules={[{ required: true, message: t('common.required') }]}>
            <Select>
              {['deployment', 'statefulset'].map((item) => (
                <Select.Option key={item} value={item}>{item}</Select.Option>
              ))}
            </Select>
          </FormItem>
          <FormItem field="workloadName" label={t('business.k8s.release.field.workloadName')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.release.field.workloadNamePlaceholder')} />
          </FormItem>
          <FormItem field="containerName" label={t('business.k8s.release.field.containerName')}>
            <Input placeholder={t('business.k8s.release.field.containerNamePlaceholder')} />
          </FormItem>
          <FormItem field="image" label={t('business.k8s.release.field.image')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.release.field.imagePlaceholder')} />
          </FormItem>
          <SubmitBar
            loading={submitting}
            onCancel={() => {
              setVisible(false);
              form.resetFields();
            }}
            submitText={t('common.create')}
            onSubmit={handleSubmit}
          />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
