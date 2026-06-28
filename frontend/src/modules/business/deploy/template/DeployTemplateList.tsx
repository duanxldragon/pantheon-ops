import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
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
import { IconCode, IconDelete, IconEdit, IconEye, IconPlus } from '@arco-design/web-react/icon';
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
  createDeployTemplate,
  deleteDeployTemplate,
  getDeployPackageList,
  getDeployTemplateList,
  updateDeployTemplate,
  type DeployPackageRow,
  type DeployTemplatePayload,
  type DeployTemplateRow,
} from '../api';
import {
  buildDeployTemplateDefaultParameters,
  getDeployFixedTemplateCatalogEntry,
} from '../catalog';
import '../../../system/components/shared/list-page.css';
import '../deploy.css';

type TemplateFormValues = {
  name: string;
  version: string;
  category?: string;
  status: 'enabled' | 'disabled';
  executionMode: 'fixed' | 'orchestrated';
  defaultAction: 'install' | 'uninstall' | 'upgrade' | 'reinstall';
  packageId?: number;
  description?: string;
  parameterValues?: Record<string, string>;
  steps?: Array<{
    stepCode?: string;
    stepName?: string;
    stepType?: 'package' | 'script';
    packageId?: number;
    action?: 'install' | 'uninstall' | 'upgrade' | 'reinstall';
    parameterValues?: Record<string, string>;
    scriptContent?: string;
    precheckCommand?: string;
    postcheckCommand?: string;
  }>;
};

const templateStatusColorMap: Record<string, string> = {
  enabled: 'green',
  disabled: 'gray',
};

const templateStepTypeColorMap: Record<'package' | 'script', string> = {
  package: 'arcoblue',
  script: 'purple',
};

