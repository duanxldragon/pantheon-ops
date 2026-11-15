# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke\business\deploy\deploy-selection-preview.spec.ts >> selection and preview confirm/conflict/refresh (light)
- Location: tests\smoke\business\deploy\deploy-selection-preview.spec.ts:12:2

# Error details

```
Test timeout of 30000ms exceeded.
```

```
Error: page.goto: Test timeout of 30000ms exceeded.
Call log:
  - navigating to "http://127.0.0.1:5173/business/deploy/task/5", waiting until "load"

```

# Test source

```ts
  1   | import { expect, test, type APIResponse } from '@playwright/test';
  2   | import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
  3   | import { startReviewedTask } from './reviewed-start';
  4   | import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';
  5   | async function success<T>(response: APIResponse): Promise<T> {
  6   |  const body = await response.json();
  7   |  expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
  8   |  expect(body.code, body.message).toBe(200);
  9   |  return body.data as T;
  10  | }
  11  | for (const mode of ['light', 'dark'] as const) {
  12  |  test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  13  |   const login = await loginByApi(page.request, adminCredentials);
  14  |   await installClientSession(page, login); await primeChineseLocale(page);
  15  |   await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  16  |   const headers = apiRequestHeaders(login);
  17  |   const token = `selection-preview-${mode}-${Date.now()}`;
  18  |   let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0;
  19  |   let failed = false;
  20  |   try {
  21  |    scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  22  |    const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
  23  |    hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
  24  |    packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
  25  |    templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  26  |    const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
  27  |    const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
  28  |    taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, templateId, templateParams: { serviceName: 'reviewed-service', port: '8080', rootPassword: '{{secret.password}}', custom: 'unrestricted-config-fixture' }, secretParams: { password: 'local-template-secret-fixture' }, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
  29  |    // Public execution cannot bypass review.
  30  |    const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
  31  |    expect((await missing.json()).code).not.toBe(200);
  32  |    await page.setViewportSize({ width: 1440, height: 900 });
  33  |    await page.goto(`/business/deploy/task?editId=${taskId}`);
  34  |    const selector = page.locator('.deploy-target-selector');
  35  |    await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
  36  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
  37  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  38  |    await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
  39  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
  40  |    await page.setViewportSize({ width: 390, height: 844 });
  41  |    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  42  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
  43  |    await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
  44  |    await selector.getByRole('button', { name: '解析并核对目标' }).click();
  45  |    await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
  46  |    await expect(selector.getByRole('status')).toContainText('去重 1');
  47  |    await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
  48  |    await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
> 49  |    await page.goto(`/business/deploy/task/${taskId}`);
      |               ^ Error: page.goto: Test timeout of 30000ms exceeded.
  50  |    let starts = 0;
  51  |    page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
  52  |    await page.getByRole('button', { name: '启动', exact: true }).click();
  53  |    const modal = page.locator('.arco-modal:visible').last();
  54  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  55  |    expect(starts).toBe(0);
  56  |    const plan = modal.getByRole('region', { name: '已冻结的执行配置' });
  57  |    await expect(plan).toContainText(`${token}-template`);
  58  |    await expect(plan).toContainText('reviewed-service');
  59  |    await expect(plan).toContainText('8080');
  60  |    await expect(plan).toContainText('当前参数值尚未测量');
  61  |    await expect(plan).toContainText('失败处理：停止执行');
  62  |    const stepToggle = plan.locator('.arco-collapse-item-header').first();
  63  |    await stepToggle.focus();
  64  |    await page.keyboard.press('Enter');
  65  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeHidden();
  66  |    await page.keyboard.press('Enter');
  67  |    await expect(plan.getByText('reviewed-service', { exact: true })).toBeVisible();
  68  |    await expect(plan).not.toContainText('local-template-secret-fixture');
  69  |    await expect(plan).not.toContainText('unrestricted-config-fixture');
  70  |    await plan.scrollIntoViewIfNeeded();
  71  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-mobile.png`) });
  72  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
  73  |    await page.setViewportSize({ width: 1440, height: 900 });
  74  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop.png`) });
  75  |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
  76  |    await page.emulateMedia({ reducedMotion: 'reduce' });
  77  |    await expect.poll(() => page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
  78  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-reduced-motion.png`) });
  79  |    await page.evaluate(() => { document.documentElement.style.zoom = '2'; });
  80  |    await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-css-zoom200.png`) });
  81  |    await page.evaluate(() => { document.documentElement.style.zoom = ''; });
  82  |    // These interceptions prove UI recovery states, not a real role revocation.
  83  |    const previewUrl = `**/business/deploy/tasks/${taskId}/preview`;
  84  |    for (const scenario of [{ status: 403, message: 'permission.denied', text: '无权查看或执行此任务', label: 'forbidden' }, { status: 503, message: 'unavailable', text: '预览加载失败，请重试', label: 'error' }]) {
  85  |     await modal.getByRole('button', { name: '取消', exact: true }).click();
  86  |     await page.route(previewUrl, (route) => route.fulfill({ status: scenario.status, contentType: 'application/json', body: JSON.stringify({ code: scenario.status, message: scenario.message, data: null }) }));
  87  |     await page.getByRole('button', { name: '启动', exact: true }).click();
  88  |     await expect(modal.getByText(scenario.text, { exact: true })).toBeVisible();
  89  |     await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  90  |     await expect(modal.getByRole('region', { name: '已冻结的执行配置' })).toHaveCount(0);
  91  |     await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-${scenario.label}.png`), fullPage: true });
  92  |     await page.unroute(previewUrl);
  93  |    }
  94  |    await modal.getByRole('button', { name: '刷新预览' }).click();
  95  |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  96  |    // Change a persisted fact after review; the running state must remain untouched.
  97  |    await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
  98  |    await modal.getByRole('button', { name: '确认并执行' }).click();
  99  |    await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
  100 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  101 |    const current = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  102 |    expect(current.status).toBe('draft');
  103 |    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
  104 |    await modal.getByRole('button', { name: '刷新预览' }).click();
  105 |    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  106 |    await modal.getByRole('button', { name: '确认并执行' }).focus();
  107 |    await page.keyboard.press('Enter');
  108 |    await expect(modal).toBeHidden();
  109 |    expect(starts).toBe(2);
  110 |    const started = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
  111 |    expect(started.status).toBe('running');
  112 |   } catch (error) { failed = true; throw error; }
  113 |   finally {
  114 |    const errors: string[] = [];
  115 |    try {
  116 |     const cleanupHeaders = await verifiedApiHeaders(page.request, login);
  117 |     if (taskId) {
  118 |      const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
  119 |      const current = await response.json();
  120 |      if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders, data: { stateVersion: current.data?.stateVersion } }));
  121 |     }
  122 |     for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
  123 |      if (!id) continue;
  124 |      try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
  125 |      catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
  126 |     }
  127 |    } catch (error) { errors.push(String(error)); }
  128 |    await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, templateId, errors }) });
  129 |    if (!failed) expect(errors).toEqual([]);
  130 |   }
  131 |  });
  132 | }
  133 |
  134 | test('rollback draft uses frozen definition and its own reviewed start', async ({ page }, testInfo) => {
  135 |  const login = await loginByApi(page.request, adminCredentials);
  136 |  const headers = apiRequestHeaders(login);
  137 |  const token = `frozen-rollback-${Date.now()}`;
  138 |  let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0, rollbackId = 0;
  139 |  let failed = false;
  140 |  try {
  141 |   scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  142 |   hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip: `10.248.${Math.floor(Math.random() * 200) + 1}.91`, os: 'linux', businessScopeId: scopeId } }))).id;
  143 |   packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo frozen-install', uninstallCommand: 'echo frozen-uninstall', status: 'enabled' } }))).id;
  144 |   templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  145 |   taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, templateId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual', templateParams: { serviceName: 'frozen-service', rootPassword: '{{secret.password}}' }, secretParams: { password: 'rollback-private-fixture' } } }))).id;
  146 |   const started = await success<{hosts: Array<{id: number}>}>(await startReviewedTask(page.request, taskId, { headers }));
  147 |   await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, { headers, data: { status: 'success', stdout: 'local manual fixture' } }));
  148 |   await success(await page.request.put(`${apiBaseUrl}/business/deploy/templates/${templateId}`, { headers, data: { status: 'disabled' } }));
  149 |   await success(await page.request.put(`${apiBaseUrl}/business/deploy/packages/${packageId}`, { headers, data: { status: 'disabled' } }));
```
