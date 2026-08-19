# BizScope Ownership Evidence

BizScope now owns a `dept_id` authorization attribute and applies the base
DataScope scope to its business-owned queries. CMDB remains the only Host owner:
BizScope uses CMDB Reader/Command interfaces for host reads and ownership writes.

The Bind command uses a transaction, visibility filter, and
`business_scope_id = 0` conditional update. It rejects any partial target set
and maps stale ownership to a conflict. Unbind requires the expected owner.

Delete and Bind use the same per-scope MySQL advisory lock. A Bind that gets the
lock after a delete must re-read the active BizScope and fails; a delete that gets
the lock after a bind sees the active Host reference and fails.

## Runtime proof (completed 2026-08-17)

Executed against the local MySQL 8.0.36 instance (`127.0.0.1:3306`) with
`PANTHEON_TEST_DSN`, and `go test -race` with the native MinGW toolchain at
`D:\msys64\mingw64\bin`:

- MySQL-backed testing exposed a latent GORM bug in `bindUnlocked`: the shared
  `query` chain accumulated the `business_scope_id <> 0` condition from the
  ownership pre-check into the subsequent conditional UPDATE and the failure
  classifier, so a legitimately bindable host matched 0 rows and surfaced as
  `cmdbhost.not_found` instead of `cmdbhost.ownership_conflict`. Fixed by
  building each query (`found` count, `owned` count, conditional UPDATE,
  classifier) on a fresh chain. The concurrent single-winner test
  `TestOwnershipBindIsSingleWinner` now passes under `-race`.
- Test harnesses were updated for the new contracts: host tests migrate
  `biz_cmdb_group` (host read paths resolve matched groups), host import tests
  wire a BizScope reader, group fixtures seed non-null `conditions` and match
  the package-local Host columns, and BizScope scoped detail/canonical-error
  tests wire a hermetic CMDB Reader/Command fake.
- Pre-existing latent bug fixed: `bizscope_import_export.go` wrote `updated_by`
  on `biz_business_scope`, but neither the model nor migration 000012 defines
  that column; the write is removed so import of existing scopes no longer
  fails with "Unknown column 'updated_by'".
