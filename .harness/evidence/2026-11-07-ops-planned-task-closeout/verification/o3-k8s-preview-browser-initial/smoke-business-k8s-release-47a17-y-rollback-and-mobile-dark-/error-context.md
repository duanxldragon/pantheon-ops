# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\k8s\release-preview.spec.ts >> K8s image preview conflict, same-request retry, rollback and mobile (dark)
- Location: tests\smoke\business\k8s\release-preview.spec.ts:7:2

# Error details

```
Error: locator.click: Error: strict mode violation: getByRole('dialog').getByRole('button', { name: '取消', exact: true }) resolved to 2 elements:
    1) <button type="button" class="arco-btn arco-btn-secondary arco-btn-size-default arco-btn-shape-square">…</button> aka locator('form').getByRole('button', { name: '取消' })
    2) <button type="button" class="arco-btn arco-btn-secondary arco-btn-size-default arco-btn-shape-square">…</button> aka getByRole('button', { name: '取消' }).nth(1)

Call log:
  - waiting for getByRole('dialog').getByRole('button', { name: '取消', exact: true })

```

# Page snapshot

```yaml
- generic [ref=e1]:
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
                        - listitem [ref=e103]: Kubernetes
                        - listitem [ref=e107]: 应用发布
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
                - tab "应用发布" [selected] [ref=e136] [cursor=pointer]
            - main [ref=e138]:
                - generic [ref=e141]:
                    - generic [ref=e143]:
                        - generic [ref=e144]:
                            - generic [ref=e145]: 运维平台 / 应用发布
                            - generic [ref=e146]: 应用发布
                        - generic [ref=e148]:
                            - generic [ref=e149]:
                                - generic [ref=e150]: 发布总数
                                - strong [ref=e151]: "1"
                            - generic [ref=e152]:
                                - generic [ref=e153]: 成功
                                - strong [ref=e154]: "1"
                            - generic [ref=e155]:
                                - generic [ref=e156]: 失败
                                - strong [ref=e157]: "0"
                    - button "创建发布" [ref=e162] [cursor=pointer]
                    - search "筛选" [ref=e166]:
                        - generic [ref=e168]:
                            - textbox "输入命名空间" [ref=e175]
                            - combobox [ref=e177] [cursor=pointer]:
                                - generic [ref=e179]:
                                    - textbox "集群"
                                    - generic [ref=e180]: 集群
                    - generic [ref=e191]:
                        - table [ref=e195]:
                            - rowgroup [ref=e205]:
                                - row [ref=e206]:
                                    - columnheader "发布名称" [ref=e207]
                                    - columnheader "命名空间" [ref=e209]
                                    - columnheader "工作负载" [ref=e211]
                                    - columnheader "发布前镜像" [ref=e213]
                                    - columnheader "发布后镜像" [ref=e215]
                                    - columnheader "状态" [ref=e217]
                                    - columnheader "发布时间" [ref=e219]
                                    - columnheader "操作" [ref=e221]
                            - rowgroup [ref=e223]:
                                - row [ref=e224]:
                                    - cell "image-release-fixture" [ref=e225]
                                    - cell "default" [ref=e227]
                                    - cell "api" [ref=e229]
                                    - cell "image:v1" [ref=e231]
                                    - cell "image:v2" [ref=e233]
                                    - cell "已成功" [ref=e238]
                                    - cell "2026-11-08T00:00:00Z" [ref=e243]
                                    - cell [ref=e245]:
                                        - button "回滚" [ref=e250] [cursor=pointer]
                        - generic [ref=e256]:
                            - generic [ref=e258]:
                                - generic [ref=e259]: 总数
                                - strong [ref=e260]: "1"
                            - list [ref=e261]:
                                - listitem "上一页" [ref=e262]: .
                                - listitem "第 1 页" [ref=e265] [cursor=pointer]: "1"
                                - listitem "下一页" [ref=e266]: .
                            - generic "页码" [ref=e269]:
                                - combobox [ref=e270] [cursor=pointer]:
                                    - generic "20 条/页" [ref=e271]:
                                        - generic [ref=e272]:
                                            - textbox [aria-hidden]
                                            - generic [ref=e273]: 20 条/页
            - generic [ref=e278]: Pantheon Platform · Enterprise Admin Foundation
    - generic:
        - generic:
            - alert [ref=e279]: 发布目标或意图已变化，请重新预览。
            - alert [ref=e283]: 网络异常，请检查连接后重试 (Network Error)
            - alert [ref=e287]: 网络异常，请检查连接后重试 (canceled)
    - dialog "预览发布变更" [ref=e293]:
        - generic [ref=e294]:
            - generic [ref=e295]: 预览发布变更
            - generic [ref=e298]:
                - generic [ref=e299]:
                    - generic [ref=e300]: 环境：non-production · default / api · 1 个工作负载
                    - region "镜像变更" [ref=e301]:
                        - generic [ref=e302]:
                            - generic [ref=e303]: 镜像变更
                            - button "展开未变化项" [active] [ref=e304] [cursor=pointer]
                        - generic [ref=e305]:
                            - generic [ref=e306]:
                                - alert [ref=e308]:
                                    - generic [ref=e312]: 本次仅修改指定容器镜像，不包含配置、资源或数据变更。
                                - alert [ref=e315]:
                                    - generic [ref=e319]: 执行后需要等待 Kubernetes 发布健康检查，镜像变更不保证应用兼容性。
                                - alert [ref=e322]:
                                    - generic [ref=e326]: 回滚只能恢复记录的镜像，不能恢复配置或撤销数据变更。
                            - generic [ref=e328]:
                                - generic [ref=e329]:
                                    - generic [ref=e330]: 修改
                                    - text: 目标镜像
                                - generic "当前镜像" [ref=e332]: image:v1
                                - generic "目标镜像" [ref=e333]: image:v3
                    - generic [ref=e334]: 可创建恢复当前镜像的发布意图；配置和数据需单独评估。
                - generic [ref=e336]:
                    - button "取消" [ref=e338] [cursor=pointer]
                    - button "重新预览" [ref=e340] [cursor=pointer]
                    - button "确认并执行" [ref=e342] [cursor=pointer]
            - button "Close" [ref=e343] [cursor=pointer]
```

