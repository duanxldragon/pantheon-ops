import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Tabs,
  Tag,
  Typography,
  TreeSelect,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconEye,
  IconPlus,
  IconSearch,
} from '@arco-design/web-react/icon';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import type { TreeSelectDataType } from '@arco-design/web-react/es/TreeSelect/interface';
import { useTranslation } from 'react-i18next';
import { showImportResult } from '../../../api/importExport';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePermission } from '../../../hooks/usePermission';
import {
  batchDeleteDepts,
  batchUpdateDeptLeader,
  batchUpdateDeptStatus,
  createDept,
  deleteDept,
  downloadDeptImportTemplate,
  exportDepts,
  exportDeptGovernanceTasks,
  getDeptGovernanceTasks,
  getDeptLeaderCandidates,
  getDeptOverview,
  getDeptTree,
  importDepts,
  updateDept,
  type DeptGovernanceTask,
  type DeptGovernanceTaskQuery,
  type DeptLeaderCandidate,
  type DeptListQuery,
  type DeptNode,
  type DeptOverviewResp,
  type DeptPayload,
} from './api';
import { createPost, getPostList, type PostPayload, type PostRow } from '../post/api';
import DeptOrgTab from './DeptOrgTab';
import {
  getUserDetail,
  getUserList,
  type UserDetail as UserDetailData,
  type UserListRow,
} from '../user/api';
import UserDetailContent from '../user/UserDetailContent';
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
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SubmitBar,
  SystemRowActions,
  TableBatchActionBar,
  PermissionAction,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import '../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

interface DeptFormValues {
  parentId: string;
  deptName: string;
  sort: number;
  leaderUserId?: number;
  leader?: string;
  phone?: string;
  email?: string;
  status: number;
}

interface DeptLeaderFormValues {
  [deptId: string]: number | undefined;
}

type OrgPostFormValues = PostPayload;

interface BatchLeaderTask {
  deptId: number;
  deptName: string;
  candidates: DeptLeaderCandidate[];
}

const emptyQuery: DeptListQuery = {
  deptName: '',
  status: undefined,
  governance: undefined,
  sortField: 'sort',
  sortOrder: 'asc',
};

const emptyForm: DeptFormValues = {
  parentId: '0',
  deptName: '',
  sort: 0,
  leaderUserId: undefined,
  leader: '',
  phone: '',
  email: '',
  status: 1,
};

const orgChartPageSize = 1000;

function isDefaultDeptListQuery(query: DeptListQuery) {
  return (
    !query.deptName &&
    query.status === undefined &&
    query.governance === undefined &&
    (query.sortField || 'sort') === 'sort' &&
    (query.sortOrder || 'asc') === 'asc'
  );
}

function findRootDept(nodes: DeptNode[]): DeptNode | undefined {
  return nodes.find((node) => node.isRoot || node.parentId === 0);
}

function flattenDeptNodes(nodes: DeptNode[]): DeptNode[] {
  return nodes.flatMap((node) => [node, ...flattenDeptNodes(node.children || [])]);
}

function groupByDept<T extends { deptId: number }>(items: T[]) {
  return items.reduce<Map<number, T[]>>((result, item) => {
    const current = result.get(item.deptId) || [];
    current.push(item);
    result.set(item.deptId, current);
    return result;
  }, new Map<number, T[]>());
}

function groupUsersByPost(users: UserListRow[]) {
  return users.reduce<Map<number, UserListRow[]>>((result, user) => {
    if (!user.postId) {
      return result;
    }
    const current = result.get(user.postId) || [];
    current.push(user);
    result.set(user.postId, current);
    return result;
  }, new Map<number, UserListRow[]>());
}

function collectDeptIDs(node: DeptNode): number[] {
  return [node.id, ...(node.children || []).flatMap((child) => collectDeptIDs(child))];
}

function findDeptNode(nodes: DeptNode[], deptID: number): DeptNode | undefined {
  for (const node of nodes) {
    if (node.id === deptID) {
      return node;
    }
    const found = findDeptNode(node.children || [], deptID);
    if (found) {
      return found;
    }
  }
  return undefined;
}

interface LoadDataOptions {
  silent?: boolean;
}

