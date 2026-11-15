# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:20:3

# Error details

```
TypeError: Cannot set properties of null (setting 'steps')
```

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

```yaml
- complementary:
    - text: P
    - menu:
        - menuitem "工作台":
            - button "工作台"
        - button "CMDB"
        - menuitem "业务域":
            - button "业务域"
        - menuitem "服务目录":
            - button "服务目录"
        - button "安装部署"
        - button "Kubernetes"
        - button "可观测中心"
        - button "访问控制"
        - button "组织架构"
        - button "安全审计"
        - button "低代码平台"
        - button "平台配置"
        - menuitem "文件分发":
            - button "文件分发"
        - menuitem "ArgoCD 应用发布":
            - button "ArgoCD 应用发布"
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 任务详情
- button "平台功能台": 搜索菜单、页面、功能…
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A
- tablist "已打开页面":
    - tab "工作台"
    - tab "任务详情" [selected]
- main:
    - list:
        - listitem
        - listitem
    - list:
        - listitem
        - listitem
        - listitem
        - listitem
        - listitem
        - listitem
        - listitem
        - listitem
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  26  |     const token = `workbench-${mode}-${Date.now()}`;
  27  |     let scopeId = 0, hostId = 0, packageId = 0, taskId = 0;
  28  |     let originalError: unknown;
  29  |     try {
  30  |       scopeId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers,
  31  |         data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  32  |       hostId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers,
  33  |         data: { hostname: token, ip: `10.251.${Math.floor(Math.random() * 200) + 1}.88`, os: 'linux', status: 'assigned', businessScopeId: scopeId } }))).id;
  34  |       packageId = (await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers,
  35  |         data: { name: token, version: '1.0.0', installCommand: 'echo fixture', status: 'enabled' } }))).id;
  36  |       const created = await success<{ id: number }>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers,
  37  |         data: { name: token, packageId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual' } }));
  38  |       taskId = created.id;
  39  |       await success(await startReviewedTask(page.request, taskId, { headers, data: {} }));
  40  |       const task = await success<{ id: number; hosts: { id: number }[] }>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  41  |       expect(task.hosts).toHaveLength(1);
  42  |       await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${task.hosts[0].id}/result`, { headers,
  43  |         data: { status: 'failed', errorMessage: 'fixture failure', stderr: 'fixture failure' } }));
  44  |       await page.route(`**/business/deploy/tasks/${taskId}`, async (route) => {
  45  |         const response = await route.fetch(); const body = await response.json();
  46  |         body.data.steps = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-install', stepOrder: 1, status: 'failed', attemptNo: 2, stateVersion: 7 }];
  47  |         body.data.stepHistory = [{ id: 901, taskHostId: task.hosts[0].id, stepKey: 'fixture-old-install', stepOrder: 1, status: 'failed', attemptNo: 1, stateVersion: 3, capturedAt: '2026-11-07T00:00:00Z' }];
  48  |         body.data.logAttempts = [{taskHostId: task.hosts[0].id,attemptNo:1},{taskHostId: task.hosts[0].id,attemptNo:2}];
  49  |         await route.fulfill({ response, json: body });
  50  |       });
  51  |       let denied = false;
  52  |       const logRequests: string[] = [];
  53  |       const fixtureLogs = [
  54  |         { sequence: 1, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 1, stream: 'stderr', content: 'fixture-old-error', createdAt: '2026-11-07T00:00:00Z' },
  55  |         { sequence: 2, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-first-error', createdAt: '2026-11-07T00:00:01Z' },
  56  |         ...Array.from({ length: 1200 }, (_, i) => ({ sequence: i + 3, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stdout', content: `fixture-output-${i}`, createdAt: '2026-11-07T00:00:02Z' })),
  57  |         { sequence: 1203, taskHostId: task.hosts[0].id, taskStepId: 901, attemptNo: 2, stream: 'stderr', content: 'fixture-error', createdAt: '2026-11-07T00:00:03Z' },
  58  |       ];
  59  |       await page.route(`**/business/deploy/tasks/${taskId}/logs?*`, async (route) => {
  60  |         const params = new URL(route.request().url()).searchParams;
  61  |         logRequests.push(params.toString());
  62  |         const firstError = params.get('firstError') === '1';
  63  |         const items = fixtureLogs.filter((row) => row.sequence > Number(params.get('after') ?? 0)
  64  |           && (!params.has('hostId') || row.taskHostId === Number(params.get('hostId')))
  65  |           && (!params.has('stepId') || row.taskStepId === Number(params.get('stepId')))
  66  |           && (!params.has('attemptNo') || row.attemptNo === Number(params.get('attemptNo')))
  67  |           && (!firstError || row.stream === 'stderr')).slice(0, firstError ? 1 : 500);
  68  |         await route.fulfill({ status: denied ? 403 : 200, json: denied ? { code: 403, message: 'permission.denied' }
  69  |           : { code: 200, data: { total: items.length, items } } });
  70  |       });
  71  |       await page.setViewportSize({ width: 1440, height: 900 });
  72  |       await page.goto(`/business/deploy/task/${taskId}`);
  73  |       const workbench = page.locator('.deploy-execution-workbench');
  74  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  75  |       await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  76  |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toHaveCount(0);
  77  |       await workbench.locator('.arco-select').nth(1).click();
  78  |       await page.getByText('#1', { exact: true }).last().click();
  79  |       await expect(page).toHaveURL(/attempt=1/);
  80  |       await expect(workbench.getByText('fixture-old-error', { exact: true })).toBeVisible();
  81  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  82  |       await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
  83  |       await expect(workbench.getByText(/最新记录时间（UTC）/)).toBeVisible();
  84  |       await expect(workbench.getByText('fixture-old-install', {exact:true})).toBeVisible();
  85  |       await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-desktop-history.png`),fullPage:true});
  86  |       await page.setViewportSize({width:390,height:844});
  87  |       await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  88  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  89  |       await expect(workbench.getByText('历史步骤检查点', {exact:true})).toBeVisible();
  90  |       // The app shell can hide overflow; document width alone cannot detect clipped children.
  91  |       await expect.poll(() => workbench.evaluate((element) => Math.max(...Array.from(element.children).map((child) => child.getBoundingClientRect().right - element.getBoundingClientRect().right)))).toBeLessThanOrEqual(1);
  92  |       await expect.poll(() => workbench.getByText(/最新记录时间（UTC）/).evaluate((element) => element.scrollWidth - element.clientWidth)).toBeLessThanOrEqual(1);
  93  |       await page.screenshot({path:testInfo.outputPath(`workbench-${mode}-mobile-history.png`),fullPage:true});
  94  |       await page.setViewportSize({width:1440,height:900});
  95  |
  96  |       await workbench.getByRole('button', { name: '定位首个失败目标' }).click();
  97  |       await expect(page).toHaveURL(/host=/);
  98  |       await expect(page).toHaveURL(/attempt=2/);
  99  |       await expect(page).toHaveURL(/firstError=1/);
  100 |       await expect(workbench.getByText('fixture-first-error', { exact: true })).toBeVisible();
  101 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  102 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-first-error.png`), fullPage: true });
  103 |       await workbench.getByRole('button', { name: '查看完整步骤日志', exact: true }).click();
  104 |       await expect(page).not.toHaveURL(/firstError=/);
  105 |       await workbench.getByRole('button', { name: '定位步骤 1' }).click();
  106 |       await expect(page).toHaveURL(/step=901/);
  107 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  108 |       await expect(workbench.getByText('fixture-error', { exact: true })).toBeVisible();
  109 |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  110 |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  111 |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
  112 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
  113 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  114 |       await page.setViewportSize({ width: 390, height: 844 });
  115 |     await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  116 |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  117 |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  118 |       const dialog = page.getByRole('dialog');
  119 |       await expect(dialog).toBeVisible();
  120 |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  121 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  122 |       await page.keyboard.press('Escape');
  123 |       await expect(dialog).toBeHidden();
  124 |       denied = true;
  125 |       await page.reload();
> 126 |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
      |                                                                         ^ Error: expect(locator).toBeVisible() failed
  127 |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  128 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  129 |       denied = false;
  130 |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  131 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  132 |       expect(logRequests.some((query) => query.includes('firstError=1') && query.includes('after=0') && query.includes('stepId=901'))).toBe(true);
  133 |       await fs.writeFile(testInfo.outputPath('provider-source-requests.json'), JSON.stringify(logRequests, null, 2));
  134 |       await testInfo.attach('provider-source-requests', { contentType: 'application/json', body: JSON.stringify(logRequests) });
  135 |     } catch (error) {
  136 |       originalError = error;
  137 |       throw error;
  138 |     } finally {
  139 |       const cleanupErrors: string[] = [];
  140 |       try {
  141 |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  142 |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  143 |           if (!id) continue;
  144 |           try {
  145 |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  146 |           } catch (error) {
  147 |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  148 |           }
  149 |         }
  150 |       } catch (error) {
  151 |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  152 |       }
  153 |       await fs.writeFile(testInfo.outputPath('fixture-cleanup.json'), JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }, null, 2));
  154 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  155 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  156 |     }
  157 |   });
  158 | }
  159 |
```
