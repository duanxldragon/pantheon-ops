import { describe, expect, it } from 'vitest';
import type { PlanRow } from '../../../../../../src/modules/system/iam/tenant/api';
import {
  assignablePlanOptions,
  definedPlanOptions,
  editorPlanOptions,
} from '../../../../../../src/modules/system/iam/tenant/planOptions';

const plan = (code: string, overrides: Partial<PlanRow> = {}): PlanRow => ({
  id: code.length,
  code,
  name: `${code} name`,
  maxUsers: 10,
  maxRoles: 5,
  maxDepts: 5,
  maxApiKeys: 5,
  maxStorageMb: 100,
  status: 'active',
  sort: 10,
  createdAt: '',
  updatedAt: '',
  tenantCount: 0,
  ...overrides,
});

const BUILT_INS = ['free', 'basic', 'professional', 'enterprise'];

describe('tenant plan assignment options', () => {
  it('falls back to the built-in tiers when the definition table is unreadable or empty', () => {
    // Mirrors PlanStore.ValidateAssignment, which accepts the built-in four
    // while the seed has not reached the database.
    for (const plans of [null, []]) {
      expect(assignablePlanOptions(plans).map((option) => option.code)).toEqual(BUILT_INS);
      expect(definedPlanOptions(plans).map((option) => option.code)).toEqual(BUILT_INS);
    }
  });

  it('offers operator-defined codes and keeps the server ordering', () => {
    const options = assignablePlanOptions([
      plan('professional', { sort: 30 }),
      plan('free', { sort: 10 }),
      plan('starter', { sort: 25 }),
    ]);
    expect(options.map((option) => option.code)).toEqual(['professional', 'free', 'starter']);
    expect(options[2]).toEqual({ code: 'starter', name: 'starter name', builtin: false });
  });

  it('hides disabled definitions from the write path only', () => {
    const plans = [plan('free'), plan('retired', { status: 'disabled' })];
    expect(assignablePlanOptions(plans).map((option) => option.code)).toEqual(['free']);
    // The filter is a read path: tenants still carry the retired code.
    expect(definedPlanOptions(plans).map((option) => option.code)).toEqual(['free', 'retired']);
  });

  it('keeps a code the tenant already carries selectable while editing', () => {
    const plans = [plan('free'), plan('retired', { status: 'disabled', name: 'Retired tier' })];
    const options = editorPlanOptions(plans, 'retired');
    expect(options.map((option) => option.code)).toEqual(['free', 'retired']);
    expect(options[1]).toEqual({ code: 'retired', name: 'Retired tier', builtin: false });

    // A code no longer defined at all still shows up, labelled by itself.
    expect(editorPlanOptions(plans, 'gone').map((option) => option.code)).toEqual(['free', 'gone']);
    expect(editorPlanOptions(plans, 'gone')[1]).toEqual({
      code: 'gone',
      name: 'gone',
      builtin: false,
    });

    // Built-ins are not duplicated, and an empty plan changes nothing.
    expect(editorPlanOptions(plans, 'free')).toEqual(assignablePlanOptions(plans));
    expect(editorPlanOptions(plans, '')).toEqual(assignablePlanOptions(plans));
  });

  it('treats a built-in code as built-in even when it came from the table', () => {
    const options = editorPlanOptions([plan('enterprise', { status: 'disabled' })], 'enterprise');
    expect(options[0].builtin).toBe(true);
  });
});
