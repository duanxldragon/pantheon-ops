/**
 * 快速复查:时间范围弹层宽度修复后的渲染 + 剩余未截取页面。
 * 用法: node scripts/capture-security-audit-recheck.mjs
 */
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const baseUrl = 'http://127.0.0.1:5199';
const outDir = path.resolve('../.harness/evidence/2026-07-18-security-audit-ui-fixes');
const username = process.env.PANTHEON_SMOKE_USER || 'admin';
const password = process.env.PANTHEON_SMOKE_PASSWORD || '123456';

fs.mkdirSync(outDir, { recursive: true });
const consoleErrors = [];

async function login(page) {
  await page.addInitScript(() => {
    try {
      localStorage.setItem('pantheon_lang', 'zh-CN');
      localStorage.setItem('pantheon_lang_explicit', '1');
    } catch {
      /* ignore */
    }
  });
  await page.goto(`${baseUrl}/login`, { waitUntil: 'networkidle' });
  await page.waitForSelector('form', { timeout: 10000 });
  await page
    .locator('input[placeholder*="username" i], input[placeholder*="用户"]')
    .first()
    .fill(username);
  await page.locator('input[type="password"]').first().fill(password);
  await page.locator('button[type="submit"], button:has-text("登录")').first().click();
  await page.waitForURL(/dashboard/, { timeout: 15000 });
}

async function shot(page, slug) {
  await page.waitForTimeout(400);
  await page.screenshot({ path: path.join(outDir, `${slug}.png`), fullPage: false });
  console.log(`captured: ${slug}`);
}

async function main() {
  const browser = await chromium.launch();
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  page.on('console', (msg) => {
    if (msg.type() === 'error') consoleErrors.push(`${page.url()}: ${msg.text()}`);
  });
  page.on('pageerror', (error) => consoleErrors.push(`${page.url()}: pageerror ${error.message}`));

  await login(page);

  // 时间范围弹层(720px 修复后)
  await page.goto(`${baseUrl}/system/login-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar', { timeout: 15000 });
  await page.locator('.auth-login-log-page__time-range-trigger').click();
  await page.waitForSelector('.arco-picker-range-container', { timeout: 8000 });
  await shot(page, 'login-log-time-range-panel');
  // 点选起始日期,验证选中反馈
  await page.locator('.arco-picker-cell-in-view').nth(10).click();
  await shot(page, 'login-log-time-range-selecting');
  // 点击"选择时间"看时分竖列
  const selectTimeBtn = page.locator('.arco-picker-btn-select-time');
  if (await selectTimeBtn.isVisible().catch(() => false)) {
    await selectTimeBtn.click();
    await shot(page, 'login-log-time-column');
  }
  await page.keyboard.press('Escape');

  // command palette
  await page.locator('.app-shell__search-trigger').click();
  await page.waitForSelector('.app-command', { timeout: 8000 });
  await shot(page, 'command-palette-open');
  await page.keyboard.press('Escape');

  // 质量栏(desc 隐藏后)
  await page.goto(`${baseUrl}/system/operation-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.governance-summary-bar', { timeout: 15000 });
  await shot(page, 'operation-log-summary-bar');

  // 安全中心
  await page.goto(`${baseUrl}/auth/security`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.page-split-layout', { timeout: 15000 });
  await shot(page, 'security-center-top');
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await shot(page, 'security-center-bottom');

  // 会话/安全事件页(清理对话框已仅保留天数)
  await page.goto(`${baseUrl}/system/session`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(800);
  await shot(page, 'session-list-page');
  await page.goto(`${baseUrl}/system/security-event`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(800);
  await shot(page, 'security-event-page');

  await context.close();
  await browser.close();

  if (consoleErrors.length) {
    console.error('\nConsole errors detected:');
    for (const err of consoleErrors) console.error(`  ${err}`);
    process.exit(1);
  }
  console.log('\nAll recheck captures complete, no console errors.');
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
