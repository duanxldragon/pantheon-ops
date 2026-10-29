# Pantheon Base v0.17.2 — Release Notes

**Release date**: 2026-10-23
**Tag**: `pantheon-base-v0.17.2`
**Previous release**: `pantheon-base-v0.17.1`

## Summary

v0.17.2 closes the five items v0.17.1 left open. Three of them were recorded as "known residual";
closing them was not a matter of re-labelling. Two were claims that had never been exercised, and
exercising them found real defects in the tenant lifecycle:

- creating two tenants at the same moment **deadlocked** inside the settings copy, and the operator
  saw a 500;
- a **deleted tenant could be revived** by a status update, because the state machine's `default`
  branch answered "allowed" for the tombstone.

Both are fixed, and the tests that found them fail on the previous behaviour. The remaining three
are closed by removing the condition: the Redis test flake by giving each package its own logical
database, the race-detector gap by locating a working toolchain, and the unrecallable v0.17.0
credential by a release-time scan that refuses to publish a bundle carrying one.

No migration, no schema change (the database stays at version 29), no route or permission code
removed or renamed.

## What changed

| Item | Before | Now |
|---|---|---|
| Concurrent tenant creation | `Error 1213 Deadlock found when trying to get lock` → 500, retry by hand | settings/dict copy is a non-locking read + insert; 4 tenants created concurrently, 8 consecutive rounds green |
| Same `code`, two callers | loser got `500 create tenant: Error 1062 (23000): Duplicate entry …` | loser gets the same conflict the pre-check returns (`tenant.code.exists` / `tenant.code.reserved`) |
| Deleted tenant | `GET`/`PUT`/member/provision endpoints still addressed it; `PUT {"status":"active"}` revived it | a tombstone answers `404 tenant.not_found` everywhere; the state machine has no outgoing edge from it |
| Redis-backed test flake | `TestBatchRevokeSessions_IsTenantScoped` failed when packages ran in parallel | each package claims its own logical database; 3 parallel rounds × 4 packages green |
| `go test -race` locally | failed with `gcc.exe failed: exit status 1`, recorded as an environment limit | the Cygwin `gcc` on `PATH` was the cause; `CC=/d/msys64/mingw64/bin/gcc.exe` runs it, clean on 5 packages |
| Release bundle | nothing checked what it carried; v0.17.0 shipped a development password | publish refuses a bundle with a credential in a DSN or on a command line, before the tag exists |
| Contract §2.1 | three states listed, `deleted` unacknowledged | V1.3 clarification: tombstone semantics, not addressable, not an input, `status` is the marker |

## Fixed

### Concurrent provisioning

`copyPlatformSettingsToTenant` and `copyPlatformDictsToTenant` copied platform rows with
`INSERT ... SELECT` against the table they were reading. MySQL takes shared next-key locks on the
rows such a statement reads, and when source and target are the same table those locks collide with
a concurrent copy's insert-intention locks. Two tenants created at the same instant produced:

```
Error 1213 (40001): Deadlock found when trying to get lock; try restarting transaction
```

The copy is now a non-locking `SELECT` followed by an insert. Idempotency is unchanged: existing
tenant rows still win, soft-deleted platform rows are still skipped, and the tenant-side existence
check still ignores `deleted_at` so a re-provision does not collide with a tombstone.

`TestConcurrentProvisioningKeepsTenantResourcesIsolated` creates four tenants concurrently and
asserts ownership and disjointness rather than error codes — every role, membership, root
department, administrator and copied setting belongs to its own tenant and to no other. It failed
with the deadlock on the previous code and passes 8 consecutive rounds now.

### Losing the race for a tenant code

The pre-check for a taken `code` cannot be atomic, so the UNIQUE index is the real arbiter. The
loser used to receive the raw driver error wrapped as a 500, which is both the wrong status for a
caller that merely picked a taken name and a leak of the index name. `pkg/database.IsDuplicateKey`
now recognises the violation and `CreateTenant` re-reads the blocking row, so the answer is the
same `tenant.code.exists` / `tenant.code.reserved` the pre-check would have produced.

### Deleted tenants were addressable

`deleted` is a tombstone written by the delete action, not a lifecycle state. Two things made it
addressable: `tenantStatusTransitionAllowed` fell through to `default: return true` for a previous
status it did not know, and `GetTenantByID` did not filter the tombstone. A single
`PUT {"status":"active"}` therefore brought a deleted tenant back — with the synthetic
`__deleted_tenant_<id>` code written at deletion time, which no operator can type, and with its
memberships still disabled.

