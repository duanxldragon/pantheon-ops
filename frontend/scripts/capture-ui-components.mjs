/**
 * UI Components Screenshot Test
 * Captures dialogs, modals, pagination, notifications, and other UI elements
 */

import { chromium } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const BASE_URL = process.env.PANTHEON_WEB_BASE_URL || 'http://127.0.0.1:5173';
const OUTPUT_DIR = path.join(process.cwd(), 'screenshots/components');

// Ensure output directory exists
if (!fs.existsSync(OUTPUT_DIR)) {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
}

async function login(page) {
  console.log('  Performing UI login...');

  await page.addInitScript(() => {
    try {
      localStorage.setItem('pantheon_lang', 'zh-CN');
      localStorage.setItem('pantheon_lang_explicit', '1');
    } catch (e) {}
  });

  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  await page.waitForSelector('form', { timeout: 10000 });

  const usernameInput = page.locator('input[placeholder*="username" i], input[placeholder*="用户"]').first();
  const passwordInput = page.locator('input[type="password"]').first();

  await usernameInput.fill('admin');
  await passwordInput.fill('123456');

  const loginButton = page.locator('button[type="submit"], button:has-text("登录")').first();
  await loginButton.click();

  await page.waitForTimeout(3000);
  await page.waitForSelector('.app-shell', { timeout: 10000 });
  console.log('  Login completed');
}

async function captureScreenshot(page, name) {
  const fileName = `${name}.png`;
  const filePath = path.join(OUTPUT_DIR, fileName);

  await page.screenshot({
    path: filePath,
    fullPage: true,
    timeout: 10000,
  });

  console.log(`  Captured: ${fileName}`);
}

async function captureElement(page, selector, name) {
  const fileName = `${name}.png`;
  const filePath = path.join(OUTPUT_DIR, fileName);

  try {
    const element = page.locator(selector).first();
    await element.screenshot({ path: filePath, timeout: 10000 });
    console.log(`  Captured: ${fileName}`);
  } catch (e) {
    console.log(`  Failed to capture ${name}: ${e.message}`);
  }
}

