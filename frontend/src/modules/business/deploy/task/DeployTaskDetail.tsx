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
  Space,
  Tag,
  Typography,
  Divider,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconLeft } from '@arco-design/web-react/icon';
import { isForbiddenRequestError, isRequestError } from '../../../../api/request';
import { AppModal, PageEmpty, PageForbidden, PageLoading, PageNotFound, PageRequestError } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import { formatDateTime } from '../../../../core/format/dateTime';
import {
  cancelDeployTask,
  deleteDeployTask,
  getDeployTaskDetail,
  markDeployHostResult,
  startDeployTask,
  type DeployTaskHostRow,
  type DeployTaskRow,
  type StartDeployTaskPayload,
} from '../api';
import '../../../system/list-page.css';
import '../deploy.css';

const statusColorMap: Record<string, string> = {
  pending: 'gray',
  running: 'arcoblue',
  success: 'green',
  failed: 'red',
  skipped: 'orange',
  canceled: 'orange',
};

const phaseColorMap: Record<string, string> = {
  start: 'gray',
  connect: 'arcoblue',
  render_failed: 'red',
  step_start: 'purple',
  precheck: 'gold',
  precheck_render_failed: 'red',
  script: 'cyan',
  postcheck: 'green',
  postcheck_render_failed: 'red',
  step_success: 'green',
  step_failed: 'red',
  result: 'blue',
  error: 'red',
  writeback: 'arcoblue',
  failed: 'red',
};

const stepTypeColorMap: Record<'package' | 'script', string> = {
  package: 'arcoblue',
  script: 'purple',
};

type MarkResultFormValues = {
  stdout?: string;
  stderr?: string;
  errorMessage?: string;
};

type DeployTaskStatus = 'draft' | 'pending' | 'running' | 'success' | 'failed' | 'canceled';

const taskStatusFlow: Array<{ key: DeployTaskStatus; kind: 'primary' | 'terminal' | 'cancel' }> = [
  { key: 'draft', kind: 'primary' },
  { key: 'pending', kind: 'primary' },
  { key: 'running', kind: 'primary' },
  { key: 'success', kind: 'terminal' },
  { key: 'failed', kind: 'terminal' },
  { key: 'canceled', kind: 'cancel' },
];

const isTaskNotFoundError = (error: unknown) =>
  isRequestError(error) &&
  (error.status === 404 || error.code === 404 || error.messageKey === 'business.deploy.task.notFound');

