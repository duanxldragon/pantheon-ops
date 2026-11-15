# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:15:3

# Error details

```
ReferenceError: request is not defined
```

# Test source

```ts
  1   | import { startReviewedTask } from './reviewed-start';
  2   | import { expect, test, type APIResponse } from '@playwright/test';
  3   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  4   |
  5   | async function success<T>(response: APIResponse): Promise<T> {
  6   |   const payload = await response.json();
  7   |   expect(response.ok(), `${response.status()} ${payload.message}`).toBeTruthy();
  8   |   expect(payload.code, payload.message).toBe(200);
  9   |   return payload.data as T;
  10  | }
  11  |
  12  | // Real local business fixtures; persisted-step and log transport fixtures are
  13  | // deliberately injected for UI states. External VM/Pod acceptance is separate.
  14  | for (const mode of ['light', 'dark'] as const) {
  15  |   test(`deployment workbench sources, logs, keyboard and mobile (${mode})`, async ({ page }, testInfo) => {
  16  |     const login = await loginByApi(page.request, adminCredentials);
  17  |     await installClientSession(page, login);
  18  |     await primeChineseLocale(page);
  19  |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  20  |     const headers = apiRequestHeaders(login);
  21  |     const token = `workbench-${mode}-${Date.now()}`;
  22  |     let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  23  |     let originalError: unknown;
  24  |     try {
  25  |       scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
  26  |         data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  27  |       hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
  28  |         data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
  29  |       packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
  30  |         data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
  31  |       const created = await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  32  |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  33  |       taskId = created.id;
> 34  |       await success(await page.startReviewedTask(request, taskId, { headers, data: {} }));
      |                                                  ^ ReferenceError: request is not defined
  35  |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  36  |       expect(task.hosts).toHaveLength(1);
  37  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
  38  |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  39  |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  40  |         const response = await route.fetch(); const body = await response.json();
  41  |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  42  |         await route.fulfill({ response, json: body });
  43  |       });
  44  |       let denied = false;
  45  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  46  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' } : {
  47  |           code: 200, data: { total: 2, items: [
  48  |             { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stdout', content: 'fixture-output', createdAt: '2026-11-07T00:00:00Z' },
  49  |             { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:01Z' },
  50  |           ] } } });
  51  |       });
  52  |       await page.setViewportSize({ width: 1440, height: 900 });
  53  |       await page.goto(`/business/deploy/task/${taskId}`);
  54  |       const workbench = page.locator('.deploy-execution-workbench');
  55  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  56  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  57  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  58  |       await expect(page).toHaveURL(/host=/);
  59  |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  60  |       await expect(page).toHaveURL(/step=901/);
  61  |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  62  |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  63  |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  64  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  65  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  66  |       await page.setViewportSize({ width: 390, height: 844 });
  67  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  68  |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  69  |       const dialog = page.getByRole('dialog');
  70  |       await expect(dialog).toBeVisible();
  71  |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  72  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  73  |       await page.keyboard.press('Escape');
  74  |       await expect(dialog).toBeHidden();
  75  |       denied = true;
  76  |       await page.reload();
  77  |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  78  |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  79  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  80  |       denied = false;
  81  |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  82  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  83  |     } catch (error) {
  84  |       originalError = error;
  85  |       throw error;
  86  |     } finally {
  87  |       const cleanupErrors: string[] = [];
  88  |       try {
  89  |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  90  |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  91  |           if (!id) continue;
  92  |           try {
  93  |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  94  |           } catch (error) {
  95  |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  96  |           }
  97  |         }
  98  |       } catch (error) {
  99  |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  100 |       }
  101 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  102 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  103 |     }
  104 |   });
  105 | }
  106 |
```
