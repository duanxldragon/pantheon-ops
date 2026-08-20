-- Deploy immutability, idempotency and lease support. These columns make a
-- Deploy task a durable, immutable change record: execution intent and target
-- are frozen as snapshots, Start carries a persisted idempotency key, callback
-- reports carry a correlation identity, and host execution is lease-guarded.

ALTER TABLE `biz_deploy_task` ADD COLUMN `execution_snapshot` JSON;
ALTER TABLE `biz_deploy_task` ADD COLUMN `target_snapshot` JSON;
ALTER TABLE `biz_deploy_task` ADD COLUMN `start_request_key` VARCHAR(128) NOT NULL DEFAULT '';

ALTER TABLE `biz_deploy_task_host` ADD COLUMN `ssh_port` INT NOT NULL DEFAULT 22;
ALTER TABLE `biz_deploy_task_host` ADD COLUMN `business_scope_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
ALTER TABLE `biz_deploy_task_host` ADD COLUMN `report_key` VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE `biz_deploy_task_host` ADD COLUMN `resolved_at` DATETIME(3) DEFAULT NULL;

CREATE TABLE IF NOT EXISTS `biz_deploy_host_lease` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `host_id` BIGINT UNSIGNED NOT NULL,
  `task_id` BIGINT UNSIGNED NOT NULL,
  `owner` VARCHAR(128) NOT NULL,
  `expires_at` DATETIME(3) NOT NULL,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_deploy_host_lease_host` (`host_id`),
  KEY `idx_deploy_host_lease_task_id` (`task_id`),
  KEY `idx_deploy_host_lease_expires_at` (`expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