`GetTenantByID` now reports a tombstone as not found (contract §5: a missing id and another
tenant's id are indistinguishable) and the state machine answers "no" for it. Deleted rows remain
visible through the explicit `status=deleted` list, which is the operator's audit view.

### Redis test isolation

`go test ./...` runs packages concurrently and every `testredis.Open` flushes its database on
cleanup, so a sibling package could wipe the session keys a test was asserting on. `testredis.OpenDB`
lets a package claim a reserved logical database from a registry of constants, which keeps two
packages from picking the same number. The shared default (1) is unchanged for tests that finish
reading within one call.

## Added

### Release-bundle credential scan

`scripts/foundation-release/check-release-secrets.mjs` runs between "bundle built" and "tag
created". A bundle carrying a credential in a MySQL DSN, a URL, or a `-p<value>` command-line flag
is refused with the file and line, and the secret is masked in the report.

This closes the gap that let v0.17.0 ship a development password inside an archived evidence file.
Secret scanning already ran on every commit and did not catch it: gitleaks has no rule for a bare
password in a shell command, and scanning the git tree says nothing about what the bundle contains.

The rules are three shapes rather than an entropy heuristic, deliberately. The first, broader rule
set (any `password = …` assignment) reported 42 findings on the real v0.17.1 bundle — every one a
test fixture, a Go struct literal or i18n copy. A gate that noisy gets bypassed, so it was narrowed
until the real bundle scans clean and the actual v0.17.0 residue is still caught.

## Changed

- `GET /api/v1/system/tenant/:id`, `PUT`, the member endpoints and `ProvisionTenant` answer
  `404 tenant.not_found` for a deleted tenant instead of `200`. This is the behaviour change to
  plan for: an operator or script that read a deleted tenant by id now gets 404. The list endpoint
  with `status=deleted` is unchanged.
- Contract `TENANT_CONTRACT_V1` moves to V1.3 (clarification only, recorded in §9 with its trigger
  and scope). §2.2's owner rule is untouched; V1.3 only states what its wording already meant —
  the requirement constrains non-archived tenants.
- `go test -race` is available locally now, so race coverage is no longer a CI-only claim.

## Compatibility

- No schema change: the database stays at version 29. No migration is added or edited.
- No route, permission code, menu seed or error envelope shape is removed or renamed.
- `pantheon-ops` can advance its foundation lock from v0.17.0 or v0.17.1 to v0.17.2;
  `sharedPaths` stays at 52 entries.

## Verification

Measured locally, real MySQL 8.0.36 and Redis 7:

- Backend: `go build ./...`, `go vet ./...`, `gofmt -l .` clean; `go test -count=1 -p 1 ./...`
  green with `PANTHEON_TEST_DSN_REQUIRED` and `PANTHEON_TEST_REDIS_REQUIRED` set, so no test could
  skip silently.
- Concurrency: `TestConcurrentProvisioning*` 8 consecutive rounds green.
- Redis isolation: 3 rounds of `./pkg/authtoken ./modules/auth/login ./modules/auth/session
  ./internal/middleware` in parallel, all green.
- Race: `-race` green on `pkg/tenant`, `pkg/testredis`, `internal/middleware`,
  `modules/auth/session` and the tenant lifecycle/concurrency tests.
- Release tooling: `npm run test:foundation-release` 55/55, including 10 new assertions on the
  credential scan; the scan reports the real v0.17.1 bundle clean (766 files) and catches both
  shapes of the v0.17.0 residue.

Decided by CI, not locally: `Release Gate` on the merge commit, `Foundation Release Verify` on the
tag, and the cross-tenant smoke jobs.

## Known residual

- The v0.17.0 asset still contains the development password. Published assets are immutable and the
  credential was rotated, so the only remedy left is the scan that now prevents a recurrence.
- `system_role_data_scope` and `system_role_permission` writes still pre-check rather than relying on
  the index, so a lost race there answers the same 500 the tenant code used to. `IsDuplicateKey`
  makes that fixable per path; only the tenant path is fixed here.
- The `deleted_at` column on `tenants` remains reserved and unwritten. Contract V1.3 records that
  explicitly instead of leaving two possible answers to "is this tenant deleted?".

## Post-publication correction (2026-10-23)

The credential gate shipped in v0.17.2 scanned the staged `bundle/` tree only. The v0.17.0 residue
lived in `repo.tar`, the full-commit snapshot, which the gate did not inspect. The published
v0.17.2 `repo.tar` was scanned afterwards and is clean, so no credential shipped in this release —
but the claim above that the gate "closes the gap" was stronger than the implementation.

A follow-up change makes the gate scan both published shapes, fail closed when `repo.tar` is
missing, and pin the `.harness/archive` case with a regression test. The published notes are kept
as-is; this section is the correction of record.
