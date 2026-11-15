import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Message, Select } from '@arco-design/web-react';
import { AppModal, PageError, PageForbidden, PageLoading } from '../../../components';
import { isForbiddenRequestError } from '../../../api/request';
import { getHostList, type HostRow } from '../cmdb/host/api';
import { getK8sClusterList, type K8sClusterRow } from '../k8s/api';
import { createServiceInstance, getServiceList, type ServiceRow } from '../service/api';

interface InstanceValues {
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

interface InstanceCreateModalProps {
  readonly onClose: () => void;
  readonly onCreated: () => void;
}

export default function InstanceCreateModal({ onClose, onCreated }: InstanceCreateModalProps) {
  const { t } = useTranslation();
  const [form] = Form.useForm<InstanceValues>();
  const [services, setServices] = useState<ServiceRow[]>([]);
  const [serviceId, setServiceId] = useState<number>();
  const [targetType, setTargetType] = useState<'vm' | 'k8s'>('vm');
  const [hosts, setHosts] = useState<HostRow[]>([]);
  const [clusters, setClusters] = useState<K8sClusterRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [targetLoading, setTargetLoading] = useState(false);
  const [targetError, setTargetError] = useState<unknown>(null);
  const [revision, setRevision] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const selectedService = services.find((service) => service.id === serviceId);

  useEffect(() => {
    const controller = new AbortController();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setLoading(true);
      setError(null);
      void getServiceList({ status: 'active', page: 1, pageSize: 100 }, controller.signal).then((response) => {
        if (!controller.signal.aborted) setServices(response.items);
      }).catch((requestError: unknown) => {
        if (!controller.signal.aborted) setError(requestError);
      }).finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    });
    return () => controller.abort();
  }, [revision]);

  useEffect(() => {
    if (!selectedService) return;
    const controller = new AbortController();
    queueMicrotask(() => {
      if (controller.signal.aborted) return;
      setTargetLoading(true);
      setTargetError(null);
      setHosts([]);
      setClusters([]);
      const request = targetType === 'vm'
        ? getHostList({ businessScopeId: selectedService.businessScopeId, page: 1, pageSize: 100 }, controller.signal).then((response) => { if (!controller.signal.aborted) setHosts(response.items); })
        : getK8sClusterList({ businessScopeId: selectedService.businessScopeId, page: 1, pageSize: 100 }).then((response) => { if (!controller.signal.aborted) setClusters(response.items); });
      void request.catch((requestError: unknown) => {
        if (!controller.signal.aborted) setTargetError(requestError);
      }).finally(() => {
        if (!controller.signal.aborted) setTargetLoading(false);
      });
    });
    return () => controller.abort();
  }, [selectedService, targetType, revision]);

  const clearTargetFields = () => form.setFieldsValue({ hostId: undefined, k8sClusterId: undefined, namespace: undefined, workloadKind: undefined, workloadName: undefined });
  const selectService = (id: number) => {
    const service = services.find((row) => row.id === id);
    const nextTargetType = service?.runtimeType === 'k8s' ? 'k8s' : 'vm';
    setServiceId(id);
    setTargetType(nextTargetType);
    setTargetError(null);
    setTargetLoading(true);
    clearTargetFields();
    form.setFieldsValue({ targetType: nextTargetType });
  };
  const selectTargetType = (value: 'vm' | 'k8s') => {
    setTargetType(value);
    setTargetLoading(true);
    setTargetError(null);
    clearTargetFields();
  };
  const submit = async () => {
    const values = await form.validate();
    setSubmitting(true);
    try {
      await createServiceInstance({ ...values, status: 'active' });
      Message.success(t('business.service.catalog.instanceCreateSuccess'));
      onCreated();
    } catch { Message.error(t('business.service.catalog.instanceCreateFailed')); }
    finally { setSubmitting(false); }
  };
  const allowedTargets = selectedService?.runtimeType === 'hybrid' ? ['vm', 'k8s'] : [selectedService?.runtimeType || 'vm'];

  return (
    <AppModal visible title={t('business.service.catalog.addInstance')} okText={t('common.save')} cancelText={t('common.cancel')} confirmLoading={submitting} okButtonProps={{ disabled: loading || Boolean(error) || !selectedService || targetLoading || Boolean(targetError) }} onOk={submit} onCancel={onClose}>
      {loading ? <PageLoading /> : null}
      {!loading && error ? <PageError onRetry={() => setRevision((value) => value + 1)} /> : null}
      {!loading && !error ? (
        <Form form={form} layout="vertical" initialValues={{ environment: 'prod', targetType: 'vm' }}>
          <Form.Item field="serviceId" label={t('business.service.catalog.field.service')} rules={[{ required: true }]}>
            <Select showSearch onChange={selectService}>{services.map((service) => <Select.Option key={service.id} value={service.id}>{service.applicationName} / {service.name}</Select.Option>)}</Select>
          </Form.Item>
          <Form.Item field="environment" label={t('business.service.instance.environment')} rules={[{ required: true }]}>
            <Select>{['dev', 'test', 'prod'].map((value) => <Select.Option key={value} value={value}>{t(`business.bizscope.environment.${value}`)}</Select.Option>)}</Select>
          </Form.Item>
          <Form.Item field="targetType" label={t('business.service.instance.target')} rules={[{ required: true }]}>
            <Select disabled={!selectedService} onChange={selectTargetType}>{allowedTargets.map((value) => <Select.Option key={value} value={value}>{t(`business.service.catalog.runtime.${value}`)}</Select.Option>)}</Select>
          </Form.Item>
          {targetLoading ? <PageLoading /> : null}
          {isForbiddenRequestError(targetError) ? <PageForbidden /> : null}
          {targetError && !isForbiddenRequestError(targetError) ? <PageError onRetry={() => setRevision((value) => value + 1)} /> : null}
          {selectedService && !targetLoading && !targetError && targetType === 'vm' ? (
            <Form.Item field="hostId" label={t('business.service.catalog.host')} rules={[{ required: true }]}>
              <Select showSearch>{hosts.map((host) => <Select.Option key={host.id} value={host.id}>{host.hostname} ({host.ip})</Select.Option>)}</Select>
            </Form.Item>
          ) : null}
          {selectedService && !targetLoading && !targetError && targetType === 'k8s' ? (
            <>
              <Form.Item field="k8sClusterId" label={t('business.service.catalog.field.cluster')} rules={[{ required: true }]}><Select showSearch>{clusters.map((cluster) => <Select.Option key={cluster.id} value={cluster.id}>{cluster.name}</Select.Option>)}</Select></Form.Item>
              <Form.Item field="namespace" label={t('business.service.catalog.field.namespace')} rules={[{ required: true }]}><Input /></Form.Item>
              <Form.Item field="workloadKind" label={t('business.service.catalog.field.workloadKind')} rules={[{ required: true }]}><Select>{['Deployment', 'StatefulSet', 'DaemonSet'].map((value) => <Select.Option key={value} value={value}>{value}</Select.Option>)}</Select></Form.Item>
              <Form.Item field="workloadName" label={t('business.service.catalog.field.workloadName')} rules={[{ required: true }]}><Input /></Form.Item>
            </>
          ) : null}
          <Form.Item field="desiredVersion" label={t('business.service.instance.version')}><Input /></Form.Item>
        </Form>
      ) : null}
    </AppModal>
  );
}
