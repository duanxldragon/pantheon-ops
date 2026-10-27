import { expect, test, type APIResponse } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T; message?: string };

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function expectNoVisibleServiceI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.service\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.service\.[A-Za-z0-9_.-]+\b/);
}

test.describe('Service business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('application, service, and VM instance complete the catalog lifecycle', async ({ page, request }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') consoleErrors.push(message.text());
    });
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `service-smoke-${Date.now()}`;
    let scopeId = 0;
    let hostId = 0;
    let applicationId = 0;
    let serviceId = 0;
    let instanceId = 0;

    try {
      const scope = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/bizscope`, {
          headers,
          data: { code: token, name: token, environment: 'prod', status: 'active' },
        }),
      );
      scopeId = scope.id;
      const host = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `${token}-host`,
            ip: `10.251.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Ubuntu 24.04',
            cpuCores: 2,
            memoryGb: 4,
            diskGb: 40,
            labels: [{ key: 'service', val: token }],
            owner: 'service-smoke',
            remark: token,
          },
        }),
      );
      hostId = host.id;
      await expectSuccess(
        await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
          headers,
          data: { hostIds: [hostId] },
        }),
      );
      const application = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/applications`, {
          headers,
          data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId, owner: 'service-smoke' },
        }),
      );
      applicationId = application.id;
      const service = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/services`, {
          headers,
          data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
        }),
      );
      serviceId = service.id;
      const instance = await expectSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/instances`, {
          headers,
          data: { serviceId, environment: 'prod', targetType: 'vm', hostId, desiredVersion: '1.2.0' },
        }),
      );
      instanceId = Number(instance.id);
      expect(instanceId).toBeGreaterThan(0);

      await page.goto('/business/service', { waitUntil: 'domcontentloaded' });
      await expect(page).toHaveURL(/\/business\/service$/);
      await expect(page.locator('.governance-summary-bar')).toContainText('服务目录');
      await expect(page.locator('.system-list__table-card:visible')).toContainText(`${token}-application`);

      await page.getByRole('tab', { name: '服务', exact: true }).click();
      const servicePanel = page.getByRole('tabpanel', { name: '服务', exact: true });
      await expect(servicePanel).toContainText(`${token}-service`);
      await page.getByRole('tab', { name: '服务实例', exact: true }).click();
      const instancePanel = page.getByRole('tabpanel', { name: '服务实例', exact: true });
      await expect(instancePanel).toContainText(`${token}-service`);
      await expect(instancePanel).toContainText(`Host #${hostId}`);
      await expectNoVisibleServiceI18nKeys(page);
      await page.screenshot({ path: testInfo.outputPath('service-catalog-desktop.png') });
      expect(consoleErrors).toEqual([]);
    } finally {
      const cleanupFailures: string[] = [];
      const cleanup = async (label: string, responsePromise: Promise<APIResponse>) => {
        try {
          const response = await responsePromise;
          const payload = await response.json().catch(() => null) as ApiEnvelope<unknown> | null;
          if (!response.ok() || payload?.code !== 200) {
            cleanupFailures.push(
              `${label}: HTTP ${response.status()} / code ${payload?.code ?? 'unknown'} / ${payload?.message ?? 'no message'}`,
            );
          }
        } catch (error) {
          cleanupFailures.push(`${label}: ${error instanceof Error ? error.message : String(error)}`);
        }
      };
      if (!instanceId && serviceId) {
        try {
          const response = await request.get(
            `${apiBaseUrl}/business/service/instances?serviceId=${serviceId}&page=1&pageSize=20`,
            { headers },
          );
          const payload = await response.json() as ApiEnvelope<{ items: Array<{ id: number }> }>;
          instanceId = Number(payload.data?.items?.[0]?.id ?? 0);
          if (!response.ok() || payload.code !== 200 || !instanceId) {
            cleanupFailures.push('resolve instance: unable to find the created service instance');
          }
        } catch (error) {
          cleanupFailures.push(`resolve instance: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
      if (instanceId) await cleanup('instance', request.delete(`${apiBaseUrl}/business/service/instances/${instanceId}`, { headers }));
      if (serviceId) await cleanup('service', request.delete(`${apiBaseUrl}/business/service/services/${serviceId}`, { headers }));
      if (applicationId) await cleanup('application', request.delete(`${apiBaseUrl}/business/service/applications/${applicationId}`, { headers }));
      if (scopeId && hostId) await cleanup('unbind host', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/${hostId}`, { headers }));
      if (hostId) await cleanup('host', request.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostId}`, { headers }));
      if (scopeId) await cleanup('scope', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}`, { headers }));
      expect(cleanupFailures).toEqual([]);
    }
  });

  test('service page remains usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/service', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.getByRole('tab', { name: '服务实例', exact: true })).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('service-list-mobile.png') });
  });
});
