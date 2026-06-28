import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Descriptions,
  Divider,
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
import { IconClose, IconEye, IconPlayArrow, IconPlus, IconTool } from '@arco-design/web-react/icon';
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
import { getGroupList, type GroupRow } from '../../cmdb/group/api';
import { getHostList, type HostRow } from '../../cmdb/host/api';
import { getBizScopeOptions, type BizScopeOptionItem } from '../../bizscope/api';
import {
  cancelDeployTask,
  createDeployTask,
  getDeployTaskDetail,
  getDeployPackageList,
  getDeployTaskList,
  getDeployTemplateList,
  startDeployTask,
  type DeployPackageRow,
  type DeployTaskPayload,
  type DeployTaskRow,
  type DeployTemplateRow,
  type StartDeployTaskPayload,
} from '../api';
import { buildDeployTemplateDefaultParameters, getDeployFixedTemplateCatalogEntry } from '../catalog';
import { formatDateTime } from '../../../../core/format/dateTime';
import '../../../system/components/shared/list-page.css';
import '../deploy.css';

type TaskFormValues = {
  name: string;
  sourceKind: 'template' | 'package';
  templateId?: number;
  packageId?: number;
  action: 'install' | 'uninstall' | 'upgrade' | 'reinstall';
  targetType: 'host' | 'group';
  businessScopeId?: number;
  targetIds: number[];
  executorType: 'manual' | 'simulated' | 'agent' | 'ssh';
  templateParams?: Record<string, string>;
  remark?: string;
};

const statusColorMap: Record<string, string> = {
  pending: 'gray',
  running: 'arcoblue',
  success: 'green',
  failed: 'red',
  canceled: 'orange',
};

