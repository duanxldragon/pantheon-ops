import test from 'node:test';
import assert from 'node:assert/strict';
import { buildCookieHeader, extractCookieValue } from './smoke-auth.mjs';

test('extractCookieValue reads cookies from combined set-cookie header', () => {
  const header =
    'pantheon_access_token=access-token; Path=/; HttpOnly, pantheon_refresh_token=refresh-token; Path=/; HttpOnly, pantheon_csrf_token=csrf-token; Path=/';

  assert.equal(extractCookieValue(header, 'pantheon_access_token'), 'access-token');
  assert.equal(extractCookieValue(header, 'pantheon_refresh_token'), 'refresh-token');
  assert.equal(extractCookieValue(header, 'pantheon_csrf_token'), 'csrf-token');
  assert.equal(extractCookieValue(header, 'missing_cookie'), null);
});

test('buildCookieHeader keeps auth and csrf cookies in one request header', () => {
  const header = buildCookieHeader([
    { name: 'pantheon_access_token', value: 'access-token' },
    { name: 'pantheon_refresh_token', value: 'refresh-token' },
    { name: 'pantheon_csrf_token', value: 'csrf-token' },
  ]);

  assert.equal(
    header,
    'pantheon_access_token=access-token; pantheon_refresh_token=refresh-token; pantheon_csrf_token=csrf-token',
  );
});
