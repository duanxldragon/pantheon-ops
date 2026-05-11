import { expect, type APIRequestContext, type Page } from '@playwright/test';

export const apiBaseUrl = process.env.PANTHEON_API_BASE_URL ?? 'http://127.0.0.1:8080/api/v1';

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
    localStorage.setItem('pantheon_lang', 'zh-CN');
    localStorage.setItem('pantheon_lang_explicit', '1');
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

function extractCookieValue(setCookieHeader: string | undefined, name: string) {
  if (!setCookieHeader) {
    return null;
  }
  const match = setCookieHeader.match(new RegExp(`(?:^|,\\s*)${name}=([^;]+)`));
  return match?.[1] ?? null;
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
  const csrfToken =
    extractCookieValue(response.headers()['set-cookie'], 'pantheon_csrf_token') ??
    `pantheon-smoke-csrf-${Date.now()}`;
  return {
    accessToken: payload.data.accessToken as string,
    refreshToken: payload.data.refreshToken as string,
    username: credentials.username,
    password: credentials.password,
    csrfToken,
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
  await expect(page.locator('.app-shell__header')).toBeVisible();
  return payload.data.accessToken as string;
}

export async function signInAsAdmin(page: Page) {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login);
  return login.accessToken;
}

export async function installClientSession(page: Page, login: BrowserLoginResult) {
  await primeChineseLocale(page);
  await page.context().addCookies([
    {
      name: 'pantheon_access_token',
      value: login.accessToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_refresh_token',
      value: login.refreshToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_csrf_token',
      value: login.csrfToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: false,
      secure: false,
      sameSite: 'Strict',
    },
  ]);
}

export async function getCsrfToken(page: Page) {
  const cookies = await page.context().cookies();
  const csrfCookie = cookies.find((cookie) => cookie.name === 'pantheon_csrf_token');
  expect(csrfCookie?.value).toBeTruthy();
  return csrfCookie!.value;
}

export async function requestHeaders(page: Page, accessToken: string) {
  return {
    ...authHeaders(accessToken),
    'X-CSRF-Token': await getCsrfToken(page),
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
    sessionStorage.setItem('pantheon_op_token', value);
  }, token);
  try {
    await page.evaluate((value) => {
      sessionStorage.setItem('pantheon_op_token', value);
    }, token);
  } catch {
    // The current document can still be about:blank in setup-heavy smoke tests.
    // The init script above installs the token on the next app navigation.
  }
}
