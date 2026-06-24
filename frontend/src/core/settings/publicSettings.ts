import { useSyncExternalStore } from 'react';
import { getPublicSettingList } from '../../modules/system/config/setting/api';
import {
  clearExplicitLanguagePreference as clearExplicitLanguagePreferenceBase,
  hasExplicitLanguagePreference,
  LANGUAGE_EXPLICIT_STORAGE_KEY,
  LANGUAGE_STORAGE_KEY,
  setExplicitLanguagePreference,
  syncDefaultLanguagePreference as syncDefaultLanguagePreferenceBase,
} from './languagePreference';
import { applyPantheonDefaultTheme, type PantheonThemeKey } from '../theme/theme';

export {
  LANGUAGE_STORAGE_KEY,
  LANGUAGE_EXPLICIT_STORAGE_KEY,
  hasExplicitLanguagePreference,
  setExplicitLanguagePreference,
};

export interface PublicSettingsState {
  siteName: string;
  siteLogo: string;
  appMode: 'enterprise' | 'consumer' | 'hybrid';
  orgEnabled: boolean;
  orgRequiredForUser: boolean;
  defaultLanguage: string;
  defaultTheme: string;
  enableTabBar: boolean;
  sessionIdleMinutes: number;
}

const PUBLIC_SETTINGS_STORAGE_KEY = 'pantheon_public_settings';
const PUBLIC_SETTINGS_FALLBACK_SITE_NAME = 'Pantheon Base';

let publicSettingsState: PublicSettingsState = readStoredPublicSettings();
const listeners = new Set<() => void>();
let publicSettingsRefreshTask: Promise<PublicSettingsState> | null = null;

function readStoredPublicSettings(): PublicSettingsState {
  if (globalThis.document === undefined) {
    return buildPublicSettingsState({});
  }
  try {
    const rawValue = globalThis.localStorage.getItem(PUBLIC_SETTINGS_STORAGE_KEY);
    if (!rawValue) {
      return buildPublicSettingsState({});
    }
    const parsed = JSON.parse(rawValue) as Record<string, string>;
    return buildPublicSettingsState(parsed);
  } catch {
    return buildPublicSettingsState({});
  }
}

function buildPublicSettingsState(settings: Record<string, string>): PublicSettingsState {
  const appMode = normalizeAppMode(settings['platform.app_mode']);
  let siteName = settings['site.name']?.trim();
  siteName ??= PUBLIC_SETTINGS_FALLBACK_SITE_NAME;
  if (siteName === '') {
    siteName = PUBLIC_SETTINGS_FALLBACK_SITE_NAME;
  }
  return {
    siteName,
    siteLogo: settings['site.logo']?.trim() ?? '',
    appMode,
    orgEnabled: settings['org.enabled']?.trim() !== 'false',
    orgRequiredForUser: settings['org.required_for_user']?.trim() === 'true',
    defaultLanguage: settings['i18n.default_language']?.trim() ?? 'zh-CN',
    defaultTheme: settings['ui.default_theme']?.trim() ?? 'indigo',
    enableTabBar: settings['ui.enable_tab_bar']?.trim() !== 'false',
    sessionIdleMinutes: Number(settings['login.session_idle_minutes']?.trim() ?? '30'),
  };
}

function normalizeAppMode(value?: string): PublicSettingsState['appMode'] {
  const normalized = value?.trim();
  if (normalized === 'consumer' || normalized === 'hybrid') {
    return normalized;
  }
  return 'enterprise';
}

function persistPublicSettings(settings: Record<string, string>) {
  if (globalThis.document === undefined) {
    return;
  }
  globalThis.localStorage.setItem(PUBLIC_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
}

function notifyPublicSettingsChanged() {
  syncDefaultLanguagePreference();
  applyPantheonDefaultTheme(publicSettingsState.defaultTheme as PantheonThemeKey);
  if (document !== undefined) {
    document.title = publicSettingsState.siteName;
  }
  listeners.forEach((listener) => listener());
}

function syncDefaultLanguagePreference() {
  syncDefaultLanguagePreferenceBase(publicSettingsState.defaultLanguage);
}

export function getPublicSettingsSnapshot() {
  return publicSettingsState;
}

export function applyPublicSettings(settings: Record<string, string>) {
  publicSettingsState = buildPublicSettingsState(settings);
  persistPublicSettings(settings);
  notifyPublicSettingsChanged();
}

export async function refreshPublicSettings() {
  if (!publicSettingsRefreshTask) {
    publicSettingsRefreshTask = getPublicSettingList()
      .then((response) => {
        applyPublicSettings(response.settings);
        return publicSettingsState;
      })
      .finally(() => {
        publicSettingsRefreshTask = null;
      });
  }
  return publicSettingsRefreshTask;
}

export async function initializePublicSettings() {
  notifyPublicSettingsChanged();
  try {
    return await refreshPublicSettings();
  } catch {
    return publicSettingsState;
  }
}

export function usePublicSettings() {
  return useSyncExternalStore(
    (listener) => {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
    getPublicSettingsSnapshot,
    getPublicSettingsSnapshot,
  );
}

export function getBrandInitial(siteName: string) {
  return siteName.trim().charAt(0).toUpperCase() || 'P';
}

export function clearExplicitLanguagePreference() {
  clearExplicitLanguagePreferenceBase(publicSettingsState.defaultLanguage);
}
