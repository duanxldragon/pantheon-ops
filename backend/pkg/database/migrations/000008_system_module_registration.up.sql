CREATE TABLE IF NOT EXISTS `system_module_registration` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) DEFAULT '',
  `display_name` VARCHAR(128) DEFAULT '',
  `scope` VARCHAR(32) DEFAULT '',
  `source` VARCHAR(32) DEFAULT '',
  `owner` VARCHAR(128) DEFAULT '',
  `bounded_context` VARCHAR(128) DEFAULT '',
  `summary` VARCHAR(255) DEFAULT '',
  `source_table` VARCHAR(128) DEFAULT '',
  `auto_recycle` TINYINT(1) NOT NULL DEFAULT 0,
  `table_name` VARCHAR(128) DEFAULT '',
  `status` INT DEFAULT 1,
  `installed_at` VARCHAR(64) DEFAULT '',
  `uninstalled_at` VARCHAR(64) DEFAULT '',
  `last_verified_at` VARCHAR(64) DEFAULT '',
  `last_error` VARCHAR(512) DEFAULT '',
  `last_verification_result` TEXT,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_system_module_registration_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

SET @legacy_module_registration_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'module_registration'
);
SET @module_registration_copy_stmt := IF(
  @legacy_module_registration_exists = 1,
  'INSERT INTO `system_module_registration` (
     `name`, `display_name`, `source`, `status`, `installed_at`
   )
   SELECT
     legacy.`name`,
     legacy.`display_name`,
     legacy.`module_type`,
     legacy.`status`,
     COALESCE(DATE_FORMAT(legacy.`registered_at`, ''%Y-%m-%dT%H:%i:%sZ''), '''')
   FROM `module_registration` legacy
   WHERE COALESCE(legacy.`name`, '''') <> ''''
     AND NOT EXISTS (
       SELECT 1
       FROM `system_module_registration` target
       WHERE target.`name` = legacy.`name`
     )',
  'SELECT 1'
);
PREPARE module_registration_copy_stmt FROM @module_registration_copy_stmt;
EXECUTE module_registration_copy_stmt;
DEALLOCATE PREPARE module_registration_copy_stmt;
