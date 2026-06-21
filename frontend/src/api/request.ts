import axios from 'axios';
import type { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { message as feedbackMessage } from '../components/feedback/message';
import i18n from 'i18next';
import { useAuthStore } from '../store/useAuthStore';
import { clearShellSessionState, persistLoginNotice } from '../core/shellState';
import { clearExplicitLanguagePreference } from '../core/settings/languagePreference';
import { clearPantheonThemePreference } from '../core/theme/theme';
import { showSecondaryVerify } from '../components/feedback/secondaryVerifyController';
import { switchI18nLanguage } from '../i18n';

export interface RequestConfig extends AxiosRequestConfig {
  _retry?: boolean;
  _opRetry?: boolean;
  skipAuthRefresh?: boolean;
  skipErrorMessage?: boolean;
}

export type RequestErrorKind =
  | 'business'
  | 'unauthorized'
  | 'forbidden'
  | 'server'
  | 'network'
  | 'timeout'
  | 'unknown';

export class RequestError extends Error {
  kind: RequestErrorKind;
  code?: number;
  status?: number;
  messageKey?: string;

  constructor(options: {
    kind: RequestErrorKind;
    message?: string;
    code?: number;
    status?: number;
    messageKey?: string;
  }) {
    super(options.messageKey || options.message || 'request.failed');
    this.name = 'RequestError';
    this.kind = options.kind;
    this.code = options.code;
    this.status = options.status;
    this.messageKey = options.messageKey;
  }
}

const I18N_KEY_PATTERN = /^[a-z0-9_]+(?:\.[a-z0-9_]+)+$/i;

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  withCredentials: true,
});

function readCsrfToken(): string {
  const match = document.cookie.match(/(?:^|;\s*)pantheon_csrf_token=([^;]+)/);
  return match ? decodeURIComponent(match[1]) : '';
}

let refreshPromise: Promise<string | null> | null = null;
let logoutTransition = false;

const clearClientSession = () => {
  axios.post('/api/v1/auth/logout', {}, { withCredentials: true }).catch(() => undefined);
  clearShellSessionState();
  clearPantheonThemePreference();
  const nextLanguage = clearExplicitLanguagePreference();
  void switchI18nLanguage(nextLanguage).catch(() => undefined);
  useAuthStore.getState().clearAuth();
};

export const beginLogoutTransition = () => {
  logoutTransition = true;
};

export const endLogoutTransition = () => {
  logoutTransition = false;
};

export const isLogoutTransitionActive = () => logoutTransition;

const redirectToLogin = () => {
  clearClientSession();
  if (globalThis.location.pathname !== '/login') {
    globalThis.location.href = '/login';
  }
};

const shouldRefresh = (config?: RequestConfig, code?: number) => {
  if (!config || config.skipAuthRefresh || config._retry || logoutTransition) {
    return false;
  }
  if (
    config.url?.includes('/system/login') ||
    config.url?.includes('/system/refresh') ||
    config.url?.includes('/auth/login') ||
    config.url?.includes('/auth/refresh')
  ) {
    return false;
  }
  return code === 401;
};

const doRefreshToken = async (): Promise<string | null> => {
  if (!refreshPromise) {
    refreshPromise = axios
      .post(
        '/api/v1/auth/refresh',
        {},
        {
          withCredentials: true,
          headers: {
            'Accept-Language': localStorage.getItem('pantheon_lang') || 'zh-CN',
          },
        },
      )
      .then((response) => {
        const { code, data } = response.data;
        if (code !== 200) {
          throw new Error(response.data.message || 'request.failed');
        }
        return data;
      })
      .then((data) => {
        return data.accessToken;
      })
      .catch(() => {
        clearClientSession();
        return null;
      })
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
};

const createBusinessError = (code?: number, message?: string) => {
  let kind: RequestErrorKind = 'business';
  if (code === 401) {
    kind = 'unauthorized';
  } else if (code === 403) {
    kind = 'forbidden';
  } else if ((code || 0) >= 500) {
    kind = 'server';
  }

  return new RequestError({
    kind,
    code,
    message,
    messageKey: message,
  });
};

const createTransportError = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status;
    let kind: RequestErrorKind = 'unknown';

    if (error.code === 'ECONNABORTED' || error.message?.toLowerCase().includes('timeout')) {
      kind = 'timeout';
    } else if (!error.response) {
      kind = 'network';
    } else if (status === 401) {
      kind = 'unauthorized';
    } else if (status === 403) {
      kind = 'forbidden';
    } else if ((status || 0) >= 500) {
      kind = 'server';
    } else {
      kind = 'business';
    }

    const message = error.response?.data?.message || error.message || 'request.failed';
    return new RequestError({
      kind,
      status,
      message,
      messageKey: error.response?.data?.message,
    });
  }

  if (error instanceof RequestError) {
    return error;
  }

  return new RequestError({
    kind: 'unknown',
    message: error instanceof Error ? error.message : 'request.failed',
  });
};

