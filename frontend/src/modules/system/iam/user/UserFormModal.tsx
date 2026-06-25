import React from 'react';
import { uploadSystemFile } from '../../../../api/upload';
import {
  Avatar,
  Button,
  Form,
  Input,
  Select,
  Space,
  Typography,
} from '@arco-design/web-react';
import { IconUpload } from '@arco-design/web-react/icon';
import { useTranslation } from 'react-i18next';
import { AppModal, FormSection, SubmitBar } from '../../../../components';
import type { FormInstance } from '@arco-design/web-react/es/Form';
import type { UserCreatePayload, UserListRow } from './api';

interface UserFormModalProps {
  visible: boolean;
  editing: UserListRow | null;
  submitting: boolean;
  orgEnabled: boolean;
  orgRequiredForUser: boolean;
  formDeptId: number;
  avatarPreview: string;
  roleOptions: Array<{ label: string; value: number }>;
  deptOptions: Array<{ label: string; value: number }>;
  filteredPostOptions: Array<{ label: string; value: number }>;
  form: FormInstance<UserCreatePayload>;
  onSubmit: () => void;
  onCancel: () => void;
  onDeptChange: (value: unknown) => void;
  onAvatarPreviewChange: (preview: string) => void;
  onAvatarUploadSuccess: (response: unknown) => void;
}

const FormItem = Form.Item;

const UserFormModal: React.FC<UserFormModalProps> = ({
  visible,
  editing,
  submitting,
  orgEnabled,
  orgRequiredForUser,
  formDeptId,
  avatarPreview,
  roleOptions,
  deptOptions,
  filteredPostOptions,
  form,
  onSubmit,
  onCancel,
  onDeptChange,
  onAvatarPreviewChange,
  onAvatarUploadSuccess,
}) => {
  const { t } = useTranslation();

  const handleAvatarUpload = async (file?: File | null) => {
    if (!file) {
      return;
    }
    const uploaded = await uploadSystemFile(file, 'user/avatar');
    onAvatarUploadSuccess(uploaded);
  };

  return (
    <AppModal
      title={editing ? t('system.user.edit') : t('system.user.create')}
      visible={visible}
      size="lg"
      onCancel={onCancel}
      footer={
        <SubmitBar
          onCancel={onCancel}
          onSubmit={onSubmit}
          loading={submitting}
          submitText={editing ? t('common.save') : t('common.add')}
        />
      }
      unmountOnExit
    >
      <Form
        form={form}
        layout="vertical"
        onSubmit={onSubmit}
      >
        <Space direction="vertical" size={20} className="dialog-form-stack">
          <FormSection title={t('common.basicInfo')}>
            <FormItem
              label={t('system.user.username')}
              field="username"
              rules={[{ required: true, message: t('auth.usernameRequired') }]}
            >
              <Input disabled={Boolean(editing)} onPressEnter={() => form.submit()} />
            </FormItem>
            {!editing ? (
              <FormItem
                label={t('system.user.password')}
                field="password"
                rules={[{ required: true, message: t('auth.passwordRequired') }]}
              >
                <Input.Password onPressEnter={() => form.submit()} />
              </FormItem>
            ) : null}
            <FormItem label={t('system.user.nickname')} field="nickname">
              <Input onPressEnter={() => form.submit()} />
            </FormItem>
            <FormItem
              label={t('system.user.email')}
              field="email"
              rules={[
                {
                  validator: (value, callback) => {
                    if (!value || /\S+@\S+\.\S+/.test(String(value))) { // NOSONAR — simple email regex
                      callback();
                      return;
                    }
                    callback(t('system.user.email.invalid'));
                  },
                },
              ]}
            >
              <Input onPressEnter={() => form.submit()} />
            </FormItem>
            <FormItem label={t('system.user.phone')} field="phone">
              <Input onPressEnter={() => form.submit()} />
            </FormItem>
            <FormItem label={t('system.user.avatar')} field="avatar">
              <Space direction="vertical" size={8} style={{ width: '100%' }}>
                <Input
                  placeholder={t('system.profile.avatarPlaceholder')}
                  onChange={(value) => onAvatarPreviewChange(value)}
                />
                <Space align="center" wrap>
                  <Avatar size={40}>
                    {avatarPreview ? (
                      <img src={avatarPreview} alt={t('system.user.avatar')} />
                    ) : (
                      t('common.user').slice(0, 1)
                    )}
                  </Avatar>
                  <Button
                    icon={<IconUpload />}
                    onClick={() => {
                      const input = document.createElement('input');
                      input.type = 'file';
                      input.accept = 'image/png,image/jpeg,image/jpg,image/webp,image/gif';
                      input.onchange = () => {
                        void handleAvatarUpload(input.files?.[0]);
                      };
                      input.click();
                    }}
                  >
                    {t('system.profile.uploadAvatar')}
                  </Button>
                  <Typography.Text type="secondary">
                    {t('system.profile.avatarUploadHint')}
                  </Typography.Text>
                </Space>
              </Space>
            </FormItem>
          </FormSection>
          <FormSection title={t('common.accessControl')}>
            {orgEnabled ? (
              <>
                <FormItem
                  label={t('system.user.dept')}
                  field="deptId"
                  rules={
                    orgRequiredForUser
                      ? [
                          {
                            validator: (value, callback) => {
                              if (Number(value || 0) > 0) {
                                callback();
                                return;
                              }
                              callback(t('system.user.dept.required'));
                            },
                          },
                        ]
                      : undefined
                  }
                >
                  <Select options={deptOptions} onChange={onDeptChange} />
                </FormItem>
                <FormItem label={t('system.user.post')} field="postId">
                  <Select options={filteredPostOptions} disabled={formDeptId === 0} />
                </FormItem>
              </>
            ) : null}
            <FormItem label={t('system.user.status')} field="status">
              <Select
                options={[
                  { label: t('system.user.status.enabled'), value: 1 },
                  { label: t('system.user.status.disabled'), value: 2 },
                ]}
              />
            </FormItem>
            <FormItem
              className="system-user-list__role-field"
              label={t('system.user.roles')}
              field="roleIds"
            >
              <Select mode="multiple" allowClear options={roleOptions} />
            </FormItem>
          </FormSection>
        </Space>
      </Form>
    </AppModal>
  );
};

export default UserFormModal;
