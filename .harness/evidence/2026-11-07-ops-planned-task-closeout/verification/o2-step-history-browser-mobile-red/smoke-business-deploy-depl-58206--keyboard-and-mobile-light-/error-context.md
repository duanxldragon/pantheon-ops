# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (light)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:16:3

# Error details

```
Error: expect(received).toBeLessThanOrEqual(expected)

Expected: <= 1
Received:    473.828125

Call Log:
- Timeout 10000ms exceeded while waiting on the predicate
```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]: P
            - menu [ref=e10]:
                - generic [ref=e11]:
                    - menuitem [ref=e12] [cursor=pointer]:
                        - button "工作台" [ref=e13]
                    - button "CMDB" [ref=e23] [cursor=pointer]
                    - menuitem [ref=e35] [cursor=pointer]:
                        - button "业务域" [ref=e36]
                    - menuitem [ref=e45] [cursor=pointer]:
                        - button "服务目录" [ref=e46]
                    - button "安装部署" [ref=e58] [cursor=pointer]
                    - button "Kubernetes" [ref=e70] [cursor=pointer]
                    - button "可观测中心" [ref=e82] [cursor=pointer]
                    - button "访问控制" [ref=e94] [cursor=pointer]
                    - button "组织架构" [ref=e108] [cursor=pointer]
                    - button "安全审计" [ref=e123] [cursor=pointer]
                    - button "低代码平台" [ref=e135] [cursor=pointer]
                    - button "平台配置" [ref=e147] [cursor=pointer]
                    - menuitem [ref=e156] [cursor=pointer]:
                        - button "文件分发" [ref=e157]
                    - menuitem [ref=e166] [cursor=pointer]:
                        - button "ArgoCD" [ref=e167]
    - generic [ref=e173]:
        - generic [ref=e174]:
            - generic [ref=e175]:
                - button "切换导航" [ref=e176] [cursor=pointer]
                - list [ref=e180]:
                    - listitem [ref=e181]: 首页
                    - listitem [ref=e185]: 任务详情
            - generic [ref=e186]:
                - button "平台功能台" [ref=e188]:
                    - generic [ref=e191]: 搜索菜单、页面、功能…
                - button "通知中心" [ref=e193] [cursor=pointer]
                - button "平台偏好" [ref=e197] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e202] [cursor=pointer]':
                    - generic [ref=e203]: A
        - tablist "已打开页面" [ref=e205]:
            - tab "工作台" [ref=e206] [cursor=pointer]
            - tab "任务详情" [selected] [ref=e210] [cursor=pointer]
        - main [ref=e212]:
            - generic [ref=e214]:
                - generic [ref=e215]:
                    - generic [ref=e216]:
                        - heading "workbench-light-1794107098315" [level=5] [ref=e218]
                        - generic [ref=e219]: 任务详情
                    - button "返回" [ref=e223] [cursor=pointer]
                - generic [ref=e226]:
                    - generic [ref=e229]:
                        - generic [ref=e230]:
                            - generic [ref=e231]:
                                - generic [ref=e232]: 运维平台 / 部署任务
                                - heading "workbench-light-1794107098315" [level=5] [ref=e233]
                                - generic [ref=e234]: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。
                            - generic [ref=e235]: 失败
                        - generic [ref=e237]:
                            - generic [ref=e238]:
                                - generic [ref=e239]: 目标主机
                                - generic [ref=e240]: "1"
                            - generic [ref=e241]:
                                - generic [ref=e242]: 成功
                                - generic [ref=e243]: "0"
                            - generic [ref=e244]:
                                - generic [ref=e245]: 失败
                                - generic [ref=e246]: "1"
                    - generic [ref=e250]:
                        - generic [ref=e251]: 任务状态流转
                        - list "任务状态流转" [ref=e253]:
                            - listitem [ref=e254]:
                                - generic [ref=e256]: 草稿
                            - listitem [ref=e257]:
                                - generic [ref=e259]: 待执行
                            - listitem [ref=e260]:
                                - generic [ref=e262]: 执行中
                            - listitem [ref=e263]:
                                - generic [ref=e265]: 成功
                            - listitem [ref=e266]:
                                - generic [ref=e268]: 失败
                            - listitem [ref=e269]:
                                - generic [ref=e271]: 已取消
                        - generic [ref=e272]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                    - generic [ref=e276]:
                        - generic [ref=e277]: 任务级操作
                        - generic [ref=e279]:
                            - generic [ref=e280]:
                                - generic [ref=e281]: 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。
                                - generic [ref=e282]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                            - generic [ref=e283]:
                                - button "重试失败步骤" [ref=e285] [cursor=pointer]
                                - button "删除" [ref=e289] [cursor=pointer]
                    - table [ref=e297]:
                        - rowgroup [ref=e298]:
                            - row [ref=e299]:
                                - cell "任务来源" [ref=e300]
                                - cell "软件组件" [ref=e301]
                                - cell "软件组件" [ref=e302]
                                - cell "workbench-light-1794107098315 1.0.0" [ref=e303]
                            - row [ref=e304]:
                                - cell "任务动作" [ref=e305]
                                - cell "安装" [ref=e306]
                                - cell "执行模式" [ref=e307]
                                - cell "固定任务" [ref=e308]
                            - row [ref=e309]:
                                - cell "模板参数" [ref=e310]
                                - cell "-" [ref=e311]
                                - cell "状态" [ref=e312]
                                - cell "失败" [ref=e313]
                            - row [ref=e314]:
                                - cell "目标来源" [ref=e315]
                                - cell "主机" [ref=e316]
                                - cell "业务域" [ref=e317]
                                - cell "workbench-light-1794107098315" [ref=e318]
                            - row [ref=e319]:
                                - cell "执行方式" [ref=e320]
                                - cell "手动执行" [ref=e321]
                                - cell "外部任务 ID" [ref=e322]
                                - cell "-" [ref=e323]
                            - row [ref=e324]:
                                - cell "开始时间" [ref=e325]
                                - cell "2026-11-08 11:04" [ref=e326]
                                - cell "结束时间" [ref=e327]
                                - cell "2026-11-08 11:04" [ref=e328]
                            - row [ref=e329]:
                                - cell "耗时" [ref=e330]
                                - cell "-" [ref=e331]
                                - cell "主机数" [ref=e332]
                                - cell "1" [ref=e333]
                            - row [ref=e334]:
                                - cell "成功数" [ref=e335]
                                - cell "0" [ref=e336]
                                - cell "失败数" [ref=e337]
                                - cell "1" [ref=e338]
                            - row [ref=e339]:
                                - cell "执行中" [ref=e340]
                                - cell "0" [ref=e341]
                                - cell "跳过数" [ref=e342]
                                - cell "0" [ref=e343]
                            - row [ref=e344]:
                                - cell "备注" [ref=e345]
                                - cell "-" [ref=e346]
                    - generic [ref=e350]:
                        - generic [ref=e353]:
                            - generic [ref=e354]: 主机执行明细
                            - generic [ref=e355]: 先查看各状态分布，再按主机展开过程日志和标准输出。
                        - generic [ref=e357]:
                            - generic [ref=e358]:
                                - generic [ref=e359]: 待执行
                                - generic [ref=e360]: "0"
                            - generic [ref=e361]:
                                - generic [ref=e362]: 执行中
                                - generic [ref=e363]: "0"
                            - generic [ref=e364]:
                                - generic [ref=e365]: 成功
                                - generic [ref=e366]: "0"
                            - generic [ref=e367]:
                                - generic [ref=e368]: 失败
                                - generic [ref=e369]: "1"
                            - generic [ref=e370]:
                                - generic [ref=e371]: 已跳过
                                - generic [ref=e372]: "0"
                        - generic [ref=e374]:
                            - generic [ref=e375]:
                                - generic [ref=e376]: 共 1 条
                                - generic [ref=e377]: 左右滑动查看更多字段
                            - generic [ref=e380]:
                                - table [ref=e384]:
                                    - rowgroup [ref=e394]:
                                        - row [ref=e395]:
                                            - columnheader "主机" [ref=e396]
                                            - columnheader "主机 IP" [ref=e398]
                                            - columnheader "状态" [ref=e400]
                                            - columnheader "开始时间" [ref=e402]
                                            - columnheader "结束时间" [ref=e404]
                                            - columnheader "耗时" [ref=e406]
                                            - columnheader "错误信息" [ref=e408]
                                            - columnheader "操作" [ref=e410]
                                    - rowgroup [ref=e412]:
                                        - row [ref=e413]:
                                            - cell "workbench-light-1794107098315" [ref=e414]
                                            - cell "10.251.58.88" [ref=e416]
                                            - cell "失败" [ref=e418]
                                            - cell "2026-11-08 11:04" [ref=e423]
                                            - cell "2026-11-08 11:04" [ref=e425]
                                            - cell "-" [ref=e427]
                                            - cell "fixture failure" [ref=e429]
                                            - cell [ref=e431]
                                - list [ref=e437]:
                                    - listitem "上一页" [ref=e438]: .
                                    - listitem "第 1 页" [ref=e441] [cursor=pointer]: "1"
                                    - listitem "下一页" [ref=e442]: .
                    - region "日志来源" [ref=e446]:
                        - generic [ref=e447]:
                            - combobox "执行目标" [ref=e449] [cursor=pointer]:
                                - generic "workbench-light-1794107098315 / 10.251.58.88" [ref=e450]:
                                    - generic [ref=e451]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e452]: workbench-light-1794107098315 / 10.251.58.88
                            - combobox "尝试次数" [ref=e458] [cursor=pointer]:
                                - generic "#1" [ref=e459]:
                                    - generic [ref=e460]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e461]: "#1"
                            - button "定位首个失败目标" [ref=e467] [cursor=pointer]
                            - button "重新连接" [ref=e469] [cursor=pointer]
                            - generic [ref=e470]: workbench-light-1794107098315 / 10.251.58.88
                        - status [ref=e473]:
                            - generic [ref=e474]: 历史步骤检查点
                            - paragraph [ref=e476]: 此视图展示各步骤在该尝试最后记录的状态；最新记录时间（UTC）：2026-11-07T00:00:00.000Z
                        - region "执行步骤" [ref=e477]:
                            - list [ref=e480]:
                                - listitem [ref=e481]:
                                    - button "定位步骤 1" [ref=e482]:
                                        - generic [ref=e484]: fixture-old-install
                                        - generic [ref=e485]:
                                            - generic [ref=e486]: failed
                                            - text: "尝试次数: 1"
                        - generic [ref=e488]:
                            - generic [ref=e489]:
                                - button "展开日志" [ref=e491] [cursor=pointer]
                                - combobox "日志流" [ref=e493] [cursor=pointer]:
                                    - generic [ref=e495]:
                                        - textbox "日志流"
                                        - generic [ref=e496]: 日志流
                            - 'region "workbench-light-1794107098315 / workbench-light-1794107098315 / #1" [ref=e501]':
                                - generic [ref=e502]:
                                    - generic [ref=e503]: "workbench-light-1794107098315 / workbench-light-1794107098315 / #1"
                                    - generic [ref=e504]:
                                        - textbox "搜索已保留日志" [ref=e508]
                                        - button "暂停追尾" [ref=e512] [cursor=pointer]
                                        - switch "自动换行" [checked] [ref=e513] [cursor=pointer]
                                - generic [ref=e515]:
                                    - generic [ref=e517]:
                                        - generic [ref=e518]: "#1"
                                        - generic [ref=e519]: stderr
                                        - generic [ref=e520]: fixture-old-error
                                    - generic [ref=e521]: 显示 1 条日志
                        - generic [ref=e526]:
                            - generic [ref=e528]:
                                - generic [ref=e529]:
                                    - generic [ref=e530]: workbench-light-1794107098315
                                    - generic [ref=e531]: 10.251.58.88 / 2026-11-08 11:04
                                - generic [ref=e533]: 失败
                            - generic [ref=e536]:
                                - generic [ref=e537]:
                                    - generic [ref=e538]: 共 3 条
                                    - generic [ref=e539]: 左右滑动查看更多字段
                                - table [ref=e546]:
                                    - rowgroup [ref=e553]:
                                        - row [ref=e554]:
                                            - columnheader "步骤" [ref=e555]
                                            - columnheader "步骤类型" [ref=e557]
                                            - columnheader "阶段" [ref=e559]
                                            - columnheader "开始时间" [ref=e561]
                                            - columnheader "过程信息" [ref=e563]
                                    - rowgroup [ref=e565]:
                                        - row [ref=e566]:
                                            - cell "- -" [ref=e567]:
                                                - generic [ref=e570]:
                                                    - generic [ref=e571]: "-"
                                                    - generic [ref=e572]: "-"
                                            - cell "-" [ref=e573]
                                            - cell "进入执行队列" [ref=e575]
                                            - cell "2026-11-08 11:04" [ref=e580]
                                            - cell "Task host queued" [ref=e582]
                                        - row [ref=e584]:
                                            - cell "- -" [ref=e585]:
                                                - generic [ref=e588]:
                                                    - generic [ref=e589]: "-"
                                                    - generic [ref=e590]: "-"
                                            - cell "-" [ref=e591]
                                            - cell "记录执行结果" [ref=e593]
                                            - cell "2026-11-08 11:04" [ref=e598]
                                            - cell "status=failed" [ref=e600]
                                        - row [ref=e602]:
                                            - cell "- -" [ref=e603]:
                                                - generic [ref=e606]:
                                                    - generic [ref=e607]: "-"
                                                    - generic [ref=e608]: "-"
                                            - cell "-" [ref=e609]
                                            - cell "错误信息" [ref=e611]
                                            - cell "2026-11-08 11:04" [ref=e616]
                                            - cell "fixture failure" [ref=e618]
                            - generic [ref=e620]:
                                - separator
                            - table [ref=e624]:
                                - rowgroup [ref=e625]:
                                    - row [ref=e626]:
                                        - cell "标准输出" [ref=e627]
                                        - cell "-" [ref=e628]
                                    - row [ref=e630]:
                                        - cell "错误输出" [ref=e631]
                                        - cell [ref=e632]:
                                            - generic [ref=e633]:
                                                - text: fixture failure
                                                - button "复制" [ref=e634] [cursor=pointer]
        - generic [ref=e637]: Pantheon Platform · Enterprise Admin Foundation
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
  70  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
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
> 87  |       await expect.poll(() => workbench.evaluate((element) => Math.max(...Array.from(element.children).map((child) => child.getBoundingClientRect().right - element.getBoundingClientRect().right)))).toBeLessThanOrEqual(1);
      |                                                                                                                                                                                                       ^ Error: expect(received).toBeLessThanOrEqual(expected)
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
