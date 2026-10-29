# Pantheon Base v0.17.4 — Release Notes

**Release date**: 2026-10-23
**Tag**: `pantheon-base-v0.17.4`
**Previous release**: `pantheon-base-v0.17.3`

## Summary

v0.17.4 closes the IAM residuals left after v0.17.3. The published v0.17.3 notes registered one
item: the `system_role_data_scope` / `system_role_permission` writes still pre-checked before
inserting. Re-reading the code against that sentence widened the list to four and found one defect
that had never been registered at all — user CSV import resolved and updated rows across tenants.

All four are fixed at the layer that owns the invariant, with regressions that run against real
MySQL. The only observable response change is that the loser of a duplicate-key race now receives
the 409 the pre-check would have produced instead of a 500 carrying the driver's message.

Schema stays at version 29. No route, permission code, menu seed or error-envelope shape is added,
removed or renamed.

## What changed

| Item | Before | Now |
|---|---|---|
| Import lookups | four maps (username, role key, department path, post code) scanned every tenant | each lookup is filtered by the caller's tenant |
| Imported user ownership | row written without `tenant_id`; no membership | `tenant_id` from the request context; active membership written in the same transaction |
| Concurrent create/rename | loser of the key race surfaced MySQL 1062 as a 500 | loser receives the existing 409 (`username_exists` / `role.key.exists`) |
| Permission/menu grants | read-then-insert per row; a concurrent repair failed the second writer | batch insert with `ON CONFLICT DO NOTHING`; the composite key is the arbiter |
| Bootstrap backfill | permission rows stamped `tenant_id = 0`; keys exposed by several menus seeded repeatedly | `tenant_id` taken from the owning role, soft-deleted roles excluded, seeds deduplicated |

## Fixed

### User CSV import no longer crosses tenants

`loadUserImportLookups` built four maps over the whole database. Usernames, role keys, department
paths and post codes are unique **per tenant**, not globally, so the maps did not merely
over-match: a tenant-A import row naming a username that existed in tenant B updated tenant B's
user, and a shared role key resolved to whichever tenant's role id the scan kept last.

Every lookup now runs through `s.scoped(...)`, including `impexp.BuildDeptPathMaps`. The create
path stamps `TenantID` from `tenantOwnerID()` — the request context, never the CSV — and calls
`ensureTenantMembership` inside the same transaction. Without that, imported users landed in the
platform namespace, were invisible to the importing tenant, and failed their first login with
`403 tenant.forbidden`.

`TestImportUsersStaysInsideTheRequestTenant` gives two tenants the same username, department path,
post code and role key, imports as tenant A, and asserts that A's user carries A's ids and an
active membership while B's row is untouched; a second import updates only A's row.

### Duplicate-key losers answer 409

`CreateUser`, `CreateRole` and `UpdateRole` pre-check the natural key, but a pre-check is not
atomic. Two callers carrying the same key both pass it and the UNIQUE index decides. The loser used
to receive `500 ... Error 1062 (23000): Duplicate entry`, which is both the wrong status and a
driver-message leak.

The transaction error is now passed through `database.IsDuplicateKey` and mapped to the conflict
the pre-check already produced: `user.create.error.username_exists` for user creation,
`role.key.exists` for role creation and rename. The mapping runs after the transaction failed, so
any partial role/menu/permission writes are already rolled back.

`TestConcurrentCreateUserMapsLoserToConflict`,
`TestConcurrentCreateRoleMapsLoserToConflict` and `TestConcurrentRenameRoleMapsLoserToConflict`
start four contenders (two for the rename) on one key and assert exactly one winner, a 409 on every
loser, and exactly one surviving row.

### Role grants are idempotent

`grantTenantRolePermissions` and `grantTenantRoleMenus` kept the read as a fast path, then inserted
one row at a time. Two concurrent repair runs that both missed the same key failed the second
writer. Both now batch the missing rows (`CreateInBatches(..., 100)`) with
`clause.OnConflict{DoNothing: true}`, so `(tenant_id, role_id, permission_key)` and
`(tenant_id, role_id, menu_id)` arbitrate. The read is no longer correctness-relevant.

