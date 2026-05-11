import { expect, test, type Page } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  authHeaders,
  loginByApi,
  requestHeaders,
} from '../helpers/auth';

type UserPlatformPreferences = {
  theme?: string;
  language?: string;
  layoutMode?: string;
  densityMode?: string;
};

async function getCurrentUserPreferences(page: Page, accessToken: string) {
  const response = await page.request.get(`${apiBaseUrl}/auth/me`, {
    headers: authHeaders(accessToken),
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
  return (payload.data?.preferences || {}) as UserPlatformPreferences;
}

async function updateCurrentUserPreferences(
  page: Page,
  accessToken: string,
  preferences: UserPlatformPreferences,
) {
  const response = await page.request.put(`${apiBaseUrl}/auth/me/preferences`, {
    headers: await requestHeaders(page, accessToken),
    data: preferences,
  });
  expect(response.ok()).toBeTruthy();
  const payload = await response.json();
  expect(payload.code).toBe(200);
}

test('login page explicit language wins over saved user preference', async ({ page }) => {
  const initialLogin = await loginByApi(page.request, adminCredentials);
  await page.context().addCookies([
    {
      name: 'pantheon_access_token',
      value: initialLogin.accessToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_refresh_token',
      value: initialLogin.refreshToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: true,
      secure: false,
      sameSite: 'Strict',
    },
    {
      name: 'pantheon_csrf_token',
      value: initialLogin.csrfToken,
      url: 'http://127.0.0.1:5173',
      httpOnly: false,
      secure: false,
      sameSite: 'Strict',
    },
  ]);

  const originalPreferences = await getCurrentUserPreferences(page, initialLogin.accessToken);

  try {
    await updateCurrentUserPreferences(page, initialLogin.accessToken, {
      ...originalPreferences,
      language: 'zh-CN',
    });

    await page.context().clearCookies();
    await page.addInitScript(() => {
      localStorage.removeItem('pantheon_lang');
      localStorage.removeItem('pantheon_lang_explicit');
      sessionStorage.clear();
    });

    await page.goto('/login', { waitUntil: 'networkidle' });
    await page.locator('.auth-login-page__tools .arco-select-view').click();
    await page.getByRole('option', { name: 'English' }).click();
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(() => ({
          language: localStorage.getItem('pantheon_lang'),
          explicit: localStorage.getItem('pantheon_lang_explicit'),
        })),
      )
      .toEqual({ language: 'en-US', explicit: '1' });

    await page.getByPlaceholder(/username|用户名/i).fill(adminCredentials.username);
    await page.getByPlaceholder(/password|密码/i).fill(adminCredentials.password);
    await page.getByRole('button', { name: /sign in|登录/i }).click();

    await expect(page).toHaveURL(/\/dashboard$/);
    await expect(page.getByRole('heading', { name: 'Workbench' }).first()).toBeVisible();
  } finally {
    const restoreLogin = await loginByApi(page.request, adminCredentials);
    await page.context().addCookies([
      {
        name: 'pantheon_access_token',
        value: restoreLogin.accessToken,
        url: 'http://127.0.0.1:5173',
        httpOnly: true,
        secure: false,
        sameSite: 'Strict',
      },
      {
        name: 'pantheon_refresh_token',
        value: restoreLogin.refreshToken,
        url: 'http://127.0.0.1:5173',
        httpOnly: true,
        secure: false,
        sameSite: 'Strict',
      },
      {
        name: 'pantheon_csrf_token',
        value: restoreLogin.csrfToken,
        url: 'http://127.0.0.1:5173',
        httpOnly: false,
        secure: false,
        sameSite: 'Strict',
      },
    ]);
    await updateCurrentUserPreferences(page, restoreLogin.accessToken, originalPreferences);
  }
});
