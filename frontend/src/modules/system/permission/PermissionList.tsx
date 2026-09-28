import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconRefresh,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../api/importExport';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePermission } from '../../../hooks/usePermission';
import { buildCrossPageRowSelection } from '../../../components/table/crossPageSelection';
import { getRoleList } from '../role/api';
import { translateRoleName } from '../role/display';
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
  SearchToolbar,
  FormSection,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  ListDataActions,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SystemRowActions,
  FormModalFooter,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  TableBatchActionBar,
  useGovernanceRail,
} from '../../../components';
import '../components/shared/list-page.css';

const FormItem = Form.Item;

const methodOptions = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

const emptyQuery: PermissionPolicyQuery = {
  keyword: '',
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
    !query.keyword &&
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

function canAccess(isAdmin: boolean, hasPerm: (permission: string) => boolean, permission: string) {
  return isAdmin || hasPerm(permission);
}

const PermissionList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = canAccess(isAdmin, hasPerm, 'system:permission:create');
  const canEdit = canAccess(isAdmin, hasPerm, 'system:permission:update');
  const canDelete = canAccess(isAdmin, hasPerm, 'system:permission:delete');
  const canBatchDelete = canAccess(isAdmin, hasPerm, 'system:permission:batch-delete');
  const canExport = canAccess(isAdmin, hasPerm, 'system:permission:export');
  const canImport = canAccess(isAdmin, hasPerm, 'system:permission:import');
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
          label: translateRoleName(item.roleName, t),
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
    const ids = selectedRowKeys.map(Number).filter((item) => item > 0);
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

  const search = (values: Partial<PermissionPolicyQuery>) => {
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      ...values,
      page: 1,
    });
  };

  const reset = () => {
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
  };

  const toArcoSortOrder = (sortOrder?: PermissionPolicyQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<PermissionPolicyQuery['sortField']>,
  ): Partial<ColumnProps<PermissionPolicyRow>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<PermissionPolicyRow>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    let nextSortOrder: 'asc' | 'desc' | undefined;
    if (currentSorter?.direction === 'ascend') {
      nextSortOrder = 'asc';
    } else if (currentSorter?.direction === 'descend') {
      nextSortOrder = 'desc';
    }
    setQuery({
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
      sortField: currentSorter?.direction ? String(currentSorter.field) : undefined,
      sortOrder: nextSortOrder,
    });
  };

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
  const governanceSummaryItems = useMemo(() => {
    let currentModeLabel = t('system.permission.policy.tab');
    if (activeTab === 'workbench') {
      currentModeLabel = t('system.permission.workbench.tab');
    } else if (activeTab === 'data-scope') {
      currentModeLabel = t('system.permission.dataScope.tab');
    }
    return [
      {
        label: t('system.permission.hero.currentMode'),
        value: currentModeLabel,
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
    ];
  }, [activeTab, canExport, t, workbench?.overview.unknownPermissionAssignmentCount]);

  const columns: ColumnProps<PermissionPolicyRow>[] = [
    {
      title: t('system.permission.roleKey'),
      dataIndex: 'roleKey',
      width: TABLE_COLUMN_WIDTH.code,
      ...sortableColumn('roleKey'),
    },
    {
      title: t('system.permission.method'),
      dataIndex: 'method',
      width: TABLE_COLUMN_WIDTH.method,
      ...sortableColumn('method'),
      render: (value: string) => <Tag color="arcoblue">{value}</Tag>,
    },
    {
      title: t('system.permission.path'),
      dataIndex: 'path',
      width: TABLE_COLUMN_WIDTH.routePath,
      ...sortableColumn('path'),
    },
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      fixed: 'right',
      render: (_: unknown, row: PermissionPolicyRow) => (
        <SystemRowActions
          actions={[
            {
              key: 'edit',
              text: t('common.edit'),
              icon: <IconEdit />,
              onClick: () => openEdit(row),
              hidden: !canEdit,
            },
            {
              key: 'delete',
              text: t('common.delete'),
              icon: <IconDelete />,
              hidden: !canDelete,
              disabled: row.roleKey === 'admin',
              status: 'danger',
              confirm: {
                title: t('common.deleteConfirm'),
                onOk: () => removePolicy(row),
                disabled: row.roleKey === 'admin',
              },
            },
          ]}
        />
      ),
    },
  ];

  const renderWorkbenchTab = () =>
    activeTab === 'workbench' ? (
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
    ) : null;

  const renderDataScopeTab = () =>
    activeTab === 'data-scope' ? (
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <PermissionDataScopeTab roleOptions={roleOptions} />
      </Space>
    ) : null;

  const apiTableRowSelection = buildCrossPageRowSelection({
    rows: data,
    getRowKey: (row) => row.id,
    selectedRowKeys,
    setSelectedRowKeys,
    fixed: true,
    checkboxProps: (row) => ({ disabled: row.roleKey === 'admin' }),
  });
  const apiTablePagination = buildStandardPagination(t, {
    current: query.page || emptyQuery.page,
    pageSize: query.pageSize || emptyQuery.pageSize,
    total,
  });

  const renderApiTableStates = () => (
    <>
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
          rowKey={(row) => row.id}
          loading={!!loading}
          scroll={{ x: 'max-content' }}
          rowSelection={apiTableRowSelection}
          onChange={handleTableChange}
          emptyText={t('common.noData')}
          pagination={apiTablePagination}
        />
      ) : null}
    </>
  );

  const renderApiTab = () =>
    activeTab !== 'workbench' && activeTab !== 'data-scope' ? (
      <Space direction="vertical" size={12} style={{ width: '100%' }}>
        <div className="permission-page__api-filter-shell">
          <SearchToolbar
            keyword={query.keyword ?? ''}
            keywordPlaceholder={t('system.permission.search.placeholder')}
            onKeywordChange={(keyword) => search({ keyword })}
            inlineFilters={
              <>
                <Select
                  allowClear
                  showSearch
                  placeholder={t('system.permission.roleKey')}
                  value={query.roleKey || undefined}
                  onChange={(value) => search({ roleKey: value ?? '' })}
                  options={roleOptions}
                />
                <Select
                  allowClear
                  placeholder={t('system.permission.method')}
                  value={query.method || undefined}
                  onChange={(value) => search({ method: value ?? '' })}
                  options={methodOptions.map((item) => ({ label: item, value: item }))}
                />
              </>
            }
            hasActiveFilters={Boolean(query.keyword || query.roleKey || query.method)}
            onClearAll={reset}
          />
        </div>
        <Card className="page-panel system-list__table-card permission-page__api-table-card">
          <TableBatchActionBar
            className="permission-page__api-action-bar"
            selectedCount={selectedRowKeys.length}
            selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
            clearText={t('common.clearSelection')}
            clearSuccessText={t('common.clearSelectionSuccess')}
            onClear={() => setSelectedRowKeys([])}
            showSelectionSummary={selectedRowKeys.length > 0}
            prefixActions={
              <ListHeaderActions
                className="permission-page__api-list-actions"
                utility={
                  <ListDataActions
                    canExport={canExport}
                    canImport={canImport}
                    onRefresh={() => void loadData(query)}
                    onExport={() => void handleExport()}
                    onDownloadTemplate={() => void handleDownloadTemplate()}
                    onImport={handleImport}
                  />
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
              <PermissionAction allowed={canBatchDelete} tooltip={t('common.noPermissionAction')}>
                <Popconfirm
                  title={t('system.permission.policy.batchDeleteConfirm')}
                  onOk={() => {
                    handleBatchDelete();
                  }}
                  disabled={batchDeleteDisabled}
                >
                  <Button status="danger" icon={<IconDelete />} disabled={batchDeleteDisabled}>
                    {t('common.deleteSelected')}
                  </Button>
                </Popconfirm>
              </PermissionAction>
            }
          />
          {renderApiTableStates()}
        </Card>
      </Space>
    ) : null;

  const renderPageContent = () => (
    <Space direction="vertical" size={16} className="system-page-template">
      <GovernanceSummaryBar
        className="permission-page__governance-bar"
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
      <div className="permission-page__content">
        <Card className="page-panel permission-workbench__tabs permission-page__tabs-panel">
          <Tabs activeTab={activeTab} onChange={(value) => setActiveTab(value as PermissionTabKey)}>
            <Tabs.TabPane key="workbench" title={t('system.permission.workbench.tab')} />
            <Tabs.TabPane key="data-scope" title={t('system.permission.dataScope.tab')} />
            <Tabs.TabPane key="api" title={t('system.permission.policy.tab')} />
          </Tabs>
        </Card>

        {renderWorkbenchTab()}
        {renderDataScopeTab()}
        {renderApiTab()}
      </div>
    </Space>
  );

  const renderGovernanceDrawer = () => (
    <GovernanceInsightDrawer
      title={t('system.permission.hero.summaryTitle')}
      visible={governanceRail.expanded}
      onClose={governanceRail.close}
      noteTitle={t('system.permission.hero.summaryTitle')}
      noteDescription={t('system.permission.hero.sideDesc')}
    >
      <GovernanceRailSummary items={governanceSummaryItems} />
    </GovernanceInsightDrawer>
  );

  const renderFormModal = () => (
    <AppModal
      title={editing ? t('system.permission.edit') : t('system.permission.create')}
      visible={visible}
      size="md"
      onCancel={() => setVisible(false)}
      footer={
        <FormModalFooter
          onCancel={() => setVisible(false)}
          onSubmit={() => void submitForm()}
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
  );

  return (
    <PageContainer>
      {renderPageContent()}
      {renderGovernanceDrawer()}
      {renderFormModal()}
    </PageContainer>
  );
};

export default PermissionList;
