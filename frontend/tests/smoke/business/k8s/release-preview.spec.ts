import { expect, test } from '@playwright/test';
import fs from 'node:fs/promises';
import { adminCredentials, installClientSession, loginByApi, primeChineseLocale } from '../../helpers/auth';

// Real local auth; K8s release transport is a browser fixture. No cluster is mutated.
for (const mode of ['light', 'dark'] as const) {
 test(`K8s image preview conflict, same-request retry, rollback and mobile (${mode})`, async ({ page }, testInfo) => {
  const login = await loginByApi(page.request, adminCredentials); await installClientSession(page, login); await primeChineseLocale(page);
  await page.addInitScript((color) => localStorage.setItem('pantheon_color_mode', color), mode);
  await page.addInitScript(() => {
   const browser = window as typeof window & { canceledMessages: string[] };
   browser.canceledMessages = [];
   new MutationObserver(() => {
    for (const message of document.querySelectorAll('.arco-message')) {
     if (/canceled/i.test(message.textContent ?? '')) browser.canceledMessages.push(message.textContent ?? '');
    }
   }).observe(document, { subtree: true, childList: true, characterData: true });
  });
  const row = { id: 77, name: 'image-release-fixture', clusterId: 901, namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', imageBefore: 'image:v1', imageAfter: 'image:v2', status: 'succeeded', createdAt: '2026-11-08T00:00:00Z' };
  const requests: { operation: string; idempotencyKey: string; intentVersion?: string; image?: string }[] = [];
  let stage: 'conflict' | 'uncertain' | 'ready' | 'forbidden' = 'conflict';
  await page.route('**/business/k8s/clusters?*', (route) => route.fulfill({ json: { code: 200, data: { items: [{ id: 901, name: 'Local cluster fixture' }], total: 1 } } }));
  await page.route('**/business/k8s/releases**', async (route) => {
   const url = new URL(route.request().url()); const path = url.pathname;
   if (route.request().method() === 'GET') { await route.fulfill({ json: { code: 200, data: { items: [row], total: 1 } } }); return; }
   const body = route.request().postDataJSON() as { idempotencyKey: string; intentVersion?: string; image?: string };
   expect(body.idempotencyKey).toBeTruthy(); requests.push({ operation: path, ...body });
   if (path.endsWith('/preview')) {
    if (stage === 'forbidden') { await route.fulfill({ status: 403, json: { code: 403, message: 'permission.denied' } }); return; }
    await route.fulfill({ json: { code: 200, data: { intentId: body.idempotencyKey, intentVersion: `ticket-${body.idempotencyKey}`, currentVersion: '15', targetVersion: 'image-intent-hash', targetCount: 1, environment: 'non-production', namespace: 'default', workloadType: 'deployment', workloadName: 'api', containerName: 'app', changes: [{ id: 'image', field: 'image', kind: 'modified', before: path.includes('rollback') ? 'image:v2' : 'image:v1', after: body.image ?? 'image:v1' }], riskKeys: ['k8s.release.preview_image_only', 'k8s.release.preview_rollout_required', 'k8s.release.preview_rollback_limits'], rollbackAvailable: true, expiresAt: new Date(Date.now() + 600000).toISOString() } } }); return;
   }
   expect(body.intentVersion).toBe(`ticket-${body.idempotencyKey}`);
   if (stage === 'conflict') { await route.fulfill({ json: { code: 409, message: 'k8s.release.preview_conflict' } }); return; }
   if (stage === 'uncertain') { await route.abort('failed'); return; }
   await route.fulfill({ json: { code: 200, data: row } });
  });
  await page.setViewportSize({ width: 1440, height: 900 }); await page.goto('/business/k8s/release');
  await expect(page.getByText('image-release-fixture', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: '回滚', exact: true }).click();
  let dialog = page.getByRole('dialog', { name: '预览发布变更', exact: true }); await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  await expect(dialog.getByText('回滚只能恢复记录的镜像，不能恢复配置或撤销数据变更。')).toBeVisible();
  await dialog.getByRole('button', { name: '确认并执行' }).focus();
  await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeFocused();
  await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-desktop-review.png`), fullPage: true });
  await dialog.getByRole('button', { name: '确认并执行' }).click();
  await expect(dialog.getByText('目标或意图已变化，请重新预览。')).toBeVisible();
  await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeDisabled();
  await page.setViewportSize({ width: 390, height: 844 });
    await expect.poll(() => page.locator('.app-shell__sider').evaluate((element) => element.getBoundingClientRect().right)).toBeLessThanOrEqual(1);
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(1);
  await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-conflict.png`), fullPage: true });
  stage = 'uncertain'; await dialog.getByRole('button', { name: '重新预览' }).click();
  await expect(dialog.getByRole('button', { name: '确认并执行' })).toBeEnabled();
  await page.screenshot({ path: testInfo.outputPath(`k8s-${mode}-mobile-review.png`), fullPage: true });
  await dialog.getByRole('button', { name: '确认并执行' }).click();
  await expect(dialog.getByRole('button', { name: '重试同一请求' })).toBeVisible();
  await expect(dialog.getByRole('button', { name: '重新预览' })).toBeDisabled();
  stage = 'ready'; await dialog.getByRole('button', { name: '重试同一请求' }).click();
  await expect(dialog.getByText('发布意图已受理，请查看列表中的执行状态。')).toBeVisible();
  const submits = requests.filter((request) => !request.operation.endsWith('/preview'));
  expect(submits[2]).toEqual(submits[1]); expect(submits[1].idempotencyKey).not.toBe(submits[0].idempotencyKey);
  await dialog.getByRole('button', { name: '关闭', exact: true }).click(); await expect(dialog).toBeHidden();
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.getByRole('button', { name: '创建发布', exact: true }).click(); dialog = page.getByRole('dialog', { name: '创建发布', exact: true });
  await dialog.getByPlaceholder('请输入发布名称', { exact: true }).fill('create-image-fixture');
  await dialog.locator('.arco-select').first().click(); await page.getByText('Local cluster fixture', { exact: true }).last().click();
  await dialog.getByPlaceholder('输入命名空间', { exact: true }).fill('default');
  await dialog.locator('.arco-select').nth(1).click(); await page.getByText('deployment', { exact: true }).last().click();
  await dialog.getByPlaceholder('输入工作负载名称', { exact: true }).fill('api');
  await dialog.getByPlaceholder('如 nginx:1.25', { exact: true }).fill('image:v3');
  await dialog.getByRole('button', { name: '预览发布变更', exact: true }).click();
  await expect(page.getByRole('dialog', { name: '预览发布变更', exact: true }).getByText('image:v3', { exact: true })).toBeVisible();
  await page.getByRole('dialog', { name: '预览发布变更', exact: true }).getByRole('button', { name: '取消', exact: true }).click();
  await expect(page.getByRole('dialog', { name: '创建发布', exact: true }).getByPlaceholder('请输入发布名称')).toHaveValue('create-image-fixture');
  await page.getByRole('dialog', { name: '创建发布', exact: true }).getByRole('button', { name: '取消', exact: true }).click();
  stage = 'forbidden'; await page.getByRole('button', { name: '回滚', exact: true }).click();
  await expect(page.getByRole('dialog').getByText('没有发布预览权限')).toBeVisible();
  await expect(page.getByRole('dialog').getByRole('button', { name: '确认并执行' })).toBeDisabled();
  expect(await page.evaluate(() => (window as typeof window & { canceledMessages: string[] }).canceledMessages)).toEqual([]);
  await fs.writeFile(testInfo.outputPath('release-preview-transport.json'), JSON.stringify({ provider: 'browser transport fixture', realLocalAuth: true, realK8sMutation: false, ownedBusinessFixtures: 0, requests }, null, 2));
 });
}
