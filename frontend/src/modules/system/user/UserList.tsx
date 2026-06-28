import React, { useMemo } from 'react';
import {
  Button,
  Card,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  Form,
} from '@arco-design/web-react';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconEye,
  IconLock,
  IconPlus,
  IconSearch,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '../../../core/format/dateTime';
import { usePermission } from '../../../hooks/usePermission';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import type { UserCreatePayload, UserListQuery, UserListRow } from './api';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
  FilterPanel,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  GovernanceSummaryBar,
  PageLoading,
  PageRequestError,
  PermissionAction,
  SystemRowActions,
  TableBatchActionBar,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import UserDetailContent from './UserDetailContent';
import UserFormModal from './UserFormModal';
import ResetPasswordModal from './ResetPasswordModal';
import { useUserList, emptyQuery } from './useUserList';
import '../components/shared/list-page.css';
import './user.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface ResetPasswordFormValues {
  newPassword: string;
  confirmPassword: string;
}

function getTableText(value: string | number | null | undefined) {
  const text = typeof value === 'number' ? String(value) : value?.trim();
  return text || '-';
}

const UserList: React.FC = () => {
  const [form] = Form.useForm<UserCreatePayload>();
  const [resetPasswordForm] = Form.useForm<ResetPasswordFormValues>();
  const [queryForm] = Form.useForm();

  const {
    state,
    orgEnabled,
    orgRequiredForUser,
    // Form modal
    openCreate,
    openEdit,
    closeForm,
    submitForm,
    handleAvatarUploadSuccess,
    handleDeptChange,
    // Detail modal
    openDetail,
    closeDetail,
    retryDetail,
    // Reset password modal
    openResetPassword,
    closeResetPassword,
    submitResetPassword,
    // Row actions
    handleRowStatus,
    // Query actions
    search: doSearch,
    resetQuery,
    setQuery,
    setSelectedRowKeys,
    // Data loading
    loadData,
    // Avatar preview
    setAvatarPreview,
    // Batch actions
    handleExport,
    handleDownloadTemplate,
    handleImport,
    handleBatchStatus,
    handleBatchDelete,
    // Derived
    filteredPostOptions,
    enabledUserCount,
    disabledUserCount,
    unassignedRoleUserCount,
    assignableRoleCount,
    availableDeptCount,
    availablePostCount,
  } = useUserList(form);

  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();
  const canView = isAdmin || hasPerm('system:user:view');
  const canCreate = isAdmin || hasPerm('system:user:create');
  const canEdit = isAdmin || hasPerm('system:user:update');
  const canResetPassword = isAdmin || hasPerm('system:user:reset');
  const canExport = isAdmin || hasPerm('system:user:export');
  const canImport = isAdmin || hasPerm('system:user:import');
  const canBatchUpdate = isAdmin || hasPerm('system:user:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:user:batch-delete');

  const batchActionDisabled = !canBatchUpdate || state.selectedRowKeys.length === 0;
  const batchDeleteDisabled = !canBatchDelete || state.selectedRowKeys.length === 0;
  const batchActionReady = !batchActionDisabled || !batchDeleteDisabled;

  // ---- Column helpers ----

  const toArcoSortOrder = (sortOrder?: UserListQuery['sortOrder']) => {
    if (sortOrder === 'asc') return 'ascend';
    if (sortOrder === 'desc') return 'descend';
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<UserListQuery['sortField']>,
  ): Partial<ColumnProps<UserListRow>> => ({
    sorter: true,
    sortOrder: state.query.sortField === field ? toArcoSortOrder(state.query.sortOrder) : undefined,
  });

  const renderCellText = (value: string | number | null | undefined, className?: string) => {
    const text = getTableText(value);
    const cell = (
      <span className={['system-user-list__cell-text', className].filter(Boolean).join(' ')}>
        {text}
      </span>
    );
    return text === '-' ? cell : <Tooltip content={text}>{cell}</Tooltip>;
  };

  const roleLabelById = useMemo(
    () => new Map(state.roleOptions.map((item) => [item.value, item.label])),
    [state.roleOptions],
  );

  // ---- Columns ----

  const columns: ColumnProps<UserListRow>[] = [
    {
      title: t('system.user.username'),
      dataIndex: 'username',
      width: TABLE_COLUMN_WIDTH.name,
      ...sortableColumn('username'),
      render: (value: string) => renderCellText(value, 'system-user-list__cell-text--strong'),
    },
    {
      title: t('system.user.nickname'),
      dataIndex: 'nickname',
      width: TABLE_COLUMN_WIDTH.identity,
      ...sortableColumn('nickname'),
      render: (value: string) => renderCellText(value),
    },
    {
      title: t('system.user.status'),
      dataIndex: 'status',
      width: TABLE_COLUMN_WIDTH.status,
      ...sortableColumn('status'),
      render: (value: number) => (
        <Tag color={value === 1 ? 'green' : 'red'}>
          {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
        </Tag>
      ),
    },
    ...(orgEnabled
      ? [
          {
            title: t('system.user.dept'),
            dataIndex: 'deptName',
            width: TABLE_COLUMN_WIDTH.name,
            render: (value: string) => renderCellText(value),
          },
          {
            title: t('system.user.post'),
            dataIndex: 'postName',
            width: TABLE_COLUMN_WIDTH.name,
            render: (value: string) => renderCellText(value),
          },
        ]
      : []),
    {
      title: t('system.user.roles'),
      dataIndex: 'roleNames',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (_: string[], row: UserListRow) => {
        const roleNames = row.roleNames?.filter(Boolean);
        const roleLabels = row.roleIds
          ?.map((roleId) => roleLabelById.get(roleId))
          .filter((value): value is string => Boolean(value));
        const roleText = roleNames?.length
          ? roleNames.join(' / ')
          : roleLabels?.length
            ? roleLabels.join(' / ')
            : row.roleKeys?.length
              ? row.roleKeys.join(' / ')
              : undefined;
        return renderCellText(roleText, 'system-user-list__role-text');
      },
    },
    {
      title: t('system.user.email'),
      dataIndex: 'email',
      width: TABLE_COLUMN_WIDTH.name,
      ...sortableColumn('email'),
      render: (value: string) => renderCellText(value),
    },
    withTableColumnPriority(
      {
        title: t('system.user.phone'),
        dataIndex: 'phone',
        width: TABLE_COLUMN_WIDTH.identity,
        ...sortableColumn('phone'),
        render: (value: string) => renderCellText(value),
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.user.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        ...sortableColumn('createdAt'),
        render: (value: string) => (
          <Typography.Text className="system-list__datetime-text">
            {formatDateTime(value)}
          </Typography.Text>
        ),
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.wide,
      fixed: 'right',
      render: (_: unknown, row: UserListRow) => (
        <SystemRowActions
          className="system-user-list__row-actions"
          actions={[
            {
              key: 'detail',
              text: t('common.detail'),
              icon: <IconEye />,
              onClick: () => openDetail(row),
              hidden: !canView,
            },
            {
              key: 'edit',
              text: t('common.edit'),
              icon: <IconEdit />,
              onClick: () => {
                void openEdit(row);
              },
              hidden: !canEdit,
            },
            {
              key: 'reset-password',
              text: t('system.user.resetPassword'),
              icon: <IconLock />,
              onClick: () => openResetPassword(row),
              hidden: !canResetPassword,
            },
            {
              key: 'status',
              text: row.status === 1 ? t('system.user.disable') : t('system.user.enable'),
              disabled: row.id === 1,
              hidden: !canBatchUpdate,
              status: row.status === 1 ? 'warning' : undefined,
              confirm: {
                title:
                  row.status === 1
                    ? t('system.user.disableConfirm')
                    : t('system.user.enableConfirm'),
                onOk: () => handleRowStatus(row, row.status === 1 ? 2 : 1),
              },
            },
          ]}
        />
      ),
    },
  ];

  // ---- Table change ----

  const handleTableChange: TableProps<UserListRow>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    const nextQuery = {
      ...state.query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || state.query.pageSize || emptyQuery.pageSize,
      sortField: currentSorter?.direction ? String(currentSorter.field) : undefined,
      sortOrder:
        currentSorter?.direction === 'ascend'
          ? 'asc'
          : currentSorter?.direction === 'descend'
            ? 'desc'
            : undefined,
    } as import('./api').UserListQuery;
    const sortChanged =
      nextQuery.sortField !== state.query.sortField ||
      nextQuery.sortOrder !== state.query.sortOrder;
    if (sortChanged) {
      setSelectedRowKeys([]);
    }
    setQuery(nextQuery);
  };

  // ---- Search ----

  const search = () => {
    const values = queryForm.getFieldsValue();
    doSearch(values);
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    resetQuery();
  };

  // ---- Governance metrics ----

  const statusMetrics = useMemo(
    () => [
      { key: 'total', label: t('system.menu.user'), value: state.total },
      { key: 'enabled', label: t('system.user.status.enabled'), value: enabledUserCount },
      { key: 'disabled', label: t('system.user.hero.disabledRows'), value: disabledUserCount },
      {
        key: 'unassigned',
        label: t('system.user.hero.unassignedRoles'),
        value: unassignedRoleUserCount,
      },
      { key: 'roles', label: t('system.user.hero.rolesReady'), value: assignableRoleCount },
    ],
    [
      assignableRoleCount,
      disabledUserCount,
      enabledUserCount,
      t,
      state.total,
      unassignedRoleUserCount,
    ],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.user.hero.orgReady'),
        value: orgEnabled ? `${availableDeptCount} / ${availablePostCount}` : t('common.disabled'),
        description: t('system.user.hero.orgHint'),
      },
      {
        label: t('system.user.hero.rolesReady'),
        value: assignableRoleCount,
        description: t('system.user.hero.rolesHint'),
      },
      {
        label: t('system.user.hero.batchActions'),
        value: batchActionReady ? t('common.yes') : t('common.no'),
        description: t('system.user.hero.batchHint'),
      },
    ],
    [availableDeptCount, availablePostCount, assignableRoleCount, batchActionReady, orgEnabled, t],
  );

  // ---- Render ----

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('system.user.hero.eyebrow')}
          title={t('system.user.hero.title')}
          description={t('system.user.hero.desc')}
          metrics={statusMetrics.slice(0, 3).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('system.user.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16}>
                <Col span={6}>
                  <FormItem label={t('system.user.username')} field="username">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.user.nickname')} field="nickname">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.user.status')} field="status">
                    <Select
                      allowClear
                      options={[
                        { label: t('system.user.status.enabled'), value: 1 },
                        { label: t('system.user.status.disabled'), value: 2 },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem className="filter-panel__action-item">
                    <Space>
                      <Button
                        type="primary"
                        htmlType="submit"
                        icon={<IconSearch />}
                        onClick={search}
                      >
                        {t('common.search')}
                      </Button>
                      <Button onClick={reset}>{t('common.reset')}</Button>
                    </Space>
                  </FormItem>
                </Col>
              </Row>
            </Form>
          </FilterPanel>
          <Card className="page-panel system-list__table-card system-user-list__table-card">
            <TableBatchActionBar
              selectedCount={state.selectedRowKeys.length}
              selectedText={t('common.selectedCount', { count: state.selectedRowKeys.length })}
              clearText={t('common.clearSelection')}
              clearSuccessText={t('common.clearSelectionSuccess')}
              onClear={() => setSelectedRowKeys([])}
              prefixActions={
                <ListHeaderActions
                  utility={
                    <>
                      <Button
                        icon={<IconDownload />}
                        onClick={() => {
                          void handleExport();
                        }}
                        disabled={!canExport}
                      >
                        {t('common.export')}
                      </Button>
                      <Button
                        onClick={() => {
                          void handleDownloadTemplate();
                        }}
                        disabled={!canImport}
                      >
                        {t('common.downloadTemplate')}
                      </Button>
                      <ImportCsvButton
                        disabled={!canImport}
                        onSelect={(file) => {
                          void handleImport(file);
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
              hint={
                !canBatchUpdate || !canBatchDelete
                  ? t('common.batchActionPermissionHint')
                  : undefined
              }
              actions={
                <>
                  <PermissionAction
                    allowed={canBatchUpdate}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.user.batchEnableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(1);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button disabled={batchActionDisabled}>{t('system.user.batchEnable')}</Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchUpdate}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.user.batchDisableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(2);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button
                        status={batchActionDisabled ? undefined : 'warning'}
                        disabled={batchActionDisabled}
                      >
                        {t('system.user.batchDisable')}
                      </Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchDelete}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.user.batchDeleteConfirm')}
                      onOk={() => {
                        void handleBatchDelete();
                      }}
                      disabled={batchDeleteDisabled}
                    >
                      <Button
                        status={batchDeleteDisabled ? undefined : 'danger'}
                        icon={<IconDelete />}
                        disabled={batchDeleteDisabled}
                      >
                        {t('common.deleteSelected')}
                      </Button>
                    </Popconfirm>
                  </PermissionAction>
                </>
              }
            />
            {state.loading && state.data.length === 0 ? <PageLoading /> : null}
            {state.error && state.data.length === 0 ? (
              <PageRequestError
                error={state.error}
                onRetry={() => {
                  void loadData(state.query);
                }}
              />
            ) : null}
            {!state.loading && !state.error && state.data.length === 0 ? (
              <PageEmpty description={t('common.noData')} />
            ) : null}
            {!state.loading &&
            !(state.error && state.data.length === 0) &&
            state.data.length > 0 ? (
              <AppTable<UserListRow>
                className="system-list__table system-user-list__table"
                data={state.data}
                columns={columns}
                rowKey="id"
                loading={state.loading}
                scroll={{ x: 'max-content' }}
                rowSelection={{
                  type: 'checkbox',
                  selectedRowKeys: state.selectedRowKeys,
                  checkCrossPage: true,
                  preserveSelectedRowKeys: true,
                  fixed: true,
                  checkboxProps: (row) => ({ disabled: row.id === 1 }),
                  onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
                }}
                onChange={handleTableChange}
                emptyText={t('common.noData')}
                pagination={buildStandardPagination(t, {
                  current: state.query.page || emptyQuery.page,
                  pageSize: state.query.pageSize || emptyQuery.pageSize,
                  total: state.total,
                })}
              />
            ) : null}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.user.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.user.hero.sideLead')}
        noteDescription={t('system.user.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <UserFormModal
        visible={state.formVisible}
        editing={state.editingRow}
        submitting={state.submitting}
        orgEnabled={orgEnabled}
        orgRequiredForUser={orgRequiredForUser}
        formDeptId={state.formDeptId}
        avatarPreview={state.avatarPreview}
        roleOptions={state.roleOptions}
        deptOptions={state.deptOptions}
        filteredPostOptions={filteredPostOptions}
        form={form}
        onSubmit={() => {
          void submitForm();
        }}
        onCancel={closeForm}
        onDeptChange={handleDeptChange}
        onAvatarPreviewChange={setAvatarPreview}
        onAvatarUploadSuccess={handleAvatarUploadSuccess}
      />

      <AppModal
        title={
          state.detailTarget
            ? `${state.detailTarget.username} / ${state.detailTarget.nickname || '-'}`
            : t('system.user.detail')
        }
        visible={Boolean(state.detailTarget)}
        size="xl"
        onCancel={closeDetail}
        footer={null}
      >
        {state.detailLoading ? <PageLoading /> : null}
        {!state.detailLoading && state.detailError ? <PageError onRetry={retryDetail} /> : null}
        {!state.detailLoading && !state.detailError && !state.detailData ? (
          <PageEmpty description={t('common.noData')} />
        ) : null}
        {!state.detailLoading && !state.detailError && state.detailData ? (
          <UserDetailContent detail={state.detailData} orgEnabled={orgEnabled} />
        ) : null}
      </AppModal>

      <ResetPasswordModal
        visible={Boolean(state.resetTarget)}
        target={state.resetTarget}
        submitting={state.submitting}
        form={resetPasswordForm}
        onSubmit={submitResetPassword}
        onCancel={closeResetPassword}
      />
    </PageContainer>
  );
};

export default UserList;
