# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-workbench.spec.ts >> Loki UTC range, cancellation, denial and recovery (light)
- Location: tests\smoke\business\observability-workbench.spec.ts:7:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('没有匹配到日志。', { exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('没有匹配到日志。', { exact: true }) with timeout 10000ms
  - waiting for getByText('没有匹配到日志。', { exact: true })

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
        - menuitem "监控中心":
            - button "监控中心"
        - menuitem "告警中心":
            - button "告警中心"
        - menuitem "日志中心":
            - button "日志中心"
        - menuitem "链路追踪":
            - button "链路追踪"
        - button "访问控制"
        - button "组织架构"
        - button "安全审计"
        - button "低代码平台"
        - button "平台配置"
        - menuitem "文件分发":
            - button "文件分发"
        - menuitem "ArgoCD":
            - button "ArgoCD"
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 可观测中心
    - listitem: 日志中心
- button "平台功能台": 搜索菜单、页面、功能…
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A
- tablist "已打开页面":
    - tab "工作台"
    - tab "日志中心" [selected]
- main:
    - text: 运维平台 / 日志中心 日志中心 数据源
    - strong: "1"
    - text: 命中数
    - strong: "0"
    - search "筛选":
        - 'textbox "输入 LogQL，例如 {app=~\".+\"}"': '{app=~".+"}'
        - button "筛选"
        - button "清空筛选"
        - button "搜索"
        - button "停止查询" [disabled]
    - text: 开始时间（UTC）
    - textbox "开始时间（UTC）": 2026-11-07T09:32
    - text: 结束时间（UTC）
    - textbox "结束时间（UTC）": 2026-11-07T09:47
    - text: 最多日志条数
    - combobox "最多日志条数": "500"
    - text: 单次范围最多 24 小时，响应最多 8 MiB；按所选条数截取结果 暂无日志结果。
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';
  3  |
  4  | // Transport fixtures exercise browser states; provider cancellation/budgets are
  5  | // separately verified against a real local HTTP server and isolated MySQL.
  6  | for (const mode of ['light', 'dark'] as const) {
  7  |   test(`Loki UTC range, cancellation, denial and recovery (${mode})`, async ({ page }, testInfo) => {
  8  |     const login = await loginByApi(page.request, adminCredentials);
  9  |     await installClientSession(page, login);
  10 |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  11 |     await page.route('**/business/observability/datasources', async (route) => {
  12 |       await route.fulfill({ json: { code: 200, data: { total: 1, items: [{ id: 901, name: 'Loki fixture', type: 'loki', isDefault: true }] } } });
  13 |     });
  14 |     let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' = 'ready';
  15 |     let release: (() => void) | undefined;
  16 |     let queryCount = 0;
  17 |     await page.route('**/business/observability/datasources/901/logs?*', async (route) => {
  18 |       queryCount++;
  19 |       const url = new URL(route.request().url());
  20 |       expect(url.searchParams.get('start')).toMatch(/Z$/);
  21 |       expect(url.searchParams.get('end')).toMatch(/Z$/);
  22 |       expect(Number(url.searchParams.get('limit'))).toBe(500);
  23 |       if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
  24 |       await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200,
  25 |         json: state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
  26 |           code: 200, data: { status: 'success', data: { resultType: 'streams', result: state === 'empty' ? [] : [
  27 |             { stream: { app: 'fixture' }, values: [['1794009600000000000', 'local-loki-result']] },
  28 |           ] } },
  29 |         } }).catch(() => { /* An aborted request has no response consumer. */ });
  30 |     });
  31 |     await page.setViewportSize({ width: 1440, height: 900 });
  32 |     await page.goto('/business/observability/logs');
  33 |     await expect(page.getByText('Loki fixture', { exact: true })).toBeVisible();
  34 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  35 |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  36 |     await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  37 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop.png`), fullPage: true });
  38 |     state = 'slow';
  39 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  40 |     await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
  41 |     await expect.poll(() => Boolean(release)).toBe(true);
  42 |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  43 |     await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
  44 |     release?.();
  45 |     state = 'forbidden';
  46 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  47 |     await expect(page.getByText('没有日志查询权限', { exact: true })).toBeVisible();
  48 |     await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
  49 |     await page.setViewportSize({ width: 390, height: 844 });
  50 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  51 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-forbidden.png`), fullPage: true });
  52 |     state = 'empty';
  53 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
> 54 |     await expect(page.getByText('没有匹配到日志。', { exact: true })).toBeVisible();
     |                                                               ^ Error: expect(locator).toBeVisible() failed
  55 |     state = 'error';
  56 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  57 |     await expect(page.locator('.page-error')).toBeVisible();
  58 |     state = 'ready';
  59 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  60 |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  61 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-ready.png`), fullPage: true });
  62 |     await testInfo.attach('query-transport', { contentType: 'application/json', body: JSON.stringify({ queryCount, mode, provider: 'browser transport fixture' }) });
  63 |   });
  64 | }
  65 |
```
