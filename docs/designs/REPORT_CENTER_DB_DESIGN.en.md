---
title: Report Center Data Model Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Report Center Data Model Design

> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.
> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。

Main design: [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md) (Chinese [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md))
API contract: [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md)
Test plan: [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md)

---

This document defines the report center's table shapes, indexes, migration numbering, down shape and
retention/purge strategy. Baseline rules come from [DATABASE.en.md](./DATABASE.en.md): InnoDB,
`utf8mb4_general_ci`, `bigint unsigned NOT NULL AUTO_INCREMENT` primary keys, the GORM soft-delete column
`deleted_at datetime(3) NULL`, and **no physical foreign keys** (§4).

## 1. Table Inventory and Tenant Classification

| Table | Purpose | Tenant class (matrix §1) | `tenant_id` column |
| :--- | :--- | :--- | :--- |
| `report_dataset` | registered fetch semantics | `platform-global` | none (§2.1) |
| `report_template` | report definition | `tenant-owned` | yes |
| `report_template_version` | immutable version snapshot | `tenant-owned` (follows the template) | yes |
| `report_run` | one render execution record | `derived` (follows the source) | yes |
| `report_artifact` | artifact metadata | `derived` (follows the source) | yes |

Registration duty: these classifications must be written into
[TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md) §2 and §3 (unique-key register) by
the implementation PR. This design package does not modify the matrix (main design §16 OD-6).

## 2. Target Table Shapes (what the migration creates)

The DDL below is the design target; implementation puts it in `backend/pkg/database/migrations/`
(numbering in §4). Every table carries `created_at / updated_at / deleted_at`
(`DATETIME(3) DEFAULT NULL`) plus an `idx_<table>_deleted_at` index, matching
`backend/pkg/database/migrations/000001_init_schema.up.sql` (e.g. `system_user` at lines 117/120).
**No physical foreign keys** ([DATABASE.en.md](./DATABASE.en.md) §4).

### 2.1 `report_dataset`

