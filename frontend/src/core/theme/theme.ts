import { useCallback, useEffect, useState } from 'react';
import { getPublicSettingList } from '../../modules/system/setting/api';

export type PantheonThemeKey = 'indigo' | 'emerald' | 'violet' | 'slate';

export interface PantheonThemeOption {
  key: PantheonThemeKey;
  labelKey: string;
  descriptionKey: string;
  accent: string;
}

const PANTHEON_THEME_STORAGE_KEY = 'pantheon_theme';
const PANTHEON_THEME_EVENT = 'pantheon-theme-change';
const PANTHEON_DEFAULT_THEME_KEY = 'pantheon_default_theme';

export const pantheonThemeOptions: PantheonThemeOption[] = [
  {
    key: 'indigo',
    labelKey: 'theme.indigo',
    descriptionKey: 'theme.indigo.description',
    accent: '#165dff',
  },
  {
    key: 'emerald',
    labelKey: 'theme.emerald',
    descriptionKey: 'theme.emerald.description',
    accent: '#00a870',
  },
  {
    key: 'violet',
    labelKey: 'theme.violet',
    descriptionKey: 'theme.violet.description',
    accent: '#722ed1',
  },
  {
    key: 'slate',
    labelKey: 'theme.slate',
    descriptionKey: 'theme.slate.description',
    accent: '#334155',
  },
];

const themeKeys = new Set<PantheonThemeKey>(pantheonThemeOptions.map((item) => item.key));

function normalizeTheme(value?: string | null): PantheonThemeKey {
  return value && themeKeys.has(value as PantheonThemeKey) ? (value as PantheonThemeKey) : 'indigo';
}

function readStoredTheme(storageKey: string): PantheonThemeKey | null {
  if (typeof globalThis.document === 'undefined') {
    return null;
  }
  const storedValue = globalThis.localStorage.getItem(storageKey);
  return storedValue && themeKeys.has(storedValue as PantheonThemeKey)
    ? (storedValue as PantheonThemeKey)
    : null;
}

export function getStoredPantheonTheme(): PantheonThemeKey {
  return (
    readStoredTheme(PANTHEON_THEME_STORAGE_KEY) ||
    readStoredTheme(PANTHEON_DEFAULT_THEME_KEY) ||
    'indigo'
  );
}

export function applyPantheonTheme(theme: PantheonThemeKey) {
  if (typeof globalThis.document === 'undefined') {
    return;
  }

  document.documentElement.dataset.pantheonTheme = theme;
  globalThis.localStorage.setItem(PANTHEON_THEME_STORAGE_KEY, theme);
  globalThis.dispatchEvent(new CustomEvent<PantheonThemeKey>(PANTHEON_THEME_EVENT, { detail: theme }));
}

export function applyPantheonDefaultTheme(theme: PantheonThemeKey) {
  if (typeof globalThis.document === 'undefined') {
    return;
  }

  const normalizedTheme = normalizeTheme(theme);
  globalThis.localStorage.setItem(PANTHEON_DEFAULT_THEME_KEY, normalizedTheme);

  if (!globalThis.localStorage.getItem(PANTHEON_THEME_STORAGE_KEY)) {
    document.documentElement.dataset.pantheonTheme = normalizedTheme;
    globalThis.dispatchEvent(
      new CustomEvent<PantheonThemeKey>(PANTHEON_THEME_EVENT, { detail: normalizedTheme }),
    );
  }
}

export async function initializePantheonTheme() {
  const localTheme = readStoredTheme(PANTHEON_THEME_STORAGE_KEY);
  if (localTheme) {
    applyPantheonTheme(localTheme);
    return localTheme;
  }

  const defaultTheme = readStoredTheme(PANTHEON_DEFAULT_THEME_KEY) || 'indigo';
  if (typeof globalThis.document !== 'undefined') {
    document.documentElement.dataset.pantheonTheme = defaultTheme;
  }

  try {
    const response = await getPublicSettingList();
    const serverTheme = normalizeTheme(response.settings['ui.default_theme']);
    applyPantheonDefaultTheme(serverTheme);
    return serverTheme;
  } catch {
    applyPantheonDefaultTheme(defaultTheme);
    return defaultTheme;
  }
}

export function clearPantheonThemePreference() {
  if (typeof globalThis.document === 'undefined') {
    return;
  }
  globalThis.localStorage.removeItem(PANTHEON_THEME_STORAGE_KEY);
  const defaultTheme = readStoredTheme(PANTHEON_DEFAULT_THEME_KEY) || 'indigo';
  document.documentElement.dataset.pantheonTheme = defaultTheme;
  globalThis.dispatchEvent(
    new CustomEvent<PantheonThemeKey>(PANTHEON_THEME_EVENT, { detail: defaultTheme }),
  );
}

export function usePantheonTheme() {
  const [theme, setThemeState] = useState<PantheonThemeKey>(() => getStoredPantheonTheme());

  useEffect(() => {
    const handleThemeChange = (event: Event) => {
      const nextTheme = (event as CustomEvent<PantheonThemeKey>).detail;
      setThemeState(normalizeTheme(nextTheme));
    };

    globalThis.addEventListener(PANTHEON_THEME_EVENT, handleThemeChange);
    return () => {
      globalThis.removeEventListener(PANTHEON_THEME_EVENT, handleThemeChange);
    };
  }, []);

  const setTheme = useCallback((nextTheme: PantheonThemeKey) => {
    const normalizedTheme = normalizeTheme(nextTheme);
    setThemeState(normalizedTheme);
    applyPantheonTheme(normalizedTheme);
  }, []);

  return { theme, setTheme, options: pantheonThemeOptions };
}
