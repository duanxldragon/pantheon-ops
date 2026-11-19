import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Alert, Button, Card, Form, Input, Message, Popconfirm, Select, Space, Switch, Tabs, Tag } from '@arco-design/web-react';
import { IconDelete, IconEdit, IconPlus, IconRefresh } from '@arco-design/web-react/icon';
import type { ColumnProps } from '@arco-design/web-react/es/Table/interface';
import type { FormInstance } from '@arco-design/web-react/es/Form';
import {
  AppModal,
  AppTable,
  GovernanceSummaryBar,
  PageContainer,
  PageEmpty,
  PageError,
  PageLoading,
  SearchToolbar,
  buildStandardPagination,
} from '../../../components';
import { isForbiddenRequestError, isRequestError } from '../../../api/request';
import { usePermission } from '@duanxldragon/pantheon-base-ui';
import { getBizScopeOptions, type BizScopeOptionItem } from '../bizscope/api';
import { getHostList, type HostRow } from '../cmdb/host/api';
import { getK8sClusterList, getK8sNamespaces, getK8sWorkloadList, type K8sClusterRow, type K8sNamespaceRow, type K8sWorkloadRow } from '../k8s/api';
import {
  createApplication,
  createService,
  createServiceInstance,
  deleteApplication,
  deleteService,
  deleteServiceInstance,
  getApplicationList,
  getServiceInstanceList,
  getService,
  getServiceList,
  reconcileServiceInstance,
  updateApplication,
  updateService,
  updateServiceInstance,
  type ApplicationRow,
  type ServiceInstanceRow,
  type ServiceRow,
} from './api';
import '../../../assets/list-page.css';
import { useCatalogOptions } from './useCatalogOptions';

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

const resolveAllowedTargetTypes = (runtimeType?: string): Array<'vm' | 'k8s'> => {
  if (runtimeType === 'vm') {
    return ['vm'];
  }
  if (runtimeType === 'k8s') {
    return ['k8s'];
  }
  return ['vm', 'k8s'];
};

interface CatalogQuery { page: number; pageSize: number; keyword?: string }
const initialQuery: CatalogQuery = { page: 1, pageSize: 20 };

function CatalogLookupNotice({ error }: Readonly<{ error: unknown }>) {
  const { t } = useTranslation();
  return error ? <Alert type="error" content={t(isForbiddenRequestError(error) ? 'business.service.catalog.optionsForbidden' : 'business.service.catalog.optionsFailed')} /> : null;
}

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

interface CatalogStateNoticesProps {
  readonly loading: boolean;
  readonly error: unknown;
  readonly applicationCount: number;
  readonly serviceCount: number;
  readonly onRetry: () => void;
}

function CatalogStateNotices({
  loading,
  error,
  applicationCount,
  serviceCount,
  onRetry,
}: Readonly<CatalogStateNoticesProps>) {
  const catalogEmpty = applicationCount === 0 && serviceCount === 0;
  return (
    <>
      {loading && catalogEmpty ? <PageLoading /> : null}
      {error ? <PageError onRetry={onRetry} /> : null}
    </>
  );
}

interface CatalogTabPanelProps<T> {
  readonly loading: boolean;
  readonly rows: T[];
  readonly columns: ColumnProps<T>[];
  readonly rowKey: string;
  readonly emptyText: string;
  readonly canCreate: boolean;
  readonly addLabel: string;
  readonly addDisabled: boolean;
  readonly onAdd: () => void;
  readonly query: CatalogQuery;
  readonly total: number;
  readonly onQueryChange: (query: CatalogQuery) => void;
  readonly keywordPlaceholder?: string;
}

function CatalogTabPanel<T>({
  loading,
  rows,
  columns,
  rowKey,
  emptyText,
  canCreate,
  addLabel,
  addDisabled,
  onAdd,
  query,
  total,
  onQueryChange,
  keywordPlaceholder,
}: Readonly<CatalogTabPanelProps<T>>) {
  const { t } = useTranslation();
  return (
    <>
      <SearchToolbar
        keyword={query.keyword ?? ''}
        keywordPlaceholder={keywordPlaceholder}
        onKeywordChange={keywordPlaceholder ? (keyword) => onQueryChange({ ...query, page: 1, keyword }) : undefined}
        hasActiveFilters={Boolean(query.keyword)}
        onClearAll={() => onQueryChange({ ...query, page: 1, keyword: '' })}
        extra={canCreate ? <Button type="primary" icon={<IconPlus />} disabled={addDisabled} onClick={onAdd}>{addLabel}</Button> : undefined}
      />
      <Card className="page-panel system-list__table-card">
        {!loading && rows.length === 0 && total === 0 ? <PageEmpty description={emptyText} /> : <AppTable rowKey={rowKey} columns={columns} data={rows} loading={loading} pagination={buildStandardPagination(t, { current: query.page, pageSize: query.pageSize, total, pageSizeChangeResetCurrent: true, onChange: (page, pageSize) => onQueryChange({ ...query, page, pageSize }) })} scroll={{ x: 'max-content' }} />}
      </Card>
    </>
  );
}

