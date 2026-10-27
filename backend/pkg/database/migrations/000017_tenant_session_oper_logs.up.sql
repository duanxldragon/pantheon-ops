-- Tenant scope for auth sessions and the operation audit log (queue-4/5
-- follow-up, migration 000017). The Go models (session.SystemUserSession,
-- middleware.SystemLogOper) carry a TenantID field; development databases got
-- the column via module AutoMigrate, but versioned-migrations deployments
-- skipped module AutoMigrate, so INSERTs with tenant_id failed with
-- "Unknown column 'tenant_id' in 'field list'" and every login returned
-- auth.session.create.error (403). This migration adds the missing columns.
--
-- Column + index are added in a single ALTER statement (same shape as
-- 000015's guarded ALTERs): golang-migrate executes each statement on the
-- same pooled connection, and a second guarded statement re-reading
-- information_schema in the same file was not reliable across the split
-- statement execution, so each table is one self-contained guarded ALTER.
SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_user_session');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_user_session' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c=0,'ALTER TABLE `system_user_session` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `user_id`, ADD INDEX `idx_system_user_session_tenant_id` (`tenant_id`)','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;

SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_log_oper');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_log_oper' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c=0,'ALTER TABLE `system_log_oper` ADD COLUMN `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER `id`, ADD INDEX `idx_system_log_oper_tenant` (`tenant_id`)','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;
