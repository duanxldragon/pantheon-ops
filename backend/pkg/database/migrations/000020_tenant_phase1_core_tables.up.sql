-- Tenant Phase 1: Core Tables Tenant Isolation
-- Add tenant_id to 16 core tables for multi-tenant data isolation
-- Contract: TENANT_CONTRACT_V1 §3.1 - Shared Schema MVP
-- Behavior: tenant_id=0 (compat mode) preserves existing behavior

-- ============================================================================
-- Phase 1.1: Core Authentication Tables
-- ============================================================================

-- Add tenant_id to system_user (guarded: the marker-driven bootstrap replay
-- window models a minimal current schema that may not include this table —
-- see 000012's header comment for the same replay hazard)
SET @user_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
);
SET @user_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND column_name = 'tenant_id'
);
SET @user_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'idx_system_user_username'
);
SET @user_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'uk_system_user_tenant_username'
);

SET @user_add_col_stmt := IF(
  @user_table_exists > 0 AND @user_tenant_col = 0,
  'ALTER TABLE `system_user` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE user_add_col_stmt FROM @user_add_col_stmt;
EXECUTE user_add_col_stmt;
DEALLOCATE PREPARE user_add_col_stmt;

-- Change unique constraint from username-only to (tenant_id, username)
SET @user_drop_old_uk_stmt := IF(
  @user_table_exists > 0 AND @user_old_uk > 0,
  'ALTER TABLE `system_user` DROP INDEX `idx_system_user_username`',
  'SELECT 1'
);
PREPARE user_drop_old_uk_stmt FROM @user_drop_old_uk_stmt;
EXECUTE user_drop_old_uk_stmt;
DEALLOCATE PREPARE user_drop_old_uk_stmt;

SET @user_add_new_uk_stmt := IF(
  @user_table_exists > 0 AND @user_new_uk = 0,
  'ALTER TABLE `system_user` ADD UNIQUE INDEX `uk_system_user_tenant_username` (`tenant_id`, `username`)',
  'SELECT 1'
);
PREPARE user_add_new_uk_stmt FROM @user_add_new_uk_stmt;
EXECUTE user_add_new_uk_stmt;
DEALLOCATE PREPARE user_add_new_uk_stmt;

-- Add tenant_id to system_role (guarded: same replay hazard as system_user)
SET @role_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role'
);
SET @role_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role'
    AND column_name = 'tenant_id'
);
SET @role_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role'
    AND index_name = 'idx_system_role_role_key'
);
SET @role_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role'
    AND index_name = 'uk_system_role_tenant_key'
);