export default function DeployTemplateList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();
  const [form] = Form.useForm<TemplateFormValues>();
  const [data, setData] = useState<DeployTemplateRow[]>([]);
  const [packages, setPackages] = useState<DeployPackageRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('');
  const [queryKeyword, setQueryKeyword] = useState('');
  const [queryStatus, setQueryStatus] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<DeployTemplateRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [selectedDefaultPackageId, setSelectedDefaultPackageId] = useState<number | undefined>();
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailRecord, setDetailRecord] = useState<DeployTemplateRow | null>(null);

  const canCreate = hasPerm('business:deploy:template:create');
  const canUpdate = hasPerm('business:deploy:template:update');
  const canDelete = hasPerm('business:deploy:template:delete');

  const packageMap = useMemo(
    () => new Map(packages.map((item) => [item.id, item])),
    [packages],
  );
  const selectedDefaultPackage = selectedDefaultPackageId ? packageMap.get(selectedDefaultPackageId) : undefined;
  const selectedDefaultTemplateEntry = useMemo(
    () => getDeployFixedTemplateCatalogEntry(selectedDefaultPackage?.templateCode),
    [selectedDefaultPackage?.templateCode],
  );

  const loadPackages = useCallback(async () => {
    const result = await getDeployPackageList({ page: 1, pageSize: 200 });
    setPackages(result.items);
  }, []);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getDeployTemplateList({
        page,
        pageSize,
        keyword: queryKeyword,
        status: queryStatus,
      });
      setData(result.items);
      setTotal(result.total);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, queryKeyword, queryStatus, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void Promise.all([loadData(), loadPackages()]);
    });
  }, [loadData, loadPackages]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.template.hero.total'), value: total },
      {
        key: 'enabled',
        label: t('business.deploy.template.hero.enabled'),
        value: data.filter((item) => item.status === 'enabled').length,
      },
      {
        key: 'steps',
        label: t('business.deploy.template.hero.steps'),
        value: data.reduce((sum, item) => sum + (item.stepCount || 0), 0),
      },
    ],
    [data, t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.deploy.template.hero.fixedCount'),
        value: data.filter((item) => item.executionMode === 'fixed').length,
        description: t('business.deploy.template.hero.fixedCountHint'),
      },
      {
        label: t('business.deploy.template.hero.orchestratedCount'),
        value: data.filter((item) => item.executionMode === 'orchestrated').length,
        description: t('business.deploy.template.hero.orchestratedCountHint'),
      },
      {
        label: t('business.deploy.template.hero.offlineReady'),
        value: data.filter((item) => {
          const pkg = item.packageId ? packageMap.get(item.packageId) : undefined;
          return Boolean(pkg?.sourceFileName);
        }).length,
        description: t('business.deploy.template.hero.offlineReadyHint'),
      },
      {
        label: t('business.deploy.template.hero.scope'),
        value: t('business.deploy.package.hero.scopeValue'),
        description: t('business.deploy.template.hero.scopeHint'),
      },
    ],
    [data, packageMap, t],
  );

  const openCreate = () => {
    setEditing(null);
    setSelectedDefaultPackageId(undefined);
    form.resetFields();
    form.setFieldsValue({
      status: 'enabled',
      executionMode: 'fixed',
      defaultAction: 'install',
      parameterValues: {},
      steps: [{ stepName: '', stepType: 'package', action: 'install' }],
    });
    setVisible(true);
  };

  const openEdit = (row: DeployTemplateRow) => {
    setEditing(row);
    setSelectedDefaultPackageId(row.packageId || undefined);
    form.setFieldsValue({
      name: row.name,
      version: row.version,
      category: row.category,
      status: (row.status as 'enabled' | 'disabled') || 'enabled',
      executionMode: row.executionMode || 'fixed',
      defaultAction: row.defaultAction || 'install',
      packageId: row.packageId || undefined,
      description: row.description,
      parameterValues: buildInitialTemplateValues(row.parameterSchema || {}),
      steps: row.steps?.length
        ? row.steps.map((step) => ({
            stepCode: step.stepCode,
            stepName: step.stepName,
            stepType: step.stepType || 'package',
            packageId: step.packageId || undefined,
            action: step.action,
            parameterValues: buildInitialTemplateValues(step.templateParams || row.parameterSchema || {}),
            scriptContent: String(step.stepConfig?.script || ''),
            precheckCommand: String(step.stepConfig?.precheckCommand || ''),
            postcheckCommand: String(step.stepConfig?.postcheckCommand || ''),
          }))
        : [{ stepName: row.name, stepType: 'package', action: row.defaultAction || 'install', packageId: row.packageId || undefined }],
    });
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    const parameterSchema = buildTemplateDefaultParams(values.parameterValues);
    const payload: DeployTemplatePayload = {
      name: values.name,
      version: values.version,
      category: values.category,
      status: values.status,
      executionMode: values.executionMode,
      defaultAction: values.defaultAction,
      packageId: values.packageId,
      description: values.description,
      parameterSchema,
      steps: (values.steps || []).map((step, index) => {
        const stepPackage = packageMap.get(Number(step.packageId || values.packageId || 0));
        const stepType = step.stepType || 'package';
        return {
          stepCode: (step.stepCode || `step_${index + 1}`).trim(),
          stepName: (step.stepName || stepPackage?.name || `step_${index + 1}`).trim(),
          stepType,
          action: step.action || values.defaultAction,
          packageId: stepType === 'package' ? (step.packageId || values.packageId) : step.packageId,
          templateParams: buildTemplateStepParams(step, stepPackage, parameterSchema),
          stepConfig: buildTemplateStepConfig(step, stepType),
          sort: index + 1,
        };
      }),
    };
    setSubmitting(true);
    try {
      if (editing) {
        await updateDeployTemplate(editing.id, payload);
        Message.success(t('common.updateSuccess'));
      } else {
        await createDeployTemplate(payload);
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
    await deleteDeployTemplate(id);
    Message.success(t('common.deleteSuccess'));
    await loadData();
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((item) => deleteDeployTemplate(Number(item))));
      setSelectedRowKeys([]);
      Message.success(t('common.deleteSuccess'));
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const openDetail = (row: DeployTemplateRow) => {
    setDetailRecord(row);
    setDetailVisible(true);
  };

  const columns: ColumnProps<DeployTemplateRow>[] = [
    { title: t('business.deploy.template.name'), dataIndex: 'name', width: 180 },
    { title: t('business.deploy.template.version'), dataIndex: 'version', width: 120 },
    {
      title: t('business.deploy.template.category'),
      dataIndex: 'category',
      width: 140,
      render: (_: unknown, row) => row.category || '-',
    },
    {
      title: t('business.deploy.template.mode'),
      dataIndex: 'executionMode',
      width: 180,
      render: (_: unknown, row) => (
        <Space size={8} wrap>
          <Tag>{t(`business.deploy.package.executionMode.${row.executionMode}`)}</Tag>
          <Tag color="arcoblue">{t(`business.deploy.task.action.${row.defaultAction}`)}</Tag>
        </Space>
      ),
    },
    {
      title: t('business.deploy.template.steps'),
      dataIndex: 'stepCount',
      width: 320,
      render: (_: unknown, row) => (
        <Space direction="vertical" size={2}>
          <span>{t('business.deploy.template.stepCountValue', { count: row.stepCount || 0 })}</span>
          <Space size={6} wrap>
            {(row.steps || []).length > 0 ? (
              row.steps.map((step) => (
                <Tag
                  key={`${row.id}-${step.id || step.stepCode}`}
                  color={templateStepTypeColorMap[(step.stepType || 'package') as 'package' | 'script'] || 'gray'}
                >
                  {(step.stepName || step.packageName || '-') + ' / ' + t(`business.deploy.template.stepType.${step.stepType || 'package'}`)}
                </Tag>
              ))
            ) : (
              <Typography.Text type="secondary">-</Typography.Text>
            )}
          </Space>
        </Space>
      ),
    },
    {
      title: t('business.deploy.template.package'),
      dataIndex: 'packageName',
      width: 220,
      render: (_: unknown, row) => {
        const pkg = row.packageId ? packageMap.get(row.packageId) : undefined;
        return (
          <Space direction="vertical" size={2}>
            <span>{row.packageName ? `${row.packageName} ${row.packageVersion}` : '-'}</span>
            {pkg?.sourceFileName ? (
              <Typography.Text type="secondary">{pkg.sourceFileName}</Typography.Text>
            ) : null}
          </Space>
        );
      },
    },
    {
      title: t('business.deploy.template.status'),
      dataIndex: 'status',
      width: 100,
      render: (_: unknown, row) => (
        <Tag color={templateStatusColorMap[row.status] || 'gray'}>
          {t(`business.deploy.package.status.${row.status}`)}
        </Tag>
      ),
    },
    { title: t('business.deploy.template.description'), dataIndex: 'description', ellipsis: true },
    {
      title: t('common.action'),
      width: 150,
      fixed: 'right',
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          <Button type="text" size="small" icon={<IconEye />} onClick={() => openDetail(row)}>
            {t('common.detail')}
          </Button>
          {canUpdate ? (
            <Button type="text" size="small" icon={<IconEdit />} onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          ) : null}
          {canDelete ? (
            <Popconfirm title={t('business.deploy.template.deleteConfirm')} onOk={() => handleDelete(row.id)}>
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ) : null}
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
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          icon={<IconCode />}
          eyebrow={t('business.deploy.template.hero.eyebrow')}
          title={t('operations.deploy.template.menu')}
          description={t('business.deploy.template.hero.title')}
          metrics={heroStats.map((item) => ({ key: item.key, label: item.label, value: item.value }))}
          action={(
            <GovernanceRailToggleButton expanded={governanceRail.expanded} onToggle={governanceRail.toggle}>
              {t('business.deploy.template.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          )}
        />
        <FilterPanel>
          <Form layout="inline">
            <Form.Item label={t('common.keyword')}>
              <Input value={keyword} onChange={setKeyword} allowClear />
            </Form.Item>
            <Form.Item label={t('business.deploy.template.status')}>
              <Select value={status} onChange={setStatus} allowClear style={{ width: 140 }}>
                <Select.Option value="enabled">{t('business.deploy.package.status.enabled')}</Select.Option>
                <Select.Option value="disabled">{t('business.deploy.package.status.disabled')}</Select.Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  onClick={() => {
                    setQueryKeyword(keyword);
                    setQueryStatus(status);
                    setSelectedRowKeys([]);
                    setPage(1);
                  }}
                >
                  {t('common.search')}
                </Button>
                <Button
                  onClick={() => {
                    setKeyword('');
                    setStatus('');
                    setQueryKeyword('');
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
                primary={(
                  <Button type="primary" icon={<IconPlus />} onClick={openCreate}>
                    {t('common.add')}
                  </Button>
                )}
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
          {data.length === 0 ? (
            <PageEmpty description={t('business.deploy.template.empty')} />
          ) : (
            <AppTable
              className="system-list__table"
              rowKey="id"
              columns={columns}
              data={data}
              loading={loading}
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
            />
          )}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('business.deploy.template.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.deploy.template.hero.sideLead')}
        noteDescription={t('business.deploy.template.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        title={detailRecord?.name || t('business.deploy.template.detailTitle')}
        visible={detailVisible}
        footer={null}
        size="detail"
        onCancel={() => {
          setDetailVisible(false);
          setDetailRecord(null);
        }}
      >
        {!detailRecord ? (
          <PageEmpty description={t('common.loadFailedDesc')} />
        ) : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.deploy.template.name'), value: detailRecord.name },
                { label: t('business.deploy.template.version'), value: detailRecord.version },
                { label: t('business.deploy.template.category'), value: detailRecord.category || '-' },
                { label: t('business.deploy.template.status'), value: t(`business.deploy.package.status.${detailRecord.status}`) },
                { label: t('business.deploy.template.mode'), value: t(`business.deploy.package.executionMode.${detailRecord.executionMode}`) },
                { label: t('business.deploy.template.defaultAction'), value: t(`business.deploy.task.action.${detailRecord.defaultAction}`) },
                {
                  label: t('business.deploy.template.package'),
                  value: detailRecord.packageName ? `${detailRecord.packageName} ${detailRecord.packageVersion}` : '-',
                },
                { label: t('business.deploy.template.steps'), value: t('business.deploy.template.stepCountValue', { count: detailRecord.stepCount || 0 }) },
              ]}
            />
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.template.description')}</Typography.Text>
                <Typography.Paragraph style={{ marginBottom: 0 }}>
                  {detailRecord.description || '-'}
                </Typography.Paragraph>
              </Space>
            </Card>
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.template.defaultParams')}</Typography.Text>
                {Object.keys(detailRecord.parameterSchema || {}).length ? (
                  <Descriptions
                    column={1}
                    data={Object.entries(detailRecord.parameterSchema || {}).map(([key, value]) => ({
                      label: key,
                      value: String(value ?? '-'),
                    }))}
                  />
                ) : (
                  <PageEmpty description={t('business.deploy.template.detailParamsEmpty')} />
                )}
              </Space>
            </Card>
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.template.steps')}</Typography.Text>
                {(detailRecord.steps || []).length ? (
                  (detailRecord.steps || []).map((step) => (
                    <Card key={`${detailRecord.id}-${step.id || step.stepCode}`} className="page-panel" style={{ padding: 12 }}>
                      <Space direction="vertical" size={6} style={{ width: '100%' }}>
                        <Space wrap>
                          <Tag color={templateStepTypeColorMap[(step.stepType || 'package') as 'package' | 'script'] || 'gray'}>
                            {t(`business.deploy.template.stepType.${step.stepType || 'package'}`)}
                          </Tag>
                          <Tag color="arcoblue">{t(`business.deploy.task.action.${step.action || detailRecord.defaultAction}`)}</Tag>
                          <Tag>{step.stepCode || '-'}</Tag>
                        </Space>
                        <Typography.Text style={{ fontWeight: 500 }}>{step.stepName || '-'}</Typography.Text>
                        <Typography.Text type="secondary">
                          {step.packageName ? `${step.packageName} ${step.packageVersion}` : '-'}
                        </Typography.Text>
                        {step.stepType === 'script' && step.stepConfig ? (
                          <Typography.Paragraph className="deploy-page__log" style={{ marginBottom: 0 }}>
                            {String(step.stepConfig.script || '-')}
                          </Typography.Paragraph>
                        ) : null}
                      </Space>
                    </Card>
                  ))
                ) : (
                  <PageEmpty description={t('business.deploy.template.empty')} />
                )}
              </Space>
            </Card>
          </Space>
        )}
      </AppModal>
      <AppModal
        title={editing ? t('business.deploy.template.editTitle') : t('business.deploy.template.createTitle')}
        visible={visible}
        footer={null}
        onCancel={() => {
          setVisible(false);
          form.resetFields();
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item field="name" label={t('business.deploy.template.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="version" label={t('business.deploy.template.version')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="category" label={t('business.deploy.template.category')}>
            <Input />
          </Form.Item>
          <Form.Item field="status" label={t('business.deploy.template.status')} initialValue="enabled">
            <Select>
              <Select.Option value="enabled">{t('business.deploy.package.status.enabled')}</Select.Option>
              <Select.Option value="disabled">{t('business.deploy.package.status.disabled')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="executionMode" label={t('business.deploy.template.mode')} initialValue="fixed">
            <Select>
              <Select.Option value="fixed">{t('business.deploy.package.executionMode.fixed')}</Select.Option>
              <Select.Option value="orchestrated">{t('business.deploy.package.executionMode.orchestrated')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="defaultAction" label={t('business.deploy.template.defaultAction')} initialValue="install">
            <Select>
              <Select.Option value="install">{t('business.deploy.task.action.install')}</Select.Option>
              <Select.Option value="uninstall">{t('business.deploy.task.action.uninstall')}</Select.Option>
              <Select.Option value="upgrade">{t('business.deploy.task.action.upgrade')}</Select.Option>
              <Select.Option value="reinstall">{t('business.deploy.task.action.reinstall')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="packageId" label={t('business.deploy.template.package')} rules={[{ required: true }]}>
            <Select
              allowClear
              showSearch
              onChange={(value) => {
                const nextPackageId = value || undefined;
                setSelectedDefaultPackageId(nextPackageId);
                const nextPackage = nextPackageId ? packageMap.get(nextPackageId) : undefined;
                const nextEntry = getDeployFixedTemplateCatalogEntry(nextPackage?.templateCode);
                form.setFieldValue('parameterValues', buildInitialTemplateValues(nextEntry ? buildDeployTemplateDefaultParameters(nextEntry.code) : {}));
              }}
            >
              {packages.map((item) => (
                <Select.Option key={item.id} value={item.id}>
                  {item.name} {item.version}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          {selectedDefaultTemplateEntry ? (
            <Form.Item label={t('business.deploy.template.defaultParams')}>
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {selectedDefaultTemplateEntry.parameters.map((item) => (
                  <Form.Item key={item.key} field={`parameterValues.${item.key}`} label={t(item.labelKey)}>
                    {item.secret ? <Input.Password placeholder={item.defaultValue} /> : <Input placeholder={item.defaultValue} />}
                  </Form.Item>
                ))}
              </Space>
            </Form.Item>
          ) : null}
          <Form.Item field="description" label={t('business.deploy.template.description')}>
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </Form.Item>
          <Form.Item label={t('business.deploy.template.steps')}>
            <Form.List field="steps">
              {(fields, { add, remove }) => (
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  {fields.map((field, index) => {
                    const currentStepType = String(
                      (form as unknown as { getFieldValue: (path: string) => unknown }).getFieldValue(`steps[${index}].stepType`) || 'package',
                    ) as 'package' | 'script';
                    const currentPackageId = Number(
                      (form as unknown as { getFieldValue: (path: string) => unknown }).getFieldValue(`steps[${index}].packageId`) ||
                        form.getFieldValue('packageId') ||
                        0,
                    );
                    const stepPackage = packageMap.get(currentPackageId);
                    const isPackageStep = currentStepType === 'package';
                    const stepTemplateEntry = isPackageStep
                      ? getDeployFixedTemplateCatalogEntry(stepPackage?.templateCode)
                      : null;
                    return (
                      <div
                        key={field.key}
                        style={{
                          border: '1px solid var(--color-border-2)',
                          borderRadius: 8,
                          padding: 12,
                        }}
                      >
                        <Space direction="vertical" size={8} style={{ width: '100%' }}>
                          <Space align="start" style={{ width: '100%', justifyContent: 'space-between' }}>
                            <Typography.Text style={{ fontWeight: 600 }}>
                              {t('business.deploy.template.stepTitle', { index: index + 1 })}
                            </Typography.Text>
                            <Popconfirm
                              title={t('common.deleteConfirm')}
                              onOk={() => remove(index)}
                              disabled={fields.length === 1}
                            >
                              <Button
                                type="text"
                                status="danger"
                                icon={<IconDelete />}
                                disabled={fields.length === 1}
                              />
                            </Popconfirm>
                          </Space>
                          <Form.Item field={`steps[${index}].stepName`} label={t('business.deploy.template.stepName')}>
                            <Input />
                          </Form.Item>
                          <Form.Item field={`steps[${index}].stepCode`} label={t('business.deploy.template.stepCode')}>
                            <Input />
                          </Form.Item>
                          <Form.Item field={`steps[${index}].stepType`} label={t('business.deploy.template.stepTypeLabel')}>
                            <Select>
                              <Select.Option value="package">{t('business.deploy.template.stepType.package')}</Select.Option>
                              <Select.Option value="script">{t('business.deploy.template.stepType.script')}</Select.Option>
                            </Select>
                          </Form.Item>
                          <Form.Item field={`steps[${index}].packageId`} label={t('business.deploy.template.stepPackage')}>
                            <Select
                              allowClear
                              showSearch
                              onChange={(value) => {
                                const nextPackage = value ? packageMap.get(Number(value)) : undefined;
                                const nextEntry = getDeployFixedTemplateCatalogEntry(nextPackage?.templateCode);
                                (form as unknown as { setFieldValue: (path: string, value: unknown) => void }).setFieldValue(
                                  `steps[${index}].parameterValues`,
                                  buildInitialTemplateValues(nextEntry ? buildDeployTemplateDefaultParameters(nextEntry.code) : {}),
                                );
                              }}
                            >
                              {packages.map((item) => (
                                <Select.Option key={item.id} value={item.id}>
                                  {item.name} {item.version}
                                </Select.Option>
                              ))}
                            </Select>
                          </Form.Item>
                          <Form.Item field={`steps[${index}].action`} label={t('business.deploy.template.stepAction')}>
                            <Select>
                              <Select.Option value="install">{t('business.deploy.task.action.install')}</Select.Option>
                              <Select.Option value="uninstall">{t('business.deploy.task.action.uninstall')}</Select.Option>
                              <Select.Option value="upgrade">{t('business.deploy.task.action.upgrade')}</Select.Option>
                              <Select.Option value="reinstall">{t('business.deploy.task.action.reinstall')}</Select.Option>
                            </Select>
                          </Form.Item>
                          {currentStepType === 'script' ? (
                            <>
                              <Form.Item
                                field={`steps[${index}].scriptContent`}
                                label={t('business.deploy.template.stepScript')}
                                rules={[{ required: true }]}
                              >
                                <Input.TextArea autoSize={{ minRows: 4, maxRows: 10 }} />
                              </Form.Item>
                              <Form.Item
                                field={`steps[${index}].precheckCommand`}
                                label={t('business.deploy.template.stepPrecheck')}
                              >
                                <Input.TextArea autoSize={{ minRows: 2, maxRows: 5 }} />
                              </Form.Item>
                              <Form.Item
                                field={`steps[${index}].postcheckCommand`}
                                label={t('business.deploy.template.stepPostcheck')}
                              >
                                <Input.TextArea autoSize={{ minRows: 2, maxRows: 5 }} />
                              </Form.Item>
                            </>
                          ) : null}
                          {stepTemplateEntry ? (
                            <>
                              {stepTemplateEntry.parameters.map((item) => (
                                <Form.Item
                                  key={`${field.key}-${item.key}`}
                                  field={`steps[${index}].parameterValues.${item.key}`}
                                  label={t(item.labelKey)}
                                >
                                  {item.secret ? <Input.Password placeholder={item.defaultValue} /> : <Input placeholder={item.defaultValue} />}
                                </Form.Item>
                              ))}
                            </>
                          ) : null}
                        </Space>
                      </div>
                    );
                  })}
                  <Button
                    type="dashed"
                    icon={<IconPlus />}
                    onClick={() => add({ stepName: '', stepType: 'package', action: form.getFieldValue('defaultAction') || 'install' })}
                  >
                    {t('business.deploy.template.addStep')}
                  </Button>
                </Space>
              )}
            </Form.List>
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={() => setVisible(false)} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
    </PageContainer>
  );
}

function buildTemplateDefaultParams(values?: Record<string, string>) {
  const result: Record<string, unknown> = {};
  Object.entries(values || {}).forEach(([key, value]) => {
    if ((value || '').trim()) {
      result[key] = value.trim();
    }
  });
  return result;
}

function buildTemplateStepParams(
  step: NonNullable<TemplateFormValues['steps']>[number],
  pkg: DeployPackageRow | undefined,
  defaults: Record<string, unknown>,
) {
  if ((step.stepType || 'package') === 'script') {
    return { ...defaults };
  }
  const entry = getDeployFixedTemplateCatalogEntry(pkg?.templateCode);
  if (!entry) {
    return {};
  }
  const result: Record<string, unknown> = { ...defaults };
  Object.entries(step.parameterValues || {}).forEach(([key, value]) => {
    if ((value || '').trim()) {
      result[key] = value.trim();
    }
  });
  return result;
}

function buildTemplateStepConfig(
  step: NonNullable<TemplateFormValues['steps']>[number],
  stepType: 'package' | 'script',
) {
  if (stepType !== 'script') {
    return {};
  }
  const result: Record<string, unknown> = {};
  if ((step.scriptContent || '').trim()) {
    result.script = step.scriptContent?.trim();
  }
  if ((step.precheckCommand || '').trim()) {
    result.precheckCommand = step.precheckCommand?.trim();
  }
  if ((step.postcheckCommand || '').trim()) {
    result.postcheckCommand = step.postcheckCommand?.trim();
  }
  return result;
}

function buildInitialTemplateValues(values: Record<string, unknown>) {
  const result: Record<string, string> = {};
  Object.entries(values || {}).forEach(([key, value]) => {
    if (value == null) {
      return;
    }
    result[key] = String(value);
  });
  return result;
}
