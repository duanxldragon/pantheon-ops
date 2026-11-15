import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Alert,
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
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
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
import { isForbiddenRequestError } from '../../../../api/request';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createK8sService,
  deleteK8sService,
  updateK8sService,
  getK8sClusterList,
  getK8sServiceList,
  type K8sClusterRow,
  type K8sServiceRow,
} from '../api';
import '../../../../assets/list-page.css';

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

  const [editing, setEditing] = useState<{ clusterId: number; row: K8sServiceRow } | null>(null);
  const [editSelector, setEditSelector] = useState('');
  const [editConflict, setEditConflict] = useState(false);
  const listRequest = useRef<AbortController | null>(null);
  const selectedCluster = useRef(clusterId);
  useEffect(() => {
    selectedCluster.current = clusterId;
  }, [clusterId]);
  const canUpdate = hasPerm('business:k8s:service:update');
  useEffect(() => () => listRequest.current?.abort(), []);

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
    if (selectedCluster.current !== clusterId) return;
    if (!clusterId) {
      setData([]);
      return;
    }
    listRequest.current?.abort();
    const controller = new AbortController();
    listRequest.current = controller;
    setLoading(true);
    setError(null);
    try {
      const result = await getK8sServiceList(clusterId, {
        namespace: namespace || undefined,
        name: name || undefined,
      }, controller.signal);
      if (listRequest.current !== controller || controller.signal.aborted || selectedCluster.current !== clusterId) return;
      setData(result.items);
    } catch (requestError) {
      if (listRequest.current !== controller || controller.signal.aborted || selectedCluster.current !== clusterId) return;
      setError(requestError);
      setData([]);
      Message.error(t('common.loadFailed'));
    } finally {
      if (listRequest.current === controller) { listRequest.current = null; setLoading(false); }
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
      if (selectedCluster.current !== clusterId) return;
      Message.success(t('business.k8s.service.createSuccess'));
      setCreateOpen(false);
      setDraft(emptyDraft);
      await loadData();
    } catch {
      if (selectedCluster.current !== clusterId) return;
      Message.error(t('business.k8s.service.createFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [clusterId, draft, loadData, t]);

  const handleUpdate = useCallback(async () => {
    if (!editing || editing.clusterId !== clusterId || !canUpdate || editConflict) return;
    const selector = parseSelector(editSelector);
    const entries = editSelector.split(',').map((entry) => entry.trim()).filter(Boolean);
    if (!entries.length || entries.length !== Object.keys(selector).length) {
      Message.warning(t('business.k8s.service.selectorRequired')); return;
    }
    setSubmitting(true);
    try {
      await updateK8sService(editing.clusterId, editing.row.namespace, editing.row.name, {
        uid: editing.row.uid, resourceVersion: editing.row.resourceVersion, selector,
      });
      if (selectedCluster.current !== editing.clusterId) return;
      Message.success(t('business.k8s.service.updateSuccess'));
      setEditing(null); await loadData();
    } catch (requestError) {
      if (selectedCluster.current !== editing.clusterId) return;
      if (isForbiddenRequestError(requestError)) { setEditing(null); setData([]); setError(requestError); }
      else if ((requestError as { status?: number }).status === 409) setEditConflict(true);
      else Message.error(t('business.k8s.service.updateFailed'));
    } finally { setSubmitting(false); }
  }, [editing, clusterId, canUpdate, editConflict, editSelector, loadData, t]);

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
          .map((port) => `${port.port}:${port.targetPortName || port.targetPort}/${port.protocol}`)
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
        render: (_: unknown, row) => (
          <Space size={4}>
            {canUpdate && <Button type="text" size="small" icon={<IconEdit />} disabled={submitting || !row.uid || !row.resourceVersion} onClick={() => {
              if (!clusterId) return;
              setEditing({ clusterId, row }); setEditSelector(formatSelector(row.selector)); setEditConflict(false);
            }}>{t('common.edit')}</Button>}
            {canDelete && <Popconfirm title={t('business.k8s.service.deleteConfirm', { name: row.name })} onOk={async () => {
              if (!clusterId || submitting || !row.uid || !row.resourceVersion) return;
              setSubmitting(true);
              try {
                await deleteK8sService(clusterId, row.namespace, row.name, row);
                if (selectedCluster.current !== clusterId) return;
                Message.success(t('common.deleteSuccess')); await loadData();
              } catch (requestError) {
                if (selectedCluster.current !== clusterId) return;
                if (isForbiddenRequestError(requestError)) { setData([]); setEditing(null); setError(requestError); }
                else if ((requestError as { status?: number }).status === 409) Message.warning(t('business.k8s.service.resourceVersionConflict'));
                else Message.error(t('common.deleteFailed'));
              } finally { setSubmitting(false); }
            }}>
              <Button type="text" size="small" status="danger" icon={<IconDelete />} disabled={submitting || !row.uid || !row.resourceVersion}>{t('common.delete')}</Button>
            </Popconfirm>}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, clusterId, loadData, submitting, t],
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
                  listRequest.current?.abort(); listRequest.current = null;
                  setEditing(null); setCreateOpen(false);
                  selectedCluster.current = value;
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
                disabled={!clusterId || submitting}
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

      <AppModal visible={Boolean(editing)} title={t('business.k8s.service.editSelector')} confirmLoading={submitting}
        okButtonProps={{ disabled: !canUpdate || editConflict || !editing || editing.clusterId !== clusterId }}
        onOk={() => void handleUpdate()} onCancel={() => { if (!submitting) setEditing(null); }}>
        <Space direction="vertical" size={14} style={{ width: '100%' }}>
          <Typography.Text>{editing?.row.namespace}/{editing?.row.name} | {t('business.k8s.service.field.resourceVersion')}: {editing?.row.resourceVersion}</Typography.Text>
          {editConflict && <Alert type="warning" content={t('business.k8s.service.resourceVersionConflict')} action={<Button size="small" onClick={() => { setEditing(null); void loadData(); }}>{t('common.refresh')}</Button>} />}
          <label>{t('business.k8s.service.field.selector')}<Input aria-label={t('business.k8s.service.field.selector')} value={editSelector} disabled={submitting || editConflict} onChange={setEditSelector} /></label>
          <Typography.Text type="secondary">{t('business.k8s.service.selectorUpdateHint')}</Typography.Text>
        </Space>
      </AppModal>

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


