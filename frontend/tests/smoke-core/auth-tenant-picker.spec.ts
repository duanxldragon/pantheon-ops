import { expect, test, type Page } from '@playwright/test';

type LoginPayload = {
  username: string;
  password: string;
  tenantId?: number;
};

const candidates = [
  { tenantId: 101, code: 'acme', name: 'Acme North', role: 'owner' },
  { tenantId: 202, code: 'globex', name: 'Globex South', role: 'member' },
];

async function installTenantLoginRoutes(page: Page) {
  const loginBodies: LoginPayload[] = [];
  await page.route('**/api/v1/auth/login', async (route) => {
    const body = route.request().postDataJSON() as LoginPayload;
    loginBodies.push(body);
    if (body.password !== 'correct-password') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 401,
          message: 'auth.login.error.invalid_credentials',
          data: null,
        }),
      });
      return;
    }
    if (!body.tenantId) {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          data: {
            tenantSelectionRequired: true,
            tenantCandidates: candidates,
          },
        }),
      });
      return;
    }
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
          tenantId: body.tenantId,
          user: {
            id: 1,
            username: body.username,
            nickname: 'Tenant Operator',
            roles: ['admin'],
            perms: ['platform:dashboard:view'],
          },
        },
      }),
    });
  });

  await page.route('**/api/v1/system/setting/public', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: { settings: { 'site.name': 'Pantheon Base', 'i18n.default_language': 'zh-CN' } },
      }),
    });
  });

  await page.route('**/api/v1/system/menu/tree**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: [{ id: 1, parentId: 0, type: 'C', path: '/dashboard', isExternal: 0 }],
      }),
    });
  });

  await page.route('**/api/v1/dashboard/summary', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ code: 200, data: {} }),
    });
  });

  return loginBodies;
}

test.describe('Tenant login picker @priority:critical @smoke:core', () => {
  test('shows candidates only after password verification and submits the selected tenant', async ({
    page,
  }) => {
    const loginBodies = await installTenantLoginRoutes(page);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('correct-password');
    await page.locator('input[type="password"]').press('Enter');

    await expect(page.locator('.auth-login-tenant-picker')).toBeVisible();
    await page.locator('.auth-login-tenant-picker .arco-select').click();
    await expect(page.getByText('Acme North (acme)')).toBeVisible();
    await expect(page.getByText('Globex South (globex)')).toBeVisible();
    await page.screenshot({
      path: '../.harness/evidence/2026-09-10-tenant-verification-and-gray/artifacts/browser/tenant-picker-desktop.png',
      fullPage: true,
    });
    expect(loginBodies).toEqual([{ username: 'tenant-user', password: 'correct-password' }]);
    expect(await page.context().cookies()).toEqual([]);
    expect(await page.evaluate(() => localStorage.getItem('pantheon_session_hint'))).toBeNull();

    await page.getByText('Acme North (acme)').click();
    await page.getByRole('button', { name: /登录|sign in/i }).click();
    await expect(page).toHaveURL(/\/dashboard$/);
    expect(loginBodies.at(-1)).toEqual({
      username: 'tenant-user',
      password: 'correct-password',
      tenantId: 101,
    });
  });

  test('invalid password does not expose tenant candidates', async ({ page }) => {
    const loginBodies = await installTenantLoginRoutes(page);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('wrong-password');
    await page.locator('input[type="password"]').press('Enter');

    await expect(page.locator('.arco-message-error, .arco-notification-error')).toBeVisible();
    await expect(page.getByText('Acme North (acme)')).toHaveCount(0);
    await expect(page.getByText('Globex South (globex)')).toHaveCount(0);
    expect(loginBodies).toEqual([{ username: 'tenant-user', password: 'wrong-password' }]);
    expect(await page.context().cookies()).toEqual([]);
  });

  test('picker remains usable on narrow viewports without horizontal overflow', async ({
    page,
  }) => {
    const loginBodies = await installTenantLoginRoutes(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('correct-password');
    await page.locator('input[type="password"]').press('Enter');
    await expect(page.locator('.auth-login-tenant-picker')).toBeVisible();
    await page.locator('.auth-login-tenant-picker .arco-select').click();
    await expect(page.getByText('Acme North (acme)')).toBeVisible();

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow).toBeLessThanOrEqual(1);
    await page.screenshot({
      path: '../.harness/evidence/2026-09-10-tenant-verification-and-gray/artifacts/browser/tenant-picker-mobile.png',
      fullPage: true,
    });
    expect(loginBodies).toHaveLength(1);
  });
});
