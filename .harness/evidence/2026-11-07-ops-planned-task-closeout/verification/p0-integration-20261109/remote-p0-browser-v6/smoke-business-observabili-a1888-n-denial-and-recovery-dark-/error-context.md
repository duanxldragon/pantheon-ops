# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-workbench.spec.ts >> Loki UTC range, cancellation, denial and recovery (dark)
- Location: tests\smoke\business\observability-workbench.spec.ts:8:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: page.waitForTimeout: Test timeout of 30000ms exceeded.
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
                        - 'textbox "输入 LogQL，例如 {app=~\".+\"}" [ref=e165]': '{app="other"}'
                        - combobox [ref=e167] [cursor=pointer]:
                            - generic "Loki fixture" [ref=e168]:
                                - generic [ref=e169]:
                                    - textbox [aria-hidden]:
                                        - /placeholder: 数据源
                                    - generic [ref=e170]: Loki fixture
                        - button "清空筛选" [ref=e175] [cursor=pointer]
                        - generic [ref=e179]:
                            - button "搜索" [ref=e181] [cursor=pointer]
                            - button "跟随日志（轮询）" [ref=e185] [cursor=pointer]
                            - button "停止查询" [disabled] [ref=e187]
                - generic [ref=e189]:
                    - generic [ref=e190]:
                        - text: 开始时间（UTC）
                        - textbox "开始时间（UTC）" [ref=e191]: 2026-11-09T03:06
                    - generic [ref=e192]:
                        - text: 结束时间（UTC）
                        - textbox "结束时间（UTC）" [ref=e193]: 2026-11-09T03:21
                    - generic [ref=e194]:
                        - text: 最多日志条数
                        - combobox "最多日志条数" [ref=e195] [cursor=pointer]:
                            - generic "500" [ref=e196]:
                                - generic [ref=e197]:
                                    - textbox [aria-hidden]
                                    - generic [ref=e198]: "500"
                    - generic [ref=e203]: 单次范围最多 24 小时，响应最多 8 MiB；按所选条数截取结果
                - generic [ref=e206]:
                    - generic [ref=e207]: 结果来源：Loki fixture · UTC 2026-11-09T03:06:26.697Z 至 2026-11-09T03:21:26.697Z · 行上限 500
                    - generic [ref=e208]: 最近滚动窗口的轮询快照
                    - alert [ref=e210]:
                        - generic [ref=e214]: 查询条件已改变，当前显示上一次查询结果；请重新运行。
                    - generic [ref=e216]: 查询已停止，已有结果来自上一次查询
                    - generic [ref=e218]:
                        - generic [ref=e222]:
                            - tab "日志" [selected] [ref=e223] [cursor=pointer]
                            - tab "时间分桶" [ref=e225] [cursor=pointer]
                            - tab "字段统计" [ref=e227] [cursor=pointer]
                            - tab "原始行" [ref=e229] [cursor=pointer]
                        - tabpanel "日志" [ref=e234]:
                            - generic [ref=e239]:
                                - table [ref=e243]:
                                    - rowgroup [ref=e248]:
                                        - row [ref=e249]:
                                            - columnheader "时间" [ref=e250]
                                            - columnheader "标签" [ref=e252]
                                            - columnheader "内容" [ref=e254]
                                    - rowgroup [ref=e256]:
                                        - row [ref=e257]:
                                            - cell "11/7/2026, 8:00:00 AM" [ref=e258]
                                            - cell "app=fixture" [ref=e260]
                                            - cell "local-loki-result" [ref=e269]
                                - generic [ref=e274]:
                                    - generic [ref=e276]:
                                        - generic [ref=e277]: 总数
                                        - strong [ref=e278]: "1"
                                    - list [ref=e279]:
                                        - listitem "上一页" [ref=e280]: .
                                        - listitem "第 1 页" [ref=e283] [cursor=pointer]: "1"
                                        - listitem "下一页" [ref=e284]: .
                                    - generic "页码" [ref=e287]:
                                        - combobox [ref=e288] [cursor=pointer]:
                                            - generic "50 条/页" [ref=e289]:
                                                - generic [ref=e290]:
                                                    - textbox [aria-hidden]
                                                    - generic [ref=e291]: 50 条/页
        - generic [ref=e296]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
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
  34  |     await expect(page.getByText('Loki fixture', { exact: true })).toBeVisible();
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
  98  |     state = 'ready';
  99  |     await page.getByRole('button', { name: '跟随日志（轮询）', exact: true }).click();
  100 |     await expect(page.getByText('最近滚动窗口的轮询快照', { exact: true })).toBeVisible();
  101 |     await expect(page.getByText('每次请求完成后间隔 5 秒刷新最近 15 分钟。每次快照有条数上限，刷新间的日志可能遗漏。', { exact: true })).toBeVisible();
  102 |     const initialFollowCount = queryCount;
  103 |     await expect.poll(() => queryCount, { timeout: 10_000 }).toBeGreaterThan(initialFollowCount);
  104 |     await expect(page.getByRole('button', { name: '停止查询', exact: true })).toBeEnabled();
  105 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-following.png`), fullPage: true });
  106 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  107 |     await page.setViewportSize({ width: 1440, height: 900 });
  108 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-following.png`), fullPage: true });
  109 |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  110 |     await expect(page.getByText('查询已停止，已有结果来自上一次查询', { exact: true })).toBeVisible();
  111 |     const stoppedCount = queryCount;
> 112 |     await page.waitForTimeout(5200);
      |                ^ Error: page.waitForTimeout: Test timeout of 30000ms exceeded.
  113 |     expect(queryCount).toBe(stoppedCount);
  114 |     await page.setViewportSize({ width: 390, height: 844 });
  115 |     state = 'wide';
  116 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  117 |     await expect(page.getByText('另有 494 个标签；详情请查看原始行。', { exact: true })).toBeVisible();
  118 |     await expect(page.getByText('标签预览已截短；详情请查看原始行。', { exact: true })).toBeVisible();
  119 |     await expect(page.locator('.observability-log-labels .arco-tag')).toHaveCount(6);
  120 |     const labelLengths = await page.locator('.observability-log-labels__value').evaluateAll((elements) => elements.map((element) => element.textContent?.length ?? 0));
  121 |     expect(labelLengths.every((length) => length <= 81)).toBeTruthy();
  122 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  123 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-mobile-wide-labels.png`), fullPage: true });
  124 |     await page.setViewportSize({ width: 1440, height: 900 });
  125 |     await page.screenshot({ path: testInfo.outputPath(`loki-${mode}-desktop-wide-labels.png`), fullPage: true });
  126 |     await page.getByRole('tab', { name: '原始行', exact: true }).click();
  127 |     await expect(page.getByRole('region', { name: '原始行', exact: true })).toContainText('label499');
  128 |     await fs.writeFile(testInfo.outputPath('query-transport.json'), JSON.stringify({ queryCount, mode, provider: 'browser transport fixture', resultModes: ['logs', 'trend', 'fields', 'raw'], storedResults: false, labelFixtureCount: 500, renderedLabels: 6, maxPreviewCharacters: 81, follow: { transport: 'bounded range polling', intervalAfterCompletionMs: 5000, rollingWindowMinutes: 15, stoppedBetweenRequests: true } }, null, 2));
  129 |     await testInfo.attach('query-transport', { contentType: 'application/json', body: JSON.stringify({ queryCount, mode, provider: 'browser transport fixture' }) });
  130 |   });
  131 | }
  132 |
```
