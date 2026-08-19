import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  Card,
  Descriptions,
  Form,
  Input,
  Message,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconLeft, IconPlus, IconSync } from '@arco-design/web-react/icon';
import {
  AppModal,
  AppTable,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
} from '../../../../components';
import BusinessPageHeader from '../../shared/BusinessPageHeader';
import FormSection from '../../../../components/patterns/feedback/FormSection';
import { formatDateTime } from '../../../../core/format/dateTime';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createK8sConfigMap,
  createK8sNamespace,
  createK8sSecret,
  deleteK8sConfigMap,
  deleteK8sNamespace,
  deleteK8sSecret,
  getK8sClusterDetail,
  getK8sClusterNodes,
  getK8sConfigMaps,
  getK8sNamespaces,
  getK8sSecrets,
  syncK8sCluster,
  type K8sClusterRow,
  type K8sConfigMapRow,
  type K8sNamespaceRow,
  type K8sNodeRow,
  type K8sSecretRow,
} from '../api';
import '../../../system/components/shared/list-page.css';

const FormItem = Form.Item;

const statusColor: Record<string, string> = {
  healthy: 'green',
  unreachable: 'red',
  unknown: 'gray',
  degraded: 'orange',
};

function parseKeyValue(text: string): Record<string, string> {
  const result: Record<string, string> = {};
  text.split('\n').forEach((line) => {
    const idx = line.indexOf('=');
    if (idx > 0) {
      result[line.slice(0, idx).trim()] = line.slice(idx + 1).trim();
    }
  });
  return result;
}

