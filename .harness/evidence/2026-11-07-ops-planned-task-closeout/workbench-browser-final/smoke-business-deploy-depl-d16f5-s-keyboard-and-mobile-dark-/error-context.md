# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:14:3

# Error details

```
Error: expect(locator).toBeHidden() failed

Locator:  getByRole('dialog')
Expected: hidden
Received: visible
Timeout:  10000ms

Call log:
  - Expect "toBeHidden" getByRole('dialog') with timeout 10000ms
  - waiting for getByRole('dialog')
    3 × locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-5" class="arco-modal app-dialog business-log-dialog zoomModal-appear zoomModal-appear-active">…</div>
      - unexpected value "visible"
    20 × locator resolved to <div role="dialog" aria-modal="true" aria-labelledby="arco-dialog-5" class="arco-modal app-dialog business-log-dialog zoomModal-appear-done zoomModal-enter-done">…</div>
       - unexpected value "visible"

```

```yaml
- 'dialog "workbench-dark-1794044420035 / workbench-dark-1794044420035 / fixture-install / #2"':
    - text: "workbench-dark-1794044420035 / workbench-dark-1794044420035 / fixture-install / #2"
    - combobox "日志流"
    - button "退出展开" [pressed]
    - 'region "workbench-dark-1794044420035 / workbench-dark-1794044420035 / fixture-install / #2"'
    - button "Close"
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
  30  |       const created = await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  31  |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  32  |       taskId = created.id;
  33  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} }));
  34  |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  35  |       expect(task.hosts).toHaveLength(1);
  36  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
  37  |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  38  |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  39  |         const response = await route.fetch(); const body = await response.json();
  40  |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  41  |         await route.fulfill({ response, json: body });
  42  |       });
  43  |       let denied = false;
  44  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  45  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' } : {
  46  |           code: 200, data: { total: 2, items: [
  47  |             { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stdout', content: 'fixture-output', createdAt: '2026-11-07T00:00:00Z' },
  48  |             { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:01Z' },
  49  |           ] } } });
  50  |       });
  51  |       await page.setViewportSize({ width: 1440, height: 900 });
  52  |       await page.goto(`/business/deploy/task/${taskId}`);
  53  |       const workbench = page.locator('.deploy-execution-workbench');
  54  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  55  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  56  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  57  |       await expect(page).toHaveURL(/host=/);
  58  |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  59  |       await expect(page).toHaveURL(/step=901/);
  60  |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  61  |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  62  |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  63  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  64  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  65  |       await page.setViewportSize({ width: 390, height: 844 });
  66  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  67  |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  68  |       const dialog = page.getByRole('dialog');
  69  |       await expect(dialog).toBeVisible();
  70  |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  71  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  72  |       await page.keyboard.press('Escape');
> 73  |       await expect(dialog).toBeHidden();
      |                            ^ Error: expect(locator).toBeHidden() failed
  74  |       denied = true;
  75  |       await page.reload();
  76  |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  77  |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  78  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  79  |       denied = false;
  80  |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  81  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(2);
  82  |     } catch (error) {
  83  |       originalError = error;
  84  |       throw error;
  85  |     } finally {
  86  |       const cleanupErrors: string[] = [];
  87  |       try {
  88  |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  89  |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  90  |           if (!id) continue;
  91  |           try {
  92  |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  93  |           } catch (error) {
  94  |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  95  |           }
  96  |         }
  97  |       } catch (error) {
  98  |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  99  |       }
  100 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  101 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  102 |     }
  103 |   });
  104 | }
  105 |
```
