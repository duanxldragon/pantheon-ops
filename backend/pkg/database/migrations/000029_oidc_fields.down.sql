-- Rollback of 000029_oidc_fields.up.sql: remove the tenant-scoped OIDC unique
-- key, the auth_type index and the four OIDC columns from system_user.
--
-- Deliberate asymmetry with `up`: the rollback returns the database to the
-- versioned-chain baseline (no OIDC schema), not to the orphaned
-- add_oidc_fields.sql shape with its globally unique oidc_subject index —
-- that index is exactly the cross-tenant leak this migration removed, and
-- recreating it could fail outright on data where two tenants share a subject.
-- Guarded per column/index, mirroring the up migration, so a partial state
-- rolls back gracefully.
SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user');

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'uk_system_user_tenant_oidc_subject');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_user` DROP INDEX `uk_system_user_tenant_oidc_subject`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_auth_type');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_user` DROP INDEX `idx_system_user_auth_type`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_oidc_subject');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_user` DROP INDEX `idx_system_user_oidc_subject`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'auth_type');
SET @s := IF(@t > 0 AND @c > 0,
  'ALTER TABLE `system_user` DROP COLUMN `auth_type`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_last_sync');
SET @s := IF(@t > 0 AND @c > 0,
  'ALTER TABLE `system_user` DROP COLUMN `oidc_last_sync`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_provider');
SET @s := IF(@t > 0 AND @c > 0,
  'ALTER TABLE `system_user` DROP COLUMN `oidc_provider`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_subject');
SET @s := IF(@t > 0 AND @c > 0,
  'ALTER TABLE `system_user` DROP COLUMN `oidc_subject`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
