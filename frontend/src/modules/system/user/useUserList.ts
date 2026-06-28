import { useCallback, useEffect, useMemo, useReducer } from 'react';
import type { FormInstance } from '@arco-design/web-react/es/Form';
import { message } from '../../../components/feedback/message';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../api/importExport';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePublicSettings } from '../../../core/settings/publicSettings';
import { getDeptTree, type DeptNode } from '../dept/api';
import { getPostList } from '../post/api';
import { getRoleList } from '../role/api';
import {
  batchDeleteUsers,
  batchUpdateUserStatus,
  createUser,
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

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

export interface UserListState {
  // List
  data: UserListRow[];
  total: number;
  loading: boolean;
  error: unknown;
  query: UserListQuery;
  selectedRowKeys: Array<string | number>;

  // Form modal
  formVisible: boolean;
  editingRow: UserListRow | null;
  submitting: boolean;
  formDeptId: number;
  avatarPreview: string;

  // Detail modal
  detailTarget: UserListRow | null;
  detailData: UserDetailData | null;
  detailLoading: boolean;
  detailError: boolean;

  // Reset password modal
  resetTarget: UserListRow | null;

  // Dropdown options
  roleOptions: Array<{ label: string; value: number }>;
  deptOptions: Array<{ label: string; value: number }>;
  postOptions: Array<{ label: string; value: number; deptId: number }>;
}

const emptyQuery: UserListQuery = {
  username: '',
  nickname: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};

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

export { emptyQuery, emptyForm, isDefaultUserListQuery };

const initialState: UserListState = {
  data: [],
  total: 0,
  loading: false,
  error: null,
  query: { ...emptyQuery },
  selectedRowKeys: [],
  formVisible: false,
  editingRow: null,
  submitting: false,
  formDeptId: 0,
  avatarPreview: '',
  detailTarget: null,
  detailData: null,
  detailLoading: false,
  detailError: false,
  resetTarget: null,
  roleOptions: [],
  deptOptions: [],
  postOptions: [],
};

// ---------------------------------------------------------------------------
// Actions
// ---------------------------------------------------------------------------

type UserListAction =
  | { type: 'SET_LIST'; data: UserListRow[]; total: number }
  | { type: 'SET_LOADING'; loading: boolean }
  | { type: 'SET_ERROR'; error: unknown }
  | { type: 'SET_QUERY'; query: UserListQuery }
  | { type: 'SET_SELECTED_ROWS'; keys: Array<string | number> }
  | { type: 'OPEN_CREATE' }
  | { type: 'OPEN_EDIT'; row: UserListRow; detail: UserDetailData; orgEnabled: boolean }
  | { type: 'CLOSE_FORM' }
  | { type: 'SET_SUBMITTING'; submitting: boolean }
  | { type: 'SET_FORM_DEPT'; deptId: number }
  | { type: 'SET_AVATAR_PREVIEW'; preview: string }
  | { type: 'OPEN_DETAIL'; row: UserListRow }
  | { type: 'SET_DETAIL_DATA'; data: UserDetailData | null }
  | { type: 'SET_DETAIL_LOADING'; loading: boolean }
  | { type: 'SET_DETAIL_ERROR'; error: boolean }
  | { type: 'CLOSE_DETAIL' }
  | { type: 'OPEN_RESET_PASSWORD'; row: UserListRow }
  | { type: 'CLOSE_RESET_PASSWORD' }
  | { type: 'SET_ROLE_OPTIONS'; options: Array<{ label: string; value: number }> }
  | { type: 'SET_DEPT_OPTIONS'; options: Array<{ label: string; value: number }> }
  | { type: 'SET_POST_OPTIONS'; options: Array<{ label: string; value: number; deptId: number }> };

function userListReducer(state: UserListState, action: UserListAction): UserListState {
  switch (action.type) {
    case 'SET_LIST':
      return { ...state, data: action.data, total: action.total };
    case 'SET_LOADING':
      return { ...state, loading: action.loading };
    case 'SET_ERROR':
      return { ...state, error: action.error };
    case 'SET_QUERY':
      return { ...state, query: action.query };
    case 'SET_SELECTED_ROWS':
      return { ...state, selectedRowKeys: action.keys };
    case 'OPEN_CREATE':
      return { ...state, formVisible: true, editingRow: null, formDeptId: 0, avatarPreview: '' };
    case 'OPEN_EDIT':
      return {
        ...state,
        formVisible: true,
        editingRow: action.row,
        formDeptId: action.orgEnabled ? action.detail.deptId || 0 : 0,
        avatarPreview: action.detail.avatar || '',
      };
    case 'CLOSE_FORM':
      return { ...state, formVisible: false, editingRow: null, avatarPreview: '' };
    case 'SET_SUBMITTING':
      return { ...state, submitting: action.submitting };
    case 'SET_FORM_DEPT':
      return { ...state, formDeptId: action.deptId };
    case 'SET_AVATAR_PREVIEW':
      return { ...state, avatarPreview: action.preview };
    case 'OPEN_DETAIL':
      return { ...state, detailTarget: action.row, detailData: null, detailError: false };
    case 'SET_DETAIL_DATA':
      return { ...state, detailData: action.data };
    case 'SET_DETAIL_LOADING':
      return { ...state, detailLoading: action.loading };
    case 'SET_DETAIL_ERROR':
      return { ...state, detailError: action.error, detailData: null };
    case 'CLOSE_DETAIL':
      return {
        ...state,
        detailTarget: null,
        detailData: null,
        detailError: false,
        detailLoading: false,
      };
    case 'OPEN_RESET_PASSWORD':
      return { ...state, resetTarget: action.row };
    case 'CLOSE_RESET_PASSWORD':
      return { ...state, resetTarget: null };
    case 'SET_ROLE_OPTIONS':
      return { ...state, roleOptions: action.options };
    case 'SET_DEPT_OPTIONS':
      return { ...state, deptOptions: action.options };
    case 'SET_POST_OPTIONS':
      return { ...state, postOptions: action.options };
    default:
      return state;
  }
}

// ---------------------------------------------------------------------------
// Hook
// ---------------------------------------------------------------------------

interface LoadDataOptions {
  silent?: boolean;
}

export function useUserList(form: FormInstance<UserCreatePayload>) {
  const [state, dispatch] = useReducer(userListReducer, initialState);
  const { t } = useTranslation();
  const publicSettings = usePublicSettings();
  const orgEnabled = publicSettings.orgEnabled;
  const orgRequiredForUser = orgEnabled && publicSettings.orgRequiredForUser;

  const invalidateUserCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/user', resourceKeys: ['list:default'] },
      { path: '/system/dept', resourceKeys: ['users:org-chart'] },
    ]);
  }, []);

  // ---- Data loading ----

  const loadData = useCallback(
    async (nextQuery: UserListQuery = state.query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        dispatch({ type: 'SET_LOADING', loading: true });
        dispatch({ type: 'SET_ERROR', error: null });
      }
      try {
        const result = isDefaultUserListQuery(nextQuery)
          ? await resolveRouteWarmData('/system/user', 'list:default', () => getUserList(nextQuery))
          : await getUserList(nextQuery);
        dispatch({ type: 'SET_LIST', data: result.items, total: result.total });
      } catch (requestError) {
        dispatch({ type: 'SET_ERROR', error: requestError });
      } finally {
        if (!silent) {
          dispatch({ type: 'SET_LOADING', loading: false });
        }
      }
    },
    [state.query],
  );

  const loadRoles = useCallback(async () => {
    try {
      const result = await resolveRouteWarmData('/system/user', 'roles:active', () =>
        getRoleList({ page: 1, pageSize: 9999, sortField: 'sort', sortOrder: 'asc', status: 1 }),
      );
      dispatch({
        type: 'SET_ROLE_OPTIONS',
        options: result.items.map((item) => ({ label: item.roleName, value: item.id })),
      });
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  const loadDeptAndPostOptions = useCallback(async () => {
    if (!orgEnabled) {
      dispatch({ type: 'SET_DEPT_OPTIONS', options: [] });
      dispatch({ type: 'SET_POST_OPTIONS', options: [] });
      return;
    }
    try {
      const [deptRows, postRows] = await Promise.all([
        resolveRouteWarmData('/system/user', 'depts:default', () =>
          getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
        ),
        resolveRouteWarmData('/system/user', 'posts:active', () =>
          getPostList({ page: 1, pageSize: 9999, sortField: 'sort', sortOrder: 'asc', status: 1 }),
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
      dispatch({
        type: 'SET_DEPT_OPTIONS',
        options: [{ label: t('system.dept.none'), value: 0 }, ...flattenDept(selectableDeptRows)],
      });
      dispatch({
        type: 'SET_POST_OPTIONS',
        options: postRows.items.map((item) => ({
          label: item.postName,
          value: item.id,
          deptId: item.deptId,
        })),
      });
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [orgEnabled, t]);

  // ---- Effects ----

  useEffect(() => {
    void loadData(state.query);
  }, [loadData, state.query]);

  useEffect(() => {
    void loadRoles();
  }, [loadRoles]);

  useEffect(() => {
    void loadDeptAndPostOptions();
  }, [loadDeptAndPostOptions]);

  useRefreshSubscription(
    ['system:user:changed', 'system:role:changed', 'system:dept:changed', 'system:post:changed'],
    (payload) => {
      if (payload.source === 'system/user') {
        return;
      }
      void loadData(state.query);
      void loadRoles();
      if (orgEnabled) {
        void loadDeptAndPostOptions();
      }
    },
  );

  // ---- Form modal actions ----

  const openCreate = useCallback(() => {
    dispatch({ type: 'OPEN_CREATE' });
    form.setFieldsValue(emptyForm);
  }, [form]);

  const openEdit = useCallback(
    async (row: UserListRow) => {
      try {
        const detail = await getUserDetail(row.id);
        dispatch({ type: 'OPEN_EDIT', row, detail, orgEnabled });
        form.setFieldsValue({
          username: detail.username,
          nickname: detail.nickname,
          email: detail.email,
          phone: detail.phone,
          avatar: detail.avatar || '',
          deptId: orgEnabled ? detail.deptId : 0,
          postId: orgEnabled ? detail.postId : 0,
          status: detail.status,
          roleIds: detail.roleIds || [],
        });
      } catch {
        message.error(t('common.loadFailed'));
      }
    },
    [form, orgEnabled, t],
  );

  const closeForm = useCallback(() => {
    dispatch({ type: 'CLOSE_FORM' });
  }, []);

  const submitForm = useCallback(async () => {
    let values: UserCreatePayload;
    try {
      values = (await form.validate()) as UserCreatePayload;
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    dispatch({ type: 'SET_SUBMITTING', submitting: true });
    try {
      if (state.editingRow) {
        await updateUser(state.editingRow.id, {
          nickname: values.nickname ?? '',
          email: values.email,
          phone: values.phone,
          avatar: values.avatar,
          deptId: orgEnabled ? (values.deptId ?? 0) : 0,
          postId: orgEnabled ? (values.postId ?? 0) : 0,
          status: values.status ?? 1,
          roleIds: values.roleIds || [],
        });
        message.success(t('common.updateSuccess'));
      } else {
        await createUser({
          username: values.username,
          password: values.password,
          nickname: values.nickname ?? '',
          avatar: values.avatar,
          email: values.email,
          phone: values.phone,
          status: values.status ?? 1,
          roleIds: values.roleIds || [],
          deptId: orgEnabled ? (values.deptId ?? 0) : 0,
          postId: orgEnabled ? (values.postId ?? 0) : 0,
          profileExt: values.profileExt,
        });
        message.success(t('common.createSuccess'));
      }
      invalidateUserCaches();
      publishRefresh('system:user:changed', 'system/user');
      dispatch({ type: 'CLOSE_FORM' });
      await loadData(state.query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      dispatch({ type: 'SET_SUBMITTING', submitting: false });
    }
  }, [form, invalidateUserCaches, loadData, orgEnabled, state.editingRow, state.query, t]);

  const handleAvatarUploadSuccess = useCallback(
    (response: unknown) => {
      const uploaded = response as { url: string };
      form.setFieldValue('avatar', uploaded.url);
      dispatch({ type: 'SET_AVATAR_PREVIEW', preview: uploaded.url });
      message.success(t('system.profile.avatarUploadSuccess'));
    },
    [form, t],
  );

  const handleDeptChange = useCallback(
    (value: unknown) => {
      const nextDeptId = Number(value || 0);
      dispatch({ type: 'SET_FORM_DEPT', deptId: nextDeptId });
      const currentPostId = Number(form.getFieldValue('postId') || 0);
      if (currentPostId > 0) {
        const currentPost = state.postOptions.find((item) => item.value === currentPostId);
        if (!currentPost || currentPost.deptId !== nextDeptId) {
          form.setFieldsValue({ postId: 0 });
        }
      }
    },
    [form, state.postOptions],
  );

  // ---- Detail modal actions ----

  const openDetail = useCallback((row: UserListRow) => {
    dispatch({ type: 'OPEN_DETAIL', row });
    dispatch({ type: 'SET_DETAIL_LOADING', loading: true });
    dispatch({ type: 'SET_DETAIL_ERROR', error: false });
    getUserDetail(row.id)
      .then((result) => {
        dispatch({ type: 'SET_DETAIL_DATA', data: result });
      })
      .catch(() => {
        dispatch({ type: 'SET_DETAIL_ERROR', error: true });
      })
      .finally(() => {
        dispatch({ type: 'SET_DETAIL_LOADING', loading: false });
      });
  }, []);

  const closeDetail = useCallback(() => {
    dispatch({ type: 'CLOSE_DETAIL' });
  }, []);

  const retryDetail = useCallback(() => {
    if (state.detailTarget) {
      openDetail(state.detailTarget);
    }
  }, [openDetail, state.detailTarget]);

  // ---- Reset password modal actions ----

  const openResetPassword = useCallback((row: UserListRow) => {
    dispatch({ type: 'OPEN_RESET_PASSWORD', row });
  }, []);

  const closeResetPassword = useCallback(() => {
    dispatch({ type: 'CLOSE_RESET_PASSWORD' });
  }, []);

  const submitResetPassword = useCallback(
    async (newPassword: string) => {
      if (!state.resetTarget) {
        return;
      }
      dispatch({ type: 'SET_SUBMITTING', submitting: true });
      try {
        const result = await resetUserPassword(state.resetTarget.id, { newPassword });
        message.success(
          t('system.user.resetPasswordSuccess', { count: result.revokedSessionCount }),
        );
        dispatch({ type: 'CLOSE_RESET_PASSWORD' });
      } catch {
        message.error(t('common.actionFailed'));
      } finally {
        dispatch({ type: 'SET_SUBMITTING', submitting: false });
      }
    },
    [state.resetTarget, t],
  );

  // ---- Row actions ----

  const handleRowStatus = useCallback(
    async (row: UserListRow, status: 1 | 2) => {
      await batchUpdateUserStatus({ userIds: [row.id], status });
      message.success(
        status === 1 ? t('system.user.enableSuccess') : t('system.user.disableSuccess'),
      );
      invalidateUserCaches();
      publishRefresh('system:user:changed', 'system/user');
      await loadData(state.query, { silent: true });
    },
    [invalidateUserCaches, loadData, state.query, t],
  );

  // ---- Search/query actions ----

  const search = useCallback(
    (values: Partial<UserListQuery>) => {
      dispatch({ type: 'SET_SELECTED_ROWS', keys: [] });
      dispatch({
        type: 'SET_QUERY',
        query: { ...state.query, ...values, page: 1 },
      });
    },
    [state.query],
  );

  const resetQuery = useCallback(() => {
    dispatch({ type: 'SET_SELECTED_ROWS', keys: [] });
    dispatch({ type: 'SET_QUERY', query: { ...emptyQuery } });
  }, []);

  const setQuery = useCallback((query: UserListQuery) => {
    dispatch({ type: 'SET_QUERY', query });
  }, []);

  const setSelectedRowKeys = useCallback((keys: Array<string | number>) => {
    dispatch({ type: 'SET_SELECTED_ROWS', keys });
  }, []);

  // ---- Batch actions ----

  const handleExport = useCallback(async () => {
    await exportUsers(state.query);
  }, [state.query]);

  const handleDownloadTemplate = useCallback(async () => {
    await downloadUserImportTemplate();
  }, []);

  const handleImport = useCallback(
    async (file: File) => {
      const result = await importUsers(file);
      showImportResult(result, t);
      if (result.applied) {
        invalidateUserCaches();
        publishRefresh('system:user:changed', 'system/user');
        await loadData(state.query, { silent: true });
        await loadRoles();
        if (orgEnabled) {
          await loadDeptAndPostOptions();
        }
      }
    },
    [invalidateUserCaches, loadData, loadDeptAndPostOptions, loadRoles, orgEnabled, state.query, t],
  );

  const handleBatchStatus = useCallback(
    async (status: 1 | 2) => {
      if (state.selectedRowKeys.length === 0) {
        message.warning(t('common.batchSelectionRequired'));
        return;
      }
      const userIds = state.selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
      const result = await batchUpdateUserStatus({ userIds, status });
      message.success(t('system.user.batchStatusSuccess', { count: result.updatedCount }));
      invalidateUserCaches();
      publishRefresh('system:user:changed', 'system/user');
      dispatch({ type: 'SET_SELECTED_ROWS', keys: [] });
      await loadData(state.query, { silent: true });
    },
    [invalidateUserCaches, loadData, state.query, state.selectedRowKeys, t],
  );

  const handleBatchDelete = useCallback(async () => {
    if (state.selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = state.selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteUsers({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateUserCaches();
    publishRefresh('system:user:changed', 'system/user');
    dispatch({ type: 'SET_SELECTED_ROWS', keys: [] });
    await loadData(state.query, { silent: true });
  }, [invalidateUserCaches, loadData, state.query, state.selectedRowKeys, t]);

  // ---- Avatar preview ----

  const setAvatarPreview = useCallback((preview: string) => {
    dispatch({ type: 'SET_AVATAR_PREVIEW', preview });
  }, []);

  // ---- Derived state ----

  const filteredPostOptions = useMemo(
    () => [
      { label: t('system.post.none'), value: 0 },
      ...state.postOptions
        .filter((item) => state.formDeptId > 0 && item.deptId === state.formDeptId)
        .map(({ label, value }) => ({ label, value })),
    ],
    [state.postOptions, state.formDeptId, t],
  );

  const enabledUserCount = useMemo(
    () => state.data.filter((item) => item.status === 1).length,
    [state.data],
  );
  const disabledUserCount = useMemo(
    () => state.data.filter((item) => item.status !== 1).length,
    [state.data],
  );
  const unassignedRoleUserCount = useMemo(
    () => state.data.filter((item) => !item.roleKeys?.length).length,
    [state.data],
  );
  const assignableRoleCount = useMemo(() => state.roleOptions.length, [state.roleOptions]);
  const availableDeptCount = useMemo(
    () => state.deptOptions.filter((item) => item.value > 0).length,
    [state.deptOptions],
  );
  const availablePostCount = useMemo(
    () => state.postOptions.filter((item) => item.value > 0).length,
    [state.postOptions],
  );

  return {
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
    search,
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
  };
}
