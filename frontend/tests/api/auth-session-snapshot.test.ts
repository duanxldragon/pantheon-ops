import assert from 'node:assert/strict';
import test from 'node:test';

import {
  COOKIE_TOKEN_PLACEHOLDER,
  clearStoredAuthUser,
  getBootstrappedAuthSession,
  persistAuthUser,
  readStoredAuthUser,
} from '../../src/core/auth/sessionSnapshot.ts';

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

test('getBootstrappedAuthSession restores placeholder tokens and stored user info when a client session hint exists', () => {
  const record = installStorage({
    pantheon_session_hint: '1',
  });

  persistAuthUser({
    id: 1,
    username: 'admin',
    nickname: '管理员',
    roles: ['admin'],
    perms: ['platform:dashboard:view'],
  });

  const session = getBootstrappedAuthSession();

  assert.equal(session.token, COOKIE_TOKEN_PLACEHOLDER);
  assert.equal(session.refreshToken, COOKIE_TOKEN_PLACEHOLDER);
  assert.equal(session.userInfo?.username, 'admin');
  assert.equal(readStoredAuthUser()?.nickname, '管理员');
  assert.equal(record.pantheon_auth_user.includes('"username":"admin"'), true);
});

test('getBootstrappedAuthSession ignores stored user info when no client session hint exists', () => {
  installStorage();
  persistAuthUser({
    id: 2,
    username: 'stale-admin',
    nickname: '过期管理员',
  });

  const session = getBootstrappedAuthSession();

  assert.equal(session.token, null);
  assert.equal(session.refreshToken, null);
  assert.equal(session.userInfo, null);
});

test('clearStoredAuthUser removes the persisted auth snapshot', () => {
  installStorage({
    pantheon_session_hint: '1',
  });
  persistAuthUser({
    id: 3,
    username: 'operator',
    nickname: '操作员',
  });

  clearStoredAuthUser();

  assert.equal(readStoredAuthUser(), null);
});
