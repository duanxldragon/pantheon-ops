const PUBLIC_SETTINGS_STORAGE_KEY = 'pantheon_public_settings';
export const LANGUAGE_STORAGE_KEY = 'pantheon_lang';
export const LANGUAGE_EXPLICIT_STORAGE_KEY = 'pantheon_lang_explicit';
const FALLBACK_LANGUAGE = 'zh-CN';

function readStoredPublicSettings() {
  if (globalThis.document === undefined) {
    return {} as Record<string, string>;
  }
  try {
    const rawValue = globalThis.localStorage.getItem(PUBLIC_SETTINGS_STORAGE_KEY);
    if (!rawValue) {
      return {} as Record<string, string>;
    }
    return JSON.parse(rawValue) as Record<string, string>;
  } catch {
    return {} as Record<string, string>;
  }
}

export function getDefaultLanguagePreference() {
  return readStoredPublicSettings()['i18n.default_language']?.trim() || FALLBACK_LANGUAGE;
}

export function hasExplicitLanguagePreference() {
  if (globalThis.document === undefined) {
    return false;
  }
  return globalThis.localStorage.getItem(LANGUAGE_EXPLICIT_STORAGE_KEY) === '1';
}

export function setExplicitLanguagePreference(language: string) {
  if (globalThis.document === undefined) {
    return;
  }
  globalThis.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
  globalThis.localStorage.setItem(LANGUAGE_EXPLICIT_STORAGE_KEY, '1');
}

export function syncDefaultLanguagePreference(defaultLanguage = getDefaultLanguagePreference()) {
  if (globalThis.document === undefined) {
    return defaultLanguage;
  }
  if (!hasExplicitLanguagePreference()) {
    globalThis.localStorage.setItem(LANGUAGE_STORAGE_KEY, defaultLanguage);
  }
  return defaultLanguage;
}

export function clearExplicitLanguagePreference(defaultLanguage = getDefaultLanguagePreference()) {
  if (globalThis.document === undefined) {
    return defaultLanguage;
  }
  globalThis.localStorage.removeItem(LANGUAGE_EXPLICIT_STORAGE_KEY);
  globalThis.localStorage.setItem(LANGUAGE_STORAGE_KEY, defaultLanguage);
  return defaultLanguage;
}
