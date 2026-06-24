const CLIENT_CSRF_TOKEN_STORAGE_KEY = 'pantheon_csrf_token';
const CLIENT_SESSION_HINT_STORAGE_KEY = 'pantheon_session_hint';
const LEGACY_CSRF_COOKIE_PATTERN = /(?:^|;\s*)pantheon_csrf_token=([^;]+)/;

function readStorage(key: string): string {
  if (globalThis.localStorage === undefined) {
    return '';
  }
  try {
    return globalThis.localStorage.getItem(key) || '';
  } catch {
    return '';
  }
}

function writeStorage(key: string, value: string) {
  if (globalThis.localStorage === undefined) {
    return;
  }
  try {
    globalThis.localStorage.setItem(key, value);
  } catch {
    // Ignore storage failures and continue with in-memory app state.
  }
}

function removeStorage(key: string) {
  if (globalThis.localStorage === undefined) {
    return;
  }
  try {
    globalThis.localStorage.removeItem(key);
  } catch {
    // Ignore storage failures and continue with in-memory app state.
  }
}

export function markAuthSessionActive() {
  writeStorage(CLIENT_SESSION_HINT_STORAGE_KEY, '1');
}

export function persistCsrfToken(token: string) {
  const normalized = token.trim();
  if (!normalized) {
    return;
  }
  writeStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY, normalized);
  markAuthSessionActive();
}

export function readStoredCsrfToken(): string {
  hydrateLegacyAuthSession();
  return readStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY);
}

export function clearClientAuthSession() {
  removeStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY);
  removeStorage(CLIENT_SESSION_HINT_STORAGE_KEY);
}

export function hasAuthSessionHint(): boolean {
  hydrateLegacyAuthSession();
  return readStorage(CLIENT_SESSION_HINT_STORAGE_KEY) === '1' || readStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY) !== '';
}

export function hydrateLegacyAuthSession() {
  if (globalThis.document === undefined) {
    return;
  }
  if (readStorage(CLIENT_SESSION_HINT_STORAGE_KEY) === '1' && readStorage(CLIENT_CSRF_TOKEN_STORAGE_KEY) !== '') {
    return;
  }
  const match = globalThis.document.cookie.match(LEGACY_CSRF_COOKIE_PATTERN);
  if (!match?.[1]) {
    return;
  }
  persistCsrfToken(decodeURIComponent(match[1]));
}
