import { randomUUID } from 'node:crypto';
import fs from 'node:fs/promises';
import path from 'node:path';
import {
  expect, test as base, type APIResponse, type Page, type Response, type TestInfo,
} from '@playwright/test';
import {
  adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession,
  loginByApi, primeChineseLocale,
} from '../../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T };
type Row = { id: number; code: string; name: string };
type OwnedFixtures = {
  token: string;
  scope: (suffix?: string) => Promise<Row>;
  environment: (scopeId: number) => Promise<Row>;
  host: (scopeId: number, suffix: string) => Promise<Row & { hostname: string }>;
  track: (suffix: string) => void;
  forget: (suffix: string) => void;
};

async function success<T>(response: APIResponse | Response): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = await response.json() as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

const test = base.extend<{ owned: OwnedFixtures }>({
  owned: async ({ request }, provide) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `bizscope-ia-${randomUUID().slice(0, 12)}`;
    const cleanup = new Set<string>();
    const track = (suffix: string) => { cleanup.add(suffix); };
    try {
      await provide({
        token,
        track,
        forget: suffix => { cleanup.delete(suffix); },
        scope: async (suffix = 'scope') => {
          const row = await success<Row>(await request.post(`${apiBaseUrl}/business/bizscope`, {
            headers, data: { code: `${token}-${suffix}`, name: `${token}-${suffix}`, environment: 'dev', status: 'active' },
          }));
          track(`/business/bizscope/${row.id}`);
          return row;
        },
        environment: async scopeId => {
          const row = await success<Row>(await request.post(`${apiBaseUrl}/business/bizscope/environments`, {
            headers,
            data: { businessScopeId: scopeId, code: `${token}-env`, name: `${token}-environment`, envType: 'dev', status: 'active' },
          }));
          track(`/business/bizscope/environments/${row.id}`);
          return row;
        },
        host: async (scopeId, suffix) => {
          const row = await success<Row & { hostname: string }>(await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
            headers,
            data: {
              hostname: `${token}-${suffix}`, ip: `10.250.${Math.floor(Math.random() * 200) + 20}.${Math.floor(Math.random() * 200) + 20}`,
              sshPort: 22, os: 'linux', cpuCores: 2, memoryGb: 4, diskGb: 40,
              labels: [{ key: 'smoke', val: token }], owner: 'bizscope-smoke', remark: token,
            },
          }));
          track(`/business/cmdb/hosts/${row.id}`);
          await success(await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
            headers, data: { hostIds: [row.id] },
          }));
          track(`/business/bizscope/${scopeId}/hosts/${row.id}`);
          return row;
        },
      });
    } finally {
      const failures: string[] = [];
      // Reverse creation order unbinds hosts and deletes child environments
      // before deleting their owning scope. No token-wide cleanup is permitted.
      for (const suffix of [...cleanup].reverse()) {
        try {
          const response = await request.delete(`${apiBaseUrl}${suffix}`, { headers });
          const payload = await response.json() as ApiEnvelope<unknown>;
          if (!response.ok() || payload.code !== 200) failures.push(`${suffix}: HTTP ${response.status()} / code ${payload.code}`);
        } catch (error) {
          failures.push(`${suffix}: ${error instanceof Error ? error.message : String(error)}`);
        }
      }
      expect(failures, 'Only captured fixture IDs are cleaned, in child-before-parent order').toEqual([]);
    }
  },
});

async function assertNoRawKeys(page: Page) {
  expect(await page.locator('body').innerText()).not.toMatch(/\b(?:business|operations)\.bizscope\.[A-Za-z0-9_.-]+\b/);
}

async function screenshot(page: Page, testInfo: TestInfo, name: string) {
  const directory = path.resolve(process.cwd(), '../.harness/evidence/2026-11-13-ops-business-menu-ia-refactor/visual');
  await fs.mkdir(directory, { recursive: true });
  const file = path.join(directory, `${name}.png`);
  await page.screenshot({ path: file, fullPage: true });
  await testInfo.attach(name, { path: file, contentType: 'image/png' });
}

async function mutation(page: Page, method: string, suffix: string, action: () => Promise<void>) {
  const pathname = new URL(`${apiBaseUrl}${suffix}`).pathname;
  const [response] = await Promise.all([
    page.waitForResponse(candidate =>
      candidate.request().method() === method && new URL(candidate.url()).pathname === pathname,
    ),
    action(),
  ]);
  return response;
}

async function confirmDelete(page: Page) {
  await page.locator('.arco-popconfirm:visible').getByRole('button', { name: /确定|确认|OK/, exact: true }).click();
}

async function showScope(page: Page, code: string) {
  await page.goto('/business/bizscope/scopes', { waitUntil: 'domcontentloaded' });
  const filtered = page.waitForResponse(response => {
    const url = new URL(response.url());
    return response.request().method() === 'GET'
      && url.pathname.endsWith('/business/bizscope/list') && url.searchParams.get('code') === code;
  });
  await page.locator('.search-toolbar input').first().fill(code);
  await success(await filtered);
  const row = page.getByRole('row').filter({ hasText: code }).first();
  await expect(row).toBeVisible();
  return row;
}

