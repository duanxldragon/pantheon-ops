import { randomUUID } from 'node:crypto';
import fs from 'node:fs/promises';
import path from 'node:path';
import { expect, test as base, type APIResponse, type Page, type TestInfo } from '@playwright/test';
import {
  adminCredentials,
  apiBaseUrl,
  apiRequestHeaders,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T; message?: string };
type CatalogFixture = { token: string; hostId: number };

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function assertNoRawKeys(page: Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\b(?:business|operations)\.service\.[A-Za-z0-9_.-]+\b/);
}

async function screenshot(page: Page, testInfo: TestInfo, name: string) {
  const directory = path.resolve(process.cwd(), '../.harness/evidence/2026-11-13-ops-business-menu-ia-refactor/visual');
  await fs.mkdir(directory, { recursive: true });
  const file = path.join(directory, `${name}.png`);
  await page.screenshot({ path: file, fullPage: true });
  await testInfo.attach(name, { path: file, contentType: 'image/png' });
}

const test = base.extend<{ catalogFixture: CatalogFixture }>({
  catalogFixture: async ({ request }, provide) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `service-ia-${randomUUID().slice(0, 12)}`;
    let scopeId = 0;
    let hostId = 0;
    let applicationId = 0;
    let serviceId = 0;
    let instanceId = 0;
    try {
      scopeId = (await expectSuccess<{ id: number }>(await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: { code: token, name: token, environment: 'prod', status: 'active' },
      }))).id;
      hostId = (await expectSuccess<{ id: number }>(await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
        headers,
        data: {
          hostname: `${token}-host`,
          ip: `10.251.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
          sshPort: 22, os: 'linux', osVersion: 'Ubuntu 24.04', cpuCores: 2,
          memoryGb: 4, diskGb: 40, labels: [{ key: 'service', val: token }],
          owner: 'service-smoke', remark: token,
        },
      }))).id;
      await expectSuccess(await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
        headers, data: { hostIds: [hostId] },
      }));
      applicationId = (await expectSuccess<{ id: number }>(await request.post(`${apiBaseUrl}/business/service/applications`, {
        headers, data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId, owner: 'service-smoke' },
      }))).id;
      serviceId = (await expectSuccess<{ id: number }>(await request.post(`${apiBaseUrl}/business/service/services`, {
        headers, data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
      }))).id;
      instanceId = Number((await expectSuccess<{ id: number }>(await request.post(`${apiBaseUrl}/business/service/instances`, {
        headers, data: { serviceId, environment: 'prod', targetType: 'vm', hostId, desiredVersion: '1.2.0' },
      }))).id);
      expect(instanceId).toBeGreaterThan(0);
      await provide({ token, hostId });
    } finally {
      const failures: string[] = [];
      const cleanup = async (label: string, url: string) => {
        try {
          const response = await request.delete(url, { headers });
          const payload = await response.json() as ApiEnvelope<unknown>;
          if (!response.ok() || payload.code !== 200) failures.push(`${label}: HTTP ${response.status()} / code ${payload.code}`);
        } catch (error) {
          failures.push(`${label}: ${error instanceof Error ? error.message : String(error)}`);
        }
      };
      if (instanceId) await cleanup('instance', `${apiBaseUrl}/business/service/instances/${instanceId}`);
      if (serviceId) await cleanup('service', `${apiBaseUrl}/business/service/services/${serviceId}`);
      if (applicationId) await cleanup('application', `${apiBaseUrl}/business/service/applications/${applicationId}`);
      if (scopeId && hostId) await cleanup('unbind host', `${apiBaseUrl}/business/bizscope/${scopeId}/hosts/${hostId}`);
      if (hostId) await cleanup('host', `${apiBaseUrl}/business/cmdb/hosts/${hostId}`);
      if (scopeId) await cleanup('scope', `${apiBaseUrl}/business/bizscope/${scopeId}`);
      expect(failures, 'Every fixture created by this test is cleaned by its captured ID').toEqual([]);
    }
  },
});

const destinations = [
  { label: '应用', path: '/business/service-catalog/applications', slug: 'service-applications', item: 'application' },
  { label: '服务', path: '/business/service-catalog/services', slug: 'service-services', item: 'service' },
  { label: '服务实例', path: '/business/service-catalog/instances', slug: 'service-instances', item: 'service' },
] as const;
const browserErrors = new WeakMap<Page, string[]>();

test.describe('Service catalog navigation and independent pages', () => {
  test.setTimeout(90_000);
  test.beforeEach(async ({ page }) => {
    const errors: string[] = [];
    browserErrors.set(page, errors);
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
    page.on('pageerror', error => { errors.push(error.message); });
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });
  test.afterEach(async ({ page }) => {
    expect(browserErrors.get(page) ?? [], 'No browser console errors or uncaught page errors').toEqual([]);
  });

  for (const destination of destinations) {
    test(`${destination.label} page shows its owned catalog resource`, async ({ page, catalogFixture }, testInfo) => {
      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto(destination.path, { waitUntil: 'domcontentloaded' });
      await expect(page).toHaveURL(new RegExp(`${destination.path}$`));
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
      const table = page.locator('.system-list__table-card');
      await expect(table).toContainText(`${catalogFixture.token}-${destination.item}`);
      if (destination.slug === 'service-instances') await expect(table).toContainText(new RegExp(`(?:主机|Host) #${catalogFixture.hostId}`));
      await assertNoRawKeys(page);
      await screenshot(page, testInfo, `${destination.slug}-desktop`);
    });

    test(`${destination.label} page keeps its controls inside a phone viewport`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width: 390, height: 844 });
      await page.goto(destination.path, { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.governance-summary-bar')).toBeVisible();
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
      await expect.poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1)).toBe(true);
      await assertNoRawKeys(page);
      await screenshot(page, testInfo, `${destination.slug}-mobile`);
    });
  }

  test('service catalog sidebar exposes three real child pages', async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const missingPages: string[] = [];
    page.on('response', response => {
      if (response.status() === 404 && response.url().includes('service')) missingPages.push(response.url());
    });
    await page.goto('/business/service-catalog/applications', { waitUntil: 'domcontentloaded' });
    const parent = page.locator('.arco-menu-inline-header', { hasText: '服务目录' }).first();
    await expect(parent).toBeVisible();
    for (const destination of destinations) {
      const child = page.locator('.arco-menu-item').filter({ has: page.getByText(destination.label, { exact: true }) }).first();
      if (!(await child.isVisible())) await parent.click();
      await expect(child).toBeVisible();
      await child.click();
      await expect(page).toHaveURL(new RegExp(`${destination.path}$`));
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
    }
    expect(missingPages).toEqual([]);
  });
});
