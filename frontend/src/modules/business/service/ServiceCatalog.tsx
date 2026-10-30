import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, Form, Input, Message, Popconfirm, Select, Space, Switch, Tabs, Tag } from '@arco-design/web-react';
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
import { getHostList, type HostRow } from '../cmdb/host/api';
import { getK8sClusterList, type K8sClusterRow } from '../k8s/api';
import {
  createApplication,
  createService,
  createServiceInstance,
  deleteApplication,
  deleteService,
  deleteServiceInstance,
  getApplicationList,
  getServiceInstanceList,
  getServiceList,
  reconcileServiceInstance,
  updateApplication,
  updateService,
  type ApplicationRow,
  type ServiceInstanceRow,
  type ServiceRow,
} from './api';
import '../../../assets/list-page.css';

const { TabPane } = Tabs;
const FormItem = Form.Item;

const instanceTargetTypeColor = (value: string) => (value === 'k8s' ? 'purple' : 'cyan');

const instanceObservedStateColor = (value: string) => {
  if (value === 'running') {
    return 'green';
  }
  if (value === 'failed') {
    return 'red';
  }
  return 'orange';
};

const instanceHealthStateColor = (value: string) => {
  if (value === 'healthy') {
    return 'green';
  }
  if (value === 'unhealthy') {
    return 'red';
  }
  return 'gray';
};

const instanceBindingLabel = (row: ServiceInstanceRow) =>
  row.targetType === 'vm'
    ? `Host #${row.hostId}`
    : `${row.namespace}/${row.workloadKind}/${row.workloadName}`;

const resolveAllowedTargetTypes = (runtimeType?: string): Array<'vm' | 'k8s'> => {
  if (runtimeType === 'vm') {
    return ['vm'];
  }
  if (runtimeType === 'k8s') {
    return ['k8s'];
  }
  return ['vm', 'k8s'];
};

interface ApplicationFormValues {
  code: string;
  name: string;
  businessScopeId: number;
  owner?: string;
  remark?: string;
}

interface ServiceFormValues {
  applicationId: number;
  code: string;
  name: string;
  runtimeType: string;
  description?: string;
}

interface InstanceFormValues {
  serviceId: number;
  environment: string;
  targetType: 'vm' | 'k8s';
  hostId?: number;
  k8sClusterId?: number;
  namespace?: string;
  workloadKind?: string;
  workloadName?: string;
  desiredVersion?: string;
}

