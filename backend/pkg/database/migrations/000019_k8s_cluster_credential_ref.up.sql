CREATE TABLE IF NOT EXISTS `biz_k8s_cluster_credential_ref` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `cluster_id` BIGINT UNSIGNED NOT NULL,
  `encrypted` TEXT NOT NULL,
  `version` BIGINT UNSIGNED NOT NULL DEFAULT 1,
  `status` VARCHAR(32) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_k8s_cluster_credential_cluster` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

ALTER TABLE `biz_k8s_cluster` ADD COLUMN `kubeconfig_credential_ref_id` BIGINT UNSIGNED NOT NULL DEFAULT 0;
CREATE INDEX `idx_k8s_cluster_credential_ref` ON `biz_k8s_cluster` (`kubeconfig_credential_ref_id`);
