-- Tenant scope for authentication login logs and security events.
SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_log_login');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_log_login' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c=0,'ALTER TABLE `system_log_login` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`, ADD INDEX `idx_system_log_login_tenant_id` (`tenant_id`)','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;
SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_auth_security_event');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_auth_security_event' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c=0,'ALTER TABLE `system_auth_security_event` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`, ADD INDEX `idx_system_auth_security_event_tenant_id` (`tenant_id`)','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;
