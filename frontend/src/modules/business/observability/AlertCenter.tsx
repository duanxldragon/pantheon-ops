import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, InputNumber, Message, Popconfirm, Select, Space, Switch, Tabs, Tag } from '@arco-design/web-react';
import { IconDelete, IconPlus } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import { AppModal, AppTable, GovernanceSummaryBar, PageContainer, PageEmpty, PageError, PageLoading, SearchToolbar } from '../../../components';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import {
  createAlertRule,
  createNotificationChannel,
  deleteAlertRule,
  deleteNotificationChannel,
  getAlertEventList,
  getAlertRuleList,
  getDatasourceList,
  getNotificationChannelList,
  toggleAlertRule,
  type AlertEventRow,
  type AlertRuleRow,
  type DatasourceRow,
  type NotificationChannelRow,
} from './api';
import '../../system/components/shared/list-page.css';

const { TabPane } = Tabs;
const FormItem = Form.Item;

interface RuleFormValues {
  name: string;
  datasourceId: number;
  query: string;
  interval: number;
  forDuration: number;
  severity: 'critical' | 'warning' | 'info';
  notificationChannels?: number[];
}

interface ChannelFormValues {
  name: string;
  type: 'webhook' | 'slack' | 'dingtalk' | 'feishu';
  webhookUrl: string;
  secret?: string;
}

