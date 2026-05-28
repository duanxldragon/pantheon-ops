import { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Button,
  Form,
  Input,
  Tag,
  Space,
  Popconfirm,
  Message,
  Typography,
  Tree,
} from '@arco-design/web-react';
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconEye,
  IconBranch,
} from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import type { TreeDataType } from '@arco-design/web-react/es/Tree/interface';
import {
  AppDrawer,
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
  TableBatchActionBar,
  buildStandardPagination,
  useGovernanceRail,
} from '../../../../components';
import { getGroupList, getGroupMembers, createGroup, updateGroup, deleteGroup } from './api';
import type { CreateGroupPayload, GroupRow, GroupMemberResp, GroupMemberRow } from './api';
import CmdbGroupForm from './CmdbGroupForm';
import { usePermission } from '../../../../hooks/usePermission';
import '../../../system/list-page.css';
import '../cmdb.css';

function flattenGroups(groups: GroupRow[]): GroupRow[] {
  return groups.flatMap((group) => [group, ...flattenGroups(group.children || [])]);
}

function findGroupById(groups: GroupRow[], id: number | null): GroupRow | null {
  if (!id) return null;
  for (const group of groups) {
    if (group.id === id) return group;
    const child = findGroupById(group.children || [], id);
    if (child) return child;
  }
  return null;
}

function findFirstGroup(groups: GroupRow[]): GroupRow | null {
  return groups[0] || null;
}

function matchesGroupKeyword(group: GroupRow, keyword: string): boolean {
  if (!keyword) {
    return true;
  }
  const normalizedKeyword = keyword.trim().toLowerCase();
  if (!normalizedKeyword) {
    return true;
  }
  const haystacks = [
    group.name,
    group.description,
    ...(group.conditions?.rules?.map((rule) => `${rule.key} ${rule.op} ${rule.val}`) || []),
  ]
    .filter(Boolean)
    .map((value) => String(value).toLowerCase());
  return haystacks.some((value) => value.includes(normalizedKeyword));
}

function filterGroupTree(groups: GroupRow[], keyword: string): GroupRow[] {
  if (!keyword.trim()) {
    return groups;
  }
  return groups.reduce<GroupRow[]>((acc, group) => {
    const children = filterGroupTree(group.children || [], keyword);
    if (matchesGroupKeyword(group, keyword) || children.length > 0) {
      acc.push({
        ...group,
        children,
      });
    }
    return acc;
  }, []);
}

