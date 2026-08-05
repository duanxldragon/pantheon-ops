-- Add missing indexes found by the review hardening pass.
-- Guarded via information_schema so the migration is idempotent and safe on
-- bootstrapped/partial schemas (same pattern as 000005-000007 compat migrations).
--
-- NOTE: a UNIQUE index on system_i18n (locale, locale_key) was considered but
-- deliberately NOT added here; i18n uniqueness is handled by 000010 on the
-- runtime (locale, `key`) columns.

SET @menu_parent_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_menu'
    AND index_name = 'idx_system_menu_parent_id'
);
SET @menu_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_menu'
);
SET @menu_stmt := IF(
  @menu_table_exists > 0 AND @menu_parent_idx = 0,
  'ALTER TABLE system_menu ADD INDEX idx_system_menu_parent_id (parent_id)',
  'SELECT 1'
);
PREPARE menu_stmt FROM @menu_stmt;
EXECUTE menu_stmt;
DEALLOCATE PREPARE menu_stmt;

SET @dept_parent_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'idx_system_dept_parent_id'
);
SET @dept_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
);
SET @dept_stmt := IF(
  @dept_table_exists > 0 AND @dept_parent_idx = 0,
  'ALTER TABLE system_dept ADD INDEX idx_system_dept_parent_id (parent_id)',
  'SELECT 1'
);
PREPARE dept_stmt FROM @dept_stmt;
EXECUTE dept_stmt;
DEALLOCATE PREPARE dept_stmt;

SET @role_menu_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
    AND index_name = 'idx_system_role_menu_menu'
);
SET @role_menu_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
);
SET @role_menu_stmt := IF(
  @role_menu_table_exists > 0 AND @role_menu_idx = 0,
  'ALTER TABLE system_role_menu ADD INDEX idx_system_role_menu_menu (menu_id)',
  'SELECT 1'
);
PREPARE role_menu_stmt FROM @role_menu_stmt;
EXECUTE role_menu_stmt;
DEALLOCATE PREPARE role_menu_stmt;
