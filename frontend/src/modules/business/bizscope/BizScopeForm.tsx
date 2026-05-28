import { useEffect } from 'react';
import { Form, Input, Select } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { SubmitBar } from '../../../components';
import type { BizScopePayload } from './api';

const FormItem = Form.Item;

export interface BizScopeFormProps {
  initialValues?: Partial<BizScopePayload>;
  mode: 'create' | 'update';
  submitting?: boolean;
  onSubmit: (values: BizScopePayload) => void | Promise<void>;
  onCancel: () => void;
}

export default function BizScopeForm({
  initialValues,
  mode,
  submitting = false,
  onSubmit,
  onCancel,
}: BizScopeFormProps) {
  const { t } = useTranslation();
  const [form] = Form.useForm<BizScopePayload>();

  useEffect(() => {
    form.setFieldsValue(
      initialValues || {
        environment: 'prod',
        status: 'active',
      },
    );
  }, [form, initialValues]);

  return (
    <Form form={form} layout="vertical" onSubmit={onSubmit}>
      <FormItem
        field="code"
        label={t('business.bizscope.field.code')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input placeholder={t('business.bizscope.field.codePlaceholder')} />
      </FormItem>
      <FormItem
        field="name"
        label={t('business.bizscope.field.name')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Input placeholder={t('business.bizscope.field.namePlaceholder')} />
      </FormItem>
      <FormItem field="owner" label={t('business.bizscope.field.owner')}>
        <Input placeholder={t('business.bizscope.field.ownerPlaceholder')} />
      </FormItem>
      <FormItem
        field="environment"
        label={t('business.bizscope.field.environment')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Select>
          {['dev', 'test', 'prod'].map((item) => (
            <Select.Option key={item} value={item}>
              {t(`business.bizscope.environment.${item}`)}
            </Select.Option>
          ))}
        </Select>
      </FormItem>
      <FormItem
        field="status"
        label={t('business.bizscope.field.status')}
        rules={[{ required: true, message: t('common.required') }]}
      >
        <Select>
          {['active', 'inactive'].map((item) => (
            <Select.Option key={item} value={item}>
              {t(`business.bizscope.status.${item}`)}
            </Select.Option>
          ))}
        </Select>
      </FormItem>
      <FormItem field="remark" label={t('business.bizscope.field.remark')}>
        <Input.TextArea
          autoSize={{ minRows: 2, maxRows: 4 }}
          placeholder={t('business.bizscope.field.remarkPlaceholder')}
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
