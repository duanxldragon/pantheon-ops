const CLIENT_SESSION_HINT_STORAGE_KEY = 'pantheon_session_hint';
const CLIENT_CSRF_TOKEN_STORAGE_KEY = 'pantheon_csrf_token';

function canUseStorage() {
  return globalThis.localStorage !== undefined;
}

function readStorage(key: string): string | null {
  if (!canUseStorage()) {
    return null;
  }
  try {
    return globalThis.localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStorage(key: string, value: string) {
  if (!canUseStorage()) {
    return;
  }
  try {
    globalThis.localStorage.setItem(key, value);
  } catch {
    // Ignore storage failures and continue with in-memory app state.
  }
}

function removeStorage(key: string) {
  if (!canUseStorage()) {
    return;
  }
  try {
    globalThis.localStorage.removeItem(key);
  } catch {
    // Ignore storage failures and continue with in-memory app state.
  }
}

let inMemoryCsrfToken = '';
let sessionHintStored = false;

export function markAuthSessionActive() {
  sessionHintStored = true;
  writeStorage(CLIENT_SESSION_HINT_STORAGE_KEY, '1');
}

export function persistCsrfToken(token: string) {
  const normalized = token.trim();
  if (!normalized) {
    return;
  }
  inMemoryCsrfToken = normalized;
  writeStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY, normalized);
  markAuthSessionActive();
}

export function readStoredCsrfToken(): string {
  if (!inMemoryCsrfToken) {
    const persistedToken = readStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY)?.trim() || '';
    if (persistedToken) {
      inMemoryCsrfToken = persistedToken;
    }
  }
  return inMemoryCsrfToken;
}

export function clearClientAuthSession() {
  inMemoryCsrfToken = '';
  sessionHintStored = false;
  removeStorage(CLIENT_SESSION_HINT_STORAGE_KEY);
  removeStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY);
}

export function hasAuthSessionHint(): boolean {
  return sessionHintStored || inMemoryCsrfToken !== '' || readStorage(CLIENT_SESSION_HINT_STORAGE_KEY) === '1';
}
