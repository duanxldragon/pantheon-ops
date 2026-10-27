import { describe, expect, it } from 'vitest';
import {
  PLAN_FEATURES,
  TENANT_PLANS,
  getIncludedPlanFeatures,
  getPlanFeature,
  isFeatureIncluded,
  isPlanAtLeast,
  normalizeTenantPlan,
  type TenantPlan,
} from '../../../../src/core/entitlements/plans';

describe('tenant plan entitlements', () => {
  it('falls back to the free plan for missing or unknown values', () => {
    expect(normalizeTenantPlan(undefined)).toBe('free');
    expect(normalizeTenantPlan(null)).toBe('free');
    expect(normalizeTenantPlan('')).toBe('free');
    expect(normalizeTenantPlan('  Enterprise ')).toBe('enterprise');
    expect(normalizeTenantPlan('platinum')).toBe('free');
  });

  it('orders plans from free to enterprise', () => {
    expect(isPlanAtLeast('free', 'free')).toBe(true);
    expect(isPlanAtLeast('free', 'basic')).toBe(false);
    expect(isPlanAtLeast('enterprise', 'professional')).toBe(true);
    expect(isPlanAtLeast('professional', 'enterprise')).toBe(false);
  });

  it('includes a feature once the plan reaches its minimum tier', () => {
    expect(isFeatureIncluded('free', 'lowcode.dynamic-module')).toBe(false);
    expect(isFeatureIncluded('basic', 'lowcode.dynamic-module')).toBe(true);
    expect(isFeatureIncluded('basic', 'lowcode.generator')).toBe(false);
    expect(isFeatureIncluded('professional', 'lowcode.generator')).toBe(true);
    expect(isFeatureIncluded('professional', 'lowcode.generator.datasource')).toBe(false);
    expect(isFeatureIncluded('enterprise', 'lowcode.generator.datasource')).toBe(true);
  });

  it('never loses a feature when the plan is upgraded', () => {
    for (let planIndex = 0; planIndex < TENANT_PLANS.length - 1; planIndex += 1) {
      const lower = TENANT_PLANS[planIndex] as TenantPlan;
      const higher = TENANT_PLANS[planIndex + 1] as TenantPlan;
      for (const feature of PLAN_FEATURES) {
        if (isFeatureIncluded(lower, feature.key)) {
          expect(isFeatureIncluded(higher, feature.key)).toBe(true);
        }
      }
    }
  });

  it('reports exactly the features included by each plan', () => {
    expect(getIncludedPlanFeatures('free').map((feature) => feature.key)).toEqual([]);
    expect(getIncludedPlanFeatures('basic').map((feature) => feature.key)).toEqual([
      'lowcode.dynamic-module',
    ]);
    expect(getIncludedPlanFeatures('professional').map((feature) => feature.key)).toEqual([
      'lowcode.dynamic-module',
      'lowcode.generator',
    ]);
    expect(getIncludedPlanFeatures('enterprise').map((feature) => feature.key)).toEqual(
      PLAN_FEATURES.map((feature) => feature.key),
    );
  });

  it('keeps the feature registry well-formed and resolvable', () => {
    const seen = new Set<string>();
    for (const feature of PLAN_FEATURES) {
      expect(seen.has(feature.key)).toBe(false);
      seen.add(feature.key);
      expect(TENANT_PLANS).toContain(feature.minPlan);
      expect(feature.minPlan).not.toBe('free');
      expect(feature.titleKey.trim()).not.toBe('');
      expect(feature.descriptionKey.trim()).not.toBe('');
      expect(getPlanFeature(feature.key)).toBe(feature);
    }
  });
});
