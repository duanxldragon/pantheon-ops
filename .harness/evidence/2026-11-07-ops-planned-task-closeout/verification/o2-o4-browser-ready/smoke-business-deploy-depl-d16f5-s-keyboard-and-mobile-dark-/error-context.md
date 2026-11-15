# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-workbench.spec.ts >> deployment workbench sources, logs, keyboard and mobile (dark)
- Location: tests\smoke\business\deploy\deploy-workbench.spec.ts:16:3

# Error details

```
Error: expect(locator).toHaveCount(expected) failed

Locator:  locator('.deploy-execution-workbench').locator('.task-log-viewer__row')
Expected: 1
Received: 0
Timeout:  10000ms

Call log:
  - Expect "toHaveCount" locator('.deploy-execution-workbench').locator('.task-log-viewer__row') with timeout 10000ms
  - waiting for locator('.deploy-execution-workbench').locator('.task-log-viewer__row')
    24 × locator resolved to 0 elements
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
            - menu [ref=e12]:
                - generic [ref=e13]:
                    - menuitem [ref=e14] [cursor=pointer]:
                        - button "工作台" [ref=e15]
                    - button "CMDB" [ref=e25] [cursor=pointer]
                    - menuitem [ref=e37] [cursor=pointer]:
                        - button "业务域" [ref=e38]
                    - menuitem [ref=e47] [cursor=pointer]:
                        - button "服务目录" [ref=e48]
                    - button "安装部署" [ref=e60] [cursor=pointer]
                    - button "Kubernetes" [ref=e72] [cursor=pointer]
                    - button "可观测中心" [ref=e84] [cursor=pointer]
                    - button "访问控制" [ref=e96] [cursor=pointer]
                    - button "组织架构" [ref=e110] [cursor=pointer]
                    - button "安全审计" [ref=e125] [cursor=pointer]
                    - button "低代码平台" [ref=e137] [cursor=pointer]
                    - button "平台配置" [ref=e149] [cursor=pointer]
                    - menuitem [ref=e158] [cursor=pointer]:
                        - button "文件分发" [ref=e159]
                    - menuitem [ref=e168] [cursor=pointer]:
                        - button "ArgoCD" [ref=e169]
    - generic [ref=e175]:
        - generic [ref=e176]:
            - generic [ref=e177]:
                - button "切换导航" [ref=e178] [cursor=pointer]
                - list [ref=e182]:
                    - listitem [ref=e183]: 首页
                    - listitem [ref=e187]: 任务详情
            - generic [ref=e188]:
                - button "平台功能台" [ref=e190]:
                    - generic [ref=e193]: 搜索菜单、页面、功能…
                    - generic [ref=e194]: Ctrl K
                - button "通知中心" [ref=e196] [cursor=pointer]
                - button "平台偏好" [ref=e200] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e205] [cursor=pointer]':
                    - generic [ref=e206]: A
                    - generic [ref=e208]:
                        - generic [ref=e209]: Administrator
                        - generic [ref=e210]: admin
        - tablist "已打开页面" [ref=e211]:
            - tab "工作台" [ref=e212] [cursor=pointer]
            - tab "任务详情" [selected] [ref=e216] [cursor=pointer]
        - main [ref=e218]:
            - generic [ref=e220]:
                - generic [ref=e221]:
                    - generic [ref=e222]:
                        - heading "workbench-dark-1794098093951" [level=5] [ref=e224]
                        - generic [ref=e225]: 任务详情
                    - button "返回" [ref=e229] [cursor=pointer]
                - generic [ref=e232]:
                    - generic [ref=e235]:
                        - generic [ref=e236]:
                            - generic [ref=e237]:
                                - generic [ref=e238]: 运维平台 / 部署任务
                                - heading "workbench-dark-1794098093951" [level=5] [ref=e239]
                                - generic [ref=e240]: 查看任务摘要、状态流转和每台目标主机的执行结果，便于运维排障与人工收口。
                            - generic [ref=e241]: 失败
                        - generic [ref=e243]:
                            - generic [ref=e244]:
                                - generic [ref=e245]: 目标主机
                                - generic [ref=e246]: "1"
                            - generic [ref=e247]:
                                - generic [ref=e248]: 成功
                                - generic [ref=e249]: "0"
                            - generic [ref=e250]:
                                - generic [ref=e251]: 失败
                                - generic [ref=e252]: "1"
                    - generic [ref=e256]:
                        - generic [ref=e257]: 任务状态流转
                        - list "任务状态流转" [ref=e259]:
                            - listitem [ref=e260]:
                                - generic [ref=e262]: 草稿
                            - listitem [ref=e263]:
                                - generic [ref=e265]: 待执行
                            - listitem [ref=e266]:
                                - generic [ref=e268]: 执行中
                            - listitem [ref=e269]:
                                - generic [ref=e271]: 成功
                            - listitem [ref=e272]:
                                - generic [ref=e274]: 失败
                            - listitem [ref=e275]:
                                - generic [ref=e277]: 已取消
                        - generic [ref=e278]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                    - generic [ref=e282]:
                        - generic [ref=e283]: 任务级操作
                        - generic [ref=e285]:
                            - generic [ref=e286]:
                                - generic [ref=e287]: 仅允许在可操作状态执行启动或取消，任务结果会同步汇总到详情页。
                                - generic [ref=e288]: 任务状态会根据启动、取消和主机结果汇总自动推进。
                            - generic [ref=e289]:
                                - button "重试失败步骤" [ref=e291] [cursor=pointer]
                                - button "删除" [ref=e295] [cursor=pointer]
                    - table [ref=e303]:
                        - rowgroup [ref=e304]:
                            - row [ref=e305]:
                                - cell "任务来源" [ref=e306]
                                - cell "软件组件" [ref=e307]
                                - cell "软件组件" [ref=e308]
                                - cell "workbench-dark-1794098093951 1.0.0" [ref=e309]
                            - row [ref=e310]:
                                - cell "任务动作" [ref=e311]
                                - cell "安装" [ref=e312]
                                - cell "执行模式" [ref=e313]
                                - cell "固定任务" [ref=e314]
                            - row [ref=e315]:
                                - cell "模板参数" [ref=e316]
                                - cell "-" [ref=e317]
                                - cell "状态" [ref=e318]
                                - cell "失败" [ref=e319]
                            - row [ref=e320]:
                                - cell "目标来源" [ref=e321]
                                - cell "主机" [ref=e322]
                                - cell "业务域" [ref=e323]
                                - cell "workbench-dark-1794098093951" [ref=e324]
                            - row [ref=e325]:
                                - cell "执行方式" [ref=e326]
                                - cell "手动执行" [ref=e327]
                                - cell "外部任务 ID" [ref=e328]
                                - cell "-" [ref=e329]
                            - row [ref=e330]:
                                - cell "开始时间" [ref=e331]
                                - cell "2026-11-08 08:34" [ref=e332]
                                - cell "结束时间" [ref=e333]
                                - cell "2026-11-08 08:34" [ref=e334]
                            - row [ref=e335]:
                                - cell "耗时" [ref=e336]
                                - cell "-" [ref=e337]
                                - cell "主机数" [ref=e338]
                                - cell "1" [ref=e339]
                            - row [ref=e340]:
                                - cell "成功数" [ref=e341]
                                - cell "0" [ref=e342]
                                - cell "失败数" [ref=e343]
                                - cell "1" [ref=e344]
                            - row [ref=e345]:
                                - cell "执行中" [ref=e346]
                                - cell "0" [ref=e347]
                                - cell "跳过数" [ref=e348]
                                - cell "0" [ref=e349]
                            - row [ref=e350]:
                                - cell "备注" [ref=e351]
                                - cell "-" [ref=e352]
                    - generic [ref=e356]:
                        - generic [ref=e359]:
                            - generic [ref=e360]: 主机执行明细
                            - generic [ref=e361]: 先查看各状态分布，再按主机展开过程日志和标准输出。
                        - generic [ref=e363]:
                            - generic [ref=e364]:
                                - generic [ref=e365]: 待执行
                                - generic [ref=e366]: "0"
                            - generic [ref=e367]:
                                - generic [ref=e368]: 执行中
                                - generic [ref=e369]: "0"
                            - generic [ref=e370]:
                                - generic [ref=e371]: 成功
                                - generic [ref=e372]: "0"
                            - generic [ref=e373]:
                                - generic [ref=e374]: 失败
                                - generic [ref=e375]: "1"
                            - generic [ref=e376]:
                                - generic [ref=e377]: 已跳过
                                - generic [ref=e378]: "0"
                        - generic [ref=e383]:
                            - table [ref=e387]:
                                - rowgroup [ref=e397]:
                                    - row [ref=e398]:
                                        - columnheader "主机" [ref=e399]
                                        - columnheader "主机 IP" [ref=e401]
                                        - columnheader "状态" [ref=e403]
                                        - columnheader "开始时间" [ref=e405]
                                        - columnheader "结束时间" [ref=e407]
                                        - columnheader "耗时" [ref=e409]
                                        - columnheader "错误信息" [ref=e411]
                                        - columnheader "操作" [ref=e413]
                                - rowgroup [ref=e415]:
                                    - row [ref=e416]:
                                        - cell "workbench-dark-1794098093951" [ref=e417]
                                        - cell "10.251.71.88" [ref=e419]
                                        - cell "失败" [ref=e421]
                                        - cell "2026-11-08 08:34" [ref=e426]
                                        - cell "2026-11-08 08:34" [ref=e428]
                                        - cell "-" [ref=e430]
                                        - cell "fixture failure" [ref=e432]
                                        - cell [ref=e434]
                            - list [ref=e440]:
                                - listitem "上一页" [ref=e441]: .
                                - listitem "第 1 页" [ref=e444] [cursor=pointer]: "1"
                                - listitem "下一页" [ref=e445]: .
                    - region "日志来源" [ref=e449]:
                        - generic [ref=e450]:
                            - combobox "执行目标" [ref=e452] [cursor=pointer]:
                                - generic "workbench-dark-1794098093951 / 10.251.71.88" [ref=e453]:
                                    - generic [ref=e454]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e455]: workbench-dark-1794098093951 / 10.251.71.88
                            - combobox "尝试次数" [ref=e461] [cursor=pointer]:
                                - generic "#2" [ref=e462]:
                                    - generic [ref=e463]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e464]: "#2"
                            - button "定位首个失败目标" [ref=e470] [cursor=pointer]
                            - button "重新连接" [ref=e472] [cursor=pointer]
                            - generic [ref=e473]: workbench-dark-1794098093951 / 10.251.71.88
                        - region "执行步骤" [ref=e476]:
                            - list [ref=e479]:
                                - listitem [ref=e480]:
                                    - button "定位步骤 1" [ref=e481]:
                                        - generic [ref=e483]: fixture-install
                                        - generic [ref=e484]:
                                            - generic [ref=e485]: failed
                                            - text: "尝试次数: 2"
                        - button "查看全部步骤日志" [ref=e487] [cursor=pointer]
                        - generic [ref=e488]:
                            - generic [ref=e489]:
                                - generic [ref=e490]: 日志窗口已截断；重连可能重放或缺失
                                - button "展开日志" [ref=e494] [cursor=pointer]
                                - combobox "日志流" [ref=e496] [cursor=pointer]:
                                    - generic [ref=e498]:
                                        - textbox "日志流"
                                        - generic [ref=e499]: 日志流
                            - 'region "workbench-dark-1794098093951 / workbench-dark-1794098093951 / #2 / fixture-install" [ref=e504]':
                                - generic [ref=e505]:
                                    - generic [ref=e506]: "workbench-dark-1794098093951 / workbench-dark-1794098093951 / #2 / fixture-install"
                                    - generic [ref=e507]:
                                        - textbox "搜索已保留日志" [active] [ref=e511]: fixture-error
                                        - button "恢复追尾" [ref=e515] [cursor=pointer]
                                        - switch "自动换行" [checked] [ref=e516] [cursor=pointer]
                                - generic [ref=e518]: 显示 0 条日志
                        - generic [ref=e525]:
                            - generic [ref=e527]:
                                - generic [ref=e528]:
                                    - generic [ref=e529]: workbench-dark-1794098093951
                                    - generic [ref=e530]: 10.251.71.88 / 2026-11-08 08:34
                                - generic [ref=e532]: 失败
                            - table [ref=e542]:
                                - rowgroup [ref=e549]:
                                    - row [ref=e550]:
                                        - columnheader "步骤" [ref=e551]
                                        - columnheader "步骤类型" [ref=e553]
                                        - columnheader "阶段" [ref=e555]
                                        - columnheader "开始时间" [ref=e557]
                                        - columnheader "过程信息" [ref=e559]
                                - rowgroup [ref=e561]:
                                    - row [ref=e562]:
                                        - cell "- -" [ref=e563]:
                                            - generic [ref=e566]:
                                                - generic [ref=e567]: "-"
                                                - generic [ref=e568]: "-"
                                        - cell "-" [ref=e569]
                                        - cell "进入执行队列" [ref=e571]
                                        - cell "2026-11-08 08:34" [ref=e576]
                                        - cell "Task host queued" [ref=e578]
                                    - row [ref=e580]:
                                        - cell "- -" [ref=e581]:
                                            - generic [ref=e584]:
                                                - generic [ref=e585]: "-"
                                                - generic [ref=e586]: "-"
                                        - cell "-" [ref=e587]
                                        - cell "记录执行结果" [ref=e589]
                                        - cell "2026-11-08 08:34" [ref=e594]
                                        - cell "status=failed" [ref=e596]
                                    - row [ref=e598]:
                                        - cell "- -" [ref=e599]:
                                            - generic [ref=e602]:
                                                - generic [ref=e603]: "-"
                                                - generic [ref=e604]: "-"
                                        - cell "-" [ref=e605]
                                        - cell "错误信息" [ref=e607]
                                        - cell "2026-11-08 08:34" [ref=e612]
                                        - cell "fixture failure" [ref=e614]
                            - generic [ref=e616]:
                                - separator
                            - table [ref=e620]:
                                - rowgroup [ref=e621]:
                                    - row [ref=e622]:
                                        - cell "标准输出" [ref=e623]
                                        - cell "-" [ref=e624]
                                    - row [ref=e626]:
                                        - cell "错误输出" [ref=e627]
                                        - cell [ref=e628]:
                                            - generic [ref=e629]:
                                                - text: fixture failure
                                                - button "复制" [ref=e630] [cursor=pointer]
        - generic [ref=e633]: Pantheon Platform · Enterprise Admin Foundation
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
  88  |       await workbench.getByRole('button', { name: '暂停追尾', exact: true }).click();
  89  |       await expect(workbench.getByRole('button', { name: '恢复追尾', exact: true })).toBeVisible();
  90  |       await workbench.locator('.task-log-viewer input').fill('fixture-error');
> 91  |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(1);
      |                                                                ^ Error: expect(locator).toHaveCount(expected) failed
  92  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-desktop.png`), fullPage: true });
  93  |       await page.setViewportSize({ width: 390, height: 844 });
  94  |       await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  95  |       await workbench.getByRole('button', { name: '展开日志', exact: true }).click();
  96  |       const dialog = page.getByRole('dialog');
  97  |       await expect(dialog).toBeVisible();
  98  |       await expect(dialog.getByText('fixture-error', { exact: true })).toBeVisible();
  99  |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-expanded.png`), fullPage: true });
  100 |       await page.keyboard.press('Escape');
  101 |       await expect(dialog).toBeHidden();
  102 |       denied = true;
  103 |       await page.reload();
  104 |       await expect(page.getByText('没有日志查看权限', { exact: true }).first()).toBeVisible();
  105 |       await expect(page.getByText('fixture-output', { exact: true })).toHaveCount(0);
  106 |       await page.screenshot({ path: testInfo.outputPath(`workbench-${mode}-mobile-forbidden.png`), fullPage: true });
  107 |       denied = false;
  108 |       await workbench.getByRole('button', { name: '重新连接', exact: true }).click();
  109 |       await expect(workbench.locator('.task-log-viewer__row')).toHaveCount(500);
  110 |       expect(logRequests.some((query) => query.includes('firstError=1') && query.includes('after=0') && query.includes('stepId=901'))).toBe(true);
  111 |       await fs.writeFile(testInfo.outputPath('provider-source-requests.json'), JSON.stringify(logRequests, null, 2));
  112 |       await testInfo.attach('provider-source-requests', { contentType: 'application/json', body: JSON.stringify(logRequests) });
  113 |     } catch (error) {
  114 |       originalError = error;
  115 |       throw error;
  116 |     } finally {
  117 |       const cleanupErrors: string[] = [];
  118 |       try {
  119 |         const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  120 |         for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  121 |           if (!id) continue;
  122 |           try {
  123 |             await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders }));
  124 |           } catch (error) {
  125 |             cleanupErrors.push(`${endpoint}/${id}: ${error instanceof Error ? error.message : String(error)}`);
  126 |           }
  127 |         }
  128 |       } catch (error) {
  129 |         cleanupErrors.push(`operation verification: ${error instanceof Error ? error.message : String(error)}`);
  130 |       }
  131 |       await fs.writeFile(testInfo.outputPath('fixture-cleanup.json'), JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }, null, 2));
  132 |       await testInfo.attach('fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, scopeId, hostId, packageId, taskId, cleanupErrors }) });
  133 |       if (!originalError) expect(cleanupErrors, 'All owned fixtures must be cleaned up').toEqual([]);
  134 |     }
  135 |   });
  136 | }
  137 |
```
