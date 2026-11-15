-- Append-only safe checkpoint observations; legacy checkpoints are not backfilled.
CREATE TABLE IF NOT EXISTS `biz_deploy_step_history` (
 `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
 `task_id` BIGINT UNSIGNED NOT NULL,
 `task_host_id` BIGINT UNSIGNED NOT NULL,
 `step_id` BIGINT UNSIGNED NOT NULL,
 `step_key` VARCHAR(128) NOT NULL,
 `step_order` INT NOT NULL,
 `attempt_no` INT NOT NULL,
 `status` VARCHAR(32) NOT NULL,
 `state_version` BIGINT UNSIGNED NOT NULL,
 `started_at` DATETIME(3) NULL,
 `finished_at` DATETIME(3) NULL,
 `captured_at` DATETIME(3) NOT NULL,
 PRIMARY KEY (`id`),
 UNIQUE KEY `uk_deploy_step_history_version` (`step_id`, `state_version`),
 KEY `idx_deploy_step_history_task` (`task_id`, `task_host_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