export default function DeployTaskList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();
  const [form] = Form.useForm<TaskFormValues>();
  const [startForm] = Form.useForm<StartDeployTaskPayload>();
  const [data, setData] = useState<DeployTaskRow[]>([]);
  const [packages, setPackages] = useState<DeployPackageRow[]>([]);
  const [templates, setTemplates] = useState<DeployTemplateRow[]>([]);
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [groups, setGroups] = useState<GroupRow[]>([]);
  const [scopeOptions, setScopeOptions] = useState<BizScopeOptionItem[]>([]);
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
  const [submitting, setSubmitting] = useState(false);
  const [startVisible, setStartVisible] = useState(false);
  const [startSubmitting, setStartSubmitting] = useState(false);
  const [startingTask, setStartingTask] = useState<DeployTaskRow | null>(null);
  const [targetType, setTargetType] = useState<'host' | 'group'>('host');
  const [taskAction, setTaskAction] = useState<'install' | 'uninstall' | 'upgrade' | 'reinstall'>('install');
  const [sourceKind, setSourceKind] = useState<'template' | 'package'>('template');
  const [selectedBusinessScopeId, setSelectedBusinessScopeId] = useState<number | undefined>();
  const [selectedPackage, setSelectedPackage] = useState<DeployPackageRow | null>(null);
  const [selectedTemplate, setSelectedTemplate] = useState<DeployTemplateRow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailTask, setDetailTask] = useState<DeployTaskRow | null>(null);

  const canCreate = hasPerm('business:deploy:task:create');
  const canDetail = hasPerm('business:deploy:task:detail');
  const canStart = hasPerm('business:deploy:task:start');
  const canCancel = hasPerm('business:deploy:task:cancel');

  const packageMap = useMemo(
    () => new Map(packages.map((item) => [item.id, item])),
    [packages],
  );

  const resolveTemplatePackage = useCallback(
    (template?: DeployTemplateRow | null) => {
      if (!template) {
        return null;
      }
      const packageId = template.packageId || template.steps?.[0]?.packageId;
      return packageId ? packageMap.get(packageId) || null : null;
    },
    [packageMap],
  );

  const selectedRuntimePackage = sourceKind === 'template'
    ? resolveTemplatePackage(selectedTemplate)
    : selectedPackage;

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getDeployTaskList({
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

  const loadOptions = useCallback(async () => {
    const [packageResp, templateResp, scopeResp, groupResp] = await Promise.all([
      getDeployPackageList({ page: 1, pageSize: 200, status: 'enabled' }),
      getDeployTemplateList({ page: 1, pageSize: 200, status: 'enabled' }),
      getBizScopeOptions(),
      getGroupList(),
    ]);
    setPackages(packageResp.items);
    setTemplates(templateResp.items);
    setScopeOptions(scopeResp);
    setGroups(flattenGroups(groupResp));
    return {
      packages: packageResp.items,
      templates: templateResp.items,
    };
  }, []);

  const loadScopedHosts = useCallback(async (businessScopeId?: number) => {
    if (!businessScopeId) {
      setHosts([]);
      return;
    }
    const hostResp = await getHostList({
      page: 1,
      pageSize: 100,
      businessScopeId,
    });
    setHosts(hostResp.items.filter((item) => item.status === 'assigned' || item.status === 'online'));
  }, []);

  useEffect(() => {
    queueMicrotask(() => {
      void Promise.all([loadData(), loadOptions()]);
    });
  }, [loadData, loadOptions]);

  const heroStats = useMemo(
    () => [
      { key: 'total', label: t('business.deploy.task.hero.total'), value: total, hint: t('business.deploy.task.hero.totalHint') },
      {
        key: 'running',
        label: t('business.deploy.task.status.running'),
        value: data.filter((item) => item.status === 'running').length,
        hint: t('business.deploy.task.hero.runningHint'),
      },
      {
        key: 'success',
        label: t('business.deploy.task.status.success'),
        value: data.filter((item) => item.status === 'success').length,
        hint: t('business.deploy.task.hero.successHint'),
      },
    ],
    [data, t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.deploy.task.hero.pending'),
        value: data.filter((item) => item.status === 'pending').length,
        description: t('business.deploy.task.hero.pendingHint'),
      },
      {
        label: t('business.deploy.task.hero.failedOrCanceled'),
        value: data.filter((item) => item.status === 'failed' || item.status === 'canceled').length,
        description: t('business.deploy.task.hero.failedOrCanceledHint'),
      },
      {
        label: t('business.deploy.task.hero.targetBreakdown'),
        value: `${data.filter((item) => item.targetType === 'host').length} / ${data.filter((item) => item.targetType === 'group').length}`,
        description: t('business.deploy.task.hero.targetBreakdownHint'),
      },
      {
        label: t('business.deploy.task.hero.executorBreakdown'),
        value: `${data.filter((item) => item.executorType === 'manual').length} / ${data.filter((item) => item.executorType === 'simulated').length} / ${data.filter((item) => item.executorType === 'ssh').length}`,
        description: t('business.deploy.task.hero.executorBreakdownHint'),
      },
    ],
    [data, t],
  );

  const targetOptions = useMemo(
    () =>
      targetType === 'host'
        ? hosts.map((item) => ({
            id: item.id,
            label: `${item.hostname} ${item.ip} · ${t(`business.cmdb.host.status.${item.status}`)}`,
          }))
        : groups.map((item) => ({ id: item.id, label: item.name })),
    [groups, hosts, t, targetType],
  );

  const selectedRuntimeIsNginxTemplate =
    selectedRuntimePackage?.executionMode === 'fixed' &&
    selectedRuntimePackage?.templateCode === 'nginx_systemd';
  const selectedRuntimeTemplateEntry = useMemo(
    () => getDeployFixedTemplateCatalogEntry(selectedRuntimePackage?.templateCode),
    [selectedRuntimePackage?.templateCode],
  );

  const runtimeParameterSchema = useMemo(() => {
    if (sourceKind === 'template' && selectedTemplate?.parameterSchema) {
      return selectedTemplate.parameterSchema as Record<string, unknown>;
    }
    if (selectedRuntimeTemplateEntry) {
      return buildDeployTemplateDefaultParameters(selectedRuntimeTemplateEntry.code);
    }
    return {};
  }, [selectedRuntimeTemplateEntry, selectedTemplate, sourceKind]);

  const openCreate = async () => {
    setTargetType('host');
    setTaskAction('install');
    setSourceKind('template');
    setSelectedBusinessScopeId(undefined);
    setSelectedPackage(null);
    setSelectedTemplate(null);
    setHosts([]);
    form.resetFields();
    const options = await loadOptions();
    const defaultTemplate = options.templates[0] || null;
    const nextPackage = defaultTemplate
      ? options.packages.find((item) => item.id === (defaultTemplate.packageId || defaultTemplate.steps?.[0]?.packageId)) || null
      : null;
    setSelectedTemplate(defaultTemplate);
    setSelectedPackage(null);
    form.setFieldsValue({
      sourceKind: defaultTemplate ? 'template' : 'package',
      templateId: defaultTemplate?.id,
      targetType: 'host',
      executorType: nextPackage?.templateCode === 'nginx_systemd' ? 'ssh' : 'manual',
      action: defaultTemplate?.defaultAction || 'install',
      templateParams: buildInitialTaskTemplateParams(defaultTemplate?.parameterSchema || {}),
      targetIds: [],
    });
    setSourceKind(defaultTemplate ? 'template' : 'package');
    setTaskAction((defaultTemplate?.defaultAction as TaskFormValues['action']) || 'install');
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    const payload: DeployTaskPayload = {
      name: values.name,
      action: values.action,
      targetType: values.targetType,
      targetIds: values.targetIds,
      executorType: values.executorType,
      businessScopeId: values.targetType === 'host' ? values.businessScopeId : undefined,
      remark: values.remark,
    };
    if (values.sourceKind === 'template') {
      payload.templateId = values.templateId;
    } else {
      payload.packageId = values.packageId as number;
    }
    if (Object.keys(runtimeParameterSchema).length > 0) {
      payload.templateParams = {
        action: values.action || 'install',
        ...buildTaskTemplateParams(values.templateParams, runtimeParameterSchema),
      };
    }
    setSubmitting(true);
    try {
      await createDeployTask(payload);
      Message.success(t('common.createSuccess'));
      setVisible(false);
      setSelectedBusinessScopeId(undefined);
      setSelectedPackage(null);
      setSelectedTemplate(null);
      setHosts([]);
      form.resetFields();
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleStart = async (row: DeployTaskRow, payload?: StartDeployTaskPayload) => {
    await startDeployTask(row.id, payload);
    Message.success(t('business.deploy.task.startSuccess'));
    await loadData();
  };

  const openStart = (row: DeployTaskRow) => {
    if (row.executorType !== 'ssh') {
      handleStart(row);
      return;
    }
    setStartingTask(row);
    startForm.resetFields();
    startForm.setFieldsValue({ authMode: 'password' });
    setStartVisible(true);
  };

  const submitStart = async () => {
    if (!startingTask) {
      return;
    }
    const values = await startForm.validate();
    setStartSubmitting(true);
    try {
      await handleStart(startingTask, values);
      setStartVisible(false);
      setStartingTask(null);
      startForm.resetFields();
    } finally {
      setStartSubmitting(false);
    }
  };

  const handleCancel = async (id: number) => {
    await cancelDeployTask(id);
    Message.success(t('business.deploy.task.cancelSuccess'));
    await loadData();
  };

  const handleBatchCancel = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      const candidates = data.filter(
        (item) =>
          selectedRowKeys.includes(item.id) && ['pending', 'running'].includes(item.status),
      );
      await Promise.all(candidates.map((item) => cancelDeployTask(item.id)));
      setSelectedRowKeys([]);
      Message.success(t('business.deploy.task.cancelSuccess'));
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const openDetail = async (id: number) => {
    setDetailVisible(true);
    setDetailLoading(true);
    try {
      setDetailTask(await getDeployTaskDetail(id));
    } finally {
      setDetailLoading(false);
    }
  };

  const columns: ColumnProps<DeployTaskRow>[] = [
    { title: t('business.deploy.task.name'), dataIndex: 'name', width: 180 },
    {
      title: t('business.deploy.task.source'),
      width: 240,
      render: (_: unknown, row) => (
        <Space direction="vertical" size={2}>
          {row.templateName ? (
            <span>{`${row.templateName} ${row.templateVersion}`}</span>
          ) : (
            <span>{t('business.deploy.task.sourcePackage')}</span>
          )}
          <Typography.Text type="secondary">
            {`${row.packageName} ${row.packageVersion}`}
          </Typography.Text>
        </Space>
      ),
    },
    {
      title: t('business.deploy.task.action'),
      dataIndex: 'action',
      width: 120,
      render: (_: unknown, row) => t(`business.deploy.task.action.${row.action || 'install'}`),
    },
    {
      title: t('business.deploy.task.targetType'),
      dataIndex: 'targetType',
      width: 120,
      render: (_: unknown, row) => t(`business.deploy.task.targetType.${row.targetType}`),
    },
    {
      title: t('business.deploy.task.businessScope'),
      dataIndex: 'businessScopeName',
      width: 160,
      render: (_: unknown, row) => row.businessScopeName || '-',
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
      title: t('business.deploy.task.hostCount'),
      dataIndex: 'hostCount',
      width: 100,
    },
    {
      title: t('common.action'),
      fixed: 'right',
      width: 260,
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          {canDetail && (
            <Button
              type="text"
              size="small"
              icon={<IconEye />}
              onClick={() => {
                openDetail(row.id);
              }}
            >
              {t('common.detail')}
            </Button>
          )}
          {canStart && row.status === 'pending' && (
            row.executorType === 'ssh' ? (
              <Button type="text" size="small" icon={<IconPlayArrow />} onClick={() => openStart(row)}>
                {t('business.deploy.task.start')}
              </Button>
            ) : (
              <Popconfirm title={t('business.deploy.task.startConfirm')} onOk={() => openStart(row)}>
                <Button type="text" size="small" icon={<IconPlayArrow />}>{t('business.deploy.task.start')}</Button>
              </Popconfirm>
            )
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
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          icon={<IconTool />}
          eyebrow={t('business.deploy.task.hero.eyebrow')}
          title={t('operations.deploy.task.menu')}
          description={t('business.deploy.task.hero.title')}
          metrics={heroStats.map((item) => ({ key: item.key, label: item.label, value: item.value }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('business.deploy.task.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
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
                <Button
                  type="primary"
                  onClick={() => {
                    setQueryKeyword(keyword);
                    setQueryStatus(status);
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
                  <Button type="primary" icon={<IconPlus />} onClick={() => openCreate()}>
                    {t('common.add')}
                  </Button>
                }
              />
            ) : undefined
          }
          actions={
            canCancel ? (
              <Popconfirm
                title={t('business.deploy.task.cancelConfirm')}
                onOk={() => {
                  handleBatchCancel();
                }}
                disabled={
                  selectedRowKeys.length === 0 ||
                  !data.some(
                    (item) =>
                      selectedRowKeys.includes(item.id) &&
                      ['pending', 'running'].includes(item.status),
                  ) ||
                  submitting
                }
              >
                <Button
                  status="danger"
                  icon={<IconClose />}
                  disabled={
                    selectedRowKeys.length === 0 ||
                    !data.some(
                      (item) =>
                        selectedRowKeys.includes(item.id) &&
                        ['pending', 'running'].includes(item.status),
                    ) ||
                    submitting
                  }
                  loading={submitting}
                >
                  {t('business.deploy.task.cancel')}
                </Button>
              </Popconfirm>
            ) : undefined
          }
        />
        <Card className="page-panel system-list__table-card">
          {data.length === 0 ? (
            <PageEmpty description={t('business.deploy.task.empty')} />
          ) : (
            <AppTable
              rowKey="id"
              className="system-list__table"
              loading={loading}
              columns={columns}
              data={data}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
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
        title={t('business.deploy.task.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.deploy.task.hero.sideLead')}
        noteDescription={t('business.deploy.task.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        title={detailTask?.name || t('operations.deploy.task.detail')}
        visible={detailVisible}
        footer={null}
        size="detail"
        onCancel={() => {
          setDetailVisible(false);
          setDetailTask(null);
        }}
      >
        {detailLoading ? <PageLoading /> : null}
        {!detailLoading && !detailTask ? (
          <PageEmpty description={t('common.loadFailedDesc')} />
        ) : null}
        {!detailLoading && detailTask ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions
              column={2}
              data={[
                {
                  label: t('business.deploy.task.source'),
                  value: detailTask.templateName
                    ? `${detailTask.templateName} ${detailTask.templateVersion}`
                    : t('business.deploy.task.sourcePackage'),
                },
                {
                  label: t('business.deploy.task.package'),
                  value: `${detailTask.packageName} ${detailTask.packageVersion}`,
                },
                {
                  label: t('business.deploy.task.action'),
                  value: t(`business.deploy.task.action.${detailTask.action || 'install'}`),
                },
                {
                  label: t('business.deploy.task.status'),
                  value: t(`business.deploy.task.status.${detailTask.status}`),
                },
                {
                  label: t('business.deploy.task.targetType'),
                  value: t(`business.deploy.task.targetType.${detailTask.targetType}`),
                },
                {
                  label: t('business.deploy.task.businessScope'),
                  value: detailTask.businessScopeName || '-',
                },
                {
                  label: t('business.deploy.task.executorType'),
                  value: t(`business.deploy.task.executorType.${detailTask.executorType}`),
                },
                {
                  label: t('business.deploy.task.startedAt'),
                  value: detailTask.startedAt || '-',
                },
                {
                  label: t('business.deploy.task.finishedAt'),
                  value: detailTask.finishedAt || '-',
                },
                {
                  label: t('business.deploy.task.duration'),
                  value: detailTask.durationSeconds ? `${detailTask.durationSeconds}s` : '-',
                },
                { label: t('business.deploy.task.hostCount'), value: detailTask.hostCount || 0 },
                { label: t('business.deploy.task.remark'), value: detailTask.remark || '-' },
              ]}
            />
            {(detailTask.hosts || []).length > 0 ? (
              detailTask.hosts.map((host) => (
                <Card key={host.id} className="page-panel">
                  <Space direction="vertical" size={12} style={{ width: '100%' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', gap: 12 }}>
                      <Space direction="vertical" size={2}>
                        <Typography.Text style={{ fontWeight: 600 }}>{host.hostname}</Typography.Text>
                        <Typography.Text type="secondary">
                          {host.hostIp}
                          {host.startedAt ? ` / ${formatDateTime(host.startedAt)}` : ''}
                          {host.durationSeconds ? ` / ${host.durationSeconds}s` : ''}
                        </Typography.Text>
                      </Space>
                      <Tag color={statusColorMap[host.status] || 'gray'}>
                        {t(`business.deploy.task.hostStatus.${host.status}`)}
                      </Tag>
                    </div>
                    <AppTable
                      rowKey={(item) => `${host.id}-${item.phase || ''}-${item.at || ''}`}
                      columns={[
                        {
                          title: t('business.deploy.task.step'),
                          width: 220,
                          render: (_: unknown, row: Record<string, string | undefined>) => (
                            <Space direction="vertical" size={2}>
                              <span>{row.stepName || row.packageName || '-'}</span>
                              <Typography.Text type="secondary">
                                {row.action ? t(`business.deploy.task.action.${row.action}`) : '-'}
                              </Typography.Text>
                            </Space>
                          ),
                        },
                        {
                          title: t('business.deploy.task.phase'),
                          dataIndex: 'phase',
                          width: 120,
                          render: (value: unknown) =>
                            value ? <Tag>{t(`business.deploy.task.phase.${String(value)}`)}</Tag> : '-',
                        },
                        {
                          title: t('business.deploy.task.startedAt'),
                          dataIndex: 'at',
                          width: 180,
                          render: (value: unknown) =>
                            value ? formatDateTime(String(value)) : '-',
                        },
                        { title: t('business.deploy.task.message'), dataIndex: 'message', ellipsis: true },
                      ]}
                      data={host.traceSteps || []}
                      pagination={false}
                      size="small"
                    />
                    <Divider style={{ margin: '0' }} />
                    <Descriptions
                      column={1}
                      data={[
                        {
                          label: t('business.deploy.task.stdout'),
                          value: (
                            <Typography.Paragraph className="deploy-page__log" copyable={Boolean(host.stdout)}>
                              {host.stdout || '-'}
                            </Typography.Paragraph>
                          ),
                        },
                        {
                          label: t('business.deploy.task.stderr'),
                          value: (
                            <Typography.Paragraph className="deploy-page__log" copyable={Boolean(host.stderr)}>
                              {host.stderr || '-'}
                            </Typography.Paragraph>
                          ),
                        },
                      ]}
                    />
                  </Space>
                </Card>
              ))
            ) : (
              <PageEmpty description={t('business.deploy.task.hostEmpty')} />
            )}
          </Space>
        ) : null}
      </AppModal>
      <AppModal
        title={t('business.deploy.task.createTitle')}
        visible={visible}
        footer={null}
        onCancel={() => {
          setVisible(false);
          setSelectedBusinessScopeId(undefined);
          setSelectedPackage(null);
          setSelectedTemplate(null);
          setHosts([]);
          form.resetFields();
        }}
      >
        <Form form={form} layout="vertical">
          <Form.Item field="name" label={t('business.deploy.task.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="sourceKind" label={t('business.deploy.task.source')} initialValue="template">
            <Select
              onChange={(value) => {
                const nextSource = value as 'template' | 'package';
                setSourceKind(nextSource);
                setSelectedTemplate(null);
                setSelectedPackage(null);
                form.setFieldValue('templateId', undefined);
                form.setFieldValue('packageId', undefined);
                form.setFieldValue('templateParams', {});
                form.setFieldValue('executorType', 'manual');
              }}
            >
              <Select.Option value="template">{t('business.deploy.task.sourceTemplate')}</Select.Option>
              <Select.Option value="package">{t('business.deploy.task.sourcePackage')}</Select.Option>
            </Select>
          </Form.Item>
          {sourceKind === 'template' ? (
            <Form.Item field="templateId" label={t('operations.deploy.template.menu')} rules={[{ required: true }]}>
              <Select
                showSearch
                allowClear
                onChange={(value) => {
                  const nextTemplate = templates.find((item) => item.id === value) || null;
                  const nextPackage = resolveTemplatePackage(nextTemplate);
                  setSelectedTemplate(nextTemplate);
                  setSelectedPackage(null);
                  form.setFieldValue('action', nextTemplate?.defaultAction || 'install');
                  form.setFieldValue('executorType', nextPackage?.templateCode === 'nginx_systemd' ? 'ssh' : 'manual');
                  form.setFieldValue('templateParams', buildInitialTaskTemplateParams(nextTemplate?.parameterSchema || {}));
                  setTaskAction((nextTemplate?.defaultAction as TaskFormValues['action']) || 'install');
                }}
              >
                {templates.map((item) => (
                  <Select.Option key={item.id} value={item.id}>
                    {item.name} {item.version}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
          ) : (
            <Form.Item field="packageId" label={t('business.deploy.task.package')} rules={[{ required: true }]}>
              <Select
                showSearch
                allowClear
                onChange={(value) => {
                  const nextPackage = packages.find((item) => item.id === value) || null;
                  setSelectedPackage(nextPackage);
                  setSelectedTemplate(null);
                  const nextTemplateEntry = getDeployFixedTemplateCatalogEntry(nextPackage?.templateCode);
                  if (nextTemplateEntry) {
                    form.setFieldValue('executorType', 'ssh');
                    form.setFieldValue('templateParams', buildInitialTaskTemplateParams(buildDeployTemplateDefaultParameters(nextTemplateEntry.code)));
                  } else {
                    form.setFieldValue('executorType', 'manual');
                    form.setFieldValue('templateParams', {});
                  }
                }}
              >
                {packages.map((item) => (
                  <Select.Option key={item.id} value={item.id}>{item.name} {item.version}</Select.Option>
                ))}
              </Select>
            </Form.Item>
          )}
          {selectedRuntimePackage ? (
            <div style={{ marginBottom: 16, padding: 12, borderRadius: 6, background: 'var(--color-fill-1)' }}>
              <Space direction="vertical" size={4}>
                <Typography.Text style={{ fontWeight: 500 }}>
                  {`${selectedRuntimePackage.name} ${selectedRuntimePackage.version}`}
                </Typography.Text>
                <Typography.Text type="secondary">
                  {sourceKind === 'template'
                    ? t('business.deploy.task.templateSourceHint', { count: selectedTemplate?.stepCount || 0 })
                    : t('business.deploy.task.packageSourceHint')}
                </Typography.Text>
              </Space>
            </div>
          ) : null}
          <Form.Item field="action" label={t('business.deploy.task.action')} initialValue="install">
            <Select
              onChange={(value) => {
                const nextAction = value as TaskFormValues['action'];
                setTaskAction(nextAction);
              }}
            >
              <Select.Option value="install">{t('business.deploy.task.action.install')}</Select.Option>
              <Select.Option value="uninstall">{t('business.deploy.task.action.uninstall')}</Select.Option>
              <Select.Option value="upgrade">{t('business.deploy.task.action.upgrade')}</Select.Option>
              <Select.Option value="reinstall">{t('business.deploy.task.action.reinstall')}</Select.Option>
            </Select>
          </Form.Item>
          {Object.keys(runtimeParameterSchema).length > 0 ? (
            <>
              <div style={{ marginBottom: 16, padding: 12, borderRadius: 6, background: 'var(--color-fill-1)', color: 'var(--color-text-3)', fontSize: 12 }}>
                {selectedRuntimeIsNginxTemplate
                  ? t('business.deploy.task.templateParams.nginxHint')
                  : t('business.deploy.task.templateParams.dynamicHint')}
              </div>
              {Object.entries(runtimeParameterSchema).map(([key, value]) => (
                <Form.Item
                  key={key}
                  field={`templateParams.${key}`}
                  label={buildTemplateParamLabel(t, key)}
                  rules={taskAction === 'uninstall' ? [] : [{ required: true }]}
                >
                  <Input placeholder={String(value || '')} />
                </Form.Item>
              ))}
            </>
          ) : null}
          <Form.Item field="targetType" label={t('business.deploy.task.targetType')} initialValue="host">
            <Select
              onChange={(value) => {
                const nextType = value as 'host' | 'group';
                setTargetType(nextType);
                setSelectedBusinessScopeId(undefined);
                setHosts([]);
                form.setFieldValue('businessScopeId', undefined);
                form.setFieldValue('targetIds', []);
              }}
            >
              <Select.Option value="host">{t('business.deploy.task.targetType.host')}</Select.Option>
              <Select.Option value="group">{t('business.deploy.task.targetType.group')}</Select.Option>
            </Select>
          </Form.Item>
          {targetType === 'host' ? (
            <Form.Item
              field="businessScopeId"
              label={t('business.deploy.task.businessScope')}
              rules={[{ required: true }]}
            >
              <Select
                allowClear
                placeholder={t('business.deploy.task.businessScopePlaceholder')}
                onChange={(value) => {
                  const nextScopeId = value || undefined;
                  setSelectedBusinessScopeId(nextScopeId);
                  form.setFieldValue('targetIds', []);
                  loadScopedHosts(nextScopeId);
                }}
              >
                {scopeOptions.map((item) => (
                  <Select.Option key={item.id} value={item.id}>
                    {item.name}
                  </Select.Option>
                ))}
              </Select>
            </Form.Item>
          ) : null}
          <Form.Item field="targetIds" label={t('business.deploy.task.targets')} rules={[{ required: true }]}>
            <Select
              mode="multiple"
              disabled={targetType === 'host' && !selectedBusinessScopeId}
              placeholder={
                targetType === 'host' && !selectedBusinessScopeId
                  ? t('business.deploy.task.targetsPlaceholder')
                  : undefined
              }
            >
              {targetOptions.map((item) => (
                <Select.Option key={item.id} value={item.id}>
                  {item.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item field="executorType" label={t('business.deploy.task.executorType')} initialValue="manual">
            <Select disabled={Boolean(selectedRuntimeTemplateEntry)}>
              <Select.Option value="manual">{t('business.deploy.task.executorType.manual')}</Select.Option>
              <Select.Option value="simulated">{t('business.deploy.task.executorType.simulated')}</Select.Option>
              <Select.Option value="agent">{t('business.deploy.task.executorType.agent')}</Select.Option>
              <Select.Option value="ssh">{t('business.deploy.task.executorType.ssh')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="remark" label={t('business.deploy.task.remark')}>
            <Input.TextArea autoSize={{ minRows: 2, maxRows: 4 }} />
          </Form.Item>
          <SubmitBar loading={submitting} onCancel={() => setVisible(false)} onSubmit={handleSubmit} />
        </Form>
      </AppModal>
      <AppModal
        title={t('business.deploy.task.startSshTitle')}
        visible={startVisible}
        footer={null}
        onCancel={() => {
          setStartVisible(false);
          setStartingTask(null);
          startForm.resetFields();
        }}
      >
        <Form form={startForm} layout="vertical">
          <div style={{ color: 'var(--color-text-3)', marginBottom: 16 }}>
            {t('business.deploy.task.startSshHint')}
          </div>
          {startingTask?.templateParams && Object.keys(startingTask.templateParams).length > 0 ? (
            <div style={{ color: 'var(--color-text-3)', marginBottom: 16 }}>
              {t('business.deploy.task.startSshTemplateHint')}
            </div>
          ) : null}
          <Form.Item field="sshUser" label={t('business.deploy.task.sshUser')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item field="authMode" label={t('business.deploy.task.authMode')} rules={[{ required: true }]}>
            <Select>
              <Select.Option value="password">{t('business.deploy.task.authMode.password')}</Select.Option>
              <Select.Option value="private_key">{t('business.deploy.task.authMode.privateKey')}</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item field="hostFingerprint" label={t('business.deploy.task.hostFingerprint')} rules={[{ required: true }]}>
            <Input placeholder={t('business.deploy.task.hostFingerprintPlaceholder')} />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, next) => prev.authMode !== next.authMode}>
            {(values) =>
              values.authMode === 'private_key' ? (
                <Form.Item field="sshPrivateKey" label={t('business.deploy.task.privateKey')} rules={[{ required: true }]}>
                  <Input.TextArea autoSize={{ minRows: 4, maxRows: 8 }} />
                </Form.Item>
              ) : (
                <Form.Item field="sshPassword" label={t('business.deploy.task.sshPassword')} rules={[{ required: true }]}>
                  <Input.Password />
                </Form.Item>
              )
            }
          </Form.Item>
          <SubmitBar
            loading={startSubmitting}
            onCancel={() => {
              setStartVisible(false);
              setStartingTask(null);
            }}
            onSubmit={submitStart}
          />
        </Form>
      </AppModal>
    </PageContainer>
  );
}

function buildInitialTaskTemplateParams(schema: Record<string, unknown>) {
  const result: Record<string, string> = {};
  Object.entries(schema || {}).forEach(([key, value]) => {
    if (value == null) {
      return;
    }
    result[key] = String(value);
  });
  return result;
}

function buildTaskTemplateParams(
  params: Record<string, string> | undefined,
  schema: Record<string, unknown>,
) {
  const result: Record<string, string> = {};
  Object.entries(schema || {}).forEach(([key, value]) => {
    const currentValue = params?.[key];
    if (typeof currentValue === 'string' && currentValue.trim()) {
      result[key] = currentValue.trim();
      return;
    }
    if (value != null && String(value).trim()) {
      result[key] = String(value).trim();
    }
  });
  return result;
}

function buildTemplateParamLabel(t: (key: string) => string, key: string) {
  if (key === 'installRoot') {
    return t('business.deploy.task.templateParams.installRoot');
  }
  if (key === 'serviceName') {
    return t('business.deploy.task.templateParams.serviceName');
  }
  return key;
}

function flattenGroups(groups: GroupRow[]): GroupRow[] {
  return groups.flatMap((group) => [group, ...flattenGroups(group.children || [])]);
}
