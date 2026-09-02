import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Card,
  Message,
  Popconfirm,
  Space,
  Tabs,
  Tag,
  Switch,
} from '@arco-design/web-react';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { IconPlus, IconDelete, IconEdit, IconSync, IconRefresh } from '@arco-design/web-react/icon';
import {
  AppTable,
  SearchToolbar,
  GovernanceSummaryBar,
  PageContainer,
  PageError,
  PageLoading,
} from '../../../components';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import {
  getDatasourceList,
  getAlertRuleList,
  getAlertEventList,
  getNotificationChannelList,
  deleteDatasource,
  deleteAlertRule,
  toggleAlertRule,
  deleteNotificationChannel,
  testDatasourceConnection,
  type DatasourceRow,
  type AlertRuleRow,
  type AlertEventRow,
  type NotificationChannelRow,
} from './api';

const { TabPane } = Tabs;

const AlertManagement = () => {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();

  const [activeTab, setActiveTab] = useState('datasources');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  // Data states
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [alertRules, setAlertRules] = useState<AlertRuleRow[]>([]);
  const [alertEvents, setAlertEvents] = useState<AlertEventRow[]>([]);
  const [channels, setChannels] = useState<NotificationChannelRow[]>([]);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [dsResult, rulesResult, eventsResult, channelsResult] = await Promise.all([
        getDatasourceList(),
        getAlertRuleList({}),
        getAlertEventList({ page: 1, pageSize: 50 }),
        getNotificationChannelList(),
      ]);
      setDatasources(dsResult.items);
      setAlertRules(rulesResult.items);
      setAlertEvents(eventsResult.items);
      setChannels(channelsResult.items);
    } catch (err) {
      setError(err as Error);
      Message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    const timeoutId = window.setTimeout(() => { void loadData(); }, 0);
    return () => window.clearTimeout(timeoutId);
  }, [loadData]);

  const heroMetrics = [
    { key: 'datasources', label: t('business.observability.alertManagement.datasources'), value: datasources.length },
    { key: 'rules', label: t('business.observability.alertManagement.rules'), value: alertRules.length },
    { key: 'activeRules', label: t('business.observability.alertManagement.activeRules'), value: alertRules.filter(r => r.enabled).length },
    { key: 'channels', label: t('business.observability.alertManagement.channels'), value: channels.length },
  ];

  // Datasource columns
  const datasourceColumns: ColumnProps<DatasourceRow>[] = [
    { title: 'Name', dataIndex: 'name', width: 200 },
    { title: 'Type', dataIndex: 'type', width: 120 },
    { title: 'URL', dataIndex: 'url', ellipsis: true },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'healthy' ? 'green' : status === 'unhealthy' ? 'red' : 'gray'}>
          {status}
        </Tag>
      ),
    },
    {
      title: 'Default',
      dataIndex: 'isDefault',
      width: 80,
      render: (isDefault: boolean) => (isDefault ? <Tag color="blue">Default</Tag> : null),
    },
    {
      title: 'Actions',
      width: 200,
      render: (_, row) => (
        <Space>
          <Button
            size="small"
            icon={<IconSync />}
            onClick={() => handleTestConnection(row.id)}
          >
            Test
          </Button>
          <Button size="small" icon={<IconEdit />}>Edit</Button>
          <Popconfirm
            title="Delete datasource?"
            onOk={() => handleDeleteDatasource(row.id)}
          >
            <Button size="small" status="danger" icon={<IconDelete />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // Alert Rule columns
  const alertRuleColumns: ColumnProps<AlertRuleRow>[] = [
    { title: 'Name', dataIndex: 'name', width: 200 },
    {
      title: 'Severity',
      dataIndex: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severity === 'critical' ? 'red' : severity === 'warning' ? 'orange' : 'blue'}>
          {severity}
        </Tag>
      ),
    },
    { title: 'Query', dataIndex: 'query', ellipsis: true },
    {
      title: 'Interval',
      dataIndex: 'interval',
      width: 100,
      render: (interval: number) => `${interval}s`,
    },
    {
      title: 'Enabled',
      dataIndex: 'enabled',
      width: 80,
      render: (enabled: boolean, row) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggleRule(row.id, checked)}
        />
      ),
    },
    {
      title: 'Fired',
      dataIndex: 'firedCount',
      width: 80,
    },
    {
      title: 'Actions',
      width: 150,
      render: (_, row) => (
        <Space>
          <Button size="small" icon={<IconEdit />}>Edit</Button>
          <Popconfirm
            title="Delete alert rule?"
            onOk={() => handleDeleteRule(row.id)}
          >
            <Button size="small" status="danger" icon={<IconDelete />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  // Alert Event columns
  const alertEventColumns: ColumnProps<AlertEventRow>[] = [
    {
      title: 'Time',
      dataIndex: 'firedAt',
      width: 180,
      render: (time: string) => new Date(time).toLocaleString(),
    },
    { title: 'Rule', dataIndex: 'ruleName', width: 200 },
    {
      title: 'Severity',
      dataIndex: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severity === 'critical' ? 'red' : severity === 'warning' ? 'orange' : 'blue'}>
          {severity}
        </Tag>
      ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'firing' ? 'red' : 'green'}>{status}</Tag>
      ),
    },
    { title: 'Value', dataIndex: 'value', width: 100 },
    {
      title: 'Resolved',
      dataIndex: 'resolvedAt',
      width: 180,
      render: (time: string) => (time ? new Date(time).toLocaleString() : '-'),
    },
  ];

  // Notification Channel columns
  const channelColumns: ColumnProps<NotificationChannelRow>[] = [
    { title: 'Name', dataIndex: 'name', width: 200 },
    {
      title: 'Type',
      dataIndex: 'type',
      width: 120,
      render: (type: string) => <Tag>{type}</Tag>,
    },
    {
      title: 'Enabled',
      dataIndex: 'enabled',
      width: 80,
      render: (enabled: boolean) => (enabled ? <Tag color="green">Yes</Tag> : <Tag>No</Tag>),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => <Tag color={status === 'active' ? 'green' : 'gray'}>{status}</Tag>,
    },
    {
      title: 'Actions',
      width: 150,
      render: (_, row) => (
        <Space>
          <Button size="small" icon={<IconEdit />}>Edit</Button>
          <Popconfirm
            title="Delete channel?"
            onOk={() => handleDeleteChannel(row.id)}
          >
            <Button size="small" status="danger" icon={<IconDelete />}>
              Delete
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const handleTestConnection = async (id: number) => {
    try {
      await testDatasourceConnection(id);
      Message.success(t('business.observability.alertManagement.connectionPassed'));
      await loadData();
    } catch {
      Message.error(t('business.observability.alertManagement.connectionFailed'));
    }
  };

  const handleDeleteDatasource = async (id: number) => {
    try {
      await deleteDatasource(id);
      Message.success('Deleted');
      await loadData();
    } catch {
      Message.error(t('common.deleteFailed'));
    }
  };

  const handleToggleRule = async (id: number, enabled: boolean) => {
    try {
      await toggleAlertRule(id, enabled);
      Message.success('Updated');
      await loadData();
    } catch {
      Message.error(t('common.updateFailed'));
    }
  };

  const handleDeleteRule = async (id: number) => {
    try {
      await deleteAlertRule(id);
      Message.success('Deleted');
      await loadData();
    } catch {
      Message.error(t('common.deleteFailed'));
    }
  };

  const handleDeleteChannel = async (id: number) => {
    try {
      await deleteNotificationChannel(id);
      Message.success('Deleted');
      await loadData();
    } catch {
      Message.error(t('common.deleteFailed'));
    }
  };

  if (loading && datasources.length === 0) {
    return <PageLoading />;
  }

  if (error) {
    return <PageError description={error.message} onRetry={loadData} />;
  }

  return (
    <PageContainer title="Alert Management">
      <GovernanceSummaryBar metrics={heroMetrics} />

      <Card style={{ marginTop: 16 }}>
        <Tabs activeTab={activeTab} onChange={setActiveTab}>
          <TabPane key="datasources" title="Data Sources">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <SearchToolbar
                onClearAll={() => undefined}
                extra={
                  <Space>
                    <Button icon={<IconRefresh />} onClick={() => void loadData()}>Refresh</Button>
                    {hasPerm('observability:datasource:create') && (
                      <Button type="primary" icon={<IconPlus />}>
                        Add Datasource
                      </Button>
                    )}
                  </Space>
                }
              />
              <AppTable
                columns={datasourceColumns}
                data={datasources}
                rowKey="id"
                pagination={false}
              />
            </Space>
          </TabPane>

          <TabPane key="rules" title="Alert Rules">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <SearchToolbar
                onClearAll={() => undefined}
                extra={
                  <Space>
                    <Button icon={<IconRefresh />} onClick={() => void loadData()}>Refresh</Button>
                    {hasPerm('observability:rule:create') && (
                      <Button type="primary" icon={<IconPlus />}>
                        Create Rule
                      </Button>
                    )}
                  </Space>
                }
              />
              <AppTable
                columns={alertRuleColumns}
                data={alertRules}
                rowKey="id"
                pagination={{ pageSize: 20 }}
              />
            </Space>
          </TabPane>

          <TabPane key="events" title="Alert Events">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <SearchToolbar onClearAll={() => undefined} extra={<Button icon={<IconRefresh />} onClick={() => void loadData()}>Refresh</Button>} />
              <AppTable
                columns={alertEventColumns}
                data={alertEvents}
                rowKey="id"
                pagination={{ pageSize: 50 }}
              />
            </Space>
          </TabPane>

          <TabPane key="channels" title="Notification Channels">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <SearchToolbar
                onClearAll={() => undefined}
                extra={
                  <Space>
                    <Button icon={<IconRefresh />} onClick={() => void loadData()}>Refresh</Button>
                    {hasPerm('observability:channel:create') && (
                      <Button type="primary" icon={<IconPlus />}>
                        Add Channel
                      </Button>
                    )}
                  </Space>
                }
              />
              <AppTable
                columns={channelColumns}
                data={channels}
                rowKey="id"
                pagination={false}
              />
            </Space>
          </TabPane>
        </Tabs>
      </Card>
    </PageContainer>
  );
};

export default AlertManagement;
