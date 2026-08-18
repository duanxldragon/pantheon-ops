import { expect, test } from '@playwright/test';
import {
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

async function expectNoVisibleK8sI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.k8s\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.k8s\.[A-Za-z0-9_.-]+\b/);
}

test.describe('K8s business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('cluster and release pages load on business routes', async ({ page }, testInfo) => {
    await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/\/business\/k8s\/cluster$/);
    await expect(page.locator('.governance-summary-bar')).toContainText('K8s 集群');
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await expectNoVisibleK8sI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('k8s-cluster-desktop.png'), fullPage: true });

    await page.goto('/business/k8s/release', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/\/business\/k8s\/release$/);
    await expect(page.locator('.governance-summary-bar')).toContainText('应用发布');
    await expect(page.locator('.system-list__table-card')).toBeVisible();
    await expectNoVisibleK8sI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('k8s-release-desktop.png'), fullPage: true });
  });

  test('cluster list remains usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/k8s/cluster', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('k8s-cluster-mobile.png'), fullPage: true });
  });
});
