# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (light)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:14:3

# Error details

```
Error: expect(received).toBeTruthy()

Received: false
```

# Test source

```ts
  1  | import { expect, test, type APIResponse } from '@playwright/test';
  2  | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale } from '../../helpers/auth';
  3  |
  4  | async function success<T>(response: APIResponse): Promise<T> {
> 5  |   expect(response.ok()).toBeTruthy();
     |                         ^ Error: expect(received).toBeTruthy()
  6  |   const payload = await response.json();
  7  |   expect(payload.code, payload.message).toBe(200);
  8  |   return payload.data as T;
  9  | }
  10 |
  11 | // Real local business fixtures; persisted-step and log transport fixtures are
  12 | // deliberately injected for UI states. External VM/Pod acceptance is separate.
  13 | for (const mode of ['light', 'dark'] as const) {
  14 |   test(`deployment workbench sources, logs, keyboard and mobile (${mode})`, async ({ page }, testInfo) => {
  15 |     const login = await loginByApi(page.request, adminCredentials);
  16 |     await installClientSession(page, login);
  17 |     await primeChineseLocale(page);
  18 |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  19 |     const headers = apiRequestHeaders(login);
  20 |     const token = `workbench-${mode}-${Date.now()}`;
  21 |     let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  22 |     try {
  23 |       scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
  24 |         data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  25 |       hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
  26 |         data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
  27 |       packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
  28 |         data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
  29 |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  30 |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  31 |       taskId = task.id;
  32 |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} }));
  33 |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
  34 |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  35 |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  36 |         const response = await route.fetch(); const body = await response.json();
  37 |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  38 |         await route.fulfill({ response, json: body });
  39 |       });
  40 |       let denied = false;
  41 |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  42 |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' } : {
  43 |           code: 200, data: { total: 2, items: [
  44 |             { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stdout', content: 'fixture-output', createdAt: '2026-11-07T00:00:00Z' },
  45 |             { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:01Z' },
  46 |           ] } } });
  47 |       });
  48 |       await page.setViewportSize({ width: 1440, height: 900 });
  49 |       await page.goto(`/business/deploy/task/${taskId}`);
  50 |       const workbench = page.locator('.deploy-execution-workbench');
  51 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  52 |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  53 |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  54 |       await expect(page).toHaveURL(/host=/);
  55 |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  56 |       await expect(page).toHaveURL(/step=901/);
  57 |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  58 |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  59 |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  60 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  61 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  62 |       await page.setViewportSize({ width: 390, height: 844 });
  63 |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  64 |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  65 |       const dialog = page.getByRole('dialog');
  66 |       await expect(dialog).toBeVisible();
  67 |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  68 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  69 |       await page.keyboard.press('Escape');
  70 |       await expect(dialog).toBeHidden();
  71 |       denied = true;
  72 |       await page.reload();
  73 |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  74 |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  75 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  76 |       denied = false;
  77 |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  78 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  79 |     } finally {
  80 |       for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  81 |         if (id) await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers }));
  82 |       }
  83 |     }
  84 |   });
  85 | }
  86 |
```
