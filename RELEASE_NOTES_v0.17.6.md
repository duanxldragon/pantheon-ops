# Pantheon Base v0.17.6 — Release Notes

**Release date**: 2026-10-27
**Tag**: `pantheon-base-v0.17.6`
**Previous release**: `pantheon-base-v0.17.5`

## Summary

v0.17.6 is the tenant-operations release. It moves plan definitions out of a hardcoded Go matrix
into a versioned table, adds three platform-scoped read surfaces behind `/system/tenant-ops/*`, and
rebuilds the tenant console around them. It also closes the compat-mode tenant-scope leaks that made
single-tenant screens read other tenants' rows, fixes five defects found while driving the running
system, gives every password field an accessible name, and empties the frontend cross-layer boundary
baseline instead of leaving the last violation frozen in it.

This release **does** change the database: migration `000030_tenant_plan_definitions` moves the
schema from version 29 to 30. It also **adds** routes, permission codes and menu seeds. Nothing is
removed or renamed except one Go type name and one frontend type's import path, both documented
below.

Quotas remain display-only. The backend still does not reject a request because a plan threshold was
crossed; that boundary is recorded deliberately in `TENANT_CONTRACT_V1` and its English twin.

## What changed

| Item | Before | Now |
|---|---|---|
| Plan definitions | a hardcoded Go matrix; changing a threshold needed a code change and a release | the `tenant_plans` table (migration `000030`), seeded with the same four built-in codes and the same thresholds the old switch produced |
| Tenant operations | one flat 租户管理 page | a 租户运营 group with an overview aggregate, the tenant list and a cross-tenant member/quota list, plus a plan editor |
| Compat-mode tenant scope | six services discarded the resolved `tenant_id = 0` context, so single-tenant screens saw every tenant's rows | the resolved context is pinned into the query and the write path for dict, dept, post, user, role and the permission workbench |
| Role policy sync | saving a role did not propagate to the Casbin enforcement view | `syncRoleAPIPolicies` runs from the same transaction boundary as the grant rows and is split into three helpers |
| Password fields | Arco's `Input.Password` rendered an unnamed visibility toggle on ten surfaces | a shared `PasswordInput` wrapper supplies the accessible name and hover treatment; `check:password-toggle` blocks a raw `Input.Password` from coming back |
| Cross-layer boundary | `platform/widgets.tsx` imported `system/iam/menu/api`; the finding sat frozen in `config/boundary-baseline.json` | `MenuNode` lives in `frontend/src/api/menu.ts`, the baseline `entries` array is empty, and any new cross-layer import fails the gate loudly |
| Schema | version 29 | version 30 |

## Added

### Stored plan definitions

`backend/modules/system/iam/tenant/tenant_plan_model.go` and `tenant_plan_store.go` define the plan
table and its store. Migration `000030_tenant_plan_definitions` creates `tenant_plans` — a
platform-global table with no `tenant_id` column, exactly like `tenants` itself, because confinement in
this module is applied by explicit `WithTenantScope` calls rather than a global plugin — and seeds
`free`, `basic`, `professional` and `enterprise` with the thresholds the previous hardcoded
`GetDefaultQuota` switch returned. The seed is guarded by plan code and is never an `UPDATE`, so
upgrading an existing database is a no-op for tenants already carrying those codes, and a replay
cannot silently undo an operator's edited threshold. `-1` means unlimited; `0` is rejected by the
service layer, because every consumer has to distinguish "no ceiling" from "ceiling reached". Plan
codes are validated against `^[a-z][a-z0-9_-]{1,31}$` because the code travels inside SQL identifiers
in the matrix views. Removing a plan is not modelled: retirement is `status = 'disabled'`, refused
while tenants still carry the code (`PlanStore.Update` → `ErrPlanInUse`).

### Tenant operations read surfaces

Five routes, five permission codes and one menu group land together so the router, the menu seed and
the copy cannot disagree:

| Route | Permission code |
|---|---|
| `GET /api/v1/system/tenant-ops/overview` | `system:tenant:overview` |
| `GET /api/v1/system/tenant-ops/member` | `system:tenant:member-list` |
| `GET /api/v1/system/tenant-ops/plan/list` | `system:tenant:plan:list` |
| `POST /api/v1/system/tenant-ops/plan` | `system:tenant:plan:create` |
| `PUT /api/v1/system/tenant-ops/plan/:code` | `system:tenant:plan:update` |

