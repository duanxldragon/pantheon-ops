import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Input,
  InputNumber,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconPlus } from '@arco-design/web-react/icon';
import {
  AppModal,
  AppTable,
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
  TABLE_ACTION_COLUMN_WIDTH,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createK8sService,
  deleteK8sService,
  getK8sClusterList,
  getK8sServiceList,
  type K8sClusterRow,
  type K8sServiceRow,
} from '../api';
import '../../../system/components/shared/list-page.css';

interface ServiceDraft {
  name: string;
  namespace: string;
  type: string;
  port: number;
  targetPort: number;
  protocol: string;
  selector: string;
}

const emptyDraft: ServiceDraft = {
  name: '',
  namespace: 'default',
  type: 'ClusterIP',
  port: 80,
  targetPort: 8080,
  protocol: 'TCP',
  selector: '',
};

function parseSelector(value: string) {
  return value
    .split(',')
    .map((entry) => entry.trim())
    .filter(Boolean)
    .reduce<Record<string, string>>((selector, entry) => {
      const separator = entry.indexOf('=');
      if (separator > 0 && separator < entry.length - 1) {
        selector[entry.slice(0, separator).trim()] = entry.slice(separator + 1).trim();
      }
      return selector;
    }, {});
}

function formatSelector(selector: Record<string, string>) {
  const entries = Object.entries(selector);
  return entries.length > 0 ? entries.map(([key, value]) => `${key}=${value}`).join(', ') : '-';
}

