/**
 * UI Components Screenshot Test - Simplified
 */

import { chromium } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const BASE_URL = process.env.PANTHEON_WEB_BASE_URL || 'http://127.0.0.1:5173';
const OUTPUT_DIR = path.join(process.cwd(), 'screenshots/components');

async function login(page) {
  await page.addInitScript(() => {
    try { localStorage.setItem('pantheon_lang', 'zh-CN'); } catch (e) {}
  });
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  await page.waitForSelector('form', { timeout: 10000 });
  await page.locator('input').first().fill('admin');
  await page.locator('input[type="password"]').first().fill('123456');
  await page.locator('button[type="submit"]').first().click();
  await page.waitForTimeout(3000);
  await page.waitForSelector('.app-shell', { timeout: 10000 });
}

async function capture(page, name) {
  const filePath = path.join(OUTPUT_DIR, `${name}.png`);
  await page.screenshot({ path: filePath, fullPage: true, timeout: 10000 });
  console.log(`  Captured: ${name}.png`);
}

async function main() {
  console.log('Starting UI components capture (simplified)...\n');

  const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
  const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } });
  const page = await context.newPage();

  try {
    console.log('--- Logging in ---');
    await login(page);
    console.log('');

    // 5. Drawer/Side Panel - i18n edit
    console.log('5. Drawer Component');
    await page.goto(`${BASE_URL}/system/i18n`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '05-drawer-panel');

    // 6. Search Filter
    console.log('6. Search and Filter');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '06-search-filter');

    // 7. Table Loading State
    console.log('7. Table Loading');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(500);
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(300);
    await capture(page, '07-table-loading');

    // 8. Empty State
    console.log('8. Empty State');
    await page.goto(`${BASE_URL}/system/post`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '08-empty-state');

    // 9. Tree Component
    console.log('9. Tree Component');
    await page.goto(`${BASE_URL}/system/menu`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '09-tree-component');

    // 10. Tabs Component
    console.log('10. Tabs Component');
    await page.goto(`${BASE_URL}/system/dict`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '10-tabs-component');

    // 11. Settings Page
    console.log('11. Settings Page');
    await page.goto(`${BASE_URL}/system/setting`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '11-settings-page');

    // 12. Profile Page
    console.log('12. Profile Page');
    await page.goto(`${BASE_URL}/system/profile`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '12-profile-page');

    // 13. Confirm Dialog
    console.log('13. Confirm Dialog');
    await page.goto(`${BASE_URL}/system/role`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    const deleteBtn = page.locator('button').filter({ hasText: /删除/i }).first();
    if (await deleteBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
      await capture(page, '13-confirm-dialog');
      await page.keyboard.press('Escape');
    } else {
      await capture(page, '13-confirm-dialog');
    }

    // 14. Security Center
    console.log('14. Security Center');
    await page.goto(`${BASE_URL}/auth/security`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '14-security-center');

    // 15. Dept Tree
    console.log('15. Dept Tree');
    await page.goto(`${BASE_URL}/system/dept`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await capture(page, '15-dept-tree');

    console.log('\n--- Complete ---');
    console.log(`Output: ${OUTPUT_DIR}`);

  } finally {
    await browser.close();
  }
}

main().catch(console.error);