async function main() {
  console.log('Starting UI components screenshot capture...');
  console.log(`Output directory: ${OUTPUT_DIR}\n`);

  const browser = await chromium.launch({
    headless: true,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });

  const context = await browser.newContext({
    viewport: { width: 1920, height: 1080 },
  });

  const page = await context.newPage();

  try {
    console.log('--- Logging in ---');
    await login(page);
    console.log('');

    console.log('--- Capturing UI Components ---\n');

    // 1. User List - Capture modal dialog for add/edit user
    console.log('1. User Add/Edit Modal');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Click add button
    const addButton = page.locator('button').filter({ hasText: /新增|添加/i }).first();
    if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addButton.click();
      await page.waitForTimeout(800);
      await captureScreenshot(page, '01-user-add-modal');
      await captureElement(page, '.arco-modal', '01-user-add-modal-focused');
      // Close modal
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
    }

    // 2. Role List - Capture role authorization modal
    console.log('2. Role Authorization Modal');
    await page.goto(`${BASE_URL}/system/role`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Click authorize button on first role
    const authButton = page.locator('button').filter({ hasText: /授权|权限/i }).first();
    if (await authButton.isVisible({ timeout: 3000 }).catch(() => false)) {
      await authButton.click();
      await page.waitForTimeout(800);
      await captureScreenshot(page, '02-role-auth-modal');
    }

    // 3. Pagination - User list pagination
    console.log('3. Pagination Component');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const pagination = page.locator('.arco-pagination, [class*="pagination"]').first();
    if (await pagination.isVisible({ timeout: 3000 }).catch(() => false)) {
      await captureScreenshot(page, '03-pagination-default');
      // Click next page
      const nextBtn = page.locator('.arco-pagination-item-next, [class*="pagination-next"]').first();
      if (await nextBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await nextBtn.click();
        await page.waitForTimeout(500);
        await captureScreenshot(page, '03-pagination-page2');
      }
    }

    // 4. Form validation
    console.log('4. Form Validation');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const addBtn2 = page.locator('button').filter({ hasText: /新增|添加/i }).first();
    if (await addBtn2.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addBtn2.click();
      await page.waitForTimeout(500);
      // Try to submit empty form
      const submitBtn = page.locator('.arco-modal button[type="submit"], .arco-modal button:has-text("确定")').first();
      if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await submitBtn.click();
        await page.waitForTimeout(500);
        await captureScreenshot(page, '04-form-validation-error');
      }
    }

    // 5. Dropdown/Select component
    console.log('5. Dropdown/Select Component');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const addBtn3 = page.locator('button').filter({ hasText: /新增|添加/i }).first();
    if (await addBtn3.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addBtn3.click();
      await page.waitForTimeout(500);
      // Open a select dropdown
      const selectInput = page.locator('.arco-select, [class*="select"]').first();
      if (await selectInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await selectInput.click();
        await page.waitForTimeout(500);
        await captureScreenshot(page, '05-dropdown-open');
      }
    }

    // 6. Drawer/Side panel
    console.log('6. Drawer/Side Panel');
    await page.goto(`${BASE_URL}/system/i18n`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Click edit on a row
    const editBtn = page.locator('button').filter({ hasText: /编辑/i }).first();
    if (await editBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await editBtn.click();
      await page.waitForTimeout(800);
      await captureScreenshot(page, '06-drawer-panel');
    }

    // 7. Notification/Message
    console.log('7. Notification/Message Toast');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Trigger a notification by clicking save without changes or similar
    const addBtn4 = page.locator('button').filter({ hasText: /新增|添加/i }).first();
    if (await addBtn4.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addBtn4.click();
      await page.waitForTimeout(500);
      // Close the modal to trigger any notification
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
      await captureScreenshot(page, '07-notification-area');
    }

    // 8. Search and Filter
    console.log('8. Search and Filter');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="请输入"]').first();
    if (await searchInput.isVisible({ timeout: 3000 }).catch(() => false)) {
      await searchInput.fill('admin');
      await page.waitForTimeout(800);
      await captureScreenshot(page, '08-search-active');
      await searchInput.clear();
      await page.waitForTimeout(300);
    }

    // 9. Table with loading state
    console.log('9. Table Loading State');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Trigger reload by clicking refresh
    const refreshBtn = page.locator('button').filter({ hasText: /刷新/i }).first();
    if (await refreshBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await refreshBtn.click();
      await page.waitForTimeout(300);
      await captureScreenshot(page, '09-table-loading');
      await page.waitForTimeout(1000);
    }

    // 10. Confirm Dialog
    console.log('10. Confirm Dialog');
    await page.goto(`${BASE_URL}/system/role`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Click delete on a role
    const deleteBtn = page.locator('button').filter({ hasText: /删除/i }).first();
    if (await deleteBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await deleteBtn.click();
      await page.waitForTimeout(500);
      await captureScreenshot(page, '10-confirm-dialog');
      // Close dialog
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    }

    // 11. Tabs Component (Dict page)
    console.log('11. Tabs Component');
    await page.goto(`${BASE_URL}/system/dict`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await captureScreenshot(page, '11-tabs-component');

    // 12. Breadcrumb Navigation
    console.log('12. Breadcrumb Navigation');
    const breadcrumb = page.locator('.arco-breadcrumb, [class*="breadcrumb"]').first();
    if (await breadcrumb.isVisible({ timeout: 3000 }).catch(() => false)) {
      await captureScreenshot(page, '12-breadcrumb');
    }

    // 13. Empty State
    console.log('13. Empty State');
    await page.goto(`${BASE_URL}/system/post`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);
    // Try to trigger empty state by searching non-existent item
    const searchInput2 = page.locator('input[placeholder*="搜索"], input[placeholder*="请输入"]').first();
    if (await searchInput2.isVisible({ timeout: 3000 }).catch(() => false)) {
      await searchInput2.fill('xyznonexistent123');
      await page.waitForTimeout(800);
      await captureScreenshot(page, '13-empty-state');
    }

    // 14. Tree Component Expanded
    console.log('14. Tree Component Expanded');
    await page.goto(`${BASE_URL}/system/menu`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await captureScreenshot(page, '14-tree-expanded');

    // 15. Tooltip
    console.log('15. Tooltip/Hover');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Hover over a button to show tooltip
    const addBtn5 = page.locator('button').filter({ hasText: /新增/i }).first();
    if (await addBtn5.isVisible({ timeout: 3000 }).catch(() => false)) {
      await addBtn5.hover();
      await page.waitForTimeout(500);
      await captureScreenshot(page, '15-tooltip-hover');
    }

    // 16. Message Box
    console.log('16. Message Box/Alert');
    await page.goto(`${BASE_URL}/system/setting`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await captureScreenshot(page, '16-message-box');

    // 17. Profile Avatar Upload
    console.log('17. Avatar Upload');
    await page.goto(`${BASE_URL}/system/profile`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await captureScreenshot(page, '17-avatar-upload');

    // 18. Password Input
    console.log('18. Password Input');
    await page.goto(`${BASE_URL}/system/profile`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    // Click change password
    const changePwdBtn = page.locator('button').filter({ hasText: /修改密码/i }).first();
    if (await changePwdBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await changePwdBtn.click();
      await page.waitForTimeout(500);
      await captureScreenshot(page, '18-password-input');
    }

    // 19. Date Picker
    console.log('19. Date Picker');
    await page.goto(`${BASE_URL}/system/audit/operation`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);

    const datePicker = page.locator('.arco-picker, [class*="date-picker"], input[type="date"]').first();
    if (await datePicker.isVisible({ timeout: 3000 }).catch(() => false)) {
      await datePicker.click();
      await page.waitForTimeout(500);
      await captureScreenshot(page, '19-date-picker');
    }

    // 20. Full Page Layout
    console.log('20. Full Page Layout');
    await page.goto(`${BASE_URL}/system/setting/general`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1000);

    await captureScreenshot(page, '20-full-layout');

    console.log('\n--- Capture complete ---');
    console.log(`Screenshots saved to: ${OUTPUT_DIR}`);

  } finally {
    await browser.close();
  }
}

main().catch(console.error);
