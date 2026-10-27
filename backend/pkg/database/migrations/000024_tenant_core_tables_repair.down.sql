-- Rollback for 000024_tenant_core_tables_repair.
--
-- Restores the pre-tenant natural keys and drops the tenant_id columns added by
-- the up migration. Guarded the same way (information_schema + PREPARE) so it is
-- idempotent.
--
-- CAVEAT: restoring the ORIGINAL unique keys re-imposes global uniqueness.
-- If two tenants now hold the same username / role_key / post_code, the
-- corresponding ADD UNIQUE INDEX will fail with a duplicate-key error and the
-- rollback stops. That is intentional — silently collapsing two tenants' rows
-- would be worse. Resolve the duplicates before rolling back.

-- ============================================================================
-- system_user
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'uk_system_user_tenant_username');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_user` DROP INDEX `uk_system_user_tenant_username`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_username');
SET @s := IF(@t > 0 AND @i = 0, 'ALTER TABLE `system_user` ADD UNIQUE INDEX `idx_system_user_username` (`username`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_user` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND index_name = 'uk_system_role_tenant_key');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_role` DROP INDEX `uk_system_role_tenant_key`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND index_name = 'idx_system_role_role_key');
SET @s := IF(@t > 0 AND @i = 0, 'ALTER TABLE `system_role` ADD UNIQUE INDEX `idx_system_role_role_key` (`role_key`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_role` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_dept
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_dept');
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_dept' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_dept` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_post
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_post');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND index_name = 'uk_system_post_tenant_code');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_post` DROP INDEX `uk_system_post_tenant_code`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND index_name = 'idx_system_post_post_code');
SET @s := IF(@t > 0 AND @i = 0, 'ALTER TABLE `system_post` ADD UNIQUE INDEX `idx_system_post_post_code` (`post_code`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_post' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_post` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_user_role
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role');
SET @pk := (SELECT COUNT(*) FROM information_schema.statistics
            WHERE table_schema = DATABASE() AND table_name = 'system_user_role'
              AND index_name = 'PRIMARY' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @pk > 0,
  'ALTER TABLE `system_user_role` DROP PRIMARY KEY, ADD PRIMARY KEY (`user_id`, `role_id`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role' AND index_name = 'idx_system_user_role_tenant_user');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_user_role` DROP INDEX `idx_system_user_role_tenant_user`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user_role' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_user_role` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_menu
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_menu');
SET @pk := (SELECT COUNT(*) FROM information_schema.statistics
            WHERE table_schema = DATABASE() AND table_name = 'system_role_menu'
              AND index_name = 'PRIMARY' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @pk > 0,
  'ALTER TABLE `system_role_menu` DROP PRIMARY KEY, ADD PRIMARY KEY (`role_id`, `menu_id`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_menu' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_role_menu` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_permission
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND index_name = 'uk_role_permission_tenant');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_role_permission` DROP INDEX `uk_role_permission_tenant`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND index_name = 'idx_role_permission_unique');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_role_permission` ADD UNIQUE INDEX `idx_role_permission_unique` (`role_id`, `permission_key`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_permission' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_role_permission` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- system_role_data_scope
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND index_name = 'uk_role_data_scope_tenant_role');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `system_role_data_scope` DROP INDEX `uk_role_data_scope_tenant_role`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND index_name = 'idx_system_role_data_scope_role_key');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_role_data_scope` ADD UNIQUE INDEX `idx_system_role_data_scope_role_key` (`role_key`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `system_role_data_scope` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- permission_role_data_scope_policy
-- ============================================================================
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND index_name = 'uk_permission_role_data_scope_policy_tenant');
SET @s := IF(@t > 0 AND @i > 0, 'ALTER TABLE `permission_role_data_scope_policy` DROP INDEX `uk_permission_role_data_scope_policy_tenant`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND index_name = 'idx_permission_role_data_scope_policy_role_key');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `permission_role_data_scope_policy` ADD UNIQUE INDEX `idx_permission_role_data_scope_policy_role_key` (`role_key`)', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy' AND column_name = 'tenant_id');
SET @s := IF(@t > 0 AND @c > 0, 'ALTER TABLE `permission_role_data_scope_policy` DROP COLUMN `tenant_id`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
