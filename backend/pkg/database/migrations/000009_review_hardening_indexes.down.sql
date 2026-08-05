SET @role_menu_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
    AND index_name = 'idx_system_role_menu_menu'
);
SET @role_menu_stmt := IF(
  @role_menu_idx > 0,
  'DROP INDEX idx_system_role_menu_menu ON system_role_menu',
  'SELECT 1'
);
PREPARE role_menu_stmt FROM @role_menu_stmt;
EXECUTE role_menu_stmt;
DEALLOCATE PREPARE role_menu_stmt;

SET @dept_parent_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'idx_system_dept_parent_id'
);
SET @dept_stmt := IF(
  @dept_parent_idx > 0,
  'DROP INDEX idx_system_dept_parent_id ON system_dept',
  'SELECT 1'
);
PREPARE dept_stmt FROM @dept_stmt;
EXECUTE dept_stmt;
DEALLOCATE PREPARE dept_stmt;

SET @menu_parent_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_menu'
    AND index_name = 'idx_system_menu_parent_id'
);
SET @menu_stmt := IF(
  @menu_parent_idx > 0,
  'DROP INDEX idx_system_menu_parent_id ON system_menu',
  'SELECT 1'
);
PREPARE menu_stmt FROM @menu_stmt;
EXECUTE menu_stmt;
DEALLOCATE PREPARE menu_stmt;
