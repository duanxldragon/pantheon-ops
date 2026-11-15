# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-pages.spec.ts >> Deploy business module smoke >> deploy pages load under operations platform
- Location: tests\smoke\business\deploy\deploy-pages.spec.ts:67:3

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
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import {
  3   |   apiBaseUrl,
  4   |   apiRequestHeaders,
  5   |   adminCredentials,
  6   |   installClientSession,
  7   |   loginByApi,
  8   |   primeChineseLocale,
  9   | } from '../../helpers/auth';
  10  |
  11  | type ApiEnvelope<T> = {
  12  |   code: number;
  13  |   data: T;
  14  |   message?: string;
  15  | };
  16  |
  17  | type HostRow = {
  18  |   id: number;
  19  |   hostname: string;
  20  |   ip: string;
  21  |   status: string;
  22  | };
  23  |
  24  | type BizScopeRow = {
  25  |   id: number;
  26  | };
  27  |
  28  | type PackageRow = {
  29  |   id: number;
  30  |   name: string;
  31  |   version: string;
  32  |   sourceFileName?: string;
  33  |   latestDeployedAt?: string;
  34  | };
  35  |
  36  | type TemplateRow = {
  37  |   id: number;
  38  | };
  39  |
  40  | type TaskRow = {
  41  |   id: number;
  42  |   status: string;
  43  |   businessScopeName?: string;
  44  |   hosts: Array<{
  45  |     id: number;
  46  |     status: string;
  47  |     hostId: number;
  48  |     stdout?: string;
  49  |     traceSteps?: Array<{ phase?: string; message?: string }>;
  50  |   }>;
  51  | };
  52  |
  53  | async function expectSuccess<T>(response: APIResponse): Promise<T> {
  54  |   expect(response.ok()).toBeTruthy();
  55  |   const payload = await response.json() as ApiEnvelope<T>;
  56  |   expect(payload.code).toBe(200);
  57  |   return payload.data;
  58  | }
  59  |
  60  | test.describe('Deploy business module smoke', () => {
  61  |   test.beforeEach(async ({ page }) => {
  62  |     const login = await loginByApi(page.request, adminCredentials);
  63  |     await installClientSession(page, login);
  64  |     await primeChineseLocale(page);
  65  |   });
  66  |
  67  |   test('deploy pages load under operations platform', async ({ page }, testInfo) => {
  68  |     const consoleErrors: string[] = [];
  69  |     page.on('console', (message) => {
  70  |       if (message.type() === 'error') {
  71  |         consoleErrors.push(message.text());
  72  |       }
  73  |     });
  74  |
  75  |     await page.goto('/business/deploy/package', { waitUntil: 'networkidle' });
> 76  |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
      |                                                           ^ Error: expect(locator).toBeVisible() failed
  77  |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  78  |     await expect(page.locator('.governance-summary-bar')).toContainText('软件组件');
  79  |     await page.locator('.governance-summary-bar .arco-btn').click();
  80  |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  81  |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  82  |     await expect(page.locator('.filter-panel')).toBeVisible();
  83  |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  84  |     await page.screenshot({ path: testInfo.outputPath('deploy-package-list.png'), fullPage: true });
  85  |
  86  |     await page.goto('/business/deploy/template', { waitUntil: 'networkidle' });
  87  |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  88  |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  89  |     await expect(page.locator('.governance-summary-bar')).toContainText('任务模板');
  90  |     await page.locator('.governance-summary-bar .arco-btn').click();
  91  |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  92  |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  93  |     await expect(page.locator('.filter-panel')).toBeVisible();
  94  |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  95  |     await page.screenshot({ path: testInfo.outputPath('deploy-template-list.png'), fullPage: true });
  96  |
  97  |     await page.goto('/business/deploy/task', { waitUntil: 'networkidle' });
  98  |     await expect(page.locator('.governance-summary-bar')).toBeVisible();
  99  |     await expect(page.locator('.governance-summary-bar__title-row')).toBeVisible();
  100 |     await expect(page.locator('.governance-summary-bar')).toContainText('部署任务');
  101 |     await page.locator('.governance-summary-bar .arco-btn').click();
  102 |     await expect(page.locator('.governance-insight-drawer')).toBeVisible();
  103 |     await expect(page.locator('.table-batch-action-bar')).toBeVisible();
  104 |     await expect(page.locator('.filter-panel')).toBeVisible();
  105 |     await expect(page.locator('.system-list__table-card')).toBeVisible();
  106 |     await page.screenshot({ path: testInfo.outputPath('deploy-task-list.png'), fullPage: true });
  107 |     expect(consoleErrors).toEqual([]);
  108 |   });
  109 |
  110 |   test('deploy task detail action opens a modal without leaving the list page', async ({ page, request }, testInfo) => {
  111 |     const login = await loginByApi(request, adminCredentials);
  112 |     await installClientSession(page, login);
  113 |     const headers = apiRequestHeaders(login);
  114 |     const token = `deploy-detail-modal-${Date.now()}`;
  115 |
  116 |     const scope = await expectSuccess<BizScopeRow>(
  117 |       await request.post(`${apiBaseUrl}/business/bizscope`, {
  118 |         headers,
  119 |         data: {
  120 |           code: token,
  121 |           name: token,
  122 |           environment: 'dev',
  123 |           status: 'active',
  124 |         },
  125 |       }),
  126 |     );
  127 |
  128 |     const host = await expectSuccess<HostRow>(
  129 |       await request.post(`${apiBaseUrl}/business/cmdb/hosts`, {
  130 |         headers,
  131 |         data: {
  132 |           hostname: `${token}-host`,
  133 |           ip: `10.248.${Math.floor(Math.random() * 200) + 1}.${Math.floor(Math.random() * 200) + 1}`,
  134 |           os: 'linux',
  135 |           businessScopeId: scope.id,
  136 |           labels: [{ key: 'biz', val: token }],
  137 |         },
  138 |       }),
  139 |     );
  140 |
  141 |     const pkg = await expectSuccess<PackageRow>(
  142 |       await request.post(`${apiBaseUrl}/business/deploy/packages`, {
  143 |         headers,
  144 |         data: {
  145 |           name: token,
  146 |           version: '1.30.2',
  147 |           installCommand: 'echo install',
  148 |           status: 'enabled',
  149 |         },
  150 |       }),
  151 |     );
  152 |
  153 |     const task = await expectSuccess<TaskRow>(
  154 |       await request.post(`${apiBaseUrl}/business/deploy/tasks`, {
  155 |         headers,
  156 |         data: {
  157 |           name: `${token}-task`,
  158 |           packageId: pkg.id,
  159 |           businessScopeId: scope.id,
  160 |           targetType: 'host',
  161 |           targetIds: [host.id],
  162 |           executorType: 'manual',
  163 |         },
  164 |       }),
  165 |     );
  166 |
  167 |     try {
  168 |       const started = await expectSuccess<TaskRow>(
  169 |         await request.post(`${apiBaseUrl}/business/deploy/tasks/${task.id}/start`, { headers }),
  170 |       );
  171 |       expect(started.hosts).toHaveLength(1);
  172 |
  173 |       await expectSuccess(
  174 |         await request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, {
  175 |           headers,
  176 |           data: { status: 'success', stdout: 'installed via modal smoke' },
```
