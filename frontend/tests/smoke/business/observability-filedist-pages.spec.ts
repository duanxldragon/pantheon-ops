import { expect, test, type APIResponse } from '@playwright/test';
import {
  apiBaseUrl,
  apiRequestHeaders,
  adminCredentials,
  installClientSession,
  loginByApi,
} from '../helpers/auth';

type ApiEnvelope<T> = { code: number; data: T; message?: string };

async function expectSuccess<T>(response: APIResponse): Promise<T> {
  expect(response.ok()).toBeTruthy();
  const payload = await response.json() as ApiEnvelope<T>;
  expect(payload.code).toBe(200);
  return payload.data;
}

test.describe('Observability and file distribution Web smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
  });

  test('observability target, datasource, rule lifecycle is visible in the Web UI', async ({ page, request }, testInfo) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    const login = await loginByApi(request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const token = `web-observability-${Date.now()}`;

    const scope = await expectSuccess<{ id: number }>(
      await request.post(`${apiBaseUrl}/business/bizscope`, {
        headers,
        data: { code: token, name: token, environment: 'dev', status: 'active' },
      }),
    );
    const target = await expectSuccess<{ id: number }>(
      await request.post(`${apiBaseUrl}/business/observability/targets`, {
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
    const datasource = await expectSuccess<{ id: number }>(
      await request.post(`${apiBaseUrl}/business/observability/datasources`, {
        headers,
        data: {
          name: `${token}-prometheus`,
          type: 'prometheus',
          url: 'http://127.0.0.1:1',
          accessMode: 'proxy',
          timeout: 1,
          isDefault: false,
        },
      }),
    );
    const rule = await expectSuccess<{ id: number }>(
      await request.post(`${apiBaseUrl}/business/observability/alert-rules`, {
        headers,
        data: {
          name: `${token}-rule`,
          datasourceId: datasource.id,
          enabled: true,
          query: 'up == 0',
          interval: 10,
          // The Go binding treats zero as missing for the required field.
          forDuration: 1,
          severity: 'warning',
        },
      }),
    );

    try {
      await page.goto('/business/observability/monitoring', { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.system-list__table-card')).toContainText(`${token}-target`);
      await expect(page.locator('.system-list__table-card')).not.toContainText('business.observability');
      await page.screenshot({ path: testInfo.outputPath('observability-monitoring-desktop.png') });

      await page.goto('/business/observability/alerts', { waitUntil: 'domcontentloaded' });
      const ruleCard = page.locator('.system-list__table-card');
      await expect(ruleCard).toContainText(`${token}-rule`);
      const ruleSwitch = ruleCard.getByRole('switch').first();
      await expect(ruleSwitch).toBeVisible();
      await ruleSwitch.click();
      await expect.poll(async () => {
        const detail = await expectSuccess<{ enabled: boolean }>(
          await request.get(`${apiBaseUrl}/business/observability/alert-rules/${rule.id}`, { headers }),
        );
        return detail.enabled;
      }).toBe(false);
      await page.screenshot({ path: testInfo.outputPath('observability-alerts-desktop.png') });

      await page.goto('/business/observability/logs', { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.governance-summary-bar')).toContainText('日志中心');
      await expect(page.locator('.system-list__table-card')).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath('observability-logs-desktop.png') });

      await page.goto('/business/observability/traces', { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.governance-summary-bar')).toContainText('链路追踪');
      await expect(page.locator('.system-list__table-card')).toBeVisible();
      await expect(page.locator('body')).not.toContainText('business.observability');
      await page.screenshot({ path: testInfo.outputPath('observability-traces-desktop.png') });
    } finally {
      await request.delete(`${apiBaseUrl}/business/observability/alert-rules/${rule.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/observability/datasources/${datasource.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/observability/targets/${target.id}`, { headers }).catch(() => undefined);
      await request.delete(`${apiBaseUrl}/business/bizscope/${scope.id}`, { headers }).catch(() => undefined);
    }
  });

  test('observability centers remain usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    const destinations = [
      { path: '/business/observability/monitoring', title: '监控中心', screenshot: 'observability-monitoring-mobile.png' },
      { path: '/business/observability/alerts', title: '告警中心', screenshot: 'observability-alerts-mobile.png' },
      { path: '/business/observability/logs', title: '日志中心', screenshot: 'observability-logs-mobile.png' },
      { path: '/business/observability/traces', title: '链路追踪', screenshot: 'observability-traces-mobile.png' },
    ];
    for (const destination of destinations) {
      await page.goto(destination.path, { waitUntil: 'domcontentloaded' });
      await expect(page.locator('.governance-summary-bar')).toContainText(destination.title);
      await expect
        .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
        .toBe(true);
      await page.screenshot({ path: testInfo.outputPath(destination.screenshot) });
    }
  });

  test('file distribution Web UI uploads and removes a file', async ({ page }, testInfo) => {
	await page.setViewportSize({ width: 1440, height: 900 });
    const filename = `filedist-web-${Date.now()}.txt`;
    await page.goto('/business/filedist/files', { waitUntil: 'domcontentloaded' });
    await expect(page.getByRole('button', { name: '上传文件' })).toBeVisible();
    await page.getByRole('button', { name: '上传文件' }).click();
    const modal = page.locator('.arco-modal').last();
    await expect(modal).toBeVisible();
    await modal.locator('input[type="file"]').setInputFiles({
      name: filename,
      mimeType: 'text/plain',
      buffer: Buffer.from('file distribution web smoke', 'utf8'),
    });
    await modal.getByRole('button', { name: '确定' }).click();
    await expect(page.locator('.system-list__table-card')).toContainText(filename);

    const login = await loginByApi(page.request, adminCredentials);
    // The API login refreshes the shared browser cookie jar. Keep the page's
    // CSRF snapshot aligned before exercising mutating UI actions.
    await installClientSession(page, login);
    const headers = apiRequestHeaders(login);
    const files = await expectSuccess<{ items: Array<{ id: number; fileName?: string; name?: string }> }>(
      await page.request.get(`${apiBaseUrl}/business/filedist/files?page=1&pageSize=100`, { headers }),
    );
    const uploaded = files.items.find((item) => (item.fileName || item.name) === filename);
    expect(uploaded).toBeTruthy();
    const taskToken = `filedist-detail-${Date.now()}`;
    const task = await expectSuccess<{ id: number }>(
      await page.request.post(`${apiBaseUrl}/business/filedist/tasks`, {
        headers,
        data: {
          name: taskToken,
          fileId: uploaded!.id,
          targets: [{ host: '127.0.0.1', port: 1, targetPath: '/tmp/smoke.txt', username: 'root', hostFingerprint: 'SHA256:test', maxRetries: 0 }],
        },
      }),
    );
    await page.goto(`/business/filedist/tasks/${task.id}`, { waitUntil: 'domcontentloaded' });
    await expect(page.locator('body')).toContainText(taskToken);
    await expect(page.locator('body')).not.toContainText('undefined');

    // Verify the tasks list route renders the task row.
    await page.goto('/business/filedist/tasks', { waitUntil: 'domcontentloaded' });
    await expect(page.locator('.system-list__table-card')).toContainText(taskToken);

    // The auto-started transfer to 127.0.0.1:1 settles as failed within a
    // couple of seconds. The backend rejects deleting a running task (409), so
    // wait for a terminal state before removing it — otherwise every run leaks
    // a task row into the database.
    await expect
      .poll(
        async () => {
          const detail = await expectSuccess<{ status: string }>(
            await page.request.get(`${apiBaseUrl}/business/filedist/tasks/${task.id}`, { headers }),
          );
          return detail.status;
        },
        { timeout: 20000 },
      )
      .not.toBe('running');
    const deletedTask = await expectSuccess<{ deleted: boolean }>(
      await page.request.delete(`${apiBaseUrl}/business/filedist/tasks/${task.id}`, { headers }),
    );
    expect(deletedTask.deleted).toBe(true);

    // Navigate to the files list by route, not by clicking the Arco tabs: the
    // active tab here is derived from the URL, and tab clicks race React
    // Router's asynchronous URL update — the previous tab's navigation can
    // land after the next click, leaving the file table hidden exactly when
    // the test tries to interact with it.
    await page.goto('/business/filedist/files', { waitUntil: 'domcontentloaded' });
    const row = page.locator('.arco-table-tr').filter({ hasText: filename }).first();
    await expect(row).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath('filedist-files-desktop.png') });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.keyboard.press('Escape');
    await page.waitForTimeout(200);
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('filedist-files-mobile.png') });
    await page.setViewportSize({ width: 1440, height: 900 });
    await row.getByRole('button', { name: '删除' }).click();
    await page.getByRole('button', { name: '确定' }).last().click();
    await page.reload({ waitUntil: 'networkidle' });
    await expect(page.locator('.system-list__table-card')).not.toContainText(filename);
  });

  test('115.190.251.143 experiment host is visible across CMDB, business scope, and deployment UI', async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    const headers = apiRequestHeaders(login);
    const hostList = await expectSuccess<{ items: Array<{ id: number; ip: string; businessScopeId: number; businessScopeName: string }> }>(
      await page.request.get(`${apiBaseUrl}/business/cmdb/hosts?keyword=115.190.251.143&page=1&pageSize=20`, { headers }),
    );
    const host = hostList.items.find((item) => item.ip === '115.190.251.143');
    expect(host).toBeTruthy();

    await page.goto(`/business/cmdb/host/${host!.id}`, { waitUntil: 'networkidle' });
    await expect(page.getByRole('tab', { name: '主机详情' })).toBeVisible();
    await expect(page.locator('body')).toContainText('115.190.251.143');
    await expect(page.locator('body')).toContainText(host!.businessScopeName);

    await page.goto(`/business/business-scope/${host!.businessScopeId}`, { waitUntil: 'networkidle' });
    await expect(page.locator('.cmdb-page__hero-title')).toHaveText(host!.businessScopeName);
    await expect(page.locator('body')).toContainText('115.190.251.143');

    const taskList = await expectSuccess<{ items: Array<{ id: number; hosts?: Array<{ hostId: number }> }> }>(
      await page.request.get(`${apiBaseUrl}/business/deploy/tasks?page=1&pageSize=100`, { headers }),
    );
    const task = taskList.items.find((item) => item.hosts?.some((itemHost) => itemHost.hostId === host!.id));
    if (task) {
      await page.goto(`/business/deploy/task/${task.id}`, { waitUntil: 'domcontentloaded' });
    } else {
      await page.goto('/business/deploy/task', { waitUntil: 'domcontentloaded' });
    }
    await expect(page.locator('.page-container')).toBeVisible();
    if (task) {
      await expect(page.locator('body')).toContainText('115.190.251.143');
    } else {
      await expect(page.locator('.governance-summary-bar')).toContainText('部署任务');
    }
  });
});