# Test source

```ts
  1  | import { expect, test } from '@playwright/test';
  2  | import fs from 'node:fs/promises';
  3  | import { adminCredentials, installClientSession, loginByApi, primeChineseLocale } from '../../helpers/auth';
  4  |
  5  | // Real local auth; K8s release transport is a browser fixture. No cluster is mutated.
  6  | for (const mode of ['light', 'dark'] as const) {
  7  |  test(`K8s image preview conflict, same-request retry, rollback and mobile (${mode})`, async ({ page }, testInfo) => {
  8  |   const login = await loginByApi(page.request, adminCredentials); await installClientSession(page, login); await primeChineseLocale(page);
  9  |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  10 |   const row = { id: 77, name: 'image-release-fixture', clusterId: 901, namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', imageBefore: 'image:v1', imageAfter: 'image:v2', status: 'succeeded', createdAt: '2026-11-08T00:00:00Z' };
  11 |   const requests: { operation: string; idempotencyKey: string; intentVersion?: string; image?: string }[] = [];
  12 |   let stage: 'conflict' | 'uncertain' | 'ready' | 'forbidden' = 'conflict';
  13 |   await page.route('**/business/k8s/clusters?*', (route) => route.fulfill({ json: { code: 200, data: { items: [{ id: 901, name: 'Local cluster fixture' }], total: 1 } } }));
  14 |   await page.route('**/business/k8s/releases**', async (route) => {
  15 |    const url = new URL(route.request().url()); const path = url.pathname;
  16 |    if (route.request().method() === 'GET') { await route.fulfill({ json: { code: 200, data: { items: [row], total: 1 } } }); return; }
  17 |    const body = route.request().postDataJSON() as { idempotencyKey: string; intentVersion?: string; image?: string };
  18 |    expect(body.idempotencyKey).toBeTruthy(); requests.push({ operation: path, ...body });
  19 |    if (path.endsWith('/preview')) {
  20 |     if (stage === 'forbidden') { await route.fulfill({ status: 403, json: { code: 403, message: 'permission.denied' } }); return; }
  21 |     await route.fulfill({ json: { code: 200, data: { intentId: body.idempotencyKey, intentVersion: `ticket-${body.idempotencyKey}`, currentVersion: '15', targetVersion: 'image-intent-hash', targetCount: 1, environment: 'non-production', namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', changes: [{ id: 'image', field: 'image', kind: 'modified', before: path.includes('rollback') ? 'image:v2' : 'image:v1', after: body.image ?? 'image:v1' }], riskKeys: ['k8s.release.preview_image_only', 'k8s.release.preview_rollout_required', 'k8s.release.preview_rollback_limits'], rollbackAvailable: true, expiresAt: new Date(Date.now() + 600000).toISOString() } } }); return;
  22 |    }
  23 |    expect(body.intentVersion).toBe(`ticket-${body.idempotencyKey}`);
  24 |    if (stage === 'conflict') { await route.fulfill({ json: { code: 409, message: 'k8s.release.preview_conflict' } }); return; }
  25 |    if (stage === 'uncertain') { await route.abort('failed'); return; }
  26 |    await route.fulfill({ json: { code: 200, data: row } });
  27 |   });
  28 |   await page.setViewportSize({ width: 1440, height: 900 }); await page.goto('/business/k8s/release');
  29 |   await expect(page.getByText('image-release-fixture', { exact: true })).toBeVisible();
  30 |   await page.getByRole('button', { name: '回滚', exact: true }).click();
  31 |   let dialog = page.getByRole('dialog'); await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  32 |   await expect(dialog.getByText('回滚只能恢复记录的镜像，不能恢复配置或撤销数据变更。')).toBeVisible();
  33 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-desktop-review.png`), fullPage: true });
  34 |   await dialog.getByRole('button', { name: '确认并执行' }).click();
  35 |   await expect(dialog.getByText('目标或意图已变化，请重新预览。')).toBeVisible();
  36 |   await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  37 |   await page.setViewportSize({ width: 390, height: 844 });
  38 |   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  39 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-conflict.png`), fullPage: true });
  40 |   stage = 'uncertain'; await dialog.getByRole('button', { name: '重新预览' }).click();
  41 |   await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  42 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-review.png`), fullPage: true });
  43 |   await dialog.getByRole('button', { name: '确认并执行' }).click();
  44 |   await expect(dialog.getByRole('button', { name: '重试同一请求' })).toBeVisible();
  45 |   await expect(dialog.getByRole('button', { name: '重新预览' })).toBeDisabled();
  46 |   stage = 'ready'; await dialog.getByRole('button', { name: '重试同一请求' }).click();
  47 |   await expect(dialog.getByText('发布意图已受理，请查看列表中的执行状态。')).toBeVisible();
  48 |   const submits = requests.filter((request) => !request.operation.endsWith('/preview'));
  49 |   expect(submits[2]).toEqual(submits[1]); expect(submits[1].idempotencyKey).not.toBe(submits[0].idempotencyKey);
  50 |   await dialog.getByRole('button', { name: '关闭', exact: true }).click(); await expect(dialog).toBeHidden();
  51 |   await page.setViewportSize({ width: 1440, height: 900 });
  52 |   await page.getByRole('button', { name: '创建发布', exact: true }).click(); dialog = page.getByRole('dialog');
  53 |   await dialog.getByPlaceholder('请输入发布名称', { exact: true }).fill('create-image-fixture');
  54 |   await dialog.locator('.arco-select').first().click(); await page.getByText('Local cluster fixture', { exact: true }).last().click();
  55 |   await dialog.getByPlaceholder('输入命名空间', { exact: true }).fill('default');
  56 |   await dialog.locator('.arco-select').nth(1).click(); await page.getByText('deployment', { exact: true }).last().click();
  57 |   await dialog.getByPlaceholder('输入工作负载名称', { exact: true }).fill('api');
  58 |   await dialog.getByPlaceholder('如 nginx:1.25', { exact: true }).fill('image:v3');
  59 |   await dialog.getByRole('button', { name: '预览发布变更', exact: true }).click();
  60 |   await expect(page.getByRole('dialog').getByText('image:v3', { exact: true })).toBeVisible();
> 61 |   await page.getByRole('dialog').getByRole('button', { name: '取消', exact: true }).click();
     |                                                                                   ^ Error: locator.click: Error: strict mode violation: getByRole('dialog').getByRole('button', { name: '取消', exact: true }) resolved to 2 elements:
  62 |   await expect(page.getByRole('dialog').getByPlaceholder('请输入发布名称')).toHaveValue('create-image-fixture');
  63 |   await page.getByRole('dialog').getByRole('button', { name: '取消', exact: true }).click();
  64 |   stage = 'forbidden'; await page.getByRole('button', { name: '回滚', exact: true }).click();
  65 |   await expect(page.getByRole('dialog').getByText('没有发布预览权限')).toBeVisible();
  66 |   await expect(page.getByRole('dialog').getByRole('button', { name: '确认并执行' })).toBeDisabled();
  67 |   await fs.writeFile(testInfo.outputPath('release-preview-transport.json'), JSON.stringify({ provider: 'browser transport fixture', realLocalAuth: true, realK8sMutation: false, ownedBusinessFixtures: 0, requests }, null, 2));
  68 |  });
  69 | }
  70 |
```