interface ApplicationFormModalProps {
  readonly visible: boolean;
  readonly submitting: boolean;
  readonly form: FormInstance<ApplicationFormValues>;
  readonly bizScopes: BizScopeOptionItem[];
  readonly editing?: ApplicationRow;
  readonly optionsLoading: boolean;
  readonly optionsError: unknown;
  readonly onOk: () => void;
  readonly onCancel: () => void;
}

function ApplicationFormModal({
  visible,
  submitting,
  form,
  bizScopes,
  editing,
  optionsLoading,
  optionsError,
  onOk,
  onCancel,
}: Readonly<ApplicationFormModalProps>) {
  const { t } = useTranslation();
  return (
    <AppModal visible={visible} title={t(editing ? 'business.service.catalog.editApplication' : 'business.service.catalog.addApplication')} confirmLoading={submitting} okButtonProps={{ disabled: !editing && (optionsLoading || Boolean(optionsError)) }} onOk={onOk} onCancel={onCancel}>
      <CatalogLookupNotice error={optionsError} />
      <Form form={form} layout="vertical">
        <FormItem field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input disabled={Boolean(editing)} /></FormItem>
        <FormItem field="name" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Input /></FormItem>
        <FormItem field="businessScopeId" label={t('business.service.catalog.field.scope')} rules={[{ required: true }]}><Select disabled={Boolean(editing)} loading={optionsLoading}>{editing ? <Select.Option value={editing.businessScopeId}>{editing.businessScopeName}</Select.Option> : bizScopes.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem>
        <FormItem field="owner" label={t('business.service.catalog.field.owner')}><Input /></FormItem>
        <FormItem field="remark" label={t('common.remark')}><Input.TextArea /></FormItem>
      </Form>
    </AppModal>
  );
}

interface ServiceFormModalProps {
  readonly visible: boolean;
  readonly submitting: boolean;
  readonly form: FormInstance<ServiceFormValues>;
  readonly applications: ApplicationRow[];
  readonly editing?: ServiceRow;
  readonly optionsLoading: boolean;
  readonly optionsError: unknown;
  readonly onApplicationSearch: (keyword: string) => void;
  readonly onOk: () => void;
  readonly onCancel: () => void;
}

function ServiceFormModal({
  visible,
  submitting,
  form,
  applications,
  editing,
  optionsLoading,
  optionsError,
  onApplicationSearch,
  onOk,
  onCancel,
}: Readonly<ServiceFormModalProps>) {
  const { t } = useTranslation();
  return (
    <AppModal visible={visible} title={t(editing ? 'business.service.catalog.editService' : 'business.service.catalog.addService')} confirmLoading={submitting} okButtonProps={{ disabled: !editing && (optionsLoading || Boolean(optionsError)) }} onOk={onOk} onCancel={onCancel}>
      <CatalogLookupNotice error={optionsError} />
      <Form form={form} layout="vertical">
        <FormItem field="applicationId" label={t('business.service.catalog.field.application')} rules={[{ required: true }]}><Select disabled={Boolean(editing)} showSearch filterOption={false} onSearch={onApplicationSearch} loading={optionsLoading}>{editing ? <Select.Option value={editing.applicationId}>{editing.applicationName}</Select.Option> : applications.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem>
        <FormItem field="code" label={t('business.service.catalog.field.code')} rules={[{ required: true }]}><Input disabled={Boolean(editing)} /></FormItem>
        <FormItem field="name" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}><Input /></FormItem>
        <FormItem field="runtimeType" label={t('business.service.catalog.field.runtime')} rules={[{ required: true }]}><Select>{['vm', 'k8s', 'hybrid'].map((runtime) => <Select.Option key={runtime} value={runtime}>{t(`business.service.catalog.runtime.${runtime}`)}</Select.Option>)}</Select></FormItem>
        <FormItem field="description" label={t('common.description')}><Input.TextArea /></FormItem>
      </Form>
    </AppModal>
  );
}

interface InstanceFormModalProps {
  readonly visible: boolean;
  readonly submitting: boolean;
  readonly form: FormInstance<InstanceFormValues>;
  readonly services: ServiceRow[];
  readonly clusters: K8sClusterRow[];
  readonly hosts: HostRow[];
  readonly namespaces: K8sNamespaceRow[];
  readonly workloads: K8sWorkloadRow[];
  readonly editing?: ServiceInstanceRow;
  readonly changingTarget: boolean;
  readonly onChangingTarget: (enabled: boolean) => void;
  readonly optionsLoading: boolean;
  readonly optionsError: unknown;
  readonly clusterId?: number;
  readonly namespace?: string;
  readonly workloadKind?: string;
  readonly onServiceSearch: (keyword: string) => void;
  readonly onTargetSearch: (keyword: string) => void;
  readonly onClusterChange: (id: number) => void;
  readonly onNamespaceChange: (name: string) => void;
  readonly onWorkloadKindChange: (kind: string) => void;
  readonly allowedTargetTypes: Array<'vm' | 'k8s'>;
  readonly targetType: 'vm' | 'k8s';
  readonly onOk: () => void;
  readonly onCancel: () => void;
  readonly onServiceChange: (serviceId: number) => void;
  readonly onTargetTypeChange: (value: 'vm' | 'k8s') => void;
}

function InstanceFormModal({
  visible,
  submitting,
  form,
  services,
  clusters,
  hosts,
  namespaces,
  workloads,
  editing,
  changingTarget,
  onChangingTarget,
  optionsLoading,
  optionsError,
  clusterId,
  namespace,
  workloadKind,
  onServiceSearch,
  onTargetSearch,
  onClusterChange,
  onNamespaceChange,
  onWorkloadKindChange,
  allowedTargetTypes,
  targetType,
  onOk,
  onCancel,
  onServiceChange,
  onTargetTypeChange,
}: Readonly<InstanceFormModalProps>) {
  const { t } = useTranslation();
  return (
    <AppModal visible={visible} title={t(editing ? 'business.service.catalog.editInstance' : 'business.service.catalog.addInstance')} confirmLoading={submitting} okButtonProps={{ disabled: (!editing || changingTarget) && (optionsLoading || Boolean(optionsError)) }} onOk={onOk} onCancel={onCancel}>
      <CatalogLookupNotice error={optionsError} />
      <Form form={form} layout="vertical" initialValues={{ environment: 'prod', targetType: 'vm' }}>
        {editing ? <>
          <FormItem label={t('business.service.catalog.field.service')}><Input value={editing.serviceName} readOnly /></FormItem>
          <FormItem label={t('business.service.catalog.field.binding')}><Input value={editing.targetType === 'vm' ? t('business.service.catalog.hostBinding', { id: editing.hostId }) : `${editing.namespace}/${editing.workloadKind}/${editing.workloadName}`} readOnly /></FormItem>
          <FormItem label={t('business.service.catalog.currentVersion')}><Input value={editing.currentVersion} readOnly /></FormItem>
          <FormItem label={t('business.service.catalog.changeTarget')}><Switch checked={changingTarget} onChange={onChangingTarget} /></FormItem>
          {changingTarget ? <Alert type="info" content={t('business.service.catalog.targetChangeHint')} /> : null}
        </> : null}
        {!editing ? <>
        <FormItem field="serviceId" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}><Select showSearch filterOption={false} onSearch={onServiceSearch} onChange={onServiceChange} loading={optionsLoading}>{services.map((item) => <Select.Option key={item.id} value={item.id}>{item.applicationName} / {item.name}</Select.Option>)}</Select></FormItem>
        <FormItem field="environment" label={t('business.service.instance.environment')} rules={[{ required: true }]}><Select><Select.Option value="dev">Dev</Select.Option><Select.Option value="test">Test</Select.Option><Select.Option value="prod">Prod</Select.Option></Select></FormItem>
        </> : null}
        {!editing || changingTarget ? <>
        <FormItem field="targetType" label={t('business.service.instance.target')} rules={[{ required: true }]}><Select onChange={onTargetTypeChange}>{allowedTargetTypes.map((value) => <Select.Option key={value} value={value}>{t(`business.service.catalog.runtime.${value}`)}</Select.Option>)}</Select></FormItem>
        {targetType === 'vm' ? (
          <FormItem field="hostId" label={t('business.service.catalog.host')} rules={[{ required: true }]}><Select showSearch filterOption={false} onSearch={onTargetSearch} loading={optionsLoading}>{hosts.map((item) => <Select.Option key={item.id} value={item.id}>{item.hostname} ({item.ip})</Select.Option>)}</Select></FormItem>
        ) : (
          <>
            <FormItem field="k8sClusterId" label={t('business.service.catalog.field.cluster')} rules={[{ required: true }]}><Select showSearch filterOption={false} onSearch={onTargetSearch} onChange={onClusterChange} loading={optionsLoading}>{clusters.map((item) => <Select.Option key={item.id} value={item.id}>{item.name}</Select.Option>)}</Select></FormItem>
            <FormItem field="namespace" label={t('business.service.catalog.field.namespace')} rules={[{ required: true }]}><Select disabled={!clusterId} loading={optionsLoading} showSearch onChange={onNamespaceChange}>{namespaces.map((item) => <Select.Option key={item.name} value={item.name}>{item.name}</Select.Option>)}</Select></FormItem>
            <FormItem field="workloadKind" label={t('business.service.catalog.field.workloadKind')} rules={[{ required: true }]}><Select disabled={!namespace} onChange={onWorkloadKindChange}>{['Deployment', 'StatefulSet', 'DaemonSet'].map((kind) => <Select.Option key={kind} value={kind}>{kind}</Select.Option>)}</Select></FormItem>
            <FormItem field="workloadName" label={t('business.service.catalog.field.workloadName')} rules={[{ required: true }]}><Select disabled={!workloadKind} loading={optionsLoading} showSearch>{workloads.map((item) => <Select.Option key={`${item.kind}/${item.name}`} value={item.name}>{item.name}</Select.Option>)}</Select></FormItem>
          </>
        )}
        </> : null}
        <FormItem field="desiredVersion" label={t('business.service.instance.version')}><Input /></FormItem>
      </Form>
    </AppModal>
  );
}

