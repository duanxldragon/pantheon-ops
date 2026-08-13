/**
 * UI Component Details Capture
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
  await page.locator('button[type="submit"]').click();
  await page.waitForTimeout(5000);
  console.log('Login URL:', page.url());
}

async function captureElement(page, selector, filename) {
  const filePath = path.join(OUTPUT_DIR, `${filename}.png`);
  try {
    const element = page.locator(selector).first();
    const isVisible = await element.isVisible({ timeout: 3000 }).catch(() => false);
    if (isVisible) {
      await element.screenshot({ path: filePath, timeout: 10000 });
      console.log(`  Captured: ${filename}.png`);
    } else {
      console.log(`  Not visible: ${selector}`);
    }
  } catch (e) {
    console.log(`  Error capturing ${filename}: ${e.message.substring(0, 50)}`);
  }
}

async function main() {
  console.log('UI Component Details Capture...\n');

  const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
  const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } });
  const page = await context.newPage();

  try {
    console.log('--- Login ---');
    await login(page);
    console.log('');

    // 1. User Add Modal - Full
    console.log('1. User Add Modal');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('button').filter({ hasText: /新增/i }).first().click();
    await page.waitForTimeout(800);
    await captureElement(page, '.arco-modal', 'modal-01-user-form');

    // Close modal
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // 2. Role Auth Modal
    console.log('2. Role Authorization Modal');
    await page.goto(`${BASE_URL}/system/role`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('button').filter({ hasText: /授权/i }).first().click();
    await page.waitForTimeout(800);
    await captureElement(page, '.arco-modal', 'modal-02-role-auth');

    // Close modal
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // 3. Pagination
    console.log('3. Pagination');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-pagination', 'component-03-pagination');

    // 4. Table
    console.log('4. Data Table');
    await captureElement(page, '.arco-table, table', 'component-04-table');

    // 5. Search/Filter Bar
    console.log('5. Search Bar');
    await captureElement(page, '.arco-table-toolbar, [class*="toolbar"]', 'component-05-toolbar');

    // 6. Tabs - Dict
    console.log('6. Tabs Component');
    await page.goto(`${BASE_URL}/system/dict`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-tabs, [class*="tabs"]', 'component-06-tabs');

    // 7. Tree - Menu
    console.log('7. Tree Component');
    await page.goto(`${BASE_URL}/system/menu`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-tree, [class*="tree"]', 'component-07-tree');

    // 8. Form Elements - Profile
    console.log('8. Form Elements');
    await page.goto(`${BASE_URL}/system/profile`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, 'form', 'component-08-form');

    // 9. Sidebar/Navigation
    console.log('9. Sidebar Navigation');
    await captureElement(page, '.app-shell__sider, .arco-layout-sider, [class*="sider"]', 'component-09-sidebar');

    // 10. Header
    console.log('10. Header');
    await captureElement(page, '.app-shell__header, .arco-layout-header, [class*="header"]', 'component-10-header');

    // 11. Dropdown
    console.log('11. Dropdown');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('button').filter({ hasText: /新增/i }).first().click();
    await page.waitForTimeout(500);
    // Click on role select
    const roleSelect = page.locator('.arco-select').first();
    if (await roleSelect.isVisible({ timeout: 2000 }).catch(() => false)) {
      await roleSelect.click();
      await page.waitForTimeout(500);
      await captureElement(page, '.arco-select-popup, .arco-select-view, [class*="select-popup"]', 'component-11-dropdown');
    }

    // Close modal
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // 12. i18n Drawer
    console.log('12. Drawer Panel');
    await page.goto(`${BASE_URL}/system/i18n`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await page.locator('button').filter({ hasText: /编辑/i }).first().click();
    await page.waitForTimeout(800);
    await captureElement(page, '.arco-drawer, [class*="drawer"]', 'component-12-drawer');

    // Close drawer
    await page.keyboard.press('Escape');
    await page.waitForTimeout(500);

    // 13. Settings Cards
    console.log('13. Settings Cards');
    await page.goto(`${BASE_URL}/system/setting`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-grid, [class*="grid"]', 'component-13-settings-grid');

    // 14. Security Charts
    console.log('14. Security Charts');
    await page.goto(`${BASE_URL}/auth/security`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-statistic, [class*="statistic"]', 'component-14-security-stats');

    // 15. Breadcrumb
    console.log('15. Breadcrumb');
    await captureElement(page, '.arco-breadcrumb, [class*="breadcrumb"]', 'component-15-breadcrumb');

    // 16. Table Pagination Info
    console.log('16. Pagination Info');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    await captureElement(page, '.arco-table-pagination', 'component-16-table-pagination');

    console.log('\n--- Complete ---');

  } catch (e) {
    console.error('Error:', e.message);
  } finally {
    await browser.close();
  }
}

main();
