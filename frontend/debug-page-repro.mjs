import { chromium } from 'playwright';

const base = 'http://127.0.0.1:5273';
const apiBase = 'http://127.0.0.1:8081/api/v1';

const browser = await chromium.launch();
const page = await browser.newPage();
const consoleErrors = [];
page.on('console', (msg) => {
  if (msg.type() === 'error') consoleErrors.push(msg.text());
});
page.on('pageerror', (err) => consoleErrors.push(`PAGEERROR: ${err.message}`));
page.on('response', (res) => {
  if (res.status() >= 500) console.log(`HTTP ${res.status()}: ${res.url()}`);
});

// login by API
const resp = await page.request.post(`${apiBase}/auth/login`, {
  data: { username: 'admin', password: '123456' },
});
const payload = await resp.json();
console.log('login code:', payload.code);
const setCookies = resp.headers()['set-cookie'] || '';
const csrf = (setCookies.match(/pantheon_csrf_token=([^;]+)/) || [])[1];
const access = (setCookies.match(/pantheon_access_token=([^;]+)/) || [])[1];

await page.context().addCookies([
  { name: 'pantheon_csrf_token', value: csrf || '', url: base },
  { name: 'pantheon_access_token', value: access || '', url: base },
  { name: 'pantheon_refresh_token', value: (setCookies.match(/pantheon_refresh_token=([^;]+)/) || [])[1] || '', url: base },
]);

await page.goto(`${base}/business/cmdb/host`, { waitUntil: 'networkidle', timeout: 30000 }).catch((e) => console.log('goto err:', e.message));
await page.waitForTimeout(3000);

const body = await page.locator('body').innerText().catch(() => '<no body>');
console.log('URL:', page.url());
console.log('=== BODY (first 1200) ===');
console.log(body.slice(0, 1200));
console.log('=== CONSOLE ERRORS ===');
console.log(consoleErrors.slice(0, 15).join('\n---\n'));
await page.screenshot({ path: 'debug-page-repro.png', fullPage: false }).catch(() => {});
await browser.close();
