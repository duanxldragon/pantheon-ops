import { expect, test, type Page } from '@playwright/test';
import { primeChineseLocale, signInAsAdmin } from '../smoke/helpers/auth';

async function waitForVisualStability(page: Page) {
  await page.waitForLoadState('networkidle');
  await page.evaluate(async () => {
    await document.fonts.ready;
  });
}

test('login page visual baseline', async ({ page }) => {
  await primeChineseLocale(page);
  await page.goto('/login', { waitUntil: 'networkidle' });
  await expect(page.locator('.auth-login-page')).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('login.png');
});

test('workspace dashboard visual baseline', async ({ page }) => {
  await signInAsAdmin(page);
  await page.goto('/dashboard', { waitUntil: 'networkidle' });
  await expect(page.locator('.dashboard-hero-card')).toBeVisible();
  await expect(page.locator('.dashboard-stat-card').first()).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('dashboard.png', {
    mask: [
      page.locator('.dashboard-stat-card__value'),
      page.locator('.dashboard-stat-card--users .arco-tag'),
      page.locator('.dashboard-focus-item__value'),
      page.locator('.dashboard-domain-card__summary'),
      page.locator('.dashboard-login-table .app-table tbody tr'),
    ],
  });
});

test('system user list visual baseline', async ({ page }) => {
  await signInAsAdmin(page);
  await page.goto('/system/user', { waitUntil: 'networkidle' });
  await expect(page.locator('.system-list__table-card')).toBeVisible();
  await expect(page.locator('.app-table')).toBeVisible();
  await waitForVisualStability(page);

  await expect(page).toHaveScreenshot('system-user-list.png', {
    mask: [
      page.locator('.governance-summary-bar__metric-value'),
      page.locator('.system-list__table-card .app-table tbody tr'),
    ],
  });
});
