# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-workbench.spec.ts >> Loki UTC range, cancellation, denial and recovery (light)
- Location: tests\smoke\business\observability-workbench.spec.ts:8:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('Loki fixture', { exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('Loki fixture', { exact: true }) with timeout 10000ms
  - waiting for getByText('Loki fixture', { exact: true })

```

# Test source

```ts
  1   | import { expect, test } from '@playwright/test';
  2   | import fs from 'node:fs/promises';
  3   | import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';
  4   |
  5   | // Transport fixtures exercise browser states; provider cancellation/budgets are
  6   | // separately verified against a real local HTTP server and isolated MySQL.
  7   | for (const mode of ['light', 'dark'] as const) {
  8   |   test(`Loki UTC range, cancellation, denial and recovery (${mode})`, async ({ page }, testInfo) => {
  9   |     const login = await loginByApi(page.request, adminCredentials);
  10  |     await installClientSession(page, login);
  11  |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  12  |     await page.route('**/business/observability/datasources', async (route) => {
  13  |       await route.fulfill({ json: { code: 200, data: { total: 1, items: [{ id: 901, name: 'Loki fixture', type: 'loki', isDefault: true }] } } });
  14  |     });
  15  |     let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' | 'warning' | 'badResponse' | 'wide' = 'ready';
  16  |     let release: (() => void) | undefined;
  17  |     let queryCount = 0;
  18  |     await page.route('**/business/observability/datasources/901/logs?*', async (route) => {
  19  |       queryCount++;
  20  |       const url = new URL(route.request().url());
  21  |       expect(url.searchParams.get('start')).toMatch(/Z$/);
  22  |       expect(url.searchParams.get('end')).toMatch(/Z$/);
  23  |       expect(Number(url.searchParams.get('limit'))).toBe(500);
  24  |       if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
  25  |       await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200,
  26  |         json: state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
  27  |           code: 200, data: { status: state === 'badResponse' ? 'error' : 'success', warnings: state === 'warning' ? ['token=private-warning'] : undefined, data: { resultType: 'streams', result: state === 'empty' ? [] : [
  28  |             { stream: state === 'wide' ? Object.fromEntries(Array.from({ length: 500 }, (_, index) => [`label${index}`, index === 0 ? 'wide-value-'.repeat(1000) : `value-${index}`])) : { app: 'fixture' }, values: [['1794009600000000000', 'local-loki-result']] },
  29  |           ] } },
  30  |         } }).catch(() => { /* An aborted request has no response consumer. */ });
  31  |     });
  32  |     await page.setViewportSize({ width: 1440, height: 900 });
  33  |     await page.goto('/business/observability/logs');
> 34  |     await expect(page.getByText('Loki fixture', { exact: true })).toBeVisible();
      |                                                                   ^ Error: expect(locator).toBeVisible() failed
  35  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  36  |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  37  |     await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  38  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop.png`), fullPage: true });
  39  |     await page.getByPlaceholder(/LogQL/).fill('{app="other"}');
  40  |     await expect(page.getByText('查询条件已改变，当前显示上一次查询结果；请重新运行。', { exact: true })).toBeVisible();
  41  |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  42  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-stale.png`), fullPage: true });
  43  |     state = 'warning'; await page.getByRole('button', { name: '搜索', exact: true }).click();
  44  |     await expect(page.getByText('数据源返回 1 条警告，结果可能不完整。', { exact: true })).toBeVisible();
  45  |     await expect(page.getByText(/private-warning/)).toHaveCount(0);
  46  |     await expect(page.getByText('查询条件已改变，当前显示上一次查询结果；请重新运行。', { exact: true })).toHaveCount(0);
  47  |     await page.setViewportSize({ width: 390, height: 844 });
  48  |     await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  49  |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  50  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-partial.png`), fullPage: true });
  51  |     await page.setViewportSize({ width: 1440, height: 900 });
  52  |     await page.getByRole('tab', { name: '字段统计', exact: true }).click();
  53  |     await expect(page.getByText('app', { exact: true })).toBeVisible();
  54  |     await expect(page.getByText('统计仅覆盖本次返回的日志行，不代表全部匹配日志。', { exact: true })).toBeVisible();
  55  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-fields.png`), fullPage: true });
  56  |     await page.getByRole('tab', { name: '时间分桶', exact: true }).click();
  57  |     await expect(page.getByText('2026-11-07T00:00:00.000Z', { exact: true })).toBeVisible();
  58  |     await page.getByRole('tab', { name: '原始行', exact: true }).click();
  59  |     await expect(page.getByRole('region', { name: '原始行', exact: true })).toContainText('local-loki-result');
  60  |     await page.getByRole('region', { name: '原始行', exact: true }).focus();
  61  |     await expect(page.getByRole('region', { name: '原始行', exact: true })).toBeFocused();
  62  |     await page.setViewportSize({ width: 390, height: 844 });
  63  |     await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  64  |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  65  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-raw.png`), fullPage: true });
  66  |     await page.getByRole('tab', { name: '日志', exact: true }).click();
  67  |     await page.setViewportSize({ width: 1440, height: 900 });
  68  |     state = 'slow';
  69  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  70  |     await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
  71  |     await expect.poll(() => Boolean(release)).toBe(true);
  72  |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  73  |     await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
  74  |     release?.();
  75  |     await expect(page.getByText(/网络异常.*canceled/)).toHaveCount(0);
  76  |     state = 'forbidden';
  77  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  78  |     await expect(page.getByText('没有日志查询权限', { exact: true })).toBeVisible();
  79  |     await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
  80  |     await page.setViewportSize({ width: 390, height: 844 });
  81  |     await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  82  |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  83  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-forbidden.png`), fullPage: true });
  84  |     state = 'empty';
  85  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  86  |     await expect(page.getByText('暂无日志结果。', { exact: true })).toBeVisible();
  87  |     state = 'error';
  88  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  89  |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  90  |     state = 'badResponse';
  91  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  92  |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  93  |     await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
  94  |     state = 'ready';
  95  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  96  |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  97  |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-ready.png`), fullPage: true });
  98  |     state = 'wide';
  99  |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  100 |     await expect(page.getByText('另有 494 个标签；详情请查看原始行。', { exact: true })).toBeVisible();
  101 |     await expect(page.getByText('标签预览已截短；详情请查看原始行。', { exact: true })).toBeVisible();
  102 |     await expect(page.locator('.observability-log-labels .arco-tag')).toHaveCount(6);
  103 |     const labelLengths = await page.locator('.observability-log-labels__value').evaluateAll((elements) => elements.map((element) => element.textContent?.length ?? 0));
  104 |     expect(labelLengths.every((length) => length <= 81)).toBeTruthy();
  105 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  106 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-wide-labels.png`), fullPage: true });
  107 |     await page.setViewportSize({ width: 1440, height: 900 });
  108 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-wide-labels.png`), fullPage: true });
  109 |     await page.getByRole('tab', { name: '原始行', exact: true }).click();
  110 |     await expect(page.getByRole('region', { name: '原始行', exact: true })).toContainText('label499');
  111 |     await fs.writeFile(testInfo.outputPath('query-transport.json'), JSON.stringify({ queryCount, mode, provider: 'browser transport fixture', resultModes: ['logs', 'trend', 'fields', 'raw'], storedResults: false, labelFixtureCount: 500, renderedLabels: 6, maxPreviewCharacters: 81 }, null, 2));
  112 |     await testInfo.attach('query-transport', { contentType: 'application/json', body: JSON.stringify({ queryCount, mode, provider: 'browser transport fixture' }) });
  113 |   });
  114 | }
  115 |
```
