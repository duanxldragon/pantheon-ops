import { useCallback, useMemo } from 'react';
import { usePlatformScope } from '../../modules/auth/tenant/usePlatformScope';
import { useTenantSessionStore } from '../../store/useTenantSessionStore';
import { findRouteByPath } from '../router/modules';
import { usePublicSettings } from '../setting/publicSettings';
import {
  isFeatureIncluded,
  normalizeTenantPlan,
  type PlanFeatureKey,
  type TenantPlan,
} from './plans';

export interface PlanEntitlements {
  /** True when the gate applies, i.e. the session is a tenant-scoped subject. */
  enforced: boolean;
  /** Effective plan for a tenant session; null for platform scope and compat. */
  plan: TenantPlan | null;
  hasFeature: (feature?: PlanFeatureKey) => boolean;
  /** Resolves a menu/route path through the module route registry. */
  isPathEntitled: (path: string) => boolean;
}

/**
 * Resolves the plan gate for the current session from the tenant context the
 * shell already loads. Compat deployments and platform-capable subjects are
 * outside the gate: there is no tenant plan to apply (compat) or the operator
 * console is not a tenant surface (platform).
 *
 * While the tenant context is still in flight the hook fails open, so a
 * platform operator never briefly sees an upgrade page on a direct URL load. A
 * tenant subject's entitlement is re-evaluated as soon as `/auth/tenants/current`
 * resolves.
 */
export function usePlanEntitlements(): PlanEntitlements {
  const publicSettings = usePublicSettings();
  const multiTenant = publicSettings.tenantMode === 'multi';
  const scope = usePlatformScope(multiTenant);
  const context = useTenantSessionStore((state) => state.context);

  const plan = useMemo<TenantPlan | null>(() => {
    if (!multiTenant || scope !== 'tenant') {
      return null;
    }
    return normalizeTenantPlan(context?.tenant?.plan);
  }, [context?.tenant?.plan, multiTenant, scope]);

  const hasFeature = useCallback(
    (feature?: PlanFeatureKey) => {
      if (!feature || plan === null) {
        return true;
      }
      return isFeatureIncluded(plan, feature);
    },
    [plan],
  );

  const isPathEntitled = useCallback(
    (path: string) => {
      if (!path) {
        return true;
      }
      return hasFeature(findRouteByPath(path)?.feature);
    },
    [hasFeature],
  );

  return {
    enforced: plan !== null,
    plan,
    hasFeature,
    isPathEntitled,
  };
}
