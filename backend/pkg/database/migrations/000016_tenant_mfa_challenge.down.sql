-- Rollback of 000016 up (guarded to tolerate the missing-table case).
SET @s := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='system_auth_mfa_challenge' AND index_name='idx_system_auth_mfa_challenge_tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_auth_mfa_challenge` DROP INDEX `idx_system_auth_mfa_challenge_tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
SET @s := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_auth_mfa_challenge' AND column_name='tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_auth_mfa_challenge` DROP COLUMN `tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
