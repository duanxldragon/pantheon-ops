-- OIDC/SSO fields on system_user, folded into the versioned chain (migration
-- 000029) from the orphaned add_oidc_fields.sql (removed in the same change).
--
-- WHY THIS FILE EXISTS
-- --------------------
-- The oidc scaffolding (modules/auth/oidc/service.go) reads and writes
-- oidc_subject / oidc_provider / oidc_last_sync / auth_type, but the only DDL
-- delivering those columns sat outside the numbered migrations and was never
-- executed by golang-migrate. Versioned deployments therefore had no OIDC
-- columns at all, and AutoMigrate deployments never saw the scaffolded model.
--
-- The orphan file also declared oidc_subject globally unique. Under the tenant
-- contract (TENANT_CONTRACT_V1 §2.3) per-tenant uniqueness is the rule — two
-- tenants fronted by the same IdP would collide on one subject only when the
-- provider is shared, so the natural key is (tenant_id, oidc_provider,
-- oidc_subject). The guarded DROP/ADD below swaps the indexes; NULL subjects
-- (every non-OIDC row) never collide in MySQL unique indexes, so existing data
-- cannot block the new constraint.
--
-- Guarded information_schema checks + PREPARE, idempotent on re-run, same
-- shape as 000015/000024: every guard is a no-op on a database where the DDL
-- already ran (manual application of the orphan file, or a replay after the
-- baseline rewind) and fires exactly once elsewhere.

SET @t := (SELECT COUNT(*) FROM information_schema.tables
           WHERE table_schema = DATABASE() AND table_name = 'system_user');

-- ============================================================================
-- columns (add_oidc_fields.sql, one guarded ALTER per column so a partially
-- OIDC-enabled database converges instead of failing the whole statement)
-- ============================================================================
SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_subject');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user` ADD COLUMN `oidc_subject` VARCHAR(255) NULL COMMENT ''OIDC sub claim (unique user ID from provider)'' AFTER `id`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_provider');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user` ADD COLUMN `oidc_provider` VARCHAR(100) NULL COMMENT ''OIDC provider identifier'' AFTER `oidc_subject`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'oidc_last_sync');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user` ADD COLUMN `oidc_last_sync` DATETIME NULL COMMENT ''Last time profile was synced from provider'' AFTER `oidc_provider`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @c := (SELECT COUNT(*) FROM information_schema.columns
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'auth_type');
SET @s := IF(@t > 0 AND @c = 0,
  'ALTER TABLE `system_user` ADD COLUMN `auth_type` VARCHAR(20) DEFAULT ''local'' COMMENT ''Authentication type: local or oidc'' AFTER `oidc_last_sync`',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- tenant-scoped unique key: drop the orphan file's global
-- idx_system_user_oidc_subject, add uk_system_user_tenant_oidc_subject
-- ============================================================================
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_oidc_subject');
SET @s := IF(@t > 0 AND @i > 0,
  'ALTER TABLE `system_user` DROP INDEX `idx_system_user_oidc_subject`', 'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

SET @tc := (SELECT COUNT(*) FROM information_schema.columns
            WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'tenant_id');
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'uk_system_user_tenant_oidc_subject');
SET @s := IF(@t > 0 AND @tc > 0 AND @i = 0,
  'ALTER TABLE `system_user` ADD UNIQUE INDEX `uk_system_user_tenant_oidc_subject` (`tenant_id`, `oidc_provider`, `oidc_subject`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- ============================================================================
-- auth_type lookup index
-- ============================================================================
SET @i := (SELECT COUNT(*) FROM information_schema.statistics
           WHERE table_schema = DATABASE() AND table_name = 'system_user' AND index_name = 'idx_system_user_auth_type');
SET @s := IF(@t > 0 AND @i = 0,
  'ALTER TABLE `system_user` ADD INDEX `idx_system_user_auth_type` (`auth_type`)',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
