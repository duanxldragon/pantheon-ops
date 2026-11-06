---
title: Notice Center Database Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/designs/NOTICE_CENTER_API_CONTRACT.md
updated_at: 2026-11-04
---

# Notice Center Database Design

Chinese version: [NOTICE_CENTER_DB_DESIGN.md](./NOTICE_CENTER_DB_DESIGN.md)

> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Linked designs:
- `NOTICE_CENTER_DESIGN.md`
- `NOTICE_CENTER_API_CONTRACT.md`
- `NOTICE_CENTER_TEST_PLAN.md`

---

This document fixes the tables, columns, indexes, migrations and purge shape of the notice center so implementers do not have to invent DDL. Every table and column name here is a contract to be implemented.

## 1. Preconditions and This Repository's Table Rules

This document inherits `docs/designs/DATABASE.md` clause by clause:

| Rule | Source | How it lands here |
|------|--------|-------------------|
| InnoDB + `utf8mb4_general_ci` | `DATABASE.md` §1 | all three tables |
| primary key `bigint unsigned NOT NULL AUTO_INCREMENT` | `DATABASE.md` §1 | auto-increment, no snowflake (the snowflake `tenants.id` in `docs/contracts/TENANT_CONTRACT_V1.md` §2.1 is an exception for tenant master data, not the general rule) |
| soft delete `deleted_at datetime(3) NULL` (GORM default) | `DATABASE.md` §1 | only on the body table, see §5 |
| base tables use the `system_` prefix | `DATABASE.md` §2 | `system_notice*`; the `platform_notice` prefix is **rejected** because §2 pins `system_` for base tables — `platform` is a logical layer, not a table prefix |
| no physical foreign keys | `DATABASE.md` §4 | relationships maintained only by the Service layer |
| low-cardinality `status` gets no standalone index | `DATABASE.md` §4 | only as part of composite indexes |
| authoritative schema path = numbered golang-migrate plus Go seeds | `DATABASE.md` §5 | see §7 |

## 2. Table Inventory

| Table | Role | Required in V1 |
|-------|------|----------------|
| `system_notice` | message body, one row per message | yes |
| `system_notice_target` | recipient view, one row per recipient × message | yes |
| `system_notice_delivery` | out-of-app delivery ledger, one row per recipient × message × channel | created, but only written once a non-`in_app` channel is registered |
| `system_notice_channel_config` | channel parameters and switches | no, decided together with D-1 in §9 |

## 3. Relationships

```
system_notice (1) ──< system_notice_target (N) ──< system_notice_delivery (0..M, M = number of out-of-app channels)
```

No foreign keys (§1), so the Service layer owns these three relationships and the deletion order is in §8.3.

## 4. Column-Level DDL Intent

"Intent" = semantics + type + nullability/default rules; the SQL itself follows `docs/designs/DATABASE.md`.

### 4.1 `system_notice` (message body)

