-- Tenant scope for MFA challenges (queue-4 slice 3, migration 000016).
-- Guarded via information_schema + PREPARE (same compat pattern as
-- 000008/000010/000011/000013/000015) so it is safe on bootstrapped/partial
-- schemas where `system_auth_mfa_challenge` may not exist yet — e.g. the
-- marker-seeded upgrade fixtures in pkg/database/migrate_test.go, or a
-- deployment whose MFA module was never bootstrapped. Unguarded ALTERs here
-- broke TestRunMigrationsAppliesLatestCompat* (2026-09-13 finding).
SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_auth_mfa_challenge');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_auth_mfa_challenge' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c=0,'ALTER TABLE `system_auth_mfa_challenge` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `user_id`, ADD INDEX `idx_system_auth_mfa_challenge_tenant_id` (`tenant_id`)','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;
