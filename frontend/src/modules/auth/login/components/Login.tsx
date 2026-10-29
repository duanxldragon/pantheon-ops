import { useEffect, useMemo, useState } from 'react';
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
import { message } from '../../../../components/feedback/message';
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
} from '../../../../api/request';
import { login, type LoginPayload, type LoginResp } from '../api';
import { verifyMFA } from '../../mfa/api';
import { findFirstNavigableMenuPath } from '../../../system/iam/menu/api';
import { useAuthStore } from '../../../../store/useAuthStore';
import { useMenuStore } from '../../../../store/useMenuStore';
import { useTenantSessionStore } from '../../../../store/useTenantSessionStore';
import PasswordInput from '../../../../components/patterns/forms/PasswordInput';
import ThemeSwitcher from '../../../../core/theme/ThemeSwitcher';
import { clearShellSessionState, LOGIN_NOTICE_STORAGE_KEY } from '../../../../core/shellState';
import { clearTenantViewpoint } from '../../../../core/tenant/viewpoint';
import {
  getBrandInitial,
  setExplicitLanguagePreference,
  usePublicSettings,
} from '../../../../core/setting/publicSettings';
import { COOKIE_TOKEN_PLACEHOLDER } from '../../../../core/auth/sessionSnapshot';
import { invalidateAllRouteWarmData } from '../../../../core/router/prefetch';
import { resolveDefaultAuthedPath } from '../../../../core/router/defaultPath';
import { SUPPORTED_LOCALES, switchI18nLanguage, type SupportedLocale } from '../../../../i18n';
import './Login.css';

const FormItem = Form.Item;
const LOGIN_NOTICE_MESSAGE_KEY_MAP: Record<string, string> = {
  'session.idle_timeout': 'auth.login.idleTimeoutNotice',
};

// The backend answers a refused tenant sign-in with its own `tenant.*` key
// (tenant.forbidden / tenant.suspended / tenant.archived), so those keys are
// rendered verbatim: a generic "login failed" hides the one fact the operator
// needs — whether the account lacks membership, or the tenant itself is not
// accepting logins. Every key admitted here has a translation in all locales.
const LOGIN_PASSTHROUGH_MESSAGE_PREFIXES = [
  'user.login.error.',
  'auth.login.error.',
  'auth.mfa.',
  'tenant.',
];

function resolveLoginErrorKey(error: unknown, fallbackKey: string) {
  if (!isRequestError(error)) {
    return fallbackKey;
  }
  const messageKey = error.messageKey || error.message;
  if (!messageKey) {
    return fallbackKey;
  }
  if (LOGIN_PASSTHROUGH_MESSAGE_PREFIXES.some((prefix) => messageKey.startsWith(prefix))) {
    return messageKey;
  }
  return fallbackKey;
}

// 开发态排障日志：非业务错误(网络/服务端/超时)打到控制台，生产静默。
function logDevLoginError(error: unknown) {
  if (
    import.meta.env.DEV &&
    (!isRequestError(error) || isServerRequestError(error) || isTimeoutRequestError(error))
  ) {
    console.error(error);
  }
}

function canEnterPlatformDashboard(user: { roles?: string[]; perms?: string[] }) {
  return Boolean(user.roles?.includes('admin') || user.perms?.includes('platform:dashboard:view'));
}

interface BrandIdentityProps {
  appName: string;
  brandInitial: string;
  siteLogo?: string;
  subtitle: string;
}

function BrandIdentity({
  appName,
  brandInitial,
  siteLogo,
  subtitle,
}: Readonly<BrandIdentityProps>) {
  return (
    <div className="auth-login-page__brand">
      <div className="auth-login-page__brand-mark">
        {siteLogo ? <img src={siteLogo} alt={appName} /> : brandInitial}
      </div>
      <div className="auth-login-page__brand-copy">
        <span className="auth-login-page__brand-title">{appName}</span>
        <span className="auth-login-page__brand-subtitle">{subtitle}</span>
      </div>
    </div>
  );
}

interface MFAChallengeFieldsProps {
  challenge: LoginResp;
  t: ReturnType<typeof useTranslation>['t'];
  onSubmit: () => void;
}

