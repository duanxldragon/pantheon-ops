import { hasAuthSessionHint } from './clientSession.ts';
import type { UserInfo } from '../../store/authTypes.ts';

const AUTH_USER_STORAGE_KEY = 'pantheon_auth_user';
export const COOKIE_TOKEN_PLACEHOLDER = '_cookie';

type BootstrappedAuthSession = {
  token: string | null;
  refreshToken: string | null;
  userInfo: UserInfo | null;
};

function canUseStorage() {
  return globalThis.localStorage !== undefined;
}

export function readStoredAuthUser(): UserInfo | null {
  if (!canUseStorage()) {
    return null;
  }
  try {
    const rawValue = globalThis.localStorage.getItem(AUTH_USER_STORAGE_KEY);
    if (!rawValue) {
      return null;
    }
    return JSON.parse(rawValue) as UserInfo;
  } catch {
    return null;
  }
}

export function persistAuthUser(userInfo: UserInfo) {
  if (!canUseStorage()) {
    return;
  }
  try {
    globalThis.localStorage.setItem(AUTH_USER_STORAGE_KEY, JSON.stringify(userInfo));
  } catch {
    // Ignore storage write failures and keep the in-memory session state usable.
  }
}

export function clearStoredAuthUser() {
  if (!canUseStorage()) {
    return;
  }
  try {
    globalThis.localStorage.removeItem(AUTH_USER_STORAGE_KEY);
  } catch {
    // Ignore storage write failures and keep the in-memory session state usable.
  }
}

export function getBootstrappedAuthSession(): BootstrappedAuthSession {
  if (!hasAuthSessionHint()) {
    return {
      token: null,
      refreshToken: null,
      userInfo: null,
    };
  }

  return {
    token: COOKIE_TOKEN_PLACEHOLDER,
    refreshToken: COOKIE_TOKEN_PLACEHOLDER,
    userInfo: readStoredAuthUser(),
  };
}
