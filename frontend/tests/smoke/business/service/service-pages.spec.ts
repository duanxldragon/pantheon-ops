import { expect, test } from '@playwright/test';
import {
  adminCredentials,
  installClientSession,
  loginByApi,
  primeChineseLocale,
} from '../../helpers/auth';

async function expectNoVisibleServiceI18nKeys(page: import('@playwright/test').Page) {
  const bodyText = await page.locator('body').innerText();
  expect(bodyText).not.toMatch(/\bbusiness\.service\.[A-Za-z0-9_.-]+\b/);
  expect(bodyText).not.toMatch(/\boperations\.service\.[A-Za-z0-9_.-]+\b/);
}

test.describe('Service business module smoke', () => {
  test.beforeEach(async ({ page }) => {
    const login = await loginByApi(page.request, adminCredentials);
    await installClientSession(page, login);
    await primeChineseLocale(page);
  });

  test('service and instance tables load on the business route', async ({ page }, testInfo) => {
    const consoleErrors: string[] = [];
    page.on('console', (message) => {
      if (message.type() === 'error') consoleErrors.push(message.text());
    });

    await page.goto('/business/service', { waitUntil: 'networkidle' });
    await expect(page).toHaveURL(/\/business\/service$/);
    await expect(page.locator('.page-container')).toBeVisible();
    await expect(page.locator('.governance-summary-bar')).toContainText('服务目录');
    await expect(page.locator('.search-toolbar')).toBeVisible();
    await expect(page.locator('.system-list__table-card')).toHaveCount(2);
    await expect(page.getByText('服务实例', { exact: true })).toBeVisible();
    await expectNoVisibleServiceI18nKeys(page);
    await page.screenshot({ path: testInfo.outputPath('service-list-desktop.png'), fullPage: true });
    expect(consoleErrors).toEqual([]);
  });

  test('service page remains usable in a phone viewport', async ({ page }, testInfo) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/business/service', { waitUntil: 'networkidle' });
    await expect(page.locator('.governance-summary-bar')).toBeVisible();
    await expect(page.getByText('服务实例', { exact: true })).toBeVisible();
    await expect
      .poll(async () => page.evaluate(() => document.body.scrollWidth <= globalThis.innerWidth + 1))
      .toBe(true);
    await page.screenshot({ path: testInfo.outputPath('service-list-mobile.png'), fullPage: true });
  });
});
