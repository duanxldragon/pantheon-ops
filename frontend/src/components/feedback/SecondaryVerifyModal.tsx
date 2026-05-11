import React, { useState } from 'react';
import { Form, Input } from '@arco-design/web-react';
import { message } from './message';
import { useTranslation } from 'react-i18next';
import { verifyOperationPassword } from '../../modules/auth/api';
import AppModal from '../patterns/AppModal';

interface SecondaryVerifyModalProps {
  visible: boolean;
  onSuccess: (token: string) => void;
  onCancel: () => void;
}

export const SecondaryVerifyModal: React.FC<SecondaryVerifyModalProps> = ({
  visible,
  onSuccess,
  onCancel,
}) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [form] = Form.useForm();

  const handleOk = async () => {
    try {
      const values = await form.validate();
      setLoading(true);
      const resp = await verifyOperationPassword(values.password);
      message.success(t('auth.operation.verify_success'));
      onSuccess(resp.operationToken);
      form.resetFields();
    } catch {
      // 错误已由拦截器处理或表单校验失败
    } finally {
      setLoading(false);
    }
  };

  return (
    <AppModal
      title={t('auth.operation.verify_title')}
      visible={visible}
      onOk={handleOk}
      onCancel={() => {
        form.resetFields();
        onCancel();
      }}
      confirmLoading={loading}
      unmountOnExit
      size="sm"
    >
      <div style={{ marginBottom: 16 }}>{t('auth.operation.verify_hint')}</div>
      <Form
        form={form}
        layout="vertical"
        onSubmit={() => {
          void handleOk();
        }}
      >
        <Form.Item
          label={t('system.user.password')}
          field="password"
          rules={[{ required: true, message: t('auth.passwordRequired') }]}
        >
          <Input.Password
            placeholder={t('auth.passwordPlaceholder')}
            onPressEnter={() => form.submit()}
          />
        </Form.Item>
      </Form>
    </AppModal>
  );
};
