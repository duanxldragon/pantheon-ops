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
  Switch,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import AppTable from '../../../../components/data-display/AppTable';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import ListHeaderActions from '../../../../components/patterns/ListHeaderActions';
import SubmitBar from '../../../../components/patterns/SubmitBar';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createLabelSchema,
  deleteLabelSchema,
  getLabelSchemaList,
  updateLabelSchema,
  type LabelSchemaPayload,
  type LabelSchemaRow,
} from './api';
import '../../../system/list-page.css';
import '../cmdb.css';

export default function CmdbLabelSchemaList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [form] = Form.useForm();
  const [data, setData] = useState<LabelSchemaRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<LabelSchemaRow | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const canCreate = hasPerm('business:cmdb:label:create');
  const canUpdate = hasPerm('business:cmdb:label:update');
  const canDelete = hasPerm('business:cmdb:label:delete');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getLabelSchemaList();
      setData(result);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  useEffect(() => {
    if (editing) {
      form.setFieldsValue(editing);
      return;
    }
    form.resetFields();
    form.setFieldsValue({ valueMode: 'free', status: 'enabled', required: false });
  }, [editing, form, visible]);

  const handleSubmit = async () => {
    const values = await form.validate();
    const payload: LabelSchemaPayload = {
      key: editing ? undefined : values.key,
      name: values.name,
      valueMode: values.valueMode,
      dictCode: values.dictCode,
      options: (values.options || [])
        .map((item: string) => String(item || '').trim())
        .filter(Boolean),
      required: Boolean(values.required),
      status: values.status,
      description: values.description,
    };
    setSubmitting(true);
    try {
      if (editing) {
        await updateLabelSchema(editing.id, payload);
        Message.success(t('common.updateSuccess'));
      } else {
        await createLabelSchema(payload);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const columns = useMemo<ColumnProps<LabelSchemaRow>[]>(
    () => [
      {
        title: t('business.cmdb.label.schema.key'),
        dataIndex: 'key',
        width: 140,
        render: (_: unknown, row) => <Tag color="arcoblue">{row.key}</Tag>,
      },
      {
        title: t('business.cmdb.label.schema.name'),
        dataIndex: 'name',
        width: 160,
      },
      {
        title: t('business.cmdb.label.schema.valueMode'),
        dataIndex: 'valueMode',
        width: 140,
        render: (_: unknown, row) => t(`business.cmdb.label.valueMode.${row.valueMode}`),
      },
      {
        title: t('business.cmdb.label.schema.dictCode'),
        dataIndex: 'dictCode',
        width: 160,
        render: (_: unknown, row) => row.dictCode || '-',
      },
      {
        title: t('business.cmdb.label.schema.options'),
        dataIndex: 'options',
        width: 240,
        render: (_: unknown, row) =>
          row.options?.length ? (
            <Space wrap size={4}>
              {row.options.map((option) => (
                <Tag key={option} size="small">
                  {option}
                </Tag>
              ))}
            </Space>
          ) : (
            '-'
          ),
      },
      {
        title: t('business.cmdb.label.schema.required'),
        dataIndex: 'required',
        width: 100,
        render: (_: unknown, row) => (
          <Tag color={row.required ? 'orange' : 'gray'}>
            {row.required ? t('common.yes') : t('common.no')}
          </Tag>
        ),
      },
      {
        title: t('business.cmdb.label.schema.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'enabled' ? 'green' : 'gray'}>
            {t(`business.cmdb.label.status.${row.status}`)}
          </Tag>
        ),
      },
      {
        title: t('business.cmdb.label.schema.description'),
        dataIndex: 'description',
        width: 240,
        render: (_: unknown, row) => row.description || '-',
      },
      {
        title: t('common.action'),
        key: 'action',
        fixed: 'right',
        width: 170,
        render: (_: unknown, row) => (
          <Space>
            {canUpdate && (
              <Button
                type="text"
                size="small"
                icon={<IconEdit />}
                onClick={() => {
                  setEditing(row);
                  setVisible(true);
                }}
              >
                {t('common.edit')}
              </Button>
            )}
            {canDelete && (
              <Popconfirm
                title={t('business.cmdb.label.deleteConfirm')}
                onOk={async () => {
                  await deleteLabelSchema(row.id);
                  Message.success(t('common.deleteSuccess'));
                  await loadData();
                }}
              >
                <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            )}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, loadData, t],
  );

  return (
    <PageContainer>
      <PageHeader
        title={t('business.cmdb.label.schema.title')}
        extra={
          <ListHeaderActions
            primary={
              canCreate ? (
                <Button
                  type="primary"
                  icon={<IconPlus />}
                  onClick={() => {
                    setEditing(null);
                    setVisible(true);
                  }}
                >
                  {t('common.add')}
                </Button>
              ) : null
            }
          />
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero cmdb-page__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">
                {t('business.cmdb.label.hero.eyebrow')}
              </span>
              <Typography.Title heading={5} className="system-page-hero__title cmdb-page__hero-title">
                {t('business.cmdb.label.hero.title')}
              </Typography.Title>
            </div>
          </div>
        </Card>
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {!loading && error && data.length === 0 ? (
            <PageError description={t('common.loadFailedDesc')} onRetry={loadData} />
          ) : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.cmdb.label.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              columns={columns}
              data={data}
              loading={loading}
              rowKey="id"
              pagination={false}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
      <AppModal
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
        title={
          editing
            ? t('business.cmdb.label.editTitle')
            : t('business.cmdb.label.createTitle')
        }
        footer={null}
        size="lg"
      >
        <Form form={form} layout="vertical" onSubmit={handleSubmit}>
          <Form.Item
            label={t('business.cmdb.label.schema.key')}
            field="key"
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Input disabled={Boolean(editing)} placeholder="env" />
          </Form.Item>
          <Form.Item
            label={t('business.cmdb.label.schema.name')}
            field="name"
            rules={[{ required: true, message: t('common.required') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.valueMode')} field="valueMode">
            <Select>
              {['free', 'enum', 'dict'].map((mode) => (
                <Select.Option key={mode} value={mode}>
                  {t(`business.cmdb.label.valueMode.${mode}`)}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.dictCode')} field="dictCode">
            <Input placeholder="cmdb_env" />
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.options')}>
            <Form.List field="options">
              {(fields, { add, remove }) => (
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  {fields.map((field, index) => (
                    <Space key={field.key} align="start">
                      <Form.Item field={`options[${index}]`} noStyle>
                        <Input
                          placeholder={t('business.cmdb.label.schema.optionPlaceholder')}
                          style={{ width: 280 }}
                        />
                      </Form.Item>
                      <Button
                        type="text"
                        status="danger"
                        icon={<IconDelete />}
                        onClick={() => remove(index)}
                      />
                    </Space>
                  ))}
                  <Button type="dashed" icon={<IconPlus />} onClick={() => add('')}>
                    {t('business.cmdb.label.schema.addOption')}
                  </Button>
                </Space>
              )}
            </Form.List>
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.required')} field="required" triggerPropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.status')} field="status">
            <Select>
              <Select.Option value="enabled">
                {t('business.cmdb.label.status.enabled')}
              </Select.Option>
              <Select.Option value="disabled">
                {t('business.cmdb.label.status.disabled')}
              </Select.Option>
            </Select>
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.description')} field="description">
            <Input.TextArea />
          </Form.Item>
          <SubmitBar
            onCancel={() => {
              setVisible(false);
              setEditing(null);
            }}
            loading={submitting}
            submitText={editing ? t('common.save') : t('common.create')}
          />
        </Form>
      </AppModal>
    </PageContainer>
  );
}
