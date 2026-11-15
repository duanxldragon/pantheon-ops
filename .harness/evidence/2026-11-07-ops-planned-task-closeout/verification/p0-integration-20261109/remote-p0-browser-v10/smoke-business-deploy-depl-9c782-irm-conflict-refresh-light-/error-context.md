# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (light)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:16:2

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('.deploy-target-selector').getByText('selection-preview-light-1794195270997 · 10.249.105.91', { exact: true }).first()
Expected: visible
Timeout: 10000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" locator('.deploy-target-selector').getByText('selection-preview-light-1794195270997 · 10.249.105.91', { exact: true }).first() with timeout 10000ms
  - waiting for locator('.deploy-target-selector').getByText('selection-preview-light-1794195270997 · 10.249.105.91', { exact: true }).first()

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
        - menuitem "软件组件":
            - button "软件组件"
        - menuitem "任务模板":
            - button "任务模板"
        - menuitem "部署任务":
            - button "部署任务"
        - button "Kubernetes"
        - button "可观测中心"
        - button "访问控制"
        - button "组织架构"
        - button "安全审计"
        - button "低代码平台"
        - button "平台配置"
        - menuitem "文件分发":
            - button "文件分发"
        - menuitem "ArgoCD 应用发布":
            - button "ArgoCD 应用发布"
- button "切换导航"
- list:
    - listitem: 首页
    - listitem: 安装部署
    - listitem: 部署任务
- button "平台功能台": 搜索菜单、页面、功能… Ctrl K
- button "通知中心"
- button "平台偏好"
- 'button "用户: Administrator"': A Administrator admin
- tablist "已打开页面":
    - tab "工作台"
    - tab "部署任务" [selected]
