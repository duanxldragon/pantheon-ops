export const OPENED_TABS_STORAGE_KEY = 'pantheon_opened_tabs';
export const SHELL_LAYOUT_MODE_STORAGE_KEY = 'pantheon_shell_layout_mode';
export const SHELL_DENSITY_MODE_STORAGE_KEY = 'pantheon_shell_density_mode';
export const SHELL_LOCKED_STORAGE_KEY = 'pantheon_shell_locked';
export const SHELL_LAST_ACTIVITY_AT_STORAGE_KEY = 'pantheon_shell_last_activity_at';
export const LOGIN_NOTICE_STORAGE_KEY = 'pantheon_login_notice';

export type ShellLayoutMode = 'vertical' | 'horizontal';
export type ShellDensityMode = 'comfortable' | 'compact';

export function readShellLayoutMode(): ShellLayoutMode {
  const rawValue = localStorage.getItem(SHELL_LAYOUT_MODE_STORAGE_KEY);
  return rawValue === 'horizontal' ? 'horizontal' : 'vertical';
}

export function persistShellLayoutMode(mode: ShellLayoutMode) {
  localStorage.setItem(SHELL_LAYOUT_MODE_STORAGE_KEY, mode);
}

export function readShellDensityMode(): ShellDensityMode {
  const rawValue = localStorage.getItem(SHELL_DENSITY_MODE_STORAGE_KEY);
  return rawValue === 'compact' ? 'compact' : 'comfortable';
}

export function persistShellDensityMode(mode: ShellDensityMode) {
  localStorage.setItem(SHELL_DENSITY_MODE_STORAGE_KEY, mode);
}

export function readShellLockedState() {
  return sessionStorage.getItem(SHELL_LOCKED_STORAGE_KEY) === '1';
}

export function persistShellLockedState(locked: boolean) {
  if (locked) {
    sessionStorage.setItem(SHELL_LOCKED_STORAGE_KEY, '1');
    return;
  }
  sessionStorage.removeItem(SHELL_LOCKED_STORAGE_KEY);
}

export function readShellLastActivityAt() {
  const rawValue = sessionStorage.getItem(SHELL_LAST_ACTIVITY_AT_STORAGE_KEY);
  const parsedValue = rawValue ? Number(rawValue) : 0;
  return Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : 0;
}

export function persistShellLastActivityAt(value: number) {
  if (!Number.isFinite(value) || value <= 0) {
    return;
  }
  sessionStorage.setItem(SHELL_LAST_ACTIVITY_AT_STORAGE_KEY, String(Math.floor(value)));
}

export function clearShellLockState() {
  sessionStorage.removeItem(SHELL_LOCKED_STORAGE_KEY);
}

export function clearShellSessionState() {
  localStorage.removeItem(OPENED_TABS_STORAGE_KEY);
  clearShellLockState();
  sessionStorage.removeItem(SHELL_LAST_ACTIVITY_AT_STORAGE_KEY);
}

export function persistLoginNotice(messageKey: string) {
  if (!messageKey) {
    return;
  }
  sessionStorage.setItem(LOGIN_NOTICE_STORAGE_KEY, messageKey);
}
