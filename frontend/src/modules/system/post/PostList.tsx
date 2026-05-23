import React, { useCallback, useEffect, useMemo, useState } from 'react';
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
  Tag,
  Typography,
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
  IconSearch,
} from '@arco-design/web-react/icon';
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
import { usePermission } from '../../../hooks/usePermission';
import {
  getVisibleSelectedRowKeys,
  mergeCrossPageSelection,
} from '../../../components/table/crossPageSelection';
import { getDeptTree, type DeptNode } from '../dept/api';
import {
  batchDeletePosts,
  batchUpdatePostStatus,
  createPost,
  deletePost,
  downloadPostImportTemplate,
  exportPosts,
  getPostList,
  importPosts,
  updatePost,
  type PostListQuery,
  type PostPayload,
  type PostRow,
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
  ImportCsvButton,
  ListHeaderActions,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  PageNetworkError,
  PageServerError,
  PermissionAction,
  SubmitBar,
  SystemRowActions,
  TABLE_ACTION_COLUMN_WIDTH,
  TABLE_COLUMN_WIDTH,
  TableBatchActionBar,
  useGovernanceRail,
  withTableColumnPriority,
} from '../../../components';
import '../list-page.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

const emptyQuery: PostListQuery = {
  postCode: '',
  postName: '',
  deptId: undefined,
  status: undefined,
  page: 1,
  pageSize: 10,
};

const emptyForm: PostPayload = {
  deptId: 0,
  postCode: '',
  postName: '',
  sort: 0,
  status: 1,
  remark: '',
};

function normalizePostRow(row: PostRow): PostRow {
  return {
    ...row,
    assignedUserCount: typeof row.assignedUserCount === 'number' ? row.assignedUserCount : 0,
    governanceTags: Array.isArray(row.governanceTags) ? row.governanceTags : [],
    governanceTagLabels: Array.isArray(row.governanceTagLabels) ? row.governanceTagLabels : [],
    governanceBlockedBy: Array.isArray(row.governanceBlockedBy) ? row.governanceBlockedBy : [],
    governanceBlockedDesc: Array.isArray(row.governanceBlockedDesc)
      ? row.governanceBlockedDesc
      : [],
    governanceActions: Array.isArray(row.governanceActions) ? row.governanceActions : [],
    governanceActionLabel: Array.isArray(row.governanceActionLabel)
      ? row.governanceActionLabel
      : [],
  };
}

function isDefaultPostListQuery(query: PostListQuery) {
  return (
    !query.postCode &&
    !query.postName &&
    query.deptId === undefined &&
    query.status === undefined &&
    (query.page ?? 1) === 1 &&
    (query.pageSize ?? 10) === 10 &&
    !query.sortField &&
    !query.sortOrder
  );
}

interface LoadDataOptions {
  silent?: boolean;
}

