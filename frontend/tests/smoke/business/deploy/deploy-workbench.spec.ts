import { startReviewedTask } from './reviewed-start';
import fs from 'node:fs/promises';
import { expect, test, type APIResponse } from '@playwright/test';
import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';

// Includes real fixture lifecycle, multiple rendered states and remote asset loading.
// Keep assertion deadlines unchanged while giving this scenario a bounded budget.
test.describe.configure({ timeout: 90_000 });

async function success<T>(response: APIResponse): Promise<T> {
  const payload = await response.json();
  expect(response.ok(), `${response.status()} ${payload.message}`).toBeTruthy();
  expect(payload.code, payload.message).toBe(200);
  return payload.data as T;
}

// Real local business fixtures; persisted-step and log transport fixtures are
// deliberately injected for UI states. External VM/Pod acceptance is separate.
for (const mode of ['light', 'dark'] as const) {
  test(`deployment workbench sources, logs, keyboard and mobile (${mode})`, async ({ page }, testInfo) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
    await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
    const headers = apiRequestHeaders(login);
    const token = `workbench-${mode}-${Date.now()}`;
    let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
    let originalError: unknown;
    try {
      scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
        data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
      hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
        data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
      packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
        data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
      const created = await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
        data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
      taskId = created.id;
      await success(await startReviewedTask(page.request, taskId, { headers, data: {} }));
      const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
      expect(task.hosts).toHaveLength(1);
      await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
        data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
      await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
        const response = await route.fetch(); const body = await response.json();
        body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
        body.data.stepHistory = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-old-install', stepOrder: 1, status: 'failed', attemptNo: 1, stateVersion: 3, capturedAt: '2026-11-07T00:00:00Z' }];
        body.data.logAttempts = [{taskHostId: task.hosts[0].id,attemptNo:1},{taskHostId: task.hosts[0].id,attemptNo:2}];
        await route.fulfill({ response, json: body });
      });
      let denied = false;
      const logRequests: string[] = [];
      const fixtureLogs = [
        { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 1, stream: 'stderr', content: 'fixture-old-error', createdAt: '2026-11-07T00:00:00Z' },
        { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-first-error', createdAt: '2026-11-07T00:00:01Z' },
        ...Array.from({ length: 1200 }, (_, i) => ({ sequence: i + 3, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stdout', content: `fixture-output-${i}`, createdAt: '2026-11-07T00:00:02Z' })),
        { sequence: 1203, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:03Z' },
      ];
      await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
        const params = new URL(route.request().url()).searchParams;
        logRequests.push(params.toString());
        const firstError = params.get('firstError') === '1';
        const items = fixtureLogs.filter((row) => row.sequence > Number(params.get('after') ?? 0)
          && (!params.has('hostId') || row.taskHostId === Number(params.get('hostId')))
          && (!params.has('stepId') || row.taskStepId === Number(params.get('stepId')))
          && (!params.has('attemptNo') || row.attemptNo === Number(params.get('attemptNo')))
          && (!firstError || row.stream === 'stderr')).slice(0, firstError ? 1 : 500);
        await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' }
          : { code: 200, data: { total: items.length, items } } });
      });
      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto(`/business/deploy/task/${taskId}`);
      const workbench = page.locator('.deploy-execution-workbench');
      await expect(workbench).toBeVisible({ timeout: 30_000 });
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
      await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
      await expect(workbench.getByText('fixture-first-error', { exact: true })).toHaveCount(0);
      await workbench.locator('.arco-select').nth(1).click();
      await page.getByText('#1', { exact: true }).last().click();
      await expect(page).toHaveURL(/attempt=1/);
      await expect(workbench.getByText('fixture-old-error', { exact: true })).toBeVisible();
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
      await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
      await expect(workbench.getByText(/最新记录时间（UTC）/)).toBeVisible();
      await expect(workbench.getByText('fixture-old-install', {exact:true})).toBeVisible();
      await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-desktop-history.png`),fullPage:true});
      await page.setViewportSize({width:390,height:844});
      await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
      await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
      // The app shell can hide overflow; document width alone cannot detect clipped children.
      await expect.poll(() => workbench.evaluate((element) => Math.max(...Array.from(element.children).map((child) => child.getBoundingClientRect().right - element.getBoundingClientRect().right)))).toBeLessThanOrEqual(1);
      await expect.poll(() => workbench.getByText(/最新记录时间（UTC）/).evaluate((element) => element.scrollWidth - element.clientWidth)).toBeLessThanOrEqual(1);
      await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-mobile-history.png`),fullPage:true});
      await page.setViewportSize({width:1440,height:900});

      await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
      await expect(page).toHaveURL(/host=/);
      await expect(page).toHaveURL(/attempt=2/);
      await expect(page).toHaveURL(/firstError=1/);
      await expect(workbench.getByText('fixture-first-error', { exact: true })).toBeVisible();
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
      await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-first-error.png`), fullPage: true });
      await workbench.getByRole('button', { name: '查看完整步骤日志', exact: true }).click();
      await expect(page).not.toHaveURL(/firstError=/);
      await workbench.getByRole('button', { name: '定位步骤 1' }).click();
      await expect(page).toHaveURL(/step=901/);
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
      await expect(workbench.getByText('fixture-error', { exact: true })).toBeVisible();
      await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
      await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
      await workbench.locator('.task-log-viewer input').fill('fixture-error');
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
      await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
      await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
      await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
      await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
      const dialog = page.getByRole('dialog');
      await expect(dialog).toBeVisible();
      await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
      await page.keyboard.press('Escape');
      await expect(dialog).toBeHidden();
      denied = true;
      await page.reload();
      await expect(workbench).toBeVisible({ timeout: 30_000 });
      await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
      await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
      await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
      denied = false;
      await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
      await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
      expect(logRequests.some((query) => query.includes('firstError=1') && query.includes('after=0') && query.includes('stepId=901'))).toBe(true);
      await fs.writeFile(testInfo.outputPath('provider-source-requests.json'), JSON.stringify(logRequests, null, 2));
      await testInfo.attach('provider-source-requests', { contentType: 'application/json', body: JSON.stringify(logRequests) });
    } catch (error) {
      originalError = error;
      throw error;
    } finally {
      // Stop page loads and polling before deleting their owned API fixtures.
      await page.close();
      const cleanupErrors: string[] = [];
      try {
        const cleanupHeaders = await verifiedApiHeaders(page.request, login);
        for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
          if (!id) continue;
          try {
            await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
          } catch (error) {
            cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
          }
        }
      } catch (error) {
        cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
      }
      await fs.writeFile(testInfo.outputPath('fixture-cleanup.json'), JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }, null, 2));
      await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
      if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
    }
  });
}
