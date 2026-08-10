SET @menu_hide_in_nav_column_exists := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_menu'
    AND column_name = 'hide_in_nav'
);

SET @menu_hide_in_nav_add_stmt := IF(
  @menu_hide_in_nav_column_exists = 0,
  'ALTER TABLE `system_menu` ADD COLUMN `hide_in_nav` INT DEFAULT 0 AFTER `active_menu`',
  'SELECT 1'
);

PREPARE menu_hide_in_nav_add_stmt FROM @menu_hide_in_nav_add_stmt;
EXECUTE menu_hide_in_nav_add_stmt;
DEALLOCATE PREPARE menu_hide_in_nav_add_stmt;