const isLikelyI18nKey = (message?: string) => Boolean(message && I18N_KEY_PATTERN.test(message));

const resolveFallbackMessageKey = (
  error: Pick<RequestError, 'kind'>,
  explicitFallbackKey?: string,
) => {
  if (explicitFallbackKey) {
    return explicitFallbackKey;
  }
  switch (error.kind) {
    case 'timeout':
      return 'network.timeout';
    case 'network':
      return 'network.error';
    case 'server':
      return 'request.failed';
    default:
      return 'request.failed';
  }
};

const translateMessage = (message: string | undefined, fallbackKey = 'request.failed') => {
  const fallbackText = i18n.t(fallbackKey, { defaultValue: fallbackKey });
  if (!message) {
    return fallbackText;
  }
  if (!isLikelyI18nKey(message)) {
    if (import.meta.env.DEV) {
      return `${fallbackText} (${message})`;
    }
    return fallbackText;
  }

  const translated = i18n.t(message, { defaultValue: message });
  if (translated !== message) {
    return translated;
  }

  if (import.meta.env.DEV) {
    return `${fallbackText} (${message})`;
  }
  return fallbackText;
};

const shouldSuppressAuthMessage = (messageKey: string | undefined, kind: RequestErrorKind) => {
  const isAuthError =
    kind === 'unauthorized' ||
    messageKey === 'session.invalid' ||
    messageKey === 'session.idle_timeout' ||
    Boolean(messageKey && messageKey.startsWith('token.'));
  if (!isAuthError) {
    return false;
  }
  return logoutTransition || globalThis.location.pathname === '/login';
};

export const isRequestError = (error: unknown): error is RequestError =>
  error instanceof RequestError;
export const isTimeoutRequestError = (error: unknown) =>
  isRequestError(error) && error.kind === 'timeout';
export const isNetworkRequestError = (error: unknown) =>
  isRequestError(error) && (error.kind === 'network' || error.kind === 'timeout');
export const isForbiddenRequestError = (error: unknown) =>
  isRequestError(error) && error.kind === 'forbidden';
export const isServerRequestError = (error: unknown) =>
  isRequestError(error) && error.kind === 'server';

const OPERATION_TOKEN_KEY = 'pantheon_op_token';
const OPERATION_TOKEN_REFRESH_BUFFER_MS = 30 * 1000;

const saveOperationToken = (token: string) => sessionStorage.setItem(OPERATION_TOKEN_KEY, token);
const readOperationToken = () => sessionStorage.getItem(OPERATION_TOKEN_KEY);
const clearOperationToken = () => sessionStorage.removeItem(OPERATION_TOKEN_KEY);

