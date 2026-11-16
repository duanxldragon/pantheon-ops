import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Select, Button, Space } from '@arco-design/web-react';
import type { EnvironmentPayload, EnvironmentRow } from './api';

interface Props {
  mode: 'create' | 'update';
  initialValues?: EnvironmentRow;
  submitting?: boolean;
  onSubmit: (values: EnvironmentPayload) => void | Promise<void>;
  onCancel: () => void;
}

export default function EnvironmentForm({ mode, initialValues, submitting, onSubmit, onCancel }: Props) {
  const { t } = useTranslation();
  const [form] = Form.useForm<EnvironmentPayload>();
  const key = (suffix: string) => `business.bizscope.environment.${suffix}`;

  useEffect(() => {
    if (mode === 'update' && initialValues) {
      form.setFieldsValue({
        code: initialValues.code,
        name: initialValues.name,
        envType: initialValues.envType,
        status: initialValues.status,
        remark: initialValues.remark,
      });
    } else {
      form.setFieldsValue({ envType: 'dev', status: 'active' });
    }
  }, [form, initialValues, mode]);

  return (
    <Form form={form} layout="vertical" onSubmit={onSubmit} onSubmitCapture={() => undefined}>
      <Form.Item label={t(key('field.code'))} field="code" rules={[{ required: true, message: t(key('codeRequired')) }]}>
        <Input placeholder={t(key('field.code'))} disabled={mode === 'update'} maxLength={50} />
      </Form.Item>
      <Form.Item label={t(key('field.name'))} field="name" rules={[{ required: true, message: t(key('nameRequired')) }]}>
        <Input placeholder={t(key('field.name'))} maxLength={255} />
      </Form.Item>
      <Form.Item label={t(key('field.envType'))} field="envType" rules={[{ required: true, message: t(key('envTypeInvalid')) }]}>
        <Select>
          <Select.Option value="dev">{t(key('dev'))}</Select.Option>
          <Select.Option value="test">{t(key('test'))}</Select.Option>
          <Select.Option value="prod">{t(key('prod'))}</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label={t('business.bizscope.field.status')} field="status" rules={[{ required: true, message: t('business.bizscope.environment.statusInvalid') }]}>
        <Select>
          <Select.Option value="active">{t('business.bizscope.status.active')}</Select.Option>
          <Select.Option value="inactive">{t('business.bizscope.status.inactive')}</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label={t(key('field.remark'))} field="remark">
        <Input.TextArea rows={3} placeholder={t(key('field.remark'))} />
      </Form.Item>
      <Form.Item label="">
        <Space style={{ justifyContent: 'flex-end', width: '100%' }}>
          <Button onClick={onCancel}>{t('common.cancel')}</Button>
          <Button type="primary" htmlType="submit" loading={submitting}>
            {t('common.confirm')}
          </Button>
        </Space>
      </Form.Item>
    </Form>
  );
}