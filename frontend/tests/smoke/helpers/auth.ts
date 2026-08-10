import { expect, type APIRequestContext, type Page } from '@playwright/test';
import { readAuthCookieSession } from '../../../scripts/lib/auth-cookie-session.mjs';
import { COOKIE_TOKEN_PLACEHOLDER } from '../../../src/core/auth/sessionSnapshot.ts';
import type { UserInfo } from '../../../src/modules/auth/security/types.ts';

export const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';
const defaultWebBaseUrl = process.env.PANTHEON_WEB_BASE_URL ?? 'http://127.0.0.1:5173';
const AUTH_USER_STORAGE_KEY = 'pantheon_auth_user';

export type BrowserLoginResult = {
  accessToken: string;
  refreshToken: string;
  username: string;
  password: string;
  csrfToken: string;
};

export type LoginCredentials = {
  username: string;
  password: string;
};

export const adminCredentials: LoginCredentials = {
  username: process.env.PANTHEON_SMOKE_ADMIN_USERNAME ?? 'admin',
  password: process.env.PANTHEON_SMOKE_ADMIN_PASSWORD ?? '123456',
};

export async function primeChineseLocale(page: Page) {
  await page.addInitScript(() => {
    try {
      globalThis.localStorage?.setItem('pantheon_lang', 'zh-CN');
      globalThis.localStorage?.setItem('pantheon_lang_explicit', '1');
    } catch {
      // about:blank and other opaque origins do not expose storage.
    }
  });
}

export function authHeaders(accessToken: string) {
  return {
    Authorization: `Bearer ${accessToken}`,
  };
}

function resolveRequestContext(requestLike: APIRequestContext | Page) {
  if ('post' in requestLike && typeof requestLike.post === 'function') {
    return requestLike as APIRequestContext;
  }
  return (requestLike as Page).request;
}

export async function loginByApi(
  requestLike: APIRequestContext | Page,
  credentials: LoginCredentials,
): Promise<BrowserLoginResult> {
  const request = resolveRequestContext(requestLike);
  const response = await request.post(`${apiBaseUrl}/auth/login`, {
    data: credentials,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  const session = readAuthCookieSession(response.headers(), {
    csrfFallback: `pantheon-smoke-csrf-${Date.now()}`,
  });
  return {
    accessToken: session.accessToken,
    refreshToken: session.refreshToken,
    username: credentials.username,
    password: credentials.password,
    csrfToken: session.csrfToken,
  };
}

export async function signInWithUi(
  page: Page,
  credentials: LoginCredentials,
) {
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'networkidle' });
  await page.getByPlaceholder(/请输入用户名|username/i).fill(credentials.username);
  await page.getByPlaceholder(/请输入密码|password/i).fill(credentials.password);
  const loginResponse = page.waitForResponse((response) =>
    response.url().includes('/api/v1/auth/login'),
  );
  await page.getByRole('button', { name: /登录|Sign in|Sign In/ }).click();
  const response = await loginResponse;
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  await page.goto('/dashboard', { waitUntil: 'networkidle' });
  await expect(page.locator('.app-shell__header')).toBeVisible();
  return COOKIE_TOKEN_PLACEHOLDER;
}

export async function signInAsAdmin(page: Page) {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  await installAuthUserSnapshot(page, await getCurrentUserInfo(page.request, login.accessToken));
  return login.accessToken;
}

async function getCurrentUserInfo(
  requestLike: APIRequestContext | Page,
  accessToken: string,
): Promise<UserInfo> {
  const request = resolveRequestContext(requestLike);
  const response = await request.get(`${apiBaseUrl}/auth/me`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data as UserInfo;
}

async function installAuthUserSnapshot(page: Page, userInfo: UserInfo) {
  const serializedUser = JSON.stringify({
    ...userInfo,
    preferences: undefined,
  } satisfies UserInfo);
  await page.addInitScript((value) => {
    try {
      globalThis.localStorage?.setItem('pantheon_auth_user', value);
    } catch {
      // about:blank and other opaque origins do not expose storage.
    }
  }, serializedUser);
  if (page.url() !== 'about:blank') {
    await page.evaluate(([storageKey, value]) => {
      try {
        globalThis.localStorage?.setItem(storageKey, value);
      } catch {
        // Ignore opaque-origin storage failures for smoke setup.
      }
    }, [AUTH_USER_STORAGE_KEY, serializedUser] as const);
  }
}

export async function installClientSession(page: Page, login: BrowserLoginResult) {
  await primeChineseLocale(page);
  await page.addInitScript((csrfToken) => {
    try {
      globalThis.localStorage?.setItem('pantheon_session_hint', '1');
      globalThis.localStorage?.setItem('pantheon_csrf_token', csrfToken);
    } catch {
      // about:blank and other opaque origins do not expose storage.
    }
  }, login.csrfToken);
  const appBaseUrl = new URL(
    page.url() === 'about:blank' ? '/' : page.url(),
    page.url() === 'about:blank' ? defaultWebBaseUrl : undefined,
  );
  const cookieUrl = appBaseUrl.origin;
  await page.context().addCookies([
    {
      name: 'pantheon_access_token',
      value: login.accessToken,
      url: cookieUrl,
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_refresh_token',
      value: login.refreshToken,
      url: cookieUrl,
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_csrf_token',
      value: login.csrfToken,
      url: cookieUrl,
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
  ]);
  if (page.url() !== 'about:blank') {
    await page.evaluate((csrfToken) => {
      try {
        globalThis.localStorage?.setItem('pantheon_session_hint', '1');
        globalThis.localStorage?.setItem('pantheon_csrf_token', csrfToken);
      } catch {
        // Ignore opaque-origin storage failures for smoke setup.
      }
    }, login.csrfToken);
  }
}

export async function getCsrfToken(page: Page) {
  const cookies = await page.context().cookies();
  const csrfCookie = cookies.find((cookie) => cookie.name === 'pantheon_csrf_token');
  expect(csrfCookie?.value).toBeTruthy();
  return csrfCookie!.value;
}

export async function requestHeaders(page: Page, accessToken: string) {
  const csrfToken = await getCsrfToken(page);
  return {
    ...authHeaders(accessToken),
    'X-CSRF-Token': csrfToken,
    Cookie: `pantheon_csrf_token=${csrfToken}`,
  };
}

export function apiRequestHeaders(login: BrowserLoginResult) {
  return {
    ...authHeaders(login.accessToken),
    'X-CSRF-Token': login.csrfToken,
    Cookie: `pantheon_csrf_token=${login.csrfToken}`,
  };
}

export async function getOperationToken(page: Page, accessToken: string) {
  const response = await page.request.post(`${apiBaseUrl}/auth/operation-verify`, {
    headers: await requestHeaders(page, accessToken),
    data: { password: adminCredentials.password },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.operationToken as string;
}

export async function getApiOperationToken(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  const request = resolveRequestContext(requestLike);
  const response = await request.post(`${apiBaseUrl}/auth/operation-verify`, {
    headers: apiRequestHeaders(login),
    data: { password: login.password },
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return payload.data.operationToken as string;
}

export async function verifiedHeaders(page: Page, accessToken: string) {
  return {
    ...(await requestHeaders(page, accessToken)),
    'X-Operation-Token': await getOperationToken(page, accessToken),
  };
}

export async function verifiedApiHeaders(requestLike: APIRequestContext | Page, login: BrowserLoginResult) {
  return {
    ...apiRequestHeaders(login),
    'X-Operation-Token': await getApiOperationToken(requestLike, login),
  };
}

export async function installOperationToken(page: Page, accessToken: string) {
  const token = await getOperationToken(page, accessToken);
  await page.addInitScript((value) => {
    try {
      globalThis.sessionStorage?.setItem('pantheon_op_token', value);
    } catch {
      // about:blank and other opaque origins do not expose storage.
    }
  }, token);
  if (page.url() !== 'about:blank') {
    await page.evaluate((value) => {
      try {
        globalThis.sessionStorage?.setItem('pantheon_op_token', value);
      } catch {
        // Ignore opaque-origin storage failures for smoke setup.
      }
    }, token);
  }
}
