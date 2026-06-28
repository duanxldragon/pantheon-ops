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
} from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus, IconTags } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppModal,
  AppTable,
  FilterPanel,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SubmitBar,
  TableBatchActionBar,
  buildStandardPagination,
  useGovernanceRail,
} from '../../../../components';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createLabelSchema,
  deleteLabelSchema,
  getLabelSchemaList,
  updateLabelSchema,
  type LabelSchemaPayload,
  type LabelSchemaRow,
} from './api';
import { labelCategoryOptions, labelPresetOptions } from './catalog';
import '../../../system/components/shared/list-page.css';
import '../cmdb.css';

type LabelFormValues = LabelSchemaPayload & {
  presetKey?: string;
};

export default function CmdbLabelSchemaList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();
  const [form] = Form.useForm<LabelFormValues>();
  const [data, setData] = useState<LabelSchemaRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<LabelSchemaRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [keyword, setKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [filterCategory, setFilterCategory] = useState('');
  const [queryKeyword, setQueryKeyword] = useState('');
  const [queryStatus, setQueryStatus] = useState('');
  const [queryCategory, setQueryCategory] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const canCreate = hasPerm('business:cmdb:label:create');
  const canUpdate = hasPerm('business:cmdb:label:update');
  const canDelete = hasPerm('business:cmdb:label:delete');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getLabelSchemaList({
        keyword: queryKeyword,
        status: queryStatus,
        category: queryCategory,
        page,
        pageSize,
      });
      setData(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, queryCategory, queryKeyword, queryStatus, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  useEffect(() => {
    if (editing) {
      form.setFieldsValue({
        ...editing,
        presetKey: undefined,
      });
      return;
    }
    form.resetFields();
    form.setFieldsValue({
      category: 'base',
      valueMode: 'free',
      status: 'enabled',
      required: false,
      options: [],
      presetKey: undefined,
    });
  }, [editing, form, visible]);

  const applyPreset = (presetKey?: string) => {
    if (!presetKey) {
      return;
    }
    const preset = labelPresetOptions.find((item) => item.value === presetKey);
    if (!preset) {
      return;
    }
    form.setFieldsValue({
      ...preset.payload,
      description: t(preset.descriptionI18nKey),
      presetKey,
      required: false,
    });
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    const payload: LabelSchemaPayload = {
      key: editing ? undefined : values.key,
      name: values.name,
      category: values.category,
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

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => deleteLabelSchema(Number(rowKey))));
      Message.success(t('common.deleteSuccess'));
      setSelectedRowKeys([]);
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('business.cmdb.label.hero.total'),
        value: total,
      },
      {
        key: 'enabled',
        label: t('business.cmdb.label.status.enabled'),
        value: data.filter((item) => item.status === 'enabled').length,
      },
      {
        key: 'required',
        label: t('business.cmdb.label.schema.required'),
        value: data.filter((item) => item.required).length,
      },
      {
        key: 'groupCount',
        label: t('business.cmdb.label.hero.categoryCount'),
        value: new Set(data.map((item) => item.category)).size,
      },
    ],
    [data, t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.cmdb.label.hero.scope'),
        value: t('business.cmdb.label.hero.scopeValue'),
        description: t('business.cmdb.label.hero.scopeHint'),
      },
      {
        label: t('business.cmdb.label.hero.enumCount'),
        value: data.filter((item) => item.valueMode === 'enum').length,
        description: t('business.cmdb.label.hero.enumCountHint'),
      },
      {
        label: t('business.cmdb.label.hero.dictCount'),
        value: data.filter((item) => item.valueMode === 'dict').length,
        description: t('business.cmdb.label.hero.dictCountHint'),
      },
      {
        label: t('business.cmdb.label.hero.categoryCount'),
        value: new Set(data.map((item) => item.category)).size,
        description: t('business.cmdb.label.hero.categoryCountHint'),
      },
    ],
    [data, t],
  );

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
        title: t('business.cmdb.label.schema.category'),
        dataIndex: 'category',
        width: 140,
        render: (_: unknown, row) => (
          <Tag color="purple">{t(`business.cmdb.label.category.${row.category}`)}</Tag>
        ),
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
        width: 260,
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
            {canUpdate ? (
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
            ) : null}
            {canDelete ? (
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
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, loadData, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          icon={<IconTags />}
          eyebrow={t('business.cmdb.label.hero.eyebrow')}
          title={t('business.cmdb.label.schema.title')}
          description={t('business.cmdb.label.hero.title')}
          metrics={heroStats}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('business.cmdb.label.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <FilterPanel>
          <Form layout="inline">
            <Form.Item label={t('common.keyword')}>
              <Input value={keyword} onChange={setKeyword} allowClear />
            </Form.Item>
            <Form.Item label={t('business.cmdb.label.schema.category')}>
              <Select value={filterCategory} onChange={setFilterCategory} allowClear style={{ width: 160 }}>
                {labelCategoryOptions.map((item) => (
                  <Select.Option key={item.value} value={item.value}>
                    {t(item.i18nKey)}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item label={t('business.cmdb.label.schema.status')}>
              <Select value={filterStatus} onChange={setFilterStatus} allowClear style={{ width: 160 }}>
                <Select.Option value="enabled">{t('business.cmdb.label.status.enabled')}</Select.Option>
                <Select.Option value="disabled">{t('business.cmdb.label.status.disabled')}</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  onClick={() => {
                    setQueryKeyword(keyword);
                    setQueryCategory(filterCategory);
                    setQueryStatus(filterStatus);
                    setSelectedRowKeys([]);
                    setPage(1);
                  }}
                >
                  {t('common.search')}
                </Button>
                <Button
                  onClick={() => {
                    setKeyword('');
                    setFilterCategory('');
                    setFilterStatus('');
                    setQueryKeyword('');
                    setQueryCategory('');
                    setQueryStatus('');
                    setSelectedRowKeys([]);
                    setPage(1);
                  }}
                >
                  {t('common.reset')}
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </FilterPanel>
        <TableBatchActionBar
          selectedCount={selectedRowKeys.length}
          selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
          clearText={t('common.clearSelection')}
          clearSuccessText={t('common.clearSelectionSuccess')}
          onClear={() => setSelectedRowKeys([])}
          prefixActions={
            canCreate ? (
              <ListHeaderActions
                primary={
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
                }
              />
            ) : undefined
          }
          actions={
            canDelete ? (
              <Popconfirm
                title={t('common.deleteConfirm')}
                onOk={() => {
                  void handleBatchDelete();
                }}
                disabled={selectedRowKeys.length === 0 || submitting}
              >
                <Button
                  status="danger"
                  icon={<IconDelete />}
                  disabled={selectedRowKeys.length === 0 || submitting}
                  loading={submitting}
                >
                  {t('common.deleteSelected')}
                </Button>
              </Popconfirm>
            ) : undefined
          }
        />
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
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
                fixed: true,
                onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
              }}
              pagination={buildStandardPagination(t, {
                current: page,
                pageSize,
                total,
                onChange: (nextPage, nextPageSize) => {
                  setPage(nextPage || 1);
                  setPageSize(nextPageSize || pageSize);
                },
                pageSizeChangeResetCurrent: true,
              })}
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('business.cmdb.label.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.cmdb.label.hero.sideLead')}
        noteDescription={t('business.cmdb.label.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
        title={editing ? t('business.cmdb.label.editTitle') : t('business.cmdb.label.createTitle')}
        footer={null}
        size="lg"
      >
        <Form form={form} layout="vertical" onSubmit={handleSubmit}>
          {!editing ? (
            <Form.Item label={t('business.cmdb.label.schema.preset')} field="presetKey">
              <Select allowClear onChange={(value) => applyPreset(value)}>
                {labelPresetOptions.map((item) => (
                  <Select.Option key={item.value} value={item.value}>
                    {t(item.i18nKey)}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
          ) : null}
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
          <Form.Item label={t('business.cmdb.label.schema.category')} field="category">
            <Select>
              {labelCategoryOptions.map((item) => (
                <Select.Option key={item.value} value={item.value}>
                  {t(item.i18nKey)}
                </Select.Option>
              ))}
            </Select>
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
                      <Popconfirm title={t('common.deleteConfirm')} onOk={() => remove(index)}>
                        <Button type="text" status="danger" icon={<IconDelete />} />
                      </Popconfirm>
                    </Space>
                  ))}
                  <Button type="dashed" icon={<IconPlus />} onClick={() => add('')}>
                    {t('business.cmdb.label.schema.addOption')}
                  </Button>
                </Space>
              )}
            </Form.List>
          </Form.Item>
          <Form.Item
            label={t('business.cmdb.label.schema.required')}
            field="required"
            triggerPropName="checked"
          >
            <Switch />
          </Form.Item>
          <Form.Item label={t('business.cmdb.label.schema.status')} field="status">
            <Select>
              <Select.Option value="enabled">{t('business.cmdb.label.status.enabled')}</Select.Option>
              <Select.Option value="disabled">{t('business.cmdb.label.status.disabled')}</Select.Option>
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
