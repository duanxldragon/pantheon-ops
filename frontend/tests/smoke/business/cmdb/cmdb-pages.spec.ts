import { test, expect, type APIRequestContext, type APIResponse } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

type ApiEnvelope<T> = {
  code: number;
  data: T;
  message?: string;
};

type CmdbHost = {
  id: number;
  hostname: string;
  ip: string;
  os: string;
  status: string;
  labelValues: Array<{ key: string; val: string }>;
};

type CmdbHostList = {
  items: CmdbHost[];
  total: number;
  page: number;
  pageSize: number;
};

type CmdbGroup = {
  id: number;
  name: string;
  description: string;
  memberCount: number;
  conditions: {
    operator: 'AND' | 'OR';
    rules: Array<{ key: string; op: 'eq' | 'neq' | 'in' | 'notIn'; val: string }>;
  };
};

type BizScope = {
  id: number;
  code: string;
  name: string;
  hostCount?: number;
};

async function expectNoVisibleCmdbI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.cmdb\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.cmdb\.[A-Za-z0-9_.-]+\b/);
}

async function expectBusinessSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = (await response.json()) as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

async function cleanupCmdbFixture(request: APIRequestContext, headers: Record<string, string>, token: string) {
  const hostListResp = await request.get(`${apiBaseUrl}/business/cmdb/hosts?keyword=${token}&page=1&pageSize=50`, {
    headers,
  });
  if (hostListResp.ok()) {
    const payload = (await hostListResp.json()) as ApiEnvelope<CmdbHostList>;
    for (const host of payload.data?.items ?? []) {
      await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${host.id}`, { headers });
    }
  }

  const groupListResp = await request.get(`${apiBaseUrl}/business/cmdb/groups`, { headers });
  if (groupListResp.ok()) {
    const payload = (await groupListResp.json()) as ApiEnvelope<CmdbGroup[]>;
    for (const group of payload.data ?? []) {
      if (group.name.includes(token)) {
        await request.delete(`${apiBaseUrl}/business/cmdb/groups/${group.id}`, { headers });
      }
    }
  }

  const scopeListResp = await request.get(`${apiBaseUrl}/business/bizscope/list?keyword=${token}&page=1&pageSize=50`, {
    headers,
  });
  if (scopeListResp.ok()) {
    const payload = (await scopeListResp.json()) as ApiEnvelope<{ items: BizScope[] }>;
    for (const scope of payload.data?.items ?? []) {
      if (scope.code.includes(token) || scope.name.includes(token)) {
        await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers });
      }
    }
  }
}

test.describe('CMDB Host Management', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('host list page loads with table', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    await page.goto('/business/cmdb/host', { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    const operationsMenuLabel = page.locator('.app-shell__menu-entry-label', {
      hasText: '运维平台',
    });
    if (await operationsMenuLabel.count()) {
      await expect(
        operationsMenuLabel.filter({ has: page.locator(':visible') }),
      ).toHaveCount(0);
    }
    await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
    await expect(page.locator('.governance-summary-bar__icon')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar')).toBeVisible();
    await expect(page.locator('.governance-summary-bar .arco-btn')).toBeVisible();
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expect(page.getByRole('button', { name: '新增' })).toBeVisible();
    await expectNoVisibleCmdbI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('cmdb-host-list.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('host detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    await installClientSession(page, login);
    const headers = apiRequestHeaders(login);
    const token = `cmdb-detail-modal-${Date.now()}`;

    await cleanupCmdbFixture(request, headers, token);

    try {
      const host = await expectBusinessSuccess<CmdbHost>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: token,
            ip: `10.252.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Ubuntu 24.04',
            cpuCores: 4,
            memoryGb: 8,
            diskGb: 100,
            labels: [{ key: 'biz', val: token }],
            owner: 'cmdb-smoke',
            remark: token,
          },
        }),
      );

      await page.goto('/business/cmdb/host', { waitUntil: 'networkidle' });
      await expect(page).toHaveURL(/\/business\/cmdb\/host$/);
      await page.locator('.search-toolbar input').first().fill(token);

      const row = page.getByRole('row').filter({ hasText: host.hostname }).first();
      await expect(row).toBeVisible();
      await row.getByRole('button', { name: '详情' }).click();

      const modal = page.locator('.arco-modal:visible').last();
      await expect(modal).toBeVisible();
      await expect(modal).toContainText(host.hostname);
      await expect(modal).toContainText(host.ip);
      await expect(modal).toContainText('已装组件');
      await expect(page).toHaveURL(/\/business\/cmdb\/host$/);
      await page.screenshot({ path: testInfo.outputPath('cmdb-host-detail-modal.png'), fullPage: true });
    } finally {
      await cleanupCmdbFixture(request, headers, token);
    }
  });

  test('host detail loads by ID', async ({ page }, testInfo) => {
    await page.goto('/business/cmdb/host/1', { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(
      page.getByRole('heading', { name: /主机详情|加载失败/ }),
    ).toBeVisible();
    const collectButton = page.getByRole('button', { name: '采集配置' });
    if (await collectButton.count()) {
      await collectButton.click();
      await expect(page.locator('.arco-modal')).toBeVisible();
      await page.keyboard.press('Escape');
    } else {
      await expect(page.getByRole('button', { name: '重试' })).toBeVisible();
    }
    await expectNoVisibleCmdbI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('cmdb-host-detail.png'), fullPage: true });
  });

  test('host detail lists its service instance and the business scope counts it', async ({ page, request }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const login = await loginByApi(request, adminCredentials);
    await installClientSession(page, login);
    const headers = apiRequestHeaders(login);
    const token = `cmdb-rel-${Date.now()}`;
    let scopeId = 0;
    let hostId = 0;
    let applicationId = 0;
    let serviceId = 0;
    let instanceId = 0;
    let packageId = 0;
    let taskId = 0;

    try {
      const scope = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/bizscope`, {
          headers,
          data: { code: token, name: token, environment: 'prod', status: 'active' },
        }),
      );
      scopeId = scope.id;
      const host = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `${token}-host`,
            ip: `10.253.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            labels: [{ key: 'rel', val: token }],
            owner: 'cmdb-relation-smoke',
            remark: token,
          },
        }),
      );
      hostId = host.id;
      await expectBusinessSuccess(
        await request.post(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/bind`, {
          headers,
          data: { hostIds: [hostId] },
        }),
      );
      const application = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/applications`, {
          headers,
          data: { code: `${token}-app`, name: `${token}-application`, businessScopeId: scopeId },
        }),
      );
      applicationId = application.id;
      const service = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/services`, {
          headers,
          data: { applicationId, code: `${token}-svc`, name: `${token}-service`, runtimeType: 'vm' },
        }),
      );
      serviceId = service.id;
      const instance = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/service/instances`, {
          headers,
          data: { serviceId, environment: 'prod', targetType: 'vm', hostId, desiredVersion: '2.0.0' },
        }),
      );
      instanceId = instance.id;
      const pkg = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/deploy/packages`, {
          headers,
          data: { name: `${token}-pkg`, version: '1.0.0', installCommand: 'echo install', status: 'enabled' },
        }),
      );
      packageId = pkg.id;
      const task = await expectBusinessSuccess<{ id: number }>(
        await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
          headers,
          data: {
            name: `${token}-task`,
            packageId,
            businessScopeId: scopeId,
            serviceId,
            serviceInstanceId: instanceId,
            targetType: 'host',
            targetIds: [hostId],
            executorType: 'manual',
          },
        }),
      );
      taskId = task.id;

      await page.goto(`/business/cmdb/host/${hostId}`, { waitUntil: 'domcontentloaded' });
      await expect(page.getByText('承载服务实例', { exact: true })).toBeVisible();
      const instanceCard = page.locator('.page-panel').filter({ hasText: '承载服务实例' }).first();
      await expect(instanceCard).toContainText(`${token}-service`);
      await expect(page.getByText('最近部署', { exact: true })).toBeVisible();
      await expect(page.getByText(`${token}-task`)).toBeVisible();
      await expectNoVisibleCmdbI18nKeys(page);
      await page.screenshot({ path: testInfo.outputPath('cmdb-host-relations-1440.png') });

      await page.goto(`/business/business-scope/${scopeId}`, { waitUntil: 'domcontentloaded' });
      const summaryCard = page.locator('.page-panel').filter({ hasText: '服务目录统计' }).first();
      await expect(summaryCard).toBeVisible();
      await expect(summaryCard).toContainText('应用');
      await expect(summaryCard).toContainText('服务');
      await expect(summaryCard).toContainText('实例');
      await page.screenshot({ path: testInfo.outputPath('bizscope-catalog-summary-1440.png') });

      await page.setViewportSize({ width: 390, height: 844 });
      await page.goto(`/business/cmdb/host/${hostId}`, { waitUntil: 'domcontentloaded' });
      await expect(page.getByText('承载服务实例', { exact: true })).toBeVisible();
      await expect
        .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
        .toBe(true);
      await page.screenshot({ path: testInfo.outputPath('cmdb-host-relations-390.png') });
    } finally {
      const cleanupFailures: string[] = [];
      const cleanup = async (label: string, responsePromise: Promise<APIResponse>) => {
        try {
          const response = await responsePromise;
          const payload = (await response.json().catch(() => null)) as ApiEnvelope<unknown> | null;
          if (!response.ok() || payload?.code !== 200) {
            cleanupFailures.push(`${label}: HTTP ${response.status()} / code ${payload?.code ?? 'unknown'}`);
          }
        } catch (error) {
          cleanupFailures.push(`${label}: ${error instanceof Error ? error.message : String(error)}`);
        }
      };
      if (taskId) await cleanup('task', request.delete(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
      if (packageId) await cleanup('package', request.delete(`${apiBaseUrl}/business/deploy/packages/${packageId}`, { headers }));
      if (instanceId) await cleanup('instance', request.delete(`${apiBaseUrl}/business/service/instances/${instanceId}`, { headers }));
      if (serviceId) await cleanup('service', request.delete(`${apiBaseUrl}/business/service/services/${serviceId}`, { headers }));
      if (applicationId) await cleanup('application', request.delete(`${apiBaseUrl}/business/service/applications/${applicationId}`, { headers }));
      if (scopeId && hostId) await cleanup('unbind host', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}/hosts/${hostId}`, { headers }));
      if (hostId) await cleanup('host', request.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostId}`, { headers }));
      if (scopeId) await cleanup('scope', request.delete(`${apiBaseUrl}/business/bizscope/${scopeId}`, { headers }));
      expect(cleanupFailures).toEqual([]);
    }
  });

  test('group list page loads', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    await page.goto('/business/cmdb/group', { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.cmdb-page__side-panel')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar .arco-btn-primary')).toBeVisible();
    const hasGroups = await page.locator('.arco-tree').count();
    if (hasGroups > 0) {
      await expect(page.locator('.arco-tree')).toBeVisible();
    } else {
      await expect(page.locator('.cmdb-page__side-panel')).toContainText('暂无分组');
    }
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expectNoVisibleCmdbI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('cmdb-group-list.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('business scope page loads as an independent module', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    await page.goto('/business/business-scope', { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toContainText('业务域');
    await expect(page.locator('.filter-panel')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('bizscope-list.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('business scope detail binds and unbinds hosts', async ({ page, request }, testInfo) => {
    const login = await loginByApi(request, adminCredentials);
    await installClientSession(page, login);
    const headers = apiRequestHeaders(login);
    const token = `bizscope-detail-${Date.now()}`;

    await cleanupCmdbFixture(request, headers, token);

    const createdHostIds: number[] = [];
    let createdScopeId = 0;

    try {
      const scope = await expectBusinessSuccess<BizScope>(
        await request.post(`${apiBaseUrl}/business/bizscope`, {
          headers,
          data: {
            code: `scope-${token}`,
            name: `业务域-${token}`,
            owner: 'bizscope-smoke',
            environment: 'prod',
            status: 'active',
            remark: token,
          },
        }),
      );
      createdScopeId = scope.id;

      const firstHost = await expectBusinessSuccess<CmdbHost>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `host-a-${token}`,
            ip: `10.240.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Ubuntu 24.04',
            cpuCores: 4,
            memoryGb: 8,
            diskGb: 100,
            labels: [{ key: 'biz', val: token }],
            owner: 'bizscope-smoke',
            remark: token,
          },
        }),
      );
      createdHostIds.push(firstHost.id);

      const secondHost = await expectBusinessSuccess<CmdbHost>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `host-b-${token}`,
            ip: `10.241.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Debian 12',
            cpuCores: 2,
            memoryGb: 4,
            diskGb: 50,
            labels: [{ key: 'biz', val: token }],
            owner: 'bizscope-smoke',
            remark: token,
          },
        }),
      );
      createdHostIds.push(secondHost.id);

      await page.goto(`/business/business-scope/${createdScopeId}`, { waitUntil: 'networkidle' });
      await expect(page.locator('.cmdb-page__hero-title')).toHaveText(scope.name);
      await expect(page.locator('.cmdb-page__hero-grid')).toContainText('0');

      await page.getByRole('button', { name: '绑定主机' }).first().click();
      const drawer = page.locator('.app-drawer:visible');
      await expect(drawer).toBeVisible();
      await expect(drawer).toContainText(firstHost.hostname);
      await expect(drawer).toContainText(secondHost.hostname);

      const availableFirstHostRow = drawer.getByRole('row').filter({ hasText: firstHost.hostname }).first();
      await availableFirstHostRow.locator('.arco-checkbox-mask').first().click();
      const bindOneHostButton = drawer.getByRole('button', { name: '绑定 1 台主机' });
      await expect(bindOneHostButton).toBeEnabled();
      await bindOneHostButton.click();

      const boundRow = page.getByRole('row').filter({ hasText: firstHost.hostname }).first();
      await expect(boundRow).toBeVisible();
      await expect(boundRow).toContainText('已分配');
      await expect(page.locator('.cmdb-page__hero-grid')).toContainText('1');

      const scopeDetail = await expectBusinessSuccess<BizScope>(
        await request.get(`${apiBaseUrl}/business/bizscope/${createdScopeId}`, { headers }),
      );
      expect(scopeDetail.hostCount).toBe(1);

      const hostDetail = await expectBusinessSuccess<CmdbHost>(
        await request.get(`${apiBaseUrl}/business/cmdb/hosts/${firstHost.id}`, { headers }),
      );
      expect(hostDetail.status).toBe('assigned');

      await boundRow.getByRole('button', { name: '解绑' }).click();
      await page.getByRole('button', { name: '确定' }).last().click();
      await expect(page.getByText(firstHost.hostname)).toHaveCount(0);
      await expect(page.locator('.cmdb-page__hero-grid')).toContainText('0');

      const unboundHost = await expectBusinessSuccess<CmdbHost>(
        await request.get(`${apiBaseUrl}/business/cmdb/hosts/${firstHost.id}`, { headers }),
      );
      expect(unboundHost.status).toBe('pending');

      await page.screenshot({ path: testInfo.outputPath('bizscope-detail-bind-unbind.png'), fullPage: true });
    } finally {
      for (const hostId of createdHostIds.reverse()) {
        await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${hostId}`, { headers });
      }
      if (createdScopeId) {
        await request.delete(`${apiBaseUrl}/business/bizscope/${createdScopeId}`, { headers });
      }
      await cleanupCmdbFixture(request, headers, token);
    }
  });

  test('label management page loads without raw i18n keys', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });
    await page.goto('/business/cmdb/label', { waitUntil: 'networkidle' });
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await page.locator('.governance-summary-bar .arco-btn').click();
    await expect(page.locator('.governance-insight-drawer')).toBeVisible();
    await expect(page.locator('.table-batch-action-bar .arco-btn-primary')).toBeVisible();
    await expectNoVisibleCmdbI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('cmdb-label-management-list.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('cmdb pages stay within a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/cmdb/host', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.getByRole('button', { name: '新增' })).toBeVisible();
    await expect
      .poll(async () =>
        page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1),
      )
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('cmdb-host-list-mobile.png'), fullPage: true });

    await page.goto('/business/cmdb/group', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('cmdb-group-list-mobile.png'), fullPage: true });

    await page.goto('/business/cmdb/label', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('cmdb-label-management-list-mobile.png'), fullPage: true });

    await page.goto('/business/business-scope', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('bizscope-list-mobile.png'), fullPage: true });
  });
});

