# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-tempo-workbench.spec.ts >> Tempo UTC range, detail, cancellation and revocation (dark)
- Location: tests\smoke\business\observability-tempo-workbench.spec.ts:11:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('Tempo fixture', { exact: true })
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByText('Tempo fixture', { exact: true }) with timeout 10000ms
  - waiting for getByText('Tempo fixture', { exact: true })

```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';
  3  |
  4  | // Includes real fixture lifecycle, multiple rendered states and remote asset loading.
  5  | // Keep assertion deadlines unchanged while giving this scenario a bounded budget.
  6  | test.describe.configure({ timeout: 90_000 });
  7  |
  8  | // Served by the remote acceptance build. Transport fixtures cover deterministic
  9  | // UI states; the real Tempo provider has a separate opt-in acceptance test.
  10 | for (const mode of ['light', 'dark'] as const) {
  11 |   test(`Tempo UTC range, detail, cancellation and revocation (${mode})`, async ({ page }, testInfo) => {
  12 |     const login = await loginByApi(page.request, adminCredentials);
  13 |     await installClientSession(page, login);
  14 |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  15 |     await page.route('**/business/observability/datasources', (route) => route.fulfill({ json: {
  16 |       code: 200, data: { total: 1, items: [{ id: 902, name: 'Tempo fixture', type: 'tempo', isDefault: true }] },
  17 |     } }));
  18 |     let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' = 'ready';
  19 |     let detailState: 'ready' | 'error' | 'forbidden' = 'ready';
  20 |     let release: (() => void) | undefined;
  21 |     await page.route('**/business/observability/datasources/902/traces?*', async (route) => {
  22 |       const url = new URL(route.request().url());
  23 |       expect(url.searchParams.get('start')).toMatch(/Z$/);
  24 |       expect(url.searchParams.get('end')).toMatch(/Z$/);
  25 |       expect(url.searchParams.get('limit')).toBe('100');
  26 |       if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
  27 |       await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200, json:
  28 |         state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
  29 |           code: 200, data: { traces: state === 'empty' ? [] : [{ traceID: '11111111111111111111111111111111', rootServiceName: 'tempo-ui-fixture', rootTraceName: 'acceptance-operation', startTimeUnixNano: '1794009600000000000', durationMs: 12 }] },
  30 |         },
  31 |       }).catch(() => { /* A stopped request no longer has a response consumer. */ });
  32 |     });
  33 |     await page.route('**/business/observability/datasources/902/traces/*', (route) => route.fulfill({
  34 |       status: detailState === 'forbidden' ? 403 : detailState === 'error' ? 502 : 200,
  35 |       json: detailState === 'ready' ? { code: 200, data: { fixture: 'safe-trace-detail' } } : { code: detailState === 'forbidden' ? 403 : 502, message: 'request.failed' },
  36 |     }));
  37 |     await page.setViewportSize({ width: 1440, height: 900 });
  38 |     await page.goto('/business/observability/traces');
> 39 |     await expect(page.getByText('Tempo fixture', { exact: true })).toBeVisible();
     |                                                                    ^ Error: expect(locator).toBeVisible() failed
  40 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  41 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
  42 |     await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  43 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop.png`), fullPage: true });
  44 |     await page.getByPlaceholder(/TraceQL/).fill('{ resource.service.name = "changed" }');
  45 |     await expect(page.getByText('查询条件已改变，当前显示上一次链路结果；请重新运行。', { exact: true })).toBeVisible();
  46 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop-stale.png`), fullPage: true });
  47 |     await page.getByRole('button', { name: '详情', exact: true }).click();
  48 |     const detail = page.getByRole('region', { name: /11111111111111111111111111111111/ });
  49 |     await expect(detail).toContainText('safe-trace-detail');
  50 |     await detail.focus(); await expect(detail).toBeFocused();
  51 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-detail.png`), fullPage: true });
  52 |     await page.keyboard.press('Escape');
  53 |     await expect(detail).toHaveCount(0);
  54 |     await page.setViewportSize({ width: 390, height: 844 });
  55 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  56 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile.png`), fullPage: true });
  57 |     state = 'slow';
  58 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  59 |     await expect.poll(() => Boolean(release)).toBe(true);
  60 |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  61 |     await expect(page.getByText('查询已停止，已有链路来自上一次查询', { exact: true })).toBeVisible();
  62 |     release?.();
  63 |     state = 'forbidden';
  64 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  65 |     await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
  66 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
  67 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile-forbidden.png`), fullPage: true });
  68 |     state = 'empty';
  69 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  70 |     await expect(page.getByText('暂无链路追踪结果。', { exact: true })).toBeVisible();
  71 |     state = 'error';
  72 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  73 |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  74 |     state = 'ready';
  75 |     await page.getByRole('button', { name: '重试', exact: true }).click();
  76 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
  77 |     await page.setViewportSize({ width: 1440, height: 900 });
  78 |     detailState = 'error';
  79 |     await page.getByRole('button', { name: '详情', exact: true }).click();
  80 |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  81 |     detailState = 'ready';
  82 |     await page.getByRole('button', { name: /重试/ }).click();
  83 |     await expect(detail).toContainText('safe-trace-detail');
  84 |     await page.keyboard.press('Escape');
  85 |     detailState = 'forbidden';
  86 |     await page.getByRole('button', { name: '详情', exact: true }).click();
  87 |     await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
  88 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
  89 |     await expect(detail).toHaveCount(0);
  90 |   });
  91 | }
  92 |
```
