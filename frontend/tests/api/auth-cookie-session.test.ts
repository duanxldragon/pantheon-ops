import assert from 'node:assert/strict';
import test from 'node:test';

import {
  extractCookieValue,
  readAuthCookieSession,
} from '../../scripts/lib/auth-cookie-session.mjs';

test('extractCookieValue parses values from a joined set-cookie header', () => {
  const setCookieHeader = [
    'pantheon_access_token=access-token-1; Path=/; HttpOnly; SameSite=Strict',
    'pantheon_refresh_token=refresh-token-1; Path=/; HttpOnly; SameSite=Strict',
    'pantheon_csrf_token=csrf-cookie-1; Path=/; HttpOnly; SameSite=Strict',
  ].join(', ');

  assert.equal(extractCookieValue(setCookieHeader, 'pantheon_access_token'), 'access-token-1');
  assert.equal(extractCookieValue(setCookieHeader, 'pantheon_refresh_token'), 'refresh-token-1');
  assert.equal(extractCookieValue(setCookieHeader, 'pantheon_csrf_token'), 'csrf-cookie-1');
});

test('readAuthCookieSession reads access refresh and csrf values from headers', () => {
  const headers = new Headers();
  headers.set(
    'set-cookie',
    [
      'pantheon_access_token=access-token-2; Path=/; HttpOnly; SameSite=Strict',
      'pantheon_refresh_token=refresh-token-2; Path=/; HttpOnly; SameSite=Strict',
      'pantheon_csrf_token=csrf-cookie-2; Path=/; HttpOnly; SameSite=Strict',
    ].join(', '),
  );
  headers.set('x-csrf-token', 'csrf-header-2');

  assert.deepEqual(readAuthCookieSession(headers), {
    accessToken: 'access-token-2',
    refreshToken: 'refresh-token-2',
    csrfToken: 'csrf-header-2',
  });
});

test('readAuthCookieSession supports newline-delimited set-cookie headers from Playwright', () => {
  const session = readAuthCookieSession({
    'set-cookie': [
      'pantheon_access_token=access-token-2b; Path=/; HttpOnly; SameSite=Strict',
      'pantheon_refresh_token=refresh-token-2b; Path=/; HttpOnly; SameSite=Strict',
      'pantheon_csrf_token=csrf-cookie-2b; Path=/; HttpOnly; SameSite=Strict',
    ].join('\n'),
    'x-csrf-token': 'csrf-header-2b',
  });

  assert.deepEqual(session, {
    accessToken: 'access-token-2b',
    refreshToken: 'refresh-token-2b',
    csrfToken: 'csrf-header-2b',
  });
});

test('readAuthCookieSession supports plain header objects and optional refresh cookies', () => {
  const session = readAuthCookieSession(
    {
      'set-cookie': [
        'pantheon_access_token=access-token-3; Path=/; HttpOnly; SameSite=Strict',
        'pantheon_csrf_token=csrf-cookie-3; Path=/; HttpOnly; SameSite=Strict',
      ].join(', '),
    },
    { includeRefreshToken: false },
  );

  assert.equal(session.accessToken, 'access-token-3');
  assert.equal(session.csrfToken, 'csrf-cookie-3');
  assert.equal(session.refreshToken, undefined);
});

test('readAuthCookieSession throws when the auth response does not include an access cookie', () => {
  assert.throws(
    () =>
      readAuthCookieSession(
        {
          'set-cookie': 'pantheon_csrf_token=csrf-cookie-4; Path=/; HttpOnly; SameSite=Strict',
        },
        { includeRefreshToken: false },
      ),
    /pantheon_access_token/,
  );
});

test('readAuthCookieSession throws when the auth response does not include a refresh cookie by default', () => {
  assert.throws(
    () =>
      readAuthCookieSession({
        'set-cookie': [
          'pantheon_access_token=access-token-5; Path=/; HttpOnly; SameSite=Strict',
          'pantheon_csrf_token=csrf-cookie-5; Path=/; HttpOnly; SameSite=Strict',
        ].join(', '),
      }),
    /pantheon_refresh_token/,
  );
});
