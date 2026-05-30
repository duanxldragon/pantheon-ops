import { useState, useEffect, useCallback, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Grid,
  Card,
  Button,
  Tag,
  Space,
  Popconfirm,
  Form,
  Input,
  Select,
  Message,
  Descriptions,
  Typography,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconCode,
  IconEye,
  IconStorage,
} from '@arco-design/web-react/icon';
import {
  AppModal,
  AppTable,
  buildStandardPagination,
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
  useGovernanceRail,
} from '../../../../components';
import { createHost, deleteHost, getHostDetail, getHostList, updateHost } from './api';
import type { CreateHostPayload, HostRow, HostListQuery } from './api';
import { getBizScopeOptions, type BizScopeOptionItem } from '../../bizscope/api';
import { usePermission } from '../../../../hooks/usePermission';
import { formatDateTime } from '../../../../core/format/dateTime';
import CmdbHostForm from './CmdbHostForm';
import '../../../system/list-page.css';
import '../cmdb.css';

const Row = Grid.Row;
const Col = Grid.Col;

const statusColorMap: Record<string, string> = {
  pending: 'gray',
  assigned: 'arcoblue',
  online: 'green',
  offline: 'red',
  maintenance: 'orange',
};

const osColorMap: Record<string, string> = {
  linux: 'blue',
  windows: 'arcoblue',
};

