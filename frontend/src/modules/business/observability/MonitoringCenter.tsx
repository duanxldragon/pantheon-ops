import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, InputNumber, Message, Popconfirm, Select, Space, Switch, Tabs, Tag } from '@arco-design/web-react';
import { IconDelete, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import {
  AppModal,
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
} from '../../../components';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { getBizScopeOptions, type BizScopeOptionItem } from '../bizscope/api';
import {
  createDatasource,
  createMonitoringTarget,
  deleteDatasource,
  deleteMonitoringTarget,
  getDatasourceList,
  getMonitoringTargetList,
  testDatasourceConnection,
  type DatasourcePayload,
  type DatasourceRow,
  type MonitoringTargetRow,
} from './api';
import PasswordInput from '../../../components/patterns/forms/PasswordInput';
import '../../../assets/list-page.css';

const { TabPane } = Tabs;
const FormItem = Form.Item;

interface TargetFormValues {
  businessScopeId: number;
  targetType: 'host' | 'k8s_cluster' | 'other';
  targetName: string;
  endpoint?: string;
  metricsPath?: string;
  remark?: string;
}

function datasourceTypeColor(value: string): string {
  if (value === 'loki') {
    return 'cyan';
  }
  if (value === 'tempo') {
    return 'purple';
  }
  return 'arcoblue';
}

function datasourceStatusColor(value: string): string {
  if (value === 'healthy') {
    return 'green';
  }
  if (value === 'unhealthy') {
    return 'red';
  }
  return 'gray';
}

export default function MonitoringCenter() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [targets, setTargets] = useState<MonitoringTargetRow[]>([]);
  const [datasources, setDatasources] = useState<DatasourceRow[]>([]);
  const [bizScopes, setBizScopes] = useState<BizScopeOptionItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [targetVisible, setTargetVisible] = useState(false);
  const [datasourceVisible, setDatasourceVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [targetForm] = Form.useForm<TargetFormValues>();
  const [datasourceForm] = Form.useForm<DatasourcePayload>();

  const canCreate = hasPerm('business:observability:metric_source:create');
  const canDelete = hasPerm('business:observability:metric_source:delete');
  const canTest = hasPerm('business:observability:metric_source:update');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [targetResponse, datasourceResponse, scopeResponse] = await Promise.all([
        getMonitoringTargetList(),
        getDatasourceList(),
        getBizScopeOptions(),
      ]);
      setTargets(targetResponse.items);
      setDatasources(datasourceResponse.items);
      setBizScopes(scopeResponse);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  const submitTarget = useCallback(async () => {
    const values = await targetForm.validate();
    setSubmitting(true);
    try {
      await createMonitoringTarget({ ...values, metricsPath: values.metricsPath || '/metrics' });
      Message.success(t('business.observability.monitoring.targetCreateSuccess'));
      setTargetVisible(false);
      targetForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.monitoring.targetCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [load, t, targetForm]);

  const submitDatasource = useCallback(async () => {
    const values = await datasourceForm.validate();
    setSubmitting(true);
    try {
      await createDatasource({ ...values, accessMode: 'proxy', timeout: values.timeout || 30 });
      Message.success(t('business.observability.monitoring.datasourceCreateSuccess'));
      setDatasourceVisible(false);
      datasourceForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.observability.monitoring.datasourceCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [datasourceForm, load, t]);

  const targetColumns = useMemo<ColumnProps<MonitoringTargetRow>[]>(
    () => [
      { title: t('observability.metric_source.name'), dataIndex: 'targetName', width: 200 },
      { title: t('observability.metric_source.type'), dataIndex: 'targetType', width: 140 },
      { title: t('observability.metric_source.endpoint'), dataIndex: 'endpoint', width: 260, ellipsis: true },
      { title: t('business.observability.metricSource.metricsPath'), dataIndex: 'metricsPath', width: 150 },
      { title: t('observability.metric_source.status'), dataIndex: 'status', width: 110, render: (_: unknown, row) => <Tag color={row.status === 'up' ? 'green' : 'orange'}>{row.status}</Tag> },
      {
        title: t('common.action'),
        width: 90,
        fixed: 'right',
        render: (_: unknown, row) => canDelete ? (
          <Popconfirm title={t('observability.metric_source.delete_confirm')} onOk={async () => { await deleteMonitoringTarget(row.id); await load(); }}>
            <Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button>
          </Popconfirm>
        ) : null,
      },
    ],
    [canDelete, load, t],
  );

  const datasourceColumns = useMemo<ColumnProps<DatasourceRow>[]>(
    () => [
      { title: t('business.observability.monitoring.field.name'), dataIndex: 'name', width: 180 },
      { title: t('business.observability.monitoring.field.type'), dataIndex: 'type', width: 140, render: (value: string) => <Tag color={datasourceTypeColor(value)}>{value}</Tag> },
      { title: t('business.observability.monitoring.field.url'), dataIndex: 'url', width: 300, ellipsis: true },
      { title: t('business.observability.monitoring.field.status'), dataIndex: 'status', width: 120, render: (value: string) => <Tag color={datasourceStatusColor(value)}>{value}</Tag> },
      { title: t('business.observability.monitoring.field.default'), dataIndex: 'isDefault', width: 100, render: (value: boolean) => value ? <Tag color="arcoblue">{t('common.yes')}</Tag> : '-' },
      {
        title: t('common.action'),
        width: 180,
        fixed: 'right',
        render: (_: unknown, row) => (
          <Space>
            {canTest ? <Button type="text" icon={<IconRefresh />} onClick={async () => { await testDatasourceConnection(row.id); await load(); }}>{t('business.observability.monitoring.test')}</Button> : null}
            {canDelete ? (
              <Popconfirm title={t('business.observability.monitoring.datasourceDeleteConfirm')} onOk={async () => { await deleteDatasource(row.id); await load(); }}>
                <Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button>
              </Popconfirm>
            ) : null}
          </Space>
        ),
      },
    ],
    [canDelete, canTest, load, t],
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar
          eyebrow={t('business.observability.monitoring.hero.eyebrow')}
          title={t('operations.observability.monitoring.menu')}
          description={t('business.observability.monitoring.hero.title')}
          metrics={[
            { key: 'targets', label: t('business.observability.monitoring.hero.targets'), value: targets.length },
            { key: 'healthy', label: t('business.observability.monitoring.hero.healthy'), value: targets.filter((item) => item.status === 'up').length },
            { key: 'datasources', label: t('business.observability.monitoring.hero.datasources'), value: datasources.length },
          ]}
        />
        {loading && targets.length === 0 && datasources.length === 0 ? <PageLoading /> : null}
        {error && targets.length === 0 && datasources.length === 0 ? <PageError onRetry={() => void load()} /> : null}
        {!error ? (
          <Tabs defaultActiveTab="targets">
            <TabPane key="targets" title={t('business.observability.monitoring.targetsTab')}>
              <SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => setTargetVisible(true)}>{t('business.observability.monitoring.addTarget')}</Button> : undefined} />
              <Card className="page-panel system-list__table-card">
                {!loading && targets.length === 0 ? <PageEmpty description={t('business.observability.metricSource.empty')} /> : <AppTable rowKey="id" columns={targetColumns} data={targets} pagination={false} scroll={{ x: 'max-content' }} />}
              </Card>
            </TabPane>
            <TabPane key="datasources" title={t('business.observability.monitoring.datasourcesTab')}>
              <SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => setDatasourceVisible(true)}>{t('business.observability.monitoring.addDatasource')}</Button> : undefined} />
              <Card className="page-panel system-list__table-card">
                {!loading && datasources.length === 0 ? <PageEmpty description={t('business.observability.monitoring.datasourceEmpty')} /> : <AppTable rowKey="id" columns={datasourceColumns} data={datasources} pagination={false} scroll={{ x: 'max-content' }} />}
              </Card>
            </TabPane>
          </Tabs>
        ) : null}
      </Space>

      <AppModal visible={targetVisible} title={t('business.observability.monitoring.addTarget')} confirmLoading={submitting} onOk={() => void submitTarget()} onCancel={() => setTargetVisible(false)}>
        <Form form={targetForm} layout="vertical" initialValues={{ targetType: 'host', metricsPath: '/metrics' }}>
          <FormItem field="businessScopeId" label={t('business.observability.monitoring.field.scope')} rules={[{ required: true }]}><Select>{bizScopes.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem>
          <FormItem field="targetType" label={t('business.observability.monitoring.field.type')} rules={[{ required: true }]}><Select><Select.Option value="host">Host</Select.Option><Select.Option value="k8s_cluster">Kubernetes</Select.Option><Select.Option value="other">Other</Select.Option></Select></FormItem>
          <FormItem field="targetName" label={t('business.observability.monitoring.field.name')} rules={[{ required: true }]}><Input /></FormItem>
          <FormItem field="endpoint" label={t('business.observability.monitoring.field.endpoint')}><Input placeholder={t('business.observability.monitoring.field.endpointPlaceholder')} /></FormItem>
          <FormItem field="metricsPath" label={t('business.observability.metricSource.metricsPath')}><Input /></FormItem>
          <FormItem field="remark" label={t('common.remark')}><Input.TextArea /></FormItem>
        </Form>
      </AppModal>

      <AppModal visible={datasourceVisible} title={t('business.observability.monitoring.addDatasource')} confirmLoading={submitting} onOk={() => void submitDatasource()} onCancel={() => setDatasourceVisible(false)}>
        <Form form={datasourceForm} layout="vertical" initialValues={{ type: 'prometheus', timeout: 30, isDefault: false }}>
          <FormItem field="name" label={t('business.observability.monitoring.field.name')} rules={[{ required: true }]}><Input /></FormItem>
          <FormItem field="type" label={t('business.observability.monitoring.field.type')} rules={[{ required: true }]}><Select><Select.Option value="prometheus">Prometheus</Select.Option><Select.Option value="victoriametrics">VictoriaMetrics</Select.Option><Select.Option value="loki">Loki</Select.Option><Select.Option value="tempo">Tempo</Select.Option></Select></FormItem>
          <FormItem field="url" label={t('business.observability.monitoring.field.url')} rules={[{ required: true, type: 'url' }]}><Input placeholder={t('business.observability.monitoring.field.urlPlaceholder')} /></FormItem>
          <FormItem field="username" label={t('business.observability.monitoring.field.username')}><Input /></FormItem>
          <FormItem field="password" label={t('business.observability.monitoring.field.password')}><PasswordInput /></FormItem>
          <FormItem field="bearerToken" label={t('business.observability.monitoring.field.token')}><PasswordInput /></FormItem>
          <FormItem field="timeout" label={t('business.observability.monitoring.field.timeout')}><InputNumber min={1} max={120} /></FormItem>
          <FormItem field="isDefault" label={t('business.observability.monitoring.field.default')} triggerPropName="checked"><Switch /></FormItem>
        </Form>
      </AppModal>
    </PageContainer>
  );
}
