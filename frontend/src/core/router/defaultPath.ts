/**
 * Resolve the landing path after login / for the authenticated index route.
 * Shared by the app-level DefaultHomeRedirect and the login page so the
 * two never drift.
 */
export function resolveDefaultAuthedPath(
  hasDashboardPermission: boolean,
  fallbackMenuPath: string | null,
) {
  if (hasDashboardPermission) {
    return '/dashboard';
  }
  if (fallbackMenuPath) {
    return fallbackMenuPath;
  }
  return '/dashboard';
}
