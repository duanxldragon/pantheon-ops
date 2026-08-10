import axios from 'axios';
import type { AxiosRequestConfig, InternalAxiosRequestConfig } from 'axios';
import { message as feedbackMessage } from '../components/feedback/message';
import i18n from 'i18next';
import {
  resolveBusinessErrorKind,
  resolveFallbackMessageKey,
  resolveTransportErrorKind,
  shouldSuppressAuthMessage,
  type RequestErrorKind,
} from './requestErrorUtils';
import { useAuthStore } from '../store/useAuthStore';
import { clearShellSessionState, persistLoginNotice } from '../core/shellState';
import { clearExplicitLanguagePreference } from '../core/settings/languagePreference';
import { clearPantheonThemePreference } from '../core/theme/theme';
import { showSecondaryVerify } from '../components/feedback/secondaryVerifyController';
import { switchI18nLanguage } from '../i18n';
import {
  clearClientAuthSession,
  persistCsrfToken,
  readStoredCsrfToken,
} from '../core/auth/clientSession';
import { COOKIE_TOKEN_PLACEHOLDER } from '../core/auth/sessionSnapshot';

export interface RequestConfig extends AxiosRequestConfig {
  _retry?: boolean;
  _opRetry?: boolean;
  skipAuthRefresh?: boolean;
  skipErrorMessage?: boolean;
}

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

function readHeaderValue(headers: unknown, name: string): string {
  if (!headers || typeof headers !== 'object') {
    return '';
  }

  if ('get' in headers && typeof headers.get === 'function') {
    const value = headers.get(name);
    return typeof value === 'string' ? value : '';
  }

  const record = headers as Record<string, unknown>;
  const value = record[name] ?? record[name.toLowerCase()];
  if (Array.isArray(value)) {
    return typeof value[0] === 'string' ? value[0] : '';
  }
  return typeof value === 'string' ? value : '';
}

function syncCsrfTokenFromHeaders(headers: unknown) {
  const csrfToken = readHeaderValue(headers, 'X-CSRF-Token').trim();
  if (!csrfToken) {
    return;
  }
  persistCsrfToken(csrfToken);
}

let refreshPromise: Promise<boolean> | null = null;
let logoutTransition = false;

const clearClientSession = () => {
  request.post('/auth/logout', {}).catch(() => undefined);
  clearShellSessionState();
  clearPantheonThemePreference();
  const nextLanguage = clearExplicitLanguagePreference();
  switchI18nLanguage(nextLanguage).catch(() => undefined);
  clearClientAuthSession();
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
  if (config.url?.includes('/auth/login') || config.url?.includes('/auth/refresh')) {
    return false;
  }
  return code === 401;
};

const doRefreshToken = async (): Promise<boolean> => {
  refreshPromise ??= axios
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
      syncCsrfTokenFromHeaders(response.headers);
      const { code, data } = response.data;
      if (code !== 200) {
        throw new Error(response.data.message || 'request.failed');
      }
      useAuthStore.getState().setTokens(COOKIE_TOKEN_PLACEHOLDER, COOKIE_TOKEN_PLACEHOLDER);
      return Boolean(data);
    })
    .catch(() => {
      clearClientSession();
      return false;
    })
    .finally(() => {
      refreshPromise = null;
    });

  return refreshPromise;
};

const createBusinessError = (code?: number, message?: string) => {
  return new RequestError({
    kind: resolveBusinessErrorKind(code),
    code,
    message,
    messageKey: message,
  });
};

