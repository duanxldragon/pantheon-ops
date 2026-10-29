# Pantheon Base v0.17.5 — Release Notes

**Release date**: 2026-10-24
**Tag**: `pantheon-base-v0.17.5`
**Previous release**: `pantheon-base-v0.17.4`

## Summary

v0.17.5 closes the five IAM residuals registered after v0.17.4. Two were concurrency defects in
the write paths that v0.17.4 had already touched: same-tenant `ProvisionTenant` still created roles
with a read-then-insert, and `replaceRolePermissions` / `replaceUserRoles` still deleted before
re-inserting without holding a parent row. One was a data-repair gap: permission rows a pre-v0.17.4
backfill had stamped `tenant_id = 0` were never rewritten. The remaining two were evidence and
record items: the race gate had no authoritative Linux run, and the published v0.17.3 notes kept a
stale sentence.

All three code defects are fixed at the layer that owns the invariant, with real-MySQL regressions
that fail against the pre-fix code. Schema stays at version 29. No route, permission code, menu seed
or error-envelope shape is added, removed or renamed.

## What changed

| Item | Before | Now |
|---|---|---|
| Same-tenant provisioning | roles, administrator, root department and bindings were read-then-insert; the loser failed the transaction on the unique key | the `tenants` row is locked for the transaction and every write is convergent; platform-default copies run in a second locked transaction on the caller's handle |
| Replace-style grants | `replaceRolePermissions` / `replaceUserRoles` deleted and re-inserted without holding the parent | the `system_role` / `system_user` row is locked first, so the surviving set is the last writer's |
| Historical permission rows | rows a pre-v0.17.4 backfill stamped `tenant_id = 0` stayed in the platform namespace and were invisible to the tenant | the bootstrap backfill deletes stale duplicates, then re-stamps the remaining rows with the owning role's tenant |
| Race gate | local Windows host cannot start TSan; no authoritative run on the change | CI `quality.yml` Backend Tests runs `go test -race ./...` on Linux with MySQL 8 + Redis; the run on this change passed |
| Reduced motion | `.org-chart__dept-card` and the top-menu item transitions ran for everyone, so SonarCloud `css:S9440` stayed open on `main` and parked the release gate at two unresolved issues | both transitions are declared inside `@media (prefers-reduced-motion: no-preference)`; reduced-motion users get no transition, default users keep the 0.18s motion |
| v0.17.3 notes | repository file and release body said `system_role_data_scope` pre-checked before writing | a dated correction block is appended in both places; the published tag and assets stay immutable |

## Fixed

### Same-tenant provisioning serialises on the tenant row

`ProvisionTenant` created the tenant role, administrator, root department, bindings and membership
as a chain of read-then-insert steps. Two concurrent calls for one tenant both missed the role row
and the loser failed the whole transaction on `uk_system_role_tenant_key`
(`system_user.uk_system_user_tenant_username` for the administrator).

The provisioning transaction now starts with `SELECT ... FOR UPDATE` on the `tenants` row. Role and
administrator writes use `ON DUPLICATE KEY UPDATE <col>=<col>` (`clause.OnConflict{DoNothing:true}`)
and read the row back through a locking (current) read, because under REPEATABLE READ a plain read
after a wait-on-conflict can still see the older snapshot. The root department — which has no
unique key — and both bindings become convergent for the same reason: every writer holds the tenant
lock.

The platform-default copies (settings, dict types, dict items) run after commit and are
read-then-insert too. They now execute in a second short transaction that re-takes the same tenant
lock and receives that transaction's handle. `system_dict_item` has no unique key, so without the
lock a lost race would silently duplicate every item.

### Replace-style grants lock the parent row

`replaceRolePermissions` and `replaceUserRoles` deleted every row for the role/user and re-inserted
the requested set. Two replacers that both delete before either inserts leave a merged set; the
index gap can also deadlock.

Both now lock the parent `system_role` / `system_user` row before the DELETE. The lock serialises
only writers of the same role/user; unrelated rows keep writing in parallel.

### Bootstrap backfill repairs historical rows

Before v0.17.4 the backfill selected role-menu rows without joining `system_role`, so tenant roles'
grants were stamped `tenant_id = 0`. Those rows are invisible to every tenant-scoped read.

`backfillRolePermissions` now repairs them before seeding: it deletes stale rows whose
`(role_id, permission_key)` the role already owns under its tenant (updating them would violate
`uk_role_permission_tenant`), then re-stamps the remaining stale rows. Platform grants and grants
of soft-deleted roles are untouched. Both statements are idempotent.

### Race gate evidenced by CI

The Windows host used for this packet cannot start ThreadSanitizer (shadow-memory allocation, error
code 87). The authoritative race signal is the CI `quality.yml` Backend Tests job on Linux with
MySQL 8 and Redis, which runs `go test -race ./...`. Its run on this change passed; the run and job
ids are recorded in the release evidence.

