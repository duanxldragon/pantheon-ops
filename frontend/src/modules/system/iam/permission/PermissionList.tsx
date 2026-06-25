import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
} from '@arco-design/web-react';
import { message } from '../../../../components/feedback/message';
import type { ColumnProps, TableProps } from '@arco-design/web-react/es/Table/interface';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconRefresh,
  IconSearch,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../../api/importExport';
import { isArcoFormValidationError } from '../../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../../core/router/prefetch';
import { usePermission } from '../../../../hooks/usePermission';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../../components/table/crossPageSelection';
import { getRoleList } from '../role/api';
import {
  batchDeletePermissionPolicies,
  createPermissionPolicy,
  deletePermissionPolicy,
  downloadPermissionImportTemplate,
  exportPermissionWorkbench,
  exportPermissionPolicies,
  getPermissionWorkbench,
  getPermissionPolicyList,
  importPermissionPolicies,
  remediatePermissionWorkbenchRole,
  updatePermissionPolicy,
  type PermissionPolicyPayload,
  type PermissionPolicyQuery,
  type PermissionPolicyRow,
  type PermissionWorkbenchQuery,
  type PermissionWorkbenchRole,
  type PermissionWorkbenchResp,
} from './api';
import { PermissionDataScopeTab } from './PermissionDataScopeTab';
import { PermissionWorkbenchTab } from './PermissionWorkbenchTab';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  FormSection,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  TableBatchActionBar,
  useGovernanceRail,
} from '../../../../components';
import '../../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

const emptyQuery: PermissionPolicyQuery = {
  roleKey: '',
  path: '',
  method: '',
  page: 1,
  pageSize: 10,
};

const emptyWorkbenchQuery: PermissionWorkbenchQuery = {
  roleKey: '',
  status: undefined,
  integrity: undefined,
  coverage: undefined,
};

function isDefaultPermissionPolicyQuery(query: PermissionPolicyQuery) {
  return (
    !query.roleKey &&
    !query.path &&
    !query.method &&
    (query.page ?? 1) === 1 &&
    (query.pageSize ?? 10) === 10
  );
}

function isDefaultPermissionWorkbenchQuery(query: PermissionWorkbenchQuery) {
  return (
    !query.roleKey &&
    query.status === undefined &&
    query.integrity === undefined &&
    query.coverage === undefined
  );
}

interface LoadDataOptions {
  silent?: boolean;
}

const emptyForm: PermissionPolicyPayload = {
  roleKey: '',
  path: '',
  method: 'GET',
};

type PermissionTabKey = 'workbench' | 'data-scope' | 'api';

