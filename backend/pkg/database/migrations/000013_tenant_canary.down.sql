-- Rollback tenant canary slice (see 000013_tenant_canary.up.sql).
-- Order: resource columns first, then memberships/master (reverse of up).
-- Guarded via information_schema so partial schemas roll back cleanly.

-- ---- system_dict_item: drop tenant index + column, if present ----
SET @dict_item_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
);
SET @dict_item_tenant_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
    AND index_name = 'idx_dict_item_tenant_code_value'
);
SET @dict_item_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
    AND column_name = 'tenant_id'
);

SET @dict_item_drop_idx_stmt := IF(
  @dict_item_table_exists > 0 AND @dict_item_tenant_idx > 0,
  'ALTER TABLE `system_dict_item` DROP INDEX `idx_dict_item_tenant_code_value`',
  'SELECT 1'
);
PREPARE dict_item_drop_idx_stmt FROM @dict_item_drop_idx_stmt;
EXECUTE dict_item_drop_idx_stmt;
DEALLOCATE PREPARE dict_item_drop_idx_stmt;

SET @dict_item_drop_col_stmt := IF(
  @dict_item_table_exists > 0 AND @dict_item_tenant_col > 0,
  'ALTER TABLE `system_dict_item` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE dict_item_drop_col_stmt FROM @dict_item_drop_col_stmt;
EXECUTE dict_item_drop_col_stmt;
DEALLOCATE PREPARE dict_item_drop_col_stmt;

-- ---- system_dict_type: restore original unique key, drop tenant artifacts ----
SET @dict_type_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
);
SET @dict_type_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND index_name = 'idx_dict_type_tenant_code'
);
SET @dict_type_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND index_name = 'idx_system_dict_type_dict_code'
);
SET @dict_type_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND column_name = 'tenant_id'
);

SET @dict_type_drop_new_uk_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_new_uk > 0,
  'ALTER TABLE `system_dict_type` DROP INDEX `idx_dict_type_tenant_code`',
  'SELECT 1'
);
PREPARE dict_type_drop_new_uk_stmt FROM @dict_type_drop_new_uk_stmt;
EXECUTE dict_type_drop_new_uk_stmt;
DEALLOCATE PREPARE dict_type_drop_new_uk_stmt;

SET @dict_type_drop_col_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_tenant_col > 0,
  'ALTER TABLE `system_dict_type` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE dict_type_drop_col_stmt FROM @dict_type_drop_col_stmt;
EXECUTE dict_type_drop_col_stmt;
DEALLOCATE PREPARE dict_type_drop_col_stmt;

SET @dict_type_restore_uk_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_old_uk = 0,
  'ALTER TABLE `system_dict_type` ADD UNIQUE INDEX `idx_system_dict_type_dict_code` (`dict_code`)',
  'SELECT 1'
);
PREPARE dict_type_restore_uk_stmt FROM @dict_type_restore_uk_stmt;
EXECUTE dict_type_restore_uk_stmt;
DEALLOCATE PREPARE dict_type_restore_uk_stmt;

DROP TABLE IF EXISTS `tenant_memberships`;
DROP TABLE IF EXISTS `tenants`;
