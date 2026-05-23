import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Grid,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Tag,
  Tree,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import type { TreeDataType } from '@arco-design/web-react/es/Tree/interface';
import {
  IconDelete,
  IconDownload,
  IconEdit,
  IconPlus,
  IconSearch,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { formatDateTime } from '../../../core/format/dateTime';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany, resolveRouteWarmData } from '../../../core/router/prefetch';
import { usePermission } from '../../../hooks/usePermission';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../components/table/crossPageSelection';
import { getMenuTree, type MenuNode } from '../menu/api';
import {
  batchDeleteRoles,
  batchUpdateRoleStatus,
  createRole,
  deleteRole,
  exportRoles,
  getRoleList,
  updateRole,
  type RoleListQuery,
  type RolePayload,
  type RoleRow,
} from './api';
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

interface RoleFormValues {
  roleName: string;
  roleKey: string;
  sort: number;
  status: number;
  menuIds: string[];
  pagePermissionKeys: string[];
  actionPermissionKeys: string[];
  unknownPermissionKeys: string[];
}

const emptyForm: RoleFormValues = {
  roleName: '',
  roleKey: '',
  sort: 0,
  status: 1,
  menuIds: [],
  pagePermissionKeys: [],
  actionPermissionKeys: [],
  unknownPermissionKeys: [],
};

const emptyQuery: RoleListQuery = {
  roleName: '',
  roleKey: '',
  status: undefined,
  page: 1,
  pageSize: 10,
};

function isDefaultRoleListQuery(query: RoleListQuery) {
  return (
    !query.roleName &&
    !query.roleKey &&
    query.status === undefined &&
    (query.page ?? 1) === 1 &&
    (query.pageSize ?? 10) === 10 &&
    !query.sortField &&
    !query.sortOrder
  );
}

const emptyAuthorizationCounts = {
  navigation: 0,
  page: 0,
  action: 0,
  unknown: 0,
};

interface LoadDataOptions {
  silent?: boolean;
}

const mergePermissionKeys = (...groups: string[][]) =>
  Array.from(new Set(groups.flat().filter(Boolean)));

interface PermissionTreeSelectorProps {
  treeData: TreeDataType[];
  permissionKeys: Set<string>;
  defaultExpandedKeys: string[];
  value?: string[];
  onChange?: (nextValue: string[]) => void;
  emptyText: string;
  searchPlaceholder: string;
  expandAllText: string;
  collapseAllText: string;
}

