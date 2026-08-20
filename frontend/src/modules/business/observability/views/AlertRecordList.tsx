import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Space,
  Select,
  Message,
  DatePicker,
  Tag,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppTable,
  buildStandardPagination,
  PageContainer,
} from '../../../../components';
import { observabilityApi } from '../api';
import type { AlertRecord, PaginationParams } from '../api';
import { formatDateTime } from '../../../../core/format/dateTime';
import '../../../system/components/shared/list-page.css';

const AlertRecordList = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<AlertRecord[]>([]);
  const [total, setTotal] = useState(0);
  const [pagination, setPagination] = useState(buildStandardPagination());
  const [filters, setFilters] = useState<{
    severity?: string;
    resolved?: boolean;
  }>({});

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params: PaginationParams = {
        page: pagination.current,
        pageSize: pagination.pageSize,
        severity: filters.severity,
      };

      const { data } = await observabilityApi.getAlertRecords(params);
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

  const handleFilterChange = (key: string, value: string | boolean | undefined) => {
    setFilters({ ...filters, [key]: value });
    setPagination({ ...pagination, current: 1 });
  };

  const calculateDuration = (firedAt: string, resolvedAt?: string): string => {
    const start = new Date(firedAt).getTime();
    const end = resolvedAt ? new Date(resolvedAt).getTime() : Date.now();
    const diff = end - start;

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));

    if (hours > 0) {
      return `${hours}小时${minutes}分钟`;
    }
    return `${minutes}分钟`;
  };

  const columns: ColumnProps<AlertRecord>[] = [
    {
      title: '告警规则',
      dataIndex: 'alertRuleName',
      width: 200,
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
      title: '触发时间',
      dataIndex: 'firedAt',
      width: 180,
      render: (firedAt: string) => formatDateTime(firedAt),
    },
    {
      title: '恢复时间',
      dataIndex: 'resolvedAt',
      width: 180,
      render: (resolvedAt?: string) => resolvedAt ? formatDateTime(resolvedAt) : '-',
    },
    {
      title: '持续时长',
      width: 120,
      render: (_, record) => calculateDuration(record.firedAt, record.resolvedAt),
    },
    {
      title: '状态',
      width: 100,
      render: (_, record) => {
        const isResolved = !!record.resolvedAt;
        return (
          <Tag color={isResolved ? 'green' : 'red'}>
            {isResolved ? '已恢复' : '活跃'}
          </Tag>
        );
      },
    },
  ];

  return (
    <PageContainer>
      <Card bordered={false}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {/* Filters */}
          <Space>
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
              value={filters.resolved}
              onChange={(value) => handleFilterChange('resolved', value)}
            >
              <Select.Option value={false}>活跃</Select.Option>
              <Select.Option value={true}>已恢复</Select.Option>
            </Select>
          </Space>

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

export default AlertRecordList;
