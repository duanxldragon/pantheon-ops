import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Form,
  Input,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
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
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
} from '../../../../components';
import PasswordInput from '../../../../components/patterns/forms/PasswordInput';
import { formatDateTime } from '../../../../core/format/dateTime';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createDeployCredential,
  deleteDeployCredential,
  getDeployCredentialList,
  getDeployCredentialScopeOptions,
  updateDeployCredential,
  type DeployCredentialPayload,
  type DeployCredentialRow,
  type DeployCredentialScopeOption,
} from '../api';
import '../../../../assets/list-page.css';
import '../deploy.css';

type CredentialFormValues = {
  name: string;
  username: string;
  authMode: 'password' | 'private_key';
  businessScopeId: number;
  secret?: string;
  status?: 'active' | 'disabled';
};

export default function DeployCredentialList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const canCreate = hasPerm('business:deploy:credential:create');
  const canUpdate = hasPerm('business:deploy:credential:update');
  const canDelete = hasPerm('business:deploy:credential:delete');

  const [data, setData] = useState<DeployCredentialRow[]>([]);
  const [scopes, setScopes] = useState<DeployCredentialScopeOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<DeployCredentialRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<CredentialFormValues>();

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(false);
    try {
      const [items, scopeOptions] = await Promise.all([getDeployCredentialList(), getDeployCredentialScopeOptions()]);
      setData(items || []);
      setScopes(scopeOptions || []);
    } catch {
      setError(true);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const scopeNames = useMemo(() => new Map(scopes.map(({ id, name }) => [id, name])), [scopes]);

  const filtered = useMemo(() => {
    const query = keyword.trim().toLowerCase();
    if (!query) {
      return data;
    }
    return data.filter((row) =>
      [row.name, row.username, scopeNames.get(row.businessScopeId) || '']
        .join(' ')
        .toLowerCase()
        .includes(query),
    );
  }, [data, keyword, scopeNames]);

  const paged = useMemo(
    () => filtered.slice((page - 1) * pageSize, page * pageSize),
    [filtered, page, pageSize],
  );

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.credential.hero.total'), value: filtered.length },
      { key: 'active', label: t('business.deploy.credential.status.active'), value: filtered.filter((row) => row.status === 'active').length },
      { key: 'scoped', label: t('business.deploy.credential.hero.scoped'), value: filtered.filter((row) => row.businessScopeId > 0).length },
    ],
    [filtered, t],
  );

  const openCreate = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({ authMode: 'password', status: 'active' });
    setVisible(true);
  };

  const openEdit = (row: DeployCredentialRow) => {
    setEditing(row);
    form.setFieldsValue({
      name: row.name,
      username: row.username,
      authMode: row.authMode,
      businessScopeId: row.businessScopeId,
      status: row.status === 'disabled' ? 'disabled' : 'active',
      secret: '',
    });
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    const payload: DeployCredentialPayload = {
      name: values.name.trim(),
      username: values.username.trim(),
      authMode: values.authMode,
      businessScopeId: values.businessScopeId,
      status: values.status,
    };
    if (values.secret) {
      payload.secret = values.secret;
    }
    setSubmitting(true);
    try {
      if (editing) {
        await updateDeployCredential(editing.id, payload);
        Message.success(t('common.updateSuccess'));
      } else {
        await createDeployCredential(payload);
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
    await deleteDeployCredential(id);
    Message.success(t('common.deleteSuccess'));
    await loadData();
  };

  const columns: ColumnProps<DeployCredentialRow>[] = [
    { title: t('business.deploy.credential.name'), dataIndex: 'name', width: 180 },
    {
      title: t('business.deploy.credential.businessScope'),
      dataIndex: 'businessScopeId',
      width: 180,
      render: (value: number) => value === 0 ? t('business.deploy.credential.platformScope') : scopeNames.get(value) ?? `#${value}`,
    },
    { title: t('business.deploy.credential.username'), dataIndex: 'username', width: 140 },
    {
      title: t('business.deploy.credential.authMode'),
      dataIndex: 'authMode',
      width: 130,
      render: (value: string) => t(`business.deploy.credential.authMode.${value}`),
    },
    { title: t('business.deploy.credential.version'), dataIndex: 'version', width: 90 },
    {
      title: t('business.deploy.credential.statusLabel'),
      dataIndex: 'status',
      width: 110,
      render: (value: string) => (
        <Tag color={value === 'active' ? 'green' : 'gray'}>
          {t(`business.deploy.credential.status.${value === 'active' ? 'active' : 'disabled'}`)}
        </Tag>
      ),
    },
    {
      title: t('business.deploy.credential.updatedAt'),
      dataIndex: 'updatedAt',
      width: 180,
      render: (value: string) => formatDateTime(value),
    },
  ];
  if (canUpdate || canDelete) {
    columns.push({
      title: t('common.actions'),
      width: canUpdate && canDelete ? TABLE_ACTION_COLUMN_WIDTH.compact : TABLE_ACTION_COLUMN_WIDTH.single,
      fixed: 'right',
      render: (_: unknown, row: DeployCredentialRow) => (
        <Space>
          {canUpdate ? (
            <Button size="mini" icon={<IconEdit />} onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          ) : null}
          {canDelete ? (
            <Popconfirm title={t('common.deleteConfirm')} onOk={() => void handleDelete(row.id)}>
              <Button size="mini" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    });
  }

  if (loading && data.length === 0) {
    return (
      <PageContainer>
        <PageLoading />
      </PageContainer>
    );
  }

  if (error && data.length === 0) {
    return (
      <PageContainer>
        <PageError description={t('common.loadFailedDesc')} onRetry={loadData} />
      </PageContainer>
    );
  }

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.deploy.credential.hero.eyebrow')}
          title={t('operations.deploy.credential.menu')}
          description={t('business.deploy.credential.hero.title')}
          metrics={heroStats.map((item) => ({ key: item.key, label: item.label, value: item.value }))}
        />
        <SearchToolbar
          keyword={keyword}
          keywordPlaceholder={t('business.deploy.credential.searchPlaceholder')}
          onKeywordChange={(value) => {
            setKeyword(value);
            setPage(1);
          }}
          hasActiveFilters={Boolean(keyword)}
          onClearAll={() => {
            setKeyword('');
            setPage(1);
          }}
        />
        <ListHeaderActions
          primary={
            canCreate ? (
              <Button type="primary" icon={<IconPlus />} onClick={openCreate}>
                {t('common.add')}
              </Button>
            ) : undefined
          }
        />
        {filtered.length === 0 ? (
          <PageEmpty description={t('business.deploy.credential.empty')} />
        ) : (
          <AppTable
            className="system-list__table"
            rowKey="id"
            columns={columns}
            data={paged}
            loading={loading}
            scroll={{ x: 'max-content' }}
            pagination={buildStandardPagination(t, {
              current: page,
              pageSize,
              total: filtered.length,
              onChange: (nextPage, nextPageSize) => {
                setPage(nextPage || 1);
                if (nextPageSize && nextPageSize !== pageSize) {
                  setPageSize(nextPageSize);
                  setPage(1);
                }
              },
            })}
          />
        )}
      </Space>
      <AppModal
        visible={visible}
        title={editing ? t('business.deploy.credential.editTitle') : t('business.deploy.credential.createTitle')}
        onCancel={() => setVisible(false)}
        footer={
          <SubmitBar
            loading={submitting}
            submitText={t('common.save')}
            cancelText={t('common.cancel')}
            onCancel={() => setVisible(false)}
            onSubmit={() => void handleSubmit()}
          />
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label={t('business.deploy.credential.name')}
            field="name"
            rules={[{ required: true, message: t('business.deploy.credential.nameRequired') }]}
          >
            <Input placeholder={t('business.deploy.credential.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('business.deploy.credential.businessScope')}
            field="businessScopeId"
            rules={[{ required: true, message: t('business.deploy.credential.businessScopeRequired') }]}
          >
            <Select
              placeholder={t('business.deploy.credential.businessScopePlaceholder')}
              options={scopes.map((scope) => ({
                label: scope.name,
                value: scope.id,
              }))}
            />
          </Form.Item>
          <Form.Item
            label={t('business.deploy.credential.username')}
            field="username"
            rules={[{ required: true, message: t('business.deploy.credential.usernameRequired') }]}
          >
            <Input autoComplete="off" />
          </Form.Item>
          <Form.Item
            label={t('business.deploy.credential.authMode')}
            field="authMode"
            rules={[{ required: true }]}
          >
            <Select
              options={[
                { label: t('business.deploy.credential.authMode.password'), value: 'password' },
                { label: t('business.deploy.credential.authMode.private_key'), value: 'private_key' },
              ]}
            />
          </Form.Item>
          <Form.Item
            label={editing ? t('business.deploy.credential.secretRotate') : t('business.deploy.credential.secret')}
            field="secret"
            rules={editing ? [] : [{ required: true, message: t('business.deploy.credential.secretRequired') }]}
            extra={t('business.deploy.credential.secretHint')}
          >
            <PasswordInput autoComplete="new-password" />
          </Form.Item>
          {editing ? (
            <Form.Item label={t('business.deploy.credential.statusLabel')} field="status">
              <Select
                options={[
                  { label: t('business.deploy.credential.status.active'), value: 'active' },
                  { label: t('business.deploy.credential.status.disabled'), value: 'disabled' },
                ]}
              />
            </Form.Item>
          ) : null}
        </Form>
      </AppModal>
    </PageContainer>
  );
}
