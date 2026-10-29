# Pantheon Base v0.17.1 — Release Notes

**Release date**: 2026-10-22
**Tag**: `pantheon-base-v0.17.1`
**Previous release**: `pantheon-base-v0.17.0`

## Summary

v0.17.1 is a patch release that closes the three blocking items an independent delivery audit
(2026-10-22) raised against `main 92b94474`, plus two tenant-lifecycle gaps the same audit found
independently. It removes no consumer-facing interface, changes no route, permission code, menu
seed or contract shape, and adds no migration.

The audit's verdict was **FAIL on delivery completeness**, not on product quality: there was no P0
and no functional regression, but `Release Gate` was red on `main` (so no release could be cut),
the remediation lived only in an uncommitted worktree, and the consumer-facing docs contradicted
both the code and the published release. Each of those is now closed on `main`.

## What was broken

| Item | Before | Now |
|---|---|---|
| `Release Gate` on `main` | `failure` — 8 unresolved SonarCloud issues blocked every release | 0 unresolved issues; gate green |
| Remediation delivery | 30 modified + 4 untracked files on a feature branch, no evidence | committed, evidence recorded, task packets closed |
| Consumer docs | README said v0.16.1; `TENANT_API` documented `/api/v1/tenants*` (404 in the running build) | README says v0.17.1; `TENANT_API` documents the mounted `/api/v1/system/tenant*` |
| Delete an archived tenant | impossible — §2.1 (archive is the retirement path) and §2.2 (last owner is protected) deadlocked | archived tenants delete cleanly; live tenants with members are still refused |
| Failed provisioning | tenant row survived, holding its UNIQUE `code` with no owner | the row is removed, so a retry is a plain re-create |
| Dictionary governance bar | three description lines under the metrics, unlike every other system list | label + value only; explanation stays in the governance rail |

## Fixed

### Tenant lifecycle

- **Removing the last active owner is refused** (contract §2.2). A tenant can no longer be left
  without anyone able to administer it. Removing a plain member, or one of several owners, is
  unchanged.
- **Archived is terminal** (contract §2.1). An archived tenant can no longer be written back to
  active, and a rejected transition no longer applies the other fields of the same request.
- **An archived tenant can be deleted even with active members.** The memberships are disabled in
  the same transaction as the code release and the status change. This is what unblocks the
  retirement path: before, the two invariants together meant a provisioned tenant could never be
  deleted through the API. A live tenant with members still answers `tenant.has_active_members`.
- **A failed provisioning no longer strands its tenant row.** Provisioning already ran in one
  transaction; the row it was provisioning did not. It now gets removed, so the `code` is reusable
  and the operator's retry is a plain re-create.

### Error shaping (contract §5)

- Reading or updating a user that belongs to another tenant answers `404 user.not_found` instead
  of `500 user.detail.error` / `500 user.update.error`, so a guessed id cannot distinguish "missing"
  from "belongs to another tenant". The write path resolves the row before mutating anything.
- A reserved role-key conflict (`role.key.reserved`) reports envelope code `400` instead of `500`.

### Migration rollback

- **The chain can be rolled back again.** `000020_tenant_phase1_core_tables.down.sql` dropped
  indexes and columns unconditionally, but `000024.down` had already removed several of them, so
  `migrate down` aborted part-way with `Error 1091: Can't DROP 'tenant_id'; check that column/key
  exists` and left the schema version dirty — an operator following the rollback runbook ended up
  with a half-dropped schema. Every destructive step is now guarded on `information_schema`, so the
  rollback completes and the chain can be re-applied. Forward migration and the runtime schema are
  unchanged; this only affects the rollback path.

### UI consistency

- The dictionary page's governance bar no longer renders a description under each metric. Every
  other system list (`/system/user`, `/system/post`, …) shows only label + value and keeps the
  explanation in the governance rail; the dictionary page was the odd one out, and the three extra
  lines pushed the table down. The three now-unused i18n keys were removed from all five locales.