### Reduced-motion preference is respected

The release gate on the merged release material failed on two unresolved SonarCloud `css:S9440`
issues that predate this packet: `.org-chart__dept-card` and the top-menu item transitions were
declared unconditionally, so a user who asks the OS to reduce motion still got the 0.18s
transitions. Both declarations now live inside `@media (prefers-reduced-motion: no-preference)`.
Default users keep the existing motion; reduced-motion users get no transition. A headless Chromium
check against the built CSS measures `0.18s` under `no-preference` and `0s` under `reduce` for both
selectors.

### v0.17.3 notes corrected forward

The published v0.17.3 notes claimed `system_role_data_scope` and `system_role_permission` still
pre-checked. `upsertRoleDataScopePolicyPreservingCustomDepts` has always written through
`clause.OnConflict`, so only the permission half was accurate. A dated correction block is appended
to `RELEASE_NOTES_v0.17.3.md` and to the mutable GitHub v0.17.3 release body. The published tag and
its four assets are immutable and still carry the original sentence.

## Added

- `TestConcurrentProvisioningOfTheSameTenantIsConvergent`: four contenders converge on one role per
  key, one administrator, one binding, one owner membership, one root department and one copy of
  each platform default.
- `TestCopyPlatformDefaultsRunOnTheLockedTenantRow`: the copies write through the caller's locked
  transaction (second connection gets MySQL 3572; rollback leaves no copied rows).
- `TestConcurrentReplaceRolePermissionsSerialisesOnTheRoleRow` and
  `TestConcurrentReplaceUserRolesSerialisesOnTheUserRow`: the parent row stays locked during a
  replace and the queued writer's set is the stored one.
- `TestBackfillRolePermissionsRepairsPlatformNamespaceGrants`: stale rows are re-stamped or
  deduplicated; platform and deleted-role grants stay put; the repair is idempotent.
- Shared lock helpers in `backend/pkg/testmysql/locking.go` so the NOWAIT probe and the serialised
  writer choreography are defined once.

## Changed

- Version material: `VERSION`, `CHANGELOG.md`, `RELEASE_NOTES_v0.17.5.md`, `README.md`,
  `README.en.md` and `.harness/STATUS.md` move to v0.17.5.
- Frontend: the org-chart card and top-menu item transitions in `frontend/src/assets/list-page.css`
  and `frontend/src/core/layout/index.css` are guarded by
  `@media (prefers-reduced-motion: no-preference)`. Default rendering and timing are unchanged.

## Compatibility

- No schema change: the database stays at version 29. No migration is added or edited.
- No route, permission code, menu seed or error-envelope shape is removed or renamed.
- Under contention, provisioning and replace-style writes may wait on a row lock. A lock wait that
  exceeds the transaction timeout surfaces as the existing transaction error; no new response shape
  is introduced.
- `sharedPaths` is unchanged. `pantheon-ops` gains the fixes by advancing its foundation pin; no
  interface or migration step is required.

## Verification

Measured locally against real MySQL/Redis (`.env.test`, with `PANTHEON_TEST_DSN_REQUIRED` and
`PANTHEON_TEST_REDIS_REQUIRED` set so a missing service fails instead of skipping):

- Full backend suite: 46 packages ok, 0 FAIL, 0 SKIP.
- Targeted regressions for provisioning convergence, the copy lock, both replace locks and the
  backfill repair — all pass individually against real MySQL.
- Each new regression fails against the pre-fix shape (unique-key duplicate, nil at the NOWAIT
  probe, or `elements differ` for the backfill).
- `go build ./...`, `go vet ./...`, `gofmt -l .` clean.
- Frontend: Prettier clean on both edited files, `check-ui-contract` 0 findings across 263 files,
  `check-shell-visual-contract` passed, and `vite build` succeeded. A headless Chromium check of the
  built CSS reports `transition-duration: 0.18s` under `prefers-reduced-motion: no-preference` and
  `0s` under `reduce` for both selectors.
- SonarCloud Quality Gate: OK, new-code duplication 0.0% (threshold 3%), zero open new-code issues.
- Governance: docs frontmatter, task packet template and instances, structure, generated modules,
  and failure registry all pass.

Decided by CI, not locally: the required checks on the pull request (including the Linux
`go test -race ./...` job), `Release Gate Summary` on the merge commit, and
`Foundation Release Verify` on the tag.

## Known residual

- **Local TSan**: the race detector still cannot start on the Windows host used for this packet
  (error 87). The race gate is CI Linux; the limitation is recorded, not worked around.
- **Copy serialisation** depends on the `tenants` row lock because `system_dict_item` has no unique
  key. Any future code that copies platform defaults must take the same lock.
- **The published v0.17.0–v0.17.4 tags and assets are immutable.** The v0.17.3 sentence is corrected
  in the repository file and the mutable release body, not in the tag contents.
