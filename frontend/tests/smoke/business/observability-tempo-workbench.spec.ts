import { expect, test } from '@playwright/test';
import { adminCredentials, installClientSession, loginByApi, primeChineseLocale } from '../helpers/auth';

// Includes real fixture lifecycle, multiple rendered states and remote asset loading.
// Keep assertion deadlines unchanged while giving this scenario a bounded budget.
test.describe.configure({ timeout: 90_000 });

// Served by the remote acceptance build. Transport fixtures cover deterministic
// UI states; the real Tempo provider has a separate opt-in acceptance test.
for (const mode of ['light', 'dark'] as const) {
  test(`Tempo UTC range, detail, cancellation and revocation (${mode})`, async ({ page }, testInfo) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
    await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
    await page.route('**/business/observability/datasources', (route) => route.fulfill({ json: {
      code: 200, data: { total: 1, items: [{ id: 902, name: 'Tempo fixture', type: 'tempo', isDefault: true }] },
    } }));
    let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' = 'ready';
    let detailState: 'ready' | 'error' | 'forbidden' = 'ready';
    let release: (() => void) | undefined;
    await page.route('**/business/observability/datasources/902/query-fields', (route) => route.fulfill({json:{code:200,data:{version:1,sourceKind:'tempo',fields:[{field:'service',ops:['exists']}],durations:['5m','15m','1h','6h','24h'],maxLimit:1000,resultModes:['table']}}}));
    await page.route('**/business/observability/datasources/902/traces', async (route) => {
      expect(route.request().method()).toBe('POST');
      expect(new URL(route.request().url()).search).toBe('');
      const body = route.request().postDataJSON();
      expect(body.rawQuery).toBeUndefined(); expect(body.recipe.sourceKind).toBe('tempo');
      expect(body.recipe.limit).toBe(100);
      if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
      await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200, json:
        state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
          code: 200, data: { historyUnavailable:false, result:{ traces: state === 'empty' ? [] : [{ traceID: '11111111111111111111111111111111', rootServiceName: 'tempo-ui-fixture', rootTraceName: 'acceptance-operation', startTimeUnixNano: '1794009600000000000', durationMs: 12 }] } },
        },
      }).catch(() => { /* A stopped request no longer has a response consumer. */ });
    });
    await page.route('**/business/observability/datasources/902/traces/*', (route) => route.fulfill({
      status: detailState === 'forbidden' ? 403 : detailState === 'error' ? 502 : 200,
      json: detailState === 'ready' ? { code: 200, data: { fixture: 'safe-trace-detail' } } : { code: detailState === 'forbidden' ? 403 : 502, message: 'request.failed' },
    }));
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/business/observability/traces');
    // The initial route loads auth and lazy frontend chunks before the adapter.
    await expect(page.getByRole('combobox', {name:'编辑方式'})).toBeVisible({ timeout: 30_000 });
    await expect(page.getByPlaceholder(/TraceQL/)).toHaveCount(0);
    await expect(page.getByText('Tempo fixture', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
    await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
    await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop.png`), fullPage: true });
    await page.locator('input[type="datetime-local"]').first().fill(new Date(Date.now() - 20 * 60_000).toISOString().slice(0, 16));
    await expect(page.getByText('查询条件已改变，当前显示上一次链路结果；请重新运行。', { exact: true })).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop-stale.png`), fullPage: true });
    await page.getByRole('button', { name: '详情', exact: true }).click();
    const detail = page.getByRole('region', { name: /11111111111111111111111111111111/ });
    await expect(detail).toContainText('safe-trace-detail');
    await detail.focus(); await expect(detail).toBeFocused();
    await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-detail.png`), fullPage: true });
    await page.keyboard.press('Escape');
    await expect(detail).toHaveCount(0);
    await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile.png`), fullPage: true, animations: 'disabled' });
    state = 'slow';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect.poll(() => Boolean(release)).toBe(true);
    await page.getByRole('button', { name: '停止查询', exact: true }).click();
    await expect(page.getByText('查询已停止，已有链路来自上一次查询', { exact: true })).toBeVisible();
    release?.();
    state = 'forbidden';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
    await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
    await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile-forbidden.png`), fullPage: true });
    state = 'empty';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('暂无链路追踪结果。', { exact: true })).toBeVisible();
    state = 'error';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
    state = 'ready';
    await page.getByRole('button', { name: '重试', exact: true }).click();
    await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
    await page.setViewportSize({ width: 1440, height: 900 });
    detailState = 'error';
    await page.getByRole('button', { name: '详情', exact: true }).click();
    await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
    detailState = 'ready';
    await page.getByRole('button', { name: /重试/ }).click();
    await expect(detail).toContainText('safe-trace-detail');
    await page.keyboard.press('Escape');
    detailState = 'forbidden';
    await page.getByRole('button', { name: '详情', exact: true }).click();
    await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
    await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
    await expect(detail).toHaveCount(0);
  });
}
