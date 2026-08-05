import { useCallback, useEffect, useState } from 'react';

export type PantheonColorMode = 'light' | 'dark';

const PANTHEON_COLOR_MODE_STORAGE_KEY = 'pantheon_color_mode';
const PANTHEON_COLOR_MODE_EVENT = 'pantheon-color-mode-change';

const colorModes = new Set<PantheonColorMode>(['light', 'dark']);

export function normalizeColorMode(value?: string | null): PantheonColorMode {
  return value && colorModes.has(value as PantheonColorMode)
    ? (value as PantheonColorMode)
    : 'light';
}

function readStoredColorMode(): PantheonColorMode | null {
  if (globalThis.document === undefined) {
    return null;
  }
  const storedValue = globalThis.localStorage.getItem(PANTHEON_COLOR_MODE_STORAGE_KEY);
  return storedValue && colorModes.has(storedValue as PantheonColorMode)
    ? (storedValue as PantheonColorMode)
    : null;
}

function readSystemColorMode(): PantheonColorMode {
  if (globalThis.matchMedia === undefined) {
    return 'light';
  }
  return globalThis.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

/** Stored preference wins; otherwise fall back to the OS-level preference. */
export function getStoredColorMode(): PantheonColorMode {
  return readStoredColorMode() ?? readSystemColorMode();
}

/**
 * Applies the color mode to the root attributes. Two attributes are set so that
 * both the Pantheon token layer (`data-color-mode`) and Arco's built-in dark
 * theme (`body[arco-theme]`) flip together — otherwise Arco components would
 * stay light while custom surfaces go dark.
 */
export function applyColorMode(mode: PantheonColorMode, persist = true) {
  if (globalThis.document === undefined) {
    return;
  }

  const normalized = normalizeColorMode(mode);
  document.documentElement.dataset.colorMode = normalized;
  if (document.body) {
    document.body.setAttribute('arco-theme', normalized);
  }
  if (persist) {
    globalThis.localStorage.setItem(PANTHEON_COLOR_MODE_STORAGE_KEY, normalized);
  }
  globalThis.dispatchEvent(
    new CustomEvent<PantheonColorMode>(PANTHEON_COLOR_MODE_EVENT, { detail: normalized }),
  );
}

export function initializePantheonColorMode() {
  const mode = getStoredColorMode();
  // Do not persist here: an unstored system preference should stay reactive.
  applyColorMode(mode, readStoredColorMode() !== null);
  return mode;
}

export function usePantheonColorMode() {
  const [mode, setMode] = useState<PantheonColorMode>(() => getStoredColorMode());

  useEffect(() => {
    const handleColorModeChange = (event: Event) => {
      const nextMode = (event as CustomEvent<PantheonColorMode>).detail;
      setMode(normalizeColorMode(nextMode));
    };
    globalThis.addEventListener(PANTHEON_COLOR_MODE_EVENT, handleColorModeChange);
    return () => {
      globalThis.removeEventListener(PANTHEON_COLOR_MODE_EVENT, handleColorModeChange);
    };
  }, []);

  const setColorMode = useCallback((nextMode: PantheonColorMode) => {
    const normalized = normalizeColorMode(nextMode);
    setMode(normalized);
    applyColorMode(normalized);
  }, []);

  const toggleColorMode = useCallback(() => {
    setMode((current) => {
      const next: PantheonColorMode = current === 'dark' ? 'light' : 'dark';
      applyColorMode(next);
      return next;
    });
  }, []);

  return { colorMode: mode, setColorMode, toggleColorMode };
}