export default function CmdbHostList() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const navigate = useNavigate();
  const governanceRail = useGovernanceRail();

  const [data, setData] = useState<HostRow[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState<HostListQuery>({ page: 1, pageSize: 10 });
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<HostRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [filterStatus, setFilterStatus] = useState<string>('');
  const [filterOS, setFilterOS] = useState<string>('');
  const [filterBusinessScopeId, setFilterBusinessScopeId] = useState<number | undefined>();
  const [scopeOptions, setScopeOptions] = useState<BizScopeOptionItem[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [error, setError] = useState<unknown>(null);
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailRecord, setDetailRecord] = useState<HostRow | null>(null);

  const canCreate = hasPerm('business:cmdb:host:create');
  const canUpdate = hasPerm('business:cmdb:host:update');
  const canDelete = hasPerm('business:cmdb:host:delete');
  const canCollect = hasPerm('business:cmdb:host:collect');

  const loadData = useCallback(
    async (nextQuery = query) => {
      setLoading(true);
      setError(null);
      try {
        const result = await getHostList(nextQuery);
        setData(result.items);
        setTotal(result.total);
      } catch (err) {
        setError(err);
        Message.error(t('common.loadFailed'));
      } finally {
        setLoading(false);
      }
    },
    [query, t],
  );

  useEffect(() => {
    queueMicrotask(() => {
      loadData();
    });
  }, [loadData]);

  useEffect(() => {
    queueMicrotask(async () => {
      try {
        const result = await getBizScopeOptions();
        setScopeOptions(result);
      } catch {
        setScopeOptions([]);
      }
    });
  }, []);

  const handleSearch = () => {
    setSelectedRowKeys([]);
    setQuery((prev) => ({
      ...prev,
      page: 1,
      keyword,
      status: filterStatus,
      os: filterOS,
      businessScopeId: filterBusinessScopeId,
    }));
  };

  const handleReset = () => {
    setKeyword('');
    setFilterStatus('');
    setFilterOS('');
    setFilterBusinessScopeId(undefined);
    setSelectedRowKeys([]);
    setQuery({ page: 1, pageSize: 10 });
  };

  const heroStats = useMemo(
    () => [
      {
        key: 'total',
        label: t('business.cmdb.host.hero.total'),
        value: total,
        hint: t('business.cmdb.host.hero.totalHint'),
      },
      {
        key: 'online',
        label: t('business.cmdb.host.hero.online'),
        value: data.filter((item) => item.status === 'online').length,
        hint: t('business.cmdb.host.hero.onlineHint'),
      },
      {
        key: 'maintenance',
        label: t('business.cmdb.host.hero.maintenance'),
        value: data.filter((item) => item.status === 'maintenance').length,
        hint: t('business.cmdb.host.hero.maintenanceHint'),
      },
      {
        key: 'assigned',
        label: t('business.cmdb.host.hero.assigned'),
        value: data.filter((item) => item.status === 'assigned').length,
        hint: t('business.cmdb.host.hero.assignedHint'),
      },
      {
        key: 'os',
        label: t('business.cmdb.host.hero.osSummary'),
        value: data.length
          ? `${t('business.cmdb.host.os.linux')} ${data.filter((item) => item.os === 'linux').length} / ${t('business.cmdb.host.os.windows')} ${data.filter((item) => item.os === 'windows').length}`
          : '-',
        hint: t('business.cmdb.host.hero.osSummaryHint'),
      },
    ],
    [data, t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.cmdb.host.hero.scope'),
        value: t('business.cmdb.host.hero.scopeValue'),
        description: t('business.cmdb.host.hero.scopeHint'),
      },
      {
        label: t('business.cmdb.host.hero.status'),
        value: data.filter((item) => item.status === 'online').length,
        description: t('business.cmdb.host.hero.statusHint'),
      },
      {
        label: t('business.cmdb.host.hero.assigned'),
        value: data.filter((item) => item.status === 'assigned').length,
        description: t('business.cmdb.host.hero.assignedHint'),
      },
      {
        label: t('business.cmdb.host.hero.os'),
        value: data.length
          ? `${data.filter((item) => item.os === 'linux').length} / ${data.filter((item) => item.os === 'windows').length}`
          : '-',
        description: t('business.cmdb.host.hero.osHint'),
      },
      {
        label: t('business.cmdb.host.hero.labels'),
        value: data.reduce((sum, item) => sum + (item.labelValues?.length || 0), 0),
        description: t('business.cmdb.host.hero.labelsHint'),
      },
      {
        label: t('business.cmdb.host.hero.components'),
        value: data.reduce((sum, item) => sum + (item.installedComponents?.length || 0), 0),
        description: t('business.cmdb.host.hero.componentsHint'),
      },
    ],
    [data, t],
  );

  const handleDelete = async (id: number) => {
    await deleteHost(id);
    Message.success(t('common.deleteSuccess'));
    loadData(query);
  };

  const handleFormSubmit = async (values: CreateHostPayload) => {
    setSubmitting(true);
    try {
      if (editing) {
        await updateHost(editing.id, values);
        Message.success(t('common.updateSuccess'));
      } else {
        await createHost(values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      loadData(query);
      setSelectedRowKeys([]);
    } finally {
      setSubmitting(false);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }
    setSubmitting(true);
    try {
      await Promise.all(selectedRowKeys.map((rowKey) => deleteHost(Number(rowKey))));
      Message.success(t('common.deleteSuccess'));
      setSelectedRowKeys([]);
      loadData(query);
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = (row: HostRow) => {
    setEditing(row);
    setVisible(true);
  };

  const handleCreate = () => {
    setEditing(null);
    setVisible(true);
  };

  const openDetail = async (id: number) => {
    setDetailVisible(true);
    setDetailLoading(true);
    try {
      setDetailRecord(await getHostDetail(id));
    } finally {
      setDetailLoading(false);
    }
  };

  const columns: ColumnProps<HostRow>[] = [
    {
      title: t('business.cmdb.host.hostname'),
      dataIndex: 'hostname',
      width: 150,
    },
    {
      title: t('business.cmdb.host.ip'),
      dataIndex: 'ip',
      width: 140,
    },
    {
      title: t('business.cmdb.host.cpuCores'),
      dataIndex: 'cpuCores',
      width: 80,
      render: (_: unknown, row: HostRow) => (row.cpuCores ? row.cpuCores : '-'),
    },
    {
      title: t('business.cmdb.host.memoryGb'),
      dataIndex: 'memoryGb',
      width: 100,
      render: (_: unknown, row: HostRow) => (row.memoryGb ? row.memoryGb : '-'),
    },
    {
      title: t('business.cmdb.host.diskGb'),
      dataIndex: 'diskGb',
      width: 100,
      render: (_: unknown, row: HostRow) => (row.diskGb ? row.diskGb : '-'),
    },
    {
      title: t('business.cmdb.host.os'),
      dataIndex: 'os',
      width: 180,
      render: (_: unknown, row: HostRow) => (
        <Space direction="vertical" size={2}>
          <Tag color={osColorMap[row.os] || 'gray'}>
            {t(`business.cmdb.host.os.${row.os}`)}
          </Tag>
          <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
            {row.osVersion || '-'}
          </span>
        </Space>
      ),
    },
    {
      title: t('business.cmdb.host.status'),
      dataIndex: 'status',
      width: 100,
      render: (_: unknown, row: HostRow) => (
        <Tag color={statusColorMap[row.status] || 'gray'}>
          {t(`business.cmdb.host.status.${row.status}`)}
        </Tag>
      ),
    },
    {
      title: t('business.cmdb.host.businessScope'),
      dataIndex: 'businessScopeName',
      width: 180,
      render: (_: unknown, row: HostRow) =>
        row.businessScopeName ? (
          <Space direction="vertical" size={2}>
            <span>{row.businessScopeName}</span>
            <span style={{ color: 'var(--text-tertiary)', fontSize: 12 }}>
              {row.businessScopeCode || '-'}
            </span>
          </Space>
        ) : (
          '-'
        ),
    },
    {
      title: t('business.cmdb.host.matchedGroups'),
      dataIndex: 'matchedGroups',
      width: 220,
      render: (_: unknown, row: HostRow) =>
        row.matchedGroups?.length ? (
          <Space wrap size={4}>
            {row.matchedGroups.slice(0, 3).map((group) => (
              <Tag key={group.id} color="arcoblue">
                {group.name}
              </Tag>
            ))}
            {row.matchedGroups.length > 3 ? (
              <Tag color="gray">+{row.matchedGroups.length - 3}</Tag>
            ) : null}
          </Space>
        ) : (
          '-'
        ),
    },
    {
      title: t('business.cmdb.host.labels'),
      dataIndex: 'labelValues',
      width: 200,
      render: (_: unknown, row: HostRow) =>
        row.labelValues?.length ? (
          <Space wrap size={4}>
            {row.labelValues.map((l, i) => (
              <Tag key={i} size="small">
                {l.key}={l.val}
              </Tag>
            ))}
          </Space>
        ) : (
          '-'
        ),
    },
    {
      title: t('common.action'),
      key: 'action',
      fixed: 'right',
      width: 220,
      render: (_: unknown, row: HostRow) => (
        <Space>
          <Button
            type="text"
            size="small"
            icon={<IconEye />}
            onClick={() => {
              openDetail(row.id);
            }}
          >
            {t('common.detail')}
          </Button>
          {canUpdate && (
            <Button
              type="text"
              size="small"
              icon={<IconEdit />}
              onClick={() => handleEdit(row)}
            >
              {t('common.edit')}
            </Button>
          )}
          {canCollect && row.os === 'linux' && (
            <Button
              type="text"
              size="small"
              icon={<IconCode />}
              onClick={() => navigate(`/operations/cmdb/host/${row.id}?collect=1`)}
            >
              {t('business.cmdb.host.collect')}
            </Button>
          )}
          {canDelete && (
            <Popconfirm
              title={t('business.cmdb.host.deleteConfirm')}
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
          icon={<IconStorage />}
          eyebrow={t('business.cmdb.host.hero.eyebrow')}
          title={t('business.cmdb.host.title')}
          description={t('business.cmdb.host.hero.title')}
          metrics={heroStats.slice(0, 4).map((item) => ({
            key: item.key,
            label: item.label,
            value: item.value,
          }))}
          action={
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('business.cmdb.host.hero.summaryTitle')}
            </GovernanceRailToggleButton>
          }
        />
        <FilterPanel>
          <Form layout="vertical" onSubmit={handleSearch}>
            <Row gutter={16}>
              <Col xs={24} md={12} lg={8}>
                <Form.Item label={t('common.keyword')}>
                  <Input
                    value={keyword}
                    onChange={setKeyword}
                    placeholder={t('common.keyword')}
                    allowClear
                  />
                </Form.Item>
              </Col>
              <Col xs={24} md={12} lg={4}>
                <Form.Item label={t('business.cmdb.host.status')}>
                  <Select
                    value={filterStatus}
                    onChange={setFilterStatus}
                    placeholder={t('common.all')}
                    allowClear
                  >
                    {['pending', 'assigned', 'online', 'offline', 'maintenance'].map((s) => (
                      <Select.Option key={s} value={s}>
                        {t(`business.cmdb.host.status.${s}`)}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col xs={24} md={12} lg={4}>
                <Form.Item label={t('business.cmdb.host.os')}>
                  <Select
                    value={filterOS}
                    onChange={setFilterOS}
                    placeholder={t('common.all')}
                    allowClear
                  >
                    {['linux', 'windows'].map((o) => (
                      <Select.Option key={o} value={o}>
                        {t(`business.cmdb.host.os.${o}`)}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col xs={24} md={12} lg={4}>
                <Form.Item label={t('business.cmdb.host.businessScope')}>
                  <Select
                    value={filterBusinessScopeId}
                    onChange={(value) => setFilterBusinessScopeId(value || undefined)}
                    placeholder={t('common.all')}
                    allowClear
                  >
                    {scopeOptions.map((item) => (
                      <Select.Option key={item.id} value={item.id}>
                        {item.name}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col xs={24} md={12} lg={8}>
                <Form.Item className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" onClick={handleSearch}>
                      {t('common.search')}
                    </Button>
                    <Button onClick={handleReset}>{t('common.reset')}</Button>
                  </Space>
                </Form.Item>
              </Col>
            </Row>
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
                primary={
                  <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
                    {t('common.add')}
                  </Button>
                }
              />
            ) : undefined
          }
          actions={
            canDelete ? (
              <Popconfirm
                title={t('common.deleteConfirm')}
                onOk={() => {
                  handleBatchDelete();
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
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {!loading && error && data.length === 0 ? (
            <PageError
              description={t('common.loadFailedDesc')}
              onRetry={() => loadData(query)}
            />
          ) : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.cmdb.host.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              columns={columns}
              data={data}
              loading={loading}
              rowSelection={{
                type: 'checkbox',
                selectedRowKeys,
                checkCrossPage: true,
                preserveSelectedRowKeys: true,
                fixed: true,
                onChange: (rowKeys) => setSelectedRowKeys(rowKeys),
              }}
              pagination={buildStandardPagination(t, {
                current: query.page || 1,
                pageSize: query.pageSize || 10,
                total,
                onChange: (nextPage) => {
                  setQuery((prev) => ({
                    ...prev,
                    page: nextPage || 1,
                  }));
                },
                onPageSizeChange: (nextPageSize) => {
                  setQuery((prev) => ({
                    ...prev,
                    page: 1,
                    pageSize: nextPageSize || prev.pageSize || 10,
                  }));
                },
                pageSizeChangeResetCurrent: true,
              })}
              rowKey="id"
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        title={t('business.cmdb.host.hero.summaryTitle')}
        visible={governanceRail.expanded}
        onClose={governanceRail.close}
        noteTitle={t('business.cmdb.host.hero.summaryTitle')}
        noteDescription={t('business.cmdb.host.hero.sideDesc')}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        visible={detailVisible}
        title={detailRecord?.hostname || t('operations.cmdb.host.detail')}
        footer={null}
        size="detail"
        onCancel={() => {
          setDetailVisible(false);
          setDetailRecord(null);
        }}
      >
        {detailLoading ? <PageLoading /> : null}
        {!detailLoading && !detailRecord ? (
          <PageEmpty description={t('common.loadFailedDesc')} />
        ) : null}
        {!detailLoading && detailRecord ? (
          <Space direction="vertical" size={16} style={{ width: '100%' }}>
            <Descriptions
              column={2}
              data={[
                { label: t('business.cmdb.host.hostname'), value: detailRecord.hostname },
                { label: t('business.cmdb.host.ip'), value: detailRecord.ip },
                { label: t('business.cmdb.host.sshPort'), value: detailRecord.sshPort || 22 },
                { label: t('business.cmdb.host.os'), value: t(`business.cmdb.host.os.${detailRecord.os}`) },
                { label: t('business.cmdb.host.osVersion'), value: detailRecord.osVersion || '-' },
                { label: t('business.cmdb.host.status'), value: t(`business.cmdb.host.status.${detailRecord.status}`) },
                { label: t('business.cmdb.host.businessScope'), value: detailRecord.businessScopeName || '-' },
                { label: t('business.cmdb.host.owner'), value: detailRecord.owner || '-' },
                { label: t('business.cmdb.host.cpuCores'), value: detailRecord.cpuCores || '-' },
                { label: t('business.cmdb.host.memoryGb'), value: detailRecord.memoryGb || '-' },
                { label: t('business.cmdb.host.diskGb'), value: detailRecord.diskGb || '-' },
                { label: t('business.cmdb.host.remark'), value: detailRecord.remark || '-' },
              ]}
            />
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>
                  {t('business.cmdb.host.installedComponents')}
                </Typography.Text>
                {detailRecord.installedComponents?.length ? (
                  detailRecord.installedComponents.map((item, index) => (
                    <Card key={`${item.name}-${index}`} className="page-panel" style={{ padding: 12 }}>
                      <Space direction="vertical" size={4}>
                        <Space wrap>
                          <Tag color="arcoblue">{`${item.name} ${item.version}`}</Tag>
                          {item.executorType ? (
                            <Tag>{t(`business.deploy.task.executorType.${item.executorType}`)}</Tag>
                          ) : null}
                        </Space>
                        <Typography.Text type="secondary">
                          {item.deployTaskName || '-'}
                          {item.deployedAt ? ` · ${formatDateTime(item.deployedAt)}` : ''}
                        </Typography.Text>
                      </Space>
                    </Card>
                  ))
                ) : (
                  <PageEmpty description={t('business.cmdb.host.componentsEmpty')} />
                )}
              </Space>
            </Card>
            <Card className="page-panel">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Typography.Text style={{ fontWeight: 600 }}>
                  {t('business.cmdb.host.matchedGroups')}
                </Typography.Text>
                {detailRecord.matchedGroups?.length ? (
                  detailRecord.matchedGroups.map((group) => (
                    <Card key={group.id} className="page-panel" style={{ padding: 12 }}>
                      <Space direction="vertical" size={2}>
                        <span>{group.name}</span>
                        <Typography.Text type="secondary">{group.fullPath}</Typography.Text>
                      </Space>
                    </Card>
                  ))
                ) : (
                  <PageEmpty description={t('business.cmdb.host.matchedGroupsEmpty')} />
                )}
              </Space>
            </Card>
          </Space>
        ) : null}
      </AppModal>
      <AppModal
        visible={visible}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
        title={
          editing
            ? t('business.cmdb.host.editTitle')
            : t('business.cmdb.host.createTitle')
        }
        footer={null}
        size="lg"
      >
        <CmdbHostForm
          editing={editing}
          onSubmit={handleFormSubmit}
          onCancel={() => {
            setVisible(false);
            setEditing(null);
          }}
          submitting={submitting}
        />
      </AppModal>
    </PageContainer>
  );
}