export default function ClusterDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [detail, setDetail] = useState<K8sClusterRow | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [nodes, setNodes] = useState<K8sNodeRow[]>([]);
  const [nodesLoading, setNodesLoading] = useState(true);
  const [nodesError, setNodesError] = useState<unknown>(null);
  const [namespaces, setNamespaces] = useState<K8sNamespaceRow[]>([]);
  const [namespacesLoading, setNamespacesLoading] = useState(true);
  const [namespacesError, setNamespacesError] = useState<unknown>(null);
  const [configMaps, setConfigMaps] = useState<K8sConfigMapRow[]>([]);
  const [configMapsLoading, setConfigMapsLoading] = useState(false);
  const [configMapsError, setConfigMapsError] = useState<unknown>(null);
  const [secrets, setSecrets] = useState<K8sSecretRow[]>([]);
  const [secretsLoading, setSecretsLoading] = useState(false);
  const [secretsError, setSecretsError] = useState<unknown>(null);
  const [syncing, setSyncing] = useState(false);

  const [namespaceVisible, setNamespaceVisible] = useState(false);
  const [configMapVisible, setConfigMapVisible] = useState(false);
  const [secretVisible, setSecretVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [namespaceForm] = Form.useForm();
  const [configMapForm] = Form.useForm();
  const [secretForm] = Form.useForm();

  const clusterId = id ? Number(id) : 0;
  const canManage = hasPerm('business:k8s:cluster:update');

  const loadDetail = useCallback(async () => {
    if (!clusterId) return;
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sClusterDetail(clusterId);
      setDetail(result);
    } catch (requestError) {
      setError(requestError);
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, [clusterId]);

  const loadNodes = useCallback(async () => {
    if (!clusterId) return;
    setNodesLoading(true);
    setNodesError(null);
    try {
      const result = await getK8sClusterNodes(clusterId);
      setNodes(result.items);
    } catch (requestError) {
      setNodesError(requestError);
      setNodes([]);
    } finally {
      setNodesLoading(false);
    }
  }, [clusterId]);

  const loadNamespaces = useCallback(async () => {
    if (!clusterId) return;
    setNamespacesLoading(true);
    setNamespacesError(null);
    try {
      const result = await getK8sNamespaces(clusterId);
      setNamespaces(result.items);
    } catch (requestError) {
      setNamespacesError(requestError);
      setNamespaces([]);
    } finally {
      setNamespacesLoading(false);
    }
  }, [clusterId]);

  const loadConfigMaps = useCallback(async () => {
    if (!clusterId) return;
    setConfigMapsLoading(true);
    setConfigMapsError(null);
    try {
      const result = await getK8sConfigMaps(clusterId);
      setConfigMaps(result.items);
    } catch (requestError) {
      setConfigMapsError(requestError);
      setConfigMaps([]);
    } finally {
      setConfigMapsLoading(false);
    }
  }, [clusterId]);

  const loadSecrets = useCallback(async () => {
    if (!clusterId) return;
    setSecretsLoading(true);
    setSecretsError(null);
    try {
      const result = await getK8sSecrets(clusterId);
      setSecrets(result.items);
    } catch (requestError) {
      setSecretsError(requestError);
      setSecrets([]);
    } finally {
      setSecretsLoading(false);
    }
  }, [clusterId]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadDetail();
      void loadNodes();
      void loadNamespaces();
      void loadConfigMaps();
      void loadSecrets();
    });
  }, [loadConfigMaps, loadDetail, loadNamespaces, loadNodes, loadSecrets]);

  const handleSync = useCallback(async () => {
    if (!clusterId) return;
    setSyncing(true);
    try {
      await syncK8sCluster(clusterId);
      Message.success(t('business.k8s.cluster.syncSuccess'));
      await loadDetail();
    } catch {
      Message.error(t('business.k8s.cluster.syncFailed'));
    } finally {
      setSyncing(false);
    }
  }, [clusterId, loadDetail, t]);

  const handleCreateNamespace = useCallback(async () => {
    setSubmitting(true);
    try {
      const values = await namespaceForm.validate();
      await createK8sNamespace(clusterId, values);
      Message.success(t('common.createSuccess'));
      setNamespaceVisible(false);
      namespaceForm.resetFields();
      await loadNamespaces();
    } finally {
      setSubmitting(false);
    }
  }, [clusterId, loadNamespaces, namespaceForm, t]);

  const handleCreateConfigMap = useCallback(async () => {
    setSubmitting(true);
    try {
      const values = await configMapForm.validate();
      await createK8sConfigMap(clusterId, values.namespace, {
        name: values.name,
        data: parseKeyValue(values.data || ''),
      });
      Message.success(t('common.createSuccess'));
      setConfigMapVisible(false);
      configMapForm.resetFields();
      await loadConfigMaps();
    } finally {
      setSubmitting(false);
    }
  }, [clusterId, configMapForm, loadConfigMaps, t]);

  const handleCreateSecret = useCallback(async () => {
    setSubmitting(true);
    try {
      const values = await secretForm.validate();
      await createK8sSecret(clusterId, values.namespace, {
        name: values.name,
        type: values.type || 'Opaque',
        data: parseKeyValue(values.data || ''),
      });
      Message.success(t('common.createSuccess'));
      setSecretVisible(false);
      secretForm.resetFields();
      await loadSecrets();
    } finally {
      setSubmitting(false);
    }
  }, [clusterId, loadSecrets, secretForm, t]);

  const nodeColumns = useMemo<ColumnProps<K8sNodeRow>[]>(
    () => [
      { title: t('business.k8s.cluster.node.name'), dataIndex: 'name', width: 200 },
      {
        title: t('business.k8s.cluster.node.status'),
        dataIndex: 'status',
        width: 110,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'ready' ? 'green' : 'orange'}>
            {t(`business.k8s.cluster.node.status.${row.status}`)}
          </Tag>
        ),
      },
      { title: t('business.k8s.cluster.node.ip'), dataIndex: 'internalIp', width: 150 },
      { title: t('business.k8s.cluster.node.os'), dataIndex: 'os', width: 180 },
      { title: t('business.k8s.cluster.node.kubelet'), dataIndex: 'kubeletVersion', width: 130 },
      { title: t('business.k8s.cluster.node.cpu'), dataIndex: 'cpuCapacity', width: 100 },
      { title: t('business.k8s.cluster.node.memory'), dataIndex: 'memoryCapacity', width: 120 },
      { title: t('business.k8s.cluster.node.pods'), dataIndex: 'allocatablePods', width: 100 },
    ],
    [t],
  );

  const namespaceColumns = useMemo<ColumnProps<K8sNamespaceRow>[]>(
    () => {
      const columns: ColumnProps<K8sNamespaceRow>[] = [
        { title: t('business.k8s.namespace.name'), dataIndex: 'name', width: 240 },
        {
          title: t('business.k8s.namespace.status'),
          dataIndex: 'status',
          width: 140,
          render: (_: unknown, row) => (
            <Tag color={row.status === 'Active' ? 'green' : 'gray'}>{row.status}</Tag>
          ),
        },
        { title: t('business.k8s.namespace.createdAt'), dataIndex: 'creationTimestamp', width: 200 },
      ];
      if (canManage) {
        columns.push({
          title: t('common.action'),
          width: 100,
          fixed: 'right',
          render: (_: unknown, row) => (
            <Popconfirm
              title={t('business.k8s.namespace.deleteConfirm', { name: row.name })}
              onOk={async () => {
                await deleteK8sNamespace(clusterId, row.name);
                Message.success(t('common.deleteSuccess'));
                await loadNamespaces();
              }}
            >
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ),
        });
      }
      return columns;
    },
    [canManage, clusterId, loadNamespaces, t],
  );

  const configMapColumns = useMemo<ColumnProps<K8sConfigMapRow>[]>(
    () => {
      const columns: ColumnProps<K8sConfigMapRow>[] = [
        { title: t('business.k8s.configmap.name'), dataIndex: 'name', width: 240 },
        { title: t('business.k8s.configmap.namespace'), dataIndex: 'namespace', width: 160 },
        { title: t('business.k8s.configmap.keyCount'), dataIndex: 'keyCount', width: 100 },
      ];
      if (canManage) {
        columns.push({
          title: t('common.action'),
          width: 100,
          fixed: 'right',
          render: (_: unknown, row) => (
            <Popconfirm
              title={t('business.k8s.configmap.deleteConfirm', { name: row.name })}
              onOk={async () => {
                await deleteK8sConfigMap(clusterId, row.namespace, row.name);
                Message.success(t('common.deleteSuccess'));
                await loadConfigMaps();
              }}
            >
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ),
        });
      }
      return columns;
    },
    [canManage, clusterId, loadConfigMaps, t],
  );

  const secretColumns = useMemo<ColumnProps<K8sSecretRow>[]>(
    () => {
      const columns: ColumnProps<K8sSecretRow>[] = [
        { title: t('business.k8s.secret.name'), dataIndex: 'name', width: 240 },
        { title: t('business.k8s.secret.namespace'), dataIndex: 'namespace', width: 160 },
        { title: t('business.k8s.secret.type'), dataIndex: 'type', width: 140 },
        { title: t('business.k8s.secret.keyCount'), dataIndex: 'keyCount', width: 100 },
      ];
      if (canManage) {
        columns.push({
          title: t('common.action'),
          width: 100,
          fixed: 'right',
          render: (_: unknown, row) => (
            <Popconfirm
              title={t('business.k8s.secret.deleteConfirm', { name: row.name })}
              onOk={async () => {
                await deleteK8sSecret(clusterId, row.namespace, row.name);
                Message.success(t('common.deleteSuccess'));
                await loadSecrets();
              }}
            >
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ),
        });
      }
      return columns;
    },
    [canManage, clusterId, loadSecrets, t],
  );

  if (loading) {
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  }

  if (error || !detail) {
    return (
      <PageContainer>
        <PageError description={t('common.loadFailedDesc')} onRetry={loadDetail} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <BusinessPageHeader
        title={detail.name}
        subtitle={t('business.k8s.cluster.detail')}
        extra={
          <Space>
            <Button icon={<IconSync />} loading={syncing} onClick={() => void handleSync()}>
              {t('business.k8s.cluster.sync')}
            </Button>
            <Button icon={<IconLeft />} onClick={() => navigate('/business/k8s/cluster')}>
              {t('common.back')}
            </Button>
          </Space>
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero cmdb-page__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('business.k8s.cluster.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title cmdb-page__hero-title">
                {detail.name}
              </Typography.Title>
              <Typography.Text type="secondary" className="system-page-hero__desc">
                {detail.apiServer}
              </Typography.Text>
            </div>
          </div>
          <div className="cmdb-page__hero-grid">
            <div className="cmdb-page__hero-metric">
              <span className="cmdb-page__hero-label">{t('business.k8s.cluster.field.status')}</span>
              <span className="cmdb-page__hero-value">
                <Tag color={statusColor[detail.status] || 'gray'}>
                  {t(`business.k8s.cluster.status.${detail.status}`)}
                </Tag>
              </span>
            </div>
            <div className="cmdb-page__hero-metric">
              <span className="cmdb-page__hero-label">{t('business.k8s.cluster.hero.nodes')}</span>
              <span className="cmdb-page__hero-value">{`${detail.readyNodes}/${detail.totalNodes}`}</span>
            </div>
            <div className="cmdb-page__hero-metric">
              <span className="cmdb-page__hero-label">{t('business.k8s.cluster.field.pods')}</span>
              <span className="cmdb-page__hero-value">{`${detail.runningPods}/${detail.totalPods}`}</span>
            </div>
            <div className="cmdb-page__hero-metric">
              <span className="cmdb-page__hero-label">{t('business.k8s.cluster.field.cpu')}</span>
              <span className="cmdb-page__hero-value">{`${detail.cpuCapacity} C`}</span>
            </div>
            <div className="cmdb-page__hero-metric">
              <span className="cmdb-page__hero-label">{t('business.k8s.cluster.field.memory')}</span>
              <span className="cmdb-page__hero-value">{`${detail.memoryCapacity} GB`}</span>
            </div>
          </div>
        </Card>
        <Card className="page-panel">
          <FormSection title={t('business.k8s.cluster.summaryTitle')}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.k8s.cluster.field.code'), value: detail.code },
                { label: t('business.k8s.cluster.field.name'), value: detail.name },
                {
                  label: t('business.k8s.cluster.field.environment'),
                  value: t(`business.k8s.cluster.environment.${detail.environment}`),
                },
                { label: t('business.k8s.cluster.field.businessScope'), value: detail.businessScopeName || '-' },
                { label: t('business.k8s.cluster.field.apiServer'), value: detail.apiServer || '-' },
                { label: t('business.k8s.cluster.field.version'), value: detail.version || '-' },
                { label: t('business.k8s.cluster.field.lastSyncedAt'), value: detail.lastSyncedAt ? formatDateTime(detail.lastSyncedAt) : '-' },
                { label: t('business.k8s.cluster.field.remark'), value: detail.remark || '-' },
              ]}
            />
          </FormSection>
        </Card>
        <Card className="page-panel">
          <Tabs defaultActiveTab="nodes">
            <Tabs.TabPane key="nodes" title={t('business.k8s.cluster.nodesTab')}>
              {nodesLoading ? <PageLoading /> : null}
              {nodesError && !nodesLoading ? (
                <PageError description={t('business.k8s.cluster.nodesLoadFailed')} onRetry={loadNodes} />
              ) : null}
              {!nodesLoading && !nodesError && nodes.length === 0 ? (
                <PageEmpty description={t('business.k8s.cluster.nodesEmpty')} />
              ) : null}
              {!nodesLoading && !nodesError && nodes.length > 0 ? (
                <AppTable rowKey="name" columns={nodeColumns} data={nodes} pagination={false} scroll={{ x: 'max-content' }} />
              ) : null}
            </Tabs.TabPane>
            <Tabs.TabPane key="namespaces" title={t('business.k8s.cluster.namespacesTab')}>
              {canManage ? (
                <Space style={{ marginBottom: 12 }}>
                  <Button type="primary" size="small" icon={<IconPlus />} onClick={() => setNamespaceVisible(true)}>
                    {t('business.k8s.namespace.create')}
                  </Button>
                </Space>
              ) : null}
              {namespacesLoading ? <PageLoading /> : null}
              {namespacesError && !namespacesLoading ? (
                <PageError description={t('business.k8s.cluster.namespacesLoadFailed')} onRetry={loadNamespaces} />
              ) : null}
              {!namespacesLoading && !namespacesError && namespaces.length === 0 ? (
                <PageEmpty description={t('business.k8s.cluster.namespacesEmpty')} />
              ) : null}
              {!namespacesLoading && !namespacesError && namespaces.length > 0 ? (
                <AppTable rowKey="name" columns={namespaceColumns} data={namespaces} pagination={false} scroll={{ x: 'max-content' }} />
              ) : null}
            </Tabs.TabPane>
            <Tabs.TabPane key="configmaps" title={t('business.k8s.cluster.configmapsTab')}>
              {canManage ? (
                <Space style={{ marginBottom: 12 }}>
                  <Button type="primary" size="small" icon={<IconPlus />} onClick={() => setConfigMapVisible(true)}>
                    {t('business.k8s.configmap.create')}
                  </Button>
                </Space>
              ) : null}
              {configMapsLoading ? <PageLoading /> : null}
              {configMapsError && !configMapsLoading ? (
                <PageError description={t('business.k8s.configmap.loadFailed')} onRetry={loadConfigMaps} />
              ) : null}
              {!configMapsLoading && !configMapsError && configMaps.length === 0 ? (
                <PageEmpty description={t('business.k8s.configmap.empty')} />
              ) : null}
              {!configMapsLoading && !configMapsError && configMaps.length > 0 ? (
                <AppTable rowKey={(record) => `${record.namespace}/${record.name}`} columns={configMapColumns} data={configMaps} pagination={false} scroll={{ x: 'max-content' }} />
              ) : null}
            </Tabs.TabPane>
            <Tabs.TabPane key="secrets" title={t('business.k8s.cluster.secretsTab')}>
              {canManage ? (
                <Space style={{ marginBottom: 12 }}>
                  <Button type="primary" size="small" icon={<IconPlus />} onClick={() => setSecretVisible(true)}>
                    {t('business.k8s.secret.create')}
                  </Button>
                </Space>
              ) : null}
              {secretsLoading ? <PageLoading /> : null}
              {secretsError && !secretsLoading ? (
                <PageError description={t('business.k8s.secret.loadFailed')} onRetry={loadSecrets} />
              ) : null}
              {!secretsLoading && !secretsError && secrets.length === 0 ? (
                <PageEmpty description={t('business.k8s.secret.empty')} />
              ) : null}
              {!secretsLoading && !secretsError && secrets.length > 0 ? (
                <AppTable rowKey={(record) => `${record.namespace}/${record.name}`} columns={secretColumns} data={secrets} pagination={false} scroll={{ x: 'max-content' }} />
              ) : null}
            </Tabs.TabPane>
          </Tabs>
        </Card>
      </Space>

      <AppModal
        visible={namespaceVisible}
        footer={null}
        title={t('business.k8s.namespace.createTitle')}
        onCancel={() => setNamespaceVisible(false)}
      >
        <Form form={namespaceForm} layout="vertical">
          <FormItem field="name" label={t('business.k8s.namespace.name')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.namespace.namePlaceholder')} />
          </FormItem>
          <Space>
            <Button onClick={() => setNamespaceVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={submitting} onClick={() => void handleCreateNamespace()}>
              {t('common.create')}
            </Button>
          </Space>
        </Form>
      </AppModal>

      <AppModal
        visible={configMapVisible}
        footer={null}
        title={t('business.k8s.configmap.createTitle')}
        onCancel={() => setConfigMapVisible(false)}
      >
        <Form form={configMapForm} layout="vertical">
          <FormItem field="namespace" label={t('business.k8s.configmap.namespace')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.configmap.namespacePlaceholder')} />
          </FormItem>
          <FormItem field="name" label={t('business.k8s.configmap.name')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.configmap.namePlaceholder')} />
          </FormItem>
          <FormItem field="data" label={t('business.k8s.configmap.data')} extra={t('business.k8s.configmap.dataHint')}>
            <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} placeholder={t('business.k8s.configmap.dataPlaceholder')} />
          </FormItem>
          <Space>
            <Button onClick={() => setConfigMapVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={submitting} onClick={() => void handleCreateConfigMap()}>
              {t('common.create')}
            </Button>
          </Space>
        </Form>
      </AppModal>

      <AppModal
        visible={secretVisible}
        footer={null}
        title={t('business.k8s.secret.createTitle')}
        onCancel={() => setSecretVisible(false)}
      >
        <Form form={secretForm} layout="vertical">
          <FormItem field="namespace" label={t('business.k8s.secret.namespace')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.secret.namespacePlaceholder')} />
          </FormItem>
          <FormItem field="name" label={t('business.k8s.secret.name')} rules={[{ required: true, message: t('common.required') }]}>
            <Input placeholder={t('business.k8s.secret.namePlaceholder')} />
          </FormItem>
          <FormItem field="type" label={t('business.k8s.secret.type')} initialValue="Opaque">
            <Select>
              {['Opaque', 'kubernetes.io/tls', 'kubernetes.io/dockerconfigjson'].map((item) => (
                <Select.Option key={item} value={item}>{item}</Select.Option>
              ))}
            </Select>
          </FormItem>
          <FormItem field="data" label={t('business.k8s.secret.data')} extra={t('business.k8s.secret.dataHint')}>
            <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} placeholder={t('business.k8s.secret.dataPlaceholder')} />
          </FormItem>
          <Space>
            <Button onClick={() => setSecretVisible(false)}>{t('common.cancel')}</Button>
            <Button type="primary" loading={submitting} onClick={() => void handleCreateSecret()}>
              {t('common.create')}
            </Button>
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
}
