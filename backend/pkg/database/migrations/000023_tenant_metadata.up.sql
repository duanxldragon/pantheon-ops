-- Tenant console remediation (P0-1 + P0-3), see
-- .harness/tasks/2026-10-13-tenant-console-remediation/manifest.json.
--
-- P0-1: `tenants.metadata` is declared by the Go model, both DTOs and the
-- console form, but 000013 never created the column. Every console write
-- therefore failed with `Error 1054 (42S22): Unknown column 'metadata' in
-- 'field list'`, while reads stayed green because the list projection is
-- `SELECT *` and the field is `json:",omitempty"`. Fix forward rather than
-- editing 000013: that migration is already applied on every environment, so
-- appending keeps history immutable.
--
-- P0-3: the reserved platform/global placeholder must sit at `id = 0`
-- (contract TENANT_CONTRACT_V1 section 2.3). 000013 writes it with
-- `INSERT ... SELECT 0, ...`, and MySQL treats an inserted 0 in an
-- AUTO_INCREMENT column as "assign the next counter value" unless
-- `NO_AUTO_VALUE_ON_ZERO` is set. Neither the local `testmysql` DSN nor CI
-- enables it, so on a fresh database the row lands at `id = 1` and leaks into
-- the console as an ordinary tenant. `UPDATE ... SET id = 0` assigns 0
-- literally, so normalization has to go through DML.
--
-- Both steps are guarded via information_schema / PREPARE so partial or
-- already-repaired schemas skip gracefully (same pattern as 000013-000022).

SET @tenants_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'tenants'
);
SET @tenants_metadata_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'tenants'
    AND column_name = 'metadata'
);

SET @tenants_add_metadata_stmt := IF(
  @tenants_table_exists > 0 AND @tenants_metadata_col = 0,
  'ALTER TABLE `tenants` ADD COLUMN `metadata` TEXT NULL AFTER `plan`',
  'SELECT 1'
);
PREPARE tenants_add_metadata_stmt FROM @tenants_add_metadata_stmt;
EXECUTE tenants_add_metadata_stmt;
DEALLOCATE PREPARE tenants_add_metadata_stmt;

-- Normalize the placeholder onto id = 0. Only run when the id = 0 slot is
-- free; otherwise the UPDATE would collide on the primary key. No row
-- references the placeholder by a non-zero id (verified: tenant_memberships,
-- system_setting, sessions and operation logs all carry zero rows for it), so
-- the id change orphans nothing.
SET @placeholder_id0 := (
  SELECT COUNT(*) FROM `tenants` WHERE `id` = 0
);
SET @placeholder_misplaced := (
  SELECT COUNT(*) FROM `tenants` WHERE `code` = '__global__' AND `id` <> 0
);

SET @normalize_placeholder_stmt := IF(
  @placeholder_id0 = 0 AND @placeholder_misplaced > 0,
  'UPDATE `tenants` SET `id` = 0 WHERE `code` = ''__global__'' AND `id` <> 0',
  'SELECT 1'
);
PREPARE normalize_placeholder_stmt FROM @normalize_placeholder_stmt;
EXECUTE normalize_placeholder_stmt;
DEALLOCATE PREPARE normalize_placeholder_stmt;

-- Keep the counter aligned with the normalized layout. InnoDB only ever raises
-- its persisted counter through ALTER TABLE, never lowers it, so this is a
-- best-effort no-op on databases whose counter already advanced past 1.
ALTER TABLE `tenants` AUTO_INCREMENT = 1;