const decodeJwtPayload = (token: string): Record<string, unknown> | null => {
  const segments = token.split('.');
  if (segments.length < 2) {
    return null;
  }
  try {
    const base64 = segments[1].replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
    const json = globalThis.atob(padded);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
};

const isOperationTokenFresh = (token: string | null | undefined) => {
  if (!token) {
    return false;
  }
  const payload = decodeJwtPayload(token);
  const exp = typeof payload?.exp === 'number' ? payload.exp : 0;
  if (!exp) {
    return false;
  }
  return exp * 1000 > Date.now() + OPERATION_TOKEN_REFRESH_BUFFER_MS;
};

export const ensureOperationVerified = async (force = false) => {
  const currentToken = readOperationToken();
  if (!force && isOperationTokenFresh(currentToken)) {
    return currentToken as string;
  }
  clearOperationToken();
  const opToken = await showSecondaryVerify();
  saveOperationToken(opToken);
  return opToken;
};

const shouldRetryOperationVerify = (config?: RequestConfig, message?: string) => {
  if (!config || config._opRetry) {
    return false;
  }
  return (
    message === 'auth.operation.verification_required' ||
    message === 'auth.operation.verification_expired' ||
    message === 'auth.operation.verification_mismatch'
  );
};

const retryWithOperationVerify = async (config: RequestConfig) => {
  clearOperationToken();
  const opToken = await showSecondaryVerify();
  saveOperationToken(opToken);
  config._opRetry = true;
  config.headers = config.headers || {};
  config.headers['X-Operation-Token'] = opToken;
  return request(config);
};

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const csrfToken = readCsrfToken();
    if (
      csrfToken &&
      config.method &&
      config.method !== 'get' &&
      config.method !== 'head' &&
      config.method !== 'options'
    ) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }
    const opToken = readOperationToken();
    if (opToken) {
      config.headers['X-Operation-Token'] = opToken;
    }
    config.headers['Accept-Language'] = localStorage.getItem('pantheon_lang') || 'zh-CN';
    return config;
  },
  (error) => Promise.reject(error),
);

request.interceptors.response.use(
  async (response) => {
    const config = response.config as RequestConfig;
    const { code, data, message } = response.data;
    if (code === 200) {
      return data;
    }

    if (code === 401 && message === 'session.idle_timeout') {
      persistLoginNotice(message);
      redirectToLogin();
    }

    // 处理二次验证请求
    if (code === 403 && shouldRetryOperationVerify(config, message)) {
      try {
        return await retryWithOperationVerify(config);
      } catch (err) {
        return Promise.reject(err);
      }
    }

    if (shouldRefresh(config, code)) {
      const nextToken = await doRefreshToken();
      if (nextToken) {
        config._retry = true;
        config.headers = config.headers || {};
        config.headers.Authorization = `Bearer ${nextToken}`;
        return request(config);
      }
      redirectToLogin();
    }

    const requestError = createBusinessError(code, message || 'request.failed');
    if (
      !config.skipErrorMessage &&
      !shouldSuppressAuthMessage(requestError.messageKey || requestError.message, requestError.kind)
    ) {
      feedbackMessage.error(
        translateMessage(
          requestError.messageKey || requestError.message,
          resolveFallbackMessageKey(requestError),
        ),
      );
    }
    return Promise.reject(requestError);
  },
  async (error) => {
    const config = error.config as RequestConfig | undefined;
    if (
      error.response?.status === 403 &&
      config &&
      shouldRetryOperationVerify(config, error.response?.data?.message)
    ) {
      try {
        return await retryWithOperationVerify(config);
      } catch (verifyError) {
        return Promise.reject(createTransportError(verifyError));
      }
    }

    if (error.response?.status === 401 && shouldRefresh(config, 401)) {
      const nextToken = await doRefreshToken();
      if (nextToken && config) {
        config._retry = true;
        config.headers = config.headers || {};
        config.headers.Authorization = `Bearer ${nextToken}`;
        return request(config);
      }
      redirectToLogin();
    }

    const requestError = createTransportError(error);
    if (
      !config?.skipErrorMessage &&
      !shouldSuppressAuthMessage(requestError.messageKey || requestError.message, requestError.kind)
    ) {
      feedbackMessage.error(
        translateMessage(
          requestError.messageKey || requestError.message,
          resolveFallbackMessageKey(requestError),
        ),
      );
    }
    return Promise.reject(requestError);
  },
);

export default request;

export const apiRequest = request as unknown as <T = unknown>(config: RequestConfig) => Promise<T>;
