import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Form,
  Input,
  Message,
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
  SubmitBar,
  buildStandardPagination,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  getK8sClusterList,
  getK8sReleaseList,
  type K8sClusterRow,
  type K8sReleaseListQuery,
  type K8sReleaseRow,
  type K8sReleasePayload,
} from '../api';
import '../../../../assets/list-page.css';

import ReleasePreview from './ReleasePreview';

const FormItem = Form.Item;

const statusColor: Record<string, string> = {
  succeeded: 'green',
  success: 'green',
  failed: 'red',
  pending: 'orange',
  applying: 'arcoblue',
  timed_out: 'orange',
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
  const [reviewIntent, setReviewIntent] = useState<{ operation: 'create'; payload: K8sReleasePayload } | { operation: 'rollback'; releaseId: number } | null>(null);
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
        value: data.filter((item) => item.status === 'succeeded').length,
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
      setReviewIntent({ operation: 'create', payload: { ...values } as K8sReleasePayload });
      setVisible(false);
    } finally {
      setSubmitting(false);
    }
  }, [form]);

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
            {canRollback && row.status === 'succeeded' && row.imageBefore ? (
              <Button type="text" size="small" icon={<IconUndo />} onClick={() => setReviewIntent({ operation: 'rollback', releaseId: row.id })}>
                {t('business.k8s.release.rollback')}
              </Button>
            ) : null}
          </Space>
        ),
      },
    ],
    [canRollback, t],
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
      {reviewIntent && <ReleasePreview intent={reviewIntent} onClose={(returnToForm) => {
        if (returnToForm && reviewIntent.operation === 'create') setVisible(true);
        setReviewIntent(null);
      }} onAccepted={async () => {
        form.resetFields();
        await loadData(query);
      }} />}
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
            submitText={t('business.k8s.release.preview.title')}
            onSubmit={handleSubmit}
          />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
