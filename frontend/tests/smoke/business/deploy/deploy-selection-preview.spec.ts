import { expect, test, type APIResponse } from '@playwright/test';
import { adminCredentials, apiBaseUrl, apiRequestHeaders, installClientSession, loginByApi, primeChineseLocale, verifiedApiHeaders } from '../../helpers/auth';
import { startReviewedTask } from './reviewed-start';
import type { DeployTargetResolution } from '../../../../src/modules/business/deploy/api';

// Includes real fixture lifecycle, multiple rendered states and remote asset loading.
// Keep assertion deadlines unchanged while giving this scenario a bounded budget.
test.describe.configure({ timeout: 90_000 });
async function success<T>(response: APIResponse): Promise<T> {
 const body = await response.json();
 expect(response.ok(), `HTTP ${response.status()} ${body.message}`).toBeTruthy();
 expect(body.code, body.message).toBe(200);
 return body.data as T;
}
for (const mode of ['light', 'dark'] as const) {
 test(`selection and preview confirm/conflict/refresh (${mode})`, async ({ page }, testInfo) => {
  const login = await loginByApi(page.request, adminCredentials);
  await installClientSession(page, login); await primeChineseLocale(page);
  await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  const headers = apiRequestHeaders(login);
  const token = `selection-preview-${mode}-${Date.now()}`;
  let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0;
  let failed = false;
  try {
   scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
   const ip = `10.249.${Math.floor(Math.random() * 200) + 1}.91`;
   hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip, os: 'linux', businessScopeId: scopeId } }))).id;
   packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo local-review-fixture', status: 'enabled' } }))).id;
   templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
   const selection = { businessScopeId: scopeId, hostIds: [hostId], groupIds: [], manual: [], excludedIds: [], includeScope: false };
   const resolved = await success<DeployTargetResolution>(await page.request.post(`${apiBaseUrl}/business/deploy/targets/resolve`, { headers, data: selection }));
   taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, packageId, templateId, templateParams: { serviceName: 'reviewed-service', port: '8080', rootPassword: '{{secret.password}}', custom: 'unrestricted-config-fixture' }, secretParams: { password: 'local-template-secret-fixture' }, executorType: 'manual', targetType: 'host', targetSelection: selection, scopeVersion: resolved.scopeVersion } }))).id;
   // Public execution cannot bypass review.
   const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/start`, { headers, data: {} });
   expect((await missing.json()).code).not.toBe(200);
   await page.setViewportSize({ width: 1440, height: 900 });
   await page.goto(`/business/deploy/task?editId=${taskId}`);
   const selector = page.locator('.deploy-target-selector');
   // Remote startup includes lazy chunks, auth hydration and option lookup.
   await expect(selector).toBeVisible({ timeout: 30_000 });
   await expect(selector.getByText(`${token} · ${ip}`, { exact: true }).first()).toBeVisible();
   await selector.getByLabel('每行输入一个主机 IP 或 ID').fill('not-a-host');
   await selector.getByRole('button', { name: '解析并核对目标' }).click();
   await expect(selector.getByText('目标无效或状态不允许，请调整选择后重新解析')).toBeVisible();
   await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-desktop-invalid.png`), fullPage: true });
   await page.setViewportSize({ width: 390, height: 844 });
   await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
   await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-invalid.png`), fullPage: true });
   await selector.getByLabel('每行输入一个主机 IP 或 ID').fill(ip);
   await selector.getByRole('button', { name: '解析并核对目标' }).click();
   await expect(selector.getByRole('status')).toContainText('已解析 1 个目标');
   await expect(selector.getByRole('status')).toContainText('去重 1');
   await page.screenshot({ path: testInfo.outputPath(`selection-${mode}-mobile-ready.png`), fullPage: true });
   await page.locator('.arco-modal:visible').last().getByRole('button', { name: '取消', exact: true }).click();
   await page.goto(`/business/deploy/task/${taskId}`);
   let starts = 0;
   page.on('request', (request) => { if (request.method() === 'POST' && request.url().endsWith(`/tasks/${taskId}/start`)) starts++; });
   await page.getByRole('button', { name: '启动', exact: true }).click();
   const modal = page.locator('.arco-modal:visible').last();
   await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
   expect(starts).toBe(0);
   const plan = modal.getByRole('region', { name: '已冻结的执行配置' });
   await expect(plan).toContainText(`${token}-template`);
   await expect(plan).toContainText('reviewed-service');
   await expect(plan).toContainText('8080');
   await expect(plan).toContainText('当前参数值尚未测量');
   await expect(plan).toContainText('失败处理：停止执行');
   const stepToggle = plan.locator('.arco-collapse-item-header').first();
   await stepToggle.focus();
   await page.keyboard.press('Enter');
   await expect(plan.getByText('reviewed-service', { exact: true })).toBeHidden();
   await page.keyboard.press('Enter');
   await expect(plan.getByText('reviewed-service', { exact: true })).toBeVisible();
   await expect(plan).not.toContainText('local-template-secret-fixture');
   await expect(plan).not.toContainText('unrestricted-config-fixture');
   await plan.scrollIntoViewIfNeeded();
   await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-mobile.png`) });
   await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-mobile-ready.png`), fullPage: true });
   await page.setViewportSize({ width: 1440, height: 900 });
   await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop.png`) });
   await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-ready.png`), fullPage: true });
   await page.emulateMedia({ reducedMotion: 'reduce' });
   await expect.poll(() => page.evaluate(() => matchMedia('(prefers-reduced-motion: reduce)').matches)).toBe(true);
   await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-reduced-motion.png`) });
   await page.evaluate(() => { document.documentElement.style.zoom = '2'; });
   await plan.screenshot({ path: testInfo.outputPath(`preview-plan-${mode}-desktop-css-zoom200.png`) });
   await page.evaluate(() => { document.documentElement.style.zoom = ''; });
   // These interceptions prove UI recovery states, not a real role revocation.
   const previewUrl = `**/business/deploy/tasks/${taskId}/preview`;
   for (const scenario of [{ status: 403, message: 'permission.denied', text: '无权查看或执行此任务', label: 'forbidden' }, { status: 503, message: 'unavailable', text: '预览加载失败，请重试', label: 'error' }]) {
    await modal.getByRole('button', { name: '取消', exact: true }).click();
    await page.route(previewUrl, (route) => route.fulfill({ status: scenario.status, contentType: 'application/json', body: JSON.stringify({ code: scenario.status, message: scenario.message, data: null }) }));
    await page.getByRole('button', { name: '启动', exact: true }).click();
    await expect(modal.getByText(scenario.text, { exact: true })).toBeVisible();
    await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
    await expect(modal.getByRole('region', { name: '已冻结的执行配置' })).toHaveCount(0);
    await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-${scenario.label}.png`), fullPage: true });
    await page.unroute(previewUrl);
   }
   await modal.getByRole('button', { name: '刷新预览' }).click();
   await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
   // Change a persisted fact after review; the running state must remain untouched.
   await success(await page.request.put(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers, data: { remark: 'changed after review' } }));
   await modal.getByRole('button', { name: '确认并执行' }).click();
   await expect(modal.getByText('预览已过期或数据已变化，请刷新预览', { exact: true })).toBeVisible();
   await expect(modal.getByRole('button', { name: '确认并执行' })).toBeDisabled();
   const current = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
   expect(current.status).toBe('draft');
   await page.screenshot({ path: testInfo.outputPath(`preview-${mode}-desktop-conflict.png`), fullPage: true });
   await modal.getByRole('button', { name: '刷新预览' }).click();
   await expect(modal.getByRole('button', { name: '确认并执行' })).toBeEnabled();
   await modal.getByRole('button', { name: '确认并执行' }).focus();
   await page.keyboard.press('Enter');
   await expect(modal).toBeHidden();
   expect(starts).toBe(2);
   const started = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }));
   expect(started.status).toBe('running');
  } catch (error) { failed = true; throw error; }
  finally {
   await page.close();
   const errors: string[] = [];
   try {
    const cleanupHeaders = await verifiedApiHeaders(page.request, login);
    if (taskId) {
     const response = await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers: cleanupHeaders });
     const current = await response.json();
     if (current.data?.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/cancel`, { headers: cleanupHeaders, data: { stateVersion: current.data?.stateVersion } }));
    }
    for (const [id, endpoint] of [[taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
     if (!id) continue;
     try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
     catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
    }
   } catch (error) { errors.push(String(error)); }
   await testInfo.attach('owned-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, hostId, scopeId, packageId, templateId, errors }) });
   if (!failed) expect(errors).toEqual([]);
  }
 });
}

