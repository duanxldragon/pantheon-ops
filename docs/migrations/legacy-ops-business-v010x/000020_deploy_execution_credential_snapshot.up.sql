ALTER TABLE `biz_deploy_task`
  ADD COLUMN `credential_ref_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `credential_ref_version` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  ADD COLUMN `ssh_host_fingerprint` VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN `execution_timeout_seconds` INT NOT NULL DEFAULT 1800;

CREATE INDEX `idx_deploy_task_credential_ref` ON `biz_deploy_task` (`credential_ref_id`);
