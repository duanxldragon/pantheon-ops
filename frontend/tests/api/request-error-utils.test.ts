import assert from 'node:assert/strict';
import test from 'node:test';

import {
  resolveBusinessErrorKind,
  resolveFallbackMessageKey,
  resolveTransportErrorKind,
  shouldSuppressAuthMessage,
  type RequestErrorKind,
} from '../../src/api/requestErrorUtils.js';

function createAxiosError(overrides: Record<string, unknown>) {
  return {
    code: undefined,
    message: 'request failed',
    response: undefined,
    ...overrides,
  };
}

test('resolveBusinessErrorKind classifies common business status codes', () => {
  assert.equal(resolveBusinessErrorKind(401), 'unauthorized');
  assert.equal(resolveBusinessErrorKind(403), 'forbidden');
  assert.equal(resolveBusinessErrorKind(500), 'server');
  assert.equal(resolveBusinessErrorKind(200), 'business');
});

test('resolveTransportErrorKind maps timeout, network, and response failures', () => {
  assert.equal(resolveTransportErrorKind(createAxiosError({ code: 'ECONNABORTED' })), 'timeout');
  assert.equal(resolveTransportErrorKind(createAxiosError({ message: 'Request timeout' })), 'timeout');
  assert.equal(resolveTransportErrorKind(createAxiosError({ response: undefined })), 'network');
  assert.equal(resolveTransportErrorKind(createAxiosError({ response: { status: 401 } })), 'unauthorized');
  assert.equal(resolveTransportErrorKind(createAxiosError({ response: { status: 403 } })), 'forbidden');
  assert.equal(resolveTransportErrorKind(createAxiosError({ response: { status: 503 } })), 'server');
});

test('resolveFallbackMessageKey preserves explicit fallbacks and network defaults', () => {
  assert.equal(resolveFallbackMessageKey('timeout'), 'network.timeout');
  assert.equal(resolveFallbackMessageKey('network'), 'network.error');
  assert.equal(resolveFallbackMessageKey('server'), 'request.failed');
  assert.equal(resolveFallbackMessageKey('business', 'custom.fallback'), 'custom.fallback');
});

test('shouldSuppressAuthMessage only hides handled auth prompts', () => {
  const authKind: RequestErrorKind = 'unauthorized';
  assert.equal(shouldSuppressAuthMessage('session.invalid', authKind, '/dashboard', false), false);
  assert.equal(shouldSuppressAuthMessage('token.refresh.expired', authKind, '/dashboard', false), false);
  assert.equal(shouldSuppressAuthMessage('session.invalid', authKind, '/login', false), true);
  assert.equal(shouldSuppressAuthMessage('session.invalid', authKind, '/dashboard', true), true);
  assert.equal(shouldSuppressAuthMessage('common.error', 'business', '/dashboard', false), false);
});
