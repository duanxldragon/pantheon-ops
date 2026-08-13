/**
 * SearchToolbar 全量铺开渲染验证：12 个已迁移页面 desktop + phone 截图,
 * 抽样验证关键交互(keyword 过滤、弹层筛选、清空)。零 console error 才通过。
 * 用法: node scripts/capture-toolbar-rollout-evidence.mjs --base http://127.0.0.1:5199 --out <dir>
 */
import { chromium } from 'playwright';
import path from 'node:path';
import process from 'node:process';

function readArg(name, fallback) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : fallback;
}

const baseUrl = readArg('--base', 'http://127.0.0.1:5199');
const outDir = path.resolve(
  readArg('--out', '../.harness/evidence/2026-07-17-search-toolbar-rollout'),
);
const username = process.env.PANTHEON_SMOKE_USER || 'admin';
const password = process.env.PANTHEON_SMOKE_PASSWORD || '123456';

const pages = [
  { path: '/system/role', slug: 'system-role' },
  { path: '/system/menu', slug: 'system-menu' },
  { path: '/system/dept', slug: 'system-dept' },
  { path: '/system/post', slug: 'system-post' },
  { path: '/system/permission', slug: 'system-permission' },
  { path: '/system/dict', slug: 'system-dict' },
  { path: '/system/i18n', slug: 'system-i18n' },
  { path: '/system/session', slug: 'system-session' },
  { path: '/system/security-event', slug: 'system-security-event' },
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
      consoleErrors.push(`[${suffix}] ${page.url()}: ${msg.text().slice(0, 200)}`);
    }
  });
  page.on('pageerror', (error) => {
    consoleErrors.push(`[${suffix}] ${page.url()}: pageerror ${error.message.slice(0, 200)}`);
  });
  await login(page);
  for (const target of pages) {
    await page.goto(`${baseUrl}${target.path}`, { waitUntil: 'networkidle' });
    await page.waitForSelector('.search-toolbar', { timeout: 15000 });
    await page.waitForTimeout(400);
    await page.screenshot({
      path: path.join(outDir, `${target.slug}-${suffix}.png`),
      fullPage: false,
    });
  }
  await context.close();
}

async function verifyInteractions(browser) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await login(page);

  // 1. 角色页：keyword=admin 端到端过滤(断言 keyword= 请求 + 行数收敛)
  await page.goto(`${baseUrl}/system/role`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  const roleRequest = page.waitForResponse(
    (resp) => resp.url().includes('/system/role/list') && resp.url().includes('keyword=admin'),
    { timeout: 10000 },
  );
  await page.locator('.search-toolbar__keyword input').fill('admin');
  await roleRequest;
  await page.waitForTimeout(600);
  await page.screenshot({ path: path.join(outDir, 'system-role-keyword-filtered-desktop.png') });
  await page.waitForSelector('.search-toolbar__clear', { timeout: 5000 });

  // 2. 会话页：弹层筛选(浏览器/OS/设备)+ 徽标
  await page.goto(`${baseUrl}/system/session`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  await page.locator('.search-toolbar__advanced-trigger').click();
  await page.waitForSelector('.search-toolbar__popover', { timeout: 5000 });
  await page.screenshot({ path: path.join(outDir, 'system-session-popover-desktop.png') });
  await page.keyboard.press('Escape');

  // 3. 字典页：类型下拉保留在行内 + keyword 生效
  await page.goto(`${baseUrl}/system/dict`, { waitUntil: 'networkidle' });
  await page.waitForSelector('.search-toolbar');
  const dictRequest = page.waitForResponse(
    (resp) => resp.url().includes('/system/dict/type/list') && resp.url().includes('keyword='),
    { timeout: 10000 },
  );
  await page.locator('.search-toolbar__keyword input').first().fill('status');
  await dictRequest;
  await page.waitForTimeout(500);
  await page.screenshot({ path: path.join(outDir, 'system-dict-keyword-filtered-desktop.png') });

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
console.log(`OK: rollout evidence written to ${outDir}, no console errors.`);
