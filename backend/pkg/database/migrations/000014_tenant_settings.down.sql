-- Rollback tenant settings slice (see 000014_tenant_settings.up.sql).
-- Restores the global-unique key and drops tenant artifacts, guarded.

SET @setting_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
);
SET @setting_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'uk_system_setting_tenant_key'
);
SET @setting_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'idx_system_setting_setting_key'
);
SET @setting_plain_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'idx_system_setting_tenant_id'
);
SET @setting_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND column_name = 'tenant_id'
);

-- Drop the plain tenant index first.
SET @setting_drop_plain_idx_stmt := IF(
  @setting_table_exists > 0 AND @setting_plain_idx > 0,
  'ALTER TABLE `system_setting` DROP INDEX `idx_system_setting_tenant_id`',
  'SELECT 1'
);
PREPARE setting_drop_plain_idx_stmt FROM @setting_drop_plain_idx_stmt;
EXECUTE setting_drop_plain_idx_stmt;
DEALLOCATE PREPARE setting_drop_plain_idx_stmt;

-- Restore the composite key to the legacy global-unique key.
SET @setting_drop_new_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_new_uk > 0 AND @setting_old_uk = 0,
  'ALTER TABLE `system_setting` DROP INDEX `uk_system_setting_tenant_key`',
  'SELECT 1'
);
PREPARE setting_drop_new_uk_stmt FROM @setting_drop_new_uk_stmt;
EXECUTE setting_drop_new_uk_stmt;
DEALLOCATE PREPARE setting_drop_new_uk_stmt;

SET @setting_add_old_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_old_uk = 0,
  'ALTER TABLE `system_setting` ADD UNIQUE INDEX `idx_system_setting_setting_key` (`setting_key`)',
  'SELECT 1'
);
PREPARE setting_add_old_uk_stmt FROM @setting_add_old_uk_stmt;
EXECUTE setting_add_old_uk_stmt;
DEALLOCATE PREPARE setting_add_old_uk_stmt;

-- Drop the tenant column last.
SET @setting_drop_col_stmt := IF(
  @setting_table_exists > 0 AND @setting_tenant_col > 0,
  'ALTER TABLE `system_setting` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE setting_drop_col_stmt FROM @setting_drop_col_stmt;
EXECUTE setting_drop_col_stmt;
DEALLOCATE PREPARE setting_drop_col_stmt;
