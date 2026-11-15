-- Business-owned environment entity. Deleted environment codes remain reserved.
CREATE TABLE IF NOT EXISTS `biz_business_environment` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `business_scope_id` bigint unsigned NOT NULL,
  `code` varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `env_type` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `status` varchar(16) COLLATE utf8mb4_unicode_ci NOT NULL,
  `remark` text COLLATE utf8mb4_unicode_ci,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `deleted_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_biz_business_environment_business_scope_id` (`business_scope_id`),
  KEY `idx_biz_business_environment_deleted_at` (`deleted_at`),
  UNIQUE KEY `idx_scope_env_code` (`business_scope_id`, `code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
