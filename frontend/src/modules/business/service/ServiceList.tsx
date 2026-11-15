import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, Message, Popconfirm, Select, Space, Tag } from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { isForbiddenRequestError } from '../../../api/request';
import {
  AppModal, AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError,
  PageForbidden, PageLoading, SearchToolbar, buildStandardPagination, TABLE_ACTION_COLUMN_WIDTH,
} from '../../../components';
import {
  createService, deleteService, getApplicationList, getServiceList, updateService,
  type ApplicationRow, type ServiceListQuery, type ServiceRow,
} from './api';
import '../../../assets/list-page.css';

interface ServiceValues {
  applicationId: number;
  code: string;
  name: string;
  runtimeType: string;
  status: string;
  description?: string;
}
const emptyQuery: ServiceListQuery = { keyword: '', status: '', page: 1, pageSize: 20 };

export default function ServiceList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const canList = hasPerm('business:service:list');
  const canCreate = hasPerm('business:service:create');
  const canUpdate = hasPerm('business:service:update');
  const canDelete = hasPerm('business:service:delete');
  const [query, setQuery] = useState<ServiceListQuery>(emptyQuery);
  const [rows, setRows] = useState<ServiceRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<ServiceRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [applications, setApplications] = useState<ApplicationRow[]>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [optionsError, setOptionsError] = useState<unknown>(null);
  const [form] = Form.useForm<ServiceValues>();

  useEffect(() => {
    const controller = new AbortController();
    if (!canList) return () => controller.abort();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void getServiceList(query, controller.signal).then((response) => {
        if (controller.signal.aborted) return;
        setRows(response.items);
        setTotal(response.total);
      }).catch((requestError: unknown) => {
        if (!controller.signal.aborted) {
          setRows([]);
          setError(requestError);
        }
      }).finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    });
    return () => controller.abort();
  }, [canList, query, revision]);

  const reload = () => setRevision((value) => value + 1);
  const loadApplications = async () => {
    setOptionsLoading(true);
    setOptionsError(null);
    try {
      const response = await getApplicationList({ status: 'active', page: 1, pageSize: 100 });
      setApplications(response.items);
    } catch (requestError) { setOptionsError(requestError); }
    finally { setOptionsLoading(false); }
  };
  const openForm = (row: ServiceRow | null) => {
    setEditing(row);
    setOptionsError(null);
    form.resetFields();
    form.setFieldsValue(row || { runtimeType: 'vm', status: 'active' });
    if (!row) void loadApplications();
    setVisible(true);
  };
  const submit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      if (editing) await updateService(editing.id, { name: values.name, runtimeType: values.runtimeType, status: values.status, description: values.description });
      else await createService(values);
      Message.success(t(editing ? 'common.updateSuccess' : 'business.service.catalog.serviceCreateSuccess'));
      setVisible(false);
      reload();
    } catch { Message.error(t('business.service.catalog.operationFailed')); }
    finally { setSubmitting(false); }
  };
  const columns: ColumnProps<ServiceRow>[] = [
    { title: t('business.service.field.code'), dataIndex: 'code', width: 140, render: (_: unknown, row) => <Tag color="arcoblue">{row.code}</Tag> },
    { title: t('business.service.field.name'), dataIndex: 'name', width: 180 },
    { title: t('business.service.field.application'), dataIndex: 'applicationName', width: 180 },
    { title: t('business.service.field.runtimeType'), dataIndex: 'runtimeType', width: 140, render: (_: unknown, row) => t(`business.service.catalog.runtime.${row.runtimeType}`, { defaultValue: row.runtimeType }) },
    { title: t('business.service.field.instances'), dataIndex: 'instanceCount', width: 100 },
    { title: t('business.service.field.status'), dataIndex: 'status', width: 110, render: (_: unknown, row) => <Tag color={row.status === 'active' ? 'green' : 'gray'}>{t(`business.service.status.${row.status}`)}</Tag> },
    {
      title: t('common.action'), width: TABLE_ACTION_COLUMN_WIDTH.medium, fixed: 'right',
      render: (_: unknown, row) => <Space className="system-list__actions">
        {canUpdate ? <Button type="text" size="small" icon={<IconEdit />} onClick={() => openForm(row)}>{t('common.edit')}</Button> : null}
        {canDelete ? <Popconfirm title={t('business.service.catalog.serviceDeleteConfirm', { name: row.name })} onOk={async () => {
          try { await deleteService(row.id); reload(); }
          catch { Message.error(t('business.service.catalog.operationFailed')); }
        }}><Button type="text" size="small" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null}
      </Space>,
    },
  ];
  const updateQuery = (patch: Partial<ServiceListQuery>) => setQuery((current) => ({ ...current, ...patch, page: 1 }));
  if (!canList || isForbiddenRequestError(error)) return <PageContainer><PageForbidden /></PageContainer>;

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.service.hero.eyebrow')} title={t('operations.service.service.menu')} description={t('business.service.hero.title')} metrics={[
          { key: 'total', label: t('business.service.hero.total'), value: total },
          { key: 'active', label: t('business.service.hero.active'), value: rows.filter((row) => row.status === 'active').length },
          { key: 'instances', label: t('business.service.hero.instances'), value: rows.reduce((sum, row) => sum + row.instanceCount, 0) },
        ]} />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('business.service.field.keywordPlaceholder')}
          onKeywordChange={(keyword) => updateQuery({ keyword })}
          inlineFilters={(
          <Select allowClear placeholder={t('business.service.field.status')} value={query.status || undefined} onChange={(status) => updateQuery({ status: status || '' })}>
            {['active', 'inactive'].map((value) => <Select.Option key={value} value={value}>{t(`business.service.status.${value}`)}</Select.Option>)}
          </Select>
          )}
          hasActiveFilters={Boolean(query.keyword || query.status)}
          onClearAll={() => setQuery(emptyQuery)}
          extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => openForm(null)}>{t('business.service.catalog.addService')}</Button> : undefined}
        />
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {!loading && error ? <PageError onRetry={reload} /> : null}
          {!loading && !error && total === 0 ? <PageEmpty description={t('business.service.empty')} /> : null}
          {!loading && !error && total > 0 ? <AppTable className="system-list__table" rowKey="id" data={rows} columns={columns} scroll={{ x: 'max-content' }} pagination={buildStandardPagination(t, {
            current: query.page ?? 1, pageSize: query.pageSize ?? 20, total,
            onChange: (page, pageSize) => setQuery((current) => ({ ...current, page, pageSize })),
          })} /> : null}
        </Card>
      </Space>
      <AppModal visible={visible} title={t(editing ? 'business.service.catalog.editService' : 'business.service.catalog.addService')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={submitting} okButtonProps={{ disabled: optionsLoading || Boolean(optionsError) }} onOk={submit} onCancel={() => setVisible(false)}>
        {optionsLoading ? <PageLoading /> : null}
        {optionsError ? <PageError onRetry={() => void loadApplications()} /> : null}
        {!optionsLoading && !optionsError ? <Form form={form} layout="vertical">
          {!editing ? <Form.Item field="applicationId" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Select showSearch>{applications.map((application) => <Select.Option key={application.id} value={application.id}>{application.name}</Select.Option>)}</Select></Form.Item> : null}
          <Form.Item field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input disabled={Boolean(editing)} /></Form.Item>
          <Form.Item field="name" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item field="runtimeType" label={t('business.service.catalog.field.runtime')} rules={[{ required: true }]}><Select>{['vm', 'k8s', 'hybrid'].map((value) => <Select.Option key={value} value={value}>{t(`business.service.catalog.runtime.${value}`)}</Select.Option>)}</Select></Form.Item>
          <Form.Item field="status" label={t('business.service.field.status')} rules={[{ required: true }]}><Select>{['active', 'inactive'].map((value) => <Select.Option key={value} value={value}>{t(`business.service.status.${value}`)}</Select.Option>)}</Select></Form.Item>
          <Form.Item field="description" label={t('common.description')}><Input.TextArea /></Form.Item>
        </Form> : null}
      </AppModal>
    </PageContainer>
  );
}
