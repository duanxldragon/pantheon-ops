import { expect, test } from '@playwright/test';
import fs from 'node:fs/promises';
import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';

// Includes real fixture lifecycle, multiple rendered states and remote asset loading.
// Keep assertion deadlines unchanged while giving this scenario a bounded budget.
test.describe.configure({ timeout: 90_000 });

// Transport fixtures exercise browser states; provider cancellation/budgets are
// separately verified against a real local HTTP server and isolated MySQL.
for (const mode of ['light', 'dark'] as const) {
  test(`Loki UTC range, cancellation, denial and recovery (${mode})`, async ({ page }, testInfo) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
    await page.route('**/business/observability/datasources', async (route) => {
      await route.fulfill({ json: { code: 200, data: { total: 1, items: [{ id: 901, name: 'Loki fixture', type: 'loki', isDefault: true }] } } });
    });
    let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' | 'warning' | 'badResponse' | 'wide' = 'ready';
    let release: (() => void) | undefined;
    let queryCount = 0;
    await page.route('**/business/observability/datasources/901/query-fields', (route) => route.fulfill({json:{code:200,data:{version:1,sourceKind:'loki',fields:[{field:'stream',ops:['exists']},{field:'severity',ops:['equals'],values:['info','warning','error']}],durations:['5m','15m','1h','6h','24h'],maxLimit:5000,resultModes:['table']}}}));
    await page.route('**/business/observability/datasources/901/logs', async (route) => {
      queryCount++;
      expect(route.request().method()).toBe('POST');
      expect(new URL(route.request().url()).search).toBe('');
      const body = route.request().postDataJSON();
      if (body.historyMode === 'coalesced') expect(body.time).toEqual({mode:'relative',duration:'15m'});
      else { expect(body.time.start).toMatch(/Z$/); expect(body.time.end).toMatch(/Z$/); }
      expect(body.limit).toBe(500);
      if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
      await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200,
        json: state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
          code: 200, data: { historyUnavailable:false,historyId:41,result:{ status: state === 'badResponse' ? 'error' : 'success', warnings: state === 'warning' ? ['token=private-warning'] : undefined, data: { resultType: 'streams', result: state === 'empty' ? [] : [
            { stream: state === 'wide' ? Object.fromEntries(Array.from({ length: 500 }, (_, index) => [`label${index}`, index === 0 ? 'wide-value-'.repeat(1000) : `value-${index}`])) : { app: 'fixture' }, values: [['1794009600000000000', 'local-loki-result']] },
          ] } } },
        } }).catch(() => { /* An aborted request has no response consumer. */ });
    });
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.goto('/business/observability/logs');
    await expect(page.getByText('Loki fixture', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
    await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop.png`), fullPage: true });
    await page.getByPlaceholder(/LogQL/).fill('{app="other"}');
    await expect(page.getByText('查询条件已改变，当前显示上一次查询结果；请重新运行。', { exact: true })).toBeVisible();
    await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-stale.png`), fullPage: true });
    state = 'warning'; await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('数据源返回 1 条警告，结果可能不完整。', { exact: true })).toBeVisible();
    await expect(page.getByText(/private-warning/)).toHaveCount(0);
    await expect(page.getByText('查询条件已改变，当前显示上一次查询结果；请重新运行。', { exact: true })).toHaveCount(0);
    await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-partial.png`), fullPage: true });
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.getByRole('tab', { name: '字段统计', exact: true }).click();
    await expect(page.getByText('app', { exact: true })).toBeVisible();
    await expect(page.getByText('统计仅覆盖本次返回的日志行，不代表全部匹配日志。', { exact: true })).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-fields.png`), fullPage: true });
    await page.getByRole('tab', { name: '时间分桶', exact: true }).click();
    await expect(page.getByText('2026-11-07T00:00:00.000Z', { exact: true })).toBeVisible();
    await page.getByRole('tab', { name: '原始行', exact: true }).click();
    await expect(page.getByRole('region', { name: '原始行', exact: true })).toContainText('local-loki-result');
    await page.getByRole('region', { name: '原始行', exact: true }).focus();
    await expect(page.getByRole('region', { name: '原始行', exact: true })).toBeFocused();
    await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-raw.png`), fullPage: true });
    await page.getByRole('tab', { name: '日志', exact: true }).click();
    await page.setViewportSize({ width: 1440, height: 900 });
    state = 'slow';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
    await expect.poll(() => Boolean(release)).toBe(true);
    await page.getByRole('button', { name: '停止查询', exact: true }).click();
    await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
    release?.();
    await expect(page.getByText(/网络异常.*canceled/)).toHaveCount(0);
    state = 'forbidden';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('没有日志查询权限', { exact: true })).toBeVisible();
    await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
    await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-forbidden.png`), fullPage: true });
    state = 'empty';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('暂无日志结果。', { exact: true })).toBeVisible();
    state = 'error';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
    state = 'badResponse';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
    await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
    state = 'ready';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-ready.png`), fullPage: true });
    state = 'ready';
    await page.getByRole('button', { name: '跟随日志（轮询）', exact: true }).click();
    await expect(page.getByText('最近滚动窗口的轮询快照', { exact: true })).toBeVisible();
    await expect(page.getByText('每次请求完成后间隔 5 秒刷新最近 15 分钟。每次快照有条数上限，刷新间的日志可能遗漏。', { exact: true })).toBeVisible();
    const initialFollowCount = queryCount;
    await expect.poll(() => queryCount, { timeout: 10_000 }).toBeGreaterThan(initialFollowCount);
    await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-following.png`), fullPage: true });
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-following.png`), fullPage: true });
    await page.getByRole('button', { name: '停止查询', exact: true }).click();
    await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
    const stoppedCount = queryCount;
    await page.waitForTimeout(5200);
    expect(queryCount).toBe(stoppedCount);
    await page.setViewportSize({ width: 390, height: 844 });
    state = 'wide';
    await page.getByRole('button', { name: '搜索', exact: true }).click();
    await expect(page.getByText('另有 494 个标签；详情请查看原始行。', { exact: true })).toBeVisible();
    await expect(page.getByText('标签预览已截短；详情请查看原始行。', { exact: true })).toBeVisible();
    await expect(page.locator('.observability-log-labels .arco-tag')).toHaveCount(6);
    const labelLengths = await page.locator('.observability-log-labels__value').evaluateAll((elements) => elements.map((element) => element.textContent?.length ?? 0));
    expect(labelLengths.every((length) => length <= 81)).toBeTruthy();
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-wide-labels.png`), fullPage: true });
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-wide-labels.png`), fullPage: true });
    await page.getByRole('tab', { name: '原始行', exact: true }).click();
    await expect(page.getByRole('region', { name: '原始行', exact: true })).toContainText('label499');
    await fs.writeFile(testInfo.outputPath('query-transport.json'), JSON.stringify({ queryCount, mode, provider: 'browser transport fixture', resultModes: ['logs', 'trend', 'fields', 'raw'], storedResults: false, labelFixtureCount: 500, renderedLabels: 6, maxPreviewCharacters: 81, follow: { transport: 'bounded range polling', intervalAfterCompletionMs: 5000, rollingWindowMinutes: 15, stoppedBetweenRequests: true } }, null, 2));
    await testInfo.attach('query-transport', { contentType: 'application/json', body: JSON.stringify({ queryCount, mode, provider: 'browser transport fixture' }) });
  });
}