export default function ServiceCatalog() {
  const { t } = useTranslation();
  const { hasPerm } = usePermission();
  const location = useLocation();
  const navigate = useNavigate();
  const suffix = location.pathname.split('/').pop();
  const activeTab = suffix === 'applications' || suffix === 'services' || suffix === 'instances' ? suffix : 'applications';
  const [applications, setApplications] = useState<ApplicationRow[]>([]);
  const [services, setServices] = useState<ServiceRow[]>([]);
  const [instances, setInstances] = useState<ServiceInstanceRow[]>([]);
  const [applicationQuery, setApplicationQuery] = useState<CatalogQuery>(initialQuery);
  const [serviceQuery, setServiceQuery] = useState<CatalogQuery>(initialQuery);
  const [instanceQuery, setInstanceQuery] = useState<CatalogQuery>(initialQuery);
  const [totals, setTotals] = useState({ applications: 0, services: 0, instances: 0 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const [submitting, setSubmitting] = useState(false);
  const [applicationVisible, setApplicationVisible] = useState(false);
  const [serviceVisible, setServiceVisible] = useState(false);
  const [instanceVisible, setInstanceVisible] = useState(false);
  const [editingApplication, setEditingApplication] = useState<ApplicationRow>();
  const [editingService, setEditingService] = useState<ServiceRow>();
  const [editingInstance, setEditingInstance] = useState<ServiceInstanceRow>();
  const [changingInstanceTarget, setChangingInstanceTarget] = useState(false);
  const [instanceTargetType, setInstanceTargetType] = useState<'vm' | 'k8s'>('vm');
  const [selectedService, setSelectedService] = useState<ServiceRow>();
  const [selectedClusterId, setSelectedClusterId] = useState<number>();
  const [selectedNamespace, setSelectedNamespace] = useState<string>();
  const [selectedWorkloadKind, setSelectedWorkloadKind] = useState<string>();
  const [applicationKeyword, setApplicationKeyword] = useState('');
  const [serviceKeyword, setServiceKeyword] = useState('');
  const [targetKeyword, setTargetKeyword] = useState('');
  const listAbort = useRef<AbortController | null>(null);
  const [applicationForm] = Form.useForm<ApplicationFormValues>();
  const [serviceForm] = Form.useForm<ServiceFormValues>();
  const [instanceForm] = Form.useForm<InstanceFormValues>();

  const canCreate = hasPerm('business:service:create');
  const canUpdate = hasPerm('business:service:update');
  const canDelete = hasPerm('business:service:delete');

  const scopeLookup = useCallback(() => getBizScopeOptions(), []);
  const scopes = useCatalogOptions(applicationVisible && !editingApplication, scopeLookup);
  const applicationLookup = useCallback((signal: AbortSignal) => getApplicationList({ page: 1, pageSize: 25, status: 'active', keyword: applicationKeyword }, signal).then((result) => result.items), [applicationKeyword]);
  const applicationOptions = useCatalogOptions(serviceVisible && !editingService, applicationLookup);
  const serviceLookup = useCallback((signal: AbortSignal) => getServiceList({ page: 1, pageSize: 25, status: 'active', keyword: serviceKeyword }, signal).then((result) => result.items), [serviceKeyword]);
  const serviceOptions = useCatalogOptions(instanceVisible && !editingInstance, serviceLookup);
  const editingServiceLookup = useCallback((signal: AbortSignal) => getService(editingInstance?.serviceId ?? 0, signal).then((row) => [row]), [editingInstance?.serviceId]);
  const editingServiceOptions = useCatalogOptions(instanceVisible && Boolean(editingInstance) && changingInstanceTarget, editingServiceLookup);
  const targetService = editingInstance ? editingServiceOptions.rows[0] : selectedService;
  const scopeId = targetService?.businessScopeId;
  const selectingTarget = instanceVisible && (!editingInstance || changingInstanceTarget);
  const hostLookup = useCallback((signal: AbortSignal) => getHostList({ page: 1, pageSize: 25, businessScopeId: scopeId, keyword: targetKeyword }, signal).then((result) => result.items.filter((item) => item.businessScopeId === scopeId)), [scopeId, targetKeyword]);
  const hostOptions = useCatalogOptions(selectingTarget && Boolean(scopeId) && instanceTargetType === 'vm', hostLookup);
  const clusterLookup = useCallback(() => getK8sClusterList({ page: 1, pageSize: 25, businessScopeId: scopeId, keyword: targetKeyword }).then((result) => result.items.filter((item) => item.businessScopeId === scopeId)), [scopeId, targetKeyword]);
  const clusterOptions = useCatalogOptions(selectingTarget && Boolean(scopeId) && instanceTargetType === 'k8s', clusterLookup);
  const namespaceLookup = useCallback(() => getK8sNamespaces(selectedClusterId ?? 0).then((result) => result.items), [selectedClusterId]);
  const namespaceOptions = useCatalogOptions(selectingTarget && instanceTargetType === 'k8s' && Boolean(selectedClusterId), namespaceLookup);
  const workloadLookup = useCallback(() => getK8sWorkloadList({ clusterId: selectedClusterId ?? 0, namespace: selectedNamespace, kind: selectedWorkloadKind }).then((result) => result.items.filter((item) => item.namespace === selectedNamespace && item.kind === selectedWorkloadKind)), [selectedClusterId, selectedNamespace, selectedWorkloadKind]);
  const workloadOptions = useCatalogOptions(selectingTarget && instanceTargetType === 'k8s' && Boolean(selectedClusterId && selectedNamespace && selectedWorkloadKind), workloadLookup);
  const instanceLookups = [serviceOptions, editingServiceOptions, hostOptions, clusterOptions, namespaceOptions, workloadOptions];
  const instanceOptionsError = instanceLookups.find((result) => result.error)?.error;
  const instanceOptionsLoading = instanceLookups.some((result) => result.loading);

  const operationFailed = useCallback((requestError: unknown) => {
    // C03 mutation wrappers keep the shared default error feedback enabled.
    // RequestError itself is not a global marker that feedback was displayed.
    if (isRequestError(requestError)) return;
    Message.error(t('business.service.catalog.operationFailed'));
  }, [t]);

  const openApplication = useCallback((row?: ApplicationRow) => {
    setEditingApplication(row);
    applicationForm.resetFields();
    if (row) applicationForm.setFieldsValue(row);
    setApplicationVisible(true);
  }, [applicationForm]);
  const openService = useCallback((row?: ServiceRow) => {
    setEditingService(row);
    setApplicationKeyword('');
    serviceForm.resetFields();
    if (row) serviceForm.setFieldsValue(row);
    setServiceVisible(true);
  }, [serviceForm]);
  const openInstance = useCallback((row?: ServiceInstanceRow) => {
    setEditingInstance(row);
    setChangingInstanceTarget(false);
    setSelectedService(undefined);
    setSelectedClusterId(undefined);
    setSelectedNamespace(undefined);
    setSelectedWorkloadKind(undefined);
    setServiceKeyword('');
    setTargetKeyword('');
    setInstanceTargetType('vm');
    instanceForm.resetFields();
    if (row) instanceForm.setFieldsValue({ desiredVersion: row.desiredVersion });
    setInstanceVisible(true);
  }, [instanceForm]);

  const load = useCallback(async () => {
    listAbort.current?.abort();
    const controller = new AbortController();
    listAbort.current = controller;
    setLoading(true);
    setError(null);
    try {
      const [applicationResponse, serviceResponse, instanceResponse] = await Promise.all([
        getApplicationList(applicationQuery, controller.signal),
        getServiceList(serviceQuery, controller.signal),
        getServiceInstanceList(instanceQuery, controller.signal),
      ]);
      if (controller.signal.aborted) return;
      setApplications(applicationResponse.items);
      setServices(serviceResponse.items);
      setInstances(instanceResponse.items);
      setTotals({ applications: applicationResponse.total, services: serviceResponse.total, instances: instanceResponse.total });
    } catch (requestError) {
      if (!controller.signal.aborted) setError(requestError);
    } finally {
      if (!controller.signal.aborted) setLoading(false);
    }
  }, [applicationQuery, serviceQuery, instanceQuery]);

  useEffect(() => {
    let active = true;
    queueMicrotask(() => { if (active) void load(); });
    return () => { active = false; listAbort.current?.abort(); };
  }, [load]);

  const submitApplication = useCallback(async () => {
    const values = await applicationForm.validate();
    setSubmitting(true);
    try {
      if (editingApplication) await updateApplication(editingApplication.id, { name: values.name, owner: values.owner, remark: values.remark });
      else await createApplication({ ...values, status: 'active' });
      Message.success(t(editingApplication ? 'business.service.catalog.saveSuccess' : 'business.service.catalog.applicationCreateSuccess'));
      setApplicationVisible(false);
      applicationForm.resetFields();
      await load();
    } catch (requestError) {
      operationFailed(requestError);
    } finally {
      setSubmitting(false);
    }
  }, [applicationForm, editingApplication, load, operationFailed, t]);

  const submitService = useCallback(async () => {
    const values = await serviceForm.validate();
    setSubmitting(true);
    try {
      if (editingService) await updateService(editingService.id, { name: values.name, runtimeType: values.runtimeType, description: values.description });
      else await createService({ ...values, status: 'active' });
      Message.success(t(editingService ? 'business.service.catalog.saveSuccess' : 'business.service.catalog.serviceCreateSuccess'));
      setServiceVisible(false);
      serviceForm.resetFields();
      await load();
    } catch (requestError) {
      operationFailed(requestError);
    } finally {
      setSubmitting(false);
    }
  }, [editingService, load, operationFailed, serviceForm, t]);

  const submitInstance = useCallback(async () => {
    const values = await instanceForm.validate();
    if ((!editingInstance || changingInstanceTarget) && (
      instanceOptionsLoading || instanceOptionsError || !targetService ||
      !resolveAllowedTargetTypes(targetService.runtimeType).includes(values.targetType) ||
      (values.targetType === 'vm' ? !hostOptions.rows.some((item) => item.id === values.hostId) :
        !clusterOptions.rows.some((item) => item.id === values.k8sClusterId) ||
        !namespaceOptions.rows.some((item) => item.name === values.namespace) ||
        !workloadOptions.rows.some((item) => item.name === values.workloadName && item.namespace === values.namespace && item.kind === values.workloadKind))
    )) {
      Message.error(t('business.service.catalog.targetUnavailable'));
      return;
    }
    setSubmitting(true);
    try {
      if (editingInstance) await updateServiceInstance(editingInstance.id, {
        desiredVersion: values.desiredVersion ?? '',
        expectedLifecycleVersion: editingInstance.lifecycleVersion,
        ...(changingInstanceTarget ? { target: values.targetType === 'vm' ? { targetType: values.targetType, hostId: values.hostId } : { targetType: values.targetType, k8sClusterId: values.k8sClusterId, namespace: values.namespace, workloadKind: values.workloadKind, workloadName: values.workloadName } } : {}),
      });
      else await createServiceInstance({ ...values, status: 'active' });
      Message.success(t(editingInstance ? 'business.service.catalog.saveSuccess' : 'business.service.catalog.instanceCreateSuccess'));
      setInstanceVisible(false);
      instanceForm.resetFields();
      await load();
    } catch (requestError) {
      operationFailed(requestError);
    } finally {
      setSubmitting(false);
    }
  }, [changingInstanceTarget, clusterOptions.rows, editingInstance, hostOptions.rows, instanceForm, instanceOptionsError, instanceOptionsLoading, load, namespaceOptions.rows, operationFailed, t, targetService, workloadOptions.rows]);

  const toggleApplication = useCallback(async (row: ApplicationRow, active: boolean) => {
    try { await updateApplication(row.id, { status: active ? 'active' : 'inactive' }); await load(); }
    catch (requestError) { operationFailed(requestError); }
  }, [load, operationFailed]);

  const toggleService = useCallback(async (row: ServiceRow, active: boolean) => {
    try { await updateService(row.id, { status: active ? 'active' : 'inactive' }); await load(); }
    catch (requestError) { operationFailed(requestError); }
  }, [load, operationFailed]);

  const applicationColumns = useMemo<ColumnProps<ApplicationRow>[]>(() => [
    { title: t('business.service.catalog.field.code'), dataIndex: 'code', width: 150, render: (value: string) => <Tag color="arcoblue">{value}</Tag> },
    { title: t('business.service.catalog.field.application'), dataIndex: 'name', width: 200 },
    { title: t('business.service.catalog.field.scope'), dataIndex: 'businessScopeName', width: 180 },
    { title: t('business.service.catalog.field.owner'), dataIndex: 'owner', width: 140 },
    { title: t('business.service.catalog.field.services'), dataIndex: 'serviceCount', width: 100 },
    { title: t('business.service.catalog.field.active'), dataIndex: 'status', width: 90, render: (_: unknown, row) => <Switch checked={row.status === 'active'} disabled={!canUpdate} onChange={(active) => void toggleApplication(row, active)} /> },
    { title: t('common.action'), width: 180, fixed: 'right', render: (_: unknown, row) => <Space>{canUpdate ? <Button type="text" icon={<IconEdit />} onClick={() => openApplication(row)}>{t('common.edit')}</Button> : null}{canDelete ? <Popconfirm title={t('business.service.catalog.applicationDeleteConfirm', { name: row.name })} onOk={async () => { await deleteApplication(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null}</Space> },
  ], [canDelete, canUpdate, load, openApplication, t, toggleApplication]);

  const serviceColumns = useMemo<ColumnProps<ServiceRow>[]>(() => [
    { title: t('business.service.catalog.field.code'), dataIndex: 'code', width: 150, render: (value: string) => <Tag color="arcoblue">{value}</Tag> },
    { title: t('business.service.catalog.field.service'), dataIndex: 'name', width: 190 },
    { title: t('business.service.catalog.field.application'), dataIndex: 'applicationName', width: 190 },
    { title: t('business.service.catalog.field.runtime'), dataIndex: 'runtimeType', width: 120 },
    { title: t('business.service.catalog.field.instances'), dataIndex: 'instanceCount', width: 100 },
    { title: t('business.service.catalog.field.active'), dataIndex: 'status', width: 90, render: (_: unknown, row) => <Switch checked={row.status === 'active'} disabled={!canUpdate} onChange={(active) => void toggleService(row, active)} /> },
    { title: t('common.action'), width: 180, fixed: 'right', render: (_: unknown, row) => <Space>{canUpdate ? <Button type="text" icon={<IconEdit />} onClick={() => openService(row)}>{t('common.edit')}</Button> : null}{canDelete ? <Popconfirm title={t('business.service.catalog.serviceDeleteConfirm', { name: row.name })} onOk={async () => { await deleteService(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />}>{t('common.delete')}</Button></Popconfirm> : null}</Space> },
  ], [canDelete, canUpdate, load, openService, t, toggleService]);

  const instanceColumns = useMemo<ColumnProps<ServiceInstanceRow>[]>(() => [
    { title: t('business.service.catalog.field.service'), dataIndex: 'serviceName', width: 180 },
    { title: t('business.service.instance.environment'), dataIndex: 'environment', width: 110 },
    { title: t('business.service.instance.target'), dataIndex: 'targetType', width: 100, render: (value: string) => <Tag color={instanceTargetTypeColor(value)}>{value}</Tag> },
    { title: t('business.service.catalog.field.binding'), width: 260, render: (_: unknown, row) => row.targetType === 'vm' ? t('business.service.catalog.hostBinding', { id: row.hostId }) : `${row.namespace}/${row.workloadKind}/${row.workloadName}` },
    { title: t('business.service.instance.observed'), dataIndex: 'observedState', width: 120, render: (value: string) => <Tag color={instanceObservedStateColor(value)}>{value}</Tag> },
    { title: t('business.service.instance.health'), dataIndex: 'healthState', width: 110, render: (value: string) => <Tag color={instanceHealthStateColor(value)}>{value}</Tag> },
    {
      title: t('common.action'),
      width: 230,
      fixed: 'right',
      render: (_: unknown, row) => <Space>{canUpdate ? <><Button type="text" icon={<IconEdit />} onClick={() => openInstance(row)}>{t('common.edit')}</Button><Button type="text" icon={<IconRefresh />} onClick={async () => { await reconcileServiceInstance(row.id); await load(); }}>{t('business.service.instance.reconcile')}</Button></> : null}{canDelete ? <Popconfirm title={t('business.service.catalog.instanceDeleteConfirm')} onOk={async () => { await deleteServiceInstance(row.id); await load(); }}><Button type="text" status="danger" icon={<IconDelete />} aria-label={t('common.delete')} /></Popconfirm> : null}</Space>,
    },
  ], [canDelete, canUpdate, load, openInstance, t]);

  const allowedTargetTypes = resolveAllowedTargetTypes(targetService?.runtimeType);

  const clearTarget = () => {
    setSelectedClusterId(undefined);
    setSelectedNamespace(undefined);
    setSelectedWorkloadKind(undefined);
    setTargetKeyword('');
    instanceForm.setFieldsValue({ hostId: undefined, k8sClusterId: undefined, namespace: undefined, workloadKind: undefined, workloadName: undefined });
  };

  const handleServiceChange = (serviceId: number) => {
    const service = serviceOptions.rows.find((item) => item.id === serviceId);
    const targetType = service?.runtimeType === 'k8s' ? 'k8s' : 'vm';
    setSelectedService(service);
    setInstanceTargetType(targetType);
    clearTarget();
    instanceForm.setFieldsValue({ serviceId, targetType });
  };

  return (
    <PageContainer>
      <Space direction="vertical" size={16} className="system-page-template">
        <GovernanceSummaryBar eyebrow={t('business.service.catalog.hero.eyebrow')} title={t('operations.service.menu')} description={t('business.service.catalog.hero.title')} metrics={[
          { key: 'applications', label: t('business.service.catalog.hero.applications'), value: totals.applications },
          { key: 'services', label: t('business.service.catalog.hero.services'), value: totals.services },
          { key: 'instances', label: t('business.service.catalog.hero.instances'), value: totals.instances },
          { key: 'healthy', label: t('business.service.catalog.healthyOnPage'), value: instances.filter((item) => item.healthState === 'healthy').length },
        ]} />
        <CatalogStateNotices
          loading={loading}
          error={error}
          applicationCount={applications.length}
          serviceCount={services.length}
          onRetry={() => void load()}
        />
        {!error ? <Tabs activeTab={activeTab} onChange={(key) => void navigate(`/business/service-catalog/${key}`)}>
          <TabPane key="applications" title={t('business.service.catalog.applicationsTab')}>
            <CatalogTabPanel
              rows={applications}
              columns={applicationColumns}
              rowKey="id"
              loading={loading}
              canCreate={canCreate}
              addDisabled={false}
              addLabel={t('business.service.catalog.addApplication')}
              emptyText={t('business.service.catalog.applicationEmpty')}
              onAdd={() => openApplication()}
              query={applicationQuery}
              total={totals.applications}
              onQueryChange={setApplicationQuery}
              keywordPlaceholder={t('business.service.catalog.applicationKeyword')}
            />
          </TabPane>
          <TabPane key="services" title={t('business.service.catalog.servicesTab')}>
            <CatalogTabPanel
              rows={services}
              columns={serviceColumns}
              rowKey="id"
              loading={loading}
              canCreate={canCreate}
              addDisabled={false}
              addLabel={t('business.service.catalog.addService')}
              emptyText={t('business.service.catalog.serviceEmpty')}
              onAdd={() => openService()}
              query={serviceQuery}
              total={totals.services}
              onQueryChange={setServiceQuery}
              keywordPlaceholder={t('business.service.catalog.serviceKeyword')}
            />
          </TabPane>
          <TabPane key="instances" title={t('business.service.catalog.instancesTab')}>
            <CatalogTabPanel
              rows={instances}
              columns={instanceColumns}
              rowKey="id"
              loading={loading}
              canCreate={canCreate}
              addDisabled={false}
              addLabel={t('business.service.catalog.addInstance')}
              emptyText={t('business.service.catalog.instanceEmpty')}
              onAdd={() => openInstance()}
              query={instanceQuery}
              total={totals.instances}
              onQueryChange={setInstanceQuery}
            />
          </TabPane>
        </Tabs> : null}
      </Space>

      <ApplicationFormModal
        visible={applicationVisible}
        submitting={submitting}
        form={applicationForm}
        bizScopes={scopes.rows}
        editing={editingApplication}
        optionsLoading={scopes.loading}
        optionsError={scopes.error}
        onOk={() => void submitApplication()}
        onCancel={() => setApplicationVisible(false)}
      />
      <ServiceFormModal
        visible={serviceVisible}
        submitting={submitting}
        form={serviceForm}
        applications={applicationOptions.rows}
        editing={editingService}
        optionsLoading={applicationOptions.loading}
        optionsError={applicationOptions.error}
        onApplicationSearch={setApplicationKeyword}
        onOk={() => void submitService()}
        onCancel={() => setServiceVisible(false)}
      />
      <InstanceFormModal
        visible={instanceVisible}
        submitting={submitting}
        form={instanceForm}
        services={serviceOptions.rows}
        clusters={clusterOptions.rows}
        hosts={hostOptions.rows}
        namespaces={namespaceOptions.rows}
        workloads={workloadOptions.rows}
        editing={editingInstance}
        changingTarget={changingInstanceTarget}
        onChangingTarget={(enabled) => { clearTarget(); setChangingInstanceTarget(enabled); const targetType = editingInstance?.targetType === 'k8s' ? 'k8s' : 'vm'; setInstanceTargetType(targetType); instanceForm.setFieldsValue({ targetType }); }}
        optionsLoading={instanceOptionsLoading}
        optionsError={instanceOptionsError}
        clusterId={selectedClusterId}
        namespace={selectedNamespace}
        workloadKind={selectedWorkloadKind}
        onServiceSearch={setServiceKeyword}
        onTargetSearch={setTargetKeyword}
        onClusterChange={(id) => { setSelectedClusterId(id); setSelectedNamespace(undefined); setSelectedWorkloadKind(undefined); instanceForm.setFieldsValue({ namespace: undefined, workloadKind: undefined, workloadName: undefined }); }}
        onNamespaceChange={(name) => { setSelectedNamespace(name); setSelectedWorkloadKind(undefined); instanceForm.setFieldsValue({ workloadKind: undefined, workloadName: undefined }); }}
        onWorkloadKindChange={(kind) => { setSelectedWorkloadKind(kind); instanceForm.setFieldsValue({ workloadName: undefined }); }}
        allowedTargetTypes={allowedTargetTypes}
        targetType={instanceTargetType}
        onOk={() => void submitInstance()}
        onCancel={() => setInstanceVisible(false)}
        onServiceChange={handleServiceChange}
        onTargetTypeChange={(value) => { clearTarget(); setInstanceTargetType(value); }}
      />
    </PageContainer>
  );
}
