# Pantheon Base v0.18.0 — Release Notes

Minor release on the `release/0.18` line. Supersedes `pantheon-base-v0.17.8` as tag
`pantheon-base-v0.18.0`.

This is the first release that carries the **13-wave code simplification programme** merged on
2026-10-27 and 2026-10-28, and with it the tenant lifecycle, membership and console waves that
landed after v0.17.8. The waves are individually reviewed and each keeps its own evidence under
`.harness/archive/2026-10/evidence/`; the programme rollup is in
`.harness/evidence/2026-10-27-codebase-simplification-program/`.

## Summary

- **No schema change.** The migration chain still ends at `000030_tenant_plan_definitions`
  (schema 30).
- **No route, permission code, menu key or i18n key is removed**, and `sharedPaths` stays at 52
  entries. One error code is added: `tenant.member.user_not_in_tenant` (400).
- **Six duplicated backend families collapse to one implementation each.** The changes are
  structural; response shapes do not change.
- Two tenant behaviour changes are deliberate and listed under **Upgrade notes** below.

## What changed

### Simplification (13 waves, S1–S4)

| Wave | Merge | Measured diff |
| --- | --- | --- |
| S1 dead-code removal | #438 | 19 files, +0 / −1,923 |
| S1 API-surface narrowing | #439 | 3 files, +11 / −9 |
| S2 shared handler helpers | #450 | 9 files, +641 / −134 (additive, no call site moved) |
| S2 path-ID parse migration | #451 | 11 files, +66 / −134 |
| S2 pagination and sort | #452 | 29 files, +685 / −423 |
| S2 error mapping | #454 | 31 files, +208 / −290 |
| S2 batch and envelope | #455 | 0 code (review only) |
| S2 import/export pipeline | #456 | 19 files, +578 / −404 |
| S3 tenant scope unification | #461 | 29 files, +278 / −373 |
| S3 role data-scope model | #461 | 7 files, +54 / −75 |
| S3 OIDC envelope | #461 | 1 file, +13 / −0 (comment only) |
| S4 governance decisions | #441 | 0 code (8 files, +411 / −1) |
| S4 re-export alias removal | #447 | `re_export.go` 112 → 52 lines; 6 files, +202 / −45 |

Re-measured structural counts at the release commit:

- inline `ParseUint(c.Param("id"))` blocks: **35 → 2**
- pagination normalizers: **13 → 1** plus per-module adapters
- duplicated clamp lines: **11 → 1**
- inline `Offset((page` calls: **14 → 4**
- `tenantOwnerID` definitions: **11 → 0**
- `errParamInvalid` definitions: **11 → 0**
- sort helpers defined inline: **8 → 0**

Redundant-code candidates: 33 of 45 covered. The twelve not covered (018, 021, 032–035, 039–042,
044, 045) are listed with their reasons in `docs/reviews/CODE_SIMPLIFICATION_PLAN.md` §11.5 and
were not silently dropped.

### Tenant lifecycle and membership

- **Delete requires the archived state (W1).** A tenant that is not archived can no longer be
  deleted; the guard runs before any cascade.
- **Member integrity (W2).** Adding an unknown user is rejected, and the owner minimum counts only
  live owner rows.
- **A membership without a role binding grants no login or switch (W4).** Role bindings are stamped
  from the owning tenant, so a membership outside it can never be a usable context. One predicate
  gates login discovery, the candidate list and the switchable list. `AddMember` narrows to users
  owned by the target tenant and returns the new `tenant.member.user_not_in_tenant` (400).
- **The login picker and the shell `TenantSwitcher` are removed** because they are unreachable by
  construction. The endpoints stay as compatibility surfaces that return at most one item;
  `/auth/current` keeps reporting unfiltered memberships. Existing cross-tenant rows are not
  deleted.
- **Legacy tenant handler, routes and bootstrap service deleted (W8).** They were never mounted, so
  no request path changes.

### Dependencies

- `github.com/coreos/go-oidc/v3` 3.9.0 → 3.21.0.
- The indirect `go-jose` module moves from v3 to v4; v3 no longer receives security fixes.
- `test:smoke:tenant` now runs `tenant-console.spec.ts` (W6).

### Harness, governance and repository layout

- Completed task packets are archived by month under `.harness/archive/<YYYY-MM>/`. Canonical
  `.harness/...` references keep resolving through `resolveArchiveAwarePath`, and a reference that
  exists in neither tree still warns.
- Release notes live in `docs/release-notes/` with bilingual indexes.
- `docs/reviews/CODE_SIMPLIFICATION_PLAN.md` §11 records the execution facts against the original
  plan, and §0–§F3 stay as the planning text on purpose.
- FR-018, FR-019 and FR-020 are registered in `docs/harness/failure-registry.md`;
  `docs/harness/AI_QUALITY_GOVERNANCE.md` §5.2 and §5.3 add the static-analysis re-test rule and
  the multi-wave closeout rule.

## Verification

- `cd backend && go build ./... && go vet ./...` clean.
- `cd backend && go test -count=1 ./...` → 47 packages ok, 0 FAIL, 6 packages without test files.
- Frontend `npm run lint`, `npm run type-check` and `npm run build` clean on the release commit.
- Repository checkers: `check:task-packet` 0 errors, and `check:harness-docs`, `check:structure`,
  `check:harness-encoding`, `check:docs-frontmatter`, `check:harness-inventory`,
  `check:harness-adoption`, `check:harness-visual`, `check:ui-quality-gate`, `check:duplication`,
  `check:failure-registry`, `check-evidence --strict` and `check-review --strict` at zero findings.
- Four consumable assets published and verified by `Foundation Release Verify` on
  `release: published`.

## Known limits

- The LOC window used to summarise the programme (`53,179 → 52,332` backend production Go,
  `68,169 → 67,828` frontend sources) also contains net-additive tenant work, so it understates the
  reduction. It is not quoted as the gain.
- `pantheon-ops` inherits the tenant behaviour changes described in the upgrade notes; its lock and
  rebuild are updated as a separate consumer commit.
- The three maintainer-gated items recorded in `docs/reviews/CODE_SIMPLIFICATION_PLAN.md` §11.6
  (RC-036/037/038 disposition, feature freeze, final acceptance) stay open.
