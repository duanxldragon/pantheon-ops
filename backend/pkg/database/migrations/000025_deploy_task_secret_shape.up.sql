-- Upgrade biz_deploy_task_secret to the current model shape.
-- The legacy structure (param_key/encrypted_value/cipher_version/secret_version)
-- was created by an earlier runtime bootstrap; the model now uses
-- secret_key/secret_encrypted. Existing v1-encrypted rows are preserved.
ALTER TABLE `biz_deploy_task_secret`
  CHANGE COLUMN `param_key` `secret_key` VARCHAR(128) NOT NULL,
  CHANGE COLUMN `encrypted_value` `secret_encrypted` TEXT NOT NULL,
  DROP COLUMN `cipher_version`,
  DROP COLUMN `secret_version`,
  ADD UNIQUE KEY `uk_deploy_task_secret` (`task_id`, `secret_key`);
