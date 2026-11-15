# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (light)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:14:3

# Error details

```
TypeError: Cannot read properties of undefined (reading 'id')
```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   |
  4   | async function success<T>(response: APIResponse): Promise<T> {
  5   |   const payload = await response.json();
  6   |   expect(response.ok(), `${response.status()} ${payload.message}`).toBeTruthy();
  7   |   expect(payload.code, payload.message).toBe(200);
  8   |   return payload.data as T;
  9   | }
  10  |
  11  | // Real local business fixtures; persisted-step and log transport fixtures are
  12  | // deliberately injected for UI states. External VM/Pod acceptance is separate.
  13  | for (const mode of ['light', 'dark'] as const) {
  14  |   test(`deployment workbench sources, logs, keyboard and mobile (${mode})`, async ({ page }, testInfo) => {
  15  |     const login = await loginByApi(page.request, adminCredentials);
  16  |     await installClientSession(page, login);
  17  |     await primeChineseLocale(page);
  18  |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  19  |     const headers = apiRequestHeaders(login);
  20  |     const token = `workbench-${mode}-${Date.now()}`;
  21  |     let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  22  |     let originalError: unknown;
  23  |     try {
  24  |       scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
  25  |         data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  26  |       hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
  27  |         data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
  28  |       packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
  29  |         data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
  30  |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  31  |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  32  |       taskId = task.id;
  33  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} }));
> 34  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
      |                                                                                                       ^ TypeError: Cannot read properties of undefined (reading 'id')
  35  |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  36  |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  37  |         const response = await route.fetch(); const body = await response.json();
  38  |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  39  |         await route.fulfill({ response, json: body });
  40  |       });
  41  |       let denied = false;
  42  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  43  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' } : {
  44  |           code: 200, data: { total: 2, items: [
  45  |             { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stdout', content: 'fixture-output', createdAt: '2026-11-07T00:00:00Z' },
  46  |             { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:01Z' },
  47  |           ] } } });
  48  |       });
  49  |       await page.setViewportSize({ width: 1440, height: 900 });
  50  |       await page.goto(`/business/deploy/task/${taskId}`);
  51  |       const workbench = page.locator('.deploy-execution-workbench');
  52  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  53  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  54  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  55  |       await expect(page).toHaveURL(/host=/);
  56  |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  57  |       await expect(page).toHaveURL(/step=901/);
  58  |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  59  |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  60  |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  61  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  62  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  63  |       await page.setViewportSize({ width: 390, height: 844 });
  64  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  65  |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  66  |       const dialog = page.getByRole('dialog');
  67  |       await expect(dialog).toBeVisible();
  68  |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  69  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  70  |       await page.keyboard.press('Escape');
  71  |       await expect(dialog).toBeHidden();
  72  |       denied = true;
  73  |       await page.reload();
  74  |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  75  |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  76  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  77  |       denied = false;
  78  |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  79  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  80  |     } catch (error) {
  81  |       originalError = error;
  82  |       throw error;
  83  |     } finally {
  84  |       const cleanupErrors: string[] = [];
  85  |       try {
  86  |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  87  |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  88  |           if (!id) continue;
  89  |           try {
  90  |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  91  |           } catch (error) {
  92  |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  93  |           }
  94  |         }
  95  |       } catch (error) {
  96  |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  97  |       }
  98  |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  99  |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  100 |     }
  101 |   });
  102 | }
  103 |
```
