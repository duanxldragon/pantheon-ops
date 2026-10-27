import { TENANT_PLANS } from '../../../../core/entitlements/plans';
import type { PlanRow } from './api';

/**
 * Plan choices for the tenant console.
 *
 * `tenants.plan` resolves against the `tenant_plans` table (contract §2.4), so
 * the codes an operator may put on a tenant are data, not the four literal tiers
 * this page used to hardcode. The rules below mirror the server rather than
 * inventing a second policy:
 *
 * - assignable = anything but `disabled` (`TenantPlan.IsEnabled` normalizes and
 *   fails open; the save path re-checks through `PlanStore.ValidateAssignment`);
 * - an unreadable or empty table falls back to the built-in four, which is the
 *   same deferral `ValidateAssignment` applies to a database the seed has not
 *   reached — a failed advisory read must not make the editor unusable.
 *
 * Ordering comes from the server (`sort ASC, id ASC`), so the built-in ladder
 * keeps its free → basic → professional → enterprise shape here for free.
 */
export interface PlanOption {
  code: string;
  name: string;
  /** A built-in tier has an i18n label; an operator-defined code only has its own name. */
  builtin: boolean;
}

function isBuiltinPlanCode(code: string): boolean {
  return (TENANT_PLANS as readonly string[]).includes(code);
}

const BUILTIN_PLAN_OPTIONS: readonly PlanOption[] = TENANT_PLANS.map((code) => ({
  code,
  name: code,
  builtin: true,
}));

function toOption(row: PlanRow): PlanOption {
  return {
    code: row.code,
    name: row.name || row.code,
    builtin: isBuiltinPlanCode(row.code),
  };
}

/** Choices for a write path: only codes the server would accept on a tenant. */
export function assignablePlanOptions(plans: PlanRow[] | null): PlanOption[] {
  if (!plans) return [...BUILTIN_PLAN_OPTIONS];
  const assignable = plans.filter((plan) => plan.status !== 'disabled').map(toOption);
  return assignable.length > 0 ? assignable : [...BUILTIN_PLAN_OPTIONS];
}

/**
 * Choices for a read filter: every defined code, because a retired plan can
 * still have tenants on it and the overview reports those as-is (§2.5).
 */
export function definedPlanOptions(plans: PlanRow[] | null): PlanOption[] {
  if (!plans) return [...BUILTIN_PLAN_OPTIONS];
  const defined = plans.map(toOption);
  return defined.length > 0 ? defined : [...BUILTIN_PLAN_OPTIONS];
}

/**
 * The tenant's own code stays selectable even when it is no longer assignable:
 * opening a retired tenant, or one whose code is no longer defined at all, must
 * not present a form that silently rewrites its plan on save.
 */
export function editorPlanOptions(plans: PlanRow[] | null, currentCode: string): PlanOption[] {
  const options = assignablePlanOptions(plans);
  const current = (currentCode || '').trim();
  if (!current || options.some((option) => option.code === current)) return options;
  const definition = (plans || []).find((plan) => plan.code === current);
  return [
    ...options,
    {
      code: current,
      name: definition?.name || current,
      builtin: isBuiltinPlanCode(current),
    },
  ];
}
