import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Button,
  Card,
  Form,
  Grid,
  Input,
  Message,
  Popconfirm,
  Select,
  Space,
  Tag,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconDelete, IconEdit, IconEye, IconPlus } from '@arco-design/web-react/icon';
import {
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
  TABLE_ACTION_COLUMN_WIDTH,
  buildStandardPagination,
  useGovernanceRail,
} from '../../../components';
import { usePermission } from '../../../hooks/usePermission';
import {
  createBizScope,
  deleteBizScope,
  getBizScopeList,
  updateBizScope,
  type BizScopeListQuery,
  type BizScopePayload,
  type BizScopeRow,
} from './api';
import BizScopeForm from './BizScopeForm';
import '../../system/list-page.css';

const { Row, Col } = Grid;

const emptyQuery: BizScopeListQuery = {
  code: '',
  name: '',
  owner: '',
  environment: '',
  status: '',
  page: 1,
  pageSize: 20,
};

export default function BizScopeList() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { hasPerm } = usePermission();
  const [queryForm] = Form.useForm<BizScopeListQuery>();
  const [data, setData] = useState<BizScopeRow[]>([]);
  const [query, setQuery] = useState<BizScopeListQuery>(emptyQuery);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [visible, setVisible] = useState(false);
  const [editing, setEditing] = useState<BizScopeRow | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const governanceRail = useGovernanceRail();

  const canCreate = hasPerm('business:bizscope:create');
  const canUpdate = hasPerm('business:bizscope:update');
  const canDelete = hasPerm('business:bizscope:delete');
  const canView = hasPerm('business:bizscope:view');

  const loadData = useCallback(async (nextQuery: BizScopeListQuery = query) => {
    setLoading(true);
    setError(null);
    try {
      const result = await getBizScopeList(nextQuery);
      setData(result.items);
      setTotal(result.total);
    } catch (requestError) {
      setError(requestError);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [query, t]);

  useEffect(() => {
    queueMicrotask(() => {
      void loadData();
    });
  }, [loadData]);

  const heroMetrics = useMemo(
    () => [
      {
        key: 'total',
        label: t('business.bizscope.hero.total'),
        value: total,
      },
      {
        key: 'active',
        label: t('business.bizscope.hero.active'),
        value: data.filter((item) => item.status === 'active').length,
      },
      {
        key: 'prod',
        label: t('business.bizscope.hero.prod'),
        value: data.filter((item) => item.environment === 'prod').length,
      },
    ],
    [data, t, total],
  );

  const governanceSummaryItems = useMemo(
    () => [
      {
        label: t('business.bizscope.hero.total'),
        value: total,
        description: t('business.bizscope.hero.title'),
      },
      {
        label: t('business.bizscope.hero.active'),
        value: data.filter((item) => item.status === 'active').length,
        description: t('business.bizscope.status.active'),
      },
      {
        label: t('business.bizscope.hero.prod'),
        value: data.filter((item) => item.environment === 'prod').length,
        description: t('business.bizscope.environment.prod'),
      },
    ],
    [data, t, total],
  );

  const search = () => {
    const values = queryForm.getFieldsValue();
    const nextQuery = {
      ...query,
      ...values,
      page: 1,
    };
    setSelectedRowKeys([]);
    setQuery(nextQuery);
    void loadData(nextQuery);
  };

  const reset = () => {
    queryForm.setFieldsValue(emptyQuery);
    setSelectedRowKeys([]);
    setQuery(emptyQuery);
    void loadData(emptyQuery);
  };

  const handleSubmit = async (values: BizScopePayload) => {
    setSubmitting(true);
    try {
      if (editing) {
        await updateBizScope(editing.id, values);
        Message.success(t('common.updateSuccess'));
      } else {
        await createBizScope(values);
        Message.success(t('common.createSuccess'));
      }
      setVisible(false);
      setEditing(null);
      await loadData(query);
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
      await Promise.all(selectedRowKeys.map((rowKey) => deleteBizScope(Number(rowKey))));
      Message.success(t('common.deleteSuccess'));
      setSelectedRowKeys([]);
      await loadData(query);
    } finally {
      setSubmitting(false);
    }
  };

  const columns = useMemo<ColumnProps<BizScopeRow>[]>(
    () => [
      {
        title: t('business.bizscope.field.code'),
        dataIndex: 'code',
        width: 160,
        render: (_: unknown, row) => <Tag color="arcoblue">{row.code}</Tag>,
      },
      {
        title: t('business.bizscope.field.name'),
        dataIndex: 'name',
        width: 180,
      },
      {
        title: t('business.bizscope.field.owner'),
        dataIndex: 'owner',
        width: 140,
        render: (_: unknown, row) => row.owner || '-',
      },
      {
        title: t('business.bizscope.field.environment'),
        dataIndex: 'environment',
        width: 120,
        render: (_: unknown, row) => t(`business.bizscope.environment.${row.environment}`),
      },
      {
        title: t('business.bizscope.field.status'),
        dataIndex: 'status',
        width: 120,
        render: (_: unknown, row) => (
          <Tag color={row.status === 'active' ? 'green' : 'gray'}>
            {t(`business.bizscope.status.${row.status}`)}
          </Tag>
        ),
      },
      {
        title: t('business.bizscope.field.remark'),
        dataIndex: 'remark',
        render: (_: unknown, row) => row.remark || '-',
      },
      {
        title: t('common.action'),
        width: TABLE_ACTION_COLUMN_WIDTH.medium,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space className="system-list__actions">
            {canView ? (
              <Button
                type="text"
                size="small"
                icon={<IconEye />}
                onClick={() => navigate(`/operations/business-scope/${row.id}`)}
              >
                {t('common.detail')}
              </Button>
            ) : null}
            {canUpdate ? (
              <Button
                type="text"
                size="small"
                icon={<IconEdit />}
                onClick={() => {
                  setEditing(row);
                  setVisible(true);
                }}
              >
                {t('common.edit')}
              </Button>
            ) : null}
            {canDelete ? (
              <Popconfirm
                title={t('business.bizscope.deleteConfirm')}
                onOk={async () => {
                  await deleteBizScope(row.id);
                  Message.success(t('common.deleteSuccess'));
                  await loadData(query);
                }}
              >
                <Button type="text" size="small" status="danger" icon={<IconDelete />}>
                  {t('common.delete')}
                </Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canUpdate, canView, loadData, navigate, query, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.bizscope.hero.eyebrow')}
          title={t('operations.bizscope.menu')}
          description={t('business.bizscope.hero.title')}
          metrics={heroMetrics}
          action={(
            <GovernanceRailToggleButton
              expanded={governanceRail.expanded}
              onToggle={governanceRail.toggle}
            >
              {t('business.bizscope.hero.title')}
            </GovernanceRailToggleButton>
          )}
        />
        <FilterPanel>
          <Form form={queryForm} layout="vertical" onSubmit={search}>
            <Row gutter={16}>
              <Col xs={24} md={6}>
                <Form.Item label={t('business.bizscope.field.code')} field="code">
                  <Input allowClear onPressEnter={() => queryForm.submit()} />
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item label={t('business.bizscope.field.name')} field="name">
                  <Input allowClear onPressEnter={() => queryForm.submit()} />
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item label={t('business.bizscope.field.owner')} field="owner">
                  <Input allowClear onPressEnter={() => queryForm.submit()} />
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item label={t('business.bizscope.field.environment')} field="environment">
                  <Select allowClear>
                    {['dev', 'test', 'prod'].map((item) => (
                      <Select.Option key={item} value={item}>
                        {t(`business.bizscope.environment.${item}`)}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item label={t('business.bizscope.field.status')} field="status">
                  <Select allowClear>
                    {['active', 'inactive'].map((item) => (
                      <Select.Option key={item} value={item}>
                        {t(`business.bizscope.status.${item}`)}
                      </Select.Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
              <Col xs={24} md={6}>
                <Form.Item className="filter-panel__action-item">
                  <Space>
                    <Button type="primary" htmlType="submit">
                      {t('common.search')}
                    </Button>
                    <Button onClick={reset}>{t('common.reset')}</Button>
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
                  <Button
                    type="primary"
                    icon={<IconPlus />}
                    onClick={() => {
                      setEditing(null);
                      setVisible(true);
                    }}
                  >
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
        <Card className="page-panel system-list__table-card">
          {loading && data.length === 0 ? <PageLoading /> : null}
          {error && data.length === 0 ? (
            <PageError onRetry={() => void loadData(query)} />
          ) : null}
          {!loading && !error && data.length === 0 ? (
            <PageEmpty description={t('business.bizscope.empty')} />
          ) : null}
          {!loading && !(error && data.length === 0) && data.length > 0 ? (
            <AppTable
              className="system-list__table"
              rowKey="id"
              data={data}
              columns={columns}
              scroll={{ x: 'max-content' }}
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
                pageSize: query.pageSize || 20,
                total,
                onChange: (page, pageSize) => {
                  const nextQuery = { ...query, page, pageSize };
                  setQuery(nextQuery);
                  void loadData(nextQuery);
                },
              })}
            />
          ) : null}
        </Card>
      </Space>
      <GovernanceInsightDrawer
        visible={governanceRail.expanded}
        title={t('operations.bizscope.menu')}
        onClose={governanceRail.close}
      >
        <GovernanceRailSummary items={governanceSummaryItems} />
      </GovernanceInsightDrawer>
      <AppModal
        visible={visible}
        footer={null}
        title={editing ? t('business.bizscope.editTitle') : t('business.bizscope.createTitle')}
        onCancel={() => {
          setVisible(false);
          setEditing(null);
        }}
      >
        <BizScopeForm
          mode={editing ? 'update' : 'create'}
          initialValues={editing || undefined}
          submitting={submitting}
          onSubmit={handleSubmit}
          onCancel={() => {
            setVisible(false);
            setEditing(null);
          }}
        />
      </AppModal>
    </PageContainer>
  );
}
