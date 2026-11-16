# Status: Business Migrations

- Status: `complete`
- Updated: `2026-08-17`
- Executor: Codex `/root`
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Workdir: `pantheon-ops/backend`

## Completed

- Task packet created from the architecture audit.
- Fresh dirty-worktree preflight recorded before implementation.
- Added `backend/pkg/database/migrations/000012_business_schema.up.sql` and
  matching controlled rollback migration.
- Verified empty-schema creation, generated active-key uniqueness, duplicate
  task-host rejection, rollback, and repeat-up against MySQL 8.0.36.
- Ran `go test -count=1 ./pkg/database/...`,
  `go test -count=1 ./modules/business/...`, and `go vet ./...` successfully.
- Maintainer decision: defer legacy AutoMigrate-table index/column conversion to
  the later production deployment migration generation pass; no local
  conditional ALTER migration is added in this task.

## Decisions

- Migration ownership and sequence must be confirmed against current dirty files.
- No local copy of a base migration runner is allowed.
- Observed dirty-file collisions: `backend/modules/business/bizscope/module.go` and
  `backend/modules/business/cmdb/module.go` are pre-existing modifications in this
  task's potential AutoMigrate-removal surface. Preserve them; do not edit either
  until their ownership and current behavior are revalidated. No existing dirty
  `backend/pkg/database/migrations/**` or `backend/pkg/database/**` files were
  reported by `git status --short`.
- Migration head is `000011`; reserve `000012_business_schema` after confirming
  no existing or dirty `000012` files.
- The base-owned `RegisterBackendModules` already skips module AutoMigrate unless
  `PANTHEON_AUTO_MIGRATE=true`; production remains versioned-migration driven.
- `000012` will create all currently modeled business tables with owner-local
  indexes, task-host uniqueness, and generated active-key columns for effective
  soft-delete uniqueness. The down migration drops only these business tables in
  reverse dependency order; no data conversion or destructive up migration is
  planned.

## Evidence

Created `.harness/evidence/ops-business-versioned-migrations/commands.json`,
`summary.md`, `review.md`, `schema-empty.txt`, `schema-upgrade.txt`, and
`schema-repeat.txt`.

## Next Atomic Action

No further action in this child. Production deployment will generate the
legacy-table upgrade scripts from finalized models under a separate,
maintainer-approved migration task.

## Blockers

Legacy AutoMigrate-table conversion is explicitly deferred by the maintainer to
the production deployment migration pass; it is not a blocker for this child.
