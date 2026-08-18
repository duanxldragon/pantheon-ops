# Task: Business Versioned Migrations

Task ID: `ops-business-versioned-migrations`  
Priority: P0  
Owner: `backend/pkg/database` plus business model owners  
Depends on: none; coordinate table names with capability task  
Blocks: all tasks that add columns or rely on new constraints

## Goal And Invariants

Make every production business schema change reproducible from an empty database
and from the current foundation database. Production startup must not depend on
GORM `AutoMigrate` silently changing schema.

The database must enforce owner-table invariants that do not require business
context: primary keys, required columns, valid status values where compatible
with MySQL 8, effective-record uniqueness, `task_id + host_id` uniqueness, and
indexes used by ownership, visibility, lease, and history queries.

## Why This Is P0

The audit found business services calling `AutoMigrate` while the repository has
only versioned system migrations. A new environment or an upgraded environment
can therefore have different schemas, missing uniqueness, or unsafe NULL/soft
delete behavior. Deployment correctness cannot be established on that basis.

## Preflight

- Read parent handoff and `docs/business-module-review-summary.md`.
- Inspect `backend/pkg/database/migrations/`, `backend/pkg/database/migrate.go`,
  and every `Migrate()` method under `backend/modules/business/`.
- Run `git status --short`; do not alter existing migration or model edits.
- Enumerate actual table names from `TableName()` methods and the current database
  migration metadata. Do not trust a stale design document.

## In Scope

- Add the next collision-free up/down migration sequence after the current
  migration head for existing business tables:
  `biz_business_scope`, `biz_cmdb_host`, CMDB group/label tables,
  `biz_deploy_package`, `biz_deploy_template`, `biz_deploy_template_step`,
  `biz_deploy_task`, `biz_deploy_task_host`, `biz_k8s_cluster`, and
  `biz_k8s_release`.
- Add only constraints and indexes justified by current models and approved child
  task contracts. New ServiceInstance tables/columns belong to task 5 but must
  use this migration numbering after coordination.
- Make production migration execution explicit. If the migration runner is
  inherited/base-owned, add only the Ops registration hook and stop for a base
  change rather than copying the runner.
- Retain `AutoMigrate` only for isolated tests or an explicitly development-only
  path; document and test the production path.
- Add empty-schema, upgrade-schema, repeat-run, and failed-migration recovery
  tests/fixtures using the existing `pantheon-base/pkg/testmysql` conventions.

## Out Of Scope

- changing business behavior, status semantics, API shapes, or UI;
- adding cross-module foreign keys solely to bypass provider APIs;
- destructive data cleanup, table drops, or automatic conversion of ambiguous
  existing status values;
- copying `pantheon-base` migration code into Ops;
- implementing ServiceInstance or deployment leases (reserve migration slots and
  coordinate instead).

## Expected Files

Create or modify only after confirming current paths:

- `backend/pkg/database/migrations/0000XX_business_*.up.sql`
- `backend/pkg/database/migrations/0000XX_business_*.down.sql`
- `backend/pkg/database/migrate.go` only if the Ops-owned registration contract
  requires it
- `backend/pkg/database/migrate_test.go` and a focused business migration test
- `backend/modules/business/*/module.go` only to remove production AutoMigrate
  reliance when the module owns that call
- relevant design/manifest evidence files

Do not touch:

- `../pantheon-base/**`;
- `backend/modules/system/**`, `backend/modules/platform/**`, or frontend files;
- service behavior, capability DTOs, Deploy execution, or K8s client code;
- unrelated dirty-tree migration files.

## Schema Decisions

1. Use InnoDB/MySQL 8 syntax already supported by the project. Verify index name
   length and NULL behavior.
2. Effective-record unique indexes must work with soft deletes. Do not assume
   `(business_key, deleted_at)` gives the desired result on MySQL; use the
   project-supported generated-column or explicit archival strategy and document
   it.
3. Add `UNIQUE(task_id, host_id)` to active task-host records, with a deliberate
   soft-delete policy. Add indexes for `business_scope_id`, lifecycle/status,
   `cluster_id`, and release lookup only where query evidence exists.
4. Do not add FK constraints across bounded contexts unless the owner and delete
   semantics are explicitly documented. An FK cannot replace provider API auth.
5. Every up migration must have a tested down or an explicit forward-only reason;
   forward-only must be a maintainer gate, not an implementer assumption.

## Verification Matrix

| Case                                   | Required result                                                    |
| -------------------------------------- | ------------------------------------------------------------------ |
| Empty database                         | all business tables/constraints are created in order               |
| Current upgrade DB                     | migration succeeds without data loss or unexpected table rewrite   |
| Repeat                                 | runner is a no-op and checksum/head remains valid                  |
| Duplicate IP/package/version/task-host | database rejects the invalid write                                 |
| Soft deleted record                    | effective uniqueness behaves as documented                         |
| Failure/retry                          | failed step leaves a recoverable state and rerun behavior is known |
| Production mode                        | no hidden AutoMigrate changes schema                               |

Commands (run from `pantheon-ops/backend` unless noted):

```text
go test -count=1 ./pkg/database/...
go test -count=1 ./modules/business/...
go test -race ./pkg/database/...
go vet ./...
```

Record database container/version and schema inspection commands in evidence;
never claim migration verification from unit tests alone.

## Evidence And Completion

Write to `.harness/evidence/ops-business-versioned-migrations/`:

- `commands.json`, `summary.md`, `review.md`, `schema-empty.txt`,
  `schema-upgrade.txt`, and `schema-repeat.txt` (or explicit environment gap).

Completion requires migration files, rollback/forward rationale, production
AutoMigrate decision, all four migration scenarios, and review of dirty-tree
collisions.

## Stop Conditions

If a required schema owner is in base, set status `blocked` with the exact base
contract and do not create a local duplicate. Also stop for destructive or
outage-requiring migration, ambiguous data conversion, or migration-number
collision with an existing dirty-tree change.
