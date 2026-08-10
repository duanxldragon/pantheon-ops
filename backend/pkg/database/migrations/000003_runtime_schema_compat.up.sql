-- Restore versioned-migration compatibility with the current runtime models.

ALTER TABLE `system_user_session`
  MODIFY COLUMN `expires_at` DATETIME(3) NULL;

ALTER TABLE `system_i18n`
  MODIFY COLUMN `locale_key` VARCHAR(128) NOT NULL DEFAULT '';

UPDATE `system_i18n`
SET `locale_key` = CASE
  WHEN COALESCE(`locale_key`, '') = '' THEN COALESCE(`key`, '')
  ELSE `locale_key`
END;

ALTER TABLE `system_log_oper`
  ADD COLUMN `source_domain` VARCHAR(32) DEFAULT '' AFTER `oper_ip`,
  ADD COLUMN `source_page` VARCHAR(32) DEFAULT '' AFTER `source_domain`,
  ADD COLUMN `failure_category` VARCHAR(32) DEFAULT '' AFTER `status`,
  ADD INDEX `idx_system_log_oper_source_domain_page` (`source_domain`, `source_page`),
  ADD INDEX `idx_system_log_oper_source_page` (`source_page`),
  ADD INDEX `idx_system_log_oper_failure_category` (`failure_category`);