```sql
CREATE TABLE IF NOT EXISTS `report_dataset` (
  `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code`              VARCHAR(64)  NOT NULL,
  `name`              VARCHAR(128) NOT NULL DEFAULT '',
  `description`       VARCHAR(512) NOT NULL DEFAULT '',
  `source_domain`     VARCHAR(32)  NOT NULL DEFAULT '',
  `provider_key`      VARCHAR(64)  NOT NULL DEFAULT '',
  `param_schema_json` TEXT         NULL,
  `scope_binding`     VARCHAR(16)  NOT NULL DEFAULT 'none',
  `row_limit`         INT          NOT NULL DEFAULT 10000,
  `timeout_ms`        INT          NOT NULL DEFAULT 5000,
  `cacheable`         TINYINT(1)   NOT NULL DEFAULT 0,
  `freshness_seconds` INT          NOT NULL DEFAULT 0,
  `status`            VARCHAR(16)  NOT NULL DEFAULT 'active',
  `created_at`        DATETIME(3)  DEFAULT NULL,
  `updated_at`        DATETIME(3)  DEFAULT NULL,
  `deleted_at`        DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_dataset_code` (`code`),
  INDEX `idx_report_dataset_source_domain` (`source_domain`),
  INDEX `idx_report_dataset_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Design notes:

1. **No `tenant_id`**: same reasoning as `tenant_plans` — this module confines tenants through explicit
   `WithTenantScope` calls in services, not a global plugin, so adding the column to a platform registry
   would empty the platform console's own reads (original rationale in
   `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:11-17`).
2. `scope_binding` allows only `none` / `dept` / `self`, one-to-one with the predicates in
   `backend/pkg/database/scope.go` (main design §9.2).
3. `row_limit` defaults to 10000 = `impexp.MaxExportRows()` (`backend/pkg/impexp/export_limits.go:9`);
   the service layer rejects anything larger (main design §8.4).
4. `param_schema_json` is a whitelist, not an executable fragment; a parse failure rejects registration.

### 2.2 `report_template`

```sql
CREATE TABLE IF NOT EXISTS `report_template` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `code`               VARCHAR(64)  NOT NULL,
  `report_type`        VARCHAR(24)  NOT NULL DEFAULT 'list',
  `name`               VARCHAR(128) NOT NULL DEFAULT '',
  `description`        VARCHAR(512) NOT NULL DEFAULT '',
  `category`           VARCHAR(32)  NOT NULL DEFAULT '',
  `dataset_code`       VARCHAR(64)  NOT NULL DEFAULT '',
  `visibility`         VARCHAR(16)  NOT NULL DEFAULT 'platform',
  `owner_user_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `supported_formats`  VARCHAR(64)  NOT NULL DEFAULT 'csv',
  `default_param_json` TEXT         NULL,
  `permission_key`     VARCHAR(64)  NOT NULL DEFAULT '',
  `sensitive`          TINYINT(1)   NOT NULL DEFAULT 0,
  `status`             VARCHAR(16)  NOT NULL DEFAULT 'draft',
  `current_version_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`         DATETIME(3)  DEFAULT NULL,
  `updated_at`         DATETIME(3)  DEFAULT NULL,
  `deleted_at`         DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_template_tenant_code` (`tenant_id`, `code`),
  INDEX `idx_report_template_tenant_visibility_status` (`tenant_id`, `visibility`, `status`),
  INDEX `idx_report_template_dataset_code` (`dataset_code`),
  INDEX `idx_report_template_owner_user` (`owner_user_id`),
  INDEX `idx_report_template_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Design notes:

1. The unique key is `(tenant_id, code)`, the same shape as `system_role_data_scope`
   (`uk_role_data_scope_tenant_role`, `backend/internal/middleware/data_scope_middleware.go:47-48`) and
   `system_user` (`uk_system_user_tenant_username`, migration 000024, registered in
   [TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md) §3).
2. `visibility` and `permission_key` are orthogonal: the first controls list visibility, the second
   controls executable actions (main design §6.1).
3. `created_by` exists so a `scope_binding = self` report can reuse the owner-column detection in
   `backend/pkg/database/scope.go:70-96`.
4. Templates flagged `sensitive = 1` route their export through `SecureActionMiddleware`
   (main design §9.4).

### 2.3 `report_template_version`

```sql
CREATE TABLE IF NOT EXISTS `report_template_version` (
  `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `version_no`       INT          NOT NULL DEFAULT 1,
  `spec_json`        TEXT         NULL,
  `dataset_snapshot` TEXT         NULL,
  `status`           VARCHAR(16)  NOT NULL DEFAULT 'draft',
  `published_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `published_at`     DATETIME(3)  DEFAULT NULL,
  `created_at`       DATETIME(3)  DEFAULT NULL,
  `updated_at`       DATETIME(3)  DEFAULT NULL,
  `deleted_at`       DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_template_version` (`template_id`, `version_no`),
  INDEX `idx_report_version_tenant_status` (`tenant_id`, `status`),
  INDEX `idx_report_version_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Design notes:

1. The unique key deliberately **excludes `tenant_id`**: `template_id` is already tenant-unique, and adding
   the column would only open an inconsistency window for cross-tenant writes.
2. The service layer refuses UPDATE on `published` rows (only INSERT of a new version, pointer advance,
   and soft delete per §5).
3. Version rows grow per template only; phase one adds no version retention. Soft-deleting a template
   soft-deletes its versions in the same transaction.

### 2.4 `report_run`

```sql
CREATE TABLE IF NOT EXISTS `report_run` (
  `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`             BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_version_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `dataset_code`          VARCHAR(64)  NOT NULL DEFAULT '',
  `trigger_source`        VARCHAR(16)  NOT NULL DEFAULT 'manual',
  `schedule_execution_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `requester_user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `format`                VARCHAR(16)  NOT NULL DEFAULT 'csv',
  `param_digest`          VARCHAR(64)  NOT NULL DEFAULT '',
  `scope_fingerprint`     VARCHAR(64)  NOT NULL DEFAULT '',
  `status`                VARCHAR(16)  NOT NULL DEFAULT 'pending',
  `row_count`             INT          NOT NULL DEFAULT 0,
  `truncated`             TINYINT(1)   NOT NULL DEFAULT 0,
  `duration_ms`           INT          NOT NULL DEFAULT 0,
  `error_key`             VARCHAR(64)  NOT NULL DEFAULT '',
  `started_at`            DATETIME(3)  DEFAULT NULL,
  `finished_at`           DATETIME(3)  DEFAULT NULL,
  `created_at`            DATETIME(3)  DEFAULT NULL,
  `updated_at`            DATETIME(3)  DEFAULT NULL,
  `deleted_at`            DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_report_run_tenant_time` (`tenant_id`, `created_at`),
  INDEX `idx_report_run_template_time` (`template_id`, `created_at`),
  INDEX `idx_report_run_requester_time` (`requester_user_id`, `created_at`),
  INDEX `idx_report_run_status` (`status`),
  INDEX `idx_report_run_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Design notes:

1. No unique key: a run is an event record and repeated renders must leave repeated rows (audit
   requirement), so only query indexes exist.
2. `param_digest` / `scope_fingerprint` store hashes (main design §13): they answer "who ran what with
   which conditions" without putting possibly sensitive large parameters into a row — the same discipline
   as the masking rule for `system_log_oper.oper_param` ([DATABASE.en.md](./DATABASE.en.md) §3.2).
3. `scope_fingerprint` must be composed of `mode` + normalized `deptIDs` + `userID`
   (main design §10 constraint 1).
4. `schedule_execution_id` references `schedule_execution.id` (scheduler-side table) logically; no
   physical foreign key.

### 2.5 `report_artifact`

```sql
CREATE TABLE IF NOT EXISTS `report_artifact` (
  `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `run_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `storage_driver`   VARCHAR(16)  NOT NULL DEFAULT 'local',
  `object_key`       VARCHAR(255) NOT NULL DEFAULT '',
  `filename`         VARCHAR(128) NOT NULL DEFAULT '',
  `content_type`     VARCHAR(64)  NOT NULL DEFAULT '',
  `size_bytes`       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `row_count`        INT          NOT NULL DEFAULT 0,
  `scope_fingerprint` VARCHAR(64) NOT NULL DEFAULT '',
  `status`           VARCHAR(16)  NOT NULL DEFAULT 'ready',
  `expires_at`       DATETIME(3)  DEFAULT NULL,
  `created_at`       DATETIME(3)  DEFAULT NULL,
  `updated_at`       DATETIME(3)  DEFAULT NULL,
  `deleted_at`       DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_artifact_object_key` (`object_key`),
  INDEX `idx_report_artifact_tenant_expires` (`tenant_id`, `expires_at`),
  INDEX `idx_report_artifact_run` (`run_id`),
  INDEX `idx_report_artifact_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

Design notes:

1. `storage_driver` / `object_key` semantics match `backend/pkg/upload/service.go`
   (`upload.storage_driver` = `local` | `s3`; key = `scope/date/uuid.ext`).
2. The first (scope) segment of `object_key` must carry the tenant dimension so lookups cannot cross
   tenants ([TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md) §2, export-file row).
3. `scope_fingerprint` gives option A of main design §16 OD-5 (re-check row scope at download time) a
   place to land.
4. `expires_at` is computed from the §6 retention at write time, so the purge task scans only this column
   instead of the whole table every run.

## 3. Indexes and Query Matching

| Query (API contract §4) | Index hit |
| :--- | :--- |
| template list filtered by tenant + visibility + status | `idx_report_template_tenant_visibility_status` |
| template lookup by code | `uk_report_template_tenant_code` |
| version list of a template | `uk_report_template_version` (template_id prefix) |
| run history pagination | `idx_report_run_tenant_time` / `idx_report_run_template_time` |
| "my exports" | `idx_report_run_requester_time` |
| artifact expiry sweep | `idx_report_artifact_tenant_expires` |

Deliberately not indexed: a standalone `status` column ([DATABASE.en.md](./DATABASE.en.md) §4: low
cardinality), appearing only inside composite indexes.

## 4. Migration Numbering and Files

The highest number currently in `backend/pkg/database/migrations/` is **000031**
(`000031_drop_permission_role_data_scope_policy.up.sql` / `.down.sql`). The report center reserves:

| Number | File | Content |
| :--- | :--- | :--- |
| 000032 | `000032_report_center_core_tables.up.sql` | create the 5 tables of §2 (all `CREATE TABLE IF NOT EXISTS`) |
| 000032 | `000032_report_center_core_tables.down.sql` | see §7 |
| 000033 | `000033_report_center_setting_seed.up.sql` | idempotent seed of the §6 `system_setting` keys |
| 000033 | `000033_report_center_setting_seed.down.sql` | delete those key rows |

Numbering rule: **a number must not be reused inside one PR**; if it collides with another concurrent
design package, the earlier merged reservation wins and the later package renumbers, recording the conflict
resolution in the PR description (this package and the scheduler package are allocated
000032/000033 and 000034/000035 respectively).

Seeding must follow the guarded insert style of 000030 (never an UPDATE overwrite); the original rationale is
in `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:22-28`:

```sql
INSERT INTO `system_setting` (`setting_key`, `setting_value`, ...)
SELECT 'report.artifact_retention_days', '7', ...
WHERE NOT EXISTS (SELECT 1 FROM `system_setting` WHERE `setting_key` = 'report.artifact_retention_days');
```

Other migration facts that must be respected:

- `currentRuntimeSchemaVersion` and the compatibility-marker mechanism baseline a database that "already
  looks current" (`backend/pkg/database/migrate.go:39`, `:157-202`). New tables do **not** need a new
  marker, but the implementation must never lean on `PANTHEON_AUTO_MIGRATE` GORM AutoMigrate as a second
  schema authority (the lesson is written out in
  `backend/internal/middleware/data_scope_middleware.go:73-85`).
- Migrations run through the golang-migrate MySQL driver, which serializes with
  `SELECT GET_LOCK(?, 10)`; in a rolling multi-replica update only the replica that wins the lock runs them
  (facts and handling in
  [MIGRATION_000031_PRODUCTION_DROP.md](../runbooks/MIGRATION_000031_PRODUCTION_DROP.md) §1.2 and
  `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md:1070`).
- Destructive production changes follow the preflight-first process; the precedent is
  `backend/pkg/database/migrations/preflight/000031_preflight.sql`.

## 5. Soft Delete and Unique-Key Release (Existing Repository Rule)

MySQL unique indexes do **not** include `deleted_at`: a soft-deleted row still occupies the key, so a delete
must rename the identifier first. Four implementations already do exactly this, and the report center reuses
the pattern instead of inventing one:

| Existing implementation | Location | Prefix constant |
| :--- | :--- | :--- |
| post `post_code` | `backend/modules/system/org/post/post_service.go:40`, `:806` | `__deleted_post_` |
| dict type / item | `backend/modules/system/config/dict/dict_service.go:1483`, `:1501` | `__deleted_dict_type_` etc. |
| tenant code | `backend/modules/system/iam/tenant/tenant_service.go:847` | `__deleted_tenant_` |
| username | [DATABASE.en.md](./DATABASE.en.md) §3.3.2 | archived to an internal reserved value |

Report-side rules:

1. Deleting a `report_template` renames `code` to `__deleted_report_<id>` in the same transaction
   (append `_<unixnano>` on conflict, 5 attempts, and a conflict error key on final failure — exactly the
   loop and error semantics of `allocateDeletedPostCode`).
2. Same for `report_dataset.code` if deletion of a registered dataset is allowed.
3. `report_artifact.object_key` is system generated and never reused, so no rename is needed; delete means
   soft delete plus background object cleanup.
4. `report_run` needs no rename (it has no unique key).
5. Uniform index shape: `uk_*` covers only the business identifier columns (plus `tenant_id`) and
   **never** includes `deleted_at`.

## 6. Retention and Purge

| Setting key (`system_setting`) | Default | Effect |
| :--- | :--- | :--- |
| `report.artifact_retention_days` | 7 | `report_artifact.expires_at = created_at + N` |
| `report.run_retention_days` | 90 | retention of `report_run` rows |
| `report.sync_row_cap` | 10000 | display only; the real ceiling stays `impexp.MaxExportRows()` (no second authority) |
| `report.default_timeout_ms` | 5000 | fallback when a dataset declares no timeout |

Purge execution (must follow the existing shape; no full-table DELETE inside a request path):

1. The service implements `RegisterMaintenanceTasks(reg *maintenance.Registry)` and registers
   `report.artifact_retention` and `report.run_retention` with `Interval = maintenance.DefaultInterval`
   (15 minutes, `backend/pkg/maintenance/defaults.go`). Precedent:
   `backend/modules/system/audit/audit_service.go:317`.
2. The registry already owns throttling and **overlap protection** (`OutcomeOverlap`), so a task only
   describes "how to purge", never "how often".
3. The background runner is controlled by `PANTHEON_MAINTENANCE_ENABLED` /
   `PANTHEON_MAINTENANCE_INTERVAL_SECONDS` (`runServer` in `backend/cmd/server/main.go`).
   **The operational fact must appear in acceptance**: when the switch is off, artifacts never expire
   automatically and are only judged expired at download time.
4. Retention days are read from `system_setting` with a fallback, the same way
   `getRetentionDaysFromSetting` does it (`backend/modules/system/audit/audit_service.go:325-341`).
5. Purge order: best-effort object removal (failure only warns, never blocks), then soft delete of
   `report_artifact`, then time-based delete of `report_run`.
6. `report_run` deletion uses a `LIMIT`-bounded loop (suggested 1000 rows per batch) to avoid long
   transactions; this is a new constraint of this design because the run table is the only report table that
   grows fast.

## 7. Down Shape

| Migration | Down behaviour | Data promise |
| :--- | :--- | :--- |
| 000032 | `DROP TABLE IF EXISTS` for the five tables | **destroys data**: rollback restores structure only; run/artifact rows and objects are unrecoverable |
| 000033 | `DELETE FROM system_setting WHERE setting_key IN (the 4 keys)` | reclaims seeded rows only; if an operator changed a value the rollback loses it, which the runbook must state |

Two hard requirements for the down file:

1. Down must recreate the shape that actually existed **before** that migration, adding no index or column.
   The counter-example and the assertion style are in `backend/pkg/database/migrate_down_shape_test.go`
   (byte comparison of `SHOW CREATE TABLE` between "migrated straight up to N" and "up to N+1 then down to
   N"). The report-side test: `000032/000033` up->down->up yields the same shape and down leaves no index
   behind.
2. The down file header must state the data promise, following
   `backend/pkg/database/migrations/000031_drop_permission_role_data_scope_policy.up.sql` / `.down.sql`
   (up declares `DATA PROMISE: this up migration destroys the table and any rows in it`; down declares the
   shape is rebuilt while `DATA IS NOT RESTORED`).

## 8. Pre-Merge Gate Checklist (the implementation PR must pass)

| Check | Command |
| :--- | :--- |
| migration up/down shape | `cd backend && go test ./pkg/database/` (fixture `backend/pkg/testmysql`; assertion style per `backend/pkg/database/migrate_down_shape_test.go`, `migrate_down_test.go`, `migrate_preflight_test.go`) |
| migration preflight (if destructive) | add a file mirroring `backend/pkg/database/migrations/preflight/000031_preflight.sql` |
| backend unit tests | `cd backend && go test ./...` (CI jobs `unit-tests` / `backend-tests`) |
| Go lint | `golangci-lint` (CI job `go-lint`) |
| permission key shape | `npm run check:harness-permission-contract` |
| DTO / response envelope | `npm run check:backend-dto-contract`, `npm run check:backend-response-contract` |
| documentation | `npm run check:docs-frontmatter`, `npm run check:harness-docs` |

## 9. Open decisions

| ID | Decision | Options | Owner |
| :--- | :--- | :--- | :--- |
| DB-OD-1 | whether new tables state `COLLATE=utf8mb4_general_ci` (required by [DATABASE.en.md](./DATABASE.en.md) §1) or follow the recent migrations' bare `DEFAULT CHARSET=utf8mb4` (000030, and 000001) | A: explicit COLLATE per DATABASE.md; B: consistent with recent migrations | maintainer |
| DB-OD-2 | whether `report_run` is partitioned monthly | A: no partitioning, rely on retention purge; B: `PARTITION BY RANGE (created_at)` | maintainer (depends on real write volume) |
| DB-OD-3 | whether artifacts keep the raw `param_json` for reproduction | A: digest only (current design); B: additionally store a masked raw copy | maintainer |
| DB-OD-4 | whether the version table needs version-retention cleanup | A: never purge (current design); B: keep the latest N versions | maintainer |
