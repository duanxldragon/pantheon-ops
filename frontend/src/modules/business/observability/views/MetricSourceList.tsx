import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Button,
  Tag,
  Space,
  Popconfirm,
  Select,
  Message,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconPlus, IconEdit, IconDelete } from '@arco-design/web-react/icon';
import {
  AppTable,
  buildStandardPagination,
  SearchToolbar,
  PageContainer,
} from '../../../../components';
import { observabilityApi } from '../api';
import type { MetricSource, PaginationParams } from '../api';
import { formatDateTime } from '../../../../core/format/dateTime';
import '../../../system/components/shared/list-page.css';

const MetricSourceList = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<MetricSource[]>([]);
  const [total, setTotal] = useState(0);
  const [pagination, setPagination] = useState(buildStandardPagination());
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filters, setFilters] = useState<{
    type?: string;
    status?: string;
  }>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: PaginationParams = {
        page: pagination.current,
        pageSize: pagination.pageSize,
        type: filters.type,
        status: filters.status,
      };

      const { data } = await observabilityApi.getMetricSources(params);
      setDataSource(data.data);
      setTotal(data.total);
    } catch (error) {
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [pagination.current, pagination.pageSize, filters, t]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const handleSearch = (keyword: string) => {
    setSearchKeyword(keyword);
    setPagination({ ...pagination, current: 1 });
  };

  const handleFilterChange = (key: string, value: string | undefined) => {
    setFilters({ ...filters, [key]: value });
    setPagination({ ...pagination, current: 1 });
  };

  const handleCreate = () => {
    Message.info('新建指标源功能开发中（Week 1 后续）');
  };

  const handleEdit = (record: MetricSource) => {
    Message.info(`编辑指标源: ${record.name}（Week 1 后续）`);
  };

  const handleDelete = async (id: number) => {
    try {
      await observabilityApi.deleteMetricSource(id);
      Message.success(t('common.deleteSuccess'));
      fetchData();
    } catch (error) {
      Message.error(t('common.deleteFailed'));
    }
  };

  const columns: ColumnProps<MetricSource>[] = [
    {
      title: '名称',
      dataIndex: 'name',
      width: 200,
    },
    {
      title: '类型',
      dataIndex: 'type',
      width: 150,
      render: (type: string) => {
        const colorMap: Record<string, string> = {
          prometheus: 'blue',
          'victoria-metrics': 'green',
        };
        const labelMap: Record<string, string> = {
          prometheus: 'Prometheus',
          'victoria-metrics': 'VictoriaMetrics',
        };
        return <Tag color={colorMap[type]}>{labelMap[type] || type}</Tag>;
      },
    },
    {
      title: '端点',
      dataIndex: 'endpoint',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const colorMap: Record<string, string> = {
          active: 'green',
          inactive: 'gray',
        };
        const labelMap: Record<string, string> = {
          active: '活跃',
          inactive: '停用',
        };
        return <Tag color={colorMap[status]}>{labelMap[status]}</Tag>;
      },
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      width: 180,
      render: (createdAt: string) => formatDateTime(createdAt),
    },
    {
      title: '操作',
      width: 150,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space>
          <Button type="text" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm
            title="确定删除此指标源吗？"
            onOk={() => handleDelete(record.id)}
          >
            <Button type="text" size="small" status="danger">
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <PageContainer>
      <Card bordered={false}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {/* Search and Actions */}
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <Space>
              <SearchToolbar
                placeholder="搜索指标源名称"
                onSearch={handleSearch}
                style={{ width: 300 }}
              />
              <Select
                placeholder="指标源类型"
                style={{ width: 150 }}
                allowClear
                value={filters.type}
                onChange={(value) => handleFilterChange('type', value)}
              >
                <Select.Option value="prometheus">Prometheus</Select.Option>
                <Select.Option value="victoria-metrics">
                  VictoriaMetrics
                </Select.Option>
              </Select>
              <Select
                placeholder="状态"
                style={{ width: 120 }}
                allowClear
                value={filters.status}
                onChange={(value) => handleFilterChange('status', value)}
              >
                <Select.Option value="active">活跃</Select.Option>
                <Select.Option value="inactive">停用</Select.Option>
              </Select>
            </Space>
            <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
              新建指标源
            </Button>
          </div>

          {/* Table */}
          <AppTable
            columns={columns}
            data={dataSource}
            loading={loading}
            pagination={{
              ...pagination,
              total,
              onChange: (current, pageSize) =>
                setPagination({ current, pageSize }),
            }}
          />
        </Space>
      </Card>
    </PageContainer>
  );
};

export default MetricSourceList;
