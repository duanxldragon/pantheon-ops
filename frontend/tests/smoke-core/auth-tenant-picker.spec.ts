import { expect, test, type Page } from '@playwright/test';

type LoginPayload = {
  username: string;
  password: string;
  tenantId?: number;
};

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
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 200,
        data: {
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

test.describe('Tenant login (picker removed, W4) @priority:critical @smoke:core', () => {
  test('signs in directly without sending a tenant and never renders the picker', async ({
    page,
  }) => {
    const loginBodies = await installTenantLoginRoutes(page);
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('correct-password');
    await page.locator('input[type="password"]').press('Enter');

    await expect(page).toHaveURL(/\/dashboard$/);
    // The client no longer names a tenant: discovery resolves the owning one
    // server-side (contract §2.2 V1.7).
    expect(loginBodies).toEqual([{ username: 'tenant-user', password: 'correct-password' }]);
    await expect(page.locator('.auth-login-tenant-picker')).toHaveCount(0);
  });

  test('a picker envelope from stale server state is refused, not rendered', async ({ page }) => {
    const loginBodies = await installTenantLoginRoutes(page);
    await page.route('**/api/v1/auth/login', async (route) => {
      const body = route.request().postDataJSON() as LoginPayload;
      loginBodies.push(body);
      if (body.password !== 'correct-password') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ code: 401, message: 'auth.login.error.invalid_credentials', data: null }),
        });
        return;
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          code: 200,
          data: {
            tenantSelectionRequired: true,
            tenantCandidates: [
              { tenantId: 101, code: 'acme', name: 'Acme North', role: 'owner' },
              { tenantId: 202, code: 'globex', name: 'Globex South', role: 'member' },
            ],
          },
        }),
      });
    });

    await page.goto('/login', { waitUntil: 'domcontentloaded' });
    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('correct-password');
    await page.locator('input[type="password"]').press('Enter');

    // Refusal surfaces the generic login failure; the UI must not grow a
    // picker retroactively and must not navigate into a half-signed-in shell.
    await expect(page.locator('.arco-message-error, .arco-notification-error')).toBeVisible();
    await expect(page.locator('.auth-login-tenant-picker')).toHaveCount(0);
    await expect(page).toHaveURL(/\/login$/);
    expect(loginBodies).toHaveLength(1);
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

  test('login form renders on narrow viewports without horizontal overflow', async ({ page }) => {
    await installTenantLoginRoutes(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/login', { waitUntil: 'domcontentloaded' });

    await page.getByPlaceholder(/请输入用户名|username/i).fill('tenant-user');
    await page.getByPlaceholder(/请输入密码|password/i).fill('correct-password');
    await page.locator('input[type="password"]').press('Enter');
    await expect(page).toHaveURL(/\/dashboard$/);

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
    expect(overflow).toBeLessThanOrEqual(1);
  });
});
