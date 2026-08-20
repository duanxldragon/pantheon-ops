SET @has_index = (
  SELECT COUNT(*) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task'
    AND index_name = 'idx_deploy_task_credential_ref'
);
SET @sql = IF(@has_index > 0,
  'DROP INDEX idx_deploy_task_credential_ref ON biz_deploy_task',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_column = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task' AND column_name = 'execution_timeout_seconds');
SET @sql = IF(@has_column > 0, 'ALTER TABLE biz_deploy_task DROP COLUMN execution_timeout_seconds', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @has_column = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task' AND column_name = 'ssh_host_fingerprint');
SET @sql = IF(@has_column > 0, 'ALTER TABLE biz_deploy_task DROP COLUMN ssh_host_fingerprint', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @has_column = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task' AND column_name = 'credential_ref_version');
SET @sql = IF(@has_column > 0, 'ALTER TABLE biz_deploy_task DROP COLUMN credential_ref_version', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @has_column = (SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task' AND column_name = 'credential_ref_id');
SET @sql = IF(@has_column > 0, 'ALTER TABLE biz_deploy_task DROP COLUMN credential_ref_id', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
