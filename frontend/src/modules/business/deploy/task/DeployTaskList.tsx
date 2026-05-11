import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
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
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconClose, IconEye, IconPlayArrow, IconPlus } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import FilterPanel from '../../../../components/patterns/FilterPanel';
import ListHeaderActions from '../../../../components/patterns/ListHeaderActions';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import { getHostList, type HostRow } from '../../cmdb/host/api';
import { getGroupList, type GroupRow } from '../../cmdb/group/api';
import {
  cancelDeployTask,
  createDeployTask,
  getDeployPackageList,
  getDeployTaskList,
  startDeployTask,
  type DeployPackageRow,
  type DeployTaskRow,
} from '../api';
import '../../../../core/styles/list-page.css';
import '../deploy.css';

const statusColorMap: Record<string, string> = {
  pending: 'gray',
  running: 'arcoblue',
  success: 'green',
  failed: 'red',
  canceled: 'orange',
};

export default function DeployTaskList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm } = usePermission();
  const [form] = Form.useForm();
  const [data, setData] = useState<DeployTaskRow[]>([]);
  const [packages, setPackages] = useState<DeployPackageRow[]>([]);
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [groups, setGroups] = useState<GroupRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [visible, setVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [targetType, setTargetType] = useState<'host' | 'group'>('host');

  const canCreate = hasPerm('business:deploy:task:create');
  const canDetail = hasPerm('business:deploy:task:detail');
  const canStart = hasPerm('business:deploy:task:start');
  const canCancel = hasPerm('business:deploy:task:cancel');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getDeployTaskList({ page, pageSize, keyword, status });
      setData(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [keyword, page, pageSize, status, t]);

  const loadOptions = useCallback(async () => {
    const [packageResp, hostResp, groupResp] = await Promise.all([
      getDeployPackageList({ page: 1, pageSize: 100, status: 'enabled' }),
      getHostList({ page: 1, pageSize: 100, status: 'online' }),
      getGroupList(),
    ]);
    setPackages(packageResp.items);
    setHosts(hostResp.items);
    setGroups(flattenGroups(groupResp));
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.task.hero.total'), value: total, hint: t('business.deploy.task.hero.totalHint') },
      { key: 'running', label: t('business.deploy.task.status.running'), value: data.filter((item) => item.status === 'running').length, hint: t('business.deploy.task.hero.runningHint') },
      { key: 'success', label: t('business.deploy.task.status.success'), value: data.filter((item) => item.status === 'success').length, hint: t('business.deploy.task.hero.successHint') },
    ],
    [data, t, total],
  );

  const openCreate = async () => {
    setTargetType('host');
    form.setFieldsValue({ targetType: 'host', executorType: 'manual' });
    await loadOptions();
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      await createDeployTask(values);
      Message.success(t('common.createSuccess'));
      setVisible(false);
      form.resetFields();
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleStart = async (id: number) => {
    await startDeployTask(id);
    Message.success(t('business.deploy.task.startSuccess'));
    await loadData();
  };

  const handleCancel = async (id: number) => {
    await cancelDeployTask(id);
    Message.success(t('business.deploy.task.cancelSuccess'));
    await loadData();
  };

  const columns: ColumnProps<DeployTaskRow>[] = [
    { title: t('business.deploy.task.name'), dataIndex: 'name', width: 180 },
    {
      title: t('business.deploy.task.package'),
      dataIndex: 'packageName',
      width: 160,
      render: (_: unknown, row) => `${row.packageName} ${row.packageVersion}`,
    },
    {
      title: t('business.deploy.task.targetType'),
      dataIndex: 'targetType',
      width: 120,
      render: (_: unknown, row) => t(`business.deploy.task.targetType.${row.targetType}`),
    },
    {
      title: t('business.deploy.task.executorType'),
      dataIndex: 'executorType',
      width: 120,
      render: (_: unknown, row) => t(`business.deploy.task.executorType.${row.executorType}`),
    },
    {
      title: t('business.deploy.task.status'),
      dataIndex: 'status',
      width: 120,
      render: (_: unknown, row) => (
        <Tag color={statusColorMap[row.status] || 'gray'}>
          {t(`business.deploy.task.status.${row.status}`)}
        </Tag>
      ),
    },
    {
      title: t('common.action'),
      fixed: 'right',
      width: 260,
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          {canDetail && (
            <Button type="text" size="small" icon={<IconEye />} onClick={() => navigate(`/operations/deploy/task/${row.id}`)}>
              {t('common.detail')}
            </Button>
          )}
          {canStart && row.status === 'pending' && (
            <Popconfirm title={t('business.deploy.task.startConfirm')} onOk={() => handleStart(row.id)}>
              <Button type="text" size="small" icon={<IconPlayArrow />}>{t('business.deploy.task.start')}</Button>
            </Popconfirm>
          )}
          {canCancel && ['pending', 'running'].includes(row.status) && (
            <Popconfirm title={t('business.deploy.task.cancelConfirm')} onOk={() => handleCancel(row.id)}>
              <Button type="text" size="small" status="danger" icon={<IconClose />}>{t('business.deploy.task.cancel')}</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  if (loading && data.length === 0) {
    return <PageContainer><PageLoading /></PageContainer>;
  }
  if (error && data.length === 0) {
    return <PageContainer><PageError description={t('common.loadFailedDesc')} onRetry={loadData} /></PageContainer>;
  }

  return (
    <PageContainer>
      <PageHeader
        title={t('operations.deploy.task.menu')}
        subtitle={t('business.deploy.task.hero.title')}
        extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={openCreate}>{t('common.create')}</Button> : null}
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero">
          <Typography.Text className="system-page-hero__eyebrow">{t('business.deploy.task.hero.eyebrow')}</Typography.Text>
          <div className="deploy-page__hero-grid">
            {heroStats.map((item) => (
              <div key={item.key} className="deploy-page__metric">
                <span className="deploy-page__metric-label">{item.label}</span>
                <span className="deploy-page__metric-value">{item.value}</span>
                <span className="deploy-page__metric-hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>
        <FilterPanel>
          <Form layout="inline">
            <Form.Item label={t('common.keyword')}><Input value={keyword} onChange={setKeyword} allowClear /></Form.Item>
            <Form.Item label={t('business.deploy.task.status')}>
              <Select value={status} onChange={setStatus} allowClear style={{ width: 150 }}>
                {['pending', 'running', 'success', 'failed', 'canceled'].map((item) => (
                  <Select.Option key={item} value={item}>{t(`business.deploy.task.status.${item}`)}</Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" onClick={() => { setPage(1); void loadData(); }}>{t('common.search')}</Button>
                <Button onClick={() => { setKeyword(''); setStatus(''); setPage(1); }}>{t('common.reset')}</Button>
              </Space>
            </Form.Item>
          </Form>
        </FilterPanel>
        <Card className="page-panel system-list__table-card">
          <ListHeaderActions />
          {data.length === 0 ? (
            <PageEmpty description={t('business.deploy.task.empty')} />
          ) : (
            <AppTable
              rowKey="id"
              className="system-list__table"
              loading={loading}
              columns={columns}
              data={data}
              pagination={{ current: page, pageSize, total, onChange: setPage, onPageSizeChange: (size) => { setPageSize(size); setPage(1); } }}
            />
          )}
        </Card>
      </Space>
      <AppModal title={t('business.deploy.task.createTitle')} visible={visible} footer={null} onCancel={() => setVisible(false)}>
        <Form form={form} layout="vertical">
          <Form.Item field="name" label={t('business.deploy.task.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="packageId" label={t('business.deploy.task.package')} rules={[{ required: true }]}>
            <Select>
              {packages.map((item) => (
                <Select.Option key={item.id} value={item.id}>{item.name} {item.version}</Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item field="targetType" label={t('business.deploy.task.targetType')} initialValue="host">
            <Select onChange={(value) => setTargetType(value)}>
              <Select.Option value="host">{t('business.deploy.task.targetType.host')}</Select.Option>
              <Select.Option value="group">{t('business.deploy.task.targetType.group')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="targetIds" label={t('business.deploy.task.targets')} rules={[{ required: true }]}>
            <Select mode="multiple">
              {(targetType === 'host' ? hosts : groups).map((item: any) => (
                <Select.Option key={item.id} value={item.id}>
                  {targetType === 'host' ? `${item.hostname} ${item.ip}` : item.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item field="executorType" label={t('business.deploy.task.executorType')} initialValue="manual">
            <Select>
              <Select.Option value="manual">{t('business.deploy.task.executorType.manual')}</Select.Option>
              <Select.Option value="simulated">{t('business.deploy.task.executorType.simulated')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="remark" label={t('business.deploy.task.remark')}>
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={() => setVisible(false)} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
    </PageContainer>
  );
}

function flattenGroups(groups: GroupRow[]): GroupRow[] {
  return groups.flatMap((group) => [group, ...flattenGroups(group.children || [])]);
}
