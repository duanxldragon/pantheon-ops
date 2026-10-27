-- Tenant core-tables isolation REPAIR (contract TENANT_CONTRACT_V1 §2.3/§3.3)
--
-- WHY THIS FILE EXISTS
-- --------------------
-- 000020_tenant_phase1_core_tables intended to add tenant_id to 16 core tables
-- and to widen each table's natural key to (tenant_id, ...). It never took
-- effect: pkg/database/migrate.go stamps any database that satisfies
-- currentRuntimeSchemaMarkers straight to `currentRuntimeSchemaVersion`, which
-- was 20 — the same version 000020 occupies. A "looks current" database is
-- therefore recorded as 20 and 000020 is skipped forever, even though none of
-- its DDL ran. Verified 2026-10-13: schema_migrations reported version 23 while
-- 13 of 000020's 14 existing targets had no tenant_id column at all.
--
-- This migration re-applies that DDL, guarded the same way (information_schema
-- + PREPARE, idempotent on re-run) so it is safe on:
--   * a database where 000020 was skipped   -> every guard fires, columns added
--   * a database where 000020 did run       -> every guard is a no-op
--   * a fresh database built by AutoMigrate -> every guard is a no-op
-- The accompanying migrate.go change raises currentRuntimeSchemaVersion to 24
-- and adds a tenant_id marker, so a database missing these columns no longer
-- counts as "current" and this file actually executes.
--
-- SEMANTICS
-- ---------
-- tenant_id defaults to 0 = the platform/global population (contract §2.3), so
-- every existing row keeps its current meaning and compat mode is unchanged.
--
-- system_user.username becomes unique PER TENANT, not globally, so two tenants
-- may each own an `admin`. Login therefore resolves the subject by
-- (tenant_id, username) — see pkg/tenant + modules/auth/login. Existing rows
-- are all tenant_id = 0, so the global uniqueness that held before still holds
-- inside tenant 0 and no existing data can collide.
--
-- Tables 000020 also targeted but which do not exist in this schema are
-- deliberately skipped rather than created: system_permission, system_user_dept,
-- system_dept_ancestors, system_dept_closures, system_i18n_locale. Creating them
-- here would invent schema the runtime does not use.

-- ============================================================================
-- system_user: tenant_id + unique (tenant_id, username)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_username');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_user` DROP INDEX `idx_system_user_username`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'uk_system_user_tenant_username');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_user` ADD UNIQUE INDEX `uk_system_user_tenant_username` (`tenant_id`, `username`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role: tenant_id + unique (tenant_id, role_key)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_role` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND index_name = 'idx_system_role_role_key');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_role` DROP INDEX `idx_system_role_role_key`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND index_name = 'uk_system_role_tenant_key');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_role` ADD UNIQUE INDEX `uk_system_role_tenant_key` (`tenant_id`, `role_key`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_dept: tenant_id
-- 000020 also wanted unique (tenant_id, dept_code); this schema has no
-- dept_code column, so no natural key is imposed here.
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_dept');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_dept' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_dept` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_post: tenant_id + unique (tenant_id, post_code)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_post');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_post` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND index_name = 'idx_system_post_post_code');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_post` DROP INDEX `idx_system_post_post_code`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND index_name = 'uk_system_post_tenant_code');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_post` ADD UNIQUE INDEX `uk_system_post_tenant_code` (`tenant_id`, `post_code`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_user_role: tenant_id + primary key (tenant_id, user_id, role_id)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user_role` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @pk := (SELECT COUNT(*) FROM information_schema.statistics
            WHERE table_schema = DATABASE() AND table_name = 'system_user_role'
              AND index_name = 'PRIMARY' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @pk = 0,
  'ALTER TABLE `system_user_role` DROP PRIMARY KEY, ADD PRIMARY KEY (`tenant_id`, `user_id`, `role_id`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role' AND index_name = 'idx_system_user_role_tenant_user');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_user_role` ADD INDEX `idx_system_user_role_tenant_user` (`tenant_id`, `user_id`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_menu: tenant_id + primary key (tenant_id, role_id, menu_id)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_menu');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_menu' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_role_menu` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @pk := (SELECT COUNT(*) FROM information_schema.statistics
            WHERE table_schema = DATABASE() AND table_name = 'system_role_menu'
              AND index_name = 'PRIMARY' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @pk = 0,
  'ALTER TABLE `system_role_menu` DROP PRIMARY KEY, ADD PRIMARY KEY (`tenant_id`, `role_id`, `menu_id`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_permission: tenant_id + unique (tenant_id, role_id, permission_key)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_role_permission` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND index_name = 'idx_role_permission_unique');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_role_permission` DROP INDEX `idx_role_permission_unique`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND index_name = 'uk_role_permission_tenant');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_role_permission` ADD UNIQUE INDEX `uk_role_permission_tenant` (`tenant_id`, `role_id`, `permission_key`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_data_scope: tenant_id + unique (tenant_id, role_key) + index
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_role_data_scope` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND index_name = 'idx_system_role_data_scope_role_key');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_role_data_scope` DROP INDEX `idx_system_role_data_scope_role_key`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND index_name = 'uk_role_data_scope_tenant_role');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_role_data_scope` ADD UNIQUE INDEX `uk_role_data_scope_tenant_role` (`tenant_id`, `role_key`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- permission_role_data_scope_policy: tenant_id + unique (tenant_id, role_key)
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `permission_role_data_scope_policy` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND index_name = 'idx_permission_role_data_scope_policy_role_key');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `permission_role_data_scope_policy` DROP INDEX `idx_permission_role_data_scope_policy_role_key`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND index_name = 'uk_permission_role_data_scope_policy_tenant');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `permission_role_data_scope_policy` ADD UNIQUE INDEX `uk_permission_role_data_scope_policy_tenant` (`tenant_id`, `role_key`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