export default function ServiceList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [namespace, setNamespace] = useState('');
  const [name, setName] = useState('');
  const [data, setData] = useState<K8sServiceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [draft, setDraft] = useState<ServiceDraft>(emptyDraft);

  const canCreate = hasPerm('business:k8s:service:create');
  const canDelete = hasPerm('business:k8s:service:delete');

  const loadClusters = useCallback(async () => {
    try {
      const result = await getK8sClusterList({ page: 1, pageSize: 100 });
      setClusters(result.items);
      setClusterId((current) => current ?? result.items[0]?.id);
    } catch (requestError) {
      setError(requestError);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadClusters();
    });
  }, [loadClusters]);

  const loadData = useCallback(async () => {
    if (!clusterId) {
      setData([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sServiceList(clusterId, {
        namespace: namespace || undefined,
        name: name || undefined,
      });
      setData(result.items);
    } catch (requestError) {
      setError(requestError);
      setData([]);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [clusterId, name, namespace, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const metrics = useMemo(
    () => [
      { key: 'total', label: t('business.k8s.service.hero.total'), value: data.length },
      {
        key: 'load-balancer',
        label: t('business.k8s.service.hero.loadBalancer'),
        value: data.filter((item) => item.type === 'LoadBalancer').length,
      },
    ],
    [data, t],
  );

  const handleCreate = useCallback(async () => {
    if (!clusterId || !draft.name.trim() || !draft.namespace.trim()) {
      return;
    }
    const selector = parseSelector(draft.selector);
    if (Object.keys(selector).length === 0) {
      Message.warning(t('business.k8s.service.selectorRequired'));
      return;
    }
    setSubmitting(true);
    try {
      await createK8sService(clusterId, draft.namespace.trim(), {
        name: draft.name.trim(),
        type: draft.type,
        ports: [{
          port: draft.port,
          targetPort: draft.targetPort,
          protocol: draft.protocol,
        }],
        selector,
      });
      Message.success(t('business.k8s.service.createSuccess'));
      setCreateOpen(false);
      setDraft(emptyDraft);
      await loadData();
    } catch {
      Message.error(t('business.k8s.service.createFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [clusterId, draft, loadData, t]);

  const columns = useMemo<ColumnProps<K8sServiceRow>[]>(
    () => [
      { title: t('business.k8s.service.field.name'), dataIndex: 'name', width: 180 },
      { title: t('business.k8s.service.field.namespace'), dataIndex: 'namespace', width: 140 },
      {
        title: t('business.k8s.service.field.type'),
        dataIndex: 'type',
        width: 130,
        render: (_: unknown, row) => <Tag color="arcoblue">{row.type}</Tag>,
      },
      {
        title: t('business.k8s.service.field.clusterIP'),
        dataIndex: 'clusterIP',
        width: 150,
        render: (_: unknown, row) => row.clusterIP || '-',
      },
      {
        title: t('business.k8s.service.field.ports'),
        dataIndex: 'ports',
        width: 210,
        render: (_: unknown, row) => row.ports
          .map((port) => `${port.port}:${port.targetPort}/${port.protocol}`)
          .join(', ') || '-',
      },
      {
        title: t('business.k8s.service.field.selector'),
        dataIndex: 'selector',
        render: (_: unknown, row) => formatSelector(row.selector),
      },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.compact,
        fixed: 'right',
        render: (_: unknown, row) => canDelete ? (
          <Popconfirm
            title={t('business.k8s.service.deleteConfirm', { name: row.name })}
            onOk={async () => {
              if (!clusterId) return;
              await deleteK8sService(clusterId, row.namespace, row.name);
              Message.success(t('common.deleteSuccess'));
              await loadData();
            }}
          >
            <Button type="text" size="small" status="danger" icon={<IconDelete />}>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        ) : null,
      },
    ],
    [canDelete, clusterId, loadData, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.k8s.service.hero.eyebrow')}
          title={t('operations.k8s.service.menu')}
          description={t('business.k8s.service.hero.title')}
          metrics={metrics}
        />
        <SearchToolbar
          keyword={name}
          keywordPlaceholder={t('business.k8s.service.field.name')}
          onKeywordChange={setName}
          inlineFilters={
            <>
              <Select
                placeholder={t('business.k8s.service.field.cluster')}
                value={clusterId}
                onChange={(value) => {
                  setClusterId(value);
                  setData([]);
                }}
                style={{ width: 220 }}
              >
                {clusters.map((cluster) => (
                  <Select.Option key={cluster.id} value={cluster.id}>
                    {cluster.name}
                  </Select.Option>
                ))}
              </Select>
              <Input
                allowClear
                value={namespace}
                placeholder={t('business.k8s.service.field.namespace')}
                onChange={setNamespace}
                style={{ width: 180 }}
              />
            </>
          }
          hasActiveFilters={Boolean(name || namespace)}
          onClearAll={() => {
            setName('');
            setNamespace('');
          }}
        />
        {canCreate ? (
          <ListHeaderActions
            primary={
              <Button
                type="primary"
                icon={<IconPlus />}
                disabled={!clusterId}
                onClick={() => setCreateOpen(true)}
              >
                {t('business.k8s.service.add')}
              </Button>
            }
          />
        ) : null}
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? <PageError onRetry={clusterId ? loadData : loadClusters} /> : null}
          {!loading && !error && !clusterId ? (
            <PageEmpty description={t('business.k8s.service.selectCluster')} />
          ) : null}
          {!loading && !error && clusterId && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.service.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="name"
              data={data}
              columns={columns}
              pagination={false}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>

      <AppModal
        visible={createOpen}
        title={t('business.k8s.service.add')}
        confirmLoading={submitting}
        onOk={() => void handleCreate()}
        onCancel={() => {
          setCreateOpen(false);
          setDraft(emptyDraft);
        }}
      >
        <Space direction="vertical" size={14} style={{ width: '100%' }}>
          <div>
            <Typography.Text>{t('business.k8s.service.field.name')}</Typography.Text>
            <Input
              value={draft.name}
              onChange={(value) => setDraft((current) => ({ ...current, name: value }))}
            />
          </div>
          <div>
            <Typography.Text>{t('business.k8s.service.field.namespace')}</Typography.Text>
            <Input
              value={draft.namespace}
              onChange={(value) => setDraft((current) => ({ ...current, namespace: value }))}
            />
          </div>
          <div>
            <Typography.Text>{t('business.k8s.service.field.type')}</Typography.Text>
            <Select
              value={draft.type}
              onChange={(value) => setDraft((current) => ({ ...current, type: value }))}
              style={{ width: '100%' }}
            >
              {['ClusterIP', 'NodePort', 'LoadBalancer'].map((type) => (
                <Select.Option key={type} value={type}>{type}</Select.Option>
              ))}
            </Select>
          </div>
          <Space size={12} style={{ width: '100%' }}>
            <div style={{ flex: 1 }}>
              <Typography.Text>{t('business.k8s.service.field.port')}</Typography.Text>
              <InputNumber
                min={1}
                max={65535}
                value={draft.port}
                onChange={(value) => setDraft((current) => ({ ...current, port: Number(value) }))}
                style={{ width: '100%' }}
              />
            </div>
            <div style={{ flex: 1 }}>
              <Typography.Text>{t('business.k8s.service.field.targetPort')}</Typography.Text>
              <InputNumber
                min={1}
                max={65535}
                value={draft.targetPort}
                onChange={(value) => setDraft((current) => ({ ...current, targetPort: Number(value) }))}
                style={{ width: '100%' }}
              />
            </div>
          </Space>
          <div>
            <Typography.Text>{t('business.k8s.service.field.protocol')}</Typography.Text>
            <Select
              value={draft.protocol}
              onChange={(value) => setDraft((current) => ({ ...current, protocol: value }))}
              style={{ width: '100%' }}
            >
              <Select.Option value="TCP">TCP</Select.Option>
              <Select.Option value="UDP">UDP</Select.Option>
            </Select>
          </div>
          <div>
            <Typography.Text>{t('business.k8s.service.field.selector')}</Typography.Text>
            <Input
              value={draft.selector}
              placeholder="app=example, component=api"
              onChange={(value) => setDraft((current) => ({ ...current, selector: value }))}
            />
          </div>
        </Space>
      </AppModal>
    </PageContainer>
  );
}
