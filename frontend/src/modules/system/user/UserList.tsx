import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Form,
  Grid,
  Input,
  Popconfirm,
  Select,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconEye,
  IconLock,
  IconPlus,
  IconSearch,
  IconUpload,
} from '@arco-design/web-react/icon';
import { uploadSystemFile } from '../../../api/upload';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../api/importExport';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { formatDateTime } from '../../../core/format/dateTime';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePublicSettings } from '../../../core/settings/publicSettings';
import { usePermission } from '../../../hooks/usePermission';
import { getDeptTree, type DeptNode } from '../dept/api';
import { getPostList } from '../post/api';
import { getRoleList } from '../role/api';
import type { PaginationProps } from '@arco-design/web-react/es/Pagination/interface';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import {
  batchDeleteUsers,
  batchUpdateUserStatus,
  createUser,
  deleteUser,
  downloadUserImportTemplate,
  exportUsers,
  getUserDetail,
  getUserList,
  importUsers,
  resetUserPassword,
  updateUser,
  type UserCreatePayload,
  type UserDetail as UserDetailData,
  type UserListQuery,
  type UserListRow,
} from './api';
import {
  AppModal,
  AppTable,
  FilterPanel,
  FormSection,
  GovernanceInsightDrawer,
  GovernanceRailSummary,
  GovernanceRailToggleButton,
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageHeader,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SubmitBar,
  TableBatchActionBar,
  PermissionAction,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import UserDetailContent from './UserDetailContent';
import '../../../core/styles/list-page.css';
import './user.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface ResetPasswordFormValues {
  newPassword: string;
  confirmPassword: string;
}

const emptyForm: UserCreatePayload = {
  username: '',
  password: '',
  nickname: '',
  email: '',
  phone: '',
  avatar: '',
  status: 1,
  deptId: 0,
  postId: 0,
  roleIds: [],
};

const emptyQuery: UserListQuery = {
  username: '',
  nickname: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};

function isDefaultUserListQuery(query: UserListQuery) {
  return (
    !query.username &&
    !query.nickname &&
    query.status === undefined &&
    query.deptId === undefined &&
    query.postId === undefined &&
    (query.page ?? 1) === 1 &&
    (query.pageSize ?? 10) === 10 &&
    !query.sortField &&
    !query.sortOrder
  );
}

interface LoadDataOptions {
  silent?: boolean;
}

