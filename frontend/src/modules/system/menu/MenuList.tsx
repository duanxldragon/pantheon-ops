import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AutoComplete,
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
  TreeSelect,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import {
  IconApps,
  IconDelete,
  IconEdit,
  IconList,
  IconPlus,
  IconSearch,
  IconUnorderedList,
} from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import {
  isNetworkRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../../api/request';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import { publishRefresh, useRefreshSubscription } from '../../../core/refresh/refreshBus';
import { invalidateRouteWarmDataMany } from '../../../core/router/prefetch';
import { usePermission } from '../../../hooks/usePermission';
import type {
  ColumnProps,
  SorterInfo,
  TableProps,
} from '@arco-design/web-react/es/Table/interface';
import type { TreeSelectDataType } from '@arco-design/web-react/es/TreeSelect/interface';
import {
  createMenu,
  deleteMenu,
  getMenuTree,
  updateMenu,
  type MenuListQuery,
  type MenuNode,
  type MenuPayload,
} from './api';
import { useMenuStore } from '../../../store/useMenuStore';
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
  PageActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  SubmitBar,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import { MENU_ICON_OPTIONS } from '../../../core/menu/icon';
import {
  isRegisteredComponentKey,
  listRegisteredComponentKeys,
} from '../../../core/router/componentRegistry';
import '../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;
const registeredComponentKeys = listRegisteredComponentKeys();

type MenuViewMode = 'table' | 'list' | 'card';

interface FlatMenuNode {
  node: MenuNode;
  depth: number;
}

type MenuFormValues = Omit<MenuPayload, 'parentId'> & {
  parentId: string;
};

const emptyForm: MenuFormValues = {
  parentId: '0',
  titleKey: '',
  path: '',
  component: '',
  pagePerm: '',
  perms: '',
  type: 'C',
  icon: 'menu',
  routeName: '',
  module: 'system',
  sort: 0,
  isVisible: 1,
  isCache: 0,
  isExternal: 0,
  activeMenu: '',
};

const emptyQuery: MenuListQuery = {
  titleKey: '',
  path: '',
  isVisible: undefined,
  sortField: 'sort',
  sortOrder: 'asc',
};

interface LoadDataOptions {
  silent?: boolean;
}

const MenuList: React.FC = () => {
  const [data, setData] = useState<MenuNode[]>([]);
  const [parentTree, setParentTree] = useState<MenuNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<MenuNode | null>(null);
  const [viewMode, setViewMode] = useState<MenuViewMode>('table');
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 10 });
  const [query, setQuery] = useState<MenuListQuery>(emptyQuery);
  const [form] = Form.useForm<MenuFormValues>();
  const [queryForm] = Form.useForm<MenuListQuery>();
  const { fetchMenuTree } = useMenuStore();
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:menu:create');
  const canEdit = isAdmin || hasPerm('system:menu:update');
  const canDelete = isAdmin || hasPerm('system:menu:delete');
  const governanceRail = useGovernanceRail();
  const invalidateMenuCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/menu', resourceKeys: ['tree:manage'] },
      { path: '/system/role', resourceKeys: ['menus:manage'] },
      { path: '/system/permission', resourceKeys: ['workbench:default'] },
    ]);
  }, []);

  const flattenedMenus = useMemo<FlatMenuNode[]>(() => {
    const rows: FlatMenuNode[] = [];
    const walk = (nodes: MenuNode[], depth: number) => {
      nodes.forEach((node) => {
        rows.push({ node, depth });
        if (node.children?.length) {
          walk(node.children, depth + 1);
        }
      });
    };
    walk(data, 0);
    return rows;
  }, [data]);

  const loadData = useCallback(
    async (nextQuery: MenuListQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      try {
        const rows = await getMenuTree({ ...nextQuery, scope: 'manage' });
        setData(rows);
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

  const loadParentTree = useCallback(async () => {
    const rows = await getMenuTree({
      scope: 'manage',
      sortField: emptyQuery.sortField,
      sortOrder: emptyQuery.sortOrder,
    });
    setParentTree(rows);
  }, []);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      loadData(query);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      loadParentTree().catch(() => undefined);
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadParentTree]);

  const tableTotalPages = useMemo(
    () => Math.max(1, Math.ceil(data.length / tablePagination.pageSize)),
    [data.length, tablePagination.pageSize],
  );
  const tableCurrentPage = useMemo(
    () => Math.min(tablePagination.current, tableTotalPages),
    [tablePagination.current, tableTotalPages],
  );

  useRefreshSubscription('system:menu:changed', (payload) => {
    if (payload.source === 'system/menu') {
      return;
    }
    loadData(query);
    loadParentTree().catch(() => undefined);
    fetchMenuTree({ force: true });
  });

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(emptyForm);
    setVisible(true);
  };

  const openCreateChild = (row: MenuNode) => {
    setEditing(null);
    form.setFieldsValue({
      ...emptyForm,
      parentId: String(row.id),
      module: row.module || emptyForm.module,
      isVisible: row.isVisible,
    });
    setVisible(true);
  };

  const openEdit = (row: MenuNode) => {
    setEditing(row);
    form.setFieldsValue({
      parentId: String(row.parentId),
      titleKey: row.titleKey,
      path: row.path,
      component: row.component,
      pagePerm: row.pagePerm,
      perms: row.perms,
      type: row.type,
      icon: row.icon,
      routeName: row.routeName,
      module: row.module,
      sort: row.sort,
      isVisible: row.isVisible,
      isCache: row.isCache,
      isExternal: row.isExternal,
      activeMenu: row.activeMenu,
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
      const normalizedValues: MenuPayload = {
        ...values,
        parentId: Number(values.parentId || 0),
      };
      if (editing) {
        await updateMenu(editing.id, normalizedValues);
        message.success(t('common.updateSuccess'));
      } else {
        await createMenu(normalizedValues);
        message.success(t('common.createSuccess'));
      }
      invalidateMenuCaches();
      publishRefresh('system:menu:changed', 'system/menu');
      setVisible(false);
      await Promise.all([
        loadData(query, { silent: true }),
        loadParentTree().catch(() => undefined),
        fetchMenuTree({ force: true }),
      ]);
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const removeMenu = async (id: number) => {
    await deleteMenu(id);
    message.success(t('common.deleteSuccess'));
    invalidateMenuCaches();
    publishRefresh('system:menu:changed', 'system/menu');
    await Promise.all([
      loadData(query, { silent: true }),
      loadParentTree().catch(() => undefined),
      fetchMenuTree({ force: true }),
    ]);
  };

  const search = () => {
    const values = queryForm.getFieldsValue();
    setTablePagination((current) => ({ ...current, current: 1 }));
    setQuery({
      ...query,
      ...values,
    });
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setTablePagination((current) => ({ ...current, current: 1 }));
    setQuery(emptyQuery);
  };

  const toArcoSortOrder = (sortOrder?: MenuListQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<MenuListQuery['sortField']>,
  ): Partial<ColumnProps<MenuNode>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<MenuNode>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    setTablePagination({
      current: pagination.current || 1,
      pageSize: pagination.pageSize || tablePagination.pageSize,
    });
    const nextQuery: MenuListQuery = {
      ...query,
      sortField: currentSorter?.direction ? String(currentSorter.field) : emptyQuery.sortField,
      sortOrder: currentSorter?.direction === 'descend' ? 'desc' : 'asc',
    };
    setQuery(nextQuery);
  };

  const getMenuTypeLabel = (value: string) => {
    const mapping: Record<string, string> = {
      M: t('system.menu.type.menuGroup'),
      C: t('system.menu.type.menu'),
      F: t('system.menu.type.button'),
    };
    return mapping[value] ?? value;
  };

  const getMenuTypeColor = (value: string) => {
    const mapping: Record<string, string> = {
      M: 'arcoblue',
      C: 'green',
      F: 'purple',
    };
    return mapping[value] ?? 'gray';
  };

  const renderTypeTag = (row: MenuNode) => (
    <Tag color={getMenuTypeColor(row.type)}>{getMenuTypeLabel(row.type)}</Tag>
  );

  const renderVisibleTag = (value: number) => (
    <Tag color={value === 1 ? 'green' : 'gray'}>
      {value === 1 ? t('common.yes') : t('common.no')}
    </Tag>
  );

  const renderMetadataTags = (row: MenuNode) => (
    <Space wrap size={4}>
      <Tag>{row.module || 'system'}</Tag>
      {row.icon ? <Tag color="arcoblue">{row.icon}</Tag> : null}
      {row.isCache === 1 ? <Tag color="green">{t('system.menu.cache')}</Tag> : null}
      {row.isExternal === 1 ? <Tag color="orange">{t('system.menu.external')}</Tag> : null}
      {row.activeMenu ? <Tag color="purple">{row.activeMenu}</Tag> : null}
    </Space>
  );

  const renderPermissionText = (row: MenuNode) => {
    if (row.pagePerm) {
      return row.pagePerm;
    }
    if (row.perms) {
      return row.perms;
    }
    return '-';
  };

  const excludedParentIDs = useMemo(() => {
    if (!editing) {
      return new Set<number>();
    }
    const blocked = new Set<number>([editing.id]);
    const collect = (nodes: MenuNode[]) => {
      nodes.forEach((node) => {
        blocked.add(node.id);
        if (node.children?.length) {
          collect(node.children);
        }
      });
    };
    collect(editing.children || []);
    return blocked;
  }, [editing]);

  const parentOptions = useMemo<TreeSelectDataType[]>(() => {
    const build = (nodes: MenuNode[]): TreeSelectDataType[] =>
      nodes
        .filter((node) => !excludedParentIDs.has(node.id))
        .map((node) => {
          const translatedTitle = t(node.titleKey, { defaultValue: node.titleKey });
          return {
            title: node.path ? `${translatedTitle} · ${node.path}` : translatedTitle,
            key: String(node.id),
            value: String(node.id),
            children: node.children?.length ? build(node.children) : undefined,
          };
        });
    return [
      {
        title: t('system.menu.root'),
        key: '0',
        value: '0',
      },
      ...build(parentTree),
    ];
  }, [excludedParentIDs, parentTree, t]);

  const renderMenuActions = (row: MenuNode) => (
    <Space size={4} className="system-list__actions menu-list-page__row-actions">
      {canCreate && row.type !== 'F' ? (
        <Button
          type="text"
          size="small"
          icon={<IconPlus />}
          onClick={() => openCreateChild(row)}
        >
          {t('system.menu.createChild')}
        </Button>
      ) : null}
      {canEdit ? (
        <Button type="text" size="small" icon={<IconEdit />} onClick={() => openEdit(row)}>
          {t('common.edit')}
        </Button>
      ) : null}
      {canDelete ? (
        <Popconfirm title={t('common.deleteConfirm')} onOk={() => removeMenu(row.id)}>
          <Button type="text" size="small" status="danger" icon={<IconDelete />}>
            {t('common.delete')}
          </Button>
        </Popconfirm>
      ) : null}
    </Space>
  );

  const renderListView = () => (
    <div className="menu-list-view">
      {flattenedMenus.map(({ node, depth }) => (
        <div
          key={node.id}
          className="menu-list-view__item"
          style={{ marginLeft: Math.min(depth * 18, 72) }}
        >
          <div className="menu-list-view__content">
            <div className="menu-list-view__title-row">
              <Typography.Text className="menu-list-view__title">
                {t(node.titleKey)}
              </Typography.Text>
              {renderTypeTag(node)}
              {renderVisibleTag(node.isVisible)}
            </div>
            <div className="menu-list-view__meta">
              <Typography.Text type="secondary">{node.path || '-'}</Typography.Text>
              <Typography.Text type="secondary">{node.routeName || '-'}</Typography.Text>
              {node.component ? (
                <Typography.Text type="secondary">{node.component}</Typography.Text>
              ) : null}
            </div>
            <div className="menu-list-view__meta">
              <Typography.Text type="secondary">{renderPermissionText(node)}</Typography.Text>
            </div>
            {renderMetadataTags(node)}
          </div>
          <div className="menu-list-view__actions">{renderMenuActions(node)}</div>
        </div>
      ))}
    </div>
  );

  const renderCardView = () => (
    <div className="menu-card-view">
      {flattenedMenus.map(({ node, depth }) => (
        <Card key={node.id} className="menu-card-view__card" bordered={false}>
          <div className="menu-card-view__header">
            <div>
              <Typography.Text className="menu-card-view__title">
                {t(node.titleKey)}
              </Typography.Text>
              <div className="menu-card-view__subtitle">{node.titleKey}</div>
            </div>
            <Tag color={depth === 0 ? 'arcoblue' : 'gray'}>
              {t('system.menu.level', { level: depth + 1 })}
            </Tag>
          </div>
          <Space wrap size={4}>
            {renderTypeTag(node)}
            {renderVisibleTag(node.isVisible)}
          </Space>
          <div className="menu-card-view__fields">
            <div>
              <span>{t('system.menu.path')}</span>
              <Typography.Text ellipsis={{ showTooltip: true }}>{node.path || '-'}</Typography.Text>
            </div>
            <div>
              <span>{t('system.menu.routeName')}</span>
              <Typography.Text ellipsis={{ showTooltip: true }}>
                {node.routeName || '-'}
              </Typography.Text>
            </div>
            <div>
              <span>{t('system.menu.component')}</span>
              <Typography.Text ellipsis={{ showTooltip: true }}>
                {node.component || '-'}
              </Typography.Text>
            </div>
            <div>
              <span>{t('system.menu.pagePerm')}</span>
              <Typography.Text ellipsis={{ showTooltip: true }}>
                {renderPermissionText(node)}
              </Typography.Text>
            </div>
          </div>
          <div className="menu-card-view__footer">
            {renderMetadataTags(node)}
            {renderMenuActions(node)}
          </div>
        </Card>
      ))}
    </div>
  );

  const columns: ColumnProps<MenuNode>[] = [
    {
      title: t('system.menu.title'),
      dataIndex: 'titleKey',
      width: TABLE_COLUMN_WIDTH.tagGroup,
      render: (value: string) => (
        <Typography.Text ellipsis={{ showTooltip: true }}>{t(value)}</Typography.Text>
      ),
      ...sortableColumn('titleKey'),
    },
    {
      title: t('system.menu.path'),
      dataIndex: 'path',
      width: TABLE_COLUMN_WIDTH.routePath,
      ...sortableColumn('path'),
      render: (value: string) => (
        <Typography.Text ellipsis={{ showTooltip: true }}>{value || '-'}</Typography.Text>
      ),
    },
    withTableColumnPriority(
      {
        title: t('system.menu.routeName'),
        dataIndex: 'routeName',
        width: TABLE_COLUMN_WIDTH.routePath,
        ...sortableColumn('routeName'),
        render: (value: string, row: MenuNode) => (
          <Space direction="vertical" size={2}>
            <Typography.Text ellipsis={{ showTooltip: true }}>{value || '-'}</Typography.Text>
            {row.component ? (
              <Typography.Text
                type="secondary"
                className="text-sm"
                ellipsis={{ showTooltip: true }}
              >
                {row.component}
              </Typography.Text>
            ) : null}
          </Space>
        ),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.menu.pagePerm'),
        dataIndex: 'pagePerm',
        width: TABLE_COLUMN_WIDTH.tagGroup,
        ...sortableColumn('pagePerm'),
        render: (value: string) => (
          <Typography.Text ellipsis={{ showTooltip: true }}>{value || '-'}</Typography.Text>
        ),
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.menu.perms'),
        dataIndex: 'perms',
        width: TABLE_COLUMN_WIDTH.tagGroup,
        ...sortableColumn('perms'),
        render: (value: string, row: MenuNode) => {
          if (value) {
            return <Typography.Text ellipsis={{ showTooltip: true }}>{value}</Typography.Text>;
          }
          if (row.type === 'C') {
            return (
              <Typography.Text type="secondary">{t('system.menu.perms.fromPage')}</Typography.Text>
            );
          }
          return <Typography.Text type="secondary">-</Typography.Text>;
        },
      },
      'low',
    ),
    {
      title: t('system.menu.type'),
      dataIndex: 'type',
      width: TABLE_COLUMN_WIDTH.status,
      ...sortableColumn('type'),
      render: (value: string) => {
        return getMenuTypeLabel(value);
      },
    },
    withTableColumnPriority(
      {
        title: t('system.menu.sort'),
        dataIndex: 'sort',
        width: TABLE_COLUMN_WIDTH.count,
        ...sortableColumn('sort'),
      },
      'medium',
    ),
    {
      title: t('system.menu.visible'),
      dataIndex: 'isVisible',
      width: TABLE_COLUMN_WIDTH.status,
      ...sortableColumn('isVisible'),
      render: renderVisibleTag,
    },
    withTableColumnPriority(
      {
        title: t('system.menu.metadata'),
        width: TABLE_COLUMN_WIDTH.diagnostics,
        render: (_: unknown, row: MenuNode) => renderMetadataTags(row),
      },
      'low',
    ),
    {
      title: t('common.action'),
      width: TABLE_ACTION_COLUMN_WIDTH.wide,
      fixed: 'right',
      render: (_: unknown, row: MenuNode) => renderMenuActions(row),
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

  const totalMenus = flattenedMenus.length;
  const visibleMenus = useMemo(
    () => flattenedMenus.filter(({ node }) => node.isVisible === 1).length,
    [flattenedMenus],
  );
  const actionMenus = useMemo(
    () => flattenedMenus.filter(({ node }) => node.type === 'F').length,
    [flattenedMenus],
  );
  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('common.total', { count: totalMenus }),
        value: totalMenus,
        hint: t('system.menu.hero.totalHint'),
      },
      {
        key: 'visible',
        label: t('system.menu.hero.visibleNodes'),
        value: visibleMenus,
        hint: t('system.menu.hero.visibleHint'),
      },
      {
        key: 'actions',
        label: t('system.menu.hero.actionNodes'),
        value: actionMenus,
        hint: t('system.menu.hero.actionHint'),
      },
      {
        key: 'view',
        label: t('system.menu.hero.currentView'),
        value: t(`system.menu.view.${viewMode}`),
        hint: t('system.menu.hero.viewHint'),
      },
    ],
    [actionMenus, t, totalMenus, viewMode, visibleMenus],
  );
  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.menu.hero.routeReady'),
        value: flattenedMenus.filter(({ node }) => Boolean(node.routeName)).length,
        description: t('system.menu.hero.routeHint'),
      },
      {
        label: t('system.menu.hero.cachedNodes'),
        value: flattenedMenus.filter(({ node }) => node.isCache === 1).length,
        description: t('system.menu.hero.cacheHint'),
      },
      {
        label: t('system.menu.hero.externalNodes'),
        value: flattenedMenus.filter(({ node }) => node.isExternal === 1).length,
        description: t('system.menu.hero.externalHint'),
      },
    ],
    [flattenedMenus, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('system.menu.hero.eyebrow')}
          title={t('system.menu.hero.title')}
          description={t('system.menu.hero.desc')}
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
              {t('system.menu.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16}>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('system.menu.titleKey')} field="titleKey">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={8}>
                  <FormItem label={t('system.menu.path')} field="path">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={4}>
                  <FormItem label={t('system.menu.visible')} field="isVisible">
                    <Select
                      allowClear
                      options={[
                        { label: t('common.yes'), value: 1 },
                        { label: t('common.no'), value: 0 },
                      ]}
                    />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={4}>
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
            <div className="system-list__work-actions">
              <PageActions>
                <Space size={4} className="menu-view-switcher">
                  <Button
                    size="small"
                    type={viewMode === 'table' ? 'primary' : 'secondary'}
                    icon={<IconList />}
                    onClick={() => setViewMode('table')}
                  >
                    {t('system.menu.view.table')}
                  </Button>
                  <Button
                    size="small"
                    type={viewMode === 'list' ? 'primary' : 'secondary'}
                    icon={<IconUnorderedList />}
                    onClick={() => setViewMode('list')}
                  >
                    {t('system.menu.view.list')}
                  </Button>
                  <Button
                    size="small"
                    type={viewMode === 'card' ? 'primary' : 'secondary'}
                    icon={<IconApps />}
                    onClick={() => setViewMode('card')}
                  >
                    {t('system.menu.view.card')}
                  </Button>
                </Space>
                <Button type="primary" icon={<IconPlus />} onClick={openCreate} disabled={!canCreate}>
                  {t('common.add')}
                </Button>
              </PageActions>
            </div>
            {loading && data.length === 0 ? <PageLoading /> : null}
            {error && data.length === 0 ? renderErrorState() : null}
            {!loading && !error && data.length === 0 ? (
              <PageEmpty description={t('common.noData')} />
            ) : null}
            {!loading &&
            !(error && data.length === 0) &&
            data.length > 0 &&
            viewMode === 'table' ? (
              <AppTable<MenuNode>
                className="system-list__table"
                data={data}
                columns={columns}
                rowKey="id"
                loading={loading}
                scroll={{ x: 'max-content' }}
                onChange={handleTableChange}
                emptyText={t('common.noData')}
                pagination={buildStandardPagination(t, {
                  current: tableCurrentPage,
                  pageSize: tablePagination.pageSize,
                  total: data.length,
                })}
              />
            ) : null}
            {!loading && !(error && data.length === 0) && data.length > 0 && viewMode === 'list'
              ? renderListView()
              : null}
            {!loading && !(error && data.length === 0) && data.length > 0 && viewMode === 'card'
              ? renderCardView()
              : null}
          </Card>
        </>
      </Space>

      <GovernanceInsightDrawer
        title={t('system.menu.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.menu.hero.summaryTitle')}
        noteDescription={t('system.menu.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.menu.edit') : t('system.menu.create')}
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
              <FormItem label={t('system.menu.parentId')} field="parentId">
                <TreeSelect
                  allowClear
                  showSearch
                  treeData={parentOptions}
                  placeholder={t('system.menu.parentId')}
                />
              </FormItem>
              <FormItem
                label={t('system.menu.titleKey')}
                field="titleKey"
                rules={[{ required: true, message: t('system.menu.titleRequired') }]}
              >
                <Input
                  placeholder={t('system.menu.titleKey.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem label={t('system.menu.path')} field="path">
                <Input
                  placeholder={t('system.menu.path.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem
                label={t('system.menu.component')}
                field="component"
                rules={[
                  {
                    validator: (value, callback) => {
                      const type = form.getFieldValue('type');
                      const isExternal = form.getFieldValue('isExternal');
                      const moduleName = String(form.getFieldValue('module') || '').trim();
                      const componentKey = String(value || '').trim();
                      if (type === 'C' && isExternal !== 1 && !componentKey) {
                        callback(t('system.menu.componentRequired'));
                        return;
                      }
                      const requiresRegisteredComponent =
                        moduleName === 'platform' ||
                        moduleName.startsWith('system.') ||
                        moduleName.startsWith('business.');
                      if (
                        type === 'C' &&
                        isExternal !== 1 &&
                        requiresRegisteredComponent &&
                        !isRegisteredComponentKey(componentKey)
                      ) {
                        callback(t('system.menu.componentInvalid'));
                        return;
                      }
                      callback();
                    },
                  },
                ]}
              >
                <AutoComplete
                  allowClear
                  data={registeredComponentKeys}
                  placeholder={t('system.menu.component.placeholder')}
                  filterOption
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem
                label={t('system.menu.routeName')}
                field="routeName"
                rules={[
                  {
                    validator: (value, callback) => {
                      const type = form.getFieldValue('type');
                      if (type === 'C' && !String(value || '').trim()) {
                        callback(t('system.menu.routeNameRequired'));
                        return;
                      }
                      callback();
                    },
                  },
                ]}
              >
                <Input
                  placeholder={t('system.menu.routeName.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem label={t('system.menu.module')} field="module">
                <Input
                  placeholder={t('system.menu.module.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem
                label={t('system.menu.pagePerm')}
                field="pagePerm"
                rules={[
                  {
                    validator: (value, callback) => {
                      const type = form.getFieldValue('type');
                      const isExternal = form.getFieldValue('isExternal');
                      if (type === 'C' && isExternal !== 1 && !String(value || '').trim()) {
                        callback(t('system.menu.pagePermRequired'));
                        return;
                      }
                      callback();
                    },
                  },
                ]}
              >
                <Input
                  placeholder={t('system.menu.pagePerm.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem
                label={t('system.menu.perms')}
                field="perms"
                rules={[
                  {
                    validator: (value, callback) => {
                      const type = form.getFieldValue('type');
                      if (type === 'F' && !String(value || '').trim()) {
                        callback(t('system.menu.permsRequired'));
                        return;
                      }
                      callback();
                    },
                  },
                ]}
              >
                <Input
                  placeholder={t('system.menu.perms.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
              <FormItem label={t('system.menu.type')} field="type">
                <Select
                  options={[
                    { label: t('system.menu.type.menuGroup'), value: 'M' },
                    { label: t('system.menu.type.menu'), value: 'C' },
                    { label: t('system.menu.type.button'), value: 'F' },
                  ]}
                />
              </FormItem>
              <FormItem label={t('system.menu.icon')} field="icon">
                <Select
                  allowClear
                  placeholder={t('system.menu.icon.placeholder')}
                  options={MENU_ICON_OPTIONS.map((item) => ({
                    label: t(item.labelKey),
                    value: item.value,
                  }))}
                />
              </FormItem>
              <FormItem label={t('system.menu.sort')} field="sort">
                <InputNumber min={0} />
              </FormItem>
              <FormItem label={t('system.menu.visible')} field="isVisible">
                <Select
                  options={[
                    { label: t('common.yes'), value: 1 },
                    { label: t('common.no'), value: 0 },
                  ]}
                />
              </FormItem>
              <FormItem label={t('system.menu.cache')} field="isCache">
                <Select
                  options={[
                    { label: t('common.yes'), value: 1 },
                    { label: t('common.no'), value: 0 },
                  ]}
                />
              </FormItem>
              <FormItem label={t('system.menu.external')} field="isExternal">
                <Select
                  options={[
                    { label: t('common.yes'), value: 1 },
                    { label: t('common.no'), value: 0 },
                  ]}
                />
              </FormItem>
              <FormItem label={t('system.menu.activeMenu')} field="activeMenu">
                <Input
                  placeholder={t('system.menu.activeMenu.placeholder')}
                  onPressEnter={() => form.submit()}
                />
              </FormItem>
            </FormSection>
          </Space>
        </Form>
      </AppModal>
    </PageContainer>
  );
};

export default MenuList;
