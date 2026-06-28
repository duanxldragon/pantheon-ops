ALTER TABLE `system_refresh_version`
  ADD COLUMN `created_at` DATETIME(3) NULL AFTER `version`;

UPDATE `system_refresh_version`
SET `created_at` = COALESCE(`created_at`, `updated_at`, NOW(3));
