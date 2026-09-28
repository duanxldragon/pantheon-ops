SET @s := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='system_log_login' AND index_name='idx_system_log_login_tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_log_login` DROP INDEX `idx_system_log_login_tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
SET @s := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_log_login' AND column_name='tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_log_login` DROP COLUMN `tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
SET @s := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='system_auth_security_event' AND index_name='idx_system_auth_security_event_tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_auth_security_event` DROP INDEX `idx_system_auth_security_event_tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
SET @s := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_auth_security_event' AND column_name='tenant_id');
SET @q := IF(@s>0,'ALTER TABLE `system_auth_security_event` DROP COLUMN `tenant_id`','SELECT 1'); PREPARE x FROM @q; EXECUTE x; DEALLOCATE PREPARE x;