- main
- text: Pantheon Platform · Enterprise Admin Foundation
```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   | import { startReviewedTask } from './reviewed-start';
  4   | import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
  5   |
  6   | // Includes real fixture lifecycle, multiple rendered states and remote asset loading.
  7   | // Keep assertion deadlines unchanged while giving this scenario a bounded budget.
  8   | test.describe.configure({ timeout: 90_000 });
  9   | async function success<T>(response: APIResponse): Promise<T> {
  10  |  const body = await response.json();
  11  |  expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
  12  |  expect(body.code, body.message).toBe(200);
  13  |  return body.data as T;
  14  | }
  15  | for (const mode of ['light', 'dark'] as const) {
  16  |  test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  17  |   const login = await loginByApi(page.request, adminCredentials);
  18  |   await installClientSession(page, login); await primeChineseLocale(page);
  19  |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  20  |   const headers = apiRequestHeaders(login);
  21  |   const token = `selection-preview-${mode}-${Date.now()}`;
  22  |   let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0;
  23  |   let failed = false;
  24  |   try {
  25  |    scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  26  |    const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
  27  |    hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
  28  |    packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
  29  |    templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  30  |    const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
  31  |    const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
  32  |    taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, templateId, templateParams: { serviceName: 'reviewed-service', port: '8080', rootPassword: '{{secret.password}}', custom: 'unrestricted-config-fixture' }, secretParams: { password: 'local-template-secret-fixture' }, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
  33  |    // Public execution cannot bypass review.
  34  |    const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
  35  |    expect((await missing.json()).code).not.toBe(200);
  36  |    await page.setViewportSize({ width: 1440, height: 900 });
  37  |    await page.goto(`/business/deploy/task?editId=${taskId}`);
  38  |    const selector = page.locator('.deploy-target-selector');
> 39  |    await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
      |                                                                                  ^ Error: expect(locator).toBeVisible() failed
  40  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
  41  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  42  |    await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
  43  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
  44  |    await page.setViewportSize({ width: 390, height: 844 });
  45  |    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  46  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
  47  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
  48  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  49  |    await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
  50  |    await expect(selector.getByRole('status')).toContainText('去重 1');
  51  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
  52  |    await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
  53  |    await page.goto(`/business/deploy/task/${taskId}`);
  54  |    let starts = 0;
  55  |    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
  56  |    await page.getByRole('button', { name: '启动', exact: true }).click();
  57  |    const modal = page.locator('.arco-modal:visible').last();
  58  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  59  |    expect(starts).toBe(0);
  60  |    const plan = modal.getByRole('region', { name: '已冻结的执行配置' });
  61  |    await expect(plan).toContainText(`${token}-template`);
  62  |    await expect(plan).toContainText('reviewed-service');
  63  |    await expect(plan).toContainText('8080');
  64  |    await expect(plan).toContainText('当前参数值尚未测量');
  65  |    await expect(plan).toContainText('失败处理：停止执行');
  66  |    const stepToggle = plan.locator('.arco-collapse-item-header').first();
  67  |    await stepToggle.focus();
  68  |    await page.keyboard.press('Enter');
  69  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeHidden();
  70  |    await page.keyboard.press('Enter');
  71  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeVisible();
  72  |    await expect(plan).not.toContainText('local-template-secret-fixture');
  73  |    await expect(plan).not.toContainText('unrestricted-config-fixture');
  74  |    await plan.scrollIntoViewIfNeeded();
  75  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-mobile.png`) });
  76  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
  77  |    await page.setViewportSize({ width: 1440, height: 900 });
  78  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop.png`) });
  79  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
  80  |    await page.emulateMedia({ reducedMotion: 'reduce' });
  81  |    await expect.poll(() => page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  82  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-reduced-motion.png`) });
  83  |    await page.evaluate(() => { document.documentElement.style.zoom = '2'; });
  84  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-css-zoom200.png`) });
  85  |    await page.evaluate(() => { document.documentElement.style.zoom = ''; });
  86  |    // These interceptions prove UI recovery states, not a real role revocation.
  87  |    const previewUrl = `**/business/deploy/tasks/${taskId}/preview`;
  88  |    for (const scenario of [{ status: 403, message: 'permission.denied', text: '无权查看或执行此任务', label: 'forbidden' }, { status: 503, message: 'unavailable', text: '预览加载失败，请重试', label: 'error' }]) {
  89  |     await modal.getByRole('button', { name: '取消', exact: true }).click();
  90  |     await page.route(previewUrl, (route) => route.fulfill({ status: scenario.status, contentType: 'application/json', body: JSON.stringify({ code: scenario.status, message: scenario.message, data: null }) }));
  91  |     await page.getByRole('button', { name: '启动', exact: true }).click();
  92  |     await expect(modal.getByText(scenario.text, { exact: true })).toBeVisible();
  93  |     await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  94  |     await expect(modal.getByRole('region', { name: '已冻结的执行配置' })).toHaveCount(0);
  95  |     await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-${scenario.label}.png`), fullPage: true });
  96  |     await page.unroute(previewUrl);
  97  |    }
  98  |    await modal.getByRole('button', { name: '刷新预览' }).click();
  99  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  100 |    // Change a persisted fact after review; the running state must remain untouched.
  101 |    await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
  102 |    await modal.getByRole('button', { name: '确认并执行' }).click();
  103 |    await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
  104 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  105 |    const current = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  106 |    expect(current.status).toBe('draft');
  107 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
  108 |    await modal.getByRole('button', { name: '刷新预览' }).click();
  109 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  110 |    await modal.getByRole('button', { name: '确认并执行' }).focus();
  111 |    await page.keyboard.press('Enter');
  112 |    await expect(modal).toBeHidden();
  113 |    expect(starts).toBe(2);
  114 |    const started = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  115 |    expect(started.status).toBe('running');
  116 |   } catch (error) { failed = true; throw error; }
  117 |   finally {
  118 |    const errors: string[] = [];
  119 |    try {
  120 |     const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  121 |     if (taskId) {
  122 |      const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
  123 |      const current = await response.json();
  124 |      if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders, data: { stateVersion: current.data?.stateVersion } }));
  125 |     }
  126 |     for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  127 |      if (!id) continue;
  128 |      try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
  129 |      catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
  130 |     }
  131 |    } catch (error) { errors.push(String(error)); }
  132 |    await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, templateId, errors }) });
  133 |    if (!failed) expect(errors).toEqual([]);
  134 |   }
  135 |  });
  136 | }
  137 |
  138 | test('rollback draft uses frozen definition and its own reviewed start', async ({ page }, testInfo) => {
  139 |  const login = await loginByApi(page.request, adminCredentials);
```