function MFAChallengeFields({ challenge, t, onSubmit }: Readonly<MFAChallengeFieldsProps>) {
  return (
    <>
      <Alert
        className="auth-login-card__notice"
        type={challenge.setupRequired ? 'warning' : 'info'}
        content={challenge.setupRequired ? t('auth.mfa.setupHint') : t('auth.mfa.verifyHint')}
      />
      {challenge.setupRequired && challenge.totpSecret ? (
        <div className="auth-login-mfa-setup">
          {challenge.totpProvisionUri ? (
            <div className="auth-login-mfa-qr">
              <div className="auth-login-mfa-qr__image" role="img" aria-label={t('auth.mfa.scanQr')}>
                <QRCodeSVG value={challenge.totpProvisionUri} size={168} level="M" includeMargin />
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
            {challenge.totpSecret}
          </Typography.Text>
          {challenge.totpProvisionUri ? (
            <>
              <Typography.Text className="auth-login-mfa-setup__label">
                {t('auth.mfa.provisionUri')}
              </Typography.Text>
              <Typography.Text copyable className="auth-login-mfa-setup__uri">
                {challenge.totpProvisionUri}
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
          autoComplete="one-time-code"
          inputMode="numeric"
          onPressEnter={onSubmit}
        />
      </FormItem>
    </>
  );
}

export function LoginPageComponent() {
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

  const completeLogin = async (
    res: LoginResp & {
      user?: {
        id: number;
        username: string;
        nickname: string;
        avatar?: string;
        email?: string;
        phone?: string;
        roles?: string[];
        perms?: string[];
        preferences?: {
          theme?: 'indigo' | 'emerald' | 'violet' | 'slate';
          language?: 'zh-CN' | 'en-US' | 'ja-JP' | 'ko-KR' | 'fr-FR';
          layoutMode?: 'vertical' | 'horizontal';
          densityMode?: 'comfortable' | 'compact';
        };
      };
    },
  ) => {
    if (!res.user) {
      throw new Error('auth.login.response_invalid');
    }
    clearShellSessionState();
    resetMenuTree();
    // The tenant context is session state: a re-login in the same tab must not
    // inherit the previous subject's plan or viewpoint.
    useTenantSessionStore.getState().resetTenantSession();
    // A viewpoint never crosses a login boundary: the new session decides its
    // own scope (tenant admins resolve their claim at login; platform
    // subjects opt back in explicitly). sessionStorage is per-tab but a
    // re-login in the same tab would otherwise inherit the stale pin.
    clearTenantViewpoint();
    // Same boundary rule for the route warm-data cache: it is keyed by path
    // only, so a sign-in that followed a logout in this tab must not be served
    // the previous subject's warmed rows. The logout path clears it too; this
    // side covers exits that never reached it.
    invalidateAllRouteWarmData();
    setTokens(COOKIE_TOKEN_PLACEHOLDER, COOKIE_TOKEN_PLACEHOLDER);
    setUserInfo(res.user);
    const menuTree = await fetchMenuTree();
    const fallbackMenuPath = findFirstNavigableMenuPath(menuTree);
    const nextPath = resolveDefaultAuthedPath(canEnterPlatformDashboard(res.user), fallbackMenuPath);
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
          tenantId: mfaChallenge.tenantId,
        });
        await completeLogin(res);
        return;
      }

      const res = await login({
        username: values.username,
        password: values.password,
      });
      if (res.tenantSelectionRequired) {
        // W4 (option A): a subject resolves at most one usable tenant — its
        // owning one — so the login picker never renders. A picker envelope
        // can only mean stale server state; refuse instead of guessing.
        throw new Error('auth.login.response_invalid');
      }
      if (res.mfaRequired) {
        setMFAChallenge(res);
        message.info(t('auth.mfa.required'));
        return;
      }
      await completeLogin(res);
    } catch (error) {
      logDevLoginError(error);
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
          <BrandIdentity
            appName={appName}
            brandInitial={brandInitial}
            siteLogo={publicSettings.siteLogo}
            subtitle={t('auth.login.consoleLabel')}
          />

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
          <BrandIdentity
            appName={appName}
            brandInitial={brandInitial}
            siteLogo={publicSettings.siteLogo}
            subtitle={t('auth.login.consoleLabel')}
          />
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
                autoComplete="username"
                onPressEnter={() => form.submit()}
              />
            </FormItem>
            <FormItem
              label={t('auth.password')}
              field="password"
              rules={[{ required: true, message: t('auth.passwordRequired') }]}
            >
              <PasswordInput
                disabled={Boolean(mfaChallenge)}
                prefix={<IconLock />}
                placeholder={t('auth.passwordPlaceholder')}
                size="large"
                autoComplete="current-password"
                onPressEnter={() => form.submit()}
              />
            </FormItem>
            {mfaChallenge ? (
              <MFAChallengeFields challenge={mfaChallenge} t={t} onSubmit={() => form.submit()} />
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
              <Button
                long
                type="text"
                disabled={loading}
                onClick={() => {
                  setMFAChallenge(null);
                }}
              >
                {t('auth.mfa.backToPassword')}
              </Button>
            ) : null}
          </Form>
        </div>
      </section>
    </div>
  );
}