async function showEnvironment(page: Page, code: string) {
  await page.goto('/business/bizscope/environments', { waitUntil: 'domcontentloaded' });
  const filtered = page.waitForResponse(response => {
    const url = new URL(response.url());
    return response.request().method() === 'GET'
      && url.pathname.endsWith('/business/bizscope/environments') && url.searchParams.get('keyword') === code;
  });
  await page.locator('.search-toolbar input').first().fill(code);
  await success(await filtered);
  const row = page.getByRole('row').filter({ hasText: code }).first();
  await expect(row).toBeVisible();
  return row;
}

const destinations = [
  { label: '业务域列表', path: '/business/bizscope/scopes', slug: 'bizscope-scopes' },
  { label: '主机绑定', path: '/business/bizscope/host-bindings', slug: 'bizscope-host-bindings' },
  { label: '环境管理', path: '/business/bizscope/environments', slug: 'bizscope-environments' },
] as const;
const browserErrors = new WeakMap<Page, string[]>();

test.describe('Business scope menu and ownership pages', () => {
  test.setTimeout(90_000);
  test.beforeEach(async ({ page }) => {
    const errors: string[] = [];
    browserErrors.set(page, errors);
    page.on('console', message => { if (message.type() === 'error') errors.push(message.text()); });
    page.on('pageerror', error => { errors.push(error.message); });
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
    await page.setViewportSize({ width: 1440, height: 900 });
  });
  test.afterEach(async ({ page }) => {
    expect(browserErrors.get(page) ?? [], 'No browser console errors or uncaught page errors').toEqual([]);
  });

  test('business scope sidebar exposes its three real child pages', async ({ page }) => {
    const missingPages: string[] = [];
    page.on('response', response => {
      if (response.status() === 404 && response.url().includes('bizscope')) missingPages.push(response.url());
    });
    await page.goto('/business/bizscope/scopes', { waitUntil: 'domcontentloaded' });
    const parent = page.locator('.arco-menu-inline-header').filter({ has: page.getByText('业务域', { exact: true }) }).first();
    await expect(parent).toBeVisible();
    for (const destination of destinations) {
      const child = page.locator('.arco-menu-item').filter({ has: page.getByText(destination.label, { exact: true }) }).first();
      if (!(await child.isVisible())) await parent.click();
      await expect(child).toBeVisible();
      await child.click();
      await expect(page).toHaveURL(new RegExp(`${destination.path}$`));
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
      await assertNoRawKeys(page);
    }
    expect(missingPages).toEqual([]);
  });

  test('scope creation form adds a real business scope to the new list route', async ({ page, owned }, testInfo) => {
    await page.goto('/business/bizscope/scopes', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const modal = page.locator('.arco-modal:visible').last();
    await modal.getByPlaceholder('请输入业务域编码', { exact: true }).fill(owned.token);
    await modal.getByPlaceholder('请输入业务域名称', { exact: true }).fill(`${owned.token}-created`);
    const created = await success<Row>(await mutation(page, 'POST', '/business/bizscope', async () => {
      await modal.getByRole('button', { name: '新增', exact: true }).click();
    }));
    owned.track(`/business/bizscope/${created.id}`);
    await expect(modal).not.toBeVisible();
    await page.locator('.search-toolbar input').first().fill(owned.token);
    await expect(page.getByRole('row').filter({ hasText: owned.token })).toContainText(`${owned.token}-created`);
    await assertNoRawKeys(page);
    await screenshot(page, testInfo, 'bizscope-scopes-desktop');
  });

  test('scope editing preserves its ID and updates the name', async ({ page, owned }) => {
    const scope = await owned.scope();
    const row = await showScope(page, scope.code);
    await row.getByRole('button', { name: '编辑', exact: true }).click();
    const modal = page.locator('.arco-modal:visible').last();
    await modal.getByPlaceholder('请输入业务域名称', { exact: true }).fill(`${owned.token}-updated`);
    await success(await mutation(page, 'PUT', `/business/bizscope/${scope.id}`, async () => {
      await modal.getByRole('button', { name: '保存', exact: true }).click();
    }));
    await expect(row).toContainText(`${owned.token}-updated`);
  });

  test('scope deletion removes its captured row from the new list route', async ({ page, owned }) => {
    const scope = await owned.scope();
    const row = await showScope(page, scope.code);
    await row.getByRole('button', { name: '删除', exact: true }).click();
    await success(await mutation(page, 'DELETE', `/business/bizscope/${scope.id}`, async () => { await confirmDelete(page); }));
    owned.forget(`/business/bizscope/${scope.id}`);
    await expect(row).toHaveCount(0);
  });

  test('host binding filter shows only hosts in the selected business scope', async ({ page, owned }, testInfo) => {
    const selectedScope = await owned.scope('selected');
    const excludedScope = await owned.scope('excluded');
    const selectedHost = await owned.host(selectedScope.id, 'selected-host');
    const excludedHost = await owned.host(excludedScope.id, 'excluded-host');
    await page.goto('/business/bizscope/host-bindings', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.governance-summary-bar')).toContainText('主机绑定');
    await page.locator('.search-toolbar input').first().fill(owned.token);
    await expect(page.locator('.system-list__table-card')).toContainText(selectedHost.hostname);
    await expect(page.locator('.system-list__table-card')).toContainText(excludedHost.hostname);
    await page.locator('.search-toolbar [aria-label="业务域"]').click();
    const filteredResponse = page.waitForResponse(response => {
      const url = new URL(response.url());
      return url.pathname.endsWith('/business/cmdb/hosts')
        && url.searchParams.get('businessScopeId') === String(selectedScope.id)
        && response.request().method() === 'GET';
    });
    await page.locator('.arco-select-option').filter({ hasText: selectedScope.name }).click();
    await success(await filteredResponse);
    const table = page.locator('.system-list__table-card');
    await expect(table).toContainText(selectedHost.hostname);
    await expect(table).not.toContainText(excludedHost.hostname);
    await expect(table).toContainText(selectedScope.name);
    await assertNoRawKeys(page);
    await screenshot(page, testInfo, 'bizscope-host-bindings-desktop');
  });

  test('environment creation form persists the selected business scope', async ({ page, owned }, testInfo) => {
    const scope = await owned.scope();
    await page.goto('/business/bizscope/environments', { waitUntil: 'domcontentloaded' });
    await page.getByRole('button', { name: '新增', exact: true }).click();
    const modal = page.locator('.arco-modal:visible').last();
    await modal.locator('.arco-form-item').filter({ has: page.getByText('业务域', { exact: true }) }).locator('.arco-select').click();
    await page.locator('.arco-select-option').filter({ hasText: scope.name }).click();
    await modal.getByPlaceholder('环境编码', { exact: true }).fill(`${owned.token}-env`);
    await modal.getByPlaceholder('环境名称', { exact: true }).fill(`${owned.token}-created`);
    await screenshot(page, testInfo, 'bizscope-environment-create-form-desktop');
    const created = await success<Row & { businessScopeId: number }>(await mutation(page, 'POST', '/business/bizscope/environments', async () => {
      await modal.getByRole('button', { name: '保存', exact: true }).click();
    }));
    owned.track(`/business/bizscope/environments/${created.id}`);
    expect(created.businessScopeId).toBe(scope.id);
    await expect(modal).not.toBeVisible();
    const row = await showEnvironment(page, created.code);
    await expect(row).toContainText(`${owned.token}-created`);
    await expect(row).toContainText(scope.name);
    await assertNoRawKeys(page);
    await screenshot(page, testInfo, 'bizscope-environments-desktop');
  });

  test('environment editing updates its name without changing ownership', async ({ page, owned }) => {
    const scope = await owned.scope();
    const environment = await owned.environment(scope.id);
    const row = await showEnvironment(page, environment.code);
    await row.getByRole('button', { name: '编辑', exact: true }).click();
    const modal = page.locator('.arco-modal:visible').last();
    await modal.getByPlaceholder('环境名称', { exact: true }).fill(`${owned.token}-updated`);
    await success(await mutation(page, 'PUT', `/business/bizscope/environments/${environment.id}`, async () => {
      await modal.getByRole('button', { name: '保存', exact: true }).click();
    }));
    await expect(row).toContainText(`${owned.token}-updated`);
    await expect(row).toContainText(scope.name);
  });

  test('environment deletion removes only its captured child row', async ({ page, owned }) => {
    const scope = await owned.scope();
    const environment = await owned.environment(scope.id);
    const row = await showEnvironment(page, environment.code);
    await row.getByRole('button', { name: '删除', exact: true }).click();
    await success(await mutation(page, 'DELETE', `/business/bizscope/environments/${environment.id}`, async () => { await confirmDelete(page); }));
    owned.forget(`/business/bizscope/environments/${environment.id}`);
    await expect(row).toHaveCount(0);
    const scopeRow = await showScope(page, scope.code);
    await expect(scopeRow).toContainText(scope.name);
  });

  for (const destination of destinations) {
    test(`${destination.label} remains readable at 390px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width: 390, height: 844 });
      await page.goto(destination.path, { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.label);
      await expect.poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1)).toBe(true);
      await assertNoRawKeys(page);
      await screenshot(page, testInfo, `${destination.slug}-mobile`);
    });
  }

  for (const width of [1440, 390]) {
    test(`application delivery uses its de-branded display name at ${width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize({ width, height: width === 1440 ? 900 : 844 });
      await page.goto('/business/cicd/argocd', { waitUntil: 'domcontentloaded' });
      await expect(page).toHaveURL(/\/business\/cicd\/argocd$/);
      await expect(page.locator('.governance-summary-bar')).toContainText('应用发布');
      await expect(page.locator('.governance-summary-bar')).not.toContainText('ArgoCD');
      await screenshot(page, testInfo, `application-delivery-${width === 1440 ? 'desktop' : 'mobile'}`);
    });
  }
});