test.describe('CMDB Business API Smoke', () => {
  test('host and group lifecycle validates labels, status, members, and cleanup', async ({ request }) => {
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `smoke-${Date.now()}`;
    const hostIds: number[] = [];
    const groupIds: number[] = [];

    await cleanupCmdbFixture(request, headers, token);

    try {
      const prodHost = await expectBusinessSuccess<CmdbHost>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `cmdb-${token}-prod`,
            ip: `10.254.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Ubuntu 24.04',
            cpuCores: 4,
            memoryGb: 8,
            diskGb: 100,
            labels: [
              { key: 'env', val: 'production' },
              { key: 'biz', val: token },
              { key: 'region', val: 'east-1' },
            ],
            owner: 'cmdb-smoke',
            remark: token,
          },
        }),
      );
      hostIds.push(prodHost.id);
      expect(prodHost.status).toBe('pending');
      expect(prodHost.labelValues).toEqual(
        expect.arrayContaining([
          expect.objectContaining({ key: 'env', val: 'production' }),
          expect.objectContaining({ key: 'biz', val: token }),
        ]),
      );

      const testHost = await expectBusinessSuccess<CmdbHost>(
        await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
          headers,
          data: {
            hostname: `cmdb-${token}-test`,
            ip: `10.253.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
            sshPort: 22,
            os: 'linux',
            osVersion: 'Debian 12',
            cpuCores: 2,
            memoryGb: 4,
            diskGb: 50,
            labels: [
              { key: 'env', val: 'test' },
              { key: 'biz', val: token },
              { key: 'region', val: 'east-1' },
            ],
            owner: 'cmdb-smoke',
            remark: token,
          },
        }),
      );
      hostIds.push(testHost.id);

      await expectBusinessSuccess<null>(
        await request.patch(`${apiBaseUrl}/business/cmdb/hosts/${prodHost.id}/status`, {
          headers,
          data: { status: 'online' },
        }),
      );

      const detail = await expectBusinessSuccess<CmdbHost>(
        await request.get(`${apiBaseUrl}/business/cmdb/hosts/${prodHost.id}`, { headers }),
      );
      expect(detail.status).toBe('online');

      const filtered = await expectBusinessSuccess<CmdbHostList>(
        await request.get(`${apiBaseUrl}/business/cmdb/hosts?keyword=${token}&status=online&page=1&pageSize=20`, {
          headers,
        }),
      );
      expect(filtered.items.map((item) => item.id)).toContain(prodHost.id);
      expect(filtered.items.map((item) => item.id)).not.toContain(testHost.id);

      const prodGroup = await expectBusinessSuccess<CmdbGroup>(
        await request.post(`${apiBaseUrl}/business/cmdb/groups`, {
          headers,
          data: {
            name: `cmdb-${token}-prod-group`,
            description: 'CMDB smoke production group',
            conditions: {
              operator: 'AND',
              rules: [
                { key: 'env', op: 'eq', val: 'production' },
                { key: 'biz', op: 'eq', val: token },
              ],
            },
          },
        }),
      );
      groupIds.push(prodGroup.id);
      expect(prodGroup.memberCount).toBe(1);

      const members = await expectBusinessSuccess<{ groupId: number; groupName: string; members: CmdbHost[] }>(
        await request.get(`${apiBaseUrl}/business/cmdb/groups/${prodGroup.id}/members`, { headers }),
      );
      expect(members.groupId).toBe(prodGroup.id);
      expect(members.members.map((item) => item.id)).toEqual([prodHost.id]);

      const updatedGroup = await expectBusinessSuccess<CmdbGroup>(
        await request.put(`${apiBaseUrl}/business/cmdb/groups/${prodGroup.id}`, {
          headers,
          data: {
            conditions: {
              operator: 'AND',
              rules: [
                { key: 'biz', op: 'eq', val: token },
                { key: 'env', op: 'in', val: 'production,test' },
              ],
            },
          },
        }),
      );
      expect(updatedGroup.memberCount).toBe(2);

      const invalidGroupResp = await request.post(`${apiBaseUrl}/business/cmdb/groups`, {
        headers,
        data: {
          name: `cmdb-${token}-invalid`,
          conditions: { operator: 'AND', rules: [] },
        },
      });
      expect(invalidGroupResp.ok()).toBeTruthy();
      const invalidPayload = (await invalidGroupResp.json()) as ApiEnvelope<unknown>;
      expect(invalidPayload.code).not.toBe(200);
      expect(invalidPayload.message).toContain('cmdbgroup');
    } finally {
      for (const id of groupIds.reverse()) {
        await request.delete(`${apiBaseUrl}/business/cmdb/groups/${id}`, { headers });
      }
      for (const id of hostIds.reverse()) {
        await request.delete(`${apiBaseUrl}/business/cmdb/hosts/${id}`, { headers });
      }
      await cleanupCmdbFixture(request, headers, token);
    }
  });
});
