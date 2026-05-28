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
import { IconApps, IconDelete, IconEdit, IconEye, IconPlus, IconUpload } from '@arco-design/web-react/icon';
import { uploadSystemFile } from '../../../../api/upload';
import { formatDateTime } from '../../../../core/format/dateTime';
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
import { Descriptions } from '@arco-design/web-react';
import { usePermission } from '../../../../hooks/usePermission';
import {
  createDeployPackage,
  deleteDeployPackage,
  getDeployPackageList,
  updateDeployPackage,
  type DeployPackagePayload,
  type DeployPackageRow,
} from '../api';
import { deployFixedTemplateCatalog, getDeployFixedTemplateCatalogEntry } from '../catalog';
import '../../../system/list-page.css';
import '../deploy.css';

export default function DeployPackageList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();
  const [form] = Form.useForm();
  const [data, setData] = useState<DeployPackageRow[]>([]);
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
  const [editing, setEditing] = useState<DeployPackageRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [executionMode, setExecutionMode] = useState<'fixed' | 'orchestrated'>('fixed');
  const [templateCode, setTemplateCode] = useState('nginx_systemd');
  const [uploadingSource, setUploadingSource] = useState(false);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailRecord, setDetailRecord] = useState<DeployPackageRow | null>(null);
  const currentTemplateEntry = useMemo(
    () => getDeployFixedTemplateCatalogEntry(templateCode),
    [templateCode],
  );

  const canCreate = hasPerm('business:deploy:package:create');
  const canUpdate = hasPerm('business:deploy:package:update');
  const canDelete = hasPerm('business:deploy:package:delete');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getDeployPackageList({
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

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.deploy.package.hero.scope'),
        value: t('business.deploy.package.hero.scopeValue'),
        description: t('business.deploy.package.hero.scopeHint'),
      },
      {
        label: t('business.deploy.package.hero.installReady'),
        value: data.filter((item) => Boolean(item.installCommand)).length,
        description: t('business.deploy.package.hero.installReadyHint'),
      },
      {
        label: t('business.deploy.package.hero.uninstallReady'),
        value: data.filter((item) => Boolean(item.uninstallCommand)).length,
        description: t('business.deploy.package.hero.uninstallReadyHint'),
      },
      {
        label: t('business.deploy.package.hero.enabledRatio'),
        value: total > 0 ? `${Math.round((data.filter((item) => item.status === 'enabled').length / total) * 100)}%` : '-',
        description: t('business.deploy.package.hero.enabledRatioHint'),
      },
    ],
    [data, t, total],
  );

  const openCreate = () => {
    setEditing(null);
    setExecutionMode('fixed');
    setTemplateCode('nginx_systemd');
    form.setFieldsValue({
      status: 'enabled',
      executionMode: 'fixed',
      templateCode: 'nginx_systemd',
      templateConfig: buildDefaultTemplateConfig('nginx_systemd'),
    });
    setVisible(true);
  };

  const openEdit = (row: DeployPackageRow) => {
    setEditing(row);
    setExecutionMode(row.executionMode || 'fixed');
    setTemplateCode(row.templateCode || 'nginx_systemd');
    form.setFieldsValue({
      ...row,
      templateConfig: row.templateConfig && Object.keys(row.templateConfig).length > 0
        ? row.templateConfig
        : buildDefaultTemplateConfig((row.templateCode as string) || 'nginx_systemd'),
    });
    setVisible(true);
  };

  const handleSubmit = async () => {
    const values = await form.validate();
    const payload: DeployPackagePayload = { ...values };
    if (payload.executionMode !== 'fixed') {
      payload.templateCode = '';
      payload.templateConfig = {};
    } else {
      payload.installCommand = '';
      payload.templateConfig = payload.templateConfig || { scenario: 'systemd' };
    }
    setSubmitting(true);
    try {
      if (editing) {
        await updateDeployPackage(editing.id, payload);
        Message.success(t('common.updateSuccess'));
      } else {
        await createDeployPackage(payload);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setExecutionMode('fixed');
      setTemplateCode('nginx_systemd');
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

  const openDetail = (row: DeployPackageRow) => {
    setDetailVisible(true);
    setDetailRecord(row);
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => deleteDeployPackage(Number(rowKey))));
      Message.success(t('common.deleteSuccess'));
      setSelectedRowKeys([]);
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleUploadSource = async (file?: File | null) => {
    if (!file) {
      return;
    }
    setUploadingSource(true);
    try {
      const uploaded = await uploadSystemFile(file, 'deploy/package');
      form.setFieldValue('sourceObjectKey', uploaded.objectKey);
      form.setFieldValue('sourceFileName', uploaded.originalName);
      form.setFieldValue('sourceUrl', uploaded.url);
      Message.success(t('common.uploadSuccess'));
    } finally {
      setUploadingSource(false);
    }
  };

  const columns: ColumnProps<DeployPackageRow>[] = [
    { title: t('business.deploy.package.name'), dataIndex: 'name', width: 160 },
    { title: t('business.deploy.package.version'), dataIndex: 'version', width: 120 },
    {
      title: t('business.deploy.package.executionMode'),
      dataIndex: 'executionMode',
      width: 220,
      render: (_: unknown, row) => (
        <Space size={8} wrap>
          <Tag>{t(`business.deploy.package.executionMode.${row.executionMode}`)}</Tag>
          {row.templateCode ? (
            <Tag color="arcoblue">
              {t(`business.deploy.package.templateCode.${row.templateCode}`)}
            </Tag>
          ) : null}
          {row.sourceFileName ? <Tag color="green">{row.sourceFileName}</Tag> : null}
        </Space>
      ),
    },
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
    {
      title: t('business.deploy.package.latestDeployedAt'),
      dataIndex: 'latestDeployedAt',
      width: 180,
      render: (_: unknown, row) => row.latestDeployedAt ? formatDateTime(row.latestDeployedAt) : '-',
    },
    {
      title: t('business.deploy.package.latestTask'),
      dataIndex: 'latestTaskName',
      width: 200,
      render: (_: unknown, row) =>
        row.latestTaskName ? (
          <Space direction="vertical" size={2}>
            <span>{row.latestTaskName}</span>
            <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
              {row.latestTaskStatus || '-'} · {row.latestHostCount || 0}/{row.latestSuccessCount || 0}
            </span>
          </Space>
        ) : (
          '-'
        ),
    },
    { title: t('business.deploy.package.description'), dataIndex: 'description', ellipsis: true },
    {
      title: t('common.action'),
      fixed: 'right',
      width: 150,
      render: (_: unknown, row) => (
        <Space className="system-list__actions">
          <Button type="text" size="small" icon={<IconEye />} onClick={() => openDetail(row)}>
            {t('common.detail')}
          </Button>
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
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          icon={<IconApps />}
          eyebrow={t('business.deploy.package.hero.eyebrow')}
          title={t('operations.deploy.package.menu')}
          description={t('business.deploy.package.hero.title')}
          metrics={heroStats.map((item) => ({ key: item.key, label: item.label, value: item.value }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('business.deploy.package.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
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
            <PageEmpty description={t('business.deploy.package.empty')} />
          ) : (
            <AppTable
              className="system-list__table"
              rowKey="id"
              loading={loading}
              columns={columns}
              data={data}
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
                onChange: (nextPage) => {
                  setPage(nextPage || 1);
                },
                onPageSizeChange: (nextPageSize) => {
                  setPageSize(nextPageSize || pageSize);
                  setPage(1);
                },
                pageSizeChangeResetCurrent: true,
              })}
            />
          )}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('business.deploy.package.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.deploy.package.hero.sideLead')}
        noteDescription={t('business.deploy.package.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        title={detailRecord?.name || t('business.deploy.package.detailTitle')}
        visible={detailVisible}
        footer={null}
        size="detail"
        onCancel={() => {
          setDetailVisible(false);
          setDetailRecord(null);
        }}
      >
        {!detailRecord ? <PageEmpty description={t('common.loadFailedDesc')} /> : (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.deploy.package.name'), value: detailRecord.name },
                { label: t('business.deploy.package.version'), value: detailRecord.version },
                { label: t('business.deploy.package.executionMode'), value: t(`business.deploy.package.executionMode.${detailRecord.executionMode}`) },
                { label: t('business.deploy.package.status'), value: t(`business.deploy.package.status.${detailRecord.status}`) },
                { label: t('business.deploy.package.latestDeployedAt'), value: detailRecord.latestDeployedAt ? formatDateTime(detailRecord.latestDeployedAt) : '-' },
                { label: t('business.deploy.package.latestTask'), value: detailRecord.latestTaskName || '-' },
                { label: t('business.deploy.package.sourceFile'), value: detailRecord.sourceFileName || '-' },
                { label: t('business.deploy.package.sourceUrl'), value: detailRecord.sourceUrl || '-' },
              ]}
            />
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.package.description')}</Typography.Text>
                <Typography.Paragraph style={{ marginBottom: 0 }}>
                  {detailRecord.description || '-'}
                </Typography.Paragraph>
              </Space>
            </Card>
            {detailRecord.executionMode === 'fixed' ? (
              <Card className="page-panel">
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.package.templateSummary')}</Typography.Text>
                  <Typography.Text type="secondary">{t(`business.deploy.package.templateCode.${detailRecord.templateCode || 'nginx_systemd'}`)}</Typography.Text>
                </Space>
              </Card>
            ) : null}
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>{t('business.deploy.package.installCommand')}</Typography.Text>
                <Typography.Paragraph className="deploy-page__log" copyable={Boolean(detailRecord.installCommand)} style={{ marginBottom: 0 }}>
                  {detailRecord.installCommand || '-'}
                </Typography.Paragraph>
              </Space>
            </Card>
          </Space>
        )}
      </AppModal>
      <AppModal
        title={editing ? t('business.deploy.package.editTitle') : t('business.deploy.package.createTitle')}
        visible={visible}
        footer={null}
        onCancel={() => {
          setVisible(false);
          setExecutionMode('fixed');
          setTemplateCode('nginx_systemd');
        }}
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
          <Form.Item field="executionMode" label={t('business.deploy.package.executionMode')} initialValue="fixed">
            <Select
              onChange={(value) => {
                const nextMode = value as 'fixed' | 'orchestrated';
                setExecutionMode(nextMode);
                if (value === 'fixed') {
                  setTemplateCode((form.getFieldValue('templateCode') as string) || 'nginx_systemd');
                  form.setFieldValue('templateCode', form.getFieldValue('templateCode') || 'nginx_systemd');
                  form.setFieldValue(
                    'templateConfig',
                    form.getFieldValue('templateConfig') || buildDefaultTemplateConfig((form.getFieldValue('templateCode') as string) || 'nginx_systemd'),
                  );
                } else {
                  setTemplateCode('');
                  form.setFieldValue('templateCode', '');
                  form.setFieldValue('templateConfig', {});
                }
              }}
            >
              <Select.Option value="fixed">{t('business.deploy.package.executionMode.fixed')}</Select.Option>
              <Select.Option value="orchestrated">{t('business.deploy.package.executionMode.orchestrated')}</Select.Option>
            </Select>
          </Form.Item>
          {executionMode === 'fixed' ? (
            <>
              <Form.Item field="templateCode" label={t('business.deploy.package.templateCode')} rules={[{ required: true }]}>
                <Select
                  onChange={(value) => {
                    const nextCode = value as string;
                    setTemplateCode(nextCode);
                    form.setFieldValue('templateConfig', buildDefaultTemplateConfig(nextCode));
                  }}
                >
                  {deployFixedTemplateCatalog.map((item) => (
                    <Select.Option key={item.code} value={item.code}>
                      {t(`business.deploy.package.templateCode.${item.code}`)}
                    </Select.Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item field="templateConfig" hidden>
                <Input />
              </Form.Item>
              {currentTemplateEntry ? (
                <div style={{ marginBottom: 16, padding: 12, borderRadius: 6, background: 'var(--color-fill-1)' }}>
                  <Space direction="vertical" size={6}>
                    <span style={{ fontWeight: 500 }}>{t('business.deploy.package.templateSummary')}</span>
                    <span style={{ color: 'var(--color-text-3)', fontSize: 12 }}>
                      {t(currentTemplateEntry.summaryKey)}
                    </span>
                  </Space>
                </div>
              ) : null}
              <Form.Item field="sourceFileName" label={t('business.deploy.package.sourceFile')}>
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Input readOnly placeholder={t('business.deploy.package.sourceFilePlaceholder')} />
                  <Space wrap>
                    <Button
                      icon={<IconUpload />}
                      loading={uploadingSource}
                      onClick={() => {
                        const input = document.createElement('input');
                        input.type = 'file';
                        input.accept = '.tar,.tar.gz,.tgz,.zip,.gz';
                        input.onchange = () => {
                          void handleUploadSource(input.files?.[0]);
                        };
                        input.click();
                      }}
                    >
                      {t('business.deploy.package.uploadSource')}
                    </Button>
                    <Typography.Text type="secondary">
                      {t('business.deploy.package.sourceFileHint')}
                    </Typography.Text>
                  </Space>
                </Space>
              </Form.Item>
              <Form.Item field="sourceObjectKey" hidden>
                <Input />
              </Form.Item>
              <Form.Item field="sourceUrl" hidden>
                <Input />
              </Form.Item>
            </>
          ) : (
            <Form.Item field="installCommand" label={t('business.deploy.package.installCommand')}>
              <Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} />
            </Form.Item>
          )}
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

function buildDefaultTemplateConfig(templateCode: string) {
  return {
    scenario: templateCode,
  };
}
