# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\k8s\release-preview.spec.ts >> K8s image preview conflict, same-request retry, rollback and mobile (light)
- Location: tests\smoke\business\k8s\release-preview.spec.ts:7:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('dialog', { name: '预览发布变更', exact: true }).getByText('目标或意图已变化，请重新预览。')
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" getByRole('dialog', { name: '预览发布变更', exact: true }).getByText('目标或意图已变化，请重新预览。') with timeout 10000ms
  - waiting for getByRole('dialog', { name: '预览发布变更', exact: true }).getByText('目标或意图已变化，请重新预览。')

```

```yaml
- complementary:
    - text: P Pantheon Base
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
        - menuitem "K8s 集群":
            - button "K8s 集群"
        - menuitem "工作负载":
            - button "工作负载"
        - menuitem "服务":
            - button "服务"
        - menuitem "应用发布":
            - button "应用发布"
        - menuitem "Ingress":
            - button "Ingress"
        - menuitem "存储管理":
            - button "存储管理"
        - menuitem "Velero 备份":
            - button "Velero 备份"
        - button "可观测中心"
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
    - listitem: Kubernetes
    - listitem: 应用发布
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A Administrator admin
- tablist "已打开页面":
    - tab "工作台"
    - tab "应用发布" [selected]
- main:
    - text: 运维平台 / 应用发布 应用发布 发布总数
    - strong: "1"
    - text: 成功
    - strong: "1"
    - text: 失败
    - strong: "0"
    - button "创建发布"
    - search "筛选":
        - textbox "输入命名空间"
        - combobox:
            - textbox "集群"
            - text: 集群
    - table:
        - rowgroup:
            - row "发布名称 命名空间 工作负载 发布前镜像 发布后镜像 状态 发布时间 操作":
                - columnheader "发布名称"
                - columnheader "命名空间"
                - columnheader "工作负载"
                - columnheader "发布前镜像"
                - columnheader "发布后镜像"
                - columnheader "状态"
                - columnheader "发布时间"
                - columnheader "操作"
        - rowgroup:
            - row "image-release-fixture default api image:v1 image:v2 已成功 2026-11-08T00:00:00Z 回滚":
                - cell "image-release-fixture"
                - cell "default"
                - cell "api"
                - cell "image:v1"
                - cell "image:v2"
                - cell "已成功"
                - cell "2026-11-08T00:00:00Z"
                - cell "回滚":
                    - button "回滚"
    - text: 总数
    - strong: "1"
    - list:
        - listitem "上一页": .
        - listitem "第 1 页": "1"
        - listitem "下一页": .
    - combobox: 20 条/页