`TestProvisionTenant_PermissionGrantsAreIdempotent` provisions, re-provisions and then deletes one
grant by hand: the second run neither duplicates nor fails, and the third restores the missing row.

### Bootstrap backfill keeps tenant ownership

`backfillRolePermissions` selected the menu grants without joining the owning role, so every
permission row it seeded carried `tenant_id = 0` — including rows for tenant roles. A permission
key exposed by more than one menu was also seeded once per menu, relying on the unique index to
absorb the duplicates.

The query now joins `system_role` for `tenant_id` and explicitly requires
`system_role.deleted_at IS NULL` (GORM does not apply the soft-delete scope to a raw join). Seeds
are deduplicated by `SystemRolePermission` before the batch and inserted with
`OnConflict{DoNothing: true}`. `TestBackfillRolePermissionsStampsTenantAndDeduplicates` covers a
platform role, a tenant role, a soft-deleted role and a repeat run.

## Added

- Real-MySQL regressions in `system/iam/user`, `system/iam/role` and `system/iam/tenant` for the
  four paths above. The duplicate-key branch bodies are confirmed executed by coverprofile, so the
  tests fail on the previous 500 behaviour.

## Changed

- None in the product surface. `VERSION`, `CHANGELOG.md`, `README.md`, `README.en.md` and
  `.harness/STATUS.md` move to v0.17.4.

## Compatibility

- No schema change: the database stays at version 29. No migration is added or edited.
- No route, permission code, menu seed or error-envelope shape is removed or renamed.
- The race-loser status changes from `500` to the existing `409` with the existing message codes.
  Clients that treated the 500 as retryable should treat the 409 as idempotent-write feedback.
- Import behaviour changes for tenants that previously shared usernames, role keys, department
  paths or post codes with another tenant: those rows are now resolved inside the caller's tenant
  and no longer touch the other tenant's data.
- `sharedPaths` is unchanged. `pantheon-ops` gains the fixes by advancing its foundation pin; no
  interface or migration step is required.

## Verification

Measured locally against real MySQL/Redis (`.env.test`, with `PANTHEON_TEST_DSN_REQUIRED` and
`PANTHEON_TEST_REDIS_REQUIRED` set so a missing service fails instead of skipping):

- Full backend suite: 46 packages ok, 0 FAIL, 0 SKIP.
- Targeted new tests: cross-tenant import isolation, backfill ownership plus dedupe, grant
  idempotency — all pass individually.
- Coverprofiles: `user_service.go` duplicate-key branch and both `role_service.go` branches (create
  and rename) recorded with count 1.
- `go build ./...`, `go vet ./...`, `gofmt -l .` clean.
- Governance: docs frontmatter, task packet template and instances, structure, generated modules,
  and failure registry all pass.

Decided by CI, not locally: `Release Gate Summary` on the merge commit, the required-check set on
the pull request, and `Foundation Release Verify` on the tag.

## Known residual

- **Concurrent same-tenant provisioning** can still collide while creating roles (read-then-insert)
  before reaching the now-idempotent grant code. Normal provisioning is serialized per tenant
  creation and repair runs are sequential; this is outside the packet.
- **`replaceRolePermissions` and `replaceUserRoles`** still delete-then-insert. They do not
  pre-check and their inputs are deduplicated, so a conflict requires a concurrent writer on the
  same role or user, which is not a supported workflow.
- **`system_role_data_scope`** keeps its `clause.OnConflict` upsert; the v0.17.3 note that described
  it as read-then-write was already stale and is corrected forward in this release. The published
  v0.17.3 notes are immutable and are not rewritten.
- **The race detector cannot start on the Windows host used for this packet** (TSan shadow-memory
  allocation, error code 87, with and without `-p 1`). Concurrency is covered by the real-MySQL
  contender tests; the race gate runs in CI on Linux.
