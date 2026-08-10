-- Revert runtime-schema compatibility adjustments.

ALTER TABLE `system_log_oper`
  DROP INDEX `idx_system_log_oper_failure_category`,
  DROP INDEX `idx_system_log_oper_source_page`,
  DROP INDEX `idx_system_log_oper_source_domain_page`,
  DROP COLUMN `failure_category`,
  DROP COLUMN `source_page`,
  DROP COLUMN `source_domain`;

UPDATE `system_i18n`
SET `locale_key` = CASE
  WHEN COALESCE(`locale_key`, '') = '' THEN COALESCE(`key`, '')
  ELSE `locale_key`
END;

ALTER TABLE `system_i18n`
  MODIFY COLUMN `locale_key` VARCHAR(128) NOT NULL;

UPDATE `system_user_session`
SET `expires_at` = COALESCE(`expires_at`, `refresh_expires_at`, `created_at`, NOW(3))
WHERE `expires_at` IS NULL;

ALTER TABLE `system_user_session`
  MODIFY COLUMN `expires_at` DATETIME(3) NOT NULL;
