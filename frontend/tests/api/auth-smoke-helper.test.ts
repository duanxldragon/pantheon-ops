import assert from 'node:assert/strict';
import test from 'node:test';

import { chromium } from 'playwright';

import {
  installClientSession,
  primeChineseLocale,
  type BrowserLoginResult,
} from '../smoke/helpers/auth.ts';

const fakeLogin: BrowserLoginResult = {
  accessToken: 'access-token-test',
  refreshToken: 'refresh-token-test',
  username: 'admin',
  password: '123456',
  csrfToken: 'csrf-token-test',
};

test('auth smoke helper does not emit storage security errors on about:blank pages', async (t) => {
  const browser = await chromium.launch();
  t.after(async () => {
    await browser.close();
  });

  const page = await browser.newPage();
  const runtimeErrors: string[] = [];

  page.on('console', (message) => {
    if (message.type() === 'error') {
      runtimeErrors.push(message.text());
    }
  });
  page.on('pageerror', (error) => {
    runtimeErrors.push(error.message);
  });

  await primeChineseLocale(page);
  await installClientSession(page, fakeLogin);
  await page.goto('about:blank', { waitUntil: 'load' });
  // Flush two frames so queued error/pageerror events are delivered before the
  // negative assertion — replaces a fixed 50ms delay.
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
      }),
  );

  assert.deepEqual(runtimeErrors, []);

  const cookieNames = new Set(
    (await page.context().cookies('http://127.0.0.1:5173')).map((cookie) => cookie.name),
  );
  assert.equal(cookieNames.has('pantheon_access_token'), true);
  assert.equal(cookieNames.has('pantheon_refresh_token'), true);
  assert.equal(cookieNames.has('pantheon_csrf_token'), true);
});
