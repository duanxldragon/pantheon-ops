import { chromium } from 'playwright';
import { mkdirSync } from 'node:fs';
import { join } from 'node:path';

const BASE = 'http://localhost:5173';
const OUT_DIR = join(process.env.HOME || process.env.USERPROFILE, '.gstack/projects/duanxldragon-pantheon-ops/designs/design-audit-20260501/screenshots');
mkdirSync(OUT_DIR, { recursive: true });

// Correct routes - frontend uses singular nouns
const pages = [
  { name: '01-login', path: '/login', desc: 'Login' },
  { name: '02-dashboard', path: '/dashboard', desc: 'Dashboard' },
  { name: '03-user', path: '/system/user', desc: 'User Management' },
  { name: '04-role', path: '/system/role', desc: 'Role Management' },
  { name: '05-dept', path: '/system/dept', desc: 'Department Management' },
  { name: '06-menu', path: '/system/menu', desc: 'Menu Management' },
  { name: '07-permission', path: '/system/permission', desc: 'Permission Management' },
  { name: '08-post', path: '/system/post', desc: 'Position/Post Management' },
  { name: '09-setting', path: '/system/setting', desc: 'System Settings' },
  { name: '10-operation-log', path: '/system/operation-log', desc: 'Operation Logs' },
  { name: '11-dict', path: '/system/dict', desc: 'Dictionary Management' },
  { name: '12-i18n', path: '/system/i18n', desc: 'i18n Management' },
  { name: '13-modules', path: '/system/modules', desc: 'Module Manager' },
];

const findings = [];

const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

// Capture console errors
let consoleErrors = [];
page.on('console', msg => {
  if (msg.type() === 'error') consoleErrors.push(msg.text());
});

// --- LOGIN ---
console.log('=== LOGIN ===');
await page.goto(BASE + '/login', { waitUntil: 'networkidle', timeout: 15000 });
await page.waitForTimeout(1000);

const usernameInput = page.locator('input[placeholder="请输入用户名"]');
await usernameInput.waitFor({ timeout: 10000 });
await usernameInput.fill('admin');
const passwordInput = page.locator('input[placeholder="请输入密码"]');
await passwordInput.fill('123456');
await page.locator('button[type="submit"]').first().click();
await page.waitForTimeout(2000);
await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => {});
console.log('Login result URL:', page.url());

// --- VISIT EACH PAGE ---
for (const p of pages) {
  console.log(`\n=== ${p.name}: ${p.desc} ===`);
  consoleErrors = [];

  try {
    await page.goto(BASE + p.path, { waitUntil: 'networkidle', timeout: 20000 });
    await page.waitForTimeout(1500);

    // Evaluate page state
    const info = await page.evaluate(() => {
      const r = {
        url: location.href,
        title: document.title,
        fontFamily: '',
        is404: false,
        hasHero: false,
        hasKpis: false,
        hasTable: false,
        hasFilter: false,
        hasErrorState: false,
        hasEmptyState: false,
        contentPreview: '',
        cardRadii: [],
        gradientBgs: [],
        inlineArcoTokens: 0,
      };

      // Font
      r.fontFamily = getComputedStyle(document.body).fontFamily?.split(',')[0]?.replace(/"/g, '') || '';

      // 404 check
      r.is404 = !!document.querySelector('.arco-result-is-404, .page-result');

      // Error/empty states
      r.hasErrorState = !!document.querySelector('.page-error, .page-network-error, .page-server-error');
      r.hasEmptyState = !!document.querySelector('.page-empty');

      // Page structure
      r.hasHero = !!document.querySelector('.system-page-hero, .dashboard-hero-card');
      r.hasKpis = !!document.querySelector('.system-page-kpi-grid');
      r.hasTable = !!document.querySelector('.arco-table-container, .arco-table');
      r.hasFilter = !!document.querySelector('.filter-panel');

      // Content preview
      const main = document.querySelector('.system-page-template, .page-container, .dashboard-page, .setting-page, .arco-layout-content');
      if (main) r.contentPreview = main.textContent?.trim().slice(0, 250) || '';

      // Card border-radius samples
      document.querySelectorAll('[class*="card"], .arco-card').forEach(c => {
        const br = getComputedStyle(c).borderRadius;
        if (br && br !== '0px') r.cardRadii.push(br);
      });
      r.cardRadii = [...new Set(r.cardRadii)].slice(0, 10);

      // Gradient backgrounds
      document.querySelectorAll('*').forEach(el => {
        const bg = getComputedStyle(el).backgroundImage || '';
        if (bg.includes('gradient')) {
          r.gradientBgs.push(bg.slice(0, 100));
        }
      });
      r.gradientBgs = [...new Set(r.gradientBgs)].slice(0, 5);

      return r;
    });

    // Screenshot
    await page.screenshot({ path: join(OUT_DIR, `${p.name}.png`), fullPage: true });

    // Report
    const status = info.is404 ? '404' : info.hasErrorState ? 'ERROR' : info.hasEmptyState ? 'EMPTY' : 'OK';
    console.log(`  Status: ${status}`);
    console.log(`  URL: ${info.url}`);
    console.log(`  Font: ${info.fontFamily}`);
    console.log(`  Hero:${info.hasHero} KPIs:${info.hasKpis} Table:${info.hasTable} Filter:${info.hasFilter}`);
    console.log(`  Card radii: ${info.cardRadii.join(', ')}`);
    if (info.gradientBgs.length) console.log(`  Gradient BGs: ${info.gradientBgs.length}`);
    if (consoleErrors.length) console.log(`  Console errors: ${consoleErrors.length}`);
    if (info.contentPreview) console.log(`  Content: ${info.contentPreview.slice(0, 180)}`);

    findings.push({
      name: p.name,
      desc: p.desc,
      path: p.path,
      status,
      font: info.fontFamily,
      is404: info.is404,
      hasHero: info.hasHero,
      hasTable: info.hasTable,
      cardRadii: info.cardRadii,
      consoleErrors: consoleErrors.length,
    });

    if (info.is404) {
      findings[findings.length - 1].note = '404 - route may be incorrect or requires menu permission';
    }

  } catch (e) {
    console.log(`  ERROR: ${e.message}`);
    findings.push({ name: p.name, desc: p.desc, path: p.path, status: 'NAV_ERROR', error: e.message });
  }
}

await browser.close();

// SUMMARY
console.log('\n\n========================================');
console.log('DESIGN AUDIT VERIFICATION SUMMARY');
console.log('========================================');
const statusCounts = {};
for (const f of findings) {
  statusCounts[f.status] = (statusCounts[f.status] || 0) + 1;
}
console.log('Statuses:', JSON.stringify(statusCounts));

const fonts = [...new Set(findings.map(f => f.font).filter(Boolean))];
console.log('Font families detected:', fonts.join(' | '));

const allRadii = [...new Set(findings.flatMap(f => f.cardRadii || []))];
console.log('Unique card radii:', allRadii.join(', '));

console.log('\n--- Per Page ---');
for (const f of findings) {
  const flag = f.status !== 'OK' ? ` [${f.status}]` : '';
  console.log(`${f.name}: ${f.desc}${flag} | Font: ${f.font} | Hero:${f.hasHero} | Table:${f.hasTable}${f.note ? ' | ' + f.note : ''}`);
}

console.log('\nAll screenshots saved to:', OUT_DIR);