export default function DeployTaskDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [form] = Form.useForm<MarkResultFormValues>();
  const [startForm] = Form.useForm<StartDeployTaskPayload>();
  const [task, setTask] = useState<DeployTaskRow | null>(null);
  const [selectedHost, setSelectedHost] = useState<DeployTaskHostRow | null>(null);
  const [visible, setVisible] = useState(false);
  const [startVisible, setStartVisible] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [startSubmitting, setStartSubmitting] = useState(false);
  const [authMode, setAuthMode] = useState<'password' | 'private_key'>('password');
  const [error, setError] = useState<unknown>(null);

  const canMark = hasPerm('business:deploy:task:mark-result');
  const canStart = hasPerm('business:deploy:task:start');
  const canCancel = hasPerm('business:deploy:task:cancel');
  const canUpdate = hasPerm('business:deploy:task:update');
  const canDelete = hasPerm('business:deploy:task:delete');

  const closeResultModal = useCallback(() => {
    setVisible(false);
    setSelectedHost(null);
    form.resetFields();
  }, [form]);

  const closeStartModal = useCallback(() => {
    setStartVisible(false);
    setAuthMode('password');
    startForm.resetFields();
  }, [startForm]);

  const loadData = useCallback(async (options?: { silent?: boolean }) => {
    if (!id) return;
    if (!options?.silent) {
      setLoading(true);
      setError(null);
    }
    try {
      setTask(await getDeployTaskDetail(Number(id)));
    } catch (err) {
      if (!options?.silent) {
        setError(err);
        setTask(null);
      }
    } finally {
      if (!options?.silent) {
        setLoading(false);
      }
    }
  }, [id]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  useEffect(() => {
    if (!task || !['pending', 'running'].includes(task.status)) {
      return;
    }
    const timer = globalThis.setInterval(() => {
      void loadData({ silent: true });
    }, 5000);
    return () => globalThis.clearInterval(timer);
  }, [loadData, task]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.task.hostTotal'), value: task?.hosts?.length || 0 },
      { key: 'success', label: t('business.deploy.task.status.success'), value: task?.hosts?.filter((item) => item.status === 'success').length || 0 },
      { key: 'failed', label: t('business.deploy.task.status.failed'), value: task?.hosts?.filter((item) => item.status === 'failed').length || 0 },
    ],
    [task, t],
  );

  const templateParamItems = useMemo(
    () => Object.entries(task?.templateParams || {}).map(([key, value]) => ({
      label: key,
      value: String(value),
    })),
    [task],
  );

  const hostSummaryItems = useMemo(() => {
    const counts = (task?.hosts || []).reduce<Record<string, number>>((acc, item) => {
      acc[item.status] = (acc[item.status] || 0) + 1;
      return acc;
    }, {});
    return [
      { key: 'pending', label: t('business.deploy.task.hostStatus.pending'), value: counts.pending || 0 },
      { key: 'running', label: t('business.deploy.task.hostStatus.running'), value: counts.running || 0 },
      { key: 'success', label: t('business.deploy.task.hostStatus.success'), value: counts.success || 0 },
      { key: 'failed', label: t('business.deploy.task.hostStatus.failed'), value: counts.failed || 0 },
      { key: 'skipped', label: t('business.deploy.task.hostStatus.skipped'), value: counts.skipped || 0 },
    ];
  }, [task, t]);

  const activeTaskStatus = (task?.status || 'draft') as DeployTaskStatus;
  const taskFlowItems = useMemo(
    () =>
      taskStatusFlow.map((item) => ({
        ...item,
        active: item.key === activeTaskStatus,
        reached:
          item.key === activeTaskStatus ||
          (activeTaskStatus === 'running' && ['draft', 'pending'].includes(item.key)) ||
          (activeTaskStatus === 'success' && ['draft', 'pending', 'running'].includes(item.key)) ||
          (activeTaskStatus === 'failed' && ['draft', 'pending', 'running'].includes(item.key)) ||
          (activeTaskStatus === 'canceled' && ['draft', 'pending', 'running'].includes(item.key)),
      })),
    [activeTaskStatus],
  );

  const canStartTask = Boolean(task && canStart && ['draft', 'pending'].includes(task.status));
  const canCancelTask = Boolean(task && canCancel && ['pending', 'running'].includes(task.status));
  const canEditTask = Boolean(task && canUpdate && ['draft', 'pending'].includes(task.status));
  const canDeleteTask = Boolean(task && canDelete && ['draft', 'pending'].includes(task.status));

  const openResult = (row: DeployTaskHostRow) => {
    setSelectedHost(row);
    form.resetFields();
    form.setFieldsValue({ stdout: '', stderr: '', errorMessage: '' });
    setVisible(true);
  };

  const submitResult = useCallback(
    async (host: DeployTaskHostRow, payload: { status: 'success' | 'failed' | 'skipped'; stdout?: string; stderr?: string; errorMessage?: string }) => {
      setSubmitting(true);
      try {
        await markDeployHostResult(host.id, payload);
        Message.success(t('business.deploy.task.markSuccess'));
        closeResultModal();
        await loadData();
      } finally {
        setSubmitting(false);
      }
    },
    [closeResultModal, loadData, t],
  );

  const handleSubmit = async () => {
    if (!selectedHost) return;
    const values = await form.validate();
    await submitResult(selectedHost, { status: 'failed', ...values });
  };

  const handleStart = useCallback(
    async (payload?: StartDeployTaskPayload) => {
      if (!task) {
        return;
      }
      await startDeployTask(task.id, payload);
      Message.success(t('business.deploy.task.startSuccess'));
      closeStartModal();
      await loadData();
    },
    [closeStartModal, loadData, t, task],
  );

  const openStart = useCallback(() => {
    if (!task) {
      return;
    }
    if (task.executorType !== 'ssh') {
      void handleStart();
      return;
    }
    startForm.resetFields();
    startForm.setFieldsValue({ authMode: 'password' });
    setAuthMode('password');
    setStartVisible(true);
  }, [handleStart, startForm, task]);

  const submitStart = useCallback(async () => {
    const values = await startForm.validate();
    setStartSubmitting(true);
    try {
      await handleStart(values);
    } finally {
      setStartSubmitting(false);
    }
  }, [handleStart, startForm]);

  const handleCancelTask = useCallback(async () => {
    if (!task) {
      return;
    }
    setSubmitting(true);
    try {
      await cancelDeployTask(task.id);
      Message.success(t('business.deploy.task.cancelSuccess'));
      await loadData();
    } finally {
      setSubmitting(false);
    }
  }, [loadData, t, task]);

  const handleDeleteTask = useCallback(async () => {
    if (!task) {
      return;
    }
    setSubmitting(true);
    try {
      await deleteDeployTask(task.id);
      Message.success(t('common.deleteSuccess'));
      navigate('/operations/deploy/task');
    } finally {
      setSubmitting(false);
    }
  }, [navigate, t, task]);

  const columns: ColumnProps<DeployTaskHostRow>[] = [
    { title: t('business.deploy.task.host'), dataIndex: 'hostname', width: 180 },
    { title: t('business.deploy.task.hostIp'), dataIndex: 'hostIp', width: 140 },
    {
      title: t('business.deploy.task.status'),
      dataIndex: 'status',
      width: 120,
      render: (_: unknown, row) => (
        <Tag color={statusColorMap[row.status] || 'gray'}>{t(`business.deploy.task.hostStatus.${row.status}`)}</Tag>
      ),
    },
    {
      title: t('business.deploy.task.startedAt'),
      dataIndex: 'startedAt',
      width: 180,
      render: (_: unknown, row) => row.startedAt ? formatDateTime(row.startedAt) : '-',
    },
    {
      title: t('business.deploy.task.finishedAt'),
      dataIndex: 'finishedAt',
      width: 180,
      render: (_: unknown, row) => row.finishedAt ? formatDateTime(row.finishedAt) : '-',
    },
    {
      title: t('business.deploy.task.duration'),
      dataIndex: 'durationSeconds',
      width: 100,
      render: (_: unknown, row) => row.durationSeconds ? `${row.durationSeconds}s` : '-',
    },
    { title: t('business.deploy.task.errorMessage'), dataIndex: 'errorMessage', ellipsis: true },
    {
      title: t('common.action'),
      fixed: 'right',
      width: 180,
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          {canMark && ['pending', 'running'].includes(row.status) && (
            <>
              <Popconfirm
                title={t('business.deploy.task.markSuccessConfirm')}
                onOk={() => {
                  void submitResult(row, { status: 'success' });
                }}
              >
                <Button type="text" size="small" loading={submitting}>
                  {t('business.deploy.task.markSuccessAction')}
                </Button>
              </Popconfirm>
              <Button type="text" size="small" status="danger" onClick={() => openResult(row)}>
                {t('business.deploy.task.markFailedAction')}
              </Button>
            </>
          )}
        </Space>
      ),
    },
  ];

  const processColumns: ColumnProps<{
    at?: string;
    phase?: string;
    message?: string;
    stepName?: string;
    stepType?: 'package' | 'script';
    packageName?: string;
    action?: string;
  }>[] = [
    {
      title: t('business.deploy.task.step'),
      width: 220,
      render: (_: unknown, row: { stepName?: string; packageName?: string; action?: string }) => (
        <Space direction="vertical" size={2}>
          <span>{row.stepName || row.packageName || '-'}</span>
          <Typography.Text type="secondary">
            {row.action ? t(`business.deploy.task.action.${row.action}`) : '-'}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('business.deploy.template.stepTypeLabel'),
      dataIndex: 'stepType',
      width: 120,
      render: (_: unknown, row) => row.stepType ? (
        <Tag color={stepTypeColorMap[row.stepType] || 'gray'}>
          {t(`business.deploy.template.stepType.${row.stepType}`)}
        </Tag>
      ) : '-',
    },
    {
      title: t('business.deploy.task.phase'),
      dataIndex: 'phase',
      width: 120,
      render: (_: unknown, row) => row.phase ? (
        <Tag color={phaseColorMap[row.phase] || 'gray'}>
          {t(`business.deploy.task.phase.${row.phase}`)}
        </Tag>
      ) : '-',
    },
    {
      title: t('business.deploy.task.startedAt'),
      dataIndex: 'at',
      width: 180,
      render: (_: unknown, row) => row.at ? formatDateTime(row.at) : '-',
    },
    { title: t('business.deploy.task.message'), dataIndex: 'message', ellipsis: true },
  ];

  if (loading) {
    return <PageContainer><PageLoading /></PageContainer>;
  }
  if (error) {
    if (isForbiddenRequestError(error)) {
      return <PageContainer><PageForbidden /></PageContainer>;
    }
    if (isTaskNotFoundError(error)) {
      return <PageContainer><PageNotFound /></PageContainer>;
    }
    return <PageContainer><PageRequestError error={error} onRetry={loadData} description={t('common.loadFailedDesc')} /></PageContainer>;
  }
  if (!task) {
    return <PageContainer><PageNotFound /></PageContainer>;
  }

  return (
    <PageContainer>
      <PageHeader
        title={task.name}
        subtitle={t('operations.deploy.task.detail')}
        extra={(
          <Space wrap>
            <Button icon={<IconLeft />} onClick={() => navigate('/operations/deploy/task')}>{t('common.back')}</Button>
          </Space>
        )}
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">
                {t('business.deploy.task.hero.eyebrow')}
              </span>
              <Typography.Title heading={5} className="system-page-hero__title">
                {task.name}
              </Typography.Title>
              <Typography.Paragraph className="system-page-hero__desc">
                {t('business.deploy.task.detailLead')}
              </Typography.Paragraph>
            </div>
            <Tag color={statusColorMap[task.status] || 'gray'}>
              {t(`business.deploy.task.status.${task.status}`)}
            </Tag>
          </div>
          <div className="deploy-page__hero-grid">
            {heroStats.map((item) => (
              <div key={item.key} className="deploy-page__metric">
                <span className="deploy-page__metric-label">{item.label}</span>
                <span className="deploy-page__metric-value">{item.value}</span>
              </div>
            ))}
          </div>
        </Card>
        <Card className="page-panel">
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Typography.Text style={{ fontWeight: 600 }}>
              {t('business.deploy.task.statusFlowTitle')}
            </Typography.Text>
            <div className="deploy-page__status-flow" role="list" aria-label={t('business.deploy.task.statusFlowTitle')}>
              {taskFlowItems.map((item) => (
                <div
                  key={item.key}
                  role="listitem"
                  className={[
                    'deploy-page__status-step',
                    item.active ? 'is-active' : '',
                    item.reached ? 'is-reached' : '',
                    item.kind === 'terminal' ? 'is-terminal' : '',
                    item.kind === 'cancel' ? 'is-cancel' : '',
                  ].filter(Boolean).join(' ')}
                >
                  <span className="deploy-page__status-dot" />
                  <span className="deploy-page__status-label">{t(`business.deploy.task.status.${item.key}`)}</span>
                </div>
              ))}
            </div>
            <Typography.Text type="secondary">
              {t('business.deploy.task.statusFlowHint')}
            </Typography.Text>
          </Space>
        </Card>
        <Card className="page-panel">
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <Typography.Text style={{ fontWeight: 600 }}>
              {t('business.deploy.task.taskActionsTitle')}
            </Typography.Text>
            <div className="deploy-page__action-panel">
              <div className="deploy-page__action-copy">
                <Typography.Text>{t('business.deploy.task.taskActionsHint')}</Typography.Text>
                <Typography.Text type="secondary">
                  {task.executorType === 'ssh'
                    ? t('business.deploy.task.startSshTemplateHint')
                    : t('business.deploy.task.statusFlowHint')}
                </Typography.Text>
              </div>
              <Space wrap>
                {canEditTask ? (
                  <Button
                    icon={<IconEdit />}
                    onClick={() => navigate(`/operations/deploy/task?editId=${task.id}`)}
                  >
                    {t('common.edit')}
                  </Button>
                ) : null}
                {canStartTask ? (
                  task.executorType === 'ssh' ? (
                    <Button type="primary" loading={startSubmitting} onClick={() => openStart()}>
                      {t('business.deploy.task.start')}
                    </Button>
                  ) : (
                    <Popconfirm title={t('business.deploy.task.startConfirm')} onOk={() => openStart()}>
                      <Button type="primary" loading={startSubmitting}>
                        {t('business.deploy.task.start')}
                      </Button>
                    </Popconfirm>
                  )
                ) : null}
                {canCancelTask ? (
                  <Popconfirm title={t('business.deploy.task.cancelConfirm')} onOk={() => void handleCancelTask()}>
                    <Button status="danger" loading={submitting}>
                      {t('business.deploy.task.cancel')}
                    </Button>
                  </Popconfirm>
                ) : null}
                {canDeleteTask ? (
                  <Popconfirm title={t('business.deploy.task.deleteConfirm')} onOk={() => void handleDeleteTask()}>
                    <Button status="danger" icon={<IconDelete />} loading={submitting}>
                      {t('common.delete')}
                    </Button>
                  </Popconfirm>
                ) : null}
              </Space>
            </div>
          </Space>
        </Card>
        <Card className="page-panel">
          <Descriptions
            column={2}
            data={[
              { label: t('business.deploy.task.source'), value: task.templateName ? `${task.templateName} ${task.templateVersion}` : t('business.deploy.task.sourcePackage') },
              { label: t('business.deploy.task.package'), value: `${task.packageName} ${task.packageVersion}` },
              { label: t('business.deploy.task.action'), value: t(`business.deploy.task.action.${task.action || 'install'}`) },
              { label: t('business.deploy.package.executionMode'), value: t(`business.deploy.package.executionMode.${task.executionMode}`) },
              { label: t('business.deploy.task.templateParams'), value: task.templateParams && Object.keys(task.templateParams).length > 0 ? t('business.deploy.task.templateParams.present') : '-' },
              { label: t('business.deploy.task.status'), value: t(`business.deploy.task.status.${task.status}`) },
              { label: t('business.deploy.task.targetType'), value: t(`business.deploy.task.targetType.${task.targetType}`) },
              { label: t('business.deploy.task.businessScope'), value: task.businessScopeName || '-' },
              { label: t('business.deploy.task.executorType'), value: t(`business.deploy.task.executorType.${task.executorType}`) },
              { label: t('business.deploy.task.externalTaskId'), value: task.externalTaskId || '-' },
              { label: t('business.deploy.task.startedAt'), value: task.startedAt ? formatDateTime(task.startedAt) : '-' },
              { label: t('business.deploy.task.finishedAt'), value: task.finishedAt ? formatDateTime(task.finishedAt) : '-' },
              { label: t('business.deploy.task.duration'), value: task.durationSeconds ? `${task.durationSeconds}s` : '-' },
              { label: t('business.deploy.task.hostCount'), value: task.hostCount || 0 },
              { label: t('business.deploy.task.successCount'), value: task.successCount || 0 },
              { label: t('business.deploy.task.failedCount'), value: task.failedCount || 0 },
              { label: t('business.deploy.task.runningCount'), value: task.runningCount || 0 },
              { label: t('business.deploy.task.skippedCount'), value: task.skippedCount || 0 },
              { label: t('business.deploy.task.remark'), value: task.remark || '-' },
            ]}
          />
        </Card>
        {templateParamItems.length > 0 ? (
          <Card className="page-panel">
            <Descriptions
              column={2}
              title={t('business.deploy.task.templateParams')}
              data={templateParamItems}
            />
          </Card>
        ) : null}
        <Card className="page-panel system-list__table-card">
          <Space direction="vertical" size={12} style={{ width: '100%' }}>
            <div className="deploy-page__section-top">
              <Space direction="vertical" size={2}>
                <Typography.Text style={{ fontWeight: 600 }}>
                  {t('business.deploy.task.hostExecutionTitle')}
                </Typography.Text>
                <Typography.Text type="secondary">
                  {t('business.deploy.task.hostSummaryHint')}
                </Typography.Text>
              </Space>
            </div>
            <div className="deploy-page__host-summary">
              {hostSummaryItems.map((item) => (
                <div key={item.key} className="deploy-page__host-summary-item">
                  <span className="deploy-page__host-summary-label">{item.label}</span>
                  <span className="deploy-page__host-summary-value">{item.value}</span>
                </div>
              ))}
            </div>
            {task.hosts?.length ? (
              <AppTable rowKey="id" className="system-list__table" columns={columns} data={task.hosts} pagination={false} />
            ) : (
              <PageEmpty description={t('business.deploy.task.hostEmpty')} />
            )}
          </Space>
        </Card>
        {task.hosts?.length ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            {task.hosts.map((host) => (
              <Card key={host.id} className="page-panel">
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12, width: '100%' }}>
                    <Space direction="vertical" size={2}>
                      <Typography.Text style={{ fontWeight: 600 }}>{host.hostname}</Typography.Text>
                      <Typography.Text type="secondary">
                        {host.hostIp}
                        {host.startedAt ? ` / ${formatDateTime(host.startedAt)}` : ''}
                        {host.durationSeconds ? ` / ${host.durationSeconds}s` : ''}
                      </Typography.Text>
                    </Space>
                    <Tag color={statusColorMap[host.status] || 'gray'}>
                      {t(`business.deploy.task.hostStatus.${host.status}`)}
                    </Tag>
                  </div>
                  <AppTable
                    rowKey={(item) => `${host.id}-${item.phase || ''}-${item.at || ''}`}
                    columns={processColumns}
                    data={host.traceSteps || []}
                    pagination={false}
                    size="small"
                  />
                  <Divider style={{ margin: '12px 0' }} />
                  <Descriptions
                    column={1}
                    data={[
                      {
                        label: t('business.deploy.task.stdout'),
                        value: (
                          <Typography.Paragraph className="deploy-page__log" copyable={Boolean(host.stdout)}>
                            {host.stdout || '-'}
                          </Typography.Paragraph>
                        ),
                      },
                      {
                        label: t('business.deploy.task.stderr'),
                        value: (
                          <Typography.Paragraph className="deploy-page__log" copyable={Boolean(host.stderr)}>
                            {host.stderr || '-'}
                          </Typography.Paragraph>
                        ),
                      },
                    ]}
                  />
                </Space>
              </Card>
            ))}
          </Space>
        ) : null}
      </Space>
      <AppModal
        title={t('business.deploy.task.markFailedAction')}
        visible={visible}
        footer={null}
        onCancel={closeResultModal}
      >
        <Form form={form} layout="vertical">
          <Form.Item field="stdout" label={t('business.deploy.task.stdout')}>
            <Input.TextArea className="deploy-page__log" autoSize={{ minRows: 2, maxRows: 5 }} />
          </Form.Item>
          <Form.Item field="stderr" label={t('business.deploy.task.stderr')}>
            <Input.TextArea className="deploy-page__log" autoSize={{ minRows: 2, maxRows: 5 }} />
          </Form.Item>
          <Form.Item
            field="errorMessage"
            label={t('business.deploy.task.errorMessage')}
            rules={[
              {
                validator: (value, callback) => {
                  if (String(value || '').trim() !== '') {
                    callback();
                    return;
                  }
                  callback(t('business.deploy.taskHost.markFailed.reasonRequired'));
                },
              },
            ]}
          >
            <Input />
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={closeResultModal} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
      <AppModal
        title={t('business.deploy.task.startSshTitle')}
        visible={startVisible}
        footer={null}
        onCancel={closeStartModal}
      >
        <Space direction="vertical" size={12} style={{ width: '100%' }}>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            {t('business.deploy.task.startSshHint')}
          </Typography.Paragraph>
          <Form form={startForm} layout="vertical">
            <Form.Item field="authMode" label={t('business.deploy.task.authMode')} initialValue="password" rules={[{ required: true }]}>
              <Space>
                <Button
                  type={authMode === 'password' ? 'primary' : 'secondary'}
                  onClick={() => {
                    setAuthMode('password');
                    startForm.setFieldValue('authMode', 'password');
                  }}
                >
                  {t('business.deploy.task.authMode.password')}
                </Button>
                <Button
                  type={authMode === 'private_key' ? 'primary' : 'secondary'}
                  onClick={() => {
                    setAuthMode('private_key');
                    startForm.setFieldValue('authMode', 'private_key');
                  }}
                >
                  {t('business.deploy.task.authMode.privateKey')}
                </Button>
              </Space>
            </Form.Item>
            <Form.Item field="sshUser" label={t('business.deploy.task.sshUser')} rules={[{ required: true, message: t('business.deploy.task.sshUserRequired') }]}>
              <Input />
            </Form.Item>
            {authMode === 'private_key' ? (
              <Form.Item field="sshPrivateKey" label={t('business.deploy.task.privateKey')} rules={[{ required: true, message: t('business.deploy.task.sshPrivateKeyRequired') }]}>
                <Input.TextArea autoSize={{ minRows: 4, maxRows: 8 }} />
              </Form.Item>
            ) : (
              <Form.Item field="sshPassword" label={t('business.deploy.task.sshPassword')} rules={[{ required: true, message: t('business.deploy.task.sshPasswordRequired') }]}>
                <Input.Password />
              </Form.Item>
            )}
            <Form.Item field="hostFingerprint" label={t('business.deploy.task.hostFingerprint')} rules={[{ required: true, message: t('business.deploy.task.sshHostKeyRequired') }]}>
              <Input placeholder={t('business.deploy.task.hostFingerprintPlaceholder')} />
            </Form.Item>
            <SubmitBar loading={startSubmitting} onCancel={closeStartModal} onSubmit={submitStart} />
          </Form>
        </Space>
      </AppModal>
    </PageContainer>
  );
}
