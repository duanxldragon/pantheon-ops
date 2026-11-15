import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, Message, Popconfirm, Select, Space, Tag } from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { isForbiddenRequestError } from '../../../../api/request';
import {
  AppModal, AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError,
  PageForbidden, PageLoading, SearchToolbar, buildStandardPagination, TABLE_ACTION_COLUMN_WIDTH,
} from '../../../../components';
import { getBizScopeOptions, type BizScopeOptionItem } from '../api';
import {
  createEnvironment, deleteEnvironment, getEnvironmentList, updateEnvironment,
  type BusinessEnvironment, type EnvironmentPayload, type EnvironmentQuery,
} from './api';
import '../../../../assets/list-page.css';

const emptyQuery: EnvironmentQuery = { keyword: '', page: 1, pageSize: 20 };

export default function EnvironmentList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const canList = hasPerm('business:bizscope:environment:list');
  const canCreate = hasPerm('business:bizscope:environment:create');
  const canUpdate = hasPerm('business:bizscope:environment:update');
  const canDelete = hasPerm('business:bizscope:environment:delete');
  const [query, setQuery] = useState<EnvironmentQuery>(emptyQuery);
  const [rows, setRows] = useState<BusinessEnvironment[]>([]);
  const [scopes, setScopes] = useState<BizScopeOptionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<BusinessEnvironment | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<EnvironmentPayload>();

  useEffect(() => {
    const controller = new AbortController();
    if (!canList) return () => controller.abort();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void Promise.all([
        getEnvironmentList(query, controller.signal), getBizScopeOptions(controller.signal),
      ]).then(([response, options]) => {
        if (controller.signal.aborted) return;
        setRows(response.items);
        setTotal(response.total);
        setScopes(options);
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
  const openForm = (row: BusinessEnvironment | null) => {
    setEditing(row);
    form.resetFields();
    form.setFieldsValue(row || { envType: 'dev', status: 'active', businessScopeId: query.businessScopeId });
    setVisible(true);
  };

  const submit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      if (editing) await updateEnvironment(editing.id, values);
      else await createEnvironment(values);
      Message.success(t(editing ? 'common.updateSuccess' : 'common.createSuccess'));
      setVisible(false);
      reload();
    } catch {
      Message.error(t('business.bizscope.environment.operationFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ColumnProps<BusinessEnvironment>[] = [
    { title: t('business.bizscope.environment.field.scope'), dataIndex: 'businessScopeName', width: 180 },
    { title: t('business.bizscope.environment.field.code'), dataIndex: 'code', width: 160 },
    { title: t('business.bizscope.environment.field.name'), dataIndex: 'name', width: 180 },
    {
      title: t('business.bizscope.environment.field.envType'), dataIndex: 'envType', width: 140,
      render: (_: unknown, row) => t(`business.bizscope.environment.${row.envType}`),
    },
    {
      title: t('business.bizscope.field.status'), dataIndex: 'status', width: 120,
      render: (_: unknown, row) => <Tag color={row.status === 'active' ? 'green' : 'gray'}>{t(`business.bizscope.status.${row.status}`)}</Tag>,
    },
    { title: t('business.bizscope.environment.field.remark'), dataIndex: 'remark', width: 220 },
    {
      title: t('common.action'), width: TABLE_ACTION_COLUMN_WIDTH.medium, fixed: 'right',
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          {canUpdate ? <Button type="text" size="small" icon={<IconEdit />} onClick={() => openForm(row)}>{t('common.edit')}</Button> : null}
          {canDelete ? (
            <Popconfirm title={t('business.bizscope.environment.deleteConfirm', { name: row.name })} onOk={async () => {
              try {
                await deleteEnvironment(row.id);
                Message.success(t('common.deleteSuccess'));
                reload();
              } catch {
                Message.error(t('business.bizscope.environment.operationFailed'));
              }
            }}>
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ];

  const updateQuery = (patch: Partial<EnvironmentQuery>) => setQuery((current) => ({ ...current, ...patch, page: 1 }));
  if (!canList || isForbiddenRequestError(error)) return <PageContainer><PageForbidden /></PageContainer>;

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.bizscope.hero.eyebrow')}
          title={t('operations.bizscope.environment.menu')}
          description={t('business.bizscope.environment.description')}
          metrics={[{ key: 'total', label: t('business.bizscope.environment.total'), value: total }]}
        />
        <SearchToolbar
          keyword={query.keyword ?? ''}
          keywordPlaceholder={t('business.bizscope.environment.keywordPlaceholder')}
          onKeywordChange={(keyword) => updateQuery({ keyword })}
          inlineFilters={(
            <>
              <Select allowClear showSearch aria-label={t('business.bizscope.environment.field.scope')} placeholder={t('business.bizscope.environment.field.scope')} value={query.businessScopeId} onChange={(businessScopeId) => updateQuery({ businessScopeId })}>
                {scopes.map((scope) => <Select.Option key={scope.id} value={scope.id}>{scope.name}</Select.Option>)}
              </Select>
              <Select allowClear aria-label={t('business.bizscope.environment.field.envType')} placeholder={t('business.bizscope.environment.field.envType')} value={query.envType} onChange={(envType) => updateQuery({ envType })}>
                {['dev', 'test', 'prod'].map((value) => <Select.Option key={value} value={value}>{t(`business.bizscope.environment.${value}`)}</Select.Option>)}
              </Select>
            </>
          )}
          hasActiveFilters={Boolean(query.keyword || query.businessScopeId || query.envType)}
          onClearAll={() => setQuery(emptyQuery)}
          extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => openForm(null)}>{t('common.add')}</Button> : undefined}
        />
        <Card className="page-panel system-list__table-card">
          {loading ? <PageLoading /> : null}
          {!loading && error ? <PageError onRetry={reload} /> : null}
          {!loading && !error && total === 0 ? <PageEmpty description={t('business.bizscope.environment.empty')} /> : null}
          {!loading && !error && total > 0 ? (
            <AppTable className="system-list__table" rowKey="id" data={rows} columns={columns} scroll={{ x: 'max-content' }} pagination={buildStandardPagination(t, {
              current: query.page ?? 1, pageSize: query.pageSize ?? 20, total,
              onChange: (page, pageSize) => setQuery((current) => ({ ...current, page, pageSize })),
            })} />
          ) : null}
        </Card>
      </Space>
      <AppModal visible={visible} title={t(editing ? 'business.bizscope.environment.editTitle' : 'business.bizscope.environment.createTitle')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={submitting} onOk={submit} onCancel={() => setVisible(false)}>
        <Form form={form} layout="vertical">
          <Form.Item field="businessScopeId" label={t('business.bizscope.environment.field.scope')} rules={[{ required: true }]}>
            <Select showSearch>{scopes.map((scope) => <Select.Option key={scope.id} value={scope.id}>{scope.name}</Select.Option>)}</Select>
          </Form.Item>
          <Form.Item field="code" label={t('business.bizscope.environment.field.code')} rules={[{ required: true }]}><Input placeholder={t('business.bizscope.environment.field.code')} maxLength={64} /></Form.Item>
          <Form.Item field="name" label={t('business.bizscope.environment.field.name')} rules={[{ required: true }]}><Input placeholder={t('business.bizscope.environment.field.name')} maxLength={100} /></Form.Item>
          <Form.Item field="envType" label={t('business.bizscope.environment.field.envType')} rules={[{ required: true }]}>
            <Select>{['dev', 'test', 'prod'].map((value) => <Select.Option key={value} value={value}>{t(`business.bizscope.environment.${value}`)}</Select.Option>)}</Select>
          </Form.Item>
          <Form.Item field="status" label={t('business.bizscope.field.status')} rules={[{ required: true }]}>
            <Select>{['active', 'inactive'].map((value) => <Select.Option key={value} value={value}>{t(`business.bizscope.status.${value}`)}</Select.Option>)}</Select>
          </Form.Item>
          <Form.Item field="remark" label={t('business.bizscope.environment.field.remark')}><Input.TextArea maxLength={500} /></Form.Item>
        </Form>
      </AppModal>
    </PageContainer>
  );
}