| Column | Type | Null/default | Intent |
|--------|------|--------------|--------|
| `id` | `bigint unsigned` | PK AI | primary key |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | tenant attribution; `0` = platform-level/global resource (`TENANT_CONTRACT_V1` §2.3); nullable is **forbidden** by the same clause |
| `category` | `varchar(32)` | `NOT NULL` | `task/alert/audit/system/business` enum per design §4.3 |
| `severity` | `varchar(16)` | `NOT NULL DEFAULT 'info'` | `info/warning/critical` |
| `status` | `varchar(16)` | `NOT NULL DEFAULT 'published'` | body state `draft/published/withdrawn`, a separate column from view state (design §5.3) |
| `source_domain` | `varchar(64)` | `NOT NULL` | source domain id such as `platform`, `system.auth`, `business.*`; used for audit and jump attribution |
| `source_type` | `varchar(64)` | `NOT NULL DEFAULT ''` | event/object type inside the source domain |
| `source_id` | `varchar(64)` | `NOT NULL DEFAULT ''` | source object id, a string rather than bigint because source domains own their key shape (design §3.2) |
| `title` | `varchar(256)` | `NOT NULL` | already rendered text (design §4.2) |
| `summary` | `varchar(512)` | `NOT NULL DEFAULT ''` | list summary; long text never enters a list field |
| `content` | `text` | `NULL` | detail body; V1 allows plain text / controlled Markdown only (decided by `content_type`) |
| `content_type` | `varchar(16)` | `NOT NULL DEFAULT 'text'` | `text/markdown`; rich text is a non-goal in design §2 |
| `payload` | `json` | `NULL` | jump parameters and a snapshot of source context; the platform never parses it, only passes it to the detail page |
| `jump_type` | `varchar(16)` | `NOT NULL DEFAULT 'none'` | `route/external-url/none` |
| `jump_target` | `varchar(512)` | `NOT NULL DEFAULT ''` | route path or URL, validated before persisting per design §6.2 |
| `jump_perm` | `varchar(128)` | `NOT NULL DEFAULT ''` | permission key required by the jump target; empty = no extra requirement. Pre-hiding for the frontend only, the server still authorizes independently |
| `dedup_key` | `varchar(191)` | `NULL DEFAULT NULL` | intake idempotency key (design §8.1). **Must be nullable**: a MySQL unique index admits unlimited `NULL` rows but treats `''` as a value, so an empty string would make the second keyless message collide with the unique index |
| `expires_at` | `datetime(3)` | `NULL` | business expiry; hidden from lists afterwards, a different concept from the retention purge of §8 |
| `created_by` | `bigint` | `NOT NULL DEFAULT 0` | the design's `actorId`; `0` for system-generated messages |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM defaults |
| `deleted_at` | `datetime(3)` | `NULL` | body soft delete, keeping the message fact for audit tracing (design §10.3) |

### 4.2 `system_notice_target` (recipient view)

| Column | Type | Null/default | Intent |
|--------|------|--------------|--------|
| `id` | `bigint unsigned` | PK AI | primary key |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | same value as the body, assigned from the request context on write (`TENANT_CONTRACT_V1` §2.3.1 clause 2) |
| `notice_id` | `bigint unsigned` | `NOT NULL` | → `system_notice.id`, no physical foreign key |
| `recipient_id` | `bigint` | `NOT NULL` | recipient `system_user.id`; no existence check — source data validity belongs to the source domain (design §3.2) |
| `read_at` | `datetime(3)` | `NULL` | non-null means read (design §9.1); never cleared once written |
| `archived_at` | `datetime(3)` | `NULL` | non-null means archived |
| `delivered_at` | `datetime(3)` | `NULL` | `in_app` delivery instant; out-of-app delivery lives in `system_notice_delivery` |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM defaults |

This table deliberately has **no** `deleted_at`; reason in §5.3.

### 4.3 `system_notice_delivery` (out-of-app delivery ledger)

| Column | Type | Null/default | Intent |
|--------|------|--------------|--------|
| `id` | `bigint unsigned` | PK AI | primary key |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | as above |
| `notice_id` | `bigint unsigned` | `NOT NULL` | redundant, for per-message aggregation and purge |
| `target_id` | `bigint unsigned` | `NOT NULL` | → `system_notice_target.id`, the owning delivery unit |
| `recipient_id` | `bigint` | `NOT NULL` | redundant, for cross-tenant operational read display |
| `channel` | `varchar(32)` | `NOT NULL` | `email/sms/webhook`; `in_app` never lands here (design §7.3) |
| `status` | `varchar(16)` | `NOT NULL DEFAULT 'pending'` | `pending/success/failed/dead/skipped` (design §5.2) |
| `attempts` | `int` | `NOT NULL DEFAULT 0` | including the first attempt, default cap 5 (design §8.2) |
| `next_attempt_at` | `datetime(3)` | `NULL` | backoff driver and worker claim condition |
| `last_attempt_at` | `datetime(3)` | `NULL` | diagnostics |
| `external_id` | `varchar(191)` | `NOT NULL DEFAULT ''` | channel-side message id for troubleshooting and secondary dedup; **never** stores credentials |
| `last_error` | `varchar(512)` | `NULL` | truncated; tokens/passwords must never be stored (`docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`) |
| `sent_at` | `datetime(3)` | `NULL` | delivery acknowledgement time |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM defaults |

Also no `deleted_at`: ledger rows are operational process data, hard-deleted at expiry (design §10.3), with `dead` rows surviving one extra period.