export default function CmdbGroupList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const governanceRail = useGovernanceRail();

  const [data, setData] = useState<GroupRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<GroupRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [membersDrawer, setMembersDrawer] = useState(false);
  const [memberData, setMemberData] = useState<GroupMemberResp | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [selectedGroupId, setSelectedGroupId] = useState<number | null>(null);
  const [initialParentId, setInitialParentId] = useState<number | null>(null);
  const [keyword, setKeyword] = useState('');
  const [queryKeyword, setQueryKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);

  const canCreate = hasPerm('business:cmdb:group:create');
  const canUpdate = hasPerm('business:cmdb:group:update');
  const canDelete = hasPerm('business:cmdb:group:delete');
  const canDetail = hasPerm('business:cmdb:group:detail');

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await getGroupList();
      setData(result);
      const first = findFirstGroup(result);
      setSelectedGroupId((current) => current ?? first?.id ?? null);
    } catch (err) {
      setError(err);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const filteredTreeData = useMemo(() => filterGroupTree(data, queryKeyword), [data, queryKeyword]);
  const filteredFlatData = useMemo(() => flattenGroups(filteredTreeData), [filteredTreeData]);
  const pagedFlatData = useMemo(
    () => filteredFlatData.slice((page - 1) * pageSize, page * pageSize),
    [filteredFlatData, page, pageSize],
  );

  useEffect(() => {
    queueMicrotask(() => {
      const flat = filteredFlatData;
      if (!selectedGroupId && flat.length > 0) {
        setSelectedGroupId(flat[0].id);
      }
      if (selectedGroupId && !flat.some((item) => item.id === selectedGroupId)) {
        setSelectedGroupId(flat[0]?.id ?? null);
      }
    });
  }, [filteredFlatData, selectedGroupId]);

  useEffect(() => {
    const maxPage = Math.max(1, Math.ceil(filteredFlatData.length / pageSize));
    if (page > maxPage) {
      setPage(maxPage);
    }
  }, [filteredFlatData.length, page, pageSize]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const handleDelete = async (id: number) => {
    await deleteGroup(id);
    Message.success(t('common.deleteSuccess'));
    void loadData();
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => deleteGroup(Number(rowKey))));
      setSelectedRowKeys([]);
      Message.success(t('common.deleteSuccess'));
      await loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleFormSubmit = async (values: CreateGroupPayload) => {
    setSubmitting(true);
    try {
      if (editing) {
        await updateGroup(editing.id, values);
        Message.success(t('common.updateSuccess'));
      } else {
        await createGroup(values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      setInitialParentId(null);
      void loadData();
    } finally {
      setSubmitting(false);
    }
  };

  const handleViewMembers = async (row: GroupRow) => {
    if (!canDetail) return;
    const result = await getGroupMembers(row.id);
    setMemberData(result);
    setMembersDrawer(true);
  };

  const selectedGroup = useMemo(
    () => findGroupById(filteredTreeData, selectedGroupId) || findFirstGroup(filteredTreeData) || null,
    [filteredTreeData, selectedGroupId],
  );

  const flatData = useMemo(() => flattenGroups(data), [data]);

  const groupTreeData = useMemo<TreeDataType[]>(() => {
    const toNode = (group: GroupRow): TreeDataType => ({
      title: (
        <div className="cmdb-page__tree-node">
          <span className="cmdb-page__tree-node-copy">
            <span className="cmdb-page__tree-node-title">{group.name}</span>
            <span className="cmdb-page__tree-node-subtitle">
              {t('business.cmdb.group.memberCount')}
              {' '}
              {group.memberCount}
              {' · '}
              {t('business.cmdb.group.aggregateMemberCount')}
              {' '}
              {group.aggregateMemberCount ?? group.memberCount}
            </span>
          </span>
          <Tag size="small" color={group.id === selectedGroupId ? 'arcoblue' : 'gray'}>
            {group.aggregateMemberCount ?? group.memberCount}
          </Tag>
        </div>
      ),
      key: String(group.id),
      searchText: `${group.name} ${group.description} ${group.conditions?.rules
        ?.map((rule) => `${rule.key} ${rule.op} ${rule.val}`)
        .join(' ')}`,
      children: group.children?.map(toNode),
    }) as TreeDataType;
    return filteredTreeData.map(toNode);
  }, [filteredTreeData, selectedGroupId, t]);

  const defaultExpandedKeys = useMemo(
    () => filteredFlatData.map((group) => String(group.id)),
    [filteredFlatData],
  );

  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('business.cmdb.group.hero.total'),
        value: flatData.length,
        hint: t('business.cmdb.group.hero.totalHint'),
      },
      {
        key: 'members',
        label: t('business.cmdb.group.hero.aggregateMembers'),
        value: selectedGroup?.aggregateMemberCount ?? selectedGroup?.memberCount ?? 0,
        hint: t('business.cmdb.group.hero.aggregateMembersHint'),
      },
      {
        key: 'children',
        label: t('business.cmdb.group.hero.children'),
        value: selectedGroup?.descendantGroupCount ?? selectedGroup?.childCount ?? 0,
        hint: t('business.cmdb.group.hero.childrenHint'),
      },
      {
        key: 'scope',
        label: t('business.cmdb.group.hero.scope'),
        value: t('business.cmdb.group.hero.scopeValue'),
        hint: t('business.cmdb.group.hero.scopeHint'),
      },
      {
        key: 'rules',
        label: t('business.cmdb.group.hero.rules'),
        value: selectedGroup?.conditions?.rules?.length || 0,
        hint: t('business.cmdb.group.hero.rulesHint'),
      },
    ],
    [flatData.length, selectedGroup, t],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.cmdb.group.hero.scope'),
        value: t('business.cmdb.group.hero.scopeValue'),
        description: t('business.cmdb.group.hero.scopeHint'),
      },
      {
        label: t('business.cmdb.group.hero.members'),
        value: selectedGroup?.memberCount ?? 0,
        description: t('business.cmdb.group.hero.membersHint'),
      },
      {
        label: t('business.cmdb.group.hero.aggregateMembers'),
        value: selectedGroup?.aggregateMemberCount ?? selectedGroup?.memberCount ?? 0,
        description: t('business.cmdb.group.hero.aggregateMembersHint'),
      },
      {
        label: t('business.cmdb.group.hero.children'),
        value: selectedGroup?.descendantGroupCount ?? selectedGroup?.childCount ?? 0,
        description: t('business.cmdb.group.hero.childrenHint'),
      },
      {
        label: t('business.cmdb.group.hero.rules'),
        value: selectedGroup?.conditions?.rules?.length || 0,
        description: t('business.cmdb.group.hero.rulesHint'),
      },
    ],
    [selectedGroup, t],
  );

  const columns: ColumnProps<GroupRow>[] = [
    {
      title: t('business.cmdb.group.name'),
      dataIndex: 'name',
      width: 200,
      render: (_: unknown, row: GroupRow) => (
        <Space>
          <IconBranch />
          {row.name}
        </Space>
      ),
    },
    {
      title: t('business.cmdb.group.parent'),
      dataIndex: 'parentId',
      width: 160,
      render: (_: unknown, row: GroupRow) =>
        row.parentId
          ? findGroupById(data, row.parentId)?.name || '-'
          : t('business.cmdb.group.noParent'),
    },
    {
      title: t('business.cmdb.group.description'),
      dataIndex: 'description',
      width: 300,
      render: (_: unknown, row: GroupRow) => row.description || '-',
    },
    {
      title: t('business.cmdb.group.conditions'),
      dataIndex: 'conditions',
      width: 300,
      render: (_: unknown, row: GroupRow) =>
        row.conditions?.rules?.length ? (
          <Space wrap size={4}>
            {row.conditions.rules.map((r, i) => (
              <Tag key={i} size="small">
                {r.key} {r.op} {r.val}
              </Tag>
            ))}
          </Space>
        ) : (
          '-'
        ),
    },
    {
      title: t('business.cmdb.group.memberCount'),
      dataIndex: 'memberCount',
      width: 100,
      render: (_: unknown, row: GroupRow) => (
        <Tag color="arcoblue">{row.memberCount}</Tag>
      ),
    },
    {
      title: t('business.cmdb.group.aggregateMemberCount'),
      dataIndex: 'aggregateMemberCount',
      width: 120,
      render: (_: unknown, row: GroupRow) => (
        <Tag color="green">{row.aggregateMemberCount ?? row.memberCount}</Tag>
      ),
    },
    {
      title: t('business.cmdb.group.descendantGroupCount'),
      dataIndex: 'descendantGroupCount',
      width: 120,
      render: (_: unknown, row: GroupRow) => (
        <Tag color="purple">{row.descendantGroupCount ?? row.childCount ?? 0}</Tag>
      ),
    },
    {
      title: t('common.action'),
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_: unknown, row: GroupRow) => (
        <Space>
          {canDetail && (
            <Button
              type="text"
              size="small"
              icon={<IconEye />}
              onClick={() => handleViewMembers(row)}
            >
              {t('business.cmdb.group.members')}
            </Button>
          )}
          {canUpdate && (
            <Button
              type="text"
              size="small"
              icon={<IconEdit />}
              onClick={() => {
                setEditing(row);
                setInitialParentId(null);
                setVisible(true);
              }}
            >
              {t('common.edit')}
            </Button>
          )}
          {canCreate && (
            <Button
              type="text"
              size="small"
              icon={<IconPlus />}
              onClick={() => {
                setEditing(null);
                setInitialParentId(row.id);
                setVisible(true);
              }}
            >
              {t('business.cmdb.group.createChild')}
            </Button>
          )}
          {canDelete && (
            <Popconfirm
              title={t('business.cmdb.group.deleteConfirm')}
              onOk={() => handleDelete(row.id)}
            >
              <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                {t('common.delete')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          icon={<IconBranch />}
          eyebrow={t('business.cmdb.group.hero.eyebrow')}
          title={t('business.cmdb.group.title')}
          description={t('business.cmdb.group.hero.title')}
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
              {t('business.cmdb.group.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <FilterPanel>
          <Form layout="inline">
            <Form.Item label={t('common.keyword')}>
              <Input value={keyword} onChange={setKeyword} allowClear />
            </Form.Item>
            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  onClick={() => {
                    setQueryKeyword(keyword);
                    setPage(1);
                  }}
                >
                  {t('common.search')}
                </Button>
                <Button
                  onClick={() => {
                    setKeyword('');
                    setQueryKeyword('');
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
                  <Button
                    type="primary"
                    icon={<IconPlus />}
                    onClick={() => {
                      setEditing(null);
                      setInitialParentId(null);
                      setVisible(true);
                    }}
                  >
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
        <div className="cmdb-page__split">
          <Card className="page-panel cmdb-page__side-panel">
            <Typography.Text className="cmdb-page__side-title">
              {t('business.cmdb.group.tree.title')}
            </Typography.Text>
            {loading && data.length === 0 ? <PageLoading /> : null}
            {!loading && error && data.length === 0 ? (
              <PageError description={t('common.loadFailedDesc')} onRetry={loadData} />
            ) : null}
            {!loading && !error && filteredTreeData.length === 0 ? (
              <PageEmpty description={t('business.cmdb.group.empty')} />
            ) : null}
            {!loading && !(error && data.length === 0) && filteredTreeData.length > 0 ? (
              <Tree
                blockNode
                showLine
                defaultExpandedKeys={defaultExpandedKeys}
                treeData={groupTreeData}
                selectedKeys={selectedGroupId ? [String(selectedGroupId)] : []}
                onSelect={(keys) => {
                  const nextKey = keys[0];
                  if (!nextKey) return;
                  const nextId = Number(nextKey);
                  if (!Number.isNaN(nextId)) {
                    setSelectedGroupId(nextId);
                  }
                }}
              />
            ) : null}
          </Card>
          <div className="cmdb-page__content-stack">
            <Card className="page-panel system-list__table-card cmdb-page__group-table-card">
              {loading && data.length === 0 ? <PageLoading /> : null}
              {!loading && error && data.length === 0 ? (
                <PageError description={t('common.loadFailedDesc')} onRetry={loadData} />
              ) : null}
              {!loading && !error && filteredFlatData.length === 0 ? (
                <PageEmpty description={t('business.cmdb.group.empty')} />
              ) : null}
              {!loading && !(error && data.length === 0) && filteredFlatData.length > 0 ? (
                <AppTable
                  columns={columns}
                  data={pagedFlatData}
                  loading={loading}
                  rowSelection={{
                    type: 'checkbox',
                    selectedRowKeys,
                    checkCrossPage: true,
                    preserveSelectedRowKeys: true,
                    onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
                  }}
                  pagination={buildStandardPagination(t, {
                    current: page,
                    pageSize,
                    total: filteredFlatData.length,
                    onChange: (nextPage) => {
                      setPage(nextPage || 1);
                    },
                    onPageSizeChange: (nextPageSize) => {
                      setPageSize(nextPageSize || pageSize);
                      setPage(1);
                    },
                    pageSizeChangeResetCurrent: true,
                  })}
                  rowKey="id"
                  rowClassName={(record) =>
                    record.id === selectedGroupId ? 'cmdb-page__group-row--active' : ''
                  }
                />
              ) : null}
            </Card>
          </div>
        </div>
      </Space>
      <GovernanceInsightDrawer
        title={t('business.cmdb.group.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.cmdb.group.hero.sideLead')}
        noteDescription={t('business.cmdb.group.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
          setInitialParentId(null);
        }}
        title={
          editing
            ? t('business.cmdb.group.editTitle')
            : t('business.cmdb.group.createTitle')
        }
        footer={null}
        size="lg"
      >
        <CmdbGroupForm
          editing={editing}
          initialParentId={initialParentId}
          groupOptions={data}
          onSubmit={handleFormSubmit}
          onCancel={() => {
            setVisible(false);
            setEditing(null);
            setInitialParentId(null);
          }}
          submitting={submitting}
        />
      </AppModal>
      <AppDrawer
        visible={membersDrawer}
        onCancel={() => setMembersDrawer(false)}
        title={
          memberData
            ? `${t('business.cmdb.group.members')} - ${memberData.groupName}`
            : t('business.cmdb.group.members')
        }
        size="lg"
        footer={null}
      >
        {memberData?.members?.length ? (
          <AppTable
            data={memberData.members}
            columns={[
              { title: t('business.cmdb.host.hostname'), dataIndex: 'hostname' },
              { title: t('business.cmdb.host.ip'), dataIndex: 'ip' },
              {
                title: t('business.cmdb.host.status'),
                dataIndex: 'status',
                render: (_: unknown, row: GroupMemberRow) => (
                  <Tag color={row.status === 'online' ? 'green' : 'gray'}>
                    {t(`business.cmdb.host.status.${row.status}`)}
                  </Tag>
                ),
              },
            ]}
            rowKey="id"
            pagination={false}
            size="small"
            scroll={{ x: 'max-content' }}
          />
        ) : (
          <div style={{ color: 'var(--text-tertiary)' }}>
            {t('business.cmdb.group.empty')}
          </div>
        )}
      </AppDrawer>
    </PageContainer>
  );
}
