import assert from 'node:assert/strict';
import test from 'node:test';

import { attachCookieTokens, collectSetCookieHeader } from './smoke-auth.mjs';

test('attachCookieTokens restores cookie-only auth tokens', () => {
  const response = {
    headers: {
      getSetCookie: () => [
        'pantheon_access_token=access-1; Path=/; HttpOnly; SameSite=Strict',
        'pantheon_refresh_token=refresh-1; Path=/; HttpOnly; SameSite=Strict',
        'pantheon_csrf_token=csrf-1; Path=/; SameSite=Strict',
      ],
    },
  };

  assert.match(collectSetCookieHeader(response), /pantheon_access_token=access-1/);
  assert.deepEqual(attachCookieTokens(response, { code: 200, data: { tokenType: 'Bearer' } }), {
    code: 200,
    data: {
      tokenType: 'Bearer',
      accessToken: 'access-1',
      refreshToken: 'refresh-1',
    },
  });
});

test('attachCookieTokens keeps explicit response tokens', () => {
  const response = {
    headers: {
      get: () =>
        'pantheon_access_token=cookie-access; Path=/, pantheon_refresh_token=cookie-refresh; Path=/',
    },
  };

  const payload = attachCookieTokens(response, {
    code: 200,
    data: { accessToken: 'body-access', refreshToken: 'body-refresh' },
  });
  assert.equal(payload.data.accessToken, 'body-access');
  assert.equal(payload.data.refreshToken, 'body-refresh');
});
