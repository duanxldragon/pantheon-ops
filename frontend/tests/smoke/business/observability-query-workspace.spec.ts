import { expect, test } from '@playwright/test';
import { adminCredentials, installClientSession, loginByApi, primeChineseLocale } from '../helpers/auth';

// Real application/auth shell, synthetic HTTP provider/workspace responses.
// Real provider/IAM/tenant acceptance remains a separate deployment gate.
test.describe.configure({ timeout: 90_000 });
for (const mode of ['light', 'dark'] as const) {
  for (const sourceKind of ['loki', 'tempo', 'prometheus'] as const) {
    test(`query history restore requires Run: ${sourceKind} ${mode}`, async ({ page }, testInfo) => {
      const login = await loginByApi(page.request, adminCredentials);
      await installClientSession(page, login); await primeChineseLocale(page);
      await page.addInitScript((mode) => localStorage.setItem('pantheon_color_mode', mode), mode);
      const datasourceId = sourceKind === 'loki' ? 901 : sourceKind === 'tempo' ? 902 : 903;
      const recipe = { version: 1, sourceKind, rules: [{ field: sourceKind === 'loki' ? 'stream' : sourceKind === 'tempo' ? 'service' : 'availability', op: sourceKind === 'prometheus' ? 'instant' : 'exists' }], time: { mode: 'relative', duration: '15m' }, limit: sourceKind === 'loki' ? 500 : 100, resultMode: 'table' };
      const history = { id: 71, datasourceId, sourceKind, time: recipe.time, limit: recipe.limit, resultMode: 'table', outcome: 'success', durationMs: 12, createdAt: '2026-11-11T00:00:00Z', recipe, expressionUnavailable: false };
      const view = { id: 81, datasourceId, revision: 3, recipe, updatedAt: '2026-11-11T00:00:00Z' };
      const prefix = '**/business/observability';
      await page.route(`${prefix}/datasources`, (route) => route.fulfill({ json: { code: 200, data: { items: [{ id: datasourceId, name: 'Query workspace fixture', type: sourceKind, isDefault: true }], total: 1 } } }));
      if (sourceKind === 'prometheus') await page.route(`${prefix}/targets?*`, (route) => route.fulfill({ status: 403, json: { code: 403, message: 'request.failed' } }));
      await page.route(`${prefix}/datasources/${datasourceId}/query-fields`, (route) => route.fulfill({ json: { code: 200, data: { version: 1, sourceKind, fields: [{ field: recipe.rules[0].field, ops: [recipe.rules[0].op] }, ...(sourceKind === 'loki' ? [{ field: 'severity', ops: ['equals'], values: ['info', 'warning', 'error'] }] : [])], durations: ['5m', '15m', '1h', '6h', '24h'], maxLimit: sourceKind === 'loki' ? 5000 : 1000, resultModes: ['table'] } } }));
      await page.route(`${prefix}/query-history?*`, (route) => route.fulfill({ json: { code: 200, data: { items: [history], total: 1, catalogVersion: 1 } } }));
      await page.route(`${prefix}/query-history/71/restore`, (route) => route.fulfill({ json: { code: 200, data: { items: [history], total: 1, catalogVersion: 1 } } }));
      await page.route(`${prefix}/saved-views?*`, (route) => route.fulfill({ json: { code: 200, data: { items: [view], total: 1, catalogVersion: 1 } } }));
      await page.route(`${prefix}/saved-views/81`, (route) => { expect(route.request().postDataJSON().expectedRevision).toBe(3); return route.fulfill({ status: 409, json: { code: 409, message: 'request.failed' } }); });
      let queryCount = 0;
      const endpoint = sourceKind === 'loki' ? 'logs' : sourceKind === 'tempo' ? 'traces' : 'query';
      await page.route(`${prefix}/datasources/${datasourceId}/${endpoint}`, (route) => {
        queryCount++; expect(route.request().method()).toBe('POST'); expect(new URL(route.request().url()).search).toBe('');
        if (sourceKind === 'tempo') expect(route.request().postDataJSON().rawQuery).toBeUndefined();
        const result = sourceKind === 'loki' ? { status: 'success', data: { resultType: 'streams', result: [{ stream: { app: 'payments' }, values: [['1794009600000000000', 'query-workspace-fixture-result']] }] } } : sourceKind === 'tempo' ? { traces: [{ traceID: '11111111111111111111111111111111', rootServiceName: 'query-workspace-fixture-result' }] } : { status: 'success', warnings: ['synthetic-private-provider-warning'], data: { resultType: 'vector', result: [{ metric: { job: 'query-workspace-fixture-result' }, value: [1794009600, '1'] }] } };
        return route.fulfill({ json: { code: 200, data: { result, historyUnavailable: true } } });
      });
      await page.setViewportSize({ width: 1440, height: 900 });
      await page.goto(`/business/observability/${sourceKind === 'loki' ? 'logs' : sourceKind === 'tempo' ? 'traces' : 'monitoring'}`);
      if (sourceKind === 'prometheus') await page.getByRole('tab', { name: '指标查询', exact: true }).click();
      await expect(page.getByRole('combobox', { name: '编辑方式' })).toBeEnabled({ timeout: 30_000 });
      if (sourceKind === 'loki') { await page.getByRole('combobox', { name: '编辑方式' }).click(); await page.getByText('目录条件', { exact: true }).last().click(); }
      expect(queryCount).toBe(0);
      const run = () => page.getByRole('button', { name: sourceKind === 'prometheus' ? '运行' : '搜索', exact: true }).click();
      await run(); await expect(page.getByText(/query-workspace-fixture-result/).first()).toBeVisible();
      if (sourceKind === 'prometheus') await expect(page.getByText('指标结果可能不完整（1 条提供方警告）。', { exact: true })).toBeVisible();
      expect(await page.locator('body').innerText()).not.toContain('synthetic-private-provider-warning');
      await expect(page.getByText('查询成功，但历史记录暂不可用。', { exact: true })).toBeVisible();
      await page.getByRole('button', { name: '历史与视图', exact: true }).click(); await expect(page.getByText('查询 #71', { exact: true })).toBeVisible();
      await page.screenshot({ path: testInfo.outputPath(`${sourceKind}-${mode}-desktop-history.png`), fullPage: true });
      await page.getByRole('button', { name: '恢复', exact: true }).click(); await expect(page.getByText('已恢复查询条件；请点击运行。原始表达式需要重新输入。', { exact: true })).toBeVisible();
      await expect(page.getByText(/query-workspace-fixture-result/)).toHaveCount(0); expect(queryCount).toBe(1);
      await run(); await expect(page.getByText(/query-workspace-fixture-result/).first()).toBeVisible(); expect(queryCount).toBe(2);
      await page.getByRole('button', { name: '历史与视图', exact: true }).click(); await page.getByRole('tab', { name: '私有视图', exact: true }).click(); await expect(page.getByText('视图 #81', { exact: true })).toBeVisible();
      await page.getByRole('button', { name: '替换条件', exact: true }).click(); await expect(page.getByText('视图已被修改，请刷新后重新选择。', { exact: true })).toBeVisible();
      await page.setViewportSize({ width: 390, height: 844 }); await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
      await page.screenshot({ path: testInfo.outputPath(`${sourceKind}-${mode}-mobile-view-conflict.png`), fullPage: true });
      await page.keyboard.press('Escape'); await expect(page.getByRole('dialog')).toHaveCount(0); expect(queryCount).toBe(2);
    });
  }
}