const createTransportError = (error: unknown) => {
  if (axios.isAxiosError(error)) {
    const status = error.response?.status;
    const message = error.response?.data?.message || error.message || 'request.failed';
    return new RequestError({
      kind: resolveTransportErrorKind(error),
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
const OPERATION_TOKEN_FALLBACK_TTL_MS = 5 * 60 * 1000;

type StoredOperationToken = {
  token: string;
  acquiredAt?: number;
  expiresAt?: number;
};

const saveOperationToken = (
  token: string,
  options?: {
    acquiredAt?: number;
    expiresAt?: number;
  },
) =>
  sessionStorage.setItem(
    OPERATION_TOKEN_KEY,
    JSON.stringify({
      token,
      acquiredAt: options?.acquiredAt ?? Date.now(),
      expiresAt: options?.expiresAt,
    } satisfies StoredOperationToken),
  );

const readStoredOperationToken = (): StoredOperationToken | null => {
  const rawValue = sessionStorage.getItem(OPERATION_TOKEN_KEY);
  if (!rawValue) {
    return null;
  }
  try {
    const parsed = JSON.parse(rawValue) as Partial<StoredOperationToken>;
    if (typeof parsed.token === 'string' && parsed.token.trim()) {
      return {
        token: parsed.token,
        acquiredAt: typeof parsed.acquiredAt === 'number' ? parsed.acquiredAt : undefined,
        expiresAt: typeof parsed.expiresAt === 'number' ? parsed.expiresAt : undefined,
      };
    }
  } catch {
    // Fall through to legacy raw-token handling.
  }

  const legacyToken = rawValue.trim();
  if (!legacyToken) {
    return null;
  }
  const migrated = {
    token: legacyToken,
    acquiredAt: Date.now(),
  } satisfies StoredOperationToken;
  saveOperationToken(migrated.token, { acquiredAt: migrated.acquiredAt });
  return migrated;
};

const readOperationToken = () => readStoredOperationToken()?.token ?? null;
const clearOperationToken = () => sessionStorage.removeItem(OPERATION_TOKEN_KEY);

const decodeJwtPayload = (token: string): Record<string, unknown> | null => {
  const segments = token.split('.');
  if (segments.length < 2) {
    return null;
  }
  try {
    const base64 = segments[1].replaceAll('-', '+').replaceAll('_', '/');
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
    const json = globalThis.atob(padded);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
};

const resolveOperationTokenExpiresAt = (storedToken: StoredOperationToken) => {
  const payload = decodeJwtPayload(storedToken.token);
  const exp = typeof payload?.exp === 'number' ? payload.exp : 0;
  if (exp) {
    return exp * 1000;
  }
  if (typeof storedToken.expiresAt === 'number' && storedToken.expiresAt > 0) {
    return storedToken.expiresAt;
  }
  if (typeof storedToken.acquiredAt === 'number' && storedToken.acquiredAt > 0) {
    return storedToken.acquiredAt + OPERATION_TOKEN_FALLBACK_TTL_MS;
  }
  return 0;
};

const isOperationTokenFresh = (storedToken: StoredOperationToken | null | undefined) => {
  if (!storedToken?.token) {
    return false;
  }
  const expiresAt = resolveOperationTokenExpiresAt(storedToken);
  if (!expiresAt) {
    return false;
  }
  return expiresAt > Date.now() + OPERATION_TOKEN_REFRESH_BUFFER_MS;
};

export const ensureOperationVerified = async (force = false) => {
  const currentToken = readStoredOperationToken();
  if (!force && isOperationTokenFresh(currentToken)) {
    return currentToken!.token;
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
    const csrfToken = readStoredCsrfToken();
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
    syncCsrfTokenFromHeaders(response.headers);
    const config = response.config as RequestConfig;
    const { code, data, message } = response.data;
    if (code === 200) {
      if (config.url?.includes('/auth/logout')) {
        clearClientAuthSession();
      }
      return data;
    }

    if (code === 401 && message === 'session.idle_timeout') {
      persistLoginNotice(message);
      redirectToLogin();
    }

    // 处理二次验证请求
    if (code === 403 && shouldRetryOperationVerify(config, message)) {
      return await retryWithOperationVerify(config);
    }

    if (shouldRefresh(config, code)) {
      const refreshed = await doRefreshToken();
      if (refreshed) {
        config._retry = true;
        return request(config);
      }
      redirectToLogin();
    }

    const requestError = createBusinessError(code, message || 'request.failed');
    if (
      !config.skipErrorMessage &&
      !shouldSuppressAuthMessage(
        requestError.messageKey || requestError.message,
        requestError.kind,
        globalThis.location.pathname,
        logoutTransition,
      )
    ) {
      feedbackMessage.error(
        translateMessage(
          requestError.messageKey || requestError.message,
          resolveFallbackMessageKey(requestError.kind),
        ),
      );
    }
    throw requestError;
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
        throw createTransportError(verifyError);
      }
    }

    if (error.response?.status === 401 && shouldRefresh(config, 401)) {
      const refreshed = await doRefreshToken();
      if (refreshed && config) {
        config._retry = true;
        return request(config);
      }
      redirectToLogin();
    }

    const requestError = createTransportError(error);
    if (
      !config?.skipErrorMessage &&
      !shouldSuppressAuthMessage(
        requestError.messageKey || requestError.message,
        requestError.kind,
        globalThis.location.pathname,
        logoutTransition,
      )
    ) {
      feedbackMessage.error(
        translateMessage(
          requestError.messageKey || requestError.message,
          resolveFallbackMessageKey(requestError.kind),
        ),
      );
    }
    throw requestError;
  },
);

export default request;

export const apiRequest = request as unknown as <T = unknown>(config: RequestConfig) => Promise<T>;
