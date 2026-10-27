# Legacy Ops business DDL (pre-overlay line, 2026-08)

Verbatim copies of the eleven migration pairs that the 2026-08 operations line
(`origin/main`, PRs #96-#118, pinned to base `v0.10.24`) wrote into
`backend/pkg/database/migrations/`. They are kept here as reference; they are
**not** executed and must not be copied back into that directory.

## Why they cannot live in the migration directory

`backend/pkg/database/migrate.go:21` embeds `migrations/*.sql` and hands it to
golang-migrate's `iofs` source. That source rejects two files carrying the same
version number in the same direction, so `RunMigrations` fails and
`cmd/server/main.go` exits at boot. Base owns those numbers now:

| Number | Base chain (current) | Ops legacy line |
|--------|----------------------|-----------------|
| 000012 | `high_traffic_indexes` | `business_schema` |
| 000013 | `tenant_canary` | `deploy_snapshot_idempotency_lease` |
| 000014 | `tenant_settings` | `service_foundation` |
| 000015 | `tenant_auth_logs` | `business_legacy_upgrade` |
| 000016 | `tenant_mfa_challenge` | `deploy_task_attempt` |
| 000017 | `tenant_session_oper_logs` | `k8s_namespace_binding` |
| 000018 | `session_pagination_indexes` | `deploy_credential_ref` |
| 000019 | `i18n_updated_by` | `k8s_cluster_credential_ref` |
| 000020 | `tenant_phase1_core_tables` | `deploy_execution_credential_snapshot` |
| 000021 | `role_data_scope_policy_backfill` | `k8s_credential_ref_backfill` |
| 000022 | *(base-owned)* | `ops_skipped_migrations_repair` |

Independently, an ops repository may not add files under `backend/pkg/**`:
`scripts/business-overlay/check-shared-drift.mjs:149` reports every such file as
`DRIFT_MISSING_IN_BASE`, and `scripts/business-overlay/rebuild-from-base.mjs:253`
throws `Business overlay cannot own generic product path` for any business path
outside `backend/modules/business`, `frontend/src/modules/business` and
`frontend/tests/smoke/business`. A clean rebuild would delete them anyway.

## What replaced them

Business schema is produced from the GORM models in `backend/modules/business/`
(31 distinct `TableName()` values, 38 `AutoMigrate` call sites). Every one of the
19 tables in these files still has a live model today, so no table definition is
lost by moving the SQL out of the runner.

Two gaps are open and belong to base, not to ops:

1. There is no consumer-side versioned-migration channel. The only extension
   point is `contracts.BackendModule.MigrateFunc`, which is GORM AutoMigrate and
   is skipped in the default versioned mode (`ShouldAutoMigrate`,
   `backend/pkg/database/migrate.go:429`). Business DDL is therefore
   unmanaged in a default deployment.
2. Five call sites bypass that gate and AutoMigrate unconditionally at boot
   (`backend/modules/business/cicd/cicd.go:18`,
   `backend/modules/business/opsinit/opsinit.go:58`,
   `backend/modules/business/opsinit/opsinit.go:70`,
   `backend/modules/business/k8s/backup/backup.go:21`,
   `backend/modules/business/k8s/rbac/rbac.go:86`).

The legacy files also carry generated columns and composite unique keys
(`active_code`, `uk_deploy_task_…` from `000013`) that AutoMigrate does not
emit, and the current `DeployTask` model
(`backend/modules/business/deploy/deploy_model.go:89`) has since added
`rollback_enabled`, `rollback_task_id`, `source_task_id`, `rollback_strategy`
and `rollback_snapshot`, which appear in no migration on either line. Treat
these SQL files as history, not as a schema you can replay.
