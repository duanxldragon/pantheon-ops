import { expect, test, type APIResponse } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  loginByApi,
  signInWithUi,
} from '../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T; message?: string };

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  const payload = (await response.json()) as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

test.describe.serial('Observability menu click verification', () => {
  test('可观测中心二级菜单均落到真实页面', async ({ page }) => {
    const badNav: string[] = [];
    page.on('response', (resp) => {
      if (resp.status() === 404 && resp.request().url().includes('observability')) badNav.push(resp.url());
    });

    // Create a scope + monitoring target via API so the metric-source table
    // renders with data after menu navigation.
    const login = await loginByApi(page.request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `menu-obs-${Date.now()}`;
    const scope = await expectSuccess<{ id: number }>(
      await page.request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: { code: token, name: token, environment: 'dev', status: 'active' },
      }),
    );
    const target = await expectSuccess<{ id: number }>(
      await page.request.post(`${apiBaseUrl}/business/observability/targets`, {
        headers,
        data: {
          businessScopeId: scope.id,
          targetType: 'host',
          targetName: `${token}-target`,
          endpoint: 'http://127.0.0.1:1',
          metricsPath: '/metrics',
        },
      }),
    );

    try {
      await signInWithUi(page, adminCredentials);

      const obsMenu = page.locator('.arco-menu-item, .arco-menu-inline-header', { hasText: '可观测中心' }).first();
      await expect(obsMenu).toBeVisible({ timeout: 15000 });
      await obsMenu.click();
      const destinations = [
        { label: '监控中心', path: '/business/observability/monitoring' },
        { label: '告警中心', path: '/business/observability/alerts' },
        { label: '日志中心', path: '/business/observability/logs' },
        { label: '链路追踪', path: '/business/observability/traces' },
      ];
      for (const destination of destinations) {
        const item = page.locator('.arco-menu-item', { hasText: destination.label }).first();
        if (!(await item.isVisible())) {
          await obsMenu.click();
        }
        await expect(item).toBeVisible({ timeout: 10000 });
        await item.click();
        await expect(page).toHaveURL(new RegExp(`${destination.path}$`));
        await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
      }

      await page.goto('/business/observability/monitoring', { waitUntil: 'domcontentloaded' });
      const card = page.locator('.system-list__table-card').first();
      await expect(card).toBeVisible({ timeout: 10000 });
      await expect(card).toContainText(`${token}-target`);
      expect(badNav).toEqual([]);
    } finally {
      await page.request.delete(`${apiBaseUrl}/business/observability/targets/${target.id}`, { headers }).catch(() => undefined);
      await page.request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
    }
  });
});