SET @role_add_col_stmt := IF(
  @role_table_exists > 0 AND @role_tenant_col = 0,
  'ALTER TABLE `system_role` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE role_add_col_stmt FROM @role_add_col_stmt;
EXECUTE role_add_col_stmt;
DEALLOCATE PREPARE role_add_col_stmt;

-- Change unique constraint from role_key-only to (tenant_id, role_key)
SET @role_drop_old_uk_stmt := IF(
  @role_table_exists > 0 AND @role_old_uk > 0,
  'ALTER TABLE `system_role` DROP INDEX `idx_system_role_role_key`',
  'SELECT 1'
);
PREPARE role_drop_old_uk_stmt FROM @role_drop_old_uk_stmt;
EXECUTE role_drop_old_uk_stmt;
DEALLOCATE PREPARE role_drop_old_uk_stmt;

SET @role_add_new_uk_stmt := IF(
  @role_table_exists > 0 AND @role_new_uk = 0,
  'ALTER TABLE `system_role` ADD UNIQUE INDEX `uk_system_role_tenant_key` (`tenant_id`, `role_key`)',
  'SELECT 1'
);
PREPARE role_add_new_uk_stmt FROM @role_add_new_uk_stmt;
EXECUTE role_add_new_uk_stmt;
DEALLOCATE PREPARE role_add_new_uk_stmt;

-- Add tenant_id to system_menu
ALTER TABLE `system_menu`
  ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`;

-- Add composite index for tenant-scoped menu queries
ALTER TABLE `system_menu`
  ADD INDEX `idx_system_menu_tenant_parent` (`tenant_id`, `parent_id`);

-- Add tenant_id to system_permission (if exists)
SET @permission_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_permission'
);

SET @permission_add_col_stmt := IF(
  @permission_table_exists > 0,
  'ALTER TABLE `system_permission` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT "system_permission table does not exist, skipping"'
);
PREPARE permission_add_col_stmt FROM @permission_add_col_stmt;
EXECUTE permission_add_col_stmt;
DEALLOCATE PREPARE permission_add_col_stmt;

-- ============================================================================
-- Phase 1.2: Organization Structure Tables
-- ============================================================================

-- Add tenant_id to system_dept (guarded: 000005 already dropped the dept_code
-- column and its unique index under the runtime schema contract, so both the
-- column and idx_system_dept_dept_code may not exist here)
SET @dept_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
);
SET @dept_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND column_name = 'tenant_id'
);
SET @dept_code_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND column_name = 'dept_code'
);
SET @dept_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'idx_system_dept_dept_code'
);
SET @dept_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept'
    AND index_name = 'uk_system_dept_tenant_code'
);

SET @dept_add_col_stmt := IF(
  @dept_table_exists > 0 AND @dept_tenant_col = 0,
  'ALTER TABLE `system_dept` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE dept_add_col_stmt FROM @dept_add_col_stmt;
EXECUTE dept_add_col_stmt;
DEALLOCATE PREPARE dept_add_col_stmt;

SET @dept_drop_old_uk_stmt := IF(
  @dept_table_exists > 0 AND @dept_old_uk > 0,
  'ALTER TABLE `system_dept` DROP INDEX `idx_system_dept_dept_code`',
  'SELECT 1'
);
PREPARE dept_drop_old_uk_stmt FROM @dept_drop_old_uk_stmt;
EXECUTE dept_drop_old_uk_stmt;
DEALLOCATE PREPARE dept_drop_old_uk_stmt;

-- Only meaningful while the legacy dept_code column still exists; after 000005
-- the tenant scoping for system_dept relies on tenant_id alone.
SET @dept_add_new_uk_stmt := IF(
  @dept_table_exists > 0 AND @dept_code_col > 0 AND @dept_new_uk = 0,
  'ALTER TABLE `system_dept` ADD UNIQUE INDEX `uk_system_dept_tenant_code` (`tenant_id`, `dept_code`)',
  'SELECT 1'
);
PREPARE dept_add_new_uk_stmt FROM @dept_add_new_uk_stmt;
EXECUTE dept_add_new_uk_stmt;
DEALLOCATE PREPARE dept_add_new_uk_stmt;

-- Add tenant_id to system_post (guarded: same replay hazard as system_user —
-- this table is absent from the minimal marker-driven bootstrap schema)
SET @post_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
);
SET @post_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND column_name = 'tenant_id'
);
SET @post_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND index_name = 'idx_system_post_post_code'
);
SET @post_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_post'
    AND index_name = 'uk_system_post_tenant_code'
);

SET @post_add_col_stmt := IF(
  @post_table_exists > 0 AND @post_tenant_col = 0,
  'ALTER TABLE `system_post` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE post_add_col_stmt FROM @post_add_col_stmt;
EXECUTE post_add_col_stmt;
DEALLOCATE PREPARE post_add_col_stmt;

-- Change unique constraint from post_code-only to (tenant_id, post_code)
SET @post_drop_old_uk_stmt := IF(
  @post_table_exists > 0 AND @post_old_uk > 0,
  'ALTER TABLE `system_post` DROP INDEX `idx_system_post_post_code`',
  'SELECT 1'
);
PREPARE post_drop_old_uk_stmt FROM @post_drop_old_uk_stmt;
EXECUTE post_drop_old_uk_stmt;
DEALLOCATE PREPARE post_drop_old_uk_stmt;

SET @post_add_new_uk_stmt := IF(
  @post_table_exists > 0 AND @post_new_uk = 0,
  'ALTER TABLE `system_post` ADD UNIQUE INDEX `uk_system_post_tenant_code` (`tenant_id`, `post_code`)',
  'SELECT 1'
);
PREPARE post_add_new_uk_stmt FROM @post_add_new_uk_stmt;
EXECUTE post_add_new_uk_stmt;
DEALLOCATE PREPARE post_add_new_uk_stmt;

-- ============================================================================
-- Phase 1.3: Relationship Tables
-- ============================================================================

-- Add tenant_id to system_role_menu (guarded: absent from the minimal
-- marker-driven bootstrap schema, same replay hazard as system_user)
SET @role_menu_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
);
SET @role_menu_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
    AND column_name = 'tenant_id'
);
SET @role_menu_new_pk_col_count := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_menu'
    AND index_name = 'PRIMARY'
    AND column_name = 'tenant_id'
);

SET @role_menu_add_col_stmt := IF(
  @role_menu_table_exists > 0 AND @role_menu_tenant_col = 0,
  'ALTER TABLE `system_role_menu` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT 1'
);
PREPARE role_menu_add_col_stmt FROM @role_menu_add_col_stmt;
EXECUTE role_menu_add_col_stmt;
DEALLOCATE PREPARE role_menu_add_col_stmt;

-- Recreate primary key with tenant_id
SET @role_menu_pk_stmt := IF(
  @role_menu_table_exists > 0 AND @role_menu_new_pk_col_count = 0,
  'ALTER TABLE `system_role_menu` DROP PRIMARY KEY, ADD PRIMARY KEY (`tenant_id`, `role_id`, `menu_id`)',
  'SELECT 1'
);
PREPARE role_menu_pk_stmt FROM @role_menu_pk_stmt;
EXECUTE role_menu_pk_stmt;
DEALLOCATE PREPARE role_menu_pk_stmt;

-- Add tenant_id to system_role_permission (guarded: same replay hazard)
SET @role_permission_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_permission'
);
SET @role_permission_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_permission'
    AND column_name = 'tenant_id'
);
SET @role_permission_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_permission'
    AND index_name = 'idx_role_permission_unique'
);
SET @role_permission_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_permission'
    AND index_name = 'uk_role_permission_tenant'
);

SET @role_permission_add_col_stmt := IF(
  @role_permission_table_exists > 0 AND @role_permission_tenant_col = 0,
  'ALTER TABLE `system_role_permission` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE role_permission_add_col_stmt FROM @role_permission_add_col_stmt;
EXECUTE role_permission_add_col_stmt;
DEALLOCATE PREPARE role_permission_add_col_stmt;

-- Update unique index to include tenant_id
SET @role_permission_drop_old_uk_stmt := IF(
  @role_permission_table_exists > 0 AND @role_permission_old_uk > 0,
  'ALTER TABLE `system_role_permission` DROP INDEX `idx_role_permission_unique`',
  'SELECT 1'
);
PREPARE role_permission_drop_old_uk_stmt FROM @role_permission_drop_old_uk_stmt;
EXECUTE role_permission_drop_old_uk_stmt;
DEALLOCATE PREPARE role_permission_drop_old_uk_stmt;

SET @role_permission_add_new_uk_stmt := IF(
  @role_permission_table_exists > 0 AND @role_permission_new_uk = 0,
  'ALTER TABLE `system_role_permission` ADD UNIQUE INDEX `uk_role_permission_tenant` (`tenant_id`, `role_id`, `permission_key`)',
  'SELECT 1'
);
PREPARE role_permission_add_new_uk_stmt FROM @role_permission_add_new_uk_stmt;
EXECUTE role_permission_add_new_uk_stmt;
DEALLOCATE PREPARE role_permission_add_new_uk_stmt;

-- Add tenant_id to system_user_role (guarded: same replay hazard)
SET @user_role_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_role'
);
SET @user_role_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_role'
    AND column_name = 'tenant_id'
);
SET @user_role_new_pk_col_count := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_role'
    AND index_name = 'PRIMARY'
    AND column_name = 'tenant_id'
);
SET @user_role_tenant_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_role'
    AND index_name = 'idx_system_user_role_tenant_user'
);

SET @user_role_add_col_stmt := IF(
  @user_role_table_exists > 0 AND @user_role_tenant_col = 0,
  'ALTER TABLE `system_user_role` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT 1'
);
PREPARE user_role_add_col_stmt FROM @user_role_add_col_stmt;
EXECUTE user_role_add_col_stmt;
DEALLOCATE PREPARE user_role_add_col_stmt;

-- Recreate primary key with tenant_id
SET @user_role_pk_stmt := IF(
  @user_role_table_exists > 0 AND @user_role_new_pk_col_count = 0,
  'ALTER TABLE `system_user_role` DROP PRIMARY KEY, ADD PRIMARY KEY (`tenant_id`, `user_id`, `role_id`)',
  'SELECT 1'
);
PREPARE user_role_pk_stmt FROM @user_role_pk_stmt;
EXECUTE user_role_pk_stmt;
DEALLOCATE PREPARE user_role_pk_stmt;

SET @user_role_add_idx_stmt := IF(
  @user_role_table_exists > 0 AND @user_role_tenant_idx = 0,
  'ALTER TABLE `system_user_role` ADD INDEX `idx_system_user_role_tenant_user` (`tenant_id`, `user_id`)',
  'SELECT 1'
);
PREPARE user_role_add_idx_stmt FROM @user_role_add_idx_stmt;
EXECUTE user_role_add_idx_stmt;
DEALLOCATE PREPARE user_role_add_idx_stmt;

-- Add tenant_id to system_user_dept (if exists)
SET @user_dept_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_dept'
);

SET @user_dept_add_col_stmt := IF(
  @user_dept_table_exists > 0,
  'ALTER TABLE `system_user_dept` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT "system_user_dept table does not exist, skipping"'
);
PREPARE user_dept_add_col_stmt FROM @user_dept_add_col_stmt;
EXECUTE user_dept_add_col_stmt;
DEALLOCATE PREPARE user_dept_add_col_stmt;

-- ============================================================================
-- Phase 1.4: Configuration Tables
-- ============================================================================

-- Add tenant_id to system_setting (guarded: 000014 already applied the same
-- tenant_id column and composite unique key swap on this table)
SET @setting_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
);
SET @setting_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND column_name = 'tenant_id'
);
SET @setting_old_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'idx_system_setting_setting_key'
);
SET @setting_new_uk := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_setting'
    AND index_name = 'uk_system_setting_tenant_key'
);

SET @setting_add_col_stmt := IF(
  @setting_table_exists > 0 AND @setting_tenant_col = 0,
  'ALTER TABLE `system_setting` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE setting_add_col_stmt FROM @setting_add_col_stmt;
EXECUTE setting_add_col_stmt;
DEALLOCATE PREPARE setting_add_col_stmt;

SET @setting_drop_old_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_old_uk > 0 AND @setting_new_uk = 0,
  'ALTER TABLE `system_setting` DROP INDEX `idx_system_setting_setting_key`',
  'SELECT 1'
);
PREPARE setting_drop_old_uk_stmt FROM @setting_drop_old_uk_stmt;
EXECUTE setting_drop_old_uk_stmt;
DEALLOCATE PREPARE setting_drop_old_uk_stmt;

SET @setting_add_new_uk_stmt := IF(
  @setting_table_exists > 0 AND @setting_new_uk = 0,
  'ALTER TABLE `system_setting` ADD UNIQUE INDEX `uk_system_setting_tenant_key` (`tenant_id`, `setting_key`)',
  'SELECT 1'
);
PREPARE setting_add_new_uk_stmt FROM @setting_add_new_uk_stmt;
EXECUTE setting_add_new_uk_stmt;
DEALLOCATE PREPARE setting_add_new_uk_stmt;

-- ============================================================================
-- Phase 1.5: Hierarchy Closure Tables (if exist)
-- ============================================================================

-- Add tenant_id to dept_ancestors (if exists)
SET @dept_ancestors_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept_ancestors'
);

SET @dept_ancestors_add_col_stmt := IF(
  @dept_ancestors_table_exists > 0,
  'ALTER TABLE `system_dept_ancestors` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT "system_dept_ancestors table does not exist, skipping"'
);
PREPARE dept_ancestors_add_col_stmt FROM @dept_ancestors_add_col_stmt;
EXECUTE dept_ancestors_add_col_stmt;
DEALLOCATE PREPARE dept_ancestors_add_col_stmt;

-- Add tenant_id to dept_closures (if exists)
SET @dept_closures_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_dept_closures'
);

SET @dept_closures_add_col_stmt := IF(
  @dept_closures_table_exists > 0,
  'ALTER TABLE `system_dept_closures` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT "system_dept_closures table does not exist, skipping"'
);
PREPARE dept_closures_add_col_stmt FROM @dept_closures_add_col_stmt;
EXECUTE dept_closures_add_col_stmt;
DEALLOCATE PREPARE dept_closures_add_col_stmt;

-- ============================================================================
-- Phase 1.6: Data Scope Tables
-- ============================================================================

-- Add tenant_id to system_role_data_scope (guarded: absent from the minimal
-- marker-driven bootstrap schema, same replay hazard as system_user).
-- Note: this table keys on `role_key` (see 000001_init_schema.up.sql), not
-- `role_id` — there is no role_id column here.
SET @role_data_scope_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_data_scope'
);
SET @role_data_scope_tenant_col := (
  SELECT COUNT(*)
  FROM information_schema.columns
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_data_scope'
    AND column_name = 'tenant_id'
);
SET @role_data_scope_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_role_data_scope'
    AND index_name = 'idx_role_data_scope_tenant_role'
);

SET @role_data_scope_add_col_stmt := IF(
  @role_data_scope_table_exists > 0 AND @role_data_scope_tenant_col = 0,
  'ALTER TABLE `system_role_data_scope` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1'
);
PREPARE role_data_scope_add_col_stmt FROM @role_data_scope_add_col_stmt;
EXECUTE role_data_scope_add_col_stmt;
DEALLOCATE PREPARE role_data_scope_add_col_stmt;

-- Add composite index for tenant-scoped queries
SET @role_data_scope_add_idx_stmt := IF(
  @role_data_scope_table_exists > 0 AND @role_data_scope_idx = 0,
  'ALTER TABLE `system_role_data_scope` ADD INDEX `idx_role_data_scope_tenant_role` (`tenant_id`, `role_key`)',
  'SELECT 1'
);
PREPARE role_data_scope_add_idx_stmt FROM @role_data_scope_add_idx_stmt;
EXECUTE role_data_scope_add_idx_stmt;
DEALLOCATE PREPARE role_data_scope_add_idx_stmt;

-- Add tenant_id to permission_role_data_scope_policy (if exists)
SET @data_scope_policy_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'permission_role_data_scope_policy'
);

SET @data_scope_policy_add_col_stmt := IF(
  @data_scope_policy_table_exists > 0,
  'ALTER TABLE `permission_role_data_scope_policy` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT "permission_role_data_scope_policy table does not exist, skipping"'
);
PREPARE data_scope_policy_add_col_stmt FROM @data_scope_policy_add_col_stmt;
EXECUTE data_scope_policy_add_col_stmt;
DEALLOCATE PREPARE data_scope_policy_add_col_stmt;

-- ============================================================================
-- Phase 1.7: I18n Tables (if exist)
-- ============================================================================

-- Add tenant_id to i18n_locale (if exists)
SET @i18n_locale_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_i18n_locale'
);

SET @i18n_locale_add_col_stmt := IF(
  @i18n_locale_table_exists > 0,
  'ALTER TABLE `system_i18n_locale` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT "system_i18n_locale table does not exist, skipping"'
);
PREPARE i18n_locale_add_col_stmt FROM @i18n_locale_add_col_stmt;
EXECUTE i18n_locale_add_col_stmt;
DEALLOCATE PREPARE i18n_locale_add_col_stmt;

-- ============================================================================
-- Verification Query
-- ============================================================================
-- Run this to verify all tenant_id columns were added:
-- SELECT table_name, column_name
-- FROM information_schema.columns
-- WHERE table_schema = DATABASE()
--   AND column_name = 'tenant_id'
-- ORDER BY table_name;