const DeptList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:dept:create');
  const canEdit = isAdmin || hasPerm('system:dept:update');
  const canDelete = isAdmin || hasPerm('system:dept:delete');
  const canExport = isAdmin || hasPerm('system:dept:export');
  const canImport = isAdmin || hasPerm('system:dept:import');
  const canBatchUpdate = isAdmin || hasPerm('system:dept:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:dept:batch-delete');
  const canViewPosts = isAdmin || hasPerm('system:post:list');
  const canViewUsers = isAdmin || hasPerm('system:user:list');
  const canCreatePost = isAdmin || hasPerm('system:post:create');
  const canViewUserDetail = isAdmin || hasPerm('system:user:view');
  const [activeTab, setActiveTab] = useState('manage');
  const [data, setData] = useState<DeptNode[]>([]);
  const [allDeptTree, setAllDeptTree] = useState<DeptNode[]>([]);
  const [orgDepts, setOrgDepts] = useState<DeptNode[]>([]);
  const [orgPosts, setOrgPosts] = useState<PostRow[]>([]);
  const [orgUsers, setOrgUsers] = useState<UserListRow[]>([]);
  const [orgLoading, setOrgLoading] = useState(false);
  const [orgError, setOrgError] = useState<unknown>(null);
  const [selectedOrgDeptId, setSelectedOrgDeptId] = useState(0);
  const [postVisible, setPostVisible] = useState(false);
  const [creatingPostDept, setCreatingPostDept] = useState<DeptNode | null>(null);
  const [postSubmitting, setPostSubmitting] = useState(false);
  const [userDetailVisible, setUserDetailVisible] = useState(false);
  const [userDetailLoading, setUserDetailLoading] = useState(false);
  const [userDetailError, setUserDetailError] = useState(false);
  const [userDetail, setUserDetail] = useState<UserDetailData | null>(null);
  const [userDetailId, setUserDetailId] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<DeptNode | null>(null);
  const [leaderVisible, setLeaderVisible] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [overview, setOverview] = useState<DeptOverviewResp | null>(null);
  const [governanceTasks, setGovernanceTasks] = useState<DeptGovernanceTask[]>([]);
  const [governanceLoading, setGovernanceLoading] = useState(false);
  const [leaderCandidates, setLeaderCandidates] = useState<DeptLeaderCandidate[]>([]);
  const [leaderCandidateLoading, setLeaderCandidateLoading] = useState(false);
  const [batchLeaderTasks, setBatchLeaderTasks] = useState<BatchLeaderTask[]>([]);
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 10 });
  const [query, setQuery] = useState<DeptListQuery>(emptyQuery);
  const [form] = Form.useForm<DeptFormValues>();
  const [leaderForm] = Form.useForm<DeptLeaderFormValues>();
  const [postForm] = Form.useForm<OrgPostFormValues>();
  const [queryForm] = Form.useForm<DeptListQuery>();
  const governanceRail = useGovernanceRail({ enabled: activeTab === 'manage' });
  const loadDataRequestIdRef = useRef(0);
  const invalidateDeptCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      {
        path: '/system/dept',
        resourceKeys: [
          'tree:default',
          'overview',
          'tree:sorted',
          'posts:org-chart',
          'users:org-chart',
        ],
      },
      { path: '/system/user', resourceKeys: ['depts:default'] },
      { path: '/system/post', resourceKeys: ['depts:sorted'] },
    ]);
  }, []);

  const buildGovernanceTaskQuery = useCallback(
    (nextQuery: DeptListQuery): DeptGovernanceTaskQuery => ({
      keyword: nextQuery.deptName || undefined,
      governance: nextQuery.governance as DeptGovernanceTaskQuery['governance'],
      scope: 'all',
    }),
    [],
  );

  const loadData = useCallback(
    async (nextQuery: DeptListQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      const requestId = loadDataRequestIdRef.current + 1;
      let rowsLoaded = false;
      loadDataRequestIdRef.current = requestId;
      if (!silent) {
        setLoading(true);
        setError(null);
        setGovernanceLoading(true);
      }
      try {
        const rows = await (isDefaultDeptListQuery(nextQuery)
          ? resolveRouteWarmData('/system/dept', 'tree:default', () => getDeptTree(nextQuery))
          : getDeptTree(nextQuery));
        if (loadDataRequestIdRef.current !== requestId) {
          return;
        }
        setData(rows);
        rowsLoaded = true;
        if (!silent) {
          setLoading(false);
        }
        const [overviewResp, taskRows] = await Promise.all([
          resolveRouteWarmData('/system/dept', 'overview', () => getDeptOverview()),
          getDeptGovernanceTasks(buildGovernanceTaskQuery(nextQuery)),
        ]);
        if (loadDataRequestIdRef.current !== requestId) {
          return;
        }
        setOverview(overviewResp);
        setGovernanceTasks(taskRows);
      } catch (requestError) {
        if (loadDataRequestIdRef.current !== requestId) {
          return;
        }
        if (!rowsLoaded) {
          setError(requestError);
        } else {
          setOverview(null);
        }
        setGovernanceTasks([]);
      } finally {
        if (!silent && loadDataRequestIdRef.current === requestId) {
          setLoading(false);
          setGovernanceLoading(false);
        }
      }
    },
    [buildGovernanceTaskQuery, query],
  );

  const loadAllDepts = useCallback(async () => {
    try {
      const rows = await resolveRouteWarmData('/system/dept', 'tree:sorted', () =>
        getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
      );
      setAllDeptTree(rows);
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  const loadOrgData = useCallback(
    async (options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setOrgLoading(true);
        setOrgError(null);
      }
      try {
        const [deptRows, postRows, userRows] = await Promise.all([
          resolveRouteWarmData('/system/dept', 'tree:sorted', () =>
            getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
          ),
          canViewPosts
            ? resolveRouteWarmData('/system/dept', 'posts:org-chart', () =>
                getPostList({
                  page: 1,
                  pageSize: orgChartPageSize,
                  sortField: 'sort',
                  sortOrder: 'asc',
                }),
              )
            : Promise.resolve({ items: [], total: 0, page: 1, pageSize: orgChartPageSize }),
          canViewUsers
            ? resolveRouteWarmData('/system/dept', 'users:org-chart', () =>
                getUserList({
                  page: 1,
                  pageSize: orgChartPageSize,
                  sortField: 'username',
                  sortOrder: 'asc',
                }),
              )
            : Promise.resolve({ items: [], total: 0, page: 1, pageSize: orgChartPageSize }),
        ]);
        setOrgDepts(deptRows);
        setOrgPosts(postRows.items);
        setOrgUsers(userRows.items);
        setSelectedOrgDeptId((current) => {
          if (current && findDeptNode(deptRows, current)) {
            return current;
          }
          return findRootDept(deptRows)?.id || flattenDeptNodes(deptRows)[0]?.id || 0;
        });
      } catch (requestError) {
        setOrgError(requestError);
      } finally {
        if (!silent) {
          setOrgLoading(false);
        }
      }
    },
    [canViewPosts, canViewUsers],
  );

  useEffect(() => {
    const timer = globalThis.setTimeout(() => loadData(query), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => loadAllDepts(), 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadAllDepts]);

  useEffect(() => {
    if (activeTab !== 'org') {
      return undefined;
    }
    const timer = globalThis.setTimeout(() => loadOrgData(), 0);
    return () => globalThis.clearTimeout(timer);
  }, [activeTab, loadOrgData]);

  useRefreshSubscription(
    ['system:dept:changed', 'system:post:changed', 'system:user:changed'],
    (payload) => {
      if (payload.source === 'system/dept') {
        return;
      }
      loadData(query);
      loadAllDepts();
      if (activeTab === 'org') {
        loadOrgData();
      }
    },
  );

  const tableTotalPages = useMemo(
    () => Math.max(1, Math.ceil(data.length / tablePagination.pageSize)),
    [data.length, tablePagination.pageSize],
  );
  const tableCurrentPage = useMemo(
    () => Math.min(tablePagination.current, tableTotalPages),
    [tablePagination.current, tableTotalPages],
  );

  useEffect(() => {
    const state =
      globalThis.history.state && typeof globalThis.history.state === 'object'
        ? (globalThis.history.state.usr as { deptId?: number; taskKey?: string } | null)
        : null;
    if (!state?.deptId) {
      return;
    }
    const deptId = state.deptId;
    const timer = globalThis.setTimeout(() => {
      setSelectedOrgDeptId(deptId);
      setActiveTab('org');
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, []);

  const deptOptions = useMemo<TreeSelectDataType[]>(() => {
    const build = (nodes: DeptNode[]): TreeSelectDataType[] =>
      nodes.map((node) => ({
        title: node.deptName,
        key: String(node.id),
        value: String(node.id),
        children: node.children?.length ? build(node.children) : undefined,
      }));
    return build(allDeptTree);
  }, [allDeptTree]);

  const flatOrgDepts = useMemo(() => flattenDeptNodes(orgDepts), [orgDepts]);
  const flatDeptRows = useMemo(() => flattenDeptNodes(data), [data]);
  const postsByDept = useMemo(() => groupByDept(orgPosts), [orgPosts]);
  const usersByDept = useMemo(() => groupByDept(orgUsers), [orgUsers]);
  const usersByPost = useMemo(() => groupUsersByPost(orgUsers), [orgUsers]);
  const selectedOrgDept = useMemo(
    () => findDeptNode(orgDepts, selectedOrgDeptId),
    [orgDepts, selectedOrgDeptId],
  );
  const selectedOrgStats = useMemo(() => {
    if (!selectedOrgDept) {
      return { deptCount: 0, postCount: 0, userCount: 0 };
    }
    const deptIDs = new Set(collectDeptIDs(selectedOrgDept));
    return {
      deptCount: deptIDs.size,
      postCount: orgPosts.filter((post) => deptIDs.has(post.deptId)).length,
      userCount: orgUsers.filter((user) => deptIDs.has(user.deptId)).length,
    };
  }, [orgPosts, orgUsers, selectedOrgDept]);
  const heroStats = useMemo(() => {
    if (!overview) {
      return [];
    }
    return [
      {
        key: 'totalDepts',
        label: t('system.dept.overview.totalDepts'),
        value: overview.totalDeptCount,
        hint: t('system.dept.hero.totalDeptsHint'),
      },
      {
        key: 'leaderless',
        label: t('system.dept.overview.leaderlessDepts'),
        value: overview.leaderlessDeptCount,
        hint: t('system.dept.hero.leaderlessHint'),
      },
      {
        key: 'issues',
        label: t('system.dept.overview.healthIssues'),
        value: overview.healthIssueCount,
        hint: t('system.dept.hero.issuesHint'),
      },
    ];
  }, [overview, t]);
  const governanceSummaryItems = useMemo(
    () =>
      overview
        ? [
            {
              label: t('system.dept.overview.healthIssues'),
              value: overview.healthIssueCount,
              description: t('system.dept.task.hint'),
            },
            {
              label: t('system.dept.overview.leaderlessDepts'),
              value: overview.leaderlessDeptCount,
              description: t('system.dept.leaderCandidateHint'),
            },
            {
              label: t('system.dept.overview.noPostDepts'),
              value: overview.noPostDeptCount,
              description: t('system.dept.governanceHint'),
            },
          ]
        : [],
    [overview, t],
  );
  const governanceTaskPreview = useMemo(() => governanceTasks.slice(0, 5), [governanceTasks]);

  const openCreate = () => {
    const rootDept = findRootDept(allDeptTree);
    setEditing(null);
    setLeaderCandidates([]);
    form.setFieldsValue({
      ...emptyForm,
      parentId: rootDept ? String(rootDept.id) : emptyForm.parentId,
    });
    setVisible(true);
  };

  const loadLeaderCandidateOptions = useCallback(
    async (deptId: number) => {
      setLeaderCandidateLoading(true);
      try {
        const items = await getDeptLeaderCandidates(deptId);
        setLeaderCandidates(items);
        return items;
      } catch {
        message.error(t('system.dept.leaderCandidateLoadFailed'));
        setLeaderCandidates([]);
        return [];
      } finally {
        setLeaderCandidateLoading(false);
      }
    },
    [t],
  );

  const openEdit = async (row: DeptNode) => {
    setEditing(row);
    setLeaderCandidates([]);
    let matchedLeaderUserId: number | undefined = row.leaderUserId || undefined;
    const items = row.isRoot ? [] : await loadLeaderCandidateOptions(row.id);
    if (!matchedLeaderUserId && row.leader) {
      const matched = items.find(
        (item) =>
          item.displayName === row.leader ||
          item.nickname === row.leader ||
          item.username === row.leader,
      );
      matchedLeaderUserId = matched?.userId;
    }
    form.setFieldsValue({
      parentId: row.isRoot ? String(row.id) : String(row.parentId),
      deptName: row.deptName,
      sort: row.sort,
      leaderUserId: matchedLeaderUserId,
      leader: row.leader,
      phone: row.phone,
      email: row.email,
      status: row.status,
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
    const payload: DeptPayload = {
      ...values,
      parentId: editing?.isRoot ? 0 : Number(values.parentId || 0),
      leaderUserId: values.leaderUserId ? Number(values.leaderUserId) : 0,
    };
    setSubmitting(true);
    try {
      if (editing) {
        await updateDept(editing.id, payload);
        message.success(t('common.updateSuccess'));
      } else {
        await createDept(payload);
        message.success(t('common.createSuccess'));
      }
      invalidateDeptCaches();
      publishRefresh('system:dept:changed', 'system/dept');
      setVisible(false);
      await loadData(query, { silent: true });
      await loadAllDepts();
      if (activeTab === 'org') {
        await loadOrgData({ silent: true });
      }
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleLeaderCandidateChange = (value: string | number | undefined) => {
    const numericValue = Number(value || 0);
    if (!numericValue) {
      return;
    }
    const matched = leaderCandidates.find((item) => item.userId === numericValue);
    if (matched) {
      form.setFieldValue('leader', matched.displayName);
    }
  };

  const removeDept = async (id: number) => {
    await deleteDept(id);
    message.success(t('common.deleteSuccess'));
    invalidateDeptCaches();
    publishRefresh('system:dept:changed', 'system/dept');
    setSelectedRowKeys((keys) => keys.filter((key) => Number(key) !== id));
    await loadData(query, { silent: true });
    await loadAllDepts();
    if (activeTab === 'org') {
      await loadOrgData({ silent: true });
    }
  };

  const search = () => {
    const values = queryForm.getFieldsValue();
    setSelectedRowKeys([]);
    setTablePagination((current) => ({ ...current, current: 1 }));
    setQuery({
      ...query,
      ...values,
    });
  };

  const applyGovernanceFilter = (governance?: 'leaderless' | 'no-post' | 'empty') => {
    const nextQuery: DeptListQuery = {
      ...query,
      governance,
    };
    queryForm.setFieldsValue({
      ...queryForm.getFieldsValue(),
      governance,
    });
    setSelectedRowKeys([]);
    setTablePagination((current) => ({ ...current, current: 1 }));
    setQuery(nextQuery);
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setTablePagination({ current: 1, pageSize: 10 });
    setQuery(emptyQuery);
  };

  const toArcoSortOrder = (sortOrder?: DeptListQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<DeptListQuery['sortField']>,
  ): Partial<ColumnProps<DeptNode>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<DeptNode>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    setTablePagination({
      current: pagination.current || 1,
      pageSize: pagination.pageSize || tablePagination.pageSize,
    });
    setSelectedRowKeys([]);
    setQuery({
      ...query,
      sortField: currentSorter?.direction ? String(currentSorter.field) : emptyQuery.sortField,
      sortOrder: currentSorter?.direction === 'descend' ? 'desc' : 'asc',
    });
  };

  const columns: ColumnProps<DeptNode>[] = [
    {
      title: t('system.dept.deptName'),
      dataIndex: 'deptName',
      width: TABLE_COLUMN_WIDTH.treeLabel,
      ...sortableColumn('deptName'),
      render: (_: unknown, row: DeptNode) => (
        <Space size={8}>
          <span>{row.deptName}</span>
          {row.isRoot ? <Tag color="arcoblue">{t('system.dept.root')}</Tag> : null}
        </Space>
      ),
    },
    withTableColumnPriority(
      {
        title: t('system.dept.leader'),
        dataIndex: 'leader',
        width: TABLE_COLUMN_WIDTH.identity,
        ...sortableColumn('leader'),
      },
      'medium',
    ),
    withTableColumnPriority(
      { title: t('system.dept.phone'), dataIndex: 'phone', width: TABLE_COLUMN_WIDTH.identity },
      'low',
    ),
    withTableColumnPriority(
      { title: t('system.dept.email'), dataIndex: 'email', width: TABLE_COLUMN_WIDTH.name },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.dept.sort'),
        dataIndex: 'sort',
        width: TABLE_COLUMN_WIDTH.count,
        ...sortableColumn('sort'),
      },
      'medium',
    ),
    {
      title: t('system.dept.governance'),
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (_: unknown, row: DeptNode) => {
        if (row.isRoot) {
          return <Tag>{t('system.dept.root')}</Tag>;
        }
        const tags = [];
        if (row.isLeaderless) {
          tags.push(
            <Tag key="leaderless" color="orange">
              {t('system.dept.governance.leaderless')}
            </Tag>,
          );
        }
        if (row.isNoPost) {
          tags.push(
            <Tag key="no-post" color="gold">
              {t('system.dept.governance.noPost')}
            </Tag>,
          );
        }
        if (row.isEmpty) {
          tags.push(
            <Tag key="empty" color="red">
              {t('system.dept.governance.empty')}
            </Tag>,
          );
        }
        if (tags.length === 0) {
          tags.push(
            <Tag key="clean" color="green">
              {t('system.dept.governance.clean')}
            </Tag>,
          );
        }
        return (
          <Space size={4} wrap>
            {tags}
          </Space>
        );
      },
    },
    {
      title: t('system.dept.status'),
      dataIndex: 'status',
      width: TABLE_COLUMN_WIDTH.status,
      ...sortableColumn('status'),
      render: (value: number) => (
        <Tag color={value === 1 ? 'green' : 'red'}>
          {value === 1 ? t('system.user.status.enabled') : t('system.user.status.disabled')}
        </Tag>
      ),
    },
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.medium,
      fixed: 'right',
      render: (_: unknown, row: DeptNode) => (
        <SystemRowActions
          actions={[
            {
              key: 'create-post',
              text: t('system.dept.action.createPost'),
              icon: <IconPlus />,
              onClick: () => openCreatePostForDept(row),
              hidden: !canCreatePost || row.isRoot || !row.isNoPost,
            },
            {
              key: 'edit',
              text: t('common.edit'),
              icon: <IconEdit />,
              onClick: () => {
                openEdit(row);
              },
              hidden: !canEdit,
            },
            {
              key: 'delete',
              text: t('common.delete'),
              icon: <IconDelete />,
              disabled: row.isRoot,
              hidden: !canDelete,
              status: 'danger',
              confirm: {
                title: t('system.dept.deleteConfirm'),
                onOk: () => removeDept(row.id),
              },
            },
          ]}
        />
      ),
    },
  ];

  const renderErrorState = () => {
    if (isNetworkRequestError(error)) {
      return (
        <PageNetworkError
          timeout={isTimeoutRequestError(error)}
          onRetry={() => {
            loadData(query);
          }}
        />
      );
    }
    if (isServerRequestError(error)) {
      return (
        <PageServerError
          onRetry={() => {
            loadData(query);
          }}
        />
      );
    }
    return (
      <PageError
        onRetry={() => {
          loadData(query);
        }}
      />
    );
  };

  const handleExport = async () => {
    await exportDepts(query);
  };

  const handleExportGovernanceTasks = async () => {
    await exportDeptGovernanceTasks(buildGovernanceTaskQuery(query));
  };

  const handleDownloadTemplate = async () => {
    await downloadDeptImportTemplate();
  };

  const handleImport = async (file: File) => {
    const result = await importDepts(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidateDeptCaches();
      publishRefresh('system:dept:changed', 'system/dept');
      await loadData(query, { silent: true });
      await loadAllDepts();
      if (activeTab === 'org') {
        await loadOrgData({ silent: true });
      }
    }
  };

  const handleBatchStatus = async (status: 1 | 2) => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const deptIds = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdateDeptStatus({ deptIds, status });
    message.success(t('system.dept.batchStatusSuccess', { count: result.updatedCount }));
    invalidateDeptCaches();
    publishRefresh('system:dept:changed', 'system/dept');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
    await loadAllDepts();
    if (activeTab === 'org') {
      await loadOrgData({ silent: true });
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteDepts({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateDeptCaches();
    publishRefresh('system:dept:changed', 'system/dept');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
    await loadAllDepts();
    if (activeTab === 'org') {
      await loadOrgData({ silent: true });
    }
  };

  const locateGovernanceTask = async (task: DeptGovernanceTask) => {
    const deptRow = flatDeptRows.find((item) => item.id === task.deptId);
    if (task.governanceScope === 'dept' && deptRow) {
      if (task.governanceAction === 'assign-leader' && canEdit) {
        await openEdit(deptRow);
        return;
      }
      if (task.governanceAction === 'create-post' && canCreatePost) {
        openCreatePostForDept(deptRow);
        return;
      }
      if (
        task.governanceTag === 'leaderless' ||
        task.governanceTag === 'no-post' ||
        task.governanceTag === 'empty'
      ) {
        applyGovernanceFilter(task.governanceTag as 'leaderless' | 'no-post' | 'empty');
        return;
      }
    }
    setActiveTab('org');
    setSelectedOrgDeptId(task.deptId);
  };

  const openBatchLeader = () => {
    const selectedDeptIds = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const selectedDepts = flatDeptRows.filter(
      (item) => selectedDeptIds.includes(item.id) && !item.isRoot,
    );
    if (selectedDepts.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    setLeaderVisible(true);
    setLeaderCandidateLoading(true);
    void Promise.all(
      selectedDepts.map(async (dept) => ({
        deptId: dept.id,
        deptName: dept.deptName,
        candidates: await getDeptLeaderCandidates(dept.id),
      })),
    )
      .then((tasks) => {
        setBatchLeaderTasks(tasks);
        const initialValues: DeptLeaderFormValues = {};
        tasks.forEach((task) => {
          if (task.candidates.length === 1) {
            initialValues[String(task.deptId)] = task.candidates[0].userId;
          }
        });
        leaderForm.setFieldsValue(initialValues);
      })
      .catch(() => {
        message.error(t('system.dept.leaderCandidateLoadFailed'));
        setLeaderVisible(false);
        setBatchLeaderTasks([]);
      })
      .finally(() => {
        setLeaderCandidateLoading(false);
      });
  };

  const submitBatchLeader = async () => {
    let values;
    try {
      values = await leaderForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    const items = batchLeaderTasks.map((task) => ({
      deptId: task.deptId,
      leaderUserId: Number(values[String(task.deptId)] || 0),
    }));
    if (items.some((item) => item.leaderUserId <= 0)) {
      message.warning(t('system.dept.batchLeaderRequired'));
      return;
    }
    const result = await batchUpdateDeptLeader({ items });
    message.success(t('system.dept.batchLeaderSuccess', { count: result.updatedCount }));
    invalidateDeptCaches();
    publishRefresh('system:dept:changed', 'system/dept');
    setLeaderVisible(false);
    setBatchLeaderTasks([]);
    leaderForm.resetFields();
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
    await loadAllDepts();
    if (activeTab === 'org') {
      await loadOrgData({ silent: true });
    }
  };

  const batchActionDisabled = !canBatchUpdate || selectedRowKeys.length === 0;
  const batchDeleteDisabled = !canBatchDelete || selectedRowKeys.length === 0;
  const batchLeaderDisabled = !canEdit || selectedRowKeys.length === 0;

  const openCreatePostForDept = (dept: DeptNode | null) => {
    if (!dept || dept.isRoot) {
      return;
    }
    setCreatingPostDept(dept);
    postForm.setFieldsValue({
      deptId: dept.id,
      postCode: '',
      postName: '',
      sort: 0,
      status: 1,
      remark: '',
    });
    setPostVisible(true);
  };

  const openCreatePost = () => {
    openCreatePostForDept(selectedOrgDept || null);
  };

  const submitPostForm = async () => {
    let values;
    try {
      values = await postForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    setPostSubmitting(true);
    try {
      await createPost(values);
      message.success(t('common.createSuccess'));
      invalidateDeptCaches();
      invalidateRouteWarmDataMany([
        { path: '/system/post', resourceKeys: ['list:default'] },
        { path: '/system/user', resourceKeys: ['posts:active'] },
      ]);
      publishRefresh(['system:dept:changed', 'system:post:changed'], 'system/dept');
      setPostVisible(false);
      setCreatingPostDept(null);
      await loadData(query, { silent: true });
      await loadAllDepts();
      await loadOrgData({ silent: true });
    } finally {
      setPostSubmitting(false);
    }
  };

  const openUserDetail = async (userId: number) => {
    if (!canViewUserDetail) {
      return;
    }
    setUserDetailId(userId);
    setUserDetailVisible(true);
    setUserDetailLoading(true);
    setUserDetailError(false);
    setUserDetail(null);
    try {
      const result = await getUserDetail(userId);
      setUserDetail(result);
    } catch {
      setUserDetailError(true);
    } finally {
      setUserDetailLoading(false);
    }
  };

  return (
    <PageContainer>
      <Space
        direction="vertical"
        size={12}
        className="system-page-template governance-workbench dept-list-page"
      >
        {overview ? (
          <GovernanceSummaryBar
            eyebrow={t('system.dept.hero.eyebrow')}
            title={t('system.dept.hero.title')}
            description={t('system.dept.hero.desc')}
            metrics={heroStats.slice(0, 4).map((item) => ({
              key: item.key,
              label: item.label,
              value: item.value,
              description: item.hint,
              role:
                item.key === 'leaderless' ||
                item.key === 'noPost' ||
                item.key === 'empty' ||
                item.key === 'issues'
                  ? 'button'
                  : undefined,
              tabIndex:
                item.key === 'leaderless' ||
                item.key === 'noPost' ||
                item.key === 'empty' ||
                item.key === 'issues'
                  ? 0
                  : undefined,
              onClick:
                item.key === 'leaderless'
                  ? () => applyGovernanceFilter('leaderless')
                  : item.key === 'noPost'
                    ? () => applyGovernanceFilter('no-post')
                    : item.key === 'empty'
                      ? () => applyGovernanceFilter('empty')
                      : item.key === 'issues'
                        ? () => applyGovernanceFilter(undefined)
                        : undefined,
              onKeyDown:
                item.key === 'leaderless' ||
                item.key === 'noPost' ||
                item.key === 'empty' ||
                item.key === 'issues'
                  ? (event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        if (item.key === 'leaderless') {
                          applyGovernanceFilter('leaderless');
                        } else if (item.key === 'noPost') {
                          applyGovernanceFilter('no-post');
                        } else if (item.key === 'empty') {
                          applyGovernanceFilter('empty');
                        } else {
                          applyGovernanceFilter(undefined);
                        }
                      }
                    }
                  : undefined,
            }))}
            action={
              <GovernanceRailToggleButton
                expanded={governanceRail.expanded}
                onToggle={governanceRail.toggle}
              >
                {t('system.dept.governance')}
              </GovernanceRailToggleButton>
            }
          />
        ) : null}
        <Tabs activeTab={activeTab} onChange={setActiveTab} className="system-dept-tabs">
          <Tabs.TabPane key="manage" title={t('system.dept.manageTab')}>
            <div className="page-main-column dept-list-page__layout">
              <FilterPanel>
                <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
                  <Row gutter={16}>
                    <Col xs={24} md={12} lg={6}>
                      <FormItem label={t('system.dept.deptName')} field="deptName">
                        <Input onPressEnter={() => queryForm.submit()} />
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12} lg={6}>
                      <FormItem label={t('system.dept.status')} field="status">
                        <Select
                          allowClear
                          options={[
                            { label: t('system.user.status.enabled'), value: 1 },
                            { label: t('system.user.status.disabled'), value: 2 },
                          ]}
                        />
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12} lg={6}>
                      <FormItem label={t('system.dept.governance')} field="governance">
                        <Select
                          allowClear
                          options={[
                            { label: t('system.dept.governance.leaderless'), value: 'leaderless' },
                            { label: t('system.dept.governance.noPost'), value: 'no-post' },
                            { label: t('system.dept.governance.empty'), value: 'empty' },
                          ]}
                        />
                      </FormItem>
                    </Col>
                    <Col xs={24} md={12} lg={6}>
                      <FormItem className="filter-panel__action-item">
                        <Space size={6}>
                          <Button
                            size="small"
                            type="primary"
                            htmlType="submit"
                            icon={<IconSearch />}
                          >
                            {t('common.search')}
                          </Button>
                          <Button size="small" onClick={reset}>
                            {t('common.reset')}
                          </Button>
                        </Space>
                      </FormItem>
                    </Col>
                  </Row>
                </Form>
              </FilterPanel>
              <Card className="page-panel system-list__table-card dept-list-page__table-card">
                <TableBatchActionBar
                  selectedCount={selectedRowKeys.length}
                  selectedText={t('common.selectedCount', { count: selectedRowKeys.length })}
                  clearText={t('common.clearSelection')}
                  clearSuccessText={t('common.clearSelectionSuccess')}
                  onClear={() => setSelectedRowKeys([])}
                  prefixActions={
                    <ListHeaderActions
                      className="dept-list-page__header-actions"
                      utility={
                        <>
                          <Button
                            size="small"
                            icon={<IconDownload />}
                            onClick={() => {
                              handleExport();
                            }}
                            disabled={!canExport}
                          >
                            {t('common.export')}
                          </Button>
                          <Button
                            size="small"
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
                          size="small"
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
                    !canBatchUpdate || !canBatchDelete || !canEdit
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
                          title={t('system.dept.batchEnableConfirm')}
                          onOk={() => {
                            handleBatchStatus(1);
                          }}
                          disabled={batchActionDisabled}
                        >
                          <Button size="small" disabled={batchActionDisabled}>
                            {t('system.dept.batchEnable')}
                          </Button>
                        </Popconfirm>
                      </PermissionAction>
                      <PermissionAction
                        allowed={canBatchUpdate}
                        tooltip={t('common.noPermissionAction')}
                      >
                        <Popconfirm
                          title={t('system.dept.batchDisableConfirm')}
                          onOk={() => {
                            handleBatchStatus(2);
                          }}
                          disabled={batchActionDisabled}
                        >
                          <Button
                            size="small"
                            status={batchActionDisabled ? undefined : 'warning'}
                            disabled={batchActionDisabled}
                          >
                            {t('system.dept.batchDisable')}
                          </Button>
                        </Popconfirm>
                      </PermissionAction>
                      <PermissionAction
                        allowed={canBatchDelete}
                        tooltip={t('common.noPermissionAction')}
                      >
                        <Popconfirm
                          title={t('system.dept.batchDeleteConfirm')}
                          onOk={() => {
                            handleBatchDelete();
                          }}
                          disabled={batchDeleteDisabled}
                        >
                          <Button
                            size="small"
                            status="danger"
                            icon={<IconDelete />}
                            disabled={batchDeleteDisabled}
                          >
                            {t('common.deleteSelected')}
                          </Button>
                        </Popconfirm>
                      </PermissionAction>
                      <PermissionAction allowed={canEdit} tooltip={t('common.noPermissionAction')}>
                        <Button
                          size="small"
                          onClick={openBatchLeader}
                          disabled={batchLeaderDisabled}
                        >
                          {t('system.dept.batchLeader')}
                        </Button>
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
                  <AppTable<DeptNode>
                    className="system-list__table"
                    data={data}
                    columns={columns}
                    rowKey="id"
                    loading={loading}
                    scroll={{ x: 'max-content' }}
                    rowSelection={{
                      type: 'checkbox',
                      selectedRowKeys,
                      fixed: true,
                      checkboxProps: (row) => ({ disabled: row.isRoot }),
                      onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
                    }}
                    onChange={handleTableChange}
                    emptyText={t('common.noData')}
                    pagination={buildStandardPagination(t, {
                      current: tableCurrentPage,
                      pageSize: tablePagination.pageSize,
                      total: data.length,
                    })}
                  />
                ) : null}
              </Card>
            </div>
          </Tabs.TabPane>
          <Tabs.TabPane key="org" title={t('system.dept.orgTab')}>
            <DeptOrgTab
              orgDepts={orgDepts}
              orgPosts={orgPosts}
              orgUsers={orgUsers}
              orgLoading={orgLoading}
              orgError={orgError}
              selectedOrgDeptId={selectedOrgDeptId}
              onSelectDept={setSelectedOrgDeptId}
              flatOrgDepts={flatOrgDepts}
              postsByDept={postsByDept}
              usersByDept={usersByDept}
              usersByPost={usersByPost}
              selectedOrgDept={selectedOrgDept}
              selectedOrgStats={selectedOrgStats}
              canViewPosts={canViewPosts}
              canViewUsers={canViewUsers}
              canCreatePost={canCreatePost}
              canViewUserDetail={canViewUserDetail}
              onRefresh={() => {
                loadOrgData();
              }}
              onCreatePost={openCreatePost}
              onViewUserDetail={(id) => {
                openUserDetail(id);
              }}
            />
          </Tabs.TabPane>
        </Tabs>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.dept.governance')}
        visible={governanceRail.expanded && Boolean(overview)}
        onClose={governanceRail.close}
        noteTitle={t('system.dept.governance')}
        noteDescription={t('system.dept.governanceHint')}
      >
        <div className="dept-governance-rail">
          <GovernanceRailSummary
            items={[
              ...governanceSummaryItems,
              {
                label: t('system.dept.task.title'),
                value: governanceTasks.length,
                description: t('system.dept.task.hint'),
              },
            ]}
          />
          <div className="dept-governance-rail__tasks">
            <div className="dept-governance-rail__tasks-head">
              <div className="dept-governance-rail__tasks-copy">
                <Typography.Text className="dept-governance-rail__tasks-title">
                  {t('system.dept.task.title')}
                </Typography.Text>
                <Typography.Text type="secondary" className="dept-governance-rail__tasks-hint">
                  {t('system.dept.task.hint')}
                </Typography.Text>
              </div>
              <Space wrap>
                <Typography.Text type="secondary">
                  {t('common.total', { count: governanceTasks.length })}
                </Typography.Text>
                <Button
                  size="small"
                  onClick={() => {
                    loadData(query);
                  }}
                  loading={governanceLoading}
                >
                  {t('common.refresh')}
                </Button>
                <Button
                  size="small"
                  icon={<IconDownload />}
                  onClick={() => {
                    handleExportGovernanceTasks();
                  }}
                  disabled={!canExport}
                >
                  {t('system.dept.task.export')}
                </Button>
              </Space>
            </div>
            {governanceTaskPreview.length > 0 ? (
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                {governanceTaskPreview.map((task) => (
                  <div key={task.taskKey} className="dept-governance-rail__task">
                    <div className="dept-governance-rail__task-head">
                      <Space wrap size={6}>
                        <Tag color="arcoblue">{task.governanceScopeLabel}</Tag>
                        <Tag color="orange">{task.governanceTagLabel}</Tag>
                        {task.relatedUserCount > 0 ? (
                          <Tag color="gold">
                            {t('system.dept.task.relatedUserCount')}: {task.relatedUserCount}
                          </Tag>
                        ) : null}
                      </Space>
                      <Button
                        type="text"
                        size="small"
                        icon={<IconEye />}
                        onClick={() => {
                          locateGovernanceTask(task);
                        }}
                      >
                        {t('system.dept.task.locate')}
                      </Button>
                    </div>
                    <Typography.Text className="dept-governance-rail__task-resource">
                      {task.governanceScope === 'post'
                        ? `${task.postName || '-'} / ${task.deptName || '-'}`
                        : task.deptName}
                    </Typography.Text>
                    <Typography.Text type="secondary" className="dept-governance-rail__task-meta">
                      {task.governanceActionLabel}
                      {task.governanceBlockedByLabel ? ` · ${task.governanceBlockedByLabel}` : ''}
                    </Typography.Text>
                  </div>
                ))}
              </Space>
            ) : (
              <Typography.Text type="secondary">{t('common.noData')}</Typography.Text>
            )}
          </div>
        </div>
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.dept.edit') : t('system.dept.create')}
        visible={visible}
        size="lg"
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
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.dept.parentId')} field="parentId">
                    <TreeSelect treeData={deptOptions} disabled={editing?.isRoot} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.dept.deptName')}
                    field="deptName"
                    rules={[{ required: true, message: t('system.dept.deptNameRequired') }]}
                  >
                    <Input onPressEnter={() => form.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.dept.leaderCandidate')}
                    field="leaderUserId"
                    extra={
                      editing
                        ? leaderCandidates.length > 0
                          ? t('system.dept.leaderCandidateHint')
                          : t('system.dept.leaderCandidateEmpty')
                        : t('system.dept.leaderCandidateCreateHint')
                    }
                  >
                    <Select
                      allowClear
                      showSearch
                      loading={leaderCandidateLoading}
                      disabled={!editing || editing.isRoot}
                      placeholder={
                        editing
                          ? t('system.dept.leaderCandidatePlaceholder')
                          : t('system.dept.leaderCandidateCreateHint')
                      }
                      options={leaderCandidates.map((item) => ({
                        label: `${item.displayName} · ${item.postName || '-'}`,
                        value: item.userId,
                      }))}
                      onChange={handleLeaderCandidateChange}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.dept.leader')} field="leader">
                    <Input
                      placeholder={t('system.dept.leaderLegacyPlaceholder')}
                      onPressEnter={() => form.submit()}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.dept.phone')} field="phone">
                    <Input onPressEnter={() => form.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.dept.email')}
                    field="email"
                    rules={[{ match: /\S+@\S+\.\S+/, message: t('system.user.email.invalid') }]}
                  >
                    <Input onPressEnter={() => form.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={6}>
                  <FormItem label={t('system.dept.sort')} field="sort">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </FormItem>
                </Col>
                <Col xs={24} md={6}>
                  <FormItem label={t('system.dept.status')} field="status">
                    <Select
                      options={[
                        { label: t('system.user.status.enabled'), value: 1 },
                        { label: t('system.user.status.disabled'), value: 2 },
                      ]}
                      disabled={editing?.isRoot}
                    />
                  </FormItem>
                </Col>
              </Row>
            </FormSection>
          </Space>
        </Form>
      </AppModal>

      <AppModal
        title={t('system.dept.batchLeader')}
        visible={leaderVisible}
        size="lg"
        onCancel={() => {
          setLeaderVisible(false);
          setBatchLeaderTasks([]);
          leaderForm.resetFields();
        }}
        footer={
          <SubmitBar
            onCancel={() => {
              setLeaderVisible(false);
              setBatchLeaderTasks([]);
              leaderForm.resetFields();
            }}
            onSubmit={() => {
              submitBatchLeader();
            }}
            submitText={t('common.save')}
          />
        }
        unmountOnExit
      >
        <Form
          form={leaderForm}
          layout="vertical"
          onSubmit={() => {
            submitBatchLeader();
          }}
        >
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <div>{t('system.dept.batchLeaderTaskHint')}</div>
            {batchLeaderTasks.map((task) => (
              <Card key={task.deptId} size="small">
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <strong>{task.deptName}</strong>
                  <FormItem
                    label={t('system.dept.leaderCandidate')}
                    field={String(task.deptId)}
                    rules={[{ required: true, message: t('dept.leader.required') }]}
                    extra={
                      task.candidates.length > 0
                        ? t('system.dept.leaderCandidateHint')
                        : t('system.dept.batchLeaderNoCandidate')
                    }
                  >
                    <Select
                      showSearch
                      allowClear
                      loading={leaderCandidateLoading}
                      disabled={task.candidates.length === 0}
                      placeholder={
                        task.candidates.length > 0
                          ? t('system.dept.leaderCandidatePlaceholder')
                          : t('system.dept.batchLeaderNoCandidate')
                      }
                      options={task.candidates.map((item) => ({
                        label: `${item.displayName} · ${item.postName || '-'}`,
                        value: item.userId,
                      }))}
                    />
                  </FormItem>
                </Space>
              </Card>
            ))}
          </Space>
        </Form>
      </AppModal>

      <AppModal
        title={t('system.dept.orgCreatePostTitle')}
        visible={postVisible}
        size="md"
        onCancel={() => {
          setPostVisible(false);
          setCreatingPostDept(null);
        }}
        footer={
          <SubmitBar
            onCancel={() => {
              setPostVisible(false);
              setCreatingPostDept(null);
            }}
            onSubmit={() => {
              submitPostForm();
            }}
            loading={postSubmitting}
            submitText={t('common.add')}
          />
        }
        unmountOnExit
      >
        <Form
          form={postForm}
          layout="vertical"
          onSubmit={() => {
            submitPostForm();
          }}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('common.basicInfo')}>
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.post.dept')}
                    field="deptId"
                    rules={[{ required: true, message: t('system.post.deptRequired') }]}
                  >
                    <Select
                      disabled
                      options={
                        creatingPostDept
                          ? [{ label: creatingPostDept.deptName, value: creatingPostDept.id }]
                          : []
                      }
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.post.postCode')}
                    field="postCode"
                    rules={[{ required: true, message: t('system.post.postCodeRequired') }]}
                  >
                    <Input onPressEnter={() => postForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.post.postName')}
                    field="postName"
                    rules={[{ required: true, message: t('system.post.postNameRequired') }]}
                  >
                    <Input onPressEnter={() => postForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.post.sort')} field="sort">
                    <InputNumber min={0} style={{ width: '100%' }} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.post.status')} field="status">
                    <Select
                      options={[
                        { label: t('system.user.status.enabled'), value: 1 },
                        { label: t('system.user.status.disabled'), value: 2 },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col span={24}>
                  <FormItem label={t('system.post.remark')} field="remark">
                    <Input.TextArea maxLength={200} autoSize={{ minRows: 3, maxRows: 5 }} />
                  </FormItem>
                </Col>
              </Row>
            </FormSection>
          </Space>
        </Form>
      </AppModal>

      <AppModal
        title={t('system.user.detail')}
        visible={userDetailVisible}
        size="detail"
        footer={null}
        onCancel={() => setUserDetailVisible(false)}
        unmountOnExit
      >
        {userDetailLoading ? <PageLoading /> : null}
        {userDetailError ? (
          <PageError
            onRetry={() => {
              if (userDetailId > 0) {
                openUserDetail(userDetailId);
              }
            }}
          />
        ) : null}
        {!userDetailLoading && !userDetailError && userDetail ? (
          <UserDetailContent detail={userDetail} />
        ) : null}
      </AppModal>
    </PageContainer>
  );
};

export default DeptList;
