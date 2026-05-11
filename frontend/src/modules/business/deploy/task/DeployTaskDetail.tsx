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
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconLeft } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import {
  getDeployTaskDetail,
  markDeployHostResult,
  type DeployTaskHostRow,
  type DeployTaskRow,
} from '../api';
import '../../../../core/styles/list-page.css';
import '../deploy.css';

const statusColorMap: Record<string, string> = {
  pending: 'gray',
  running: 'arcoblue',
  success: 'green',
  failed: 'red',
  skipped: 'orange',
  canceled: 'orange',
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
    void loadData();
  }, [loadData]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.task.hostTotal'), value: task?.hosts?.length || 0 },
      { key: 'success', label: t('business.deploy.task.status.success'), value: task?.hosts?.filter((item) => item.status === 'success').length || 0 },
      { key: 'failed', label: t('business.deploy.task.status.failed'), value: task?.hosts?.filter((item) => item.status === 'failed').length || 0 },
    ],
    [task, t],
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
          <Typography.Text className="system-page-hero__eyebrow">{t('business.deploy.task.hero.eyebrow')}</Typography.Text>
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
              { label: t('business.deploy.task.package'), value: `${task.packageName} ${task.packageVersion}` },
              { label: t('business.deploy.task.status'), value: t(`business.deploy.task.status.${task.status}`) },
              { label: t('business.deploy.task.targetType'), value: t(`business.deploy.task.targetType.${task.targetType}`) },
              { label: t('business.deploy.task.executorType'), value: t(`business.deploy.task.executorType.${task.executorType}`) },
              { label: t('business.deploy.task.remark'), value: task.remark || '-' },
            ]}
          />
        </Card>
        <Card className="page-panel system-list__table-card">
          {task.hosts?.length ? (
            <AppTable rowKey="id" className="system-list__table" columns={columns} data={task.hosts} pagination={false} />
          ) : (
            <PageEmpty description={t('business.deploy.task.hostEmpty')} />
          )}
        </Card>
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