## 5. Indexes and Unique Keys

### 5.1 The hard rule in this repository: MySQL unique indexes exclude soft-delete columns

- `docs/designs/DATABASE.md` §3.3.2 records the username approach: on delete the unique identifier is **archived by renaming** first, then the row is soft-deleted, so the key can be reused; §3.3.3 extends the same strategy to `system_role.role_key`, `system_post.post_code`, `system_dict_type.dict_code` and `system_dict_item(dict_code, item_value)`.
- Verifiable in code: `backend/modules/system/iam/user/user_username.go:18` (`deletedUsernamePrefix = "__deleted_user_"`), `:144`, `:164` (candidate rename with conflict retry).
- Conclusion: never stuff `deleted_at` into a unique index to "distinguish deleted rows". Releasing a unique key has exactly two exits: **rename**, or — when the row has no retention value — **delete it physically**.

The notice center chooses:

| Table | Exit | Reason |
|-------|------|--------|
| `system_notice` | renaming does not apply → no business unique key except `dedup_key`; soft delete is fine | the body has no "recreate with the same name" need |
| `system_notice_target` | physical delete | the view row has no audit value; renaming (suffixing `recipient_id`) would corrupt the recipient identity |
| `system_notice_delivery` | physical delete | same |

### 5.2 Key inventory

| Table | Key | Columns | Intent |
|-------|-----|---------|--------|
| `system_notice` | `uk_notice_tenant_dedup` | `(tenant_id, dedup_key)` | intake idempotency (design §8.1); `NULL` in `dedup_key` opts out of the constraint |
| `system_notice` | `idx_notice_tenant_created` | `(tenant_id, created_at)` | platform-surface list ordered per tenant |
| `system_notice` | `idx_notice_source` | `(source_domain, source_type, source_id)` | source domain reverse lookup "which messages did this business record emit" |
| `system_notice` | `idx_notice_status_expires` | `(status, expires_at)` | withdrawal/expiry sweep; satisfies the "no standalone status index" rule as a composite |
| `system_notice_target` | `uk_notice_target_notice_recipient` | `(notice_id, recipient_id)` | fan-out cannot duplicate; no soft-delete column (§5.1) |
| `system_notice_target` | `idx_notice_target_recipient_read` | `(tenant_id, recipient_id, read_at)` | covering index for the unread count (design §9.2) |
| `system_notice_target` | `idx_notice_target_recipient_created` | `(recipient_id, id)` | recipient list pagination cursor (§6.2) |
| `system_notice_target` | `idx_notice_target_archived` | `(recipient_id, archived_at)` | archived view filtering |
| `system_notice_delivery` | `uk_notice_delivery_target_channel` | `(target_id, channel)` | one row per delivery unit, retries update it in place (§5.4) |
| `system_notice_delivery` | `idx_notice_delivery_claim` | `(status, next_attempt_at)` | worker claim scan |
| `system_notice_delivery` | `idx_notice_delivery_notice` | `(notice_id)` | per-message cleanup/aggregation |

`external_id` gets no unique key: channel namespaces are not interchangeable, and a cross-channel key would create false conflicts.

### 5.3 Why the view table has no soft-delete column

If `system_notice_target` carried `deleted_at`, then after a user deletes a message, receiving the same `dedup_key` again would find `uk(notice_id, recipient_id)` (or the idempotency chain) pinned by an invisible soft-deleted row — precisely the problem `DATABASE.md` §3.3.2 describes and works around by renaming. The notice view has no room to rename (§5.1) and no audit retention value (message facts live in the body table and the operation log), so delete is a physical delete. The "delete" row of the design's state machine §5.1 and §10.3 align with this.

### 5.4 Why there is no attempt-history table

`attempts + last_error + last_attempt_at` answers "why was it not delivered" and "should we retry"; per-attempt history is an observability enhancement. V1 does not create a fourth table (minimum-complexity ladder) and instead delegates failure distribution to metrics (`pantheon_notice_delivery_total{outcome}` in design §14). Whether per-attempt history is needed is registered as D-3 in §9.

## 6. Data Shape and Read-Path Constraints

### 6.1 Bounded pagination