- text: Pantheon Platform · Enterprise Admin Foundation
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
  10 |   await page.addInitScript(() => {
  11 |    const browser = window as typeof window & { canceledMessages: string[] };
  12 |    browser.canceledMessages = [];
  13 |    new MutationObserver(() => {
  14 |     for (const message of document.querySelectorAll('.arco-message')) {
  15 |      if (/canceled/i.test(message.textContent ?? '')) browser.canceledMessages.push(message.textContent ?? '');
  16 |     }
  17 |    }).observe(document, { subtree: true, childList: true, characterData: true });
  18 |   });
  19 |   const row = { id: 77, name: 'image-release-fixture', clusterId: 901, namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', imageBefore: 'image:v1', imageAfter: 'image:v2', status: 'succeeded', createdAt: '2026-11-08T00:00:00Z' };
  20 |   const requests: { operation: string; idempotencyKey: string; intentVersion?: string; image?: string }[] = [];
  21 |   let stage: 'conflict' | 'uncertain' | 'ready' | 'forbidden' = 'conflict';
  22 |   await page.route('**/business/k8s/clusters?*', (route) => route.fulfill({ json: { code: 200, data: { items: [{ id: 901, name: 'Local cluster fixture' }], total: 1 } } }));
  23 |   await page.route('**/business/k8s/releases**', async (route) => {
  24 |    const url = new URL(route.request().url()); const path = url.pathname;
  25 |    if (route.request().method() === 'GET') { await route.fulfill({ json: { code: 200, data: { items: [row], total: 1 } } }); return; }
  26 |    const body = route.request().postDataJSON() as { idempotencyKey: string; intentVersion?: string; image?: string };
  27 |    expect(body.idempotencyKey).toBeTruthy(); requests.push({ operation: path, ...body });
  28 |    if (path.endsWith('/preview')) {
  29 |     if (stage === 'forbidden') { await route.fulfill({ status: 403, json: { code: 403, message: 'permission.denied' } }); return; }
  30 |     await route.fulfill({ json: { code: 200, data: { intentId: body.idempotencyKey, intentVersion: `ticket-${body.idempotencyKey}`, currentVersion: '15', targetVersion: 'image-intent-hash', targetCount: 1, environment: 'non-production', namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', changes: [{ id: 'image', field: 'image', kind: 'modified', before: path.includes('rollback') ? 'image:v2' : 'image:v1', after: body.image ?? 'image:v1' }], riskKeys: ['k8s.release.preview_image_only', 'k8s.release.preview_rollout_required', 'k8s.release.preview_rollback_limits'], rollbackAvailable: true, expiresAt: new Date(Date.now() + 600000).toISOString() } } }); return;
  31 |    }
  32 |    expect(body.intentVersion).toBe(`ticket-${body.idempotencyKey}`);
  33 |    if (stage === 'conflict') { await route.fulfill({ json: { code: 409, message: 'k8s.release.preview_conflict' } }); return; }
  34 |    if (stage === 'uncertain') { await route.abort('failed'); return; }
  35 |    await route.fulfill({ json: { code: 200, data: row } });
  36 |   });
  37 |   await page.setViewportSize({ width: 1440, height: 900 }); await page.goto('/business/k8s/release');
  38 |   await expect(page.getByText('image-release-fixture', { exact: true })).toBeVisible();
  39 |   await page.getByRole('button', { name: '回滚', exact: true }).click();
  40 |   let dialog = page.getByRole('dialog', { name: '预览发布变更', exact: true }); await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  41 |   await expect(dialog.getByText('回滚只能恢复记录的镜像，不能恢复配置或撤销数据变更。')).toBeVisible();
  42 |   await dialog.getByRole('button', { name: '确认并执行' }).focus();
  43 |   await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeFocused();
  44 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-desktop-review.png`), fullPage: true });
  45 |   await dialog.getByRole('button', { name: '确认并执行' }).click();
> 46 |   await expect(dialog.getByText('目标或意图已变化，请重新预览。')).toBeVisible();
     |                                                     ^ Error: expect(locator).toBeVisible() failed
  47 |   await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  48 |   await page.setViewportSize({ width: 390, height: 844 });
  49 |   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  50 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-conflict.png`), fullPage: true });
  51 |   stage = 'uncertain'; await dialog.getByRole('button', { name: '重新预览' }).click();
  52 |   await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  53 |   await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-review.png`), fullPage: true });
  54 |   await dialog.getByRole('button', { name: '确认并执行' }).click();
  55 |   await expect(dialog.getByRole('button', { name: '重试同一请求' })).toBeVisible();
  56 |   await expect(dialog.getByRole('button', { name: '重新预览' })).toBeDisabled();
  57 |   stage = 'ready'; await dialog.getByRole('button', { name: '重试同一请求' }).click();
  58 |   await expect(dialog.getByText('发布意图已受理，请查看列表中的执行状态。')).toBeVisible();
  59 |   const submits = requests.filter((request) => !request.operation.endsWith('/preview'));
  60 |   expect(submits[2]).toEqual(submits[1]); expect(submits[1].idempotencyKey).not.toBe(submits[0].idempotencyKey);
  61 |   await dialog.getByRole('button', { name: '关闭', exact: true }).click(); await expect(dialog).toBeHidden();
  62 |   await page.setViewportSize({ width: 1440, height: 900 });
  63 |   await page.getByRole('button', { name: '创建发布', exact: true }).click(); dialog = page.getByRole('dialog', { name: '创建发布', exact: true });
  64 |   await dialog.getByPlaceholder('请输入发布名称', { exact: true }).fill('create-image-fixture');
  65 |   await dialog.locator('.arco-select').first().click(); await page.getByText('Local cluster fixture', { exact: true }).last().click();
  66 |   await dialog.getByPlaceholder('输入命名空间', { exact: true }).fill('default');
  67 |   await dialog.locator('.arco-select').nth(1).click(); await page.getByText('deployment', { exact: true }).last().click();
  68 |   await dialog.getByPlaceholder('输入工作负载名称', { exact: true }).fill('api');
  69 |   await dialog.getByPlaceholder('如 nginx:1.25', { exact: true }).fill('image:v3');
  70 |   await dialog.getByRole('button', { name: '预览发布变更', exact: true }).click();
  71 |   await expect(page.getByRole('dialog', { name: '预览发布变更', exact: true }).getByText('image:v3', { exact: true })).toBeVisible();
  72 |   await page.getByRole('dialog', { name: '预览发布变更', exact: true }).getByRole('button', { name: '取消', exact: true }).click();
  73 |   await expect(page.getByRole('dialog', { name: '创建发布', exact: true }).getByPlaceholder('请输入发布名称')).toHaveValue('create-image-fixture');
  74 |   await page.getByRole('dialog', { name: '创建发布', exact: true }).getByRole('button', { name: '取消', exact: true }).click();
  75 |   stage = 'forbidden'; await page.getByRole('button', { name: '回滚', exact: true }).click();
  76 |   await expect(page.getByRole('dialog').getByText('没有发布预览权限')).toBeVisible();
  77 |   await expect(page.getByRole('dialog').getByRole('button', { name: '确认并执行' })).toBeDisabled();
  78 |   expect(await page.evaluate(() => (window as typeof window & { canceledMessages: string[] }).canceledMessages)).toEqual([]);
  79 |   await fs.writeFile(testInfo.outputPath('release-preview-transport.json'), JSON.stringify({ provider: 'browser transport fixture', realLocalAuth: true, realK8sMutation: false, ownedBusinessFixtures: 0, requests }, null, 2));
  80 |  });
  81 | }
  82 |
```
