import { create } from 'zustand';
import { getCurrentTenant, type CurrentTenantResp } from '../modules/auth/tenant/api';

let tenantSessionPromise: Promise<CurrentTenantResp | null> | null = null;

interface TenantSessionState {
  context: CurrentTenantResp | null;
  loading: boolean;
  loaded: boolean;
  loadError: boolean;
  fetchTenantSession: (options?: { force?: boolean }) => Promise<CurrentTenantResp | null>;
  resetTenantSession: () => void;
}

/**
 * The tenant context `/auth/tenants/current` resolved for this session.
 *
 * One request serves the whole shell: capability filtering, the plan/feature
 * gate and the scope switches all read the same fact, and a second caller must
 * not see a different answer while the first request is in flight.
 */
export const useTenantSessionStore = create<TenantSessionState>((set, get) => ({
  context: null,
  loading: false,
  loaded: false,
  loadError: false,
  fetchTenantSession: async (options) => {
    const force = Boolean(options?.force);
    const current = get();
    if (!force && current.loaded) {
      return current.context;
    }
    if (!force && tenantSessionPromise) {
      return tenantSessionPromise;
    }
    set({ loading: true, loadError: false });
    tenantSessionPromise = getCurrentTenant()
      .then((context) => {
        set({ context, loading: false, loaded: true, loadError: false });
        return context;
      })
      .catch(() => {
        set({ context: null, loading: false, loaded: true, loadError: true });
        return null;
      })
      .finally(() => {
        tenantSessionPromise = null;
      });
    return tenantSessionPromise;
  },
  resetTenantSession: () => {
    tenantSessionPromise = null;
    set({ context: null, loading: false, loaded: false, loadError: false });
  },
}));

/**
 * True when the session is pinned to a concrete tenant (multi mode, tenant id
 * above the platform-global 0). Compat deployments resolve every request to the
 * platform scope and are not tenant sessions.
 */
export function isTenantScopedSession(
  context: CurrentTenantResp | null | undefined,
): boolean {
  return Boolean(context?.mode === 'multi' && (context?.tenantId ?? 0) > 0);
}
