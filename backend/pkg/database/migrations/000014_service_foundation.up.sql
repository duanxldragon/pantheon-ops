CREATE TABLE IF NOT EXISTS `biz_application` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(128) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `business_scope_id` BIGINT UNSIGNED NOT NULL,
  `business_scope_name` VARCHAR(255) DEFAULT '',
  `dept_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `owner` VARCHAR(128) DEFAULT '',
  `remark` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `created_by` VARCHAR(64) DEFAULT '',
  `updated_by` VARCHAR(64) DEFAULT '',
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_biz_application_code_active` (`code`, `deleted_at`),
  KEY `idx_biz_application_scope` (`business_scope_id`),
  KEY `idx_biz_application_dept` (`dept_id`),
  KEY `idx_biz_application_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `biz_service` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `application_id` BIGINT UNSIGNED NOT NULL,
  `code` VARCHAR(128) NOT NULL,
  `name` VARCHAR(255) NOT NULL,
  `runtime_type` VARCHAR(64) NOT NULL,
  `description` TEXT,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `created_by` VARCHAR(64) DEFAULT '',
  `updated_by` VARCHAR(64) DEFAULT '',
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_biz_service_application_code_active` (`application_id`, `code`, `deleted_at`),
  KEY `idx_biz_service_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `biz_service_instance` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `service_id` BIGINT UNSIGNED NOT NULL,
  `environment` VARCHAR(32) NOT NULL,
  `target_type` VARCHAR(32) NOT NULL,
  `host_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `k8s_cluster_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `namespace` VARCHAR(255) DEFAULT '',
  `workload_kind` VARCHAR(64) DEFAULT '',
  `workload_name` VARCHAR(255) DEFAULT '',
  `desired_version` VARCHAR(128) DEFAULT '',
  `current_version` VARCHAR(128) DEFAULT '',
  `lifecycle_version` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `created_by` VARCHAR(64) DEFAULT '',
  `updated_by` VARCHAR(64) DEFAULT '',
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_biz_service_instance_service` (`service_id`),
  KEY `idx_biz_service_instance_vm_target` (`service_id`, `target_type`, `host_id`),
  KEY `idx_biz_service_instance_k8s_target` (`service_id`, `target_type`, `k8s_cluster_id`, `namespace`, `workload_kind`, `workload_name`),
  KEY `idx_biz_service_instance_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `biz_deploy_task`
  ADD COLUMN `service_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `business_scope_name`,
  ADD COLUMN `service_instance_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `service_id`,
  ADD COLUMN `service_name` VARCHAR(255) DEFAULT '' AFTER `service_instance_id`,
  ADD COLUMN `service_instance_name` VARCHAR(255) DEFAULT '' AFTER `service_name`,
  ADD KEY `idx_deploy_task_service_instance` (`service_instance_id`);

ALTER TABLE `biz_k8s_release`
  ADD COLUMN `service_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `dept_id`,
  ADD COLUMN `service_instance_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `service_id`,
  ADD COLUMN `service_name` VARCHAR(255) DEFAULT '' AFTER `service_instance_id`,
  ADD COLUMN `service_instance_name` VARCHAR(255) DEFAULT '' AFTER `service_name`,
  ADD KEY `idx_k8s_release_service_instance` (`service_instance_id`);
