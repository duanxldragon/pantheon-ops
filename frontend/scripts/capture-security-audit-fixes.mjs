/**
 * 安全审计 UI 修复取证：截取 2026-07-18 8 项修复的渲染证据。
 * 覆盖: 时间范围选择器(日历/选中/时分列)、清理对话框(仅保留天数模式)、
 * 全局搜索触发器宽度、质量栏(隐藏三行描述)、安全中心侧栏。
 * 用法: node scripts/capture-security-audit-fixes.mjs --base http://127.0.0.1:5199 --out ../.harness/evidence/2026-07-18-security-audit-ui-fixes
 */
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

function readArg(name, fallback) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : fallback;
}

const baseUrl = readArg('--base', 'http://127.0.0.1:5199');
const outDir = path.resolve(
  readArg('--out', '../.harness/evidence/2026-07-18-security-audit-ui-fixes'),
);
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

  // 1+2+3. 登录日志:打开时间范围弹层(日历不换行/选中反馈/时分竖列)
  await page.goto(`${baseUrl}/system/login-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar', { timeout: 15000 });
  await shot(page, 'login-log-page');
  await page.locator('.auth-login-log-page__time-range-trigger').click();
  await page.waitForSelector('.arco-picker-range-container', { timeout: 8000 });
  await shot(page, 'login-log-time-range-panel');
  // 点击起始日期后再截一张,验证选中反馈(加粗+背景)
  const dateCell = page.locator('.arco-picker-cell-in-view').first();
  await dateCell.click();
  await shot(page, 'login-log-time-range-selecting');
  await page.keyboard.press('Escape');

  // 4+5. 清理对话框: 仅保留天数模式(range 模式已移除)
  const cleanupBtn = page.locator('button:has-text("清理")').first();
  if (await cleanupBtn.isVisible().catch(() => false)) {
    await cleanupBtn.click();
    await page.waitForSelector('.arco-modal', { timeout: 8000 });
    await shot(page, 'login-log-cleanup-dialog');
    await page.keyboard.press('Escape');
  }

  // 6. 全局搜索触发器(变宽后) + 打开弹窗
  await shot(page, 'header-search-trigger');
  await page.locator('.app-shell__search-trigger').click();
  await page.waitForSelector('.app-command', { timeout: 8000 });
  await shot(page, 'command-palette-open');
  await page.keyboard.press('Escape');

  // 7. 质量栏: 描述行已隐藏(仅 eyebrow + title 两行)
  await page.goto(`${baseUrl}/system/operation-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.governance-summary-bar', { timeout: 15000 });
  await shot(page, 'operation-log-summary-bar');

  // 8. 安全中心: 侧栏与主列平衡
  await page.goto(`${baseUrl}/auth/security`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.page-split-layout', { timeout: 15000 });
  await shot(page, 'security-center-top');
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await shot(page, 'security-center-bottom');

  // 会话管理与安全事件的清理对话框抽查
  await page.goto(`${baseUrl}/auth/session`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(800);
  await shot(page, 'session-list-page');
  await page.goto(`${baseUrl}/auth/security-event`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(800);
  await shot(page, 'security-event-page');

  await context.close();
  await browser.close();

  if (consoleErrors.length) {
    console.error('\nConsole errors detected:');
    for (const err of consoleErrors) console.error(`  ${err}`);
    process.exit(1);
  }
  console.log('\nAll captures complete, no console errors.');
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
