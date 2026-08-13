/**
 * SearchToolbar 试点页渲染验证：登录后截取 3 个试点页 desktop + phone 视口,
 * 并验证关键交互(下拉即查、弹层筛选、清空入口)。
 * 用法: node scripts/capture-search-toolbar-evidence.mjs --base http://127.0.0.1:5199 --out ../.harness/evidence/<task-id>
 */
import { chromium } from 'playwright';
import path from 'node:path';
import process from 'node:process';

function readArg(name, fallback) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : fallback;
}

const baseUrl = readArg('--base', 'http://127.0.0.1:5199');
const outDir = path.resolve(readArg('--out', '../.harness/evidence/2026-07-16-search-toolbar-pilot'));
const username = process.env.PANTHEON_SMOKE_USER || 'admin';
const password = process.env.PANTHEON_SMOKE_PASSWORD || '123456';

const pages = [
  { path: '/system/user', slug: 'system-user' },
  { path: '/system/operation-log', slug: 'system-operation-log' },
  { path: '/system/login-log', slug: 'system-login-log' },
];

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

async function captureAll(browser, viewport, suffix) {
  const context = await browser.newContext({ viewport });
  const page = await context.newPage();
  page.on('console', (msg) => {
    if (msg.type() === 'error') {
      consoleErrors.push(`[${suffix}] ${page.url()}: ${msg.text()}`);
    }
  });
  page.on('pageerror', (error) => {
    consoleErrors.push(`[${suffix}] ${page.url()}: pageerror ${error.message}`);
  });
  await login(page);
  for (const target of pages) {
    await page.goto(`${baseUrl}${target.path}`, { waitUntil: 'networkidle' });
    await page.waitForSelector('.search-toolbar', { timeout: 15000 });
    await page.waitForTimeout(500);
    await page.screenshot({ path: path.join(outDir, `${target.slug}-${suffix}.png`), fullPage: false });
  }
  await context.close();
}

async function verifyInteractions(browser) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await login(page);

  // 1. 用户页：状态下拉即查 + 清空入口出现
  await page.goto(`${baseUrl}/system/user`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  const listRequest = page.waitForResponse(
    (resp) => resp.url().includes('/system/user/list') && resp.url().includes('status=1'),
    { timeout: 10000 },
  );
  await page.locator('.search-toolbar__inline .arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '启用' }).first().click();
  await listRequest;
  await page.waitForSelector('.search-toolbar__clear', { timeout: 5000 });
  await page.screenshot({ path: path.join(outDir, 'system-user-active-filter-desktop.png') });
  await page.locator('.search-toolbar__clear').click();
  await page.waitForFunction(
    () => !document.querySelector('.search-toolbar__clear'),
    { timeout: 5000 },
  );

  // 2. 用户页：关键词防抖搜索触发 keyword= 请求
  const keywordRequest = page.waitForResponse(
    (resp) => resp.url().includes('/system/user/list') && resp.url().includes('keyword=admin'),
    { timeout: 10000 },
  );
  await page.locator('.search-toolbar__keyword input').fill('admin');
  await keywordRequest;

  // 3. 操作日志：弹层筛选 + 徽标计数
  await page.goto(`${baseUrl}/system/operation-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  await page.locator('.search-toolbar__advanced-trigger').click();
  await page.waitForSelector('.search-toolbar__popover', { timeout: 5000 });
  await page.screenshot({ path: path.join(outDir, 'system-operation-log-popover-desktop.png') });
  const categoryRequest = page.waitForResponse(
    (resp) => resp.url().includes('/system/operation-log/list') && resp.url().includes('failureCategory=validation'),
    { timeout: 10000 },
  );
  await page.locator('.search-toolbar__popover .arco-select-view').first().click();
  await page.locator('.arco-select-option').filter({ hasText: '参数/校验失败' }).first().click();
  await categoryRequest;
  await page.keyboard.press('Escape');
  await page.waitForSelector('.search-toolbar__badge', { timeout: 5000 });
  await page.screenshot({ path: path.join(outDir, 'system-operation-log-badge-desktop.png') });

  // 4. 登录日志：时间范围快捷选择仍然工作
  await page.goto(`${baseUrl}/system/login-log`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  await page.locator('.auth-login-log-page__time-range-trigger').click();
  await page.waitForTimeout(600);
  await page.screenshot({ path: path.join(outDir, 'system-login-log-timerange-desktop.png') });

  await context.close();
}

const browser = await chromium.launch();
try {
  await captureAll(browser, { width: 1440, height: 900 }, 'desktop');
  await captureAll(browser, { width: 390, height: 844 }, 'phone');
  await verifyInteractions(browser);
} finally {
  await browser.close();
}

if (consoleErrors.length > 0) {
  console.error('Console errors captured:');
  for (const line of consoleErrors) {
    console.error(`  ${line}`);
  }
  process.exit(1);
}
console.log(`OK: screenshots written to ${outDir}, no console errors.`);