const PermissionList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:permission:create');
  const canEdit = isAdmin || hasPerm('system:permission:update');
  const canDelete = isAdmin || hasPerm('system:permission:delete');
  const canBatchDelete = isAdmin || hasPerm('system:permission:batch-delete');
  const canExport = isAdmin || hasPerm('system:permission:export');
  const canImport = isAdmin || hasPerm('system:permission:import');
  const [activeTab, setActiveTab] = useState<PermissionTabKey>('workbench');
  const [data, setData] = useState<PermissionPolicyRow[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [loading, setLoading] = useState(false);
  const [policyError, setPolicyError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<PermissionPolicyRow | null>(null);
  const [query, setQuery] = useState<PermissionPolicyQuery>(emptyQuery);
  const [roleOptions, setRoleOptions] = useState<Array<{ label: string; value: string }>>([]);
  const [workbenchLoading, setWorkbenchLoading] = useState(false);
  const [workbenchError, setWorkbenchError] = useState<unknown>(null);
  const [workbench, setWorkbench] = useState<PermissionWorkbenchResp | null>(null);
  const [workbenchQuery, setWorkbenchQuery] =
    useState<PermissionWorkbenchQuery>(emptyWorkbenchQuery);
  const [detailRole, setDetailRole] = useState<PermissionWorkbenchRole | null>(null);
  const [remediatingRoleKey, setRemediatingRoleKey] = useState<string>('');
  const [form] = Form.useForm<PermissionPolicyPayload>();
  const [queryForm] = Form.useForm<PermissionPolicyQuery>();
  const governanceRail = useGovernanceRail();
  const invalidatePermissionCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/permission', resourceKeys: ['list:default', 'workbench:default'] },
    ]);
  }, []);

  const loadData = useCallback(
    async (nextQuery: PermissionPolicyQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setPolicyError(null);
      }
      try {
        const result = isDefaultPermissionPolicyQuery(nextQuery)
          ? await resolveRouteWarmData('/system/permission', 'list:default', () =>
              getPermissionPolicyList(nextQuery),
            )
          : await getPermissionPolicyList(nextQuery);
        setData(result.items);
        setTotal(result.total);
      } catch (requestError) {
        setPolicyError(requestError);
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [query],
  );

  const loadWorkbench = useCallback(
    async (nextQuery: PermissionWorkbenchQuery = workbenchQuery, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setWorkbenchLoading(true);
        setWorkbenchError(null);
      }
      try {
        const result = isDefaultPermissionWorkbenchQuery(nextQuery)
          ? await resolveRouteWarmData('/system/permission', 'workbench:default', () =>
              getPermissionWorkbench(nextQuery),
            )
          : await getPermissionWorkbench(nextQuery);
        setWorkbench(result);
        setDetailRole((current) =>
          current ? result.roles.find((item) => item.roleKey === current.roleKey) || null : current,
        );
      } catch (requestError) {
        setWorkbenchError(requestError);
      } finally {
        if (!silent) {
          setWorkbenchLoading(false);
        }
      }
    },
    [workbenchQuery],
  );

  const loadRoles = useCallback(async () => {
    try {
      const result = await resolveRouteWarmData('/system/permission', 'roles:default', () =>
        getRoleList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc' }),
      );
      setRoleOptions(
        result.items.map((item) => ({
          label: item.roleName,
          value: item.roleKey,
        })),
      );
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => loadData(query), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => loadWorkbench(workbenchQuery), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadWorkbench, workbenchQuery]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => loadRoles(), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadRoles]);

  useRefreshSubscription(
    ['system:permission:changed', 'system:role:changed', 'system:menu:changed'],
    (payload) => {
      if (payload.source === 'system/permission') {
        return;
      }
      loadWorkbench(workbenchQuery);
      if (payload.topic !== 'system:menu:changed') {
        loadData(query);
      }
      if (payload.topic === 'system:role:changed') {
        loadRoles();
      }
    },
  );

  const handleExport = async () => {
    await exportPermissionPolicies(query);
  };

  const handleDownloadTemplate = async () => {
    await downloadPermissionImportTemplate();
  };

  const handleImport = async (file: File) => {
    const result = await importPermissionPolicies(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidatePermissionCaches();
      publishRefresh('system:permission:changed', 'system/permission');
      await Promise.all([
        loadData(query, { silent: true }),
        loadWorkbench(workbenchQuery, { silent: true }),
      ]);
    }
  };

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(emptyForm);
    setVisible(true);
  };

  const openEdit = (row: PermissionPolicyRow) => {
    setEditing(row);
    form.setFieldsValue({
      roleKey: row.roleKey,
      path: row.path,
      method: row.method,
    });
    setVisible(true);
  };

  const submitForm = async () => {
    let values;
    try {
      values = await form.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    setSubmitting(true);
    try {
      if (editing) {
        await updatePermissionPolicy(editing.id, values);
        message.success(t('common.updateSuccess'));
      } else {
        await createPermissionPolicy(values);
        message.success(t('common.createSuccess'));
      }
      invalidatePermissionCaches();
      publishRefresh('system:permission:changed', 'system/permission');
      setVisible(false);
      await Promise.all([
        loadData(query, { silent: true }),
        loadWorkbench(workbenchQuery, { silent: true }),
      ]);
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const removePolicy = async (row: PermissionPolicyRow) => {
    await deletePermissionPolicy(row.id);
    message.success(t('common.deleteSuccess'));
    invalidatePermissionCaches();
    publishRefresh('system:permission:changed', 'system/permission');
    const nextPage =
      data.length === 1 && (query.page || 1) > 1 ? (query.page || 1) - 1 : query.page || 1;
    const nextQuery = { ...query, page: nextPage };
    setQuery(nextQuery);
    await loadWorkbench(workbenchQuery, { silent: true });
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeletePermissionPolicies({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidatePermissionCaches();
    publishRefresh('system:permission:changed', 'system/permission');
    setSelectedRowKeys([]);
    await Promise.all([
      loadData(query, { silent: true }),
      loadWorkbench(workbenchQuery, { silent: true }),
    ]);
  };

  const search = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const handleTableChange: TableProps<PermissionPolicyRow>['onChange'] = (pagination) => {
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
    });
  };

  const visibleSelectedRowKeys = useMemo(() => {
    return getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.id));
  }, [data, selectedRowKeys]);

  const batchDeleteDisabled = !canBatchDelete || selectedRowKeys.length === 0;

  const remediateRolePolicies = async (role: PermissionWorkbenchRole) => {
    setRemediatingRoleKey(role.roleKey);
    try {
      const result = await remediatePermissionWorkbenchRole({ roleKey: role.roleKey });
      if (result.createdCount > 0) {
        message.success(
          t('system.permission.workbench.remediateSuccess', { count: result.createdCount }),
        );
      } else {
        message.info(t('system.permission.workbench.remediateNoop'));
      }
      invalidatePermissionCaches();
      publishRefresh('system:permission:changed', 'system/permission');
      await Promise.all([
        loadWorkbench(workbenchQuery, { silent: true }),
        loadData(query, { silent: true }),
      ]);
    } finally {
      setRemediatingRoleKey('');
    }
  };

  const heroStats = useMemo(
    () => [
      {
        key: 'roles',
        label: t('system.permission.workbench.roleCount'),
        value: workbench?.overview.roleCount ?? 0,
        hint: t('system.permission.hero.rolesHint'),
      },
      {
        key: 'assignments',
        label: t('system.permission.workbench.permissionAssignments'),
        value: workbench
          ? workbench.overview.pagePermissionAssignmentCount +
            workbench.overview.actionPermissionAssignmentCount
          : 0,
        hint: t('system.permission.hero.assignmentsHint'),
      },
      {
        key: 'api',
        label: t('system.permission.workbench.apiAssignments'),
        value: workbench?.overview.apiActionCount ?? total,
        hint: t('system.permission.hero.apiHint'),
      },
      {
        key: 'gaps',
        label: t('system.permission.hero.gaps'),
        value: workbench
          ? workbench.overview.pageGapRoleCount + workbench.overview.apiGapRoleCount
          : 0,
        hint: t('system.permission.hero.gapsHint'),
      },
    ],
    [t, total, workbench],
  );
  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.permission.hero.currentMode'),
        value:
          activeTab === 'workbench'
            ? t('system.permission.workbench.tab')
            : activeTab === 'data-scope'
              ? t('system.permission.dataScope.tab')
              : t('system.permission.policy.tab'),
        description: t('system.permission.hero.modeHint'),
      },
      {
        label: t('system.permission.hero.unknownAssignments'),
        value: workbench?.overview.unknownPermissionAssignmentCount ?? 0,
        description: t('system.permission.hero.unknownHint'),
      },
      {
        label: t('system.permission.hero.exportReady'),
        value: canExport ? t('common.yes') : t('common.no'),
        description: t('system.permission.hero.exportHint'),
      },
    ],
    [activeTab, canExport, t, workbench?.overview.unknownPermissionAssignmentCount],
  );

  const columns: ColumnProps<PermissionPolicyRow>[] = [
    {
      title: t('system.permission.roleKey'),
      dataIndex: 'roleKey',
      width: TABLE_COLUMN_WIDTH.code,
    },
    {
      title: t('system.permission.method'),
      dataIndex: 'method',
      width: TABLE_COLUMN_WIDTH.method,
      render: (value: string) => <Tag color="arcoblue">{value}</Tag>,
    },
    {
      title: t('system.permission.path'),
      dataIndex: 'path',
      width: TABLE_COLUMN_WIDTH.routePath,
    },
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      fixed: 'right',
      render: (_: unknown, row: PermissionPolicyRow) => (
        <Space size={4} className="system-list__actions">
          {canEdit ? (
            <Button type="text" size="small" icon={<IconEdit />} onClick={() => openEdit(row)}>
              {t('common.edit')}
            </Button>
          ) : null}
          {canDelete ? (
            <Popconfirm
              title={t('common.deleteConfirm')}
              onOk={() => removePolicy(row)}
              disabled={row.roleKey === 'admin'}
            >
              <Button
                type="text"
                size="small"
                status="danger"
                icon={<IconDelete />}
                disabled={row.roleKey === 'admin'}
              >
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ];

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('system.permission.hero.eyebrow')}
          title={t('system.permission.hero.title')}
          description={t('system.permission.hero.desc')}
          metrics={heroStats.slice(0, 3).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('system.permission.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <Card className="page-panel permission-workbench__tabs">
            <Tabs
              activeTab={activeTab}
              onChange={(value) => setActiveTab(value as PermissionTabKey)}
            >
              <Tabs.TabPane key="workbench" title={t('system.permission.workbench.tab')} />
              <Tabs.TabPane key="data-scope" title={t('system.permission.dataScope.tab')} />
              <Tabs.TabPane key="api" title={t('system.permission.policy.tab')} />
            </Tabs>
          </Card>

          {activeTab === 'workbench' ? (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <PermissionWorkbenchTab
                roleOptions={roleOptions}
                utilityActions={
                  <>
                    <Button
                      icon={<IconRefresh />}
                      onClick={() => {
                        loadWorkbench(workbenchQuery);
                      }}
                    >
                      {t('common.refresh')}
                    </Button>
                    <Button
                      icon={<IconDownload />}
                      onClick={() => {
                        exportPermissionWorkbench(workbenchQuery);
                      }}
                      disabled={!canExport}
                    >
                      {t('system.permission.workbench.export')}
                    </Button>
                  </>
                }
                workbench={workbench}
                workbenchLoading={workbenchLoading}
                workbenchError={workbenchError}
                workbenchQuery={workbenchQuery}
                onWorkbenchQueryChange={setWorkbenchQuery}
                onRetryLoadWorkbench={() => {
                  loadWorkbench(workbenchQuery);
                }}
                detailRole={detailRole}
                onDetailRoleChange={setDetailRole}
                remediateRolePolicies={remediateRolePolicies}
                remediatingRoleKey={remediatingRoleKey}
              />
            </Space>
          ) : activeTab === 'data-scope' ? (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <div className="system-list__work-actions">
                <ListHeaderActions
                  utility={
                    <Button
                      icon={<IconRefresh />}
                      onClick={() => {
                        publishRefresh('system:permission:changed', 'system/permission');
                      }}
                    >
                      {t('common.refresh')}
                    </Button>
                  }
                />
              </div>
              <PermissionDataScopeTab roleOptions={roleOptions} />
            </Space>
          ) : (
            <Space direction="vertical" size={16} style={{ width: '100%' }}>
              <FilterPanel>
                <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
                  <Row gutter={16}>
                    <Col span={8}>
                      <FormItem label={t('system.permission.roleKey')} field="roleKey">
                        <Select allowClear options={roleOptions} />
                      </FormItem>
                    </Col>
                    <Col span={8}>
                      <FormItem label={t('system.permission.path')} field="path">
                        <Input onPressEnter={() => queryForm.submit()} />
                      </FormItem>
                    </Col>
                    <Col span={4}>
                      <FormItem label={t('system.permission.method')} field="method">
                        <Select
                          allowClear
                          options={methodOptions.map((item) => ({ label: item, value: item }))}
                        />
                      </FormItem>
                    </Col>
                    <Col span={4}>
                      <FormItem className="filter-panel__action-item">
                        <Space>
                          <Button type="primary" htmlType="submit" icon={<IconSearch />}>
                            {t('common.search')}
                          </Button>
                          <Button onClick={reset}>{t('common.reset')}</Button>
                        </Space>
                      </FormItem>
                    </Col>
                  </Row>
                </Form>
              </FilterPanel>
              <Card className="page-panel system-list__table-card">
                <TableBatchActionBar
                  selectedCount={selectedRowKeys.length}
                  selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
                  clearText={t('common.clearSelection')}
                  clearSuccessText={t('common.clearSelectionSuccess')}
                  onClear={() => setSelectedRowKeys([])}
                  prefixActions={
                    <ListHeaderActions
                      utility={
                        <>
                          <Button
                            icon={<IconRefresh />}
                            onClick={() => {
                              loadData(query);
                            }}
                          >
                            {t('common.refresh')}
                          </Button>
                          <Button
                            icon={<IconDownload />}
                            onClick={() => {
                              handleExport();
                            }}
                            disabled={!canExport}
                          >
                            {t('common.export')}
                          </Button>
                          <Button
                            onClick={() => {
                              handleDownloadTemplate();
                            }}
                            disabled={!canImport}
                          >
                            {t('common.downloadTemplate')}
                          </Button>
                          <ImportCsvButton
                            disabled={!canImport}
                            onSelect={(file) => {
                              handleImport(file);
                            }}
                          >
                            {t('common.import')}
                          </ImportCsvButton>
                        </>
                      }
                      primary={
                        <Button
                          type="primary"
                          icon={<IconPlus />}
                          onClick={openCreate}
                          disabled={!canCreate}
                        >
                          {t('common.add')}
                        </Button>
                      }
                    />
                  }
                  hint={!canBatchDelete ? t('common.batchActionPermissionHint') : undefined}
                  actions={
                    <PermissionAction
                      allowed={canBatchDelete}
                      tooltip={t('common.noPermissionAction')}
                    >
                      <Popconfirm
                        title={t('system.permission.policy.batchDeleteConfirm')}
                        onOk={() => {
                          handleBatchDelete();
                        }}
                        disabled={batchDeleteDisabled}
                      >
                        <Button
                          status="danger"
                          icon={<IconDelete />}
                          disabled={batchDeleteDisabled}
                        >
                          {t('common.deleteSelected')}
                        </Button>
                      </Popconfirm>
                    </PermissionAction>
                  }
                />
                {loading && data.length === 0 ? <PageLoading /> : null}
                {policyError && data.length === 0 ? (
                  <PageRequestError
                    error={policyError}
                    onRetry={() => {
                      loadData(query);
                    }}
                  />
                ) : null}
                {!loading && !policyError && data.length === 0 ? (
                  <PageEmpty description={t('common.noData')} />
                ) : null}
                {!loading && !(policyError && data.length === 0) && data.length > 0 ? (
                  <AppTable<PermissionPolicyRow>
                    className="system-list__table"
                    data={data}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    scroll={{ x: 'max-content' }}
                    rowSelection={{
                      type: 'checkbox',
                      selectedRowKeys: visibleSelectedRowKeys,
                      checkCrossPage: true,
                      preserveSelectedRowKeys: true,
                      fixed: true,
                      checkboxProps: (row) => ({ disabled: row.roleKey === 'admin' }),
                      onChange: (rowKeys) =>
                        setSelectedRowKeys((keys) =>
                          mergeCrossPageSelection(keys, rowKeys, data.map((item) => item.id)),
                        ),
                    }}
                    onChange={handleTableChange}
                    emptyText={t('common.noData')}
                    pagination={buildStandardPagination(t, {
                      current: query.page || emptyQuery.page,
                      pageSize: query.pageSize || emptyQuery.pageSize,
                      total,
                    })}
                  />
                ) : null}
              </Card>
            </Space>
          )}
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.permission.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.permission.hero.summaryTitle')}
        noteDescription={t('system.permission.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.permission.edit') : t('system.permission.create')}
        visible={visible}
        size="md"
        onCancel={() => setVisible(false)}
        footer={
          <SubmitBar
            onCancel={() => setVisible(false)}
            onSubmit={() => {
              submitForm();
            }}
            loading={submitting}
            submitText={editing ? t('common.save') : t('common.add')}
          />
        }
        unmountOnExit
      >
        <Form
          form={form}
          layout="vertical"
          onSubmit={() => {
            submitForm();
          }}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('common.basicInfo')}>
              <FormItem
                label={t('system.permission.roleKey')}
                field="roleKey"
                rules={[{ required: true, message: t('system.permission.roleRequired') }]}
              >
                <Select options={roleOptions} />
              </FormItem>
              <FormItem
                label={t('system.permission.path')}
                field="path"
                rules={[{ required: true, message: t('system.permission.pathRequired') }]}
              >
                <Input placeholder="/api/v1/system/user/list" onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem
                label={t('system.permission.method')}
                field="method"
                rules={[{ required: true, message: t('system.permission.methodRequired') }]}
              >
                <Select options={methodOptions.map((item) => ({ label: item, value: item }))} />
              </FormItem>
            </FormSection>
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
};

export default PermissionList;
