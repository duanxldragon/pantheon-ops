CREATE TABLE IF NOT EXISTS `biz_deploy_task_attempt` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `task_id` BIGINT UNSIGNED NOT NULL,
  `task_host_id` BIGINT UNSIGNED NOT NULL,
  `attempt_no` INT NOT NULL,
  `status` VARCHAR(32) NOT NULL DEFAULT 'running',
  `worker_id` VARCHAR(128) DEFAULT '',
  `lease_expires_at` DATETIME(3) DEFAULT NULL,
  `started_at` DATETIME(3) DEFAULT NULL,
  `finished_at` DATETIME(3) DEFAULT NULL,
  `error_message` VARCHAR(512) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_deploy_attempt_task_host` (`task_host_id`),
  KEY `idx_deploy_attempt_task_status` (`task_id`, `status`),
  KEY `idx_deploy_attempt_lease` (`lease_expires_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
