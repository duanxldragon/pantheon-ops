/**
 * Page Screenshot Test Script
 * Captures screenshots of all pages for UI evaluation
 */

import { chromium } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';

const BASE_URL = process.env.PANTHEON_WEB_BASE_URL || 'http://127.0.0.1:5173';
const OUTPUT_DIR = path.join(process.cwd(), 'screenshots');

// Ensure output directory exists
if (!fs.existsSync(OUTPUT_DIR)) {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
}

// Pages to capture (skip login page, already captured)
const PAGES_TO_TEST = [
  { name: '02-dashboard', path: '/dashboard', waitFor: '.app-shell__header' },
  { name: '03-user-list', path: '/system/user', waitFor: '[class*="table"]' },
  { name: '05-role-list', path: '/system/role', waitFor: '[class*="table"]' },
  { name: '06-menu-list', path: '/system/menu', waitFor: '[class*="tree"]' },
  { name: '07-dept-list', path: '/system/dept', waitFor: '[class*="tree"]' },
  { name: '08-post-list', path: '/system/post', waitFor: '[class*="table"]' },
  { name: '09-dict-page', path: '/system/dict', waitFor: '[class*="tabs"]' },
  { name: '10-permission-list', path: '/system/permission', waitFor: '[class*="table"]' },
  { name: '11-permission-workbench', path: '/system/permission/workbench', waitFor: '[class*="table"]' },
  { name: '12-i18n-list', path: '/system/i18n', waitFor: '[class*="table"]' },
  { name: '13-setting-overview', path: '/system/setting', waitFor: '.app-shell__header' },
  { name: '14-setting-group', path: '/system/setting/general', waitFor: 'form' },
  { name: '15-operation-log', path: '/system/audit/operation', waitFor: '[class*="table"]' },
  { name: '16-login-log', path: '/auth/security/login-log', waitFor: '[class*="table"]' },
  { name: '17-security-center', path: '/auth/security', waitFor: '.app-shell__header' },
  { name: '18-session-list', path: '/auth/session', waitFor: '[class*="table"]' },
  { name: '19-profile', path: '/system/profile', waitFor: 'form' },
  { name: '20-generator-wizard', path: '/lowcode/generator', waitFor: '[class*="wizard"]' },
  { name: '21-module-manager', path: '/lowcode/module-manager', waitFor: '[class*="table"]' },
];

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

  const url = page.url();
  console.log(`  Current URL: ${url}`);

  try {
    await page.waitForSelector('.app-shell, [class*="layout"], [class*="container"]', { timeout: 10000 });
    console.log('  App shell found');
  } catch (e) {
    console.log('  Warning: App shell not found');
  }

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

async function main() {
  console.log('Starting page screenshot capture...');
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

    console.log('--- Capturing pages ---\n');
    for (const pageConfig of PAGES_TO_TEST) {
      try {
        console.log(`Capturing: ${pageConfig.name}`);

        await page.goto(`${BASE_URL}${pageConfig.path}`, { waitUntil: 'networkidle', timeout: 15000 });
        await page.waitForTimeout(800);

        if (pageConfig.waitFor) {
          try {
            await page.waitForSelector(pageConfig.waitFor, { timeout: 5000 });
          } catch {
            console.log(`  Note: Element "${pageConfig.waitFor}" not found`);
          }
        }

        await captureScreenshot(page, pageConfig.name);
      } catch (error) {
        console.log(`  Failed: ${error.message}`);
        await captureScreenshot(page, `${pageConfig.name}-error`);
      }
    }

    console.log('\n--- Capture complete ---');
    console.log(`Screenshots saved to: ${OUTPUT_DIR}`);

  } finally {
    await browser.close();
  }
}

main().catch(console.error);
