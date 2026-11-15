# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\observability-tempo-workbench.spec.ts >> Tempo UTC range, detail, cancellation and revocation (light)
- Location: tests\smoke\business\observability-tempo-workbench.spec.ts:7:3

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: locator.click: Test timeout of 30000ms exceeded.
Call log:
  - waiting for getByRole('button', { name: '查看', exact: true })

```

# Page snapshot

```yaml
- generic [ref=e3]:
    - complementary [ref=e4]:
        - generic [ref=e5]:
            - generic [ref=e6]:
                - generic [ref=e7]: P
                - generic [ref=e8]: Pantheon Base
            - menu [ref=e12]:
                - generic [ref=e13]:
                    - menuitem [ref=e14] [cursor=pointer]:
                        - button "工作台" [ref=e15]
                    - button "CMDB" [ref=e25] [cursor=pointer]
                    - menuitem [ref=e37] [cursor=pointer]:
                        - button "业务域" [ref=e38]
                    - menuitem [ref=e47] [cursor=pointer]:
                        - button "服务目录" [ref=e48]
                    - button "安装部署" [ref=e60] [cursor=pointer]
                    - button "Kubernetes" [ref=e72] [cursor=pointer]
                    - generic [ref=e81]:
                        - button "可观测中心" [ref=e84] [cursor=pointer]
                        - generic [ref=e93]:
                            - menuitem [ref=e94] [cursor=pointer]:
                                - button "监控中心" [ref=e97]
                            - menuitem [ref=e106] [cursor=pointer]:
                                - button "告警中心" [ref=e109]
                            - menuitem [ref=e115] [cursor=pointer]:
                                - button "日志中心" [ref=e118]
                            - menuitem [ref=e125] [cursor=pointer]:
                                - button "链路追踪" [ref=e128]
                    - button "访问控制" [ref=e137] [cursor=pointer]
                    - button "组织架构" [ref=e151] [cursor=pointer]
                    - button "安全审计" [ref=e166] [cursor=pointer]
                    - button "低代码平台" [ref=e178] [cursor=pointer]
                    - button "平台配置" [ref=e190] [cursor=pointer]
                    - menuitem [ref=e199] [cursor=pointer]:
                        - button "文件分发" [ref=e200]
                    - menuitem [ref=e209] [cursor=pointer]:
                        - button "ArgoCD" [ref=e210]
    - generic [ref=e216]:
        - generic [ref=e217]:
            - generic [ref=e218]:
                - button "切换导航" [ref=e219] [cursor=pointer]
                - list [ref=e223]:
                    - listitem [ref=e224]: 首页
                    - listitem [ref=e228]: 可观测中心
                    - listitem [ref=e232]: 链路追踪
            - generic [ref=e233]:
                - button "平台功能台" [ref=e235]:
                    - generic [ref=e238]: 搜索菜单、页面、功能…
                    - generic [ref=e239]: Ctrl K
                - button "通知中心" [ref=e241] [cursor=pointer]
                - button "平台偏好" [ref=e245] [cursor=pointer]
                - 'button "用户: Administrator" [ref=e250] [cursor=pointer]':
                    - generic [ref=e251]: A
                    - generic [ref=e253]:
                        - generic [ref=e254]: Administrator
                        - generic [ref=e255]: admin
        - tablist "已打开页面" [ref=e256]:
            - tab "工作台" [ref=e257] [cursor=pointer]
            - tab "链路追踪" [selected] [ref=e261] [cursor=pointer]
        - main [ref=e263]:
            - generic [ref=e266]:
                - generic [ref=e268]:
                    - generic [ref=e269]:
                        - generic [ref=e270]: 运维平台 / 链路追踪
                        - generic [ref=e271]: 链路追踪
                    - generic [ref=e273]:
                        - generic [ref=e274]:
                            - generic [ref=e275]: 数据源
                            - strong [ref=e276]: "1"
                        - generic [ref=e277]:
                            - generic [ref=e278]: 命中数
                            - strong [ref=e279]: "1"
                - search "筛选" [ref=e281]:
                    - generic [ref=e283]:
                        - 'textbox "输入 TraceQL，例如 { resource.service.name != nil }" [active] [ref=e290]': '{ resource.service.name = "changed" }'
                        - combobox "数据源" [ref=e292] [cursor=pointer]:
                            - generic "Tempo fixture" [ref=e293]:
                                - generic [ref=e294]:
                                    - textbox [aria-hidden]:
                                        - /placeholder: 数据源
                                    - generic [ref=e295]: Tempo fixture
                        - button "清空筛选" [ref=e300] [cursor=pointer]
                        - generic [ref=e304]:
                            - button "搜索" [ref=e306] [cursor=pointer]
                            - button "停止查询" [disabled] [ref=e310]
                - generic [ref=e312]:
                    - generic [ref=e313]:
                        - text: 开始时间（UTC）
                        - textbox "开始时间（UTC）" [ref=e314]: 2026-11-09T03:04
                    - generic [ref=e315]:
                        - text: 结束时间（UTC）
                        - textbox "结束时间（UTC）" [ref=e316]: 2026-11-09T03:19
                    - generic [ref=e317]: 单次范围最多 24 小时，响应最多 8 MiB，最多展示 100 条链路
                - generic [ref=e320]:
                    - generic [ref=e321]: 结果来源：Tempo fixture · UTC 2026-11-09T03:04:00.000Z 至 2026-11-09T03:19:00.000Z · 最多 100 条
                    - alert [ref=e322]:
                        - generic [ref=e326]: 查询条件已改变，当前显示上一次链路结果；请重新运行。
                    - generic [ref=e331]:
                        - table [ref=e335]:
                            - rowgroup [ref=e343]:
                                - row [ref=e344]:
                                    - columnheader "Trace ID" [ref=e345]
                                    - columnheader "服务" [ref=e347]
                                    - columnheader "操作" [ref=e349]
                                    - columnheader "开始时间" [ref=e351]
                                    - columnheader "持续时间" [ref=e353]
                                    - columnheader "操作" [ref=e355]
                            - rowgroup [ref=e357]:
                                - row [ref=e358]:
                                    - cell "11111111111111111111111111111111" [ref=e359]
                                    - cell "tempo-ui-fixture" [ref=e361]
                                    - cell "acceptance-operation" [ref=e363]
                                    - cell "11/7/2026, 8:00:00 AM" [ref=e365]
                                    - cell "12.00 ms" [ref=e367]
                                    - cell [ref=e369]:
                                        - button "common.view" [ref=e372] [cursor=pointer]
                        - generic [ref=e379]:
                            - generic [ref=e381]:
                                - generic [ref=e382]: 总数
                                - strong [ref=e383]: "1"
                            - list [ref=e384]:
                                - listitem "上一页" [ref=e385]: .
                                - listitem "第 1 页" [ref=e388] [cursor=pointer]: "1"
                                - listitem "下一页" [ref=e389]: .
                            - generic "页码" [ref=e392]:
                                - combobox [ref=e393] [cursor=pointer]:
                                    - generic "20 条/页" [ref=e394]:
                                        - generic [ref=e395]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e396]: 20 条/页
        - generic [ref=e401]: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import { adminCredentials, installClientSession, loginByApi } from '../helpers/auth';
  3  |
  4  | // Served by the remote acceptance build. Transport fixtures cover deterministic
  5  | // UI states; the real Tempo provider has a separate opt-in acceptance test.
  6  | for (const mode of ['light', 'dark'] as const) {
  7  |   test(`Tempo UTC range, detail, cancellation and revocation (${mode})`, async ({ page }, testInfo) => {
  8  |     const login = await loginByApi(page.request, adminCredentials);
  9  |     await installClientSession(page, login);
  10 |     await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  11 |     await page.route('**/business/observability/datasources', (route) => route.fulfill({ json: {
  12 |       code: 200, data: { total: 1, items: [{ id: 902, name: 'Tempo fixture', type: 'tempo', isDefault: true }] },
  13 |     } }));
  14 |     let state: 'ready' | 'empty' | 'error' | 'forbidden' | 'slow' = 'ready';
  15 |     let detailState: 'ready' | 'error' | 'forbidden' = 'ready';
  16 |     let release: (() => void) | undefined;
  17 |     await page.route('**/business/observability/datasources/902/traces?*', async (route) => {
  18 |       const url = new URL(route.request().url());
  19 |       expect(url.searchParams.get('start')).toMatch(/Z$/);
  20 |       expect(url.searchParams.get('end')).toMatch(/Z$/);
  21 |       expect(url.searchParams.get('limit')).toBe('100');
  22 |       if (state === 'slow') await new Promise<void>((resolve) => { release = resolve; });
  23 |       await route.fulfill({ status: state === 'forbidden' ? 403 : state === 'error' ? 502 : 200, json:
  24 |         state === 'forbidden' || state === 'error' ? { code: state === 'forbidden' ? 403 : 502, message: 'request.failed' } : {
  25 |           code: 200, data: { traces: state === 'empty' ? [] : [{ traceID: '11111111111111111111111111111111', rootServiceName: 'tempo-ui-fixture', rootTraceName: 'acceptance-operation', startTimeUnixNano: '1794009600000000000', durationMs: 12 }] },
  26 |         },
  27 |       }).catch(() => { /* A stopped request no longer has a response consumer. */ });
  28 |     });
  29 |     await page.route('**/business/observability/datasources/902/traces/*', (route) => route.fulfill({
  30 |       status: detailState === 'forbidden' ? 403 : detailState === 'error' ? 502 : 200,
  31 |       json: detailState === 'ready' ? { code: 200, data: { fixture: 'safe-trace-detail' } } : { code: detailState === 'forbidden' ? 403 : 502, message: 'request.failed' },
  32 |     }));
  33 |     await page.setViewportSize({ width: 1440, height: 900 });
  34 |     await page.goto('/business/observability/traces');
  35 |     await expect(page.getByText('Tempo fixture', { exact: true })).toBeVisible();
  36 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  37 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
  38 |     await expect(page.locator('html')).toHaveAttribute('data-color-mode', mode);
  39 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop.png`), fullPage: true });
  40 |     await page.getByPlaceholder(/TraceQL/).fill('{ resource.service.name = "changed" }');
  41 |     await expect(page.getByText('查询条件已改变，当前显示上一次链路结果；请重新运行。', { exact: true })).toBeVisible();
  42 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-desktop-stale.png`), fullPage: true });
