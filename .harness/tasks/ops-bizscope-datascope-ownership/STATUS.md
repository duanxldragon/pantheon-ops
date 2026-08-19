# Status: BizScope Ownership

- Status: `complete`
- Updated: `2026-08-18`
- Executor: Codex `/root`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Dirty-file collisions: BizScope/CMDB service and module files are pre-existing
  dirty changes from the capability child; preserved and extended in place.

## Completed

- Added `BizScope.dept_id` with aligned DTO/query/migration support and server-side
  `database.WithDataScope` enforcement for list, options, detail, active lookup,
  code resolution, export, and scoped mutations.
- Added CMDB-owned conditional Bind/Unbind commands. Bind rejects existing owners
  and rolls back a partial batch; Unbind requires the expected owner.
- Ordinary Host Update now rejects `businessScopeId`; ownership changes go through
  the CMDB ownership command only.
- Added same-database MySQL advisory-lock coordination. Bind revalidates an active
  BizScope Reader inside the lock; Delete holds the same lock while checking host
  references and soft deleting the scope.
- Added focused scope, ownership-conflict, wrong-owner, and ordinary-update tests.

## Runtime proof (Buffy, 2026-08-17)

The two blockers from the previous handoff are cleared:

- `PANTHEON_TEST_DSN` now points at the local MySQL 8.0.36 instance
  (`127.0.0.1:3306`), so the MySQL-backed tests execute instead of skipping.
- `go test -race` builds and runs with the native MinGW GCC toolchain at
  `D:\msys64\mingw64\bin` (`PATH=.../mingw64/bin:$PATH CC=gcc CGO_ENABLED=1`).

Results:

- `go test -count=1 ./modules/business/bizscope/... ./modules/business/cmdb/...` — passed.
- `go test -race -count=1 ./modules/business/bizscope/... ./modules/business/cmdb/...` — passed,
  including the concurrent single-winner bind test under the race detector.
- `go test -race -count=1 ./modules/business/deploy/...` — passed.
- `go test -count=1 ./...` — all packages pass except the pre-existing
  `TestPurgeModuleAllowsBusinessStaticModuleWithoutTable` (modules/lowcode/
  dynamicmodule), which reproduces identically at the clean base commit
  `ee21350` (verified via a throwaway worktree) and is outside this task's scope.
- `go vet ./...`, `node scripts/check-business-module-boundaries.mjs`,
  `git diff --check` — passed.

## Fixes made during verification

- **`cmdb/capability.go` `bindUnlocked`**: the shared GORM `query` chain
  accumulated the `business_scope_id <> 0` ownership pre-check condition into
  the conditional UPDATE and failure classifier, so bindable hosts matched 0
  rows and surfaced as `cmdbhost.not_found`. Each query is now built on a fresh
  chain; `classifyOwnershipBindFailure` takes `tx` + data scope.
- **`bizscope/bizscope_import_export.go`**: removed the `updated_by` write on
  `biz_business_scope` (column exists in neither the model nor migration
  000012); import of existing scopes no longer fails with "Unknown column".
- **Test harnesses** updated for the new contracts: host tests migrate
  `biz_cmdb_group`; host import test wires a BizScope reader; group fixtures
  seed non-null `conditions` and match the package-local Host columns; bizscope
  scoped-detail and canonical-error tests wire a hermetic CMDB Reader/Command
  fake; `seeds_test` role_menu count corrected to 4 (1 M + 3 C seeds).

## Decisions

- Recommended minimal scope mapping is `BizScope.dept_id` aligned with base
  `DataScopeReq`; confirm against current base before implementation.
- No tenant/member hierarchy is part of this task.
- Scope-owner mapping is `BizScope.dept_id`, aligned with base
  `common.DataScopeReq`; this is a business-owned field, not a base IAM change.

## Evidence

- `.harness/evidence/ops-bizscope-datascope-ownership/commands.json`
- `.harness/evidence/ops-bizscope-datascope-ownership/summary.md`
- `.harness/evidence/ops-bizscope-datascope-ownership/review.md`

## Next Atomic Action

Start dependent child `ops-service-instance-foundation` using the frozen
BizScope `dept_id` scope-owner mapping and CMDB Reader/Command ownership
contract recorded above.

## Blockers

None within scope. The pre-existing `lowcode/dynamicmodule` purge test failure
is tracked as an explicit out-of-scope gap (reproduces at clean base).
