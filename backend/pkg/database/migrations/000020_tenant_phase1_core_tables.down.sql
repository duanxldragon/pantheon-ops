-- Rollback: Tenant Phase 1 Core Tables Tenant Isolation
-- Remove tenant_id columns and restore original unique constraints

-- ============================================================================
-- Phase 1.7: I18n Tables (reverse order)
-- ============================================================================

SET @i18n_locale_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n_locale'
);

SET @i18n_locale_drop_col_stmt := IF(
  @i18n_locale_table_exists > 0,
  'ALTER TABLE `system_i18n_locale` DROP COLUMN `tenant_id`',
  'SELECT "system_i18n_locale table does not exist, skipping"'
);
PREPARE i18n_locale_drop_col_stmt FROM @i18n_locale_drop_col_stmt;
EXECUTE i18n_locale_drop_col_stmt;
DEALLOCATE PREPARE i18n_locale_drop_col_stmt;

-- ============================================================================
-- Phase 1.6: Data Scope Tables
-- ============================================================================

SET @data_scope_policy_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_role_data_scope_policy'
);

SET @data_scope_policy_drop_col_stmt := IF(
  @data_scope_policy_table_exists > 0,
  'ALTER TABLE `permission_role_data_scope_policy` DROP COLUMN `tenant_id`',
  'SELECT "permission_role_data_scope_policy table does not exist, skipping"'
);
PREPARE data_scope_policy_drop_col_stmt FROM @data_scope_policy_drop_col_stmt;
EXECUTE data_scope_policy_drop_col_stmt;
DEALLOCATE PREPARE data_scope_policy_drop_col_stmt;

ALTER TABLE `system_role_data_scope`
  DROP INDEX `idx_role_data_scope_tenant_role`,
  DROP COLUMN `tenant_id`;

-- ============================================================================
-- Phase 1.5: Hierarchy Closure Tables
-- ============================================================================

SET @dept_closures_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept_closures'
);

SET @dept_closures_drop_col_stmt := IF(
  @dept_closures_table_exists > 0,
  'ALTER TABLE `system_dept_closures` DROP COLUMN `tenant_id`',
  'SELECT "system_dept_closures table does not exist, skipping"'
);
PREPARE dept_closures_drop_col_stmt FROM @dept_closures_drop_col_stmt;
EXECUTE dept_closures_drop_col_stmt;
DEALLOCATE PREPARE dept_closures_drop_col_stmt;

SET @dept_ancestors_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept_ancestors'
);

SET @dept_ancestors_drop_col_stmt := IF(
  @dept_ancestors_table_exists > 0,
  'ALTER TABLE `system_dept_ancestors` DROP COLUMN `tenant_id`',
  'SELECT "system_dept_ancestors table does not exist, skipping"'
);
PREPARE dept_ancestors_drop_col_stmt FROM @dept_ancestors_drop_col_stmt;
EXECUTE dept_ancestors_drop_col_stmt;
DEALLOCATE PREPARE dept_ancestors_drop_col_stmt;

-- ============================================================================
-- Phase 1.4: Configuration Tables
-- ============================================================================

-- system_setting rollback: guarded because the tenant_id column and composite
-- key may have been applied earlier by 000014_tenant_settings.
SET @setting_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
);
SET @setting_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'uk_system_setting_tenant_key'
);
SET @setting_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'idx_system_setting_setting_key'
);
SET @setting_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND column_name = 'tenant_id'
);

SET @setting_drop_new_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_new_uk > 0 AND @setting_old_uk = 0,
  'ALTER TABLE `system_setting` DROP INDEX `uk_system_setting_tenant_key`',
  'SELECT 1'
);
PREPARE setting_drop_new_uk_stmt FROM @setting_drop_new_uk_stmt;
EXECUTE setting_drop_new_uk_stmt;
DEALLOCATE PREPARE setting_drop_new_uk_stmt;

SET @setting_add_old_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_old_uk = 0,
  'ALTER TABLE `system_setting` ADD UNIQUE INDEX `idx_system_setting_setting_key` (`setting_key`)',
  'SELECT 1'
);
PREPARE setting_add_old_uk_stmt FROM @setting_add_old_uk_stmt;
EXECUTE setting_add_old_uk_stmt;
DEALLOCATE PREPARE setting_add_old_uk_stmt;

SET @setting_drop_col_stmt := IF(
  @setting_table_exists > 0 AND @setting_tenant_col > 0,
  'ALTER TABLE `system_setting` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE setting_drop_col_stmt FROM @setting_drop_col_stmt;
EXECUTE setting_drop_col_stmt;
DEALLOCATE PREPARE setting_drop_col_stmt;

-- ============================================================================
-- Phase 1.3: Relationship Tables
-- ============================================================================

SET @user_dept_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_dept'
);

SET @user_dept_drop_col_stmt := IF(
  @user_dept_table_exists > 0,
  'ALTER TABLE `system_user_dept` DROP COLUMN `tenant_id`',
  'SELECT "system_user_dept table does not exist, skipping"'
);
PREPARE user_dept_drop_col_stmt FROM @user_dept_drop_col_stmt;
EXECUTE user_dept_drop_col_stmt;
DEALLOCATE PREPARE user_dept_drop_col_stmt;

ALTER TABLE `system_user_role`
  DROP INDEX `idx_system_user_role_tenant_user`,
  DROP PRIMARY KEY,
  ADD PRIMARY KEY (`user_id`, `role_id`),
  DROP COLUMN `tenant_id`;