> 43 |     await page.getByRole('button', { name: '查看', exact: true }).click();
     |                                                                 ^ Error: locator.click: Test timeout of 30000ms exceeded.
  44 |     const detail = page.getByRole('region', { name: /11111111111111111111111111111111/ });
  45 |     await expect(detail).toContainText('safe-trace-detail');
  46 |     await detail.focus(); await expect(detail).toBeFocused();
  47 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-detail.png`), fullPage: true });
  48 |     await page.keyboard.press('Escape');
  49 |     await expect(detail).toHaveCount(0);
  50 |     await page.setViewportSize({ width: 390, height: 844 });
  51 |     await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  52 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile.png`), fullPage: true });
  53 |     state = 'slow';
  54 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  55 |     await expect.poll(() => Boolean(release)).toBe(true);
  56 |     await page.getByRole('button', { name: '停止查询', exact: true }).click();
  57 |     await expect(page.getByText('查询已停止，已有链路来自上一次查询', { exact: true })).toBeVisible();
  58 |     release?.();
  59 |     state = 'forbidden';
  60 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  61 |     await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
  62 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
  63 |     await page.screenshot({ path: testInfo.outputPath(`tempo-${mode}-mobile-forbidden.png`), fullPage: true });
  64 |     state = 'empty';
  65 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  66 |     await expect(page.getByText('暂无链路追踪结果。', { exact: true })).toBeVisible();
  67 |     state = 'error';
  68 |     await page.getByRole('button', { name: '搜索', exact: true }).click();
  69 |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  70 |     state = 'ready';
  71 |     await page.getByRole('button', { name: '重试', exact: true }).click();
  72 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toBeVisible();
  73 |     await page.setViewportSize({ width: 1440, height: 900 });
  74 |     detailState = 'error';
  75 |     await page.getByRole('button', { name: '查看', exact: true }).click();
  76 |     await expect(page.getByRole('button', { name: /重试/ })).toBeVisible();
  77 |     detailState = 'ready';
  78 |     await page.getByRole('button', { name: /重试/ }).click();
  79 |     await expect(detail).toContainText('safe-trace-detail');
  80 |     await page.keyboard.press('Escape');
  81 |     detailState = 'forbidden';
  82 |     await page.getByRole('button', { name: '查看', exact: true }).click();
  83 |     await expect(page.getByText('没有链路追踪查询权限', { exact: true })).toBeVisible();
  84 |     await expect(page.getByText('tempo-ui-fixture', { exact: true })).toHaveCount(0);
  85 |     await expect(detail).toHaveCount(0);
  86 |   });
  87 | }
  88 |
```
