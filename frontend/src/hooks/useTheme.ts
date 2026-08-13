import { useCallback, useEffect, useState } from 'react';
import {
  applyPantheonTheme,
  getStoredPantheonTheme,
  normalizeTheme as _normalizeTheme,
  pantheonThemeOptions,
  type PantheonThemeKey as _PantheonThemeKey,
} from '../core/theme/theme';

// Re-export types and options from the theme module
export type { PantheonThemeKey } from '../core/theme/theme';
export { pantheonThemeOptions };

export function useTheme() {
  const [theme, setTheme] = useState<_PantheonThemeKey>(() => getStoredPantheonTheme());

  useEffect(() => {
    const handleThemeChange = (event: Event) => {
      const nextTheme = (event as CustomEvent<_PantheonThemeKey>).detail;
      setTheme(_normalizeTheme(nextTheme));
    };

    globalThis.addEventListener('pantheon-theme-change', handleThemeChange);
    return () => {
      globalThis.removeEventListener('pantheon-theme-change', handleThemeChange);
    };
  }, []);

  const updateTheme = useCallback((nextTheme: _PantheonThemeKey) => {
    const normalizedTheme = _normalizeTheme(nextTheme);
    setTheme(normalizedTheme);
    applyPantheonTheme(normalizedTheme);
  }, []);

  return { theme, setTheme: updateTheme, options: pantheonThemeOptions };
}
