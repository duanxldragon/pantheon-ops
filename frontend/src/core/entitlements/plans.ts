/**
 * Frontend-owned tenant plan entitlements.
 *
 * The backend stores `tenants.plan` and remains the authority for authorization
 * (role permissions, tenant scope). It does not seed a per-plan feature set:
 * which console surfaces a plan exposes is a presentation decision, so the
 * matrix lives here in one place. Modules tag routes and dashboard widgets with
 * a feature key; the shell hides those surfaces when the session's tenant plan
 * is below the feature's minimum.
 *
 * This gate is UX-only. Hiding a page does not grant access to it, and showing
 * it still requires the page/action permission. Platform scope (tenant 0)
 * bypasses the gate because plan tiers describe tenants, not the operator
 * console itself.
 */
export type TenantPlan = 'free' | 'basic' | 'professional' | 'enterprise';

export const TENANT_PLANS: readonly TenantPlan[] = [
  'free',
  'basic',
  'professional',
  'enterprise',
];

export type PlanFeatureKey =
  | 'lowcode.dynamic-module'
  | 'lowcode.generator'
  | 'lowcode.generator.datasource';

export interface PlanFeatureDefinition {
  key: PlanFeatureKey;
  minPlan: TenantPlan;
  titleKey: string;
  descriptionKey: string;
}

export const PLAN_FEATURES: readonly PlanFeatureDefinition[] = [
  {
    key: 'lowcode.dynamic-module',
    minPlan: 'basic',
    titleKey: 'system.tenant.planFeature.lowcode.dynamicModule',
    descriptionKey: 'system.tenant.planFeature.lowcode.dynamicModuleDesc',
  },
  {
    key: 'lowcode.generator',
    minPlan: 'professional',
    titleKey: 'system.tenant.planFeature.lowcode.generator',
    descriptionKey: 'system.tenant.planFeature.lowcode.generatorDesc',
  },
  {
    key: 'lowcode.generator.datasource',
    minPlan: 'enterprise',
    titleKey: 'system.tenant.planFeature.lowcode.generatorDatasource',
    descriptionKey: 'system.tenant.planFeature.lowcode.generatorDatasourceDesc',
  },
];

const PLAN_RANK: Record<TenantPlan, number> = {
  free: 0,
  basic: 1,
  professional: 2,
  enterprise: 3,
};

/**
 * Unknown or missing plan values fall back to `free`: a tenant whose plan a
 * stale backend or a failed summary lookup could not report must not silently
 * receive paid surfaces.
 */
export function normalizeTenantPlan(value: string | null | undefined): TenantPlan {
  const normalized = (value ?? '').trim().toLowerCase();
  return (TENANT_PLANS as readonly string[]).includes(normalized)
    ? (normalized as TenantPlan)
    : 'free';
}

export function isPlanAtLeast(plan: TenantPlan, minimum: TenantPlan): boolean {
  return PLAN_RANK[plan] >= PLAN_RANK[minimum];
}

export function getPlanFeature(key: PlanFeatureKey): PlanFeatureDefinition | undefined {
  return PLAN_FEATURES.find((feature) => feature.key === key);
}

/**
 * Unknown feature keys stay visible: the union makes a typo a compile error,
 * and failing open here avoids hiding a working page because of stale runtime
 * configuration after an upgrade.
 */
export function isFeatureIncluded(plan: TenantPlan, feature: PlanFeatureKey): boolean {
  const definition = getPlanFeature(feature);
  return definition ? isPlanAtLeast(plan, definition.minPlan) : true;
}

export function getIncludedPlanFeatures(plan: TenantPlan): PlanFeatureDefinition[] {
  return PLAN_FEATURES.filter((feature) => isPlanAtLeast(plan, feature.minPlan));
}
