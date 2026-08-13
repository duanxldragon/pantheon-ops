/**
 * UI Components Capture - Using cookies from successful login
 */

import { chromium } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const BASE_URL = process.env.PANTHEON_WEB_BASE_URL || 'http://127.0.0.1:5173';
const API_URL = process.env.PANTHEON_API_BASE_URL || 'http://127.0.0.1:8080/api/v1';
const OUTPUT_DIR = path.join(process.cwd(), 'screenshots/components');

if (!fs.existsSync(OUTPUT_DIR)) {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
}

async function login(page) {
  // Set locale
  await page.addInitScript(() => {
    try { localStorage.setItem('pantheon_lang', 'zh-CN'); } catch (e) {}
  });

  // First do API login to get tokens
  const response = await page.request.post(`${API_URL}/auth/login`, {
    data: { username: 'admin', password: '123456' }
  });

  if (!response.ok()) {
    console.log('API login failed, trying UI login...');
    await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
    await page.waitForSelector('form', { timeout: 10000 });
    await page.locator('input').first().fill('admin');
    await page.locator('input[type="password"]').first().fill('123456');
    await page.locator('button[type="submit"]').click();
    await page.waitForTimeout(5000);
  } else {
    // Set cookies from response
    const data = await response.json();
    if (data.data) {
      const { accessToken, refreshToken, csrfToken } = data.data;
      if (accessToken) {
        await page.context().addCookies([
          { name: 'pantheon_access_token', value: accessToken, url: BASE_URL, httpOnly: true, secure: false, sameSite: 'Strict' },
          { name: 'pantheon_refresh_token', value: refreshToken || '', url: BASE_URL, httpOnly: true, secure: false, sameSite: 'Strict' },
          { name: 'pantheon_csrf_token', value: csrfToken || 'csrf', url: BASE_URL, httpOnly: true, secure: false, sameSite: 'Strict' },
        ]);
      }
    }
    await page.goto(`${BASE_URL}/dashboard`, { waitUntil: 'networkidle' });
  }

  await page.waitForTimeout(2000);
  console.log('Logged in, current URL:', page.url());
}

async function capture(page, name) {
  const filePath = path.join(OUTPUT_DIR, `${name}.png`);
  await page.screenshot({ path: filePath, fullPage: true, timeout: 15000 });
  console.log(`  Captured: ${name}.png`);
}

async function main() {
  console.log('UI Components Capture...\n');

  const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
  const context = await browser.newContext({ viewport: { width: 1920, height: 1080 } });
  const page = await context.newPage();

  try {
    await login(page);

    console.log('\n--- Capturing UI Components ---\n');

    // 5. i18n Page (Drawer)
    console.log('5. i18n List with Drawer');
    await page.goto(`${BASE_URL}/system/i18n`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '05-i18n-list');

    // 6. Settings Page
    console.log('6. Settings Page');
    await page.goto(`${BASE_URL}/system/setting`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '06-settings-page');

    // 7. Dict Page (Tabs)
    console.log('7. Dict Page with Tabs');
    await page.goto(`${BASE_URL}/system/dict`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '07-dict-tabs');

    // 8. Menu Tree
    console.log('8. Menu Tree');
    await page.goto(`${BASE_URL}/system/menu`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '08-menu-tree');

    // 9. Dept Tree
    console.log('9. Dept Tree');
    await page.goto(`${BASE_URL}/system/dept`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '09-dept-tree');

    // 10. Security Center
    console.log('10. Security Center');
    await page.goto(`${BASE_URL}/auth/security`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '10-security-center');

    // 11. Profile Page
    console.log('11. Profile Page');
    await page.goto(`${BASE_URL}/system/profile`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '11-profile-page');

    // 12. Role List
    console.log('12. Role List');
    await page.goto(`${BASE_URL}/system/role`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '12-role-list');

    // 13. Permission List
    console.log('13. Permission List');
    await page.goto(`${BASE_URL}/system/permission`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '13-permission-list');

    // 14. Post List (Empty Search)
    console.log('14. Post List');
    await page.goto(`${BASE_URL}/system/post`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '14-post-list');

    // 15. User Table with Selection
    console.log('15. User Table');
    await page.goto(`${BASE_URL}/system/user`, { waitUntil: 'networkidle' });
    await page.waitForTimeout(1500);
    await capture(page, '15-user-table');

    console.log('\n--- Complete ---');
    console.log(`Output: ${OUTPUT_DIR}`);

  } catch (e) {
    console.error('Error:', e.message);
  } finally {
    await browser.close();
  }
}

main();