- The settings page's governance bar lays its four metrics on one row instead of wrapping to a 2×2
  block. The shared metric grid starts at 120px, which fits only three across, so the fourth metric
  dropped to a second row and the bar doubled in height (152px → 82px). The page now widens the
  metrics column and lowers the floor to 96px: one row down to a 960px viewport, 2×2 only at the
  existing ≤920px mobile breakpoint. Verified at 1600 / 1440 / 1280 / 1100 / 960 / 900 px with no
  horizontal overflow at any width.

## Changed

- The eight SonarCloud findings that blocked `Release Gate` are resolved without suppressions:
  two duplicate SQL literals became named constants, the platform-grant repair was split into
  collect/delete helpers, `useLayoutMenu` capability filtering was split into pure helpers, three
  component prop types are `Readonly`, and `isTenantScopedSession` uses optional chaining.
- `docs/api/TENANT_API.md` documents the mounted `/api/v1/system/tenant*` surface (platform-only),
  the envelope codes it actually returns, and points "my tenants" at
  `/api/v1/auth/tenants/switchable`.
- `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` (zh/en) no longer claims the runtime is
  single-tenant; it is the governance template for the delivered compat/multi model.
- `docs/DEPLOYMENT_GUIDE.md` gained a tenant-mode section: `compat` default, `multi` opt-in,
  one-way switch, platform-subject requirement, and the verification steps.

## Compatibility

- **No consumer-facing interface change**: no route, permission code, menu seed, i18n key shape or
  contract shape is removed or renamed.
- **No migration**: the schema stays at version `29`.
- **Behaviour change to be aware of**: `DELETE /api/v1/system/tenant/:id` now succeeds for an
  archived tenant that still has members (it disables them). It still refuses a live tenant with
  members. Callers that relied on "archive is a no-op for deletion" should archive first, then
  delete.
- `pantheon-ops` can advance its foundation lock from v0.16.1 (or v0.17.0) to v0.17.1; the
  `sharedPaths` list is unchanged.

## Verification

- `go build ./...`, `go vet ./...`, `gofmt -l .` clean; `go test ./...` against real MySQL + Redis
  with `PANTHEON_TEST_DSN_REQUIRED=true`: 0 failures, 0 skips.
- New real-MySQL tests: `TestDeleteTenantArchivedWithMembersSucceeds`,
  `TestCreateTenantRollsBackWhenProvisioningFails`, `TestRemoveMemberRefusesLastOwner`,
  `TestUpdateTenantRejectsArchivedReactivation`, `TestMigrationChainRollsBackAndReapplies`.
  The last one migrates a throwaway database up → down → up; it fails against the previous
  `000020` down file with `Error 1091`, which is how the rollback defect was confirmed.
- Frontend: `tsc --noEmit`, `eslint src --max-warnings 0`, `vitest run` (30 files / 214 tests),
  `vite build`, and the 13-step prebuild contract chain all pass.
- Browser acceptance: 17 system routes render with 0 console errors and 0 `/api/` responses ≥400;
  the dictionary page's governance bar now matches the other system lists.
- `Release Gate`, `CI`, `Quality Gates`, `Security Gates`, `Core Smoke`, `Full Smoke` and
  `Tenant Smoke (multi-mode)` green on the release commit; `Foundation Release Verify` green on
  publication.

## Known residual

- The v0.17.0 artifacts (`repo.tar`, and the copy inside the `.tgz`) were built from `728feb2b`,
  before the credential redaction, and still contain two **already-invalidated** dev credentials in
  plaintext. v0.17.1 is built from the sanitized tree; the v0.17.0 assets are immutable and were
  not replaced.
- Plan/entitlement gating remains a frontend concern (UX), with role permissions as the
  authorization boundary; the backend does not restrict features by plan.
- Concurrent tenant provisioning is not serialized; it is idempotent but relies on the unique keys
  rather than a lock.