They are read/definition surfaces only; the existing `/system/tenant/*` write routes are unchanged.
The new menu group `tenant-ops` (`/system/tenant-ops`, `M`, sort 25) now parents the overview page,
the existing tenant list and the member/quota list.

### Frontend console

`frontend/src/modules/system/iam/tenant/` gains `TenantOverview`, `TenantMemberList`,
`TenantPlanEditor`, `QuotaRatio`, `planOptions.ts`, `quota.ts` and the shared list shell
(`TenantListParts.tsx`, `tenantListCells.tsx`). Quota cells render as ratios against the plan
threshold, not as an enforcement verdict. All five locale bundles were updated in the same change.

### Accessibility

`frontend/src/components/patterns/forms/PasswordInput.tsx` wraps Arco's `Input.Password` with an
`aria-label` on the visibility toggle and a Ghost hover treatment. `npm run check:password-toggle`
(`frontend/scripts/check-password-toggle.mjs`) fails the build if a raw `Input.Password` reappears,
and it is part of `prebuild`.

### Tests

Real-MySQL/Redis regressions for the plan store, the tenant-ops aggregate and member reads, the
compat scope pins in `dept`, `post` and `dict`, the import path-map boundary, role policy
synchronisation, tenant switching inside one tenant, security-event confirmation codes and
password-reset lock counters.

## Fixed

### Compat mode keeps its tenant scope

Under compat the resolved tenant context (`tenant_id = 0`) was discarded by six services. A
single-tenant screen therefore read every tenant's rows, and dict/dept deletes could not target the
caller's record. The scope is now pinned into dict reorder, the dept and post import/export path maps
and tenant stamping on writes. A missing parent is treated as a path boundary rather than an error,
so one historical orphan row cannot block an entire import or export.

### Five defects found by driving the running system

- Saving a role did not propagate to the Casbin enforcement view.
- Switching to another tenant inside the same tenant returned 500.
- Confirming a security event returned the wrong error code.
- Resetting a password left the lock counters intact.
- The permission workbench and department paths leaked across tenant scope.

### Generator and department dialogs

- The generator table picker listed every table regardless of the module being generated, so a
  system-scope module could select `biz_*` tables. It now filters by the active scope's prefix.
- The department dialog could open before the tree finished loading and left the parent field on the
  placeholder the backend rejects; the root is backfilled once the tree arrives and never overrides a
  parent the operator already chose.

### Login page and locale signalling

- `<html lang>` now comes from the active locale — before paint from a static script and again as
  soon as i18n initialises — because assistive tech takes pronunciation from that attribute.
- A best-effort logout no longer raises the CSRF error toast when the POST fails.
- The login page layout balance, control placement and mobile rendering were corrected.

### Lint and duplication findings cleared in code, not by relaxing the gate