const PermissionTreeSelector: React.FC<PermissionTreeSelectorProps> = ({
  treeData,
  permissionKeys,
  defaultExpandedKeys,
  value = [],
  onChange,
  emptyText,
  searchPlaceholder,
  expandAllText,
  collapseAllText,
}) => {
  const [keyword, setKeyword] = useState('');
  const [expandedKeys, setExpandedKeys] = useState<string[]>(defaultExpandedKeys);

  const collectExpandableKeys = useCallback((nodes: TreeDataType[]) => {
    const nextKeys: string[] = [];
    const walk = (items: TreeDataType[]) => {
      items.forEach((item) => {
        const children = Array.isArray(item.children) ? item.children : [];
        if (children.length > 0) {
          nextKeys.push(String(item.key));
          walk(children);
        }
      });
    };
    walk(nodes);
    return nextKeys;
  }, []);

  const filteredTreeData = useMemo(() => {
    const normalizedKeyword = keyword.trim().toLowerCase();
    if (!normalizedKeyword) {
      return treeData;
    }
    const walk = (nodes: TreeDataType[]): TreeDataType[] =>
      nodes.reduce<TreeDataType[]>((items, node) => {
        const children = Array.isArray(node.children) ? walk(node.children) : [];
        const searchText = String(node.searchText || '').toLowerCase();
        if (!searchText.includes(normalizedKeyword) && children.length === 0) {
          return items;
        }
        items.push({
          ...node,
          children: children.length > 0 ? children : undefined,
        });
        return items;
      }, []);
    return walk(treeData);
  }, [keyword, treeData]);

  const effectiveExpandedKeys = useMemo(() => {
    if (keyword.trim()) {
      return collectExpandableKeys(filteredTreeData);
    }
    return expandedKeys;
  }, [collectExpandableKeys, expandedKeys, filteredTreeData, keyword]);

  if (treeData.length === 0) {
    return <PageEmpty description={emptyText} />;
  }

  return (
    <div className="role-permission-tree-panel">
      <div className="role-permission-tree__toolbar">
        <Input
          allowClear
          className="role-permission-tree__search"
          prefix={<IconSearch />}
          placeholder={searchPlaceholder}
          value={keyword}
          onChange={setKeyword}
        />
        <Space size={4}>
          <Button
            type="text"
            size="mini"
            onClick={() => setExpandedKeys(collectExpandableKeys(treeData))}
          >
            {expandAllText}
          </Button>
          <Button type="text" size="mini" onClick={() => setExpandedKeys([])}>
            {collapseAllText}
          </Button>
        </Space>
      </div>
      <Tree
        className="role-permission-tree"
        blockNode
        checkable
        selectable={false}
        showLine
        expandedKeys={effectiveExpandedKeys}
        autoExpandParent
        checkedKeys={value}
        treeData={filteredTreeData}
        onExpand={(nextExpandedKeys) => setExpandedKeys(nextExpandedKeys)}
        onCheck={(checkedKeys) => {
          onChange?.(checkedKeys.filter((permissionKey) => permissionKeys.has(permissionKey)));
        }}
      />
    </div>
  );
};