- All list paging goes through `common.NormalizePagination(page, pageSize, maxPageSize)` (`backend/pkg/common/handler.go:39-50`: `page<=0 → 1`, `pageSize<=0 → 10`, clamps to the module cap and derives the offset).
- The notice module cap is **100**, matching existing modules: `backend/modules/system/audit/audit_service.go:47` (`maxOperationLogPageSize = 100`), `backend/modules/system/config/dict/dict_service.go:24`, `permission_service.go:937`, `role_helper.go:534`, `post_service.go:663` all pass 100. The contract only promises "bounded", not that oversized input is silently shrunk (following the wording of `TENANT_CONTRACT_V1` §2.5 rule 4).
- Exports (if ever opened) get their own row cap, following the existing `maxOperationLogExportRows` approach (constant at `backend/modules/system/audit/audit_service.go:48`, applied by `Limit()` at `:372`).

### 6.2 Deep paging

The recipient list sorts by `(recipient_id, id)` as a cursor; constant time for arbitrary `page` is not promised. A platform admin surface needing deep paging filters by a time window instead of a large offset jump.

### 6.3 Volume assumptions

- One platform-level message = 1 body row + N target rows (N = recipient count). A broadcast message produces user-count-scale rows, so the §8.2 fan-out must batch.
- V1 does not promise broadcast performance: the suggested fan-out batch is 500 rows/batch, in the same spirit as the batch input cap at `audit_service.go:357`.

## 7. Migration Numbering and Shape

### 7.1 Numbering

The current highest numbered migration is `backend/pkg/database/migrations/000031_drop_permission_role_data_scope_policy`, and 000001-000031 are contiguous in that directory. Therefore:

| Migration | Content |
|-----------|---------|
| `000032_notice_center.up.sql` | create `system_notice`, `system_notice_target`, `system_notice_delivery` (one version per feature package, three tables in one) |
| `000032_notice_center.down.sql` | `DROP TABLE IF EXISTS` the same three tables in reverse order |
| `000033_task_center.up.sql` | task center tables (see `TASK_CENTER_DB_DESIGN.md`) |

Basis: migrations run through golang-migrate over an embedded FS (`backend/pkg/database/migrate.go:14-22`), and files must exist as `<version>_<name>.up.sql` / `.down.sql` pairs; `000030` and `000031` are the paired examples.

### 7.2 Up-shape requirements

Following `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql`:

1. A header comment block stating **WHY THIS TABLE EXISTS / SEMANTICS**, pinning the reasoning next to the DDL (000030 does exactly this for "platform-global table has no `tenant_id`", "-1 means unlimited", "seed inserts, never updates").
2. `CREATE TABLE IF NOT EXISTS` with `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`.
3. Where seeding is needed, use the per-key idempotent `INSERT ... SELECT ... WHERE NOT EXISTS (...)` form (000030 comment lines 23-28 give the reason: migration replay is a real state here and rewriting stored values would silently undo the operator; the same rule appears in `TENANT_CONTRACT_V1` §2.4 invariant 6).
4. The notice center V1 **seeds no message data in a migration**; settings, menus and permissions go through Go seeds (§7.4).

### 7.3 Down-shape requirements

- The down shape of this migration is "these tables do not exist at version 31", so it must drop only the three tables it created and never touch neighbours.
- Pinned with the existing test shape: `backend/pkg/database/migrate_down_shape_test.go` (`TestDropPolicyTableDownReconstructsVersion30Shape` byte-compares `SHOW CREATE TABLE` between "migrated straight up to N" and "up to N+1 then back down to N") and `backend/pkg/database/migrate_down_test.go`. The notice package should add an isomorphic test: up to 32 → down to 31 → assert the three tables are gone and the rest of the schema equals the version-31 baseline.
- Ordering: 000031 already drops a table; this package must not re-introduce it.

### 7.4 The dual AutoMigrate path

`docs/contracts/TENANT_CONTRACT_V1.md` §2.4 records the repository's dual path: numbered DDL is authoritative, while AutoMigrate deployments go through the module's own `MigrateFunc` (the `PlanStore.Migrate()` example). The notice center needs both:

