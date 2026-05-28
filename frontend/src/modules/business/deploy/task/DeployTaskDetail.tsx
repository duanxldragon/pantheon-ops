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
  Tag,
  Typography,
  Divider,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconLeft } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import { formatDateTime } from '../../../../core/format/dateTime';
import {
  getDeployTaskDetail,
  markDeployHostResult,
  type DeployTaskHostRow,
  type DeployTaskRow,
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
  connect: 'arcoblue',
  step_start: 'purple',
  precheck: 'gold',
  script: 'cyan',
  postcheck: 'green',
  step_success: 'green',
  step_failed: 'red',
  writeback: 'arcoblue',
  failed: 'red',
};

const stepTypeColorMap: Record<'package' | 'script', string> = {
  package: 'arcoblue',
  script: 'purple',
};

export default function DeployTaskDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [form] = Form.useForm();
  const [task, setTask] = useState<DeployTaskRow | null>(null);
  const [selectedHost, setSelectedHost] = useState<DeployTaskHostRow | null>(null);
  const [visible, setVisible] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown>(null);

  const canMark = hasPerm('business:deploy:task:mark-result');

  const loadData = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      setTask(await getDeployTaskDetail(Number(id)));
    } catch (err) {
      setError(err);
      setTask(null);
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

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

  const openResult = (row: DeployTaskHostRow, status: 'success' | 'failed') => {
    setSelectedHost(row);
    form.setFieldsValue({ status });
    setVisible(true);
  };

  const handleSubmit = async () => {
    if (!selectedHost) return;
    const values = await form.validate();
    setSubmitting(true);
    try {
      await markDeployHostResult(selectedHost.id, values);
      Message.success(t('business.deploy.task.markSuccess'));
      setVisible(false);
      form.resetFields();
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

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
              <Popconfirm title={t('business.deploy.task.markSuccessConfirm')} onOk={() => openResult(row, 'success')}>
                <Button type="text" size="small">{t('business.deploy.task.markSuccessAction')}</Button>
              </Popconfirm>
              <Button type="text" size="small" status="danger" onClick={() => openResult(row, 'failed')}>
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
  if (error || !task) {
    return <PageContainer><PageError description={t('common.loadFailedDesc')} onRetry={loadData} /></PageContainer>;
  }

  return (
    <PageContainer>
      <PageHeader
        title={task.name}
        subtitle={t('operations.deploy.task.detail')}
        extra={<Button icon={<IconLeft />} onClick={() => navigate('/operations/deploy/task')}>{t('common.back')}</Button>}
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero">
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
          {task.hosts?.length ? (
            <AppTable rowKey="id" className="system-list__table" columns={columns} data={task.hosts} pagination={false} />
          ) : (
            <PageEmpty description={t('business.deploy.task.hostEmpty')} />
          )}
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
      <AppModal title={t('business.deploy.task.markResultTitle')} visible={visible} footer={null} onCancel={() => setVisible(false)}>
        <Form form={form} layout="vertical">
          <Form.Item field="status" label={t('business.deploy.task.status')} rules={[{ required: true }]}>
            <Select>
              <Select.Option value="success">{t('business.deploy.task.hostStatus.success')}</Select.Option>
              <Select.Option value="failed">{t('business.deploy.task.hostStatus.failed')}</Select.Option>
              <Select.Option value="skipped">{t('business.deploy.task.hostStatus.skipped')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="stdout" label={t('business.deploy.task.stdout')}>
            <Input.TextArea className="deploy-page__log" autoSize={{ minRows: 2, maxRows: 5 }} />
          </Form.Item>
          <Form.Item field="stderr" label={t('business.deploy.task.stderr')}>
            <Input.TextArea className="deploy-page__log" autoSize={{ minRows: 2, maxRows: 5 }} />
          </Form.Item>
          <Form.Item field="errorMessage" label={t('business.deploy.task.errorMessage')}>
            <Input />
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={() => setVisible(false)} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