const RoleList: React.FC = () => {
  const [data, setData] = useState<RoleRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<RoleRow | null>(null);
  const [query, setQuery] = useState<RoleListQuery>(emptyQuery);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [menuTree, setMenuTree] = useState<MenuNode[]>([]);
  const [authorizationCounts, setAuthorizationCounts] = useState(emptyAuthorizationCounts);
  const [form] = Form.useForm<RoleFormValues>();
  const [queryForm] = Form.useForm<RoleListQuery>();
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:role:create');
  const canEdit = isAdmin || hasPerm('system:role:update');
  const canDelete = isAdmin || hasPerm('system:role:delete');
  const canBatchUpdate = isAdmin || hasPerm('system:role:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:role:batch-delete');
  const canExport = isAdmin || hasPerm('system:role:export');
  const governanceRail = useGovernanceRail();
  const invalidateRoleCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/role', resourceKeys: ['list:default'] },
      { path: '/system/user', resourceKeys: ['roles:active'] },
      { path: '/system/permission', resourceKeys: ['roles:default', 'workbench:default'] },
    ]);
  }, []);

  const loadData = useCallback(
    async (nextQuery: RoleListQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      try {
        const result = isDefaultRoleListQuery(nextQuery)
          ? await resolveRouteWarmData('/system/role', 'list:default', () => getRoleList(nextQuery))
          : await getRoleList(nextQuery);
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

  const loadMenus = useCallback(async () => {
    try {
      const rows = await resolveRouteWarmData('/system/role', 'menus:manage', () =>
        getMenuTree({ scope: 'manage' }),
      );
      setMenuTree(rows);
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadData(query);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadMenus();
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadMenus]);

  useRefreshSubscription(
    ['system:menu:changed', 'system:permission:changed', 'system:role:changed'],
    (payload) => {
      if (payload.source === 'system/role') {
        return;
      }
      void loadData(query);
      if (payload.topic === 'system:menu:changed') {
        void loadMenus();
      }
    },
  );

  const visibleSelectedRowKeys = useMemo(() => {
    return getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.id));
  }, [data, selectedRowKeys]);

  const permissionCatalog = useMemo(() => {
    const navigationKeys = new Set<string>();
    const pageKeys = new Set<string>();
    const actionKeys = new Set<string>();
    const walk = (nodes: MenuNode[]) => {
      nodes.forEach((item) => {
        if (item.type !== 'F') {
          navigationKeys.add(String(item.id));
        }
        if (item.type === 'C' && item.pagePerm && !pageKeys.has(item.pagePerm)) {
          pageKeys.add(item.pagePerm);
        }
        if (item.type === 'F' && item.perms && !actionKeys.has(item.perms)) {
          actionKeys.add(item.perms);
        }
        if (item.children?.length) {
          walk(item.children);
        }
      });
    };
    walk(menuTree);
    return {
      navigationKeys,
      pageKeys,
      actionKeys,
    };
  }, [menuTree]);

  const splitPermissionKeys = useCallback(
    (permissionKeys: string[] = []) => {
      const pagePermissionKeys: string[] = [];
      const actionPermissionKeys: string[] = [];
      const unknownPermissionKeys: string[] = [];

      permissionKeys.forEach((permissionKey) => {
        if (permissionCatalog.pageKeys.has(permissionKey)) {
          pagePermissionKeys.push(permissionKey);
          return;
        }
        if (permissionCatalog.actionKeys.has(permissionKey)) {
          actionPermissionKeys.push(permissionKey);
          return;
        }
        unknownPermissionKeys.push(permissionKey);
      });

      return {
        pagePermissionKeys,
        actionPermissionKeys,
        unknownPermissionKeys,
      };
    },
    [permissionCatalog],
  );

  const updateAuthorizationCounts = (values: Partial<RoleFormValues>) => {
    setAuthorizationCounts({
      navigation: values.menuIds?.length || 0,
      page: values.pagePermissionKeys?.length || 0,
      action: values.actionPermissionKeys?.length || 0,
      unknown: values.unknownPermissionKeys?.length || 0,
    });
  };

  const authorizationTitle = (label: string, count: number, color: string) => (
    <Space size={8}>
      <Typography.Text className="font-semibold">{label}</Typography.Text>
      <Tag color={color}>{t('system.role.selectedCount', { count })}</Tag>
    </Space>
  );

  const navigationPermissionTree = useMemo(() => {
    const expandedKeys: string[] = [];
    const buildTree = (nodes: MenuNode[]): TreeDataType[] =>
      nodes
        .filter((item) => item.type !== 'F')
        .map((item) => ({
          key: String(item.id),
          searchText: `${t(item.titleKey)} ${item.path || ''}`,
          title: (
            <span className="role-permission-tree__node">
              <span className="role-permission-tree__title">{t(item.titleKey)}</span>
              {item.path ? (
                <Tag size="small" color="arcoblue">
                  {item.path}
                </Tag>
              ) : null}
            </span>
          ),
          children: item.children?.length ? buildTree(item.children) : undefined,
        }));
    const treeData = buildTree(menuTree);
    treeData.forEach((item) => {
      if (Array.isArray(item.children) && item.children.length > 0) {
        expandedKeys.push(String(item.key));
      }
    });
    return {
      treeData,
      expandedKeys,
    };
  }, [menuTree, t]);

  const pagePermissionTree = useMemo(() => {
    const expandedKeys: string[] = [];
    const buildTree = (nodes: MenuNode[], depth = 0): TreeDataType[] =>
      nodes
        .filter((item) => item.type !== 'F')
        .map((item) => {
          const children = item.children?.length ? buildTree(item.children, depth + 1) : [];
          if (!item.pagePerm && children.length === 0) {
            return null;
          }
          const key = item.pagePerm || `menu-${item.id}`;
          if (children.length > 0 && depth === 0) {
            expandedKeys.push(key);
          }
          return {
            key,
            searchText: `${t(item.titleKey)} ${item.pagePerm || ''}`,
            disableCheckbox: !item.pagePerm && children.length === 0,
            title: (
              <span className="role-permission-tree__node">
                <span className="role-permission-tree__title">{t(item.titleKey)}</span>
                {item.pagePerm ? (
                  <Tag size="small" color="green">
                    {item.pagePerm}
                  </Tag>
                ) : null}
              </span>
            ),
            children: children.length > 0 ? children : undefined,
          } as TreeDataType;
        })
        .filter((item): item is TreeDataType => Boolean(item));
    return {
      treeData: buildTree(menuTree),
      expandedKeys,
    };
  }, [menuTree, t]);

  const actionPermissionTree = useMemo(() => {
    const expandedKeys: string[] = [];
    const buildTree = (nodes: MenuNode[], depth = 0): TreeDataType[] =>
      nodes
        .map((item) => {
          if (item.type === 'F') {
            if (!item.perms) {
              return null;
            }
            return {
              key: item.perms,
              searchText: `${t(item.titleKey)} ${item.perms}`,
              title: (
                <span className="role-permission-tree__node">
                  <span className="role-permission-tree__title">{t(item.titleKey)}</span>
                  <Tag size="small" color="orange">
                    {item.perms}
                  </Tag>
                </span>
              ),
            } as TreeDataType;
          }
          const children = item.children?.length ? buildTree(item.children, depth + 1) : [];
          if (children.length === 0) {
            return null;
          }
          const key = `action-menu-${item.id}`;
          if (depth === 0) {
            expandedKeys.push(key);
          }
          return {
            key,
            searchText: `${t(item.titleKey)} ${item.pagePerm || ''}`,
            disableCheckbox: children.length === 0,
            title: (
              <span className="role-permission-tree__node">
                <span className="role-permission-tree__title">{t(item.titleKey)}</span>
                {item.pagePerm ? (
                  <Tag size="small" color="arcoblue">
                    {item.pagePerm}
                  </Tag>
                ) : null}
              </span>
            ),
            children,
          } as TreeDataType;
        })
        .filter((item): item is TreeDataType => Boolean(item));
    return {
      treeData: buildTree(menuTree),
      expandedKeys,
    };
  }, [menuTree, t]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(emptyForm);
    updateAuthorizationCounts(emptyForm);
    setVisible(true);
  };

  const openEdit = (row: RoleRow) => {
    const splitPermissions = splitPermissionKeys(row.permissionKeys);
    const formValues = {
      roleName: row.roleName,
      roleKey: row.roleKey,
      sort: row.sort,
      status: row.status,
      menuIds: row.menuIds.map(String),
      ...splitPermissions,
    };
    setEditing(row);
    form.setFieldsValue(formValues);
    updateAuthorizationCounts(formValues);
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
    const payload: RolePayload = {
      roleName: values.roleName,
      roleKey: values.roleKey,
      sort: values.sort,
      status: values.status,
      menuIds: values.menuIds.map((item) => Number(item)),
      permissionKeys: mergePermissionKeys(
        values.pagePermissionKeys,
        values.actionPermissionKeys,
        values.unknownPermissionKeys,
      ),
    };
    setSubmitting(true);
    try {
      if (editing) {
        await updateRole(editing.id, payload);
        message.success(t('common.updateSuccess'));
      } else {
        await createRole(payload);
        message.success(t('common.createSuccess'));
      }
      invalidateRoleCaches();
      publishRefresh('system:role:changed', 'system/role');
      setVisible(false);
      await loadData(query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const removeRole = async (row: RoleRow) => {
    await deleteRole(row.id);
    message.success(t('common.deleteSuccess'));
    invalidateRoleCaches();
    publishRefresh('system:role:changed', 'system/role');
    setSelectedRowKeys((keys) => keys.filter((key) => Number(key) !== row.id));
    const nextPage =
      data.length === 1 && (query.page || 1) > 1 ? (query.page || 1) - 1 : query.page || 1;
    const nextQuery = { ...query, page: nextPage };
    setQuery(nextQuery);
  };

  const handleBatchStatus = async (status: 1 | 2) => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const roleIds = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdateRoleStatus({ roleIds, status });
    message.success(t('system.role.batchStatusSuccess', { count: result.updatedCount }));
    invalidateRoleCaches();
    publishRefresh('system:role:changed', 'system/role');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeleteRoles({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidateRoleCaches();
    publishRefresh('system:role:changed', 'system/role');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const handleExport = async () => {
    await exportRoles(query);
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

  const toArcoSortOrder = (sortOrder?: RoleListQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<RoleListQuery['sortField']>,
  ): Partial<ColumnProps<RoleRow>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<RoleRow>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    const nextQuery: RoleListQuery = {
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
    const sortChanged =
      nextQuery.sortField !== query.sortField || nextQuery.sortOrder !== query.sortOrder;
    if (sortChanged) {
      setSelectedRowKeys([]);
    }
    setQuery(nextQuery);
  };

  const columns: ColumnProps<RoleRow>[] = [
    {
      title: t('system.role.roleName'),
      dataIndex: 'roleName',
      width: TABLE_COLUMN_WIDTH.name,
      ...sortableColumn('roleName'),
    },
    withTableColumnPriority(
      {
        title: t('system.role.roleKey'),
        dataIndex: 'roleKey',
        width: TABLE_COLUMN_WIDTH.code,
        ...sortableColumn('roleKey'),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.role.sort'),
        dataIndex: 'sort',
        width: TABLE_COLUMN_WIDTH.count,
        ...sortableColumn('sort'),
      },
      'low',
    ),
    {
      title: t('system.role.status'),
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
        title: t('system.role.createdAt'),
        dataIndex: 'createdAt',
        width: TABLE_COLUMN_WIDTH.datetime,
        ...sortableColumn('createdAt'),
        render: (value: string) => formatDateTime(value),
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.compact,
      fixed: 'right',
      render: (_: unknown, row: RoleRow) => (
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
              disabled: row.roleKey === 'admin',
              hidden: !canDelete,
              status: 'danger',
              confirm: {
                title: t('common.deleteConfirm'),
                onOk: () => removeRole(row),
              },
            },
          ]}
        />
      ),
    },
  ];

  const protectedRole = editing?.roleKey === 'admin';
  const unknownPermissionKeys = form.getFieldValue('unknownPermissionKeys');
  const unknownPermissionOptions = Array.isArray(unknownPermissionKeys)
    ? unknownPermissionKeys.map((permissionKey) => ({
        label: String(permissionKey),
        value: String(permissionKey),
      }))
    : [];

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

  const batchActionDisabled = !canBatchUpdate || selectedRowKeys.length === 0;
  const batchDeleteDisabled = !canBatchDelete || selectedRowKeys.length === 0;
  const enabledRoleCount = useMemo(() => data.filter((item) => item.status === 1).length, [data]);
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('common.total', { count: total }),
        value: total,
        hint: t('system.role.hero.totalHint'),
      },
      {
        key: 'enabled',
        label: t('system.user.status.enabled'),
        value: enabledRoleCount,
        hint: t('system.role.hero.enabledHint'),
      },
      {
        key: 'selected',
        label: t('system.role.hero.selectedRows'),
        value: selectedRowKeys.length,
        hint: t('system.role.hero.selectedHint'),
      },
      {
        key: 'menus',
        label: t('system.role.hero.menuReady'),
        value: permissionCatalog.navigationKeys.size,
        hint: t('system.role.hero.menuHint'),
      },
    ],
    [enabledRoleCount, permissionCatalog.navigationKeys.size, selectedRowKeys.length, t, total],
  );
  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.role.hero.pagePerms'),
        value: permissionCatalog.pageKeys.size,
        description: t('system.role.hero.pagePermsHint'),
      },
      {
        label: t('system.role.hero.actionPerms'),
        value: permissionCatalog.actionKeys.size,
        description: t('system.role.hero.actionPermsHint'),
      },
      {
        label: t('system.role.hero.batchActions'),
        value: batchActionDisabled ? t('common.no') : t('common.yes'),
        description: t('system.role.hero.batchHint'),
      },
    ],
    [batchActionDisabled, permissionCatalog.actionKeys.size, permissionCatalog.pageKeys.size, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('system.role.hero.eyebrow')}
          title={t('system.role.hero.title')}
          description={t('system.role.hero.desc')}
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
              {t('system.role.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16}>
                <Col span={6}>
                  <FormItem label={t('system.role.roleName')} field="roleName">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.role.roleKey')} field="roleKey">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={6}>
                  <FormItem label={t('system.role.status')} field="status">
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
                    <Button
                      icon={<IconDownload />}
                      onClick={() => {
                        void handleExport();
                      }}
                      disabled={!canExport}
                    >
                      {t('common.export')}
                    </Button>
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
                      title={t('system.role.batchEnableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(1);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button disabled={batchActionDisabled}>{t('system.role.batchEnable')}</Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchUpdate}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.role.batchDisableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(2);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button
                        status={batchActionDisabled ? undefined : 'warning'}
                        disabled={batchActionDisabled}
                      >
                        {t('system.role.batchDisable')}
                      </Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchDelete}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.role.batchDeleteConfirm')}
                      onOk={() => {
                        void handleBatchDelete();
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
                </>
              }
            />
            {loading && data.length === 0 ? <PageLoading /> : null}
            {error && data.length === 0 ? renderErrorState() : null}
            {!loading && !error && data.length === 0 ? (
              <PageEmpty description={t('common.noData')} />
            ) : null}
            {!loading && !(error && data.length === 0) && data.length > 0 ? (
              <AppTable<RoleRow>
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
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.role.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.role.hero.summaryTitle')}
        noteDescription={t('system.role.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.role.edit') : t('system.role.create')}
        visible={visible}
        size="xl"
        onCancel={() => setVisible(false)}
        footer={
          <SubmitBar
            onCancel={() => setVisible(false)}
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
          onValuesChange={(_, values) => updateAuthorizationCounts(values)}
        >
          <Space direction="vertical" size={20} className="dialog-form-stack">
            <FormSection title={t('common.basicInfo')}>
              <FormItem
                label={t('system.role.roleName')}
                field="roleName"
                rules={[{ required: true, message: t('system.role.roleName.required') }]}
              >
                <Input onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem
                label={t('system.role.roleKey')}
                field="roleKey"
                rules={[{ required: true, message: t('system.role.roleKey.required') }]}
              >
                <Input disabled={protectedRole} onPressEnter={() => form.submit()} />
              </FormItem>
              <FormItem label={t('system.role.sort')} field="sort">
                <InputNumber min={0} />
              </FormItem>
              <FormItem label={t('system.role.status')} field="status">
                <Select
                  disabled={protectedRole}
                  options={[
                    { label: t('system.user.status.enabled'), value: 1 },
                    { label: t('system.user.status.disabled'), value: 2 },
                  ]}
                />
              </FormItem>
            </FormSection>
            <FormSection
              title={t('common.accessControl')}
              description={t('system.role.accessControlDesc')}
            >
              <Alert type="info" content={t('system.role.apiPolicyHint')} />
              <Row gutter={[16, 16]}>
                <Col span={24}>
                  <Card
                    className="dialog-grid-card"
                    size="small"
                    title={authorizationTitle(
                      t('system.role.navigationAuth'),
                      authorizationCounts.navigation,
                      'arcoblue',
                    )}
                  >
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Typography.Text type="secondary">
                        {t('system.role.navigationAuthHint')}
                      </Typography.Text>
                      <FormItem label={t('system.role.menuIds')} field="menuIds">
                        <PermissionTreeSelector
                          key={`navigation-${navigationPermissionTree.expandedKeys.join('|')}`}
                          treeData={navigationPermissionTree.treeData}
                          defaultExpandedKeys={navigationPermissionTree.expandedKeys}
                          permissionKeys={permissionCatalog.navigationKeys}
                          emptyText={t('system.role.permissionTree.empty')}
                          searchPlaceholder={t('system.role.permissionTree.searchPlaceholder')}
                          expandAllText={t('system.role.permissionTree.expandAll')}
                          collapseAllText={t('system.role.permissionTree.collapseAll')}
                        />
                      </FormItem>
                    </Space>
                  </Card>
                </Col>
                <Col span={24}>
                  <Card
                    className="dialog-grid-card"
                    size="small"
                    title={authorizationTitle(
                      t('system.role.pageAuth'),
                      authorizationCounts.page,
                      'green',
                    )}
                  >
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Typography.Text type="secondary">
                        {t('system.role.pageAuthHint')}
                      </Typography.Text>
                      <FormItem
                        label={t('system.role.pagePermissionKeys')}
                        field="pagePermissionKeys"
                      >
                        <PermissionTreeSelector
                          key={`page-${pagePermissionTree.expandedKeys.join('|')}`}
                          treeData={pagePermissionTree.treeData}
                          defaultExpandedKeys={pagePermissionTree.expandedKeys}
                          permissionKeys={permissionCatalog.pageKeys}
                          emptyText={t('system.role.permissionTree.empty')}
                          searchPlaceholder={t('system.role.permissionTree.searchPlaceholder')}
                          expandAllText={t('system.role.permissionTree.expandAll')}
                          collapseAllText={t('system.role.permissionTree.collapseAll')}
                        />
                      </FormItem>
                    </Space>
                  </Card>
                </Col>
                <Col span={24}>
                  <Card
                    className="dialog-grid-card"
                    size="small"
                    title={authorizationTitle(
                      t('system.role.actionAuth'),
                      authorizationCounts.action,
                      'orange',
                    )}
                  >
                    <Space direction="vertical" size={12} style={{ width: '100%' }}>
                      <Typography.Text type="secondary">
                        {t('system.role.actionAuthHint')}
                      </Typography.Text>
                      <FormItem
                        label={t('system.role.actionPermissionKeys')}
                        field="actionPermissionKeys"
                      >
                        <PermissionTreeSelector
                          key={`action-${actionPermissionTree.expandedKeys.join('|')}`}
                          treeData={actionPermissionTree.treeData}
                          defaultExpandedKeys={actionPermissionTree.expandedKeys}
                          permissionKeys={permissionCatalog.actionKeys}
                          emptyText={t('system.role.permissionTree.empty')}
                          searchPlaceholder={t('system.role.permissionTree.searchPlaceholder')}
                          expandAllText={t('system.role.permissionTree.expandAll')}
                          collapseAllText={t('system.role.permissionTree.collapseAll')}
                        />
                      </FormItem>
                    </Space>
                  </Card>
                </Col>
                {authorizationCounts.unknown > 0 ? (
                  <Col span={24}>
                    <Card
                      className="dialog-grid-card dialog-grid-card--danger"
                      size="small"
                      title={authorizationTitle(
                        t('system.role.unknownAuth'),
                        authorizationCounts.unknown,
                        'red',
                      )}
                    >
                      <Space direction="vertical" size={12} style={{ width: '100%' }}>
                        <Typography.Text type="secondary">
                          {t('system.role.unknownAuthHint')}
                        </Typography.Text>
                        <FormItem
                          label={t('system.role.unknownPermissionKeys')}
                          field="unknownPermissionKeys"
                        >
                          <Select mode="multiple" disabled options={unknownPermissionOptions} />
                        </FormItem>
                      </Space>
                    </Card>
                  </Col>
                ) : null}
              </Row>
            </FormSection>
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
};

export default RoleList;
