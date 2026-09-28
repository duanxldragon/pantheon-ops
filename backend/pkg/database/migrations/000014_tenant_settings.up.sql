-- Tenant settings slice (queue-5, contract TENANT_CONTRACT_V1 + scope matrix:
-- `system_setting` is `tenant-overridable`, uniqueness becomes (tenant_id, setting_key)).
-- Behavior under compat (default): tenant_id=0 everywhere — no behavior change.
-- Guarded via information_schema (same compat pattern as 000008/000010/000011/000013).

-- ---- system_setting: add tenant_id, swap global-unique key for composite ----

SET @setting_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
);
SET @setting_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND column_name = 'tenant_id'
);
SET @setting_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'idx_system_setting_setting_key'
);
SET @setting_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'uk_system_setting_tenant_key'
);

SET @setting_add_col_stmt := IF(
  @setting_table_exists > 0 AND @setting_tenant_col = 0,
  'ALTER TABLE `system_setting` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE setting_add_col_stmt FROM @setting_add_col_stmt;
EXECUTE setting_add_col_stmt;
DEALLOCATE PREPARE setting_add_col_stmt;

SET @setting_drop_old_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_old_uk > 0 AND @setting_new_uk = 0,
  'ALTER TABLE `system_setting` DROP INDEX `idx_system_setting_setting_key`',
  'SELECT 1'
);
PREPARE setting_drop_old_uk_stmt FROM @setting_drop_old_uk_stmt;
EXECUTE setting_drop_old_uk_stmt;
DEALLOCATE PREPARE setting_drop_old_uk_stmt;

SET @setting_add_new_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_new_uk = 0,
  'ALTER TABLE `system_setting` ADD UNIQUE INDEX `uk_system_setting_tenant_key` (`tenant_id`, `setting_key`)',
  'SELECT 1'
);
PREPARE setting_add_new_uk_stmt FROM @setting_add_new_uk_stmt;
EXECUTE setting_add_new_uk_stmt;
DEALLOCATE PREPARE setting_add_new_uk_stmt;

SET @setting_add_plain_idx_stmt := IF(
  @setting_table_exists > 0 AND @setting_tenant_col > 0 AND (
    SELECT COUNT(*) FROM information_schema.statistics
    WHERE table_schema = DATABASE()
      AND table_name = 'system_setting'
      AND index_name = 'idx_system_setting_tenant_id'
  ) = 0,
  'ALTER TABLE `system_setting` ADD INDEX `idx_system_setting_tenant_id` (`tenant_id`)',
  'SELECT 1'
);
PREPARE setting_add_plain_idx_stmt FROM @setting_add_plain_idx_stmt;
EXECUTE setting_add_plain_idx_stmt;
DEALLOCATE PREPARE setting_add_plain_idx_stmt;
