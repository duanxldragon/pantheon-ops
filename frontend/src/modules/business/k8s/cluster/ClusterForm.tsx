import { useEffect, useState } from 'react';
import { Form, Input, Select } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { SubmitBar } from '../../../../components';
import { getBizScopeOptions } from '../../bizscope/api';

const FormItem = Form.Item;

export interface ClusterFormValues {
  code: string;
  name: string;
  environment: string;
  businessScopeId?: number;
  kubeconfig: string;
  remark?: string;
}

export interface ClusterFormProps {
  readonly initialValues?: Partial<ClusterFormValues>;
  readonly mode: 'create' | 'update';
  readonly submitting?: boolean;
  readonly onSubmit: (values: ClusterFormValues) => void | Promise<void>;
  readonly onCancel: () => void;
}

export default function ClusterForm({
  initialValues,
  mode,
  submitting = false,
  onSubmit,
  onCancel,
}: ClusterFormProps) {
  const { t } = useTranslation();
  const [form] = Form.useForm<ClusterFormValues>();
  const [scopeOptions, setScopeOptions] = useState<Array<{ label: string; value: number }>>([]);

  useEffect(() => {
    form.setFieldsValue(
      initialValues || {
        environment: 'prod',
      },
    );
  }, [form, initialValues]);

  useEffect(() => {
    let mounted = true;
    getBizScopeOptions()
      .then((items) => {
        if (!mounted) return;
        setScopeOptions(items.map((item) => ({ label: item.name, value: item.id })));
      })
      .catch(() => {
        // options load is best-effort; the form remains usable without them
      });
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <Form form={form} layout="vertical" onSubmit={onSubmit}>
      <FormItem
        field="code"
        label={t('business.k8s.cluster.field.code')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input placeholder={t('business.k8s.cluster.field.codePlaceholder')} disabled={mode === 'update'} />
      </FormItem>
      <FormItem
        field="name"
        label={t('business.k8s.cluster.field.name')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input placeholder={t('business.k8s.cluster.field.namePlaceholder')} />
      </FormItem>
      <FormItem
        field="environment"
        label={t('business.k8s.cluster.field.environment')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Select>
          {['dev', 'test', 'prod'].map((item) => (
            <Select.Option key={item} value={item}>
              {t(`business.k8s.cluster.environment.${item}`)}
            </Select.Option>
          ))}
        </Select>
      </FormItem>
      <FormItem field="businessScopeId" label={t('business.k8s.cluster.field.businessScope')}>
        <Select allowClear placeholder={t('business.k8s.cluster.field.businessScopePlaceholder')}>
          {scopeOptions.map((item) => (
            <Select.Option key={item.value} value={item.value}>
              {item.label}
            </Select.Option>
          ))}
        </Select>
      </FormItem>
      <FormItem
        field="kubeconfig"
        label={t('business.k8s.cluster.field.kubeconfig')}
        rules={mode === 'create' ? [{ required: true, message: t('common.required') }] : undefined}
        extra={mode === 'update' ? t('business.k8s.cluster.field.kubeconfigUpdateHint') : undefined}
      >
        <Input.TextArea
          autoSize={{ minRows: 4, maxRows: 10 }}
          placeholder={t('business.k8s.cluster.field.kubeconfigPlaceholder')}
        />
      </FormItem>
      <FormItem field="remark" label={t('business.k8s.cluster.field.remark')}>
        <Input.TextArea
          autoSize={{ minRows: 2, maxRows: 4 }}
          placeholder={t('business.k8s.cluster.field.remarkPlaceholder')}
        />
      </FormItem>
      <SubmitBar
        loading={submitting}
        onCancel={onCancel}
        submitText={mode === 'create' ? t('common.create') : t('common.save')}
      />
    </Form>
  );
}
