/**
 * 第二轮审计 UI 取证：清理按钮移除 + 4 页搜索栏时间范围 + 治理栏两行收敛。
 * 用法: node scripts/capture-audit-round2.mjs
 */
import { chromium } from 'playwright';
import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const baseUrl = 'http://127.0.0.1:5199';
const outDir = path.resolve('../.harness/evidence/2026-07-19-audit-toolbar-cleanup');
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
  await page.waitForTimeout(500);
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

  const pages = [
    { url: '/system/login-log', slug: 'login-log', wait: '.search-toolbar' },
    { url: '/system/session', slug: 'session', wait: '.search-toolbar' },
    { url: '/system/operation-log', slug: 'operation-log', wait: '.search-toolbar' },
    { url: '/system/security-event', slug: 'security-event', wait: '.search-toolbar' },
  ];

  for (const item of pages) {
    await page.goto(`${baseUrl}${item.url}`, { waitUntil: 'networkidle' });
    await page.waitForSelector(item.wait, { timeout: 15000 });
    await shot(page, `${item.slug}-page`);
    // 打开时间范围弹层验证共享组件渲染
    const trigger = page.locator('.time-range-filter__trigger').first();
    if (await trigger.isVisible().catch(() => false)) {
      await trigger.click();
      await page.waitForSelector('.arco-picker-range-container', { timeout: 8000 });
      await shot(page, `${item.slug}-time-range`);
      await page.keyboard.press('Escape');
      await page.waitForTimeout(300);
    }
  }

  // 治理栏高度收敛抽查（非审计页）
  await page.goto(`${baseUrl}/system/user`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.governance-summary-bar', { timeout: 15000 });
  await shot(page, 'user-list-summary-bar');

  await context.close();
  await browser.close();

  if (consoleErrors.length) {
    console.error('\nConsole errors detected:');
    for (const err of consoleErrors) console.error(`  ${err}`);
    process.exit(1);
  }
  console.log('\nAll round-2 captures complete, no console errors.');
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