const PostList: React.FC = () => {
  const { t } = useTranslation();
  const { isAdmin, hasPerm } = usePermission();
  const canCreate = isAdmin || hasPerm('system:post:create');
  const canEdit = isAdmin || hasPerm('system:post:update');
  const canDelete = isAdmin || hasPerm('system:post:delete');
  const canExport = isAdmin || hasPerm('system:post:export');
  const canImport = isAdmin || hasPerm('system:post:import');
  const canBatchUpdate = isAdmin || hasPerm('system:post:batch-update');
  const canBatchDelete = isAdmin || hasPerm('system:post:batch-delete');
  const governanceRail = useGovernanceRail();
  const [data, setData] = useState<PostRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<PostRow | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [deptOptions, setDeptOptions] = useState<Array<{ label: string; value: number }>>([]);
  const [query, setQuery] = useState<PostListQuery>(emptyQuery);
  const [form] = Form.useForm<PostPayload>();
  const [queryForm] = Form.useForm<PostListQuery>();
  const invalidatePostCaches = useCallback(() => {
    invalidateRouteWarmDataMany([
      { path: '/system/post', resourceKeys: ['list:default'] },
      { path: '/system/user', resourceKeys: ['posts:active'] },
      { path: '/system/dept', resourceKeys: ['posts:org-chart'] },
    ]);
  }, []);

  const loadData = useCallback(
    async (nextQuery: PostListQuery = query, options?: LoadDataOptions) => {
      const silent = options?.silent === true;
      if (!silent) {
        setLoading(true);
        setError(null);
      }
      try {
        const result = isDefaultPostListQuery(nextQuery)
          ? await resolveRouteWarmData('/system/post', 'list:default', () => getPostList(nextQuery))
          : await getPostList(nextQuery);
        setData(result.items.map(normalizePostRow));
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

  const loadDeptOptions = useCallback(async () => {
    try {
      const deptRows = await resolveRouteWarmData('/system/post', 'depts:sorted', () =>
        getDeptTree({ sortField: 'sort', sortOrder: 'asc' }),
      );
      const flattenDept = (nodes: DeptNode[], depth = 0): Array<{ label: string; value: number }> =>
        nodes.flatMap((item) => [
          { label: `${'— '.repeat(depth)}${item.deptName}`, value: item.id },
          ...(item.children?.length ? flattenDept(item.children, depth + 1) : []),
        ]);
      const selectableDeptRows = deptRows.flatMap((item) =>
        item.isRoot ? item.children || [] : [item],
      );
      setDeptOptions(flattenDept(selectableDeptRows));
    } catch {
      message.error(t('common.loadFailed'));
    }
  }, [t]);

  useEffect(() => {
    const timer = window.setTimeout(() => void loadData(query), 0);
    return () => window.clearTimeout(timer);
  }, [loadData, query]);

  useEffect(() => {
    const timer = window.setTimeout(() => void loadDeptOptions(), 0);
    return () => window.clearTimeout(timer);
  }, [loadDeptOptions]);

  useRefreshSubscription(
    ['system:dept:changed', 'system:post:changed', 'system:user:changed'],
    (payload) => {
      if (payload.source === 'system/post') {
        return;
      }
      void loadData(query);
      void loadDeptOptions();
    },
  );

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue(emptyForm);
    setVisible(true);
  };

  const openEdit = (row: PostRow) => {
    setEditing(row);
    form.setFieldsValue({
      deptId: row.deptId,
      postCode: row.postCode,
      postName: row.postName,
      sort: row.sort,
      status: row.status,
      remark: row.remark,
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
        await updatePost(editing.id, values);
        message.success(t('common.updateSuccess'));
      } else {
        await createPost(values);
        message.success(t('common.createSuccess'));
      }
      invalidatePostCaches();
      publishRefresh('system:post:changed', 'system/post');
      setVisible(false);
      await loadData(query, { silent: true });
    } catch {
      message.error(t('common.actionFailed'));
    } finally {
      setSubmitting(false);
    }
  };

  const removePost = async (id: number) => {
    await deletePost(id);
    message.success(t('common.deleteSuccess'));
    invalidatePostCaches();
    publishRefresh('system:post:changed', 'system/post');
    setSelectedRowKeys((keys) => keys.filter((key) => Number(key) !== id));
    const nextPage =
      data.length === 1 && (query.page || 1) > 1 ? (query.page || 1) - 1 : query.page || 1;
    setQuery({ ...query, page: nextPage });
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

  const toArcoSortOrder = (sortOrder?: PostListQuery['sortOrder']) => {
    if (sortOrder === 'asc') {
      return 'ascend';
    }
    if (sortOrder === 'desc') {
      return 'descend';
    }
    return undefined;
  };

  const sortableColumn = (
    field: NonNullable<PostListQuery['sortField']>,
  ): Partial<ColumnProps<PostRow>> => ({
    sorter: true,
    sortOrder: query.sortField === field ? toArcoSortOrder(query.sortOrder) : undefined,
  });

  const handleTableChange: TableProps<PostRow>['onChange'] = (pagination, sorter) => {
    const currentSorter = Array.isArray(sorter) ? sorter[0] : (sorter as SorterInfo | undefined);
    const nextQuery: PostListQuery = {
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
    await exportPosts(query);
  };

  const handleDownloadTemplate = async () => {
    await downloadPostImportTemplate();
  };

  const handleImport = async (file: File) => {
    const result = await importPosts(file);
    showImportResult(result, t);
    if (result.applied) {
      invalidatePostCaches();
      publishRefresh('system:post:changed', 'system/post');
      await loadData(query, { silent: true });
    }
  };

  const handleBatchStatus = async (status: 1 | 2) => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const postIds = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchUpdatePostStatus({ postIds, status });
    message.success(t('system.post.batchStatusSuccess', { count: result.updatedCount }));
    invalidatePostCaches();
    publishRefresh('system:post:changed', 'system/post');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('common.batchSelectionRequired'));
      return;
    }
    const ids = selectedRowKeys.map((item) => Number(item)).filter((item) => item > 0);
    const result = await batchDeletePosts({ ids });
    const messageKey =
      result.failedCount > 0 ? 'common.batchDeletePartialSuccess' : 'common.batchDeleteSuccess';
    message[result.failedCount > 0 ? 'warning' : 'success'](
      t(messageKey, { deleted: result.deletedCount, failed: result.failedCount }),
    );
    invalidatePostCaches();
    publishRefresh('system:post:changed', 'system/post');
    setSelectedRowKeys([]);
    await loadData(query, { silent: true });
  };

  const visibleSelectedRowKeys = useMemo(() => {
    return getVisibleSelectedRowKeys(selectedRowKeys, data.map((item) => item.id));
  }, [data, selectedRowKeys]);

  const heroStats = useMemo(() => {
    const inUseCount = data.filter((item) => item.assignedUserCount > 0).length;
    const disabledCount = data.filter((item) => item.status === 2).length;
    const assignedUsers = data.reduce((sum, item) => sum + item.assignedUserCount, 0);
    return [
      {
        key: 'total',
        label: t('common.total', { count: total }),
        value: total,
        hint: t('system.post.hero.totalHint'),
      },
      {
        key: 'inUse',
        label: t('system.post.hero.inUse'),
        value: inUseCount,
        hint: t('system.post.hero.inUseHint'),
      },
      {
        key: 'assignedUsers',
        label: t('system.post.hero.assignedUsers'),
        value: assignedUsers,
        hint: t('system.post.hero.assignedUsersHint'),
      },
      {
        key: 'disabled',
        label: t('system.user.status.disabled'),
        value: disabledCount,
        hint: t('system.post.hero.disabledHint'),
      },
    ];
  }, [data, t, total]);

  const governanceSummary = useMemo(
    () => ({
      inUse: data.filter((item) => item.assignedUserCount > 0).length,
      disabled: data.filter((item) => item.status === 2).length,
      clean: data.filter((item) => item.governanceTags.includes('clean')).length,
    }),
    [data],
  );

  const renderGovernanceTags = (labels: string[], tone: 'tag' | 'note' = 'tag') => {
    if (labels.length === 0) {
      return '-';
    }
    if (tone === 'note') {
      return (
        <Space size={[6, 6]} wrap>
          {labels.map((label) => (
            <Tag key={label} size="small">
              {label}
            </Tag>
          ))}
        </Space>
      );
    }
    return (
      <Space size={[6, 6]} wrap>
        {labels.map((label) => (
          <Tag key={label} size="small" color="arcoblue">
            {label}
          </Tag>
        ))}
      </Space>
    );
  };

  const columns: ColumnProps<PostRow>[] = [
    withTableColumnPriority(
      { title: t('system.post.dept'), dataIndex: 'deptName', width: TABLE_COLUMN_WIDTH.name },
      'medium',
    ),
    {
      title: t('system.post.postCode'),
      dataIndex: 'postCode',
      width: TABLE_COLUMN_WIDTH.code,
      ...sortableColumn('postCode'),
    },
    {
      title: t('system.post.postName'),
      dataIndex: 'postName',
      width: TABLE_COLUMN_WIDTH.name,
      ...sortableColumn('postName'),
    },
    withTableColumnPriority(
      {
        title: t('system.post.hero.assignedUsers'),
        dataIndex: 'assignedUserCount',
        width: TABLE_COLUMN_WIDTH.count,
        render: (value: number) => <span>{value}</span>,
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.dept.governance'),
        dataIndex: 'governanceTagLabels',
        width: TABLE_COLUMN_WIDTH.tagGroup,
        render: (_: unknown, row: PostRow) => renderGovernanceTags(row.governanceTagLabels),
      },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.post.hero.blockedBy'),
        dataIndex: 'governanceBlockedDesc',
        width: TABLE_COLUMN_WIDTH.diagnostics,
        render: (_: unknown, row: PostRow) =>
          renderGovernanceTags(row.governanceBlockedDesc, 'note'),
      },
      'medium',
    ),
    withTableColumnPriority(
      {
        title: t('system.post.hero.nextAction'),
        dataIndex: 'governanceActionLabel',
        width: TABLE_COLUMN_WIDTH.tagGroup,
        render: (_: unknown, row: PostRow) => (
          <Typography.Text className="post-governance__action-text">
            {row.governanceActionLabel.join(' / ') || '-'}
          </Typography.Text>
        ),
      },
      'low',
    ),
    {
      title: t('system.post.status'),
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
        title: t('system.post.sort'),
        dataIndex: 'sort',
        width: TABLE_COLUMN_WIDTH.count,
        ...sortableColumn('sort'),
      },
      'medium',
    ),
    withTableColumnPriority(
      { title: t('system.post.remark'), dataIndex: 'remark', width: TABLE_COLUMN_WIDTH.name },
      'low',
    ),
    withTableColumnPriority(
      {
        title: t('system.post.createdAt'),
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
      render: (_: unknown, row: PostRow) => (
        <SystemRowActions
          className="post-list-page__row-actions"
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
              status: 'danger',
              confirm: {
                title: t('system.post.deleteConfirm'),
                onOk: () => removePost(row.id),
              },
            },
          ]}
        />
      ),
    },
  ];

  const batchActionDisabled = !canBatchUpdate || selectedRowKeys.length === 0;
  const batchDeleteDisabled = !canBatchDelete || selectedRowKeys.length === 0;
  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('system.post.hero.inUse'),
        value: governanceSummary.inUse,
        description: t('system.post.hero.inUseHint'),
      },
      {
        label: t('system.user.status.disabled'),
        value: governanceSummary.disabled,
        description: t('system.post.hero.disabledHint'),
      },
      {
        label: t('system.dept.governance.clean'),
        value: governanceSummary.clean,
        description: t('system.post.hero.cleanHint'),
      },
    ],
    [governanceSummary.clean, governanceSummary.disabled, governanceSummary.inUse, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template post-list-page">
        <GovernanceSummaryBar
          eyebrow={t('system.post.header.eyebrow')}
          title={t('system.post.header.title')}
          description={t('system.post.hero.desc')}
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
              {t('system.post.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <>
          <FilterPanel>
            <Form form={queryForm} layout="vertical" onSubmit={() => search()}>
              <Row gutter={16}>
                <Col xs={24} md={12} lg={5}>
                  <FormItem label={t('system.post.postCode')} field="postCode">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={5}>
                  <FormItem label={t('system.post.postName')} field="postName">
                    <Input onPressEnter={() => queryForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={5}>
                  <FormItem label={t('system.post.dept')} field="deptId">
                    <Select allowClear options={deptOptions} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12} lg={5}>
                  <FormItem label={t('system.post.status')} field="status">
                    <Select
                      allowClear
                      options={[
                        { label: t('system.user.status.enabled'), value: 1 },
                        { label: t('system.user.status.disabled'), value: 2 },
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
                      title={t('system.post.batchEnableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(1);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button disabled={batchActionDisabled}>{t('system.post.batchEnable')}</Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchUpdate}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.post.batchDisableConfirm')}
                      onOk={() => {
                        void handleBatchStatus(2);
                      }}
                      disabled={batchActionDisabled}
                    >
                      <Button
                        status={batchActionDisabled ? undefined : 'warning'}
                        disabled={batchActionDisabled}
                      >
                        {t('system.post.batchDisable')}
                      </Button>
                    </Popconfirm>
                  </PermissionAction>
                  <PermissionAction
                    allowed={canBatchDelete}
                    tooltip={t('common.noPermissionAction')}
                  >
                    <Popconfirm
                      title={t('system.post.batchDeleteConfirm')}
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
              <PageEmpty description={t('system.post.empty')} />
            ) : null}
            {!loading && !(error && data.length === 0) && data.length > 0 ? (
              <AppTable<PostRow>
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
                  onChange: (rowKeys) =>
                    setSelectedRowKeys((keys) =>
                      mergeCrossPageSelection(keys, rowKeys, data.map((item) => item.id)),
                    ),
                }}
                onChange={handleTableChange}
                emptyText={t('system.post.empty')}
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
        title={t('system.post.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('system.post.hero.summaryTitle')}
        noteDescription={t('system.post.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>

      <AppModal
        title={editing ? t('system.post.edit') : t('system.post.create')}
        visible={visible}
        size="md"
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
                    <Select options={deptOptions} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.post.postCode')}
                    field="postCode"
                    rules={[{ required: true, message: t('system.post.postCodeRequired') }]}
                  >
                    <Input disabled={Boolean(editing)} onPressEnter={() => form.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.post.postName')}
                    field="postName"
                    rules={[{ required: true, message: t('system.post.postNameRequired') }]}
                  >
                    <Input onPressEnter={() => form.submit()} />
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
    </PageContainer>
  );
};

export default PostList;
