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
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import FilterPanel from '../../../../components/patterns/FilterPanel';
import ListHeaderActions from '../../../../components/patterns/ListHeaderActions';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createDeployPackage,
  deleteDeployPackage,
  getDeployPackageList,
  updateDeployPackage,
  type DeployPackageRow,
} from '../api';
import '../../../system/list-page.css';
import '../deploy.css';

export default function DeployPackageList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [form] = Form.useForm();
  const [data, setData] = useState<DeployPackageRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<DeployPackageRow | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const canCreate = hasPerm('business:deploy:package:create');
  const canUpdate = hasPerm('business:deploy:package:update');
  const canDelete = hasPerm('business:deploy:package:delete');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getDeployPackageList({ page, pageSize, keyword, status });
      setData(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [keyword, page, pageSize, status, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.package.hero.total'), value: total, hint: t('business.deploy.package.hero.totalHint') },
      { key: 'enabled', label: t('business.deploy.package.status.enabled'), value: data.filter((item) => item.status === 'enabled').length, hint: t('business.deploy.package.hero.enabledHint') },
      { key: 'disabled', label: t('business.deploy.package.status.disabled'), value: data.filter((item) => item.status === 'disabled').length, hint: t('business.deploy.package.hero.disabledHint') },
    ],
    [data, t, total],
  );

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue({ status: 'enabled' });
    setVisible(true);
  };

  const openEdit = (row: DeployPackageRow) => {
    setEditing(row);
    form.setFieldsValue(row);
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      if (editing) {
        await updateDeployPackage(editing.id, values);
        Message.success(t('common.updateSuccess'));
      } else {
        await createDeployPackage(values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      form.resetFields();
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: number) => {
    await deleteDeployPackage(id);
    Message.success(t('common.deleteSuccess'));
    await loadData();
  };

  const columns: ColumnProps<DeployPackageRow>[] = [
    { title: t('business.deploy.package.name'), dataIndex: 'name', width: 160 },
    { title: t('business.deploy.package.version'), dataIndex: 'version', width: 120 },
    {
      title: t('business.deploy.package.status'),
      dataIndex: 'status',
      width: 100,
      render: (_: unknown, row) => (
        <Tag color={row.status === 'enabled' ? 'green' : 'gray'}>
          {t(`business.deploy.package.status.${row.status}`)}
        </Tag>
      ),
    },
    { title: t('business.deploy.package.description'), dataIndex: 'description', ellipsis: true },
    {
      title: t('common.action'),
      fixed: 'right',
      width: 150,
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          {canUpdate && (
            <Button type="text" size="small" icon={<IconEdit />} onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          )}
          {canDelete && (
            <Popconfirm title={t('business.deploy.package.deleteConfirm')} onOk={() => handleDelete(row.id)}>
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
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
        title={t('operations.deploy.package.menu')}
        subtitle={t('business.deploy.package.hero.title')}
        extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={openCreate}>{t('common.create')}</Button> : null}
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero">
          <Typography.Text className="system-page-hero__eyebrow">{t('business.deploy.package.hero.eyebrow')}</Typography.Text>
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
            <Form.Item label={t('common.keyword')}>
              <Input value={keyword} onChange={setKeyword} allowClear />
            </Form.Item>
            <Form.Item label={t('business.deploy.package.status')}>
              <Select value={status} onChange={setStatus} allowClear style={{ width: 140 }}>
                <Select.Option value="enabled">{t('business.deploy.package.status.enabled')}</Select.Option>
                <Select.Option value="disabled">{t('business.deploy.package.status.disabled')}</Select.Option>
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
            <PageEmpty description={t('business.deploy.package.empty')} />
          ) : (
            <AppTable
              className="system-list__table"
              rowKey="id"
              loading={loading}
              columns={columns}
              data={data}
              pagination={{
                current: page,
                pageSize,
                total,
                onChange: setPage,
                onPageSizeChange: (size) => {
                  setPageSize(size);
                  setPage(1);
                },
              }}
            />
          )}
        </Card>
      </Space>
      <AppModal
        title={editing ? t('business.deploy.package.editTitle') : t('business.deploy.package.createTitle')}
        visible={visible}
        footer={null}
        onCancel={() => setVisible(false)}
      >
        <Form form={form} layout="vertical">
          <Form.Item field="name" label={t('business.deploy.package.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="version" label={t('business.deploy.package.version')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="status" label={t('business.deploy.package.status')} initialValue="enabled">
            <Select>
              <Select.Option value="enabled">{t('business.deploy.package.status.enabled')}</Select.Option>
              <Select.Option value="disabled">{t('business.deploy.package.status.disabled')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="installCommand" label={t('business.deploy.package.installCommand')}>
            <Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} />
          </Form.Item>
          <Form.Item field="uninstallCommand" label={t('business.deploy.package.uninstallCommand')}>
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </Form.Item>
          <Form.Item field="description" label={t('business.deploy.package.description')}>
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={() => setVisible(false)} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
