import { useEffect } from 'react';
import { useTenantSessionStore } from '../../../store/useTenantSessionStore';

/**
 * Whether the current session is operating in the platform/global scope.
 *
 * This replaces the old `hasPerm('system:tenant:list')` proxy: a tenant
 * administrator legitimately holds that console permission key, so the proxy
 * made every tenant page call `GET /system/tenant/list`, receive
 * `403 tenant.platformOnly` and surface two red toasts (acceptance F5).
 *
 * A permission key answers "may this subject use the tenant console", not
 * "is this subject outside every tenant". The scope fact comes from
 * `/auth/tenants/current` (contract §3.1):
 *   - `tenantId === 0` / `resolvedBy === 'platform-fallback'`: platform scope;
 *   - `resolvedBy === 'header'`: a platform-capable subject pinned to a tenant
 *     viewpoint — still platform-capable, and it needs the control to leave;
 *   - `resolvedBy === 'subject'`: a tenant-scoped session (or a login choice);
 *   - unknown while the request is in flight, so consumers stay hidden.
 */
export type PlatformScope = 'unknown' | 'platform' | 'tenant';

export function usePlatformScope(enabled: boolean): PlatformScope {
  const context = useTenantSessionStore((state) => state.context);
  const loaded = useTenantSessionStore((state) => state.loaded);
  const loadError = useTenantSessionStore((state) => state.loadError);
  const fetchTenantSession = useTenantSessionStore((state) => state.fetchTenantSession);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void fetchTenantSession();
  }, [enabled, fetchTenantSession]);

  // Compat mode has no tenant scope at all: derive it at render time instead of
  // writing state from an effect (which would cascade a render).
  if (!enabled) {
    return 'tenant';
  }
  if (!loaded) {
    return 'unknown';
  }
  if (loadError || !context) {
    // No tenant context means this subject is not platform-capable; the
    // downstream calls would be refused anyway, so stay hidden.
    return 'tenant';
  }
  const platform =
    context.tenantId === 0 ||
    context.resolvedBy === 'platform-fallback' ||
    context.resolvedBy === 'header';
  return platform ? 'platform' : 'tenant';
}
