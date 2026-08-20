import { useState, useEffect, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Card,
  Button,
  Tag,
  Space,
  Message,
  Statistic,
  Empty,
  List,
} from '@arco-design/web-react';
import { IconRefresh, IconNotification, IconCheckCircleFill } from '@arco-design/web-react/icon';
import { PageContainer } from '../../../../components';
import { observabilityApi } from '../api';
import type { AlertRecord } from '../api';
import { formatDateTime } from '../../../../core/format/dateTime';
import '../../../system/components/shared/list-page.css';

const ActiveAlertList = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<AlertRecord[]>([]);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const { data } = await observabilityApi.getActiveAlerts();
      setDataSource(data.data);
    } catch (error) {
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    fetchData();

    // Auto refresh every 30 seconds
    const interval = setInterval(fetchData, 30000);
    return () => clearInterval(interval);
  }, [fetchData]);

  const calculateDuration = (firedAt: string): string => {
    const start = new Date(firedAt).getTime();
    const now = Date.now();
    const diff = now - start;

    const hours = Math.floor(diff / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((diff % (1000 * 60)) / 1000);

    if (hours > 0) {
      return `${hours}小时${minutes}分钟`;
    } else if (minutes > 0) {
      return `${minutes}分${seconds}秒`;
    } else {
      return `${seconds}秒`;
    }
  };

  return (
    <PageContainer>
      <Card bordered={false}>
        <Space direction="vertical" size={16} style={{ width: '100%' }}>
          {/* Header */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Statistic
              title={
                <Space>
                  <IconNotification style={{ color: '#ff4d4f', fontSize: 20 }} />
                  <span style={{ fontSize: 16, fontWeight: 500 }}>活跃告警</span>
                </Space>
              }
              value={dataSource.length}
              suffix="个"
              valueStyle={{ color: dataSource.length > 0 ? '#ff4d4f' : '#00b42a' }}
            />
            <Button icon={<IconRefresh />} onClick={fetchData}>
              刷新
            </Button>
          </div>

          {/* Alert List */}
          {!loading && dataSource.length > 0 && (
            <List
              dataSource={dataSource}
              render={(item: AlertRecord, index) => (
                <List.Item key={index}>
                  <List.Item.Meta
                    title={
                      <Space>
                        {item.severity === 'critical' && <Tag color="red">严重</Tag>}
                        {item.severity === 'warning' && <Tag color="orange">警告</Tag>}
                        {item.severity === 'info' && <Tag color="blue">信息</Tag>}
                        <span>{item.alertRuleName}</span>
                      </Space>
                    }
                    description={
                      <div>
                        <div>触发时间: {formatDateTime(item.firedAt)}</div>
                        <div>持续时长: {calculateDuration(item.firedAt)}</div>
                        {item.annotations && (
                          <div style={{ marginTop: 8, color: '#4e5969' }}>
                            {(item.annotations as any).summary || (item.annotations as any).description}
                          </div>
                        )}
                      </div>
                    }
                  />
                </List.Item>
              )}
            />
          )}

          {/* Empty State */}
          {!loading && dataSource.length === 0 && (
            <Empty
              description="暂无活跃告警"
              icon={<IconCheckCircleFill style={{ fontSize: 64, color: '#00b42a' }} />}
            />
          )}
        </Space>
      </Card>
    </PageContainer>
  );
};

export default ActiveAlertList;