const UserList: React.FC = () => {
  const [data, setData] = useState<UserListRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<UserListRow | null>(null);
  const [detailTarget, setDetailTarget] = useState<UserListRow | null>(null);
  const [detailData, setDetailData] = useState<UserDetailData | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState(false);
  const [resetTarget, setResetTarget] = useState<UserListRow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [query, setQuery] = useState<UserListQuery>(emptyQuery);
  const [roleOptions, setRoleOptions] = useState<Array<{ label: string; value: number }>>([]);
  const [deptOptions, setDeptOptions] = useState<Array<{ label: string; value: number }>>([]);
  const [postOptions, setPostOptions] = useState<
    Array<{ label: string; value: number; deptId: number }>
  >([]);
  const [formDeptId, setFormDeptId] = useState(0);
  const [avatarPreview, setAvatarPreview] = useState('');
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [form] = Form.useForm<UserCreatePayload>();
  const [resetPasswordForm] = Form.useForm<ResetPasswordFormValues>();
  const [queryForm] = Form.useForm<UserListQuery>();
  const { t } = useTranslation();
  const publicSettings = usePublicSettings();
  const orgEnabled = publicSettings.orgEnabled;
  const orgRequiredForUser = orgEnabled && publicSettings.orgRequiredForUser;
  const { isAdmin, hasPerm } = usePermission();
  const canView = isAdmin || hasPerm('system:user:view');
  const canCreate = isAdmin || hasPerm('system:user:create');
  const canEdit = isAdmin || hasPerm('system:user:update');
  const canDelete = isAdmin || hasPerm('system:user:delete');
  const canResetPassword = isAdmin || hasPerm('system:user:reset');
  const canExport = isAdmin || hasPerm('system:user:export');
  const canImport = isAdmin || hasPerm('system:user:import');
  const canBatchUpdate = isAdmin || hasPerm('system:user:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:user:batch-delete');
  const governanceRail = useGovernanceRail();
  const invalidateUserCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/user', resourceKeys: ['list:default'] },
      { path: '/system/dept', resourceKeys: ['users:org-chart'] },
    ]);
  }, []);

  const loadData = useCallback(
    async (nextQuery: UserListQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      try {
        const result = isDefaultUserListQuery(nextQuery)
          ? await resolveRouteWarmData('/system/user', 'list:default', () => getUserList(nextQuery))
          : await getUserList(nextQuery);
        setData(result.items);
        setTotal(result.total);
      } catch (requestError) {
        setError(requestError);
      } finally {
        if (!silent) {
          setLoading(false);
        }
      }
    },
    [query],
  );

  const loadRoles = useCallback(async () => {
    try {
      const result = await resolveRouteWarmData('/system/user', 'roles:active', () =>
        getRoleList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc', status: 1 }),
      );
      setRoleOptions(
        result.items.map((item) => ({
          label: item.roleName,
          value: item.id,
        })),
      );
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  const loadDeptAndPostOptions = useCallback(async () => {
    if (!orgEnabled) {
      setDeptOptions([]);
      setPostOptions([]);
      return;
    }
    try {
      const [deptRows, postRows] = await Promise.all([
        resolveRouteWarmData('/system/user', 'depts:default', () =>
          getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
        ),
        resolveRouteWarmData('/system/user', 'posts:active', () =>
          getPostList({ page: 1, pageSize: 100, sortField: 'sort', sortOrder: 'asc', status: 1 }),
        ),
      ]);
      const flattenDept = (nodes: DeptNode[], depth = 0): Array<{ label: string; value: number }> =>
        nodes.flatMap((item) => [
          { label: `${'— '.repeat(depth)}${item.deptName}`, value: item.id },
          ...(item.children?.length ? flattenDept(item.children, depth + 1) : []),
        ]);
      const selectableDeptRows = deptRows.flatMap((item) =>
        item.isRoot ? item.children || [] : [item],
      );
      setDeptOptions([
        { label: t('system.dept.none'), value: 0 },
        ...flattenDept(selectableDeptRows),
      ]);
      setPostOptions(
        postRows.items.map((item) => ({
          label: item.postName,
          value: item.id,
          deptId: item.deptId,
        })),
      );
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [orgEnabled, t]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadRoles();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadRoles]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadDeptAndPostOptions();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadDeptAndPostOptions]);

  useRefreshSubscription(
    ['system:user:changed', 'system:role:changed', 'system:dept:changed', 'system:post:changed'],
    (payload) => {
      if (payload.source === 'system/user') {
        return;
      }
      void loadData(query);
      void loadRoles();
      if (orgEnabled) {
        void loadDeptAndPostOptions();
      }
    },
  );

  const openCreate = () => {
    setEditing(null);
    setFormDeptId(emptyForm.deptId || 0);
    setAvatarPreview('');
    form.setFieldsValue(emptyForm);
    setVisible(true);
  };

  const openEdit = async (row: UserListRow) => {
    try {
      const detail = await getUserDetail(row.id);
      setEditing(row);
      setFormDeptId(orgEnabled ? detail.deptId || 0 : 0);
      setAvatarPreview(detail.avatar || '');
      form.setFieldsValue({
        username: detail.username,
        nickname: detail.nickname,
        email: detail.email,
        phone: detail.phone,
        avatar: detail.avatar || '',
        deptId: orgEnabled ? detail.deptId : 0,
        postId: orgEnabled ? detail.postId : 0,
        status: detail.status,
        roleIds: detail.roleIds,
      });
      setVisible(true);
    } catch {
      message.error(t('common.loadFailed'));
    }
  };

  const loadDetail = useCallback(async (id: number) => {
    setDetailLoading(true);
    setDetailError(false);
    try {
      const result = await getUserDetail(id);
      setDetailData(result);
    } catch {
      setDetailError(true);
      setDetailData(null);
    } finally {
      setDetailLoading(false);
    }
  }, []);

  const openDetail = (row: UserListRow) => {
    setDetailTarget(row);
    setDetailData(null);
    setDetailError(false);
    void loadDetail(row.id);
  };

  const closeDetail = () => {
    setDetailTarget(null);
    setDetailData(null);
    setDetailError(false);
    setDetailLoading(false);
  };

  const submitForm = async () => {
    let values;
    try {
      values = await form.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setSubmitting(true);
    try {
      if (editing) {
        await updateUser(editing.id, {
          nickname: values.nickname,
          email: values.email,
          phone: values.phone,
          avatar: values.avatar,
          deptId: orgEnabled ? values.deptId : 0,
          postId: orgEnabled ? values.postId : 0,
          status: values.status,
          roleIds: values.roleIds,
        });
        message.success(t('common.updateSuccess'));
      } else {
        await createUser({
          ...values,
          deptId: orgEnabled ? values.deptId : 0,
          postId: orgEnabled ? values.postId : 0,
        });
        message.success(t('common.createSuccess'));
      }
      invalidateUserCaches();
      publishRefresh('system:user:changed', 'system/user');
      setVisible(false);
      setAvatarPreview('');
      await loadData(query, { silent: true });
    } finally {
      setSubmitting(false);
    }
  };

  const handleUploadAvatar = async (file?: File | null) => {
    if (!file) {
      return;
    }
    setUploadingAvatar(true);
    try {
      const uploaded = await uploadSystemFile(file, 'user/avatar');
      form.setFieldValue('avatar', uploaded.url);
      setAvatarPreview(uploaded.url);
      message.success(t('system.profile.avatarUploadSuccess'));
    } finally {
      setUploadingAvatar(false);
    }
  };

  const openResetPassword = (row: UserListRow) => {
    resetPasswordForm.setFieldsValue({ newPassword: '', confirmPassword: '' });
    setResetTarget(row);
  };

  const submitResetPassword = async () => {
    if (!resetTarget) {
      return;
    }
    let values;
    try {
      values = await resetPasswordForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setSubmitting(true);
    try {
      const result = await resetUserPassword(resetTarget.id, { newPassword: values.newPassword });
      message.success(t('system.user.resetPasswordSuccess', { count: result.revokedSessionCount }));
      setResetTarget(null);
      resetPasswordForm.resetFields();
    } finally {
      setSubmitting(false);
    }
  };

  const removeUser = async (id: number) => {
    await deleteUser(id);
    message.success(t('common.deleteSuccess'));
    invalidateUserCaches();
    publishRefresh('system:user:changed', 'system/user');
    setSelectedRowKeys((keys) => keys.filter((key) => Number(key) !== id));
    const nextPage =
      data.length === 1 && (query.page || 1) > 1 ? (query.page || 1) - 1 : query.page || 1;
    const nextQuery = { ...query, page: nextPage };
    setQuery(nextQuery);
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

  const toArcoSortOrder = (sortOrder?: UserListQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<UserListQuery['sortField']>,
  ): Partial<ColumnProps<UserListRow>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<UserListRow>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    const nextQuery: UserListQuery = {
      ...query,
      page: pagination.current || 1,
      pageSize: pagination.pageSize || query.pageSize || emptyQuery.pageSize,
      sortField: currentSorter?.direction ? String(currentSorter.field) : undefined,
      sortOrder:
        currentSorter?.direction === 'ascend'
          ? 'asc'
          : currentSorter?.direction === 'descend'
            ? 'desc'
            : undefined,
    };
    setSelectedRowKeys([]);
    setQuery(nextQuery);
  };

  const columns: ColumnProps<UserListRow>[] = [
    {
      title: t('system.user.username'),
      dataIndex: 'username',
      width: TABLE_COLUMN_WIDTH.identity,
      ...sortableColumn('username'),
    },
    {
      title: t('system.user.nickname'),
      dataIndex: 'nickname',
      width: TABLE_COLUMN_WIDTH.identity,
      ...sortableColumn('nickname'),
    },
    ...(orgEnabled
      ? [
          {
            title: t('system.user.dept'),
            dataIndex: 'deptName',
            width: TABLE_COLUMN_WIDTH.identity,
          },
          withTableColumnPriority(
            {
              title: t('system.user.post'),
              dataIndex: 'postName',
              width: TABLE_COLUMN_WIDTH.identity,
            },
            'medium',
          ),
        ]
      : []),
    withTableColumnPriority(
      {
        title: t('system.user.roles'),
        dataIndex: 'roleKeys',
        width: TABLE_COLUMN_WIDTH.name,
        render: (value: string[]) => (
          <span className="system-user-list__role-text">
            {value?.length ? value.join(' / ') : '-'}
          </span>
        ),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.user.email'),
        dataIndex: 'email',
        width: TABLE_COLUMN_WIDTH.name,
        ...sortableColumn('email'),
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.user.phone'),
        dataIndex: 'phone',
        width: TABLE_COLUMN_WIDTH.identity,
        ...sortableColumn('phone'),
      },
      'low',
    ),
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
    withTableColumnPriority(
      {
        title: t('system.user.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        ...sortableColumn('createdAt'),
        render: (value: string) => formatDateTime(value),
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.wide,
      fixed: 'right',
      render: (_: unknown, row: UserListRow) => (
        <Space size={4} className="system-list__actions">
          {canView ? (
            <Button type="text" size="small" icon={<IconEye />} onClick={() => openDetail(row)}>
              {t('common.detail')}
            </Button>
          ) : null}
          {canEdit ? (
            <Button
              type="text"
              size="small"
              icon={<IconEdit />}
              onClick={() => {
                void openEdit(row);
              }}
            >
              {t('common.edit')}
            </Button>
          ) : null}
          {canResetPassword ? (
            <Button
              type="text"
              size="small"
              icon={<IconLock />}
              onClick={() => openResetPassword(row)}
            >
              {t('system.user.resetPassword')}
            </Button>
          ) : null}
          {canDelete ? (
            <Popconfirm
              title={t('common.deleteConfirm')}
              onOk={() => removeUser(row.id)}
              disabled={row.id === 1}
            >
              <Button
                type="text"
                size="small"
                status="danger"
                icon={<IconDelete />}
                disabled={row.id === 1}
              >
                {t('common.delete')}
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ];

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError
          timeout={isTimeoutRequestError(error)}
          onRetry={() => {
            void loadData(query);
          }}
        />
      );
    }
    if (isServerRequestError(error)) {
      return (
        <PageServerError
          onRetry={() => {
            void loadData(query);
          }}
        />
      );
    }
    return (
      <PageError
        onRetry={() => {
          void loadData(query);
        }}
      />
    );
  };

  const handleExport = async () => {
    await exportUsers(query);
  };

  const handleDownloadTemplate = async () => {
    await downloadUserImportTemplate();
  };

  const handleImport = async (file: File) => {
    const result = await importUsers(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidateUserCaches();
      publishRefresh('system:user:changed', 'system/user');
      await loadData(query, { silent: true });
      await loadRoles();
      if (orgEnabled) {
        await loadDeptAndPostOptions();
      }
    }
  };

  const handleBatchStatus = async (status: 1 | 2) => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const userIds = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdateUserStatus({ userIds, status });
    message.success(t('system.user.batchStatusSuccess', { count: result.updatedCount }));
    invalidateUserCaches();
    publishRefresh('system:user:changed', 'system/user');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteUsers({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateUserCaches();
    publishRefresh('system:user:changed', 'system/user');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const batchActionDisabled = !canBatchUpdate || selectedRowKeys.length === 0;
  const batchDeleteDisabled = !canBatchDelete || selectedRowKeys.length === 0;
  const filteredPostOptions = [
    { label: t('system.post.none'), value: 0 },
    ...postOptions
      .filter((item) => formDeptId > 0 && item.deptId === formDeptId)
      .map(({ label, value }) => ({ label, value })),
  ];
  const handleDeptChange = (value: unknown) => {
    const nextDeptId = Number(value || 0);
    setFormDeptId(nextDeptId);
    const currentPostId = Number(form.getFieldValue('postId') || 0);
    if (currentPostId > 0) {
      const currentPost = postOptions.find((item) => item.value === currentPostId);
      if (!currentPost || currentPost.deptId !== nextDeptId) {
        form.setFieldsValue({ postId: 0 });
      }
    }
  };

  const enabledUserCount = useMemo(() => data.filter((item) => item.status === 1).length, [data]);
  const disabledUserCount = useMemo(() => data.filter((item) => item.status !== 1).length, [data]);
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('common.total', { count: total }),
        value: total,
        hint: t('system.user.hero.totalHint'),
      },
      {
        key: 'enabled',
        label: t('system.user.status.enabled'),
        value: enabledUserCount,
        hint: t('system.user.hero.enabledHint'),
      },
      {
        key: 'selected',
        label: t('system.user.hero.selectedRows'),
        value: selectedRowKeys.length,
        hint: t('system.user.hero.selectedHint'),
      },
      {
        key: 'roles',
        label: t('system.user.hero.rolesReady'),
        value: roleOptions.length,
        hint: t('system.user.hero.rolesHint'),
      },
    ],
    [enabledUserCount, roleOptions.length, selectedRowKeys.length, t, total],
  );
  const governanceSummaryItems = useMemo(
    () => [
      ...(orgEnabled
        ? [
            {
              label: t('system.user.hero.orgReady'),
              value: `${Math.max(deptOptions.length - 1, 0)} / ${postOptions.length}`,
              description: t('system.user.hero.orgHint'),
            },
          ]
        : []),
      {
        label: t('system.user.hero.disabledRows'),
        value: disabledUserCount,
        description: t('system.user.hero.disabledHint'),
      },
      {
        label: t('system.user.hero.batchActions'),
        value: batchActionDisabled ? t('common.no') : t('common.yes'),
        description: t('system.user.hero.batchHint'),
      },
    ],
    [batchActionDisabled, deptOptions.length, disabledUserCount, orgEnabled, postOptions.length, t],
  );

  return (
    <PageContainer>
      <PageHeader
        title={t('system.menu.user')}
        extra={
          <ListHeaderActions
            utility={
              <>
                <GovernanceRailToggleButton
                  expanded={governanceRail.expanded}
                  onToggle={governanceRail.toggle}
                >
                  {t('system.user.hero.summaryTitle')}
                </GovernanceRailToggleButton>
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
              <Button type="primary" icon={<IconPlus />} onClick={openCreate} disabled={!canCreate}>
                {t('common.add')}
              </Button>
            }
          />
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero system-user-list__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('system.user.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title">
                {t('system.user.hero.title')}
              </Typography.Title>
            </div>
          </div>
          <div className="system-page-kpi-grid">
            {heroStats.map((item) => (
              <div key={item.key} className="system-page-kpi">
                <span className="system-page-kpi__label">{item.label}</span>
                <span className="system-page-kpi__value">{item.value}</span>
                <span className="system-page-kpi__hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>
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
              selectedCount={selectedRowKeys.length}
              selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
              clearText={t('common.clearSelection')}
              clearSuccessText={t('common.clearSelectionSuccess')}
              onClear={() => setSelectedRowKeys([])}
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
            {loading && data.length === 0 ? <PageLoading /> : null}
            {error && data.length === 0 ? renderErrorState() : null}
            {!loading && !error && data.length === 0 ? (
              <PageEmpty description={t('common.noData')} />
            ) : null}
            {!loading && !(error && data.length === 0) && data.length > 0 ? (
              <AppTable<UserListRow>
                className="system-user-list__table"
                data={data}
                columns={columns}
                rowKey="id"
                loading={loading}
                scroll={{ x: 'max-content' }}
                rowSelection={{
                  type: 'checkbox',
                  selectedRowKeys,
                  fixed: true,
                  checkboxProps: (row) => ({ disabled: row.id === 1 }),
                  onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
                }}
                onChange={handleTableChange}
                emptyText={t('common.noData')}
                pagination={
                  {
                    current: query.page || emptyQuery.page,
                    pageSize: query.pageSize || emptyQuery.pageSize,
                    total,
                    showJumper: true,
                    pageSizeChangeResetCurrent: false,
                    sizeCanChange: true,
                    sizeOptions: [10, 20, 50, 100],
                    size: 'small',
                    showTotal: (count: number) => t('common.total', { count }),
                  } as PaginationProps
                }
              />
            ) : null}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.user.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.user.hero.summaryTitle')}
        noteDescription={t('system.user.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.user.edit') : t('system.user.create')}
        visible={visible}
        size="lg"
        onCancel={() => {
          setVisible(false);
          setAvatarPreview('');
        }}
        footer={
          <SubmitBar
            onCancel={() => {
              setVisible(false);
              setAvatarPreview('');
            }}
            onSubmit={() => {
              void submitForm();
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
            void submitForm();
          }}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('common.basicInfo')}>
              <FormItem
                label={t('system.user.username')}
                field="username"
                rules={[{ required: true, message: t('auth.usernameRequired') }]}
              >
                <Input disabled={Boolean(editing)} onPressEnter={() => form.submit()} />
              </FormItem>
              {!editing ? (
                <FormItem
                  label={t('system.user.password')}
                  field="password"
                  rules={[{ required: true, message: t('auth.passwordRequired') }]}
                >
                  <Input.Password onPressEnter={() => form.submit()} />
                </FormItem>
              ) : null}
              <FormItem label={t('system.user.nickname')} field="nickname">
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem
                label={t('system.user.email')}
                field="email"
                rules={[{ match: /\S+@\S+\.\S+/, message: t('system.user.email.invalid') }]}
              >
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem label={t('system.user.phone')} field="phone">
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem label={t('system.user.avatar')} field="avatar">
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  <Input
                    placeholder={t('system.profile.avatarPlaceholder')}
                    onChange={(value) => setAvatarPreview(value)}
                  />
                  <Space align="center" wrap>
                    <Avatar size={40}>
                      {avatarPreview ? (
                        <img src={avatarPreview} alt={t('system.user.avatar')} />
                      ) : (
                        t('common.user').slice(0, 1)
                      )}
                    </Avatar>
                    <Button
                      icon={<IconUpload />}
                      loading={uploadingAvatar}
                      onClick={() => {
                        const input = document.createElement('input');
                        input.type = 'file';
                        input.accept = 'image/png,image/jpeg,image/jpg,image/webp,image/gif';
                        input.onchange = () => {
                          void handleUploadAvatar(input.files?.[0]);
                        };
                        input.click();
                      }}
                    >
                      {t('system.profile.uploadAvatar')}
                    </Button>
                    <Typography.Text type="secondary">
                      {t('system.profile.avatarUploadHint')}
                    </Typography.Text>
                  </Space>
                </Space>
              </FormItem>
            </FormSection>
            <FormSection title={t('common.accessControl')}>
              {orgEnabled ? (
                <>
                  <FormItem
                    label={t('system.user.dept')}
                    field="deptId"
                    rules={
                      orgRequiredForUser
                        ? [
                            {
                              validator: (value, callback) => {
                                if (Number(value || 0) > 0) {
                                  callback();
                                  return;
                                }
                                callback(t('system.user.dept.required'));
                              },
                            },
                          ]
                        : undefined
                    }
                  >
                    <Select options={deptOptions} onChange={handleDeptChange} />
                  </FormItem>
                  <FormItem label={t('system.user.post')} field="postId">
                    <Select options={filteredPostOptions} disabled={formDeptId === 0} />
                  </FormItem>
                </>
              ) : null}
              <FormItem label={t('system.user.status')} field="status">
                <Select
                  options={[
                    { label: t('system.user.status.enabled'), value: 1 },
                    { label: t('system.user.status.disabled'), value: 2 },
                  ]}
                />
              </FormItem>
              <FormItem
                label={t('system.user.roles')}
                field="roleIds"
                rules={[
                  {
                    required: true,
                    type: 'array',
                    minLength: 1,
                    message: t('system.user.role.required'),
                  },
                ]}
              >
                <Select mode="multiple" options={roleOptions} />
              </FormItem>
            </FormSection>
          </Space>
        </Form>
      </AppModal>

      <AppModal
        title={
          detailTarget
            ? `${detailTarget.username} / ${detailTarget.nickname || '-'}`
            : t('system.user.detail')
        }
        visible={Boolean(detailTarget)}
        size="xl"
        onCancel={closeDetail}
        footer={null}
      >
        {detailLoading ? <PageLoading /> : null}
        {!detailLoading && detailError ? (
          <PageError
            onRetry={() => {
              if (detailTarget) {
                void loadDetail(detailTarget.id);
              }
            }}
          />
        ) : null}
        {!detailLoading && !detailError && !detailData ? (
          <PageEmpty description={t('common.noData')} />
        ) : null}
        {!detailLoading && !detailError && detailData ? (
          <UserDetailContent detail={detailData} orgEnabled={orgEnabled} />
        ) : null}
      </AppModal>

      <AppModal
        title={t('system.user.resetPasswordTitle')}
        visible={Boolean(resetTarget)}
        size="sm"
        onCancel={() => {
          setResetTarget(null);
          resetPasswordForm.resetFields();
        }}
        footer={
          <SubmitBar
            onCancel={() => {
              setResetTarget(null);
              resetPasswordForm.resetFields();
            }}
            onSubmit={() => {
              void submitResetPassword();
            }}
            loading={submitting}
            submitText={t('system.user.resetPassword')}
          />
        }
        unmountOnExit
      >
        <Form
          form={resetPasswordForm}
          layout="vertical"
          onSubmit={() => {
            void submitResetPassword();
          }}
        >
          <Space direction="vertical" size={16} className="dialog-form-stack">
            <FormItem label={t('system.user.resetPasswordTarget')}>
              <Input
                value={
                  resetTarget ? `${resetTarget.username} / ${resetTarget.nickname || '-'}` : ''
                }
                disabled
              />
            </FormItem>
            <FormItem
              label={t('system.user.newPassword')}
              field="newPassword"
              rules={[{ required: true, message: t('auth.passwordRequired') }]}
            >
              <Input.Password onPressEnter={() => resetPasswordForm.submit()} />
            </FormItem>
            <FormItem
              label={t('system.user.confirmPassword')}
              field="confirmPassword"
              rules={[
                { required: true, message: t('system.profile.confirmPasswordRequired') },
                {
                  validator: (value, callback) => {
                    const nextPassword = resetPasswordForm.getFieldValue('newPassword');
                    if (!value || value === nextPassword) {
                      callback();
                      return;
                    }
                    callback(t('system.profile.confirmPasswordMismatch'));
                  },
                },
              ]}
            >
              <Input.Password onPressEnter={() => resetPasswordForm.submit()} />
            </FormItem>
            <Tag color="orange">{t('system.user.resetPasswordHint')}</Tag>
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
};

export default UserList;