test('rollback draft uses frozen definition and its own reviewed start', async ({ page }, testInfo) => {
 const login = await loginByApi(page.request, adminCredentials);
 const headers = apiRequestHeaders(login);
 const token = `frozen-rollback-${Date.now()}`;
 let scopeId = 0, hostId = 0, packageId = 0, templateId = 0, taskId = 0, rollbackId = 0;
 let failed = false;
 try {
  scopeId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/bizscope`, { headers, data: { code: token, name: token, environment: 'dev', status: 'active' } }))).id;
  hostId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/cmdb/hosts`, { headers, data: { hostname: token, ip: `10.248.${Math.floor(Math.random() * 200) + 1}.91`, os: 'linux', businessScopeId: scopeId } }))).id;
  packageId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/packages`, { headers, data: { name: token, version: '2.0.0', installCommand: 'echo frozen-install', uninstallCommand: 'echo frozen-uninstall', status: 'enabled' } }))).id;
  templateId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/templates`, { headers, data: { name: `${token}-template`, version: '1.0.0', packageId, status: 'enabled' } }))).id;
  taskId = (await success<{id: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks`, { headers, data: { name: token, templateId, businessScopeId: scopeId, targetType: 'host', targetIds: [hostId], executorType: 'manual', templateParams: { serviceName: 'frozen-service', rootPassword: '{{secret.password}}' }, secretParams: { password: 'rollback-private-fixture' } } }))).id;
  const started = await success<{hosts: Array<{id: number}>}>(await startReviewedTask(page.request, taskId, { headers }));
  await success(await page.request.post(`${apiBaseUrl}/business/deploy/task-hosts/${started.hosts[0].id}/result`, { headers, data: { status: 'success', stdout: 'local manual fixture' } }));
  await success(await page.request.put(`${apiBaseUrl}/business/deploy/templates/${templateId}`, { headers, data: { status: 'disabled' } }));
  await success(await page.request.put(`${apiBaseUrl}/business/deploy/packages/${packageId}`, { headers, data: { status: 'disabled' } }));
  const rollback = await success<{id: number; status: string; action: string; sourceTaskId: number}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${taskId}/rollback`, { headers }));
  rollbackId = rollback.id;
  expect(rollback.status).toBe('draft'); expect(rollback.action).toBe('uninstall'); expect(rollback.sourceTaskId).toBe(taskId);
  const info = await success<{id: number}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}/rollback`, { headers })); expect(info.id).toBe(rollbackId);
  const preview = await success<{intentVersion: string; plan: {templateVersion: string; steps: Array<{packageVersion: string; action: string}>}}>(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${rollbackId}/preview`, { headers, data: {} }));
  expect(preview.plan.templateVersion).toBe('1.0.0'); expect(preview.plan.steps[0].packageVersion).toBe('2.0.0'); expect(preview.plan.steps[0].action).toBe('uninstall');
  expect(JSON.stringify(preview)).not.toContain('rollback-private-fixture'); expect(JSON.stringify(preview)).not.toContain('frozen-uninstall');
  const missing = await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${rollbackId}/start`, { headers, data: {} }); expect((await missing.json()).code).not.toBe(200);
  await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${rollbackId}/start`, { headers, data: { intentVersion: preview.intentVersion } }));
  expect((await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${taskId}`, { headers }))).status).toBe('success');
 } catch (error) { failed = true; throw error; }
 finally {
  const errors: string[] = [];
  try {
   const cleanupHeaders = await verifiedApiHeaders(page.request, login);
   if (rollbackId) {
    const current = await success<{status: string; stateVersion: string}>(await page.request.get(`${apiBaseUrl}/business/deploy/tasks/${rollbackId}`, { headers: cleanupHeaders }));
    if (current.status === 'running') await success(await page.request.post(`${apiBaseUrl}/business/deploy/tasks/${rollbackId}/cancel`, { headers: cleanupHeaders, data: { stateVersion: current.stateVersion } }));
   }
   for (const [id, endpoint] of [[rollbackId, 'deploy/tasks'], [taskId, 'deploy/tasks'], [templateId, 'deploy/templates'], [hostId, 'cmdb/hosts'], [scopeId, 'bizscope'], [packageId, 'deploy/packages']] as const) {
    if (!id) continue;
    try { await success(await page.request.delete(`${apiBaseUrl}/business/${endpoint}/${id}`, { headers: cleanupHeaders })); }
    catch (error) { errors.push(`${endpoint}/${id}: ${String(error)}`); }
   }
  } catch (error) { errors.push(String(error)); }
  await testInfo.attach('owned-rollback-fixture-cleanup', { contentType: 'application/json', body: JSON.stringify({ token, taskId, rollbackId, errors }) });
  if (!failed) expect(errors).toEqual([]);
 }
});
