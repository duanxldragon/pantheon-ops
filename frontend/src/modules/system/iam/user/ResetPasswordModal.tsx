import React from 'react';
import { Form, Input, Space, Tag } from '@arco-design/web-react';
import { useTranslation } from 'react-i18next';
import { isArcoFormValidationError } from '../../../../core/arco/formValidation';
import { AppModal, SubmitBar } from '../../../../components';
import type { UserListRow } from './api';

interface ResetPasswordFormValues {
  newPassword: string;
  confirmPassword: string;
}

interface ResetPasswordModalProps {
  visible: boolean;
  target: UserListRow | null;
  submitting: boolean;
  form: ReturnType<typeof Form.useForm<ResetPasswordFormValues>>[0];
  onSubmit: (newPassword: string) => void;
  onCancel: () => void;
}

const FormItem = Form.Item;

const ResetPasswordModal: React.FC<ResetPasswordModalProps> = ({
  visible,
  target,
  submitting,
  form,
  onSubmit,
  onCancel,
}) => {
  const { t } = useTranslation();

  const handleSubmit = async () => {
    let values;
    try {
      values = await form.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      return;
    }
    onSubmit(values.newPassword);
  };

  const handleCancel = () => {
    onCancel();
    form.resetFields();
  };

  return (
    <AppModal
      title={t('system.user.resetPasswordTitle')}
      visible={visible}
      size="sm"
      onCancel={handleCancel}
      footer={
        <SubmitBar
          onCancel={handleCancel}
          onSubmit={handleSubmit}
          loading={submitting}
          submitText={t('system.user.resetPassword')}
        />
      }
      unmountOnExit
    >
      <Form
        form={form}
        layout="vertical"
        onSubmit={handleSubmit}
      >
        <Space direction="vertical" size={16} className="dialog-form-stack">
          <FormItem label={t('system.user.resetPasswordTarget')}>
            <Input
              value={target ? `${target.username} / ${target.nickname || '-'}` : ''}
              disabled
            />
          </FormItem>
          <FormItem
            label={t('system.user.newPassword')}
            field="newPassword"
            rules={[{ required: true, message: t('auth.passwordRequired') }]}
          >
            <Input.Password onPressEnter={() => form.submit()} />
          </FormItem>
          <FormItem
            label={t('system.user.confirmPassword')}
            field="confirmPassword"
            rules={[
              { required: true, message: t('system.profile.confirmPasswordRequired') },
              {
                validator: (value, callback) => {
                  const nextPassword = form.getFieldValue('newPassword');
                  if (!value || value === nextPassword) {
                    callback();
                    return;
                  }
                  callback(t('system.profile.confirmPasswordMismatch'));
                },
              },
            ]}
          >
            <Input.Password onPressEnter={() => form.submit()} />
          </FormItem>
          <Tag color="orange">{t('system.user.resetPasswordHint')}</Tag>
        </Space>
      </Form>
    </AppModal>
  );
};

export default ResetPasswordModal;
