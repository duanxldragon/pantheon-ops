CREATE TABLE IF NOT EXISTS `biz_k8s_namespace_binding` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `cluster_id` BIGINT UNSIGNED NOT NULL,
  `namespace` VARCHAR(255) NOT NULL,
  `business_scope_id` BIGINT UNSIGNED NOT NULL,
  `environment` VARCHAR(32) NOT NULL,
  `allowed_actions` TEXT,
  `created_by` VARCHAR(64) DEFAULT '',
  `updated_by` VARCHAR(64) DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_k8s_namespace_binding` (`cluster_id`, `namespace`),
  KEY `idx_k8s_namespace_binding_scope` (`business_scope_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
