-- Rollback of 000017_tenant_session_oper_logs.up.sql: drop the tenant columns
-- (MySQL drops the column's index with it) added to system_user_session and
-- system_log_oper. Guarded so partial schemas skip gracefully, mirroring the
-- up migration.
SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_user_session');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_user_session' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c>0,'ALTER TABLE `system_user_session` DROP COLUMN `tenant_id`','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;

SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=DATABASE() AND table_name='system_log_oper');
SET @c := (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='system_log_oper' AND column_name='tenant_id');
SET @s := IF(@t>0 AND @c>0,'ALTER TABLE `system_log_oper` DROP COLUMN `tenant_id`','SELECT 1');
PREPARE x FROM @s; EXECUTE x; DEALLOCATE PREPARE x;
