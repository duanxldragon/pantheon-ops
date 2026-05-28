import React, { useCallback, useEffect, useState } from 'react';
import {
  Avatar,
  Button,
  Card,
  Descriptions,
  Form,
  Grid,
  Input,
  Space,
  Tag,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../../components/feedback/message';
import { IconLock, IconUpload, IconUser } from '@arco-design/web-react/icon';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { uploadSystemFile } from '../../../api/upload';
import { isArcoFormValidationError } from '../../../core/arco/formValidation';
import {
  getProfile,
  updateProfile,
  type UserProfile,
  type UserProfileUpdatePayload,
} from '../user/api';
import { formatDateTime } from '../../../core/format/dateTime';
import { useAuthStore } from '../../../store/useAuthStore';
import {
  FormSection,
  PageContainer,
  PageLoading,
  SubmitBar,
} from '../../../components';
import './profile.css';

const Row = Grid.Row;
const Col = Grid.Col;
const FormItem = Form.Item;

const ProfileCenter: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { setUserInfo } = useAuthStore();
  const [loading, setLoading] = useState(false);
  const [savingProfile, setSavingProfile] = useState(false);
  const [uploadingAvatar, setUploadingAvatar] = useState(false);
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [avatarPreview, setAvatarPreview] = useState('');
  const [profileForm] = Form.useForm<UserProfileUpdatePayload>();

  const loadProfile = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getProfile();
      setProfile(result);
      profileForm.setFieldsValue({
        nickname: result.nickname || '',
        avatar: result.avatar || '',
        email: result.email || '',
        phone: result.phone || '',
      });
      setAvatarPreview(result.avatar || '');
      setUserInfo({
        id: result.id,
        username: result.username,
        nickname: result.nickname,
        avatar: result.avatar,
        email: result.email,
        phone: result.phone,
        roles: result.roles,
        perms: result.perms,
        preferences: result.preferences,
      });
    } catch {
      message.error(t('common.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [profileForm, setUserInfo, t]);

  useEffect(() => {
    const timer = globalThis.setTimeout(() => {
      void loadProfile();
    }, 0);
    return () => globalThis.clearTimeout(timer);
  }, [loadProfile]);

  const handleSaveProfile = async () => {
    let values;
    try {
      values = await profileForm.validate();
    } catch (error) {
      if (isArcoFormValidationError(error)) {
        return;
      }
      throw error;
    }
    setSavingProfile(true);
    try {
      const result = await updateProfile(values);
      setProfile(result);
      setAvatarPreview(result.avatar || '');
      setUserInfo({
        id: result.id,
        username: result.username,
        nickname: result.nickname,
        avatar: result.avatar,
        email: result.email,
        phone: result.phone,
        roles: result.roles,
        perms: result.perms,
        preferences: result.preferences,
      });
      message.success(t('common.updateSuccess'));
    } finally {
      setSavingProfile(false);
    }
  };

  const handleUploadAvatar = async (file?: File | null) => {
    if (!file) {
      return;
    }
    setUploadingAvatar(true);
    try {
      const uploaded = await uploadSystemFile(file, 'profile/avatar');
      profileForm.setFieldValue('avatar', uploaded.url);
      setAvatarPreview(uploaded.url);
      message.success(t('system.profile.avatarUploadSuccess'));
    } finally {
      setUploadingAvatar(false);
    }
  };

  if (loading && !profile) {
    return <PageLoading />;
  }

  return (
    <PageContainer className="profile-center-page">
      <Space
        direction="vertical"
        size={16}
        style={{ width: '100%' }}
        className="profile-center-page__stack"
      >
        <Card className="page-panel page-panel--soft profile-center-page__hero">
          <Row gutter={24} align="center">
            <Col xs={24} md={16}>
              <Space align="start" size={16}>
                <Avatar size={56}>
                  {avatarPreview || profile?.avatar ? (
                    <img
                      src={avatarPreview || profile?.avatar}
                      alt={profile?.nickname || profile?.username || 'U'}
                    />
                  ) : (
                    profile?.nickname?.[0] || profile?.username?.[0] || 'U'
                  )}
                </Avatar>
                <Space direction="vertical" size={4}>
                  <Typography.Title heading={5} style={{ margin: 0 }}>
                    {profile?.nickname || profile?.username || t('system.profile.title')}
                  </Typography.Title>
                  <Typography.Text type="secondary">{profile?.username}</Typography.Text>
                  <Space wrap>
                    {profile?.roles?.map((role) => (
                      <Tag key={role} color="arcoblue">
                        {role}
                      </Tag>
                    ))}
                  </Space>
                </Space>
              </Space>
            </Col>
            <Col xs={24} md={8}>
              <Descriptions
                colon=" : "
                column={1}
                data={[
                  { label: t('system.profile.email'), value: profile?.email || '-' },
                  { label: t('system.profile.phone'), value: profile?.phone || '-' },
                  {
                    label: t('system.profile.createdAt'),
                    value: formatDateTime(profile?.createdAt),
                  },
                ]}
              />
            </Col>
          </Row>
        </Card>

        <Card className="page-panel" title={t('system.profile.basicTitle')}>
          <Form
            form={profileForm}
            layout="vertical"
            onSubmit={() => {
              void handleSaveProfile();
            }}
          >
            <FormSection title={t('common.basicInfo')}>
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.profile.username')}>
                    <Input value={profile?.username || ''} disabled prefix={<IconUser />} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.profile.nickname')}
                    field="nickname"
                    rules={[{ required: true, message: t('system.profile.nicknameRequired') }]}
                  >
                    <Input onPressEnter={() => profileForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem
                    label={t('system.profile.email')}
                    field="email"
                    rules={[{ match: /\S+@\S+\.\S+/, message: t('system.user.email.invalid') }]}
                  >
                    <Input onPressEnter={() => profileForm.submit()} />
                  </FormItem>
                </Col>
                <Col xs={24} md={12}>
                  <FormItem label={t('system.profile.phone')} field="phone">
                    <Input onPressEnter={() => profileForm.submit()} />
                  </FormItem>
                </Col>
                <Col span={24}>
                  <FormItem label={t('system.profile.avatar')} field="avatar">
                    <Space direction="vertical" size={8} style={{ width: '100%' }}>
                      <Input
                        placeholder={t('system.profile.avatarPlaceholder')}
                        onChange={(value) => setAvatarPreview(value)}
                        onPressEnter={() => profileForm.submit()}
                      />
                      <Space wrap>
                        <Button
                          icon={<IconUpload />}
                          loading={uploadingAvatar}
                          onClick={() => {
                            const input = document.createElement('input');
                            input.type = 'file';
                            input.accept = 'image/png,image/jpeg,image/jpg,image/webp,image/gif';
                            input.onchange = () => {
                              void handleUploadAvatar(input.files?.[0]);
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
                </Col>
              </Row>
            </FormSection>
            <SubmitBar
              onSubmit={() => {
                void handleSaveProfile();
              }}
              loading={savingProfile}
              submitText={t('system.profile.saveProfile')}
            />
          </Form>
        </Card>

        <Card className="page-panel" title={t('system.profile.securityTab')}>
          <Space direction="vertical" size={8} style={{ width: '100%' }}>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              {t('system.profile.passwordHint')}
            </Typography.Paragraph>
            <Button type="outline" icon={<IconLock />} onClick={() => navigate('/auth/security')}>
              {t('auth.security.title')}
            </Button>
          </Space>
        </Card>
      </Space>
    </PageContainer>
  );
};

export default ProfileCenter;