`tenant.TenantPlan` was Go stutter and is now `tenant.Plan`. Repeated GORM predicates were named once
per package. The built-in plan codes were spelled in three places that had to agree; one list now
carries seed, assignability and default quota. `syncRoleAPIPolicies` (25 steps in one function,
blocked by Sonar's cognitive-complexity rule) was split into route-set, retire and missing-rules
helpers with identical writes. New-code duplication was brought under the 3% threshold by extracting
the shared tenant list shell rather than by suppressing the rule.

## Changed

- **Governance chain zeroed.** 31 historical strict errors were repaired across `commands.json`
  (missing `linkage`/`knownGaps`) and `review.md` (missing Machine-Readable block, out-of-enum or
  dead-path fields). Nothing was re-judged: every backfilled string restates text already present in
  that same round's files, the appended `review.md` diffs are pure insertions, and no
  `commands[].status` line moved.
- **Root cause registered as FR-016 and FR-017 (status `open`).** `check-evidence`, `check-review`,
  `check-runtime-evidence` and `check-inheritance-contract` have no npm script and appear in no
  workflow, and `check-boundaries` is only ever run with a baseline, so a schema or boundary addition
  retroactively rots older artifacts with no failing signal. Wiring them into CI is a gate-policy
  decision left to the maintainer; the READMEs now document the invocation forms that were being
  misread.
- Frontend dependencies: `i18next` 26.4.6 → 26.5.6, `react-i18next` 17.0.18 → 17.1.3,
  `zustand` 5.0.15 → 5.0.16, `eslint` ^10.12.0 → ^10.13.0,
  `typescript-eslint` ^8.71.2 → ^8.72.0, `vite` ^8.3.4 → ^8.3.5.
- CI actions: `actions/setup-python` and `github/codeql-action/{init,analyze}` moved to newer
  commit SHAs.
- Version material: `VERSION`, `CHANGELOG.md`, `RELEASE_NOTES_v0.17.6.md`, `README.md`,
  `README.en.md` and `.harness/STATUS.md`.

## Compatibility

- **Schema 29 → 30.** `000030_tenant_plan_definitions` must run before the new read surfaces are
  used. `CREATE TABLE IF NOT EXISTS` plus a per-code guarded seed makes it safe on a database that
  already has tenants; the down migration `DROP TABLE IF EXISTS` discards edited thresholds, so
  rehearse it before using it in a rollback.
- **New permission codes are additive.** Existing roles keep working; the new codes only grant the
  new pages. An operator who wants them must re-run the runtime seed or grant them explicitly.
- **Menu tree moved.** 租户管理 (`/system/tenant`) is now a child of the new 租户运营 group instead of
  sitting directly under the system root. Existing permission grants are unaffected, but any
  hand-written menu-tree assumption keyed on the old parent needs review.
- **Go type rename**: `tenant.TenantPlan` → `tenant.Plan`. Internal to the module; no JSON shape,
  table name or route changed.
- **Frontend import path**: `MenuNode` is now exported from `frontend/src/api/menu.ts`.
  `modules/system/iam/menu/api` re-exports it, so existing consumers keep working.
- **Quotas are not enforced.** No request is rejected because a threshold was crossed.
- `sharedPaths` is unchanged, so `pantheon-ops` consumes this release by advancing its foundation
  pin and running its overlay drift checks.

## Verification

Measured locally against real MySQL/Redis (`.env.test`, with `PANTHEON_TEST_DSN_REQUIRED` and
`PANTHEON_TEST_REDIS_REQUIRED` set so a missing service fails instead of skipping):

- Full backend suite `go test ./...`: packages ok, 0 FAIL, 0 SKIP.
- Frontend `tsc -b`, `eslint src tests --max-warnings 0` and `vite build` clean, with the full
  `prebuild` contract chain (menu, i18n hardcode and generated-scope, datetime presentation,
  shell-visual, ui-contract, password-toggle, search-toolbar, important-budget, page-admission,
  smoke web base, smoke-coverage, import-path-case) passing.
- Backend lint at the CI form `golangci-lint run --new-from-rev=main ./...`: no new findings.
- Governance chain: task packet, structure, generated modules, failure registry, boundaries (bare
  `--strict` and the CI `--baseline` form), runtime evidence, evidence and review all green.
- SonarCloud Quality Gate: OK, zero open new-code issues, zero unresolved issues on `main`.
- Dual-mode smoke evidence captured on freshly created databases (compat and multi), including the
  tenant-operations surfaces.

Decided by CI, not locally: the required checks on the pull request, `Release Gate Summary` on the
merge commit, and `Foundation Release Verify` on the tag.

## Known residual

- **FR-016 / FR-017 remain open by decision.** Four governance checkers still have no npm script and
  appear in no workflow, and `check-boundaries` still runs only with a baseline. Registering the gap
  and clearing it are done; wiring it into CI is a gate-policy change reserved for the maintainer.
- **Quota display, not enforcement.** Cross-threshold requests still succeed. Enforcing a plan is a
  commercial-semantics change and is out of scope for this release.
- **Password recovery is an administrator action.** Self-service reset from the login page is
  deliberately out of scope.
- **Local ThreadSanitizer is still unavailable** on the Windows host used for this packet (error 87);
  the race gate is the CI Linux `go test -race ./...` job.
- **Dependabot-triggered runs cannot read the repository secret** `RELEASE_GATE_TOKEN`, so the
  `Automate Solo PR` job fails on those runs. The context is not required for merge; the limitation
  is recorded, not worked around.
