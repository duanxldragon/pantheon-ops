ALTER TABLE `system_dept`
  DROP INDEX `idx_system_dept_dept_code`,
  DROP COLUMN `dept_code`;

SET @legacy_generator_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'generator_datasource'
);
SET @current_generator_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_generator_datasource'
);
SET @generator_table_rename_stmt := IF(
  @legacy_generator_table_exists = 1 AND @current_generator_table_exists = 0,
  'RENAME TABLE `generator_datasource` TO `system_generator_datasource`',
  'SELECT 1'
);
PREPARE generator_table_rename_stmt FROM @generator_table_rename_stmt;
EXECUTE generator_table_rename_stmt;
DEALLOCATE PREPARE generator_table_rename_stmt;

CREATE TABLE IF NOT EXISTS `system_generator_datasource` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `driver` VARCHAR(32) NOT NULL DEFAULT 'mysql',
  `status` TINYINT DEFAULT 1,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `system_generator_datasource`
  ADD COLUMN `host` VARCHAR(255) NOT NULL DEFAULT '' AFTER `driver`,
  ADD COLUMN `port` INT NOT NULL DEFAULT 3306 AFTER `host`,
  ADD COLUMN `database_name` VARCHAR(128) NOT NULL DEFAULT '' AFTER `port`,
  ADD COLUMN `username` VARCHAR(128) NOT NULL DEFAULT '' AFTER `database_name`,
  ADD COLUMN `password_encrypted` VARCHAR(1024) DEFAULT '' AFTER `username`,
  ADD COLUMN `readonly_scope` VARCHAR(32) NOT NULL DEFAULT 'metadata_only' AFTER `status`,
  ADD COLUMN `remark` VARCHAR(255) DEFAULT '' AFTER `readonly_scope`,
  ADD COLUMN `last_checked_at` DATETIME(3) NULL AFTER `remark`,
  ADD COLUMN `last_check_status` VARCHAR(32) DEFAULT '' AFTER `last_checked_at`,
  ADD COLUMN `last_check_error` VARCHAR(255) DEFAULT '' AFTER `last_check_status`;
