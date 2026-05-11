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
  Typography,
} from '@arco-design/web-react';
import type { PaginationProps } from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconPlus, IconEdit, IconDelete, IconCode, IconEye } from '@arco-design/web-react/icon';
import { AppModal, PageEmpty, PageError, PageLoading } from '../../../../components';
import PageContainer from '../../../../components/patterns/PageContainer';
import PageHeader from '../../../../components/patterns/PageHeader';
import FilterPanel from '../../../../components/patterns/FilterPanel';
import AppTable from '../../../../components/data-display/AppTable';
import ListHeaderActions from '../../../../components/patterns/ListHeaderActions';
import { createHost, deleteHost, getHostList, updateHost } from './api';
import type { HostRow, HostListQuery } from './api';
import { usePermission } from '../../../../hooks/usePermission';
import CmdbHostForm from './CmdbHostForm';
import '../../../../core/styles/list-page.css';
import '../cmdb.css';

const Row = Grid.Row;
const Col = Grid.Col;

const statusColorMap: Record<string, string> = {
  pending: 'gray',
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
  const [error, setError] = useState<unknown>(null);

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
    loadData();
  }, [loadData]);

  const handleSearch = () => {
    setQuery((prev) => ({ ...prev, page: 1, keyword, status: filterStatus, os: filterOS }));
  };

  const handleReset = () => {
    setKeyword('');
    setFilterStatus('');
    setFilterOS('');
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

  const handleDelete = async (id: number) => {
    await deleteHost(id);
    Message.success(t('common.deleteSuccess'));
    loadData(query);
  };

  const handleFormSubmit = async (values: any) => {
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

  const handlePageChange = (page: number) => {
    setQuery((prev) => ({ ...prev, page }));
  };

  const handlePageSizeChange = (pageSize: number) => {
    setQuery((prev) => ({ ...prev, page: 1, pageSize }));
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
            onClick={() => navigate(`/operations/cmdb/host/${row.id}`)}
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

  const pagination: PaginationProps = {
    current: query.page || 1,
    pageSize: query.pageSize || 10,
    total,
    sizeOptions: [10, 20, 50, 100],
    onChange: handlePageChange,
    onPageSizeChange: handlePageSizeChange,
    showTotal: true,
  };

  return (
    <PageContainer>
      <PageHeader
        title={t('business.cmdb.host.title')}
        extra={
          <ListHeaderActions
            primary={
              canCreate ? (
                <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
                  {t('common.add')}
                </Button>
              ) : null
            }
          />
        }
      />
      <Space direction="vertical" size={16} className="system-page-template">
        <Card className="page-panel system-page-hero cmdb-page__hero">
          <div className="system-page-hero__top">
            <div className="system-page-hero__copy">
              <span className="system-page-hero__eyebrow">{t('business.cmdb.host.hero.eyebrow')}</span>
              <Typography.Title heading={5} className="system-page-hero__title cmdb-page__hero-title">
                {t('business.cmdb.host.hero.title')}
              </Typography.Title>
            </div>
          </div>
          <div className="cmdb-page__hero-grid">
            {heroStats.map((item) => (
              <div key={item.key} className="cmdb-page__hero-metric">
                <span className="cmdb-page__hero-label">{item.label}</span>
                <span className="cmdb-page__hero-value">{item.value}</span>
                <span className="cmdb-page__hero-hint">{item.hint}</span>
              </div>
            ))}
          </div>
        </Card>
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
                    {['pending', 'online', 'offline', 'maintenance'].map((s) => (
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
              pagination={pagination}
              rowKey="id"
              scroll={{ x: 'max-content' }}
            />
          ) : null}
        </Card>
      </Space>
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