ALTER TABLE `system_role_permission`
  DROP INDEX `uk_role_permission_tenant`,
  ADD UNIQUE INDEX `idx_role_permission_unique` (`role_id`, `permission_key`),
  DROP COLUMN `tenant_id`;

ALTER TABLE `system_role_menu`
  DROP PRIMARY KEY,
  ADD PRIMARY KEY (`role_id`, `menu_id`),
  DROP COLUMN `tenant_id`;

-- ============================================================================
-- Phase 1.2: Organization Structure Tables
-- ============================================================================

-- system_post / system_dept rollback: guarded because the dept_code column and
-- its unique index were removed by 000005 under the runtime schema contract, so
-- the legacy (tenant_id, dept_code) / dept_code-only indexes may not exist.
SET @post_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
);
SET @post_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND index_name = 'uk_system_post_tenant_code'
);
SET @post_legacy_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND index_name = 'idx_system_post_post_code'
);
SET @post_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND column_name = 'tenant_id'
);

SET @post_drop_new_uk_stmt := IF(
  @post_table_exists > 0 AND @post_old_uk > 0,
  'ALTER TABLE `system_post` DROP INDEX `uk_system_post_tenant_code`',
  'SELECT 1'
);
PREPARE post_drop_new_uk_stmt FROM @post_drop_new_uk_stmt;
EXECUTE post_drop_new_uk_stmt;
DEALLOCATE PREPARE post_drop_new_uk_stmt;

SET @post_add_legacy_uk_stmt := IF(
  @post_table_exists > 0 AND @post_legacy_uk = 0,
  'ALTER TABLE `system_post` ADD UNIQUE INDEX `idx_system_post_post_code` (`post_code`)',
  'SELECT 1'
);
PREPARE post_add_legacy_uk_stmt FROM @post_add_legacy_uk_stmt;
EXECUTE post_add_legacy_uk_stmt;
DEALLOCATE PREPARE post_add_legacy_uk_stmt;

SET @post_drop_col_stmt := IF(
  @post_table_exists > 0 AND @post_tenant_col > 0,
  'ALTER TABLE `system_post` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE post_drop_col_stmt FROM @post_drop_col_stmt;
EXECUTE post_drop_col_stmt;
DEALLOCATE PREPARE post_drop_col_stmt;

SET @dept_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
);
SET @dept_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'uk_system_dept_tenant_code'
);
SET @dept_legacy_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'idx_system_dept_dept_code'
);
SET @dept_code_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND column_name = 'dept_code'
);
SET @dept_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND column_name = 'tenant_id'
);

SET @dept_drop_new_uk_stmt := IF(
  @dept_table_exists > 0 AND @dept_new_uk > 0,
  'ALTER TABLE `system_dept` DROP INDEX `uk_system_dept_tenant_code`',
  'SELECT 1'
);
PREPARE dept_drop_new_uk_stmt FROM @dept_drop_new_uk_stmt;
EXECUTE dept_drop_new_uk_stmt;
DEALLOCATE PREPARE dept_drop_new_uk_stmt;

-- Only restorable while the legacy dept_code column exists; after 000005 it does not.
SET @dept_add_legacy_uk_stmt := IF(
  @dept_table_exists > 0 AND @dept_code_col > 0 AND @dept_legacy_uk = 0,
  'ALTER TABLE `system_dept` ADD UNIQUE INDEX `idx_system_dept_dept_code` (`dept_code`)',
  'SELECT 1'
);
PREPARE dept_add_legacy_uk_stmt FROM @dept_add_legacy_uk_stmt;
EXECUTE dept_add_legacy_uk_stmt;
DEALLOCATE PREPARE dept_add_legacy_uk_stmt;

SET @dept_drop_col_stmt := IF(
  @dept_table_exists > 0 AND @dept_tenant_col > 0,
  'ALTER TABLE `system_dept` DROP COLUMN `tenant_id`',
  'SELECT 1'
);
PREPARE dept_drop_col_stmt FROM @dept_drop_col_stmt;
EXECUTE dept_drop_col_stmt;
DEALLOCATE PREPARE dept_drop_col_stmt;

-- ============================================================================
-- Phase 1.1: Core Authentication Tables
-- ============================================================================

SET @permission_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_permission'
);

SET @permission_drop_col_stmt := IF(
  @permission_table_exists > 0,
  'ALTER TABLE `system_permission` DROP COLUMN `tenant_id`',
  'SELECT "system_permission table does not exist, skipping"'
);
PREPARE permission_drop_col_stmt FROM @permission_drop_col_stmt;
EXECUTE permission_drop_col_stmt;
DEALLOCATE PREPARE permission_drop_col_stmt;

ALTER TABLE `system_menu`
  DROP INDEX `idx_system_menu_tenant_parent`,
  DROP COLUMN `tenant_id`;

ALTER TABLE `system_role`
  DROP INDEX `uk_system_role_tenant_key`,
  ADD UNIQUE INDEX `idx_system_role_role_key` (`role_key`),
  DROP COLUMN `tenant_id`;

ALTER TABLE `system_user`
  DROP INDEX `uk_system_user_tenant_username`,
  ADD UNIQUE INDEX `idx_system_user_username` (`username`),
  DROP COLUMN `tenant_id`;
