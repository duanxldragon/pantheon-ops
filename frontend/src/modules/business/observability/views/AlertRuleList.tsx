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
import { IconPlus } from '@arco-design/web-react/icon';
import {
  AppTable,
  buildStandardPagination,
  SearchToolbar,
  PageContainer,
} from '../../../../components';
import { observabilityApi } from '../api';
import type { AlertRule, PaginationParams } from '../api';
import { formatDateTime } from '../../../../core/format/dateTime';
import '../../../system/components/shared/list-page.css';

const AlertRuleList = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<AlertRule[]>([]);
  const [total, setTotal] = useState(0);
  const [pagination, setPagination] = useState(buildStandardPagination());
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filters, setFilters] = useState<{
    environment?: string;
    severity?: string;
    status?: string;
  }>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: PaginationParams = {
        page: pagination.current,
        pageSize: pagination.pageSize,
        environment: filters.environment,
        severity: filters.severity,
        status: filters.status,
      };

      const { data } = await observabilityApi.getAlertRules(params);
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
    Message.info('新建告警规则功能开发中（Week 3）');
  };

  const handleEdit = (record: AlertRule) => {
    Message.info(`编辑告警规则: ${record.name}（Week 3）`);
  };

  const handleToggleStatus = async (record: AlertRule) => {
    const newStatus = record.status === 'enabled' ? 'disabled' : 'enabled';
    try {
      await observabilityApi.updateAlertRule(record.id, { status: newStatus });
      Message.success(newStatus === 'enabled' ? '已启用' : '已停用');
      fetchData();
    } catch (error) {
      Message.error('操作失败');
    }
  };

  const handleDelete = async (id: number) => {
    try {
      await observabilityApi.deleteAlertRule(id);
      Message.success(t('common.deleteSuccess'));
      fetchData();
    } catch (error) {
      Message.error(t('common.deleteFailed'));
    }
  };

  const columns: ColumnProps<AlertRule>[] = [
    {
      title: '规则名称',
      dataIndex: 'name',
      width: 200,
    },
    {
      title: '环境',
      dataIndex: 'environment',
      width: 100,
      render: (env: string) => {
        const colorMap: Record<string, string> = {
          prod: 'red',
          test: 'orange',
          dev: 'blue',
        };
        const labelMap: Record<string, string> = {
          prod: '生产',
          test: '测试',
          dev: '开发',
        };
        return env ? <Tag color={colorMap[env]}>{labelMap[env]}</Tag> : '-';
      },
    },
    {
      title: 'PromQL',
      dataIndex: 'promql',
      ellipsis: true,
    },
    {
      title: '严重性',
      dataIndex: 'severity',
      width: 100,
      render: (severity: string) => {
        const colorMap: Record<string, string> = {
          critical: 'red',
          warning: 'orange',
          info: 'blue',
        };
        const labelMap: Record<string, string> = {
          critical: '严重',
          warning: '警告',
          info: '信息',
        };
        return <Tag color={colorMap[severity]}>{labelMap[severity]}</Tag>;
      },
    },
    {
      title: '持续时间',
      dataIndex: 'duration',
      width: 100,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const colorMap: Record<string, string> = {
          enabled: 'green',
          disabled: 'gray',
        };
        const labelMap: Record<string, string> = {
          enabled: '启用',
          disabled: '停用',
        };
        return <Tag color={colorMap[status]}>{labelMap[status]}</Tag>;
      },
    },
    {
      title: '操作',
      width: 200,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space>
          <Button type="text" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button
            type="text"
            size="small"
            status={record.status === 'enabled' ? 'warning' : 'success'}
            onClick={() => handleToggleStatus(record)}
          >
            {record.status === 'enabled' ? '停用' : '启用'}
          </Button>
          <Popconfirm
            title="确定删除此告警规则吗？"
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
                placeholder="搜索告警规则"
                onSearch={handleSearch}
                style={{ width: 300 }}
              />
              <Select
                placeholder="环境"
                style={{ width: 120 }}
                allowClear
                value={filters.environment}
                onChange={(value) => handleFilterChange('environment', value)}
              >
                <Select.Option value="prod">生产</Select.Option>
                <Select.Option value="test">测试</Select.Option>
                <Select.Option value="dev">开发</Select.Option>
              </Select>
              <Select
                placeholder="严重性"
                style={{ width: 120 }}
                allowClear
                value={filters.severity}
                onChange={(value) => handleFilterChange('severity', value)}
              >
                <Select.Option value="critical">严重</Select.Option>
                <Select.Option value="warning">警告</Select.Option>
                <Select.Option value="info">信息</Select.Option>
              </Select>
              <Select
                placeholder="状态"
                style={{ width: 120 }}
                allowClear
                value={filters.status}
                onChange={(value) => handleFilterChange('status', value)}
              >
                <Select.Option value="enabled">启用</Select.Option>
                <Select.Option value="disabled">停用</Select.Option>
              </Select>
            </Space>
            <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
              新建告警规则
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

export default AlertRuleList;
