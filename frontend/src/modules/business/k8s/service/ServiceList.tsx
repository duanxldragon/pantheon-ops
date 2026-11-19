import {
  forwardRef,
  memo,
  useCallback,
  useEffect,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useTranslation } from 'react-i18next';
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
import { IconDelete, IconEdit, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
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
import { isRequestError } from '../../../../api/request';
import { resolveFallbackMessageKey } from '../../../../api/requestErrorUtils';
import {
  createK8sService,
  deleteK8sService,
  getK8sClusterList,
  getK8sServiceList,
  updateK8sService,
  type K8sClusterRow,
  type K8sServiceRow,
} from '../api';
import '../../../../assets/list-page.css';
import ServiceEditor from './ServiceEditor';
import { createServiceDraft, serviceDraftPayload } from './serviceDraft';

function formatSelector(selector: Record<string, string> | null) {
  const entries = Object.entries(selector ?? {});
  return entries.length > 0 ? entries.map(([key, value]) => `${key}=${value}`).join(', ') : '-';
}

interface ServiceFeedbackHandle {
  error: (content: string) => void;
}

// Keep one Message hook instance for the lifetime of this modal. Form renders
// must not recreate its notice owner; closing the modal removes only its toast.
const ServiceWriteFeedback = memo(
  forwardRef<ServiceFeedbackHandle>(function ServiceWriteFeedback(_, ref) {
    const [feedback, holder] = Message.useMessage({ maxCount: 1 });
    useImperativeHandle(
      ref,
      () => ({
        error: (content) => {
          // Arco's first ref callback may replay in React StrictMode. The
          // modal-local ID makes that registration update the same notice.
          feedback.error?.({ id: 'c04-service-write-error', content });
        },
      }),
      [feedback],
    );
    return holder;
  }),
);

export default function ServiceList() {
  const { t } = useTranslation();
  const feedback = useRef<ServiceFeedbackHandle>(null);
  const { hasPerm } = usePermission();
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>();
  const [namespace, setNamespace] = useState('');
  const [name, setName] = useState('');
  const [data, setData] = useState<K8sServiceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [editor, setEditor] = useState<{ clusterId: number; row: K8sServiceRow | null } | null>(
    null,
  );
  const [submitting, setSubmitting] = useState(false);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [draft, setDraft] = useState(createServiceDraft);
  const [validation, setValidation] = useState<string | null>(null);

  const canCreate = hasPerm('business:k8s:service:create');
  const canUpdate = hasPerm('business:k8s:service:update');
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
      if (!isRequestError(requestError)) Message.error(t('common.loadFailed'));
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

  const handleSave = useCallback(async () => {
    if (!editor || submitting || !(editor.row ? canUpdate : canCreate)) return;
    const payload = serviceDraftPayload(draft);
    if (typeof payload === 'string') {
      setValidation(payload);
      return;
    }
    setValidation(null);
    setSubmitting(true);
    try {
      if (editor.row) {
        await updateK8sService(
          editor.clusterId,
          editor.row.namespace,
          editor.row.name,
          {
            ...payload,
            type: draft.type,
            uid: editor.row.uid,
            resourceVersion: editor.row.resourceVersion,
          },
          { skipErrorMessage: true },
        );
      } else {
        await createK8sService(
          editor.clusterId,
          draft.namespace.trim(),
          {
            ...payload,
            name: draft.name.trim(),
            type: draft.type,
          },
          { skipErrorMessage: true },
        );
      }
      Message.success(
        t(editor.row ? 'business.k8s.service.updateSuccess' : 'business.k8s.service.createSuccess'),
      );
      setEditor(null);
      await loadData();
    } catch (requestError) {
      // This form owns its error feedback inside the existing modal layer.
      // Preserve the draft and original identity; never fetch a fresh token.
      const fallback = isRequestError(requestError)
        ? resolveFallbackMessageKey(requestError.kind)
        : editor.row
          ? 'business.k8s.service.updateFailed'
          : 'business.k8s.service.createFailed';
      const key = isRequestError(requestError) ? requestError.messageKey : undefined;
      feedback.current?.error(key ? t(key, { defaultValue: t(fallback) }) : t(fallback));
    } finally {
      setSubmitting(false);
    }
  }, [canCreate, canUpdate, draft, editor, loadData, submitting, t]);

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
        render: (_: unknown, row) =>
          row.ports.map((port) => `${port.port}:${port.targetPort}/${port.protocol}`).join(', ') ||
          '-',
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
            {canUpdate ? (
              <Button
                type="text"
                size="small"
                icon={<IconEdit />}
                disabled={
                  !row.uid ||
                  !row.resourceVersion ||
                  !['ClusterIP', 'NodePort', 'LoadBalancer'].includes(row.type) ||
                  deleting !== null
                }
                onClick={() => {
                  if (!clusterId) return;
                  setDraft(createServiceDraft(row));
                  setValidation(null);
                  setEditor({ clusterId, row: { ...row } });
                }}
              >
                {t('common.edit')}
              </Button>
            ) : null}
            {canDelete ? (
              <Popconfirm
                title={t('business.k8s.service.deleteConfirm', { name: row.name })}
                onOk={async () => {
                  if (
                    !clusterId ||
                    deleting !== null ||
                    !canDelete ||
                    !row.uid ||
                    !row.resourceVersion
                  )
                    return;
                  setDeleting(row.uid);
                  try {
                    await deleteK8sService(clusterId, row.namespace, row.name, {
                      uid: row.uid,
                      resourceVersion: row.resourceVersion,
                    });
                    Message.success(t('common.deleteSuccess'));
                    await loadData();
                  } catch (requestError) {
                    if (!isRequestError(requestError)) Message.error(t('common.deleteFailed'));
                  } finally {
                    setDeleting(null);
                  }
                }}
              >
                <Button
                  type="text"
                  size="small"
                  status="danger"
                  icon={<IconDelete />}
                  disabled={!row.uid || !row.resourceVersion || deleting !== null}
                  loading={deleting === row.uid}
                >
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, clusterId, deleting, loadData, t],
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
                disabled={Boolean(editor) || deleting !== null}
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
        <ListHeaderActions
          utility={
            <Button
              type="text"
              icon={<IconRefresh />}
              loading={loading}
              disabled={!clusterId || Boolean(editor) || deleting !== null}
              onClick={() => void loadData()}
            >
              {t('common.refresh')}
            </Button>
          }
          primary={
            canCreate ? (
              <Button
                type="primary"
                icon={<IconPlus />}
                disabled={!clusterId || deleting !== null}
                onClick={() => {
                  if (!clusterId) return;
                  setDraft(createServiceDraft());
                  setValidation(null);
                  setEditor({ clusterId, row: null });
                }}
              >
                {t('business.k8s.service.add')}
              </Button>
            ) : null
          }
        />
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? (
            <PageError onRetry={clusterId ? loadData : loadClusters} />
          ) : null}
          {!loading && !error && !clusterId ? (
            <PageEmpty description={t('business.k8s.service.selectCluster')} />
          ) : null}
          {!loading && !error && clusterId && data.length === 0 ? (
            <PageEmpty description={t('business.k8s.service.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey={(row) => `${row.namespace}/${row.name}/${row.uid}`}
              data={data}
              columns={columns}
              pagination={false}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>

      <AppModal
        visible={Boolean(editor)}
        title={
          <>
            {t(editor?.row ? 'business.k8s.service.editSelector' : 'business.k8s.service.add')}
            <ServiceWriteFeedback ref={feedback} />
          </>
        }
        confirmLoading={submitting}
        okButtonProps={{ disabled: !editor || !(editor.row ? canUpdate : canCreate) }}
        cancelButtonProps={{ disabled: submitting }}
        closable={!submitting}
        escToExit={!submitting}
        onOk={() => void handleSave()}
        onCancel={() => {
          if (!submitting) setEditor(null);
        }}
      >
        <ServiceEditor
          draft={draft}
          editing={editor?.row ?? null}
          disabled={submitting}
          validation={validation}
          onChange={(next) => {
            setDraft(next);
            setValidation(null);
          }}
        />
      </AppModal>
    </PageContainer>
  );
}
