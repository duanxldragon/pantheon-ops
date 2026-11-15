# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:16:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('没有日志查看权限', { exact: true }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('没有日志查看权限', { exact: true }).first() with timeout 10000ms
  - waiting for getByText('没有日志查看权限', { exact: true }).first()

```

# Test source

```ts
  5   |
  6   | async function success<T>(response: APIResponse): Promise<T> {
  7   |   const payload = await response.json();
  8   |   expect(response.ok(), `${response.status()} ${payload.message}`).toBeTruthy();
  9   |   expect(payload.code, payload.message).toBe(200);
  10  |   return payload.data as T;
  11  | }
  12  |
  13  | // Real local business fixtures; persisted-step and log transport fixtures are
  14  | // deliberately injected for UI states. External VM/Pod acceptance is separate.
  15  | for (const mode of ['light', 'dark'] as const) {
  16  |   test(`deployment workbench sources, logs, keyboard and mobile (${mode})`, async ({ page }, testInfo) => {
  17  |     const login = await loginByApi(page.request, adminCredentials);
  18  |     await installClientSession(page, login);
  19  |     await primeChineseLocale(page);
  20  |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  21  |     const headers = apiRequestHeaders(login);
  22  |     const token = `workbench-${mode}-${Date.now()}`;
  23  |     let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  24  |     let originalError: unknown;
  25  |     try {
  26  |       scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
  27  |         data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  28  |       hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
  29  |         data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
  30  |       packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
  31  |         data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
  32  |       const created = await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  33  |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  34  |       taskId = created.id;
  35  |       await success(await startReviewedTask(page.request, taskId, { headers, data: {} }));
  36  |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  37  |       expect(task.hosts).toHaveLength(1);
  38  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
  39  |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  40  |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  41  |         const response = await route.fetch(); const body = await response.json();
  42  |         body.data.steps = [{ id: 900, taskHostId: task.hosts[0].id, stepKey: 'fixture-old-install', stepOrder: 1, status: 'failed', attemptNo: 1, stateVersion: 1 }, { id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  43  |         await route.fulfill({ response, json: body });
  44  |       });
  45  |       let denied = false;
  46  |       const logRequests: string[] = [];
  47  |       const fixtureLogs = [
  48  |         { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 900, attemptNo: 1, stream: 'stderr', content: 'fixture-old-error', createdAt: '2026-11-07T00:00:00Z' },
  49  |         { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-first-error', createdAt: '2026-11-07T00:00:01Z' },
  50  |         ...Array.from({ length: 1200 }, (_, i) => ({ sequence: i + 3, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stdout', content: `fixture-output-${i}`, createdAt: '2026-11-07T00:00:02Z' })),
  51  |         { sequence: 1203, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:03Z' },
  52  |       ];
  53  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  54  |         const params = new URL(route.request().url()).searchParams;
  55  |         logRequests.push(params.toString());
  56  |         const firstError = params.get('firstError') === '1';
  57  |         const items = fixtureLogs.filter((row) => row.sequence > Number(params.get('after') ?? 0)
  58  |           && (!params.has('hostId') || row.taskHostId === Number(params.get('hostId')))
  59  |           && (!params.has('stepId') || row.taskStepId === Number(params.get('stepId')))
  60  |           && (!params.has('attemptNo') || (row.taskStepId === 900 ? 1 : 2) === Number(params.get('attemptNo')))
  61  |           && (!firstError || row.stream === 'stderr')).slice(0, firstError ? 1 : 500);
  62  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' }
  63  |           : { code: 200, data: { total: items.length, items } } });
  64  |       });
  65  |       await page.setViewportSize({ width: 1440, height: 900 });
  66  |       await page.goto(`/business/deploy/task/${taskId}`);
  67  |       const workbench = page.locator('.deploy-execution-workbench');
  68  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  69  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  70  |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toHaveCount(0);
  71  |       await workbench.locator('.arco-select').nth(1).click();
  72  |       await page.getByText('#1', { exact: true }).last().click();
  73  |       await expect(page).toHaveURL(/attempt=1/);
  74  |       await expect(workbench.getByText('fixture-old-error', { exact: true })).toBeVisible();
  75  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  76  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  77  |       await expect(page).toHaveURL(/host=/);
  78  |       await expect(page).toHaveURL(/attempt=2/);
  79  |       await expect(page).toHaveURL(/firstError=1/);
  80  |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toBeVisible();
  81  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  82  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-first-error.png`), fullPage: true });
  83  |       await workbench.getByRole('button', { name: '查看完整步骤日志', exact: true }).click();
  84  |       await expect(page).not.toHaveURL(/firstError=/);
  85  |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  86  |       await expect(page).toHaveURL(/step=901/);
  87  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  88  |       await expect(workbench.getByText('fixture-error', { exact: true })).toBeVisible();
  89  |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  90  |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  91  |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  92  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  93  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  94  |       await page.setViewportSize({ width: 390, height: 844 });
  95  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  96  |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  97  |       const dialog = page.getByRole('dialog');
  98  |       await expect(dialog).toBeVisible();
  99  |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  100 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  101 |       await page.keyboard.press('Escape');
  102 |       await expect(dialog).toBeHidden();
  103 |       denied = true;
  104 |       await page.reload();
> 105 |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
      |                                                                         ^ Error: expect(locator).toBeVisible() failed
  106 |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  107 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  108 |       denied = false;
  109 |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  110 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  111 |       expect(logRequests.some((query) => query.includes('firstError=1') && query.includes('after=0') && query.includes('stepId=901'))).toBe(true);
  112 |       await fs.writeFile(testInfo.outputPath('provider-source-requests.json'), JSON.stringify(logRequests, null, 2));
  113 |       await testInfo.attach('provider-source-requests', { contentType: 'application/json', body: JSON.stringify(logRequests) });
  114 |     } catch (error) {
  115 |       originalError = error;
  116 |       throw error;
  117 |     } finally {
  118 |       const cleanupErrors: string[] = [];
  119 |       try {
  120 |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  121 |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  122 |           if (!id) continue;
  123 |           try {
  124 |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  125 |           } catch (error) {
  126 |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  127 |           }
  128 |         }
  129 |       } catch (error) {
  130 |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  131 |       }
  132 |       await fs.writeFile(testInfo.outputPath('fixture-cleanup.json'), JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }, null, 2));
  133 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  134 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  135 |     }
  136 |   });
  137 | }
  138 |
```