export default function AlertCenter() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [rules, setRules] = useState<AlertRuleRow[]>([]);
  const [events, setEvents] = useState<AlertEventRow[]>([]);
  const [channels, setChannels] = useState<NotificationChannelRow[]>([]);
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [ruleVisible, setRuleVisible] = useState(false);
  const [channelVisible, setChannelVisible] = useState(false);
  const [ruleForm] = Form.useForm<RuleFormValues>();
  const [channelForm] = Form.useForm<ChannelFormValues>();

  const canCreateRule = hasPerm('business:observability:alert_rule:create');
  const canUpdateRule = hasPerm('business:observability:alert_rule:update');
  const canDeleteRule = hasPerm('business:observability:alert_rule:delete');
  const canManageChannels = hasPerm('business:observability:notification:update');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [ruleResponse, eventResponse, channelResponse, datasourceResponse] = await Promise.all([
        getAlertRuleList({ page: 1, pageSize: 100 }),
        getAlertEventList({ page: 1, pageSize: 100 }),
        getNotificationChannelList(),
        getDatasourceList(),
      ]);
      setRules(ruleResponse.items);
      setEvents(eventResponse.items);
      setChannels(channelResponse.items);
      setDatasources(datasourceResponse.items.filter((item) => item.type === 'prometheus' || item.type === 'victoriametrics'));
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  const submitRule = useCallback(async () => {
    const values = await ruleForm.validate();
    setSubmitting(true);
    try {
      await createAlertRule({ ...values, enabled: true, labels: {}, annotations: {}, notificationChannels: values.notificationChannels ?? [] });
      Message.success(t('business.observability.alertCenter.ruleCreateSuccess'));
      setRuleVisible(false);
      ruleForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.alertCenter.ruleCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [load, ruleForm, t]);

  const submitChannel = useCallback(async () => {
    const values = await channelForm.validate();
    const config = values.type === 'webhook'
      ? { url: values.webhookUrl, method: 'POST', timeout: 10 }
      : { webhookUrl: values.webhookUrl, secret: values.secret || '' };
    setSubmitting(true);
    try {
      await createNotificationChannel({ name: values.name, type: values.type, enabled: true, config });
      Message.success(t('business.observability.alertCenter.channelCreateSuccess'));
      setChannelVisible(false);
      channelForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.alertCenter.channelCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [channelForm, load, t]);

  const ruleColumns = useMemo<ColumnProps<AlertRuleRow>[]>(() => [
    { title: t('observability.alert_rule.name'), dataIndex: 'name', width: 200 },
    { title: t('observability.alert_rule.metric_source'), dataIndex: 'datasourceName', width: 180 },
    { title: t('observability.alert_rule.promql'), dataIndex: 'query', ellipsis: true },
    { title: t('observability.alert_rule.severity'), dataIndex: 'severity', width: 110, render: (value: string) => <Tag color={value === 'critical' ? 'red' : value === 'warning' ? 'orange' : 'arcoblue'}>{value}</Tag> },
    { title: t('observability.alert_rule.status'), dataIndex: 'enabled', width: 90, render: (enabled: boolean, row) => <Switch checked={enabled} disabled={!canUpdateRule} onChange={async (checked) => { await toggleAlertRule(row.id, checked); await load(); }} /> },
    { title: t('business.observability.alertCenter.firedCount'), dataIndex: 'firedCount', width: 90 },
    { title: t('common.action'), width: 90, fixed: 'right', render: (_: unknown, row) => canDeleteRule ? <Popconfirm title={t('observability.alert_rule.delete_confirm')} onOk={async () => { await deleteAlertRule(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null },
  ], [canDeleteRule, canUpdateRule, load, t]);

  const eventColumns = useMemo<ColumnProps<AlertEventRow>[]>(() => [
    { title: t('observability.alert_record.rule_name'), dataIndex: 'ruleName', width: 220 },
    { title: t('observability.alert_record.severity'), dataIndex: 'severity', width: 110, render: (value: string) => <Tag color={value === 'critical' ? 'red' : value === 'warning' ? 'orange' : 'arcoblue'}>{value}</Tag> },
    { title: t('observability.alert_record.status'), dataIndex: 'status', width: 110, render: (value: string) => <Tag color={value === 'firing' ? 'red' : 'green'}>{value}</Tag> },
    { title: t('observability.alert_record.fired_at'), dataIndex: 'firedAt', width: 190, render: (value: string) => new Date(value).toLocaleString() },
    { title: t('observability.alert_record.resolved_at'), dataIndex: 'resolvedAt', width: 190, render: (value?: string) => value ? new Date(value).toLocaleString() : '-' },
  ], [t]);

  const channelColumns = useMemo<ColumnProps<NotificationChannelRow>[]>(() => [
    { title: t('business.observability.alertCenter.field.name'), dataIndex: 'name', width: 220 },
    { title: t('business.observability.alertCenter.field.type'), dataIndex: 'type', width: 130, render: (value: string) => <Tag color="cyan">{value}</Tag> },
    { title: t('business.observability.alertCenter.field.status'), dataIndex: 'status', width: 120, render: (value: string) => <Tag color={value === 'active' ? 'green' : 'gray'}>{value}</Tag> },
    { title: t('common.action'), width: 90, fixed: 'right', render: (_: unknown, row) => canManageChannels ? <Popconfirm title={t('business.observability.alertCenter.channelDeleteConfirm')} onOk={async () => { await deleteNotificationChannel(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null },
  ], [canManageChannels, load, t]);

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.observability.alertCenter.hero.eyebrow')} title={t('operations.observability.alertCenter.menu')} description={t('business.observability.alertCenter.hero.title')} metrics={[
          { key: 'rules', label: t('business.observability.alertCenter.hero.rules'), value: rules.length },
          { key: 'active', label: t('business.observability.alertCenter.hero.active'), value: rules.filter((item) => item.enabled).length },
          { key: 'firing', label: t('business.observability.alertCenter.hero.firing'), value: events.filter((item) => item.status === 'firing').length },
          { key: 'channels', label: t('business.observability.alertCenter.hero.channels'), value: channels.length },
        ]} />
        {loading && rules.length === 0 && events.length === 0 ? <PageLoading /> : null}
        {error && rules.length === 0 && events.length === 0 ? <PageError onRetry={() => void load()} /> : null}
        {!error ? <Tabs defaultActiveTab="rules">
          <TabPane key="rules" title={t('business.observability.alertCenter.rulesTab')}><SearchToolbar onClearAll={() => undefined} extra={canCreateRule ? <Button type="primary" icon={<IconPlus />} disabled={datasources.length === 0} onClick={() => setRuleVisible(true)}>{t('business.observability.alertCenter.addRule')}</Button> : undefined} /><Card className="page-panel system-list__table-card">{!loading && rules.length === 0 ? <PageEmpty description={t('business.observability.alertRule.empty')} /> : <AppTable rowKey="id" columns={ruleColumns} data={rules} pagination={false} scroll={{ x: 'max-content' }} />}</Card></TabPane>
          <TabPane key="events" title={t('business.observability.alertCenter.eventsTab')}><Card className="page-panel system-list__table-card">{!loading && events.length === 0 ? <PageEmpty description={t('business.observability.alertRecord.empty')} /> : <AppTable rowKey="id" columns={eventColumns} data={events} pagination={false} scroll={{ x: 'max-content' }} />}</Card></TabPane>
          <TabPane key="channels" title={t('business.observability.alertCenter.channelsTab')}><SearchToolbar onClearAll={() => undefined} extra={canManageChannels ? <Button type="primary" icon={<IconPlus />} onClick={() => setChannelVisible(true)}>{t('business.observability.alertCenter.addChannel')}</Button> : undefined} /><Card className="page-panel system-list__table-card">{!loading && channels.length === 0 ? <PageEmpty description={t('business.observability.alertCenter.channelEmpty')} /> : <AppTable rowKey="id" columns={channelColumns} data={channels} pagination={false} />}</Card></TabPane>
        </Tabs> : null}
      </Space>

      <AppModal visible={ruleVisible} title={t('business.observability.alertCenter.addRule')} confirmLoading={submitting} onOk={() => void submitRule()} onCancel={() => setRuleVisible(false)}><Form form={ruleForm} layout="vertical" initialValues={{ interval: 60, forDuration: 300, severity: 'warning' }}><FormItem field="name" label={t('observability.alert_rule.name')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="datasourceId" label={t('observability.alert_rule.metric_source')} rules={[{ required: true }]}><Select>{datasources.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem><FormItem field="query" label="PromQL" rules={[{ required: true }]}><Input.TextArea autoSize={{ minRows: 3, maxRows: 6 }} /></FormItem><FormItem field="severity" label={t('observability.alert_rule.severity')} rules={[{ required: true }]}><Select><Select.Option value="critical">critical</Select.Option><Select.Option value="warning">warning</Select.Option><Select.Option value="info">info</Select.Option></Select></FormItem><FormItem field="interval" label={t('business.observability.alertCenter.field.interval')}><InputNumber min={10} /></FormItem><FormItem field="forDuration" label={t('business.observability.alertCenter.field.forDuration')}><InputNumber min={0} /></FormItem><FormItem field="notificationChannels" label={t('business.observability.alertCenter.channelsTab')}><Select mode="multiple">{channels.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem></Form></AppModal>
      <AppModal visible={channelVisible} title={t('business.observability.alertCenter.addChannel')} confirmLoading={submitting} onOk={() => void submitChannel()} onCancel={() => setChannelVisible(false)}><Form form={channelForm} layout="vertical" initialValues={{ type: 'webhook' }}><FormItem field="name" label={t('business.observability.alertCenter.field.name')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="type" label={t('business.observability.alertCenter.field.type')} rules={[{ required: true }]}><Select><Select.Option value="webhook">Webhook</Select.Option><Select.Option value="slack">Slack</Select.Option><Select.Option value="dingtalk">DingTalk</Select.Option><Select.Option value="feishu">Feishu</Select.Option></Select></FormItem><FormItem field="webhookUrl" label="Webhook URL" rules={[{ required: true, type: 'url' }]}><Input /></FormItem><FormItem field="secret" label={t('business.observability.alertCenter.field.secret')}><Input.Password /></FormItem></Form></AppModal>
    </PageContainer>
  );
}
