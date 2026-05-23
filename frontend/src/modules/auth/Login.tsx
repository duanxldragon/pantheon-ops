import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Form,
  Input,
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
} from '@arco-design/web-react';
import { message } from '../../components/feedback/message';
import {
  IconCheckCircle,
  IconLanguage,
  IconLock,
  IconQrcode,
  IconSafe,
  IconStorage,
  IconUser,
} from '@arco-design/web-react/icon';
import { QRCodeSVG } from 'qrcode.react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  endLogoutTransition,
  isRequestError,
  isServerRequestError,
  isTimeoutRequestError,
} from '../../api/request';
import { login, verifyMFA, type LoginPayload, type LoginResp } from './api';
import { findFirstNavigableMenuPath } from '../system/menu/api';
import { useAuthStore } from '../../store/useAuthStore';
import { useMenuStore } from '../../store/useMenuStore';
import ThemeSwitcher from '../../core/theme/ThemeSwitcher';
import { clearShellSessionState, LOGIN_NOTICE_STORAGE_KEY } from '../../core/shellState';
import {
  getBrandInitial,
  setExplicitLanguagePreference,
  usePublicSettings,
} from '../../core/settings/publicSettings';
import { SUPPORTED_LOCALES, switchI18nLanguage, type SupportedLocale } from '../../i18n';
import './Login.css';

const FormItem = Form.Item;
const LOGIN_NOTICE_MESSAGE_KEY_MAP: Record<string, string> = {
  'session.idle_timeout': 'auth.login.idleTimeoutNotice',
};

function resolvePostLoginPath(hasDashboardPermission: boolean, fallbackMenuPath: string | null) {
  if (hasDashboardPermission) {
    return '/dashboard';
  }
  if (fallbackMenuPath) {
    return fallbackMenuPath;
  }
  return '/dashboard';
}

function resolveLoginErrorKey(error: unknown, fallbackKey: string) {
  if (!isRequestError(error)) {
    return fallbackKey;
  }
  const messageKey = error.messageKey || error.message;
  if (!messageKey) {
    return fallbackKey;
  }
  if (
    messageKey.startsWith('user.login.error.') ||
    messageKey.startsWith('auth.login.error.') ||
    messageKey.startsWith('auth.mfa.')
  ) {
    return messageKey;
  }
  return fallbackKey;
}