export default function ServiceCatalog() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const [applications, setApplications] = useState<ApplicationRow[]>([]);
  const [services, setServices] = useState<ServiceRow[]>([]);
  const [instances, setInstances] = useState<ServiceInstanceRow[]>([]);
  const [bizScopes, setBizScopes] = useState<BizScopeOptionItem[]>([]);
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [applicationVisible, setApplicationVisible] = useState(false);
  const [serviceVisible, setServiceVisible] = useState(false);
  const [instanceVisible, setInstanceVisible] = useState(false);
  const [instanceTargetType, setInstanceTargetType] = useState<'vm' | 'k8s'>('vm');
  const [selectedServiceId, setSelectedServiceId] = useState<number>();
  const [applicationForm] = Form.useForm<ApplicationFormValues>();
  const [serviceForm] = Form.useForm<ServiceFormValues>();
  const [instanceForm] = Form.useForm<InstanceFormValues>();

  const canCreate = hasPerm('business:service:create');
  const canUpdate = hasPerm('business:service:update');
  const canDelete = hasPerm('business:service:delete');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [applicationResponse, serviceResponse, instanceResponse, scopeResponse, hostResponse, clusterResponse] = await Promise.all([
        getApplicationList({ page: 1, pageSize: 100 }),
        getServiceList({ page: 1, pageSize: 100 }),
        getServiceInstanceList({ page: 1, pageSize: 100 }),
        getBizScopeOptions(),
        getHostList({ page: 1, pageSize: 100 }),
        getK8sClusterList({ page: 1, pageSize: 100 }),
      ]);
      setApplications(applicationResponse.items);
      setServices(serviceResponse.items);
      setInstances(instanceResponse.items);
      setBizScopes(scopeResponse);
      setHosts(hostResponse.items);
      setClusters(clusterResponse.items);
    } catch (requestError) {
      setError(requestError);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  const submitApplication = useCallback(async () => {
    const values = await applicationForm.validate();
    setSubmitting(true);
    try {
      await createApplication({ ...values, status: 'active' });
      Message.success(t('business.service.catalog.applicationCreateSuccess'));
      setApplicationVisible(false);
      applicationForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.service.catalog.applicationCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [applicationForm, load, t]);

  const submitService = useCallback(async () => {
    const values = await serviceForm.validate();
    setSubmitting(true);
    try {
      await createService({ ...values, status: 'active' });
      Message.success(t('business.service.catalog.serviceCreateSuccess'));
      setServiceVisible(false);
      serviceForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.service.catalog.serviceCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [load, serviceForm, t]);

  const submitInstance = useCallback(async () => {
    const values = await instanceForm.validate();
    setSubmitting(true);
    try {
      await createServiceInstance({ ...values, status: 'active' });
      Message.success(t('business.service.catalog.instanceCreateSuccess'));
      setInstanceVisible(false);
      instanceForm.resetFields();
      await load();
    } catch {
      Message.error(t('business.service.catalog.instanceCreateFailed'));
    } finally {
      setSubmitting(false);
    }
  }, [instanceForm, load, t]);

  const toggleApplication = useCallback(async (row: ApplicationRow, active: boolean) => {
    await updateApplication(row.id, { status: active ? 'active' : 'inactive' });
    await load();
  }, [load]);

  const toggleService = useCallback(async (row: ServiceRow, active: boolean) => {
    await updateService(row.id, { status: active ? 'active' : 'inactive' });
    await load();
  }, [load]);

  const applicationColumns = useMemo<ColumnProps<ApplicationRow>[]>(() => [
    { title: t('business.service.catalog.field.code'), dataIndex: 'code', width: 150, render: (value: string) => <Tag color="arcoblue">{value}</Tag> },
    { title: t('business.service.catalog.field.application'), dataIndex: 'name', width: 200 },
    { title: t('business.service.catalog.field.scope'), dataIndex: 'businessScopeName', width: 180 },
    { title: t('business.service.catalog.field.owner'), dataIndex: 'owner', width: 140 },
    { title: t('business.service.catalog.field.services'), dataIndex: 'serviceCount', width: 100 },
    { title: t('business.service.catalog.field.active'), dataIndex: 'status', width: 90, render: (_: unknown, row) => <Switch checked={row.status === 'active'} disabled={!canUpdate} onChange={(active) => void toggleApplication(row, active)} /> },
    { title: t('common.action'), width: 90, fixed: 'right', render: (_: unknown, row) => canDelete ? <Popconfirm title={t('business.service.catalog.applicationDeleteConfirm', { name: row.name })} onOk={async () => { await deleteApplication(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null },
  ], [canDelete, canUpdate, load, t, toggleApplication]);

  const serviceColumns = useMemo<ColumnProps<ServiceRow>[]>(() => [
    { title: t('business.service.catalog.field.code'), dataIndex: 'code', width: 150, render: (value: string) => <Tag color="arcoblue">{value}</Tag> },
    { title: t('business.service.catalog.field.service'), dataIndex: 'name', width: 190 },
    { title: t('business.service.catalog.field.application'), dataIndex: 'applicationName', width: 190 },
    { title: t('business.service.catalog.field.runtime'), dataIndex: 'runtimeType', width: 120 },
    { title: t('business.service.catalog.field.instances'), dataIndex: 'instanceCount', width: 100 },
    { title: t('business.service.catalog.field.active'), dataIndex: 'status', width: 90, render: (_: unknown, row) => <Switch checked={row.status === 'active'} disabled={!canUpdate} onChange={(active) => void toggleService(row, active)} /> },
    { title: t('common.action'), width: 90, fixed: 'right', render: (_: unknown, row) => canDelete ? <Popconfirm title={t('business.service.catalog.serviceDeleteConfirm', { name: row.name })} onOk={async () => { await deleteService(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null },
  ], [canDelete, canUpdate, load, t, toggleService]);

  const instanceColumns = useMemo<ColumnProps<ServiceInstanceRow>[]>(() => [
    { title: t('business.service.catalog.field.service'), dataIndex: 'serviceName', width: 180 },
    { title: t('business.service.instance.environment'), dataIndex: 'environment', width: 110 },
    { title: t('business.service.instance.target'), dataIndex: 'targetType', width: 100, render: (value: string) => <Tag color={instanceTargetTypeColor(value)}>{value}</Tag> },
    { title: t('business.service.catalog.field.binding'), width: 260, render: (_: unknown, row) => instanceBindingLabel(row) },
    { title: t('business.service.instance.observed'), dataIndex: 'observedState', width: 120, render: (value: string) => <Tag color={instanceObservedStateColor(value)}>{value}</Tag> },
    { title: t('business.service.instance.health'), dataIndex: 'healthState', width: 110, render: (value: string) => <Tag color={instanceHealthStateColor(value)}>{value}</Tag> },
    {
      title: t('common.action'),
      width: 160,
      fixed: 'right',
      render: (_: unknown, row) => <Space>{canUpdate ? <Button type="text" icon={<IconRefresh />} onClick={async () => { await reconcileServiceInstance(row.id); await load(); }}>{t('business.service.instance.reconcile')}</Button> : null}{canDelete ? <Popconfirm title={t('business.service.catalog.instanceDeleteConfirm')} onOk={async () => { await deleteServiceInstance(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />} aria-label={t('common.delete')} /></Popconfirm> : null}</Space>,
    },
  ], [canDelete, canUpdate, load, t]);

  const selectedService = services.find((item) => item.id === selectedServiceId);
  const visibleHosts = selectedService ? hosts.filter((item) => item.businessScopeId === selectedService.businessScopeId) : hosts;
  const allowedTargetTypes = resolveAllowedTargetTypes(selectedService?.runtimeType);

  const handleServiceChange = (serviceId: number) => {
    const service = services.find((item) => item.id === serviceId);
    const targetType = service?.runtimeType === 'k8s' ? 'k8s' : 'vm';
    setSelectedServiceId(serviceId);
    setInstanceTargetType(targetType);
    instanceForm.setFieldsValue({
      serviceId,
      targetType,
      hostId: undefined,
      k8sClusterId: undefined,
      namespace: undefined,
      workloadKind: undefined,
      workloadName: undefined,
    });
  };

  const renderTable = <T,>(rows: T[], columns: ColumnProps<T>[], rowKey: string, emptyText: string) => (
    <Card className="page-panel system-list__table-card">
      {!loading && rows.length === 0 ? <PageEmpty description={emptyText} /> : <AppTable rowKey={rowKey} columns={columns} data={rows} pagination={false} scroll={{ x: 'max-content' }} />}
    </Card>
  );

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.service.catalog.hero.eyebrow')} title={t('operations.service.menu')} description={t('business.service.catalog.hero.title')} metrics={[
          { key: 'applications', label: t('business.service.catalog.hero.applications'), value: applications.length },
          { key: 'services', label: t('business.service.catalog.hero.services'), value: services.length },
          { key: 'instances', label: t('business.service.catalog.hero.instances'), value: instances.length },
          { key: 'healthy', label: t('business.service.catalog.hero.healthy'), value: instances.filter((item) => item.healthState === 'healthy').length },
        ]} />
        {loading && applications.length === 0 && services.length === 0 ? <PageLoading /> : null}
        {error && applications.length === 0 && services.length === 0 ? <PageError onRetry={() => void load()} /> : null}
        {!error ? <Tabs defaultActiveTab="applications">
          <TabPane key="applications" title={t('business.service.catalog.applicationsTab')}><SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} onClick={() => setApplicationVisible(true)}>{t('business.service.catalog.addApplication')}</Button> : undefined} />{renderTable(applications, applicationColumns, 'id', t('business.service.catalog.applicationEmpty'))}</TabPane>
          <TabPane key="services" title={t('business.service.catalog.servicesTab')}><SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} disabled={applications.length === 0} onClick={() => setServiceVisible(true)}>{t('business.service.catalog.addService')}</Button> : undefined} />{renderTable(services, serviceColumns, 'id', t('business.service.catalog.serviceEmpty'))}</TabPane>
          <TabPane key="instances" title={t('business.service.catalog.instancesTab')}><SearchToolbar onClearAll={() => undefined} extra={canCreate ? <Button type="primary" icon={<IconPlus />} disabled={services.length === 0} onClick={() => setInstanceVisible(true)}>{t('business.service.catalog.addInstance')}</Button> : undefined} />{renderTable(instances, instanceColumns, 'id', t('business.service.catalog.instanceEmpty'))}</TabPane>
        </Tabs> : null}
      </Space>

      <AppModal visible={applicationVisible} title={t('business.service.catalog.addApplication')} confirmLoading={submitting} onOk={() => void submitApplication()} onCancel={() => setApplicationVisible(false)}><Form form={applicationForm} layout="vertical"><FormItem field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="name" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="businessScopeId" label={t('business.service.catalog.field.scope')} rules={[{ required: true }]}><Select>{bizScopes.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem><FormItem field="owner" label={t('business.service.catalog.field.owner')}><Input /></FormItem><FormItem field="remark" label={t('common.remark')}><Input.TextArea /></FormItem></Form></AppModal>
      <AppModal visible={serviceVisible} title={t('business.service.catalog.addService')} confirmLoading={submitting} onOk={() => void submitService()} onCancel={() => setServiceVisible(false)}><Form form={serviceForm} layout="vertical"><FormItem field="applicationId" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Select>{applications.filter((item) => item.status === 'active').map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem><FormItem field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="name" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="runtimeType" label={t('business.service.catalog.field.runtime')} rules={[{ required: true }]}><Select><Select.Option value="vm">VM</Select.Option><Select.Option value="k8s">Kubernetes</Select.Option><Select.Option value="hybrid">Hybrid</Select.Option></Select></FormItem><FormItem field="description" label={t('common.description')}><Input.TextArea /></FormItem></Form></AppModal>
      <AppModal visible={instanceVisible} title={t('business.service.catalog.addInstance')} confirmLoading={submitting} onOk={() => void submitInstance()} onCancel={() => setInstanceVisible(false)}><Form form={instanceForm} layout="vertical" initialValues={{ environment: 'prod', targetType: 'vm' }}><FormItem field="serviceId" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}><Select onChange={handleServiceChange}>{services.filter((item) => item.status === 'active').map((item) => <Select.Option key={item.id} value={item.id}>{item.applicationName} / {item.name}</Select.Option>)}</Select></FormItem><FormItem field="environment" label={t('business.service.instance.environment')} rules={[{ required: true }]}><Select><Select.Option value="dev">Dev</Select.Option><Select.Option value="test">Test</Select.Option><Select.Option value="prod">Prod</Select.Option></Select></FormItem><FormItem field="targetType" label={t('business.service.instance.target')} rules={[{ required: true }]}><Select onChange={setInstanceTargetType}>{allowedTargetTypes.includes('vm') ? <Select.Option value="vm">VM Host</Select.Option> : null}{allowedTargetTypes.includes('k8s') ? <Select.Option value="k8s">Kubernetes Workload</Select.Option> : null}</Select></FormItem>{instanceTargetType === 'vm' ? <FormItem field="hostId" label="CMDB Host" rules={[{ required: true }]}><Select showSearch>{visibleHosts.map((item) => <Select.Option key={item.id} value={item.id}>{item.hostname} ({item.ip})</Select.Option>)}</Select></FormItem> : <><FormItem field="k8sClusterId" label={t('business.service.catalog.field.cluster')} rules={[{ required: true }]}><Select>{clusters.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem><FormItem field="namespace" label={t('business.service.catalog.field.namespace')} rules={[{ required: true }]}><Input /></FormItem><FormItem field="workloadKind" label={t('business.service.catalog.field.workloadKind')} rules={[{ required: true }]}><Select><Select.Option value="Deployment">Deployment</Select.Option><Select.Option value="StatefulSet">StatefulSet</Select.Option><Select.Option value="DaemonSet">DaemonSet</Select.Option></Select></FormItem><FormItem field="workloadName" label={t('business.service.catalog.field.workloadName')} rules={[{ required: true }]}><Input /></FormItem></>}<FormItem field="desiredVersion" label={t('business.service.instance.version')}><Input /></FormItem></Form></AppModal>
    </PageContainer>
  );
}
