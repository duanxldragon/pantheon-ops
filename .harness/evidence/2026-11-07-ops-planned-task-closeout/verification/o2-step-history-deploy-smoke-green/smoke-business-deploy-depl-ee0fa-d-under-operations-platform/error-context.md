# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy pages load under operations platform
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:69:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.governance-summary-bar')
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.governance-summary-bar') with timeout 10000ms
  - waiting for locator('.governance-summary-bar')

```

```yaml
- complementary:
    - text: P Pantheon Base
    - menu
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 软件组件
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "平台偏好"
- 'button "用户: 用户"': 用 用户
- tablist "已打开页面"
- main:
    - text: 加载失败 当前区块暂时无法返回数据，请重试或稍后再查看。
    - button "重试"
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1   | import { startReviewedTask } from './reviewed-start';
  2   | import { expect, test, type APIResponse } from '@playwright/test';
  3   | import {
  4   |   apiBaseUrl,
  5   |   apiRequestHeaders,
  6   |   adminCredentials,
  7   |   installClientSession,
  8   |   loginByApi,
  9   |   primeChineseLocale,
  10  |   verifiedApiHeaders,
  11  | } from '../../helpers/auth';
  12  |
  13  | type ApiEnvelope<T> = {
  14  |   code: number;
  15  |   data: T;
  16  |   message?: string;
  17  | };
  18  |
  19  | type HostRow = {
  20  |   id: number;
  21  |   hostname: string;
  22  |   ip: string;
  23  |   status: string;
  24  | };
  25  |
  26  | type BizScopeRow = {
  27  |   id: number;
  28  | };
  29  |
  30  | type PackageRow = {
  31  |   id: number;
  32  |   name: string;
  33  |   version: string;
  34  |   sourceFileName?: string;
  35  |   latestDeployedAt?: string;
  36  | };
  37  |
  38  | type TemplateRow = {
  39  |   id: number;
  40  | };
  41  |
  42  | type TaskRow = {
  43  |   id: number;
  44  |   status: string;
  45  |   businessScopeName?: string;
  46  |   hosts: Array<{
  47  |     id: number;
  48  |     status: string;
  49  |     hostId: number;
  50  |     stdout?: string;
  51  |     traceSteps?: Array<{ phase?: string; message?: string }>;
  52  |   }>;
  53  | };
  54  |
  55  | async function expectSuccess<T>(response: APIResponse): Promise<T> {
  56  |   expect(response.ok()).toBeTruthy();
  57  |   const payload = await response.json() as ApiEnvelope<T>;
  58  |   expect(payload.code).toBe(200);
  59  |   return payload.data;
  60  | }
  61  |
  62  | test.describe('Deploy business module smoke', () => {
  63  |   test.beforeEach(async ({ page }) => {
  64  |     const login = await loginByApi(page.request, adminCredentials);
  65  |     await installClientSession(page, login);
  66  |     await primeChineseLocale(page);
  67  |   });
  68  |
  69  |   test('deploy pages load under operations platform', async ({ page }, testInfo) => {
  70  |     const consoleErrors: string[] = [];
  71  |     page.on('console', (message) => {
  72  |       if (message.type() === 'error') {
  73  |         consoleErrors.push(message.text());
  74  |       }
  75  |     });
  76  |
  77  |     await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
> 78  |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
      |                                                           ^ Error: expect(locator).toBeVisible() failed
  79  |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  80  |     await expect(page.locator('.governance-summary-bar')).toContainText('软件组件');
  81  |     await page.locator('.governance-summary-bar .arco-btn').click();
  82  |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  83  |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  84  |     await expect(page.locator('.filter-panel')).toBeVisible();
  85  |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  86  |     await page.screenshot({ path: testInfo.outputPath('deploy-package-list.png'), fullPage: true });
  87  |
  88  |     await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  89  |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  90  |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  91  |     await expect(page.locator('.governance-summary-bar')).toContainText('任务模板');
  92  |     await page.locator('.governance-summary-bar .arco-btn').click();
  93  |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  94  |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  95  |     await expect(page.locator('.filter-panel')).toBeVisible();
  96  |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  97  |     await page.screenshot({ path: testInfo.outputPath('deploy-template-list.png'), fullPage: true });
  98  |
  99  |     await page.goto('/business/deploy/task', { waitUntil: 'networkidle' });
  100 |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  101 |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  102 |     await expect(page.locator('.governance-summary-bar')).toContainText('部署任务');
  103 |     await page.locator('.governance-summary-bar .arco-btn').click();
  104 |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  105 |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  106 |     await expect(page.locator('.filter-panel')).toBeVisible();
  107 |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  108 |     await page.screenshot({ path: testInfo.outputPath('deploy-task-list.png'), fullPage: true });
  109 |     expect(consoleErrors).toEqual([]);
  110 |   });
  111 |
  112 |   test('deploy task detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  113 |     const login = await loginByApi(request, adminCredentials);
  114 |     await installClientSession(page, login);
  115 |     const headers = apiRequestHeaders(login);
  116 |     const token = `deploy-detail-modal-${Date.now()}`;
  117 |
  118 |     const scope = await expectSuccess<BizScopeRow>(
  119 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  120 |         headers,
  121 |         data: {
  122 |           code: token,
  123 |           name: token,
  124 |           environment: 'dev',
  125 |           status: 'active',
  126 |         },
  127 |       }),
  128 |     );
  129 |
  130 |     const host = await expectSuccess<HostRow>(
  131 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  132 |         headers,
  133 |         data: {
  134 |           hostname: `${token}-host`,
  135 |           ip: `10.248.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  136 |           os: 'linux',
  137 |           businessScopeId: scope.id,
  138 |           labels: [{ key: 'biz', val: token }],
  139 |         },
  140 |       }),
  141 |     );
  142 |
  143 |     const pkg = await expectSuccess<PackageRow>(
  144 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  145 |         headers,
  146 |         data: {
  147 |           name: token,
  148 |           version: '1.30.2',
  149 |           installCommand: 'echo install',
  150 |           status: 'enabled',
  151 |         },
  152 |       }),
  153 |     );
  154 |
  155 |     const task = await expectSuccess<TaskRow>(
  156 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  157 |         headers,
  158 |         data: {
  159 |           name: `${token}-task`,
  160 |           packageId: pkg.id,
  161 |           businessScopeId: scope.id,
  162 |           targetType: 'host',
  163 |           targetIds: [host.id],
  164 |           executorType: 'manual',
  165 |         },
  166 |       }),
  167 |     );
  168 |
  169 |     try {
  170 |       const started = await expectSuccess<TaskRow>(
  171 |         await startReviewedTask(request, task.id, { headers }),
  172 |       );
  173 |       expect(started.hosts).toHaveLength(1);
  174 |
  175 |       await expectSuccess(
  176 |         await request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
  177 |           headers,
  178 |           data: { status: 'success', stdout: 'installed via modal smoke' },
```
