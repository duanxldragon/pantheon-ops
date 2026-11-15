import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, Message, Popconfirm, Select, Space, Switch, Tag } from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { isForbiddenRequestError } from '../../../api/request';
import {
  AppModal, AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError,
  PageForbidden, PageLoading, SearchToolbar, buildStandardPagination, TABLE_ACTION_COLUMN_WIDTH,
} from '../../../components';
import { getBizScopeOptions, type BizScopeOptionItem } from '../bizscope/api';
import { createApplication, deleteApplication, getApplicationList, updateApplication, type ApplicationRow } from './api';
import '../../../assets/list-page.css';

interface ApplicationValues {
  code: string;
  name: string;
  businessScopeId: number;
  owner?: string;
  remark?: string;
}
const emptyQuery = { keyword: '', status: '', page: 1, pageSize: 20 };

export default function ApplicationList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const canList = hasPerm('business:service:list');
  const canCreate = hasPerm('business:service:create');
  const canUpdate = hasPerm('business:service:update');
  const canDelete = hasPerm('business:service:delete');
  const [query, setQuery] = useState(emptyQuery);
  const [rows, setRows] = useState<ApplicationRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<ApplicationRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [scopes, setScopes] = useState<BizScopeOptionItem[]>([]);
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [optionsError, setOptionsError] = useState<unknown>(null);
  const [form] = Form.useForm<ApplicationValues>();

  useEffect(() => {
    const controller = new AbortController();
    if (!canList) return () => controller.abort();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void getApplicationList(query, controller.signal).then((response) => {
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
  const loadScopes = async () => {
    setOptionsLoading(true);
    setOptionsError(null);
    try { setScopes(await getBizScopeOptions()); }
    catch (requestError) { setOptionsError(requestError); }
    finally { setOptionsLoading(false); }
  };
  const openForm = (row: ApplicationRow | null) => {
    setEditing(row);
    setOptionsError(null);
    form.resetFields();
    if (row) {
      form.setFieldsValue(row);
      setScopes([{ id: row.businessScopeId, value: row.businessScopeId, label: row.businessScopeName, name: row.businessScopeName }]);
    } else void loadScopes();
    setVisible(true);
  };
  const submit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      if (editing) await updateApplication(editing.id, { name: values.name, owner: values.owner, remark: values.remark });
      else await createApplication({ ...values, status: 'active' });
      Message.success(t(editing ? 'common.updateSuccess' : 'business.service.catalog.applicationCreateSuccess'));
      setVisible(false);
      reload();
    } catch { Message.error(t('business.service.catalog.operationFailed')); }
    finally { setSubmitting(false); }
  };
  const toggleStatus = async (row: ApplicationRow, active: boolean) => {
    try {
      await updateApplication(row.id, { status: active ? 'active' : 'inactive' });
      reload();
    } catch { Message.error(t('business.service.catalog.operationFailed')); }
  };
  const columns: ColumnProps<ApplicationRow>[] = [
    { title: t('business.service.catalog.field.code'), dataIndex: 'code', width: 150, render: (value: string) => <Tag color="arcoblue">{value}</Tag> },
    { title: t('business.service.catalog.field.application'), dataIndex: 'name', width: 200 },
    { title: t('business.service.catalog.field.scope'), dataIndex: 'businessScopeName', width: 180 },
    { title: t('business.service.catalog.field.owner'), dataIndex: 'owner', width: 140 },
    { title: t('business.service.catalog.field.services'), dataIndex: 'serviceCount', width: 100 },
    {
      title: t('business.service.catalog.field.active'), dataIndex: 'status', width: 100,
      render: (_: unknown, row) => <Switch aria-label={t('business.service.catalog.field.active')} checked={row.status === 'active'} disabled={!canUpdate} onChange={(active) => void toggleStatus(row, active)} />,
    },
    {
      title: t('common.action'), width: TABLE_ACTION_COLUMN_WIDTH.medium, fixed: 'right',
      render: (_: unknown, row) => <Space className="system-list__actions">
        {canUpdate ? <Button type="text" size="small" icon={<IconEdit />} onClick={() => openForm(row)}>{t('common.edit')}</Button> : null}
        {canDelete ? <Popconfirm title={t('business.service.catalog.applicationDeleteConfirm', { name: row.name })} onOk={async () => {
          try { await deleteApplication(row.id); reload(); }
          catch { Message.error(t('business.service.catalog.operationFailed')); }
        }}><Button type="text" size="small" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null}
      </Space>,
    },
  ];
  if (!canList || isForbiddenRequestError(error)) return <PageContainer><PageForbidden /></PageContainer>;

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.service.catalog.hero.eyebrow')} title={t('operations.service.application.menu')} description={t('business.service.catalog.hero.title')} metrics={[
          { key: 'total', label: t('business.service.catalog.hero.applications'), value: total },
        ]} />
        <SearchToolbar
          keyword={query.keyword}
          keywordPlaceholder={t('business.service.catalog.applicationKeyword')}
          onKeywordChange={(keyword) => setQuery((current) => ({ ...current, keyword, page: 1 }))}
          inlineFilters={(
          <Select allowClear placeholder={t('business.service.field.status')} value={query.status || undefined} onChange={(status) => setQuery((current) => ({ ...current, status: status || '', page: 1 }))}>
            {['active', 'inactive'].map((value) => <Select.Option key={value} value={value}>{t(`business.service.status.${value}`)}</Select.Option>)}
          </Select>
          )}
          hasActiveFilters={Boolean(query.keyword || query.status)}
          onClearAll={() => setQuery(emptyQuery)}
          extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => openForm(null)}>{t('business.service.catalog.addApplication')}</Button> : undefined}
        />
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {!loading && error ? <PageError onRetry={reload} /> : null}
          {!loading && !error && total === 0 ? <PageEmpty description={t('business.service.catalog.applicationEmpty')} /> : null}
          {!loading && !error && total > 0 ? <AppTable className="system-list__table" rowKey="id" data={rows} columns={columns} scroll={{ x: 'max-content' }} pagination={buildStandardPagination(t, {
            current: query.page, pageSize: query.pageSize, total, onChange: (page, pageSize) => setQuery((current) => ({ ...current, page, pageSize })),
          })} /> : null}
        </Card>
      </Space>
      <AppModal visible={visible} title={t(editing ? 'business.service.catalog.editApplication' : 'business.service.catalog.addApplication')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={submitting} okButtonProps={{ disabled: optionsLoading || Boolean(optionsError) }} onOk={submit} onCancel={() => setVisible(false)}>
        {optionsLoading ? <PageLoading /> : null}
        {optionsError ? <PageError onRetry={() => void loadScopes()} /> : null}
        {!optionsLoading && !optionsError ? <Form form={form} layout="vertical">
          <Form.Item field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input disabled={Boolean(editing)} /></Form.Item>
          <Form.Item field="name" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item field="businessScopeId" label={t('business.service.catalog.field.scope')} rules={[{ required: true }]}><Select disabled={Boolean(editing)} showSearch>{scopes.map((scope) => <Select.Option key={scope.id} value={scope.id}>{scope.name}</Select.Option>)}</Select></Form.Item>
          <Form.Item field="owner" label={t('business.service.catalog.field.owner')}><Input /></Form.Item>
          <Form.Item field="remark" label={t('business.service.catalog.field.remark')}><Input.TextArea /></Form.Item>
        </Form> : null}
      </AppModal>
    </PageContainer>
  );
}
