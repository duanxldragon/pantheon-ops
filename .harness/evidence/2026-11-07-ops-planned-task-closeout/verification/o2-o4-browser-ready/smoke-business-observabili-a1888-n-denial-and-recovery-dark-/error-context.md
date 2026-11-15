# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-workbench.spec.ts >> Loki UTC range, cancellation, denial and recovery (dark)
- Location: tests\smoke\business\observability-workbench.spec.ts:8:3

# Error details

```
Error: expect(locator).toContainText(expected) failed

Locator: getByLabel('原始行', { exact: true })
Expected substring: "local-loki-result"
Error: strict mode violation: getByLabel('原始行', { exact: true }) resolved to 2 elements:
    1) <div tabindex="0" role="tabpanel" id="arco-tabs-1-panel-3" aria-labelledby="arco-tabs-1-tab-3" class="arco-tabs-content-item arco-tabs-content-item-active">…</div> aka getByRole('tabpanel', { name: '原始行' })
    2) <pre aria-label="原始行" class="observability-log-results__raw">[↵{"id":"1794009600000000000-0","timestamp":"1794…</pre> aka getByText('[ {"id":"1794009600000000000-')

Call log:
  - Expect "toContainText" getByLabel('原始行', { exact: true }) with timeout 10000ms
  - waiting for getByLabel('原始行', { exact: true })

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
                        - button [ref=e13]
                    - generic [ref=e18] [cursor=pointer]:
                        - button
                    - menuitem [ref=e25] [cursor=pointer]:
                        - button [ref=e26]
                    - menuitem [ref=e33] [cursor=pointer]:
                        - button [ref=e34]
                    - generic [ref=e41] [cursor=pointer]:
                        - button
                    - generic [ref=e45] [cursor=pointer]:
                        - button
                    - generic [ref=e49] [cursor=pointer]:
                        - button
                    - generic [ref=e53] [cursor=pointer]:
                        - button
                    - generic [ref=e59] [cursor=pointer]:
                        - button
                    - generic [ref=e66] [cursor=pointer]:
                        - button
                    - generic [ref=e70] [cursor=pointer]:
                        - button
                    - generic [ref=e74] [cursor=pointer]:
                        - button
                    - menuitem [ref=e78] [cursor=pointer]:
                        - button [ref=e79]
                    - menuitem [ref=e86] [cursor=pointer]:
                        - button [ref=e87]
    - generic [ref=e91]:
        - generic [ref=e92]:
            - generic [ref=e93]:
                - button "切换导航" [ref=e94] [cursor=pointer]
                - list [ref=e98]:
                    - listitem [ref=e99]: 首页
                    - listitem [ref=e103]: 可观测中心
                    - listitem [ref=e107]: 日志中心
            - generic [ref=e108]:
                - button "平台功能台" [ref=e110]:
                    - generic [ref=e113]: 搜索菜单、页面、功能…
                    - generic [ref=e114]: Ctrl K
                - button "通知中心" [ref=e116] [cursor=pointer]
                - button "平台偏好" [ref=e120] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e125] [cursor=pointer]':
                    - generic [ref=e126]: A
                    - generic [ref=e128]:
                        - generic [ref=e129]: Administrator
                        - generic [ref=e130]: admin
        - tablist "已打开页面" [ref=e131]:
            - tab "工作台" [ref=e132] [cursor=pointer]
            - tab "日志中心" [selected] [ref=e136] [cursor=pointer]
        - main [ref=e138]:
            - generic [ref=e141]:
                - generic [ref=e143]:
                    - generic [ref=e144]:
                        - generic [ref=e145]: 运维平台 / 日志中心
                        - generic [ref=e146]: 日志中心
                    - generic [ref=e148]:
                        - generic [ref=e149]:
                            - generic [ref=e150]: 数据源
                            - strong [ref=e151]: "1"
                        - generic [ref=e152]:
                            - generic [ref=e153]: 命中数
                            - strong [ref=e154]: "1"
                - search "筛选" [ref=e156]:
                    - generic [ref=e158]:
                        - 'textbox "输入 LogQL，例如 {app=~\".+\"}" [ref=e165]': '{app=~".+"}'
                        - combobox [ref=e167] [cursor=pointer]:
                            - generic "Loki fixture" [ref=e168]:
                                - generic [ref=e169]:
                                    - textbox [aria-hidden]:
                                        - /placeholder: 数据源
                                    - generic [ref=e170]: Loki fixture
                        - button "清空筛选" [ref=e175] [cursor=pointer]
                        - generic [ref=e179]:
                            - button "搜索" [ref=e181] [cursor=pointer]
                            - button "停止查询" [disabled] [ref=e185]
                - generic [ref=e187]:
                    - generic [ref=e188]:
                        - text: 开始时间（UTC）
                        - textbox "开始时间（UTC）" [ref=e189]: 2026-11-08T00:20
                    - generic [ref=e190]:
                        - text: 结束时间（UTC）
                        - textbox "结束时间（UTC）" [ref=e191]: 2026-11-08T00:35
                    - generic [ref=e192]:
                        - text: 最多日志条数
                        - combobox "最多日志条数" [ref=e193] [cursor=pointer]:
                            - generic "500" [ref=e194]:
                                - generic [ref=e195]:
                                    - textbox [aria-hidden]
                                    - generic [ref=e196]: "500"
                    - generic [ref=e201]: 单次范围最多 24 小时，响应最多 8 MiB；按所选条数截取结果
                - generic [ref=e205]:
                    - generic [ref=e209]:
                        - tab "日志" [ref=e210] [cursor=pointer]
                        - tab "时间分桶" [ref=e212] [cursor=pointer]
                        - tab "字段统计" [ref=e214] [cursor=pointer]
                        - tab "原始行" [active] [selected] [ref=e216] [cursor=pointer]
                    - generic [ref=e220]:
                        - generic [ref=e225]:
                            - table [ref=e229]:
                                - rowgroup [ref=e234]:
                                    - row [ref=e235]:
                                        - columnheader [ref=e236]:
                                            - generic [ref=e237]: 时间
                                        - columnheader [ref=e238]:
                                            - generic [ref=e239]: 标签
                                        - columnheader [ref=e240]:
                                            - generic [ref=e241]: 内容
                                - rowgroup [ref=e242]:
                                    - row [ref=e243]:
                                        - cell [ref=e244]:
                                            - generic [ref=e245]: 11/7/2026, 8:00:00 AM
                                        - cell [ref=e246]:
                                            - generic [ref=e247]: app=fixture
                                        - cell [ref=e253]:
                                            - generic [ref=e254]: local-loki-result
                            - generic [ref=e258]:
                                - generic [ref=e260]:
                                    - generic [ref=e261]: 总数
                                    - strong [ref=e262]: "1"
                                - list [ref=e263]:
                                    - listitem [ref=e264]: .
                                    - listitem [ref=e267] [cursor=pointer]: "1"
                                    - listitem [ref=e268]: .
                                - combobox [ref=e272] [cursor=pointer]:
                                    - generic [ref=e274]:
                                        - textbox [aria-hidden]
                                        - generic [ref=e275]: 50 条/页
                        - generic [ref=e280]:
                            - paragraph [ref=e281]: 统计仅覆盖本次返回的日志行，不代表全部匹配日志。
                            - generic [ref=e285]:
                                - table [ref=e289]:
                                    - rowgroup [ref=e293]:
                                        - row [ref=e294]:
                                            - columnheader [ref=e295]:
                                                - generic [ref=e296]: 开始时间（UTC）
                                            - columnheader [ref=e297]:
                                                - generic [ref=e298]: 返回行数
                                    - rowgroup [ref=e299]:
                                        - row [ref=e300]:
                                            - cell [ref=e301]:
                                                - generic [ref=e302]: 2026-11-07T00:00:00.000Z
                                            - cell [ref=e303]:
                                                - generic [ref=e304]: "1"
                                - generic [ref=e308]:
                                    - generic [ref=e310]:
                                        - generic [ref=e311]: 总数
                                        - strong [ref=e312]: "1"
                                    - list [ref=e313]:
                                        - listitem [ref=e314]: .
                                        - listitem [ref=e317] [cursor=pointer]: "1"
                                        - listitem [ref=e318]: .
                                    - combobox [ref=e322] [cursor=pointer]:
                                        - generic [ref=e324]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e325]: 20 条/页
                        - generic [ref=e330]:
                            - paragraph [ref=e331]: 统计仅覆盖本次返回的日志行，不代表全部匹配日志。
                            - generic [ref=e335]:
                                - table [ref=e339]:
                                    - rowgroup [ref=e344]:
                                        - row [ref=e345]:
                                            - columnheader [ref=e346]:
                                                - generic [ref=e347]: 字段
                                            - columnheader [ref=e348]:
                                                - generic [ref=e349]: 返回行数
                                            - columnheader [ref=e350]:
                                                - generic [ref=e351]: 返回值去重数
                                    - rowgroup [ref=e352]:
                                        - row [ref=e353]:
                                            - cell [ref=e354]:
                                                - generic [ref=e355]: app
                                            - cell [ref=e356]:
                                                - generic [ref=e357]: "1"
                                            - cell [ref=e358]:
                                                - generic [ref=e359]: "1"
                                - generic [ref=e363]:
                                    - generic [ref=e365]:
                                        - generic [ref=e366]: 总数
                                        - strong [ref=e367]: "1"
                                    - list [ref=e368]:
                                        - listitem [ref=e369]: .
                                        - listitem [ref=e372] [cursor=pointer]: "1"
                                        - listitem [ref=e373]: .
                                    - combobox [ref=e377] [cursor=pointer]:
                                        - generic [ref=e379]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e380]: 20 条/页
                        - tabpanel "原始行" [ref=e385]:
                            - generic [ref=e386]:
                                - paragraph [ref=e387]: 只读展示返回日志行的 JSON，最多 256 KiB，不保存结果。
                                - generic "原始行" [ref=e388]: '[ {"id":"1794009600000000000-0","timestamp":"1794009600000000000","labels":{"app":"fixture"},"message":"local-loki-result"} ]'
        - generic [ref=e389]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import fs from 'node:fs/promises';
  3  | import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';
  4  |
  5  | // Transport fixtures exercise browser states; provider cancellation/budgets are
  6  | // separately verified against a real local HTTP server and isolated MySQL.
  7  | for (const mode of ['light', 'dark'] as const) {
  8  |   test(`Loki UTC range, cancellation, denial and recovery (${mode})`, async ({ page }, testInfo) => {
  9  |     const login = await loginByApi(page.request, adminCredentials);
  10 |     await installClientSession(page, login);
  11 |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  12 |     await page.route('**/business/observability/datasources', async (route) => {
  13 |       await route.fulfill({ json: { code: 200, data: { total: 1, items: [{ id: 901, name: 'Loki fixture', type: 'loki', isDefault: true }] } } });
  14 |     });
  15 |     let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' = 'ready';
  16 |     let release: (() => void) | undefined;
  17 |     let queryCount = 0;
  18 |     await page.route('**/business/observability/datasources/901/logs?*', async (route) => {
  19 |       queryCount++;
  20 |       const url = new URL(route.request().url());
  21 |       expect(url.searchParams.get('start')).toMatch(/Z$/);
  22 |       expect(url.searchParams.get('end')).toMatch(/Z$/);
  23 |       expect(Number(url.searchParams.get('limit'))).toBe(500);
  24 |       if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
  25 |       await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200,
  26 |         json: state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
  27 |           code: 200, data: { status: 'success', data: { resultType: 'streams', result: state === 'empty' ? [] : [
  28 |             { stream: { app: 'fixture' }, values: [['1794009600000000000', 'local-loki-result']] },
  29 |           ] } },
  30 |         } }).catch(() => { /* An aborted request has no response consumer. */ });
  31 |     });
  32 |     await page.setViewportSize({ width: 1440, height: 900 });
  33 |     await page.goto('/business/observability/logs');
  34 |     await expect(page.getByText('Loki fixture', { exact: true })).toBeVisible();
  35 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  36 |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  37 |     await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  38 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop.png`), fullPage: true });
  39 |     await page.getByRole('tab', { name: '字段统计', exact: true }).click();
  40 |     await expect(page.getByText('app', { exact: true })).toBeVisible();
  41 |     await expect(page.getByText('统计仅覆盖本次返回的日志行，不代表全部匹配日志。', { exact: true })).toBeVisible();
  42 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-fields.png`), fullPage: true });
  43 |     await page.getByRole('tab', { name: '时间分桶', exact: true }).click();
  44 |     await expect(page.getByText('2026-11-07T00:00:00.000Z', { exact: true })).toBeVisible();
  45 |     await page.getByRole('tab', { name: '原始行', exact: true }).click();
> 46 |     await expect(page.getByLabel('原始行', { exact: true })).toContainText('local-loki-result');
     |                                                           ^ Error: expect(locator).toContainText(expected) failed
  47 |     await page.setViewportSize({ width: 390, height: 844 });
  48 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  49 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-raw.png`), fullPage: true });
  50 |     await page.getByRole('tab', { name: '日志', exact: true }).click();
  51 |     await page.setViewportSize({ width: 1440, height: 900 });
  52 |     state = 'slow';
  53 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  54 |     await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
  55 |     await expect.poll(() => Boolean(release)).toBe(true);
  56 |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  57 |     await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
  58 |     release?.();
  59 |     await expect(page.getByText(/网络异常.*canceled/)).toHaveCount(0);
  60 |     state = 'forbidden';
  61 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  62 |     await expect(page.getByText('没有日志查询权限', { exact: true })).toBeVisible();
  63 |     await expect(page.getByText('local-loki-result', { exact: true })).toHaveCount(0);
  64 |     await page.setViewportSize({ width: 390, height: 844 });
  65 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  66 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-forbidden.png`), fullPage: true });
  67 |     state = 'empty';
  68 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  69 |     await expect(page.getByText('暂无日志结果。', { exact: true })).toBeVisible();
  70 |     state = 'error';
  71 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  72 |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  73 |     state = 'ready';
  74 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  75 |     await expect(page.getByText('local-loki-result', { exact: true })).toBeVisible();
  76 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-ready.png`), fullPage: true });
  77 |     await fs.writeFile(testInfo.outputPath('query-transport.json'), JSON.stringify({ queryCount, mode, provider: 'browser transport fixture', resultModes: ['logs', 'trend', 'fields', 'raw'], storedResults: false }, null, 2));
  78 |     await testInfo.attach('query-transport', { contentType: 'application/json', body: JSON.stringify({ queryCount, mode, provider: 'browser transport fixture' }) });
  79 |   });
  80 | }
  81 |
```
