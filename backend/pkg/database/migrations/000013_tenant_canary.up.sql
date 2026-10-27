-- Tenant canary slice (shared-schema MVP, contract TENANT_CONTRACT_V1)
-- Scope: dict resources only (see docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md Phase 1).
-- Behavior under compat (default): tenant_id=0 everywhere — no behavior change.
-- Guarded via information_schema (same compat pattern as 000008/000010/000011)
-- so it is safe on bootstrapped/partial schemas where dict tables may not exist
-- (e.g. marker-seeded upgrade fixtures that never ran 000001).

-- ============================================================================
-- Tenant master data (contract §2.1)
-- ============================================================================

CREATE TABLE IF NOT EXISTS `tenants` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code` VARCHAR(64) NOT NULL,
  `name` VARCHAR(128) NOT NULL,
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `plan` VARCHAR(32) NOT NULL DEFAULT '',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  `deleted_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `idx_tenants_code` (`code`),
  INDEX `idx_tenants_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Global placeholder: owns the compat-mode population (id=0, never login-able).
-- Replayed upgrade windows can carry a dev-seeded `__global__` row whose id is
-- NOT 0 (auto-increment assignment); inserting id=0 then collides on the
-- unique `code` index (1062). Normalize first: if the id=0 slot is free but
-- the code is taken, adopt that row onto id=0; only then insert-if-missing
-- on either identity. (Guarded via PREPARE, same pattern as 000014+.)
SET @placeholder_id0 := (SELECT COUNT(*) FROM `tenants` WHERE `id` = 0);
SET @placeholder_code := (SELECT COUNT(*) FROM `tenants` WHERE `code` = '__global__');
SET @adopt_placeholder_stmt := IF(
  @placeholder_id0 = 0 AND @placeholder_code > 0,
  'UPDATE `tenants` SET `id` = 0 WHERE `code` = ''__global__''',
  'SELECT 1'
);
PREPARE adopt_placeholder_stmt FROM @adopt_placeholder_stmt;
EXECUTE adopt_placeholder_stmt;
DEALLOCATE PREPARE adopt_placeholder_stmt;

INSERT INTO `tenants` (`id`, `code`, `name`, `status`)
SELECT 0, '__global__', 'Platform Global', 'archived'
WHERE NOT EXISTS (SELECT 1 FROM `tenants` WHERE `id` = 0 OR `code` = '__global__');
ALTER TABLE `tenants` AUTO_INCREMENT = 1;

CREATE TABLE IF NOT EXISTS `tenant_memberships` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id` BIGINT UNSIGNED NOT NULL,
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role` VARCHAR(32) NOT NULL DEFAULT 'member',
  `status` VARCHAR(16) NOT NULL DEFAULT 'active',
  `created_at` DATETIME(3) DEFAULT NULL,
  `updated_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_tenant_membership` (`tenant_id`, `user_id`),
  INDEX `idx_tenant_memberships_user` (`user_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ============================================================================
-- Canary resource scoping: dict tables get tenant_id. Guarded so partial
-- schemas (dict tables absent) skip gracefully instead of failing the upgrade.
-- ============================================================================

-- ---- system_dict_type ----
SET @dict_type_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
);
SET @dict_type_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND column_name = 'tenant_id'
);
SET @dict_type_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND index_name = 'idx_system_dict_type_dict_code'
);
SET @dict_type_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_type'
    AND index_name = 'idx_dict_type_tenant_code'
);

SET @dict_type_add_col_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_tenant_col = 0,
  'ALTER TABLE `system_dict_type` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE dict_type_add_col_stmt FROM @dict_type_add_col_stmt;
EXECUTE dict_type_add_col_stmt;
DEALLOCATE PREPARE dict_type_add_col_stmt;

SET @dict_type_drop_old_uk_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_old_uk > 0 AND @dict_type_new_uk = 0,
  'ALTER TABLE `system_dict_type` DROP INDEX `idx_system_dict_type_dict_code`',
  'SELECT 1'
);
PREPARE dict_type_drop_old_uk_stmt FROM @dict_type_drop_old_uk_stmt;
EXECUTE dict_type_drop_old_uk_stmt;
DEALLOCATE PREPARE dict_type_drop_old_uk_stmt;

SET @dict_type_add_new_uk_stmt := IF(
  @dict_type_table_exists > 0 AND @dict_type_new_uk = 0,
  'ALTER TABLE `system_dict_type` ADD UNIQUE INDEX `idx_dict_type_tenant_code` (`tenant_id`, `dict_code`)',
  'SELECT 1'
);
PREPARE dict_type_add_new_uk_stmt FROM @dict_type_add_new_uk_stmt;
EXECUTE dict_type_add_new_uk_stmt;
DEALLOCATE PREPARE dict_type_add_new_uk_stmt;

-- ---- system_dict_item ----
SET @dict_item_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
);
SET @dict_item_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
    AND column_name = 'tenant_id'
);
SET @dict_item_tenant_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dict_item'
    AND index_name = 'idx_dict_item_tenant_code_value'
);

SET @dict_item_add_col_stmt := IF(
  @dict_item_table_exists > 0 AND @dict_item_tenant_col = 0,
  'ALTER TABLE `system_dict_item` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE dict_item_add_col_stmt FROM @dict_item_add_col_stmt;
EXECUTE dict_item_add_col_stmt;
DEALLOCATE PREPARE dict_item_add_col_stmt;

SET @dict_item_add_idx_stmt := IF(
  @dict_item_table_exists > 0 AND @dict_item_tenant_idx = 0 AND @dict_item_tenant_col > 0,
  'ALTER TABLE `system_dict_item` ADD INDEX `idx_dict_item_tenant_code_value` (`tenant_id`, `dict_code`, `item_value`)',
  'SELECT 1'
);
PREPARE dict_item_add_idx_stmt FROM @dict_item_add_idx_stmt;
EXECUTE dict_item_add_idx_stmt;
DEALLOCATE PREPARE dict_item_add_idx_stmt;