- numbered migration: `000032_notice_center`
- module `MigrateFunc`: `contracts.FuncModule{ModuleName: "notice", MigrateFunc: ...}` (shape at `backend/modules/system/system_modules.go:332-355`), so a development database without migrations still boots.

Seed side: menus via `SeedMenusFunc` (idempotent `ensureMenuSeeds`, `backend/modules/system/seed.go:88-103`), i18n canonical entries, and the retention keys in `system_setting` (`backend/modules/system/config/setting/setting_seed.go:65-68` is the same shape).

### 7.5 Tenant scope registration duty

All three tables carry `tenant_id`, so they must be registered in `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §2/§3 before merge (`TENANT_CONTRACT_V1` §2.3: a new table may only merge after registration). Registration entries:

| Table | Scope | Target unique-key form |
|-------|-------|------------------------|
| `system_notice` | `tenant-replicated` (platform-level messages use `tenant_id = 0`) | `uk(tenant_id, dedup_key)`, unique inside the tenant |
| `system_notice_target` | same | `uk(notice_id, recipient_id)` |
| `system_notice_delivery` | same | `uk(target_id, channel)` |

Read/write obligations cite `TENANT_CONTRACT_V1` §2.3.1 directly (reads carry the tenant predicate, writes take the tenant from context, bare-key existence checks count as reads), so review can reject by clause number.

## 8. Lifecycle and Cleanup

### 8.1 Intake fan-out

1. Validate `dedup_key`; a hit on `uk_notice_tenant_dedup` returns the existing `notice_id` without inserting (design §8.1).
2. Insert the body row (`tenant_id` from context, never from the request body).
3. Insert target rows in batches of 500, ignoring conflicts (`INSERT IGNORE` semantics or equivalent `(notice_id, recipient_id)` de-duplication).
4. For out-of-app channels, generate delivery rows per target × channel.

### 8.2 Fan-out must be re-entrant

If steps 3/4 crash and replay, no duplicates may appear — guaranteed by the two unique keys of §5.2, not by "select then insert".

### 8.3 Deletion order

Withdrawal (body `status = withdrawn`) deletes no rows. Cleanup runs delivery → target → body (soft or hard), so no orphan view row can survive its body; without foreign keys, ordering is the only guarantee.

### 8.4 Cleanup task

`maintenance.Task{Name: "notice.retention", Interval: maintenance.DefaultInterval, Run: ...}`, registered exactly like `backend/modules/system/audit/audit_service.go:317-323`; retention days are read from `system_setting` with a default fallback (the `getRetentionDaysFromSetting` pattern at line 325 of the same file).

## 9. Open Decisions (Data Layer)

| ID | Topic | Options | Recommendation | Decider |
|----|-------|---------|----------------|---------|
| D-1 | carrier of channel configuration | A `system_notice_channel_config` table; B `system_setting` keys; C config file / env var | V1 has no out-of-app channel → start with B (zero new table), revisit A when a channel lands | maintainer |
| D-2 | does a broadcast need its own table | A fan-out into target rows; B body plus lazy view (row created on first read) | A: lazy views make the unread count an uncomputable aggregate, a far larger cost than the fan-out rows | maintainer |
| D-3 | per-attempt history | A single row retried in place (this document); B a fourth `system_notice_delivery_attempt` table | A, until someone actually needs "why did attempt 3 fail" | maintainer |
| D-4 | type of the `payload` column | A `json`; B `text` holding a JSON string | A: MySQL is the standard runtime (`DATABASE.md` §3.1 migration constraint) and the platform never parses the column | maintainer |
| D-5 | view-row retention and "restore a deleted message" | A physical delete (this document); B body soft delete plus view physical delete | B as written is this document; if "restore what the user deleted" becomes a requirement, the whole package moves to soft delete plus a rename strategy | maintainer |

## 10. Definition of Done

This schema design is implementable if and only if:

- the three tables match §4/§5, with both the numbered migration and the `MigrateFunc` path
- up/down are paired and the down shape is pinned by a test (§7.3)
- tenant columns and unique keys are registered in `TENANT_RESOURCE_SCOPE_MATRIX` (§7.5)
- pagination and batch caps follow §6.1
- D-1/D-2 of §9 are decided before implementation
