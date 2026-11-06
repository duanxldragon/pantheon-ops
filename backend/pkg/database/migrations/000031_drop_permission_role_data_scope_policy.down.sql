-- Rebuilds the table structure exactly as it stood at version 30:
-- 000001 (columns + PK) + 000020 (tenant_id) + 000024 (drops the role_key-only
-- unique index and replaces it with the tenant-scoped one).
--
-- DATA IS NOT RESTORED: only the shape comes back; rows dropped by the up
-- migration are gone. This is not a rollback of data.
--
-- The role_key-only unique index deliberately does NOT come back: 000024
-- dropped `idx_permission_role_data_scope_policy_role_key` when it introduced
-- `uk_permission_role_data_scope_policy_tenant`, because a role_key must be
-- unique per tenant, not globally. Recreating it here would make the restored
-- table reject the same role_key in two different tenants — a constraint that
-- did not exist at version 30.
CREATE TABLE IF NOT EXISTS `permission_role_data_scope_policy` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `role_key` VARCHAR(64) NOT NULL,
  `mode` VARCHAR(32) NOT NULL DEFAULT 'all',
  `scope_defs` TEXT,
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_permission_role_data_scope_policy_tenant` (`tenant_id`, `role_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
