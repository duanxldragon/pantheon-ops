# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (light)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:16:3

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.deploy-execution-workbench').locator('.task-log-viewer__row')
Expected: 500
Received: 0
Timeout:  10000ms

Call log:
  - Expect "toHaveCount" locator('.deploy-execution-workbench').locator('.task-log-viewer__row') with timeout 10000ms
  - waiting for locator('.deploy-execution-workbench').locator('.task-log-viewer__row')
    23 × locator resolved to 0 elements
       - unexpected value "0"

```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]:
                - generic [ref=e7]: P
                - generic [ref=e8]: Pantheon Base
            - menu [ref=e12]
    - generic [ref=e13]:
        - generic [ref=e14]:
            - generic [ref=e15]:
                - button "切换导航" [ref=e16] [cursor=pointer]
                - list [ref=e20]:
                    - listitem [ref=e21]: 首页
                    - listitem [ref=e25]: 任务详情
            - generic [ref=e26]:
                - button "平台功能台" [ref=e28]:
                    - generic [ref=e31]: 搜索菜单、页面、功能…
                    - generic [ref=e32]: Ctrl K
                - button "平台偏好" [ref=e34] [cursor=pointer]
                - 'button "用户: 用户" [ref=e39] [cursor=pointer]':
                    - generic [ref=e40]: 用
                    - generic [ref=e42]: 用户
        - tablist "已打开页面" [ref=e44]
        - main [ref=e45]:
            - generic [ref=e47]:
                - generic [ref=e52]: 加载失败
                - generic [ref=e53]: 当前区块暂时无法返回数据，请重试或稍后再查看。
                - button "重试" [ref=e55] [cursor=pointer]
        - generic [ref=e56]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1   | import { startReviewedTask } from './reviewed-start';
  2   | import fs from 'node:fs/promises';
  3   | import { expect, test, type APIResponse } from '@playwright/test';
  4   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
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
  42  |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  43  |         body.data.stepHistory = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-old-install', stepOrder: 1, status: 'failed', attemptNo: 1, stateVersion: 3, capturedAt: '2026-11-07T00:00:00Z' }];
  44  |         body.data.logAttempts = [{taskHostId: task.hosts[0].id,attemptNo:1},{taskHostId: task.hosts[0].id,attemptNo:2}];
  45  |         await route.fulfill({ response, json: body });
  46  |       });
  47  |       let denied = false;
  48  |       const logRequests: string[] = [];
  49  |       const fixtureLogs = [
  50  |         { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 1, stream: 'stderr', content: 'fixture-old-error', createdAt: '2026-11-07T00:00:00Z' },
  51  |         { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-first-error', createdAt: '2026-11-07T00:00:01Z' },
  52  |         ...Array.from({ length: 1200 }, (_, i) => ({ sequence: i + 3, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stdout', content: `fixture-output-${i}`, createdAt: '2026-11-07T00:00:02Z' })),
  53  |         { sequence: 1203, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:03Z' },
  54  |       ];
  55  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  56  |         const params = new URL(route.request().url()).searchParams;
  57  |         logRequests.push(params.toString());
  58  |         const firstError = params.get('firstError') === '1';
  59  |         const items = fixtureLogs.filter((row) => row.sequence > Number(params.get('after') ?? 0)
  60  |           && (!params.has('hostId') || row.taskHostId === Number(params.get('hostId')))
  61  |           && (!params.has('stepId') || row.taskStepId === Number(params.get('stepId')))
  62  |           && (!params.has('attemptNo') || row.attemptNo === Number(params.get('attemptNo')))
  63  |           && (!firstError || row.stream === 'stderr')).slice(0, firstError ? 1 : 500);
  64  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' }
  65  |           : { code: 200, data: { total: items.length, items } } });
  66  |       });
  67  |       await page.setViewportSize({ width: 1440, height: 900 });
  68  |       await page.goto(`/business/deploy/task/${taskId}`);
  69  |       const workbench = page.locator('.deploy-execution-workbench');
> 70  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
      |                                                                ^ Error: expect(locator).toHaveCount(expected) failed
  71  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  72  |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toHaveCount(0);
  73  |       await workbench.locator('.arco-select').nth(1).click();
  74  |       await page.getByText('#1', { exact: true }).last().click();
  75  |       await expect(page).toHaveURL(/attempt=1/);
  76  |       await expect(workbench.getByText('fixture-old-error', { exact: true })).toBeVisible();
  77  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  78  |       await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
  79  |       await expect(workbench.getByText(/最新记录时间（UTC）/)).toBeVisible();
  80  |       await expect(workbench.getByText('fixture-old-install', {exact:true})).toBeVisible();
  81  |       await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-desktop-history.png`),fullPage:true});
  82  |       await page.setViewportSize({width:390,height:844});
  83  |       await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  84  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  85  |       await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
  86  |       // The app shell can hide overflow; document width alone cannot detect clipped children.
  87  |       await expect.poll(() => workbench.evaluate((element) => Math.max(...Array.from(element.children).map((child) => child.getBoundingClientRect().right - element.getBoundingClientRect().right)))).toBeLessThanOrEqual(1);
  88  |       await expect.poll(() => workbench.getByText(/最新记录时间（UTC）/).evaluate((element) => element.scrollWidth - element.clientWidth)).toBeLessThanOrEqual(1);
  89  |       await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-mobile-history.png`),fullPage:true});
  90  |       await page.setViewportSize({width:1440,height:900});
  91  |
  92  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  93  |       await expect(page).toHaveURL(/host=/);
  94  |       await expect(page).toHaveURL(/attempt=2/);
  95  |       await expect(page).toHaveURL(/firstError=1/);
  96  |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toBeVisible();
  97  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  98  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-first-error.png`), fullPage: true });
  99  |       await workbench.getByRole('button', { name: '查看完整步骤日志', exact: true }).click();
  100 |       await expect(page).not.toHaveURL(/firstError=/);
  101 |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  102 |       await expect(page).toHaveURL(/step=901/);
  103 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  104 |       await expect(workbench.getByText('fixture-error', { exact: true })).toBeVisible();
  105 |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  106 |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  107 |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  108 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  109 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  110 |       await page.setViewportSize({ width: 390, height: 844 });
  111 |     await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  112 |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  113 |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  114 |       const dialog = page.getByRole('dialog');
  115 |       await expect(dialog).toBeVisible();
  116 |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  117 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  118 |       await page.keyboard.press('Escape');
  119 |       await expect(dialog).toBeHidden();
  120 |       denied = true;
  121 |       await page.reload();
  122 |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  123 |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  124 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  125 |       denied = false;
  126 |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  127 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  128 |       expect(logRequests.some((query) => query.includes('firstError=1') && query.includes('after=0') && query.includes('stepId=901'))).toBe(true);
  129 |       await fs.writeFile(testInfo.outputPath('provider-source-requests.json'), JSON.stringify(logRequests, null, 2));
  130 |       await testInfo.attach('provider-source-requests', { contentType: 'application/json', body: JSON.stringify(logRequests) });
  131 |     } catch (error) {
  132 |       originalError = error;
  133 |       throw error;
  134 |     } finally {
  135 |       const cleanupErrors: string[] = [];
  136 |       try {
  137 |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  138 |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  139 |           if (!id) continue;
  140 |           try {
  141 |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  142 |           } catch (error) {
  143 |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  144 |           }
  145 |         }
  146 |       } catch (error) {
  147 |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  148 |       }
  149 |       await fs.writeFile(testInfo.outputPath('fixture-cleanup.json'), JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }, null, 2));
  150 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  151 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  152 |     }
  153 |   });
  154 | }
  155 |
```
