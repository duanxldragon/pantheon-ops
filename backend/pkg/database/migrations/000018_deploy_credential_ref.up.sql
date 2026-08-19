CREATE TABLE IF NOT EXISTS `biz_deploy_credential_ref` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(128) NOT NULL,
  `username` VARCHAR(128) NOT NULL,
  `auth_mode` VARCHAR(32) NOT NULL,
  `secret_encrypted` TEXT NOT NULL,
  `version` BIGINT UNSIGNED NOT NULL DEFAULT 1,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`), UNIQUE KEY `uk_deploy_credential_name_deleted` (`name`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
