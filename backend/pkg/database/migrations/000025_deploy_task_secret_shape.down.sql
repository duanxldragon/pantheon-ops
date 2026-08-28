-- Restore the legacy biz_deploy_task_secret shape (down migration).
ALTER TABLE `biz_deploy_task_secret`
  DROP INDEX `uk_deploy_task_secret`,
  CHANGE COLUMN `secret_key` `param_key` VARCHAR(128) NOT NULL,
  CHANGE COLUMN `secret_encrypted` `encrypted_value` TEXT NOT NULL,
  ADD COLUMN `cipher_version` VARCHAR(16) NOT NULL DEFAULT 'v1',
  ADD COLUMN `secret_version` INT NOT NULL DEFAULT 1;