const LoginPage: React.FC = () => {
  const [form] = Form.useForm<LoginPayload & { mfaCode?: string }>();
  const [loading, setLoading] = useState(false);
  const [mfaChallenge, setMFAChallenge] = useState<LoginResp | null>(null);
  const [loginNotice] = useState<string | null>(() =>
    sessionStorage.getItem(LOGIN_NOTICE_STORAGE_KEY),
  );
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();
  const { setTokens, setUserInfo } = useAuthStore();
  const { fetchMenuTree, resetMenuTree } = useMenuStore();
  const publicSettings = usePublicSettings();
  const currentLang = (
    SUPPORTED_LOCALES.includes(i18n.language as SupportedLocale) ? i18n.language : 'zh-CN'
  ) as SupportedLocale;
  const appName = publicSettings.siteName || t('app.name');
  const brandInitial = getBrandInitial(appName);
  const featureItems = useMemo(
    () => [
      { icon: <IconCheckCircle />, label: t('auth.login.feature.modules') },
      { icon: <IconSafe />, label: t('auth.login.feature.security') },
      { icon: <IconStorage />, label: t('auth.login.feature.i18n') },
    ],
    [t],
  );

  useEffect(() => {
    if (!loginNotice) {
      return;
    }
    sessionStorage.removeItem(LOGIN_NOTICE_STORAGE_KEY);
  }, [loginNotice]);

  const completeLogin = async (res: LoginResp) => {
    if (!res.accessToken || !res.refreshToken || !res.user) {
      throw new Error('auth.login.response_invalid');
    }
    clearShellSessionState();
    resetMenuTree();
    setTokens(res.accessToken, res.refreshToken);
    setUserInfo(res.user);
    const menuTree = await fetchMenuTree();
    const fallbackMenuPath = findFirstNavigableMenuPath(menuTree);
    const nextPath = resolvePostLoginPath(
      Boolean(
        res.user.roles?.includes('admin') || res.user.perms?.includes('platform:dashboard:view'),
      ),
      fallbackMenuPath,
    );
    message.success(t('auth.loginSuccess'));
    navigate(nextPath, { replace: true });
  };

  const onSubmit = async (values: LoginPayload & { mfaCode?: string }) => {
    setLoading(true);
    try {
      endLogoutTransition();
      if (mfaChallenge?.challengeId) {
        const res = await verifyMFA({
          challengeId: mfaChallenge.challengeId,
          code: values.mfaCode || '',
        });
        await completeLogin(res);
        return;
      }

      const res: LoginResp = await login(values);
      if (res.mfaRequired) {
        setMFAChallenge(res);
        message.info(t('auth.mfa.required'));
        return;
      }
      await completeLogin(res);
    } catch (error) {
      if (
        import.meta.env.DEV &&
        (!isRequestError(error) || isServerRequestError(error) || isTimeoutRequestError(error))
      ) {
        console.error(error);
      }
      const fallbackKey = mfaChallenge ? 'auth.mfa.verifyFailed' : 'auth.loginFailed';
      message.error(t(resolveLoginErrorKey(error, fallbackKey)));
    } finally {
      setLoading(false);
    }
  };

  const changeLanguage = (language: string) => {
    const nextLang = language as SupportedLocale;
    if (!SUPPORTED_LOCALES.includes(nextLang) || nextLang === currentLang) {
      return;
    }
    setExplicitLanguagePreference(nextLang);
    void switchI18nLanguage(nextLang);
  };

  const loginNoticeText = loginNotice
    ? t(LOGIN_NOTICE_MESSAGE_KEY_MAP[loginNotice] || loginNotice, {
        defaultValue: t('auth.login.idleTimeoutNotice'),
      })
    : null;

  return (
    <div className="auth-login-page">
      <section className="auth-login-page__brand-pane">
        <div className="auth-login-page__brand-inner">
          <div className="auth-login-page__brand">
            <div className="auth-login-page__brand-mark">
              {publicSettings.siteLogo ? (
                <img src={publicSettings.siteLogo} alt={appName} />
              ) : (
                brandInitial
              )}
            </div>
            <div className="auth-login-page__brand-copy">
              <span className="auth-login-page__brand-title">{appName}</span>
              <span className="auth-login-page__brand-subtitle">
                {t('auth.login.consoleLabel')}
              </span>
            </div>
          </div>

          <div className="auth-login-page__intro">
            <Tag color="arcoblue" bordered={false} className="auth-login-page__tag">
              {t('auth.login.entryTag')}
            </Tag>
            <Typography.Title heading={2} className="auth-login-page__headline">
              {t('auth.login.visualTitle')}
            </Typography.Title>
            <Typography.Paragraph className="auth-login-page__description">
              {t('auth.login.visualDesc')}
            </Typography.Paragraph>
          </div>

          <Space wrap className="auth-login-page__features">
            {featureItems.map((item) => (
              <Tag key={item.label} bordered={false} color="gray" icon={item.icon}>
                {item.label}
              </Tag>
            ))}
          </Space>

          <div className="auth-login-page__footer">{t('app.footer')}</div>
        </div>
      </section>

      <section className="auth-login-page__form-pane" aria-label={t('auth.login.visualAria')}>
        <div className="auth-login-page__mobile-brand">
          <div className="auth-login-page__brand">
            <div className="auth-login-page__brand-mark">
              {publicSettings.siteLogo ? (
                <img src={publicSettings.siteLogo} alt={appName} />
              ) : (
                brandInitial
              )}
            </div>
            <div className="auth-login-page__brand-copy">
              <span className="auth-login-page__brand-title">{appName}</span>
              <span className="auth-login-page__brand-subtitle">
                {t('auth.login.consoleLabel')}
              </span>
            </div>
          </div>
        </div>

        <div className="auth-login-page__tools">
          <ThemeSwitcher className="auth-login-page__tool-btn" />
          <Tooltip content={t('app.toggleLanguage')}>
            <Select
              size="small"
              className="auth-login-page__tool-btn"
              value={currentLang}
              prefix={<IconLanguage />}
              bordered={false}
              triggerProps={{ autoAlignPopupMinWidth: true }}
              onChange={changeLanguage}
            >
              {SUPPORTED_LOCALES.map((language) => (
                <Select.Option key={language} value={language}>
                  {t(`app.language.${language}`)}
                </Select.Option>
              ))}
            </Select>
          </Tooltip>
        </div>

        <div className="auth-login-card">
          <div className="auth-login-card__header">
            <Tag color="arcoblue" bordered={false} className="auth-login-card__tag">
              {t('auth.login.consoleTitle')}
            </Tag>
            <Typography.Title heading={3} className="auth-login-card__title">
              {t('auth.login.title')}
            </Typography.Title>
            <Typography.Paragraph className="auth-login-card__subtitle">
              {t('auth.login.subtitle')}
            </Typography.Paragraph>
          </div>

          {loginNoticeText ? (
            <Alert className="auth-login-card__notice" type="warning" content={loginNoticeText} />
          ) : null}
          <Alert
            className="auth-login-card__notice"
            type="info"
            content={t('auth.login.securityNotice')}
          />

          <Form form={form} layout="vertical" onSubmit={onSubmit}>
            <FormItem
              label={t('auth.username')}
              field="username"
              rules={[{ required: true, message: t('auth.usernameRequired') }]}
            >
              <Input
                disabled={Boolean(mfaChallenge)}
                prefix={<IconUser />}
                placeholder={t('auth.usernamePlaceholder')}
                size="large"
                onPressEnter={() => form.submit()}
              />
            </FormItem>
            <FormItem
              label={t('auth.password')}
              field="password"
              rules={[{ required: true, message: t('auth.passwordRequired') }]}
            >
              <Input.Password
                disabled={Boolean(mfaChallenge)}
                prefix={<IconLock />}
                placeholder={t('auth.passwordPlaceholder')}
                size="large"
                onPressEnter={() => form.submit()}
              />
            </FormItem>
            {mfaChallenge ? (
              <>
                <Alert
                  className="auth-login-card__notice"
                  type={mfaChallenge.setupRequired ? 'warning' : 'info'}
                  content={
                    mfaChallenge.setupRequired ? t('auth.mfa.setupHint') : t('auth.mfa.verifyHint')
                  }
                />
                {mfaChallenge.setupRequired && mfaChallenge.totpSecret ? (
                  <div className="auth-login-mfa-setup">
                    {mfaChallenge.totpProvisionUri ? (
                      <div className="auth-login-mfa-qr">
                        <div className="auth-login-mfa-qr__image" aria-label={t('auth.mfa.scanQr')}>
                          <QRCodeSVG
                            value={mfaChallenge.totpProvisionUri}
                            size={168}
                            level="M"
                            includeMargin
                          />
                        </div>
                        <Typography.Text className="auth-login-mfa-qr__hint">
                          <IconQrcode />
                          {t('auth.mfa.scanQr')}
                        </Typography.Text>
                      </div>
                    ) : null}
                    <Typography.Text className="auth-login-mfa-setup__label">
                      {t('auth.mfa.manualSecret')}
                    </Typography.Text>
                    <Typography.Text copyable className="auth-login-mfa-setup__secret">
                      {mfaChallenge.totpSecret}
                    </Typography.Text>
                    {mfaChallenge.totpProvisionUri ? (
                      <>
                        <Typography.Text className="auth-login-mfa-setup__label">
                          {t('auth.mfa.provisionUri')}
                        </Typography.Text>
                        <Typography.Text copyable className="auth-login-mfa-setup__uri">
                          {mfaChallenge.totpProvisionUri}
                        </Typography.Text>
                      </>
                    ) : null}
                  </div>
                ) : null}
                <FormItem
                  label={t('auth.mfa.code')}
                  field="mfaCode"
                  rules={[{ required: true, message: t('auth.mfa.codeRequired') }]}
                >
                  <Input
                    placeholder={t('auth.mfa.codePlaceholder')}
                    size="large"
                    maxLength={6}
                    onPressEnter={() => form.submit()}
                  />
                </FormItem>
              </>
            ) : null}
            <Button
              type="primary"
              htmlType="submit"
              long
              loading={loading}
              className="auth-login-card__submit"
            >
              {mfaChallenge ? t('auth.mfa.verifyAndSignIn') : t('auth.signIn')}
            </Button>
            {mfaChallenge ? (
              <Button long type="text" disabled={loading} onClick={() => setMFAChallenge(null)}>
                {t('auth.mfa.backToPassword')}
              </Button>
            ) : null}
          </Form>
        </div>
      </section>
    </div>
  );
};

export default LoginPage;
