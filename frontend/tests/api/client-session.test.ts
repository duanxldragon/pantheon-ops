import assert from 'node:assert/strict';
import test from 'node:test';

import {
  clearClientAuthSession,
  hasAuthSessionHint,
  markAuthSessionActive,
  persistCsrfToken,
  readStoredCsrfToken,
} from '../../src/core/auth/clientSession.ts';

type StorageRecord = Record<string, string>;

function installStorage(record: StorageRecord = {}) {
  const storage = {
    getItem(key: string) {
      return key in record ? record[key] : null;
    },
    setItem(key: string, value: string) {
      record[key] = String(value);
    },
    removeItem(key: string) {
      delete record[key];
    },
  };

  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: storage,
  });

  return record;
}

test.beforeEach(() => {
  installStorage();
  clearClientAuthSession();
});

test('hasAuthSessionHint restores session from persisted storage hint', () => {
  installStorage({
    pantheon_session_hint: '1',
  });

  assert.equal(hasAuthSessionHint(), true);
});

test('markAuthSessionActive persists the storage hint', () => {
  const record = installStorage();

  markAuthSessionActive();

  assert.equal(record.pantheon_session_hint, '1');
  assert.equal(hasAuthSessionHint(), true);
});

test('persistCsrfToken stores in-memory csrf token and marks session active', () => {
  const record = installStorage();

  persistCsrfToken('csrf-token-1');

  assert.equal(readStoredCsrfToken(), 'csrf-token-1');
  assert.equal(record.pantheon_session_hint, '1');
  assert.equal(hasAuthSessionHint(), true);
});

test('readStoredCsrfToken restores csrf token from persisted storage', () => {
  installStorage({
    pantheon_csrf_token: 'csrf-token-from-storage',
  });

  assert.equal(readStoredCsrfToken(), 'csrf-token-from-storage');
});

test('clearClientAuthSession clears storage-backed auth hint', () => {
  const record = installStorage();

  markAuthSessionActive();
  persistCsrfToken('csrf-token-2');
  clearClientAuthSession();

  assert.equal(readStoredCsrfToken(), '');
  assert.equal('pantheon_session_hint' in record, false);
  assert.equal(hasAuthSessionHint(), false);
});
