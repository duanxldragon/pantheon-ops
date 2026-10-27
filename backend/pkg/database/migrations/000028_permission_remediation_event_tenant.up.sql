-- Permission workbench remediation events: add the tenant dimension.
--
-- WHY THIS FILE EXISTS
-- --------------------
-- `permission_workbench_remediation_event` recorded only `role_key`. Role keys
-- are unique PER TENANT (migration 000024), so two tenants can both own a role
-- `tenant_admin`; the console's remediation history then showed one tenant the
-- other tenant's events. The service now stamps the request tenant and filters
-- on it, so the column has to exist on versioned-migration deployments too
-- (the Go model alone only reaches databases booted with AutoMigrate).
--
-- Existing rows stay at tenant_id = 0. They predate the column and their true
-- tenant cannot be derived from `role_key` alone; attributing them to a tenant
-- would be a guess, and tenant 0 means "platform/global history" which is the
-- honest reading of the old rows.
--
-- IDEMPOTENT: guarded on information_schema, so a re-run is a no-op.
--
-- DOWN: drops the column and its index (see the .down.sql).

SET @has_table := (SELECT COUNT(*) FROM information_schema.tables
                   WHERE table_schema = DATABASE()
                     AND table_name = 'permission_workbench_remediation_event');
SET @has_tenant_id := (SELECT COUNT(*) FROM information_schema.columns
                       WHERE table_schema = DATABASE()
                         AND table_name = 'permission_workbench_remediation_event'
                         AND column_name = 'tenant_id');

SET @s := IF(@has_table > 0 AND @has_tenant_id = 0,
  'ALTER TABLE `permission_workbench_remediation_event` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`, ADD INDEX `idx_permission_remediation_tenant_created` (`tenant_id`, `created_at`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
