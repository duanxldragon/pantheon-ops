/**
 * Tenant viewpoint state for platform-capable subjects (contract §3.1).
 *
 * A platform subject operates from `tenant_id = 0` by default. The platform
 * may pin a real tenant as the working scope through the platform-only
 * `X-Tenant-Id` header, which the axios request interceptor attaches once
 * this module reports an active viewpoint. The value is kept per-tab in
 * `sessionStorage` (same lifecycle as the other shell session state) so:
 *
 *   - a new tab or a fresh login never inherits a stale viewpoint, and
 *   - switching viewpoint is a full shell reload (mirrors TenantSwitcher),
 *     which guarantees menus/permissions/caches stay coherent.
 *
 * Values:
 *   null  — viewpoint off: no header attached, backend resolves normally
 *   0     — explicit platform-global viewpoint: no header attached (the
 *           backend platform fallback already resolves to tenant 0), the
 *           shell still renders the "platform scope" marker
 *   > 0   — pinned tenant: header `X-Tenant-Id: <id>` attached to every
 *           non-`/auth/` call; the backend accepts the header only from a
 *           platform-capable subject and validates the tenant against the
 *           master table, refusing anything else (missing/suspended/archived
 *           tenant or a header from a non-platform subject => 403). So the
 *           frontend never carries security semantics on its own.
 */

const VIEWPOINT_KEY = 'pantheon_tenant_viewpoint';

/** Platform/global scope marker stored in the key (contract §2.3 reserved 0). */
export const PLATFORM_VIEWPOINT_TENANT_ID = 0;

export function readTenantViewpointId(): number | null {
  const raw = globalThis.sessionStorage.getItem(VIEWPOINT_KEY);
  if (raw === null) {
    return null;
  }
  const parsed = Number(raw);
  // An empty value would coerce to `0` and silently read as the platform
  // viewpoint; we never write it, so it counts as corruption like any other
  // unparsable value. Same treatment for non-integers and negatives.
  if (raw.trim() === '' || !Number.isFinite(parsed) || parsed < 0 || !Number.isInteger(parsed)) {
    globalThis.sessionStorage.removeItem(VIEWPOINT_KEY);
    return null;
  }
  return parsed;
}

export function writeTenantViewpointId(id: number | null): void {
  if (id === null) {
    globalThis.sessionStorage.removeItem(VIEWPOINT_KEY);
    return;
  }
  globalThis.sessionStorage.setItem(VIEWPOINT_KEY, String(id));
}

export const clearTenantViewpoint = (): void => writeTenantViewpointId(null);

export const isTenantViewpointActive = (): boolean => readTenantViewpointId() !== null;

/**
 * The id that should ride on `X-Tenant-Id`. Only a pinned tenant (>0) ever
 * needs the header: platform scope (0) and viewpoint-off (null) both resolve
 * platform-globally server-side.
 */
export function businessTenantIdForHeader(): number | null {
  const id = readTenantViewpointId();
  return id !== null && id > 0 ? id : null;
}
