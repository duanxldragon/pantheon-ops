-- Persist the step attempt at log-write time. Historical rows remain unattributed (0).
-- Additive and replay-safe; does not reconstruct history or replace retained output.
SET @has_log_table := (SELECT COUNT(*) FROM information_schema.tables
 WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task_log_chunk');
SET @has_log_attempt := (SELECT COUNT(*) FROM information_schema.columns
 WHERE table_schema = DATABASE() AND table_name = 'biz_deploy_task_log_chunk' AND column_name = 'attempt_no');
SET @log_attempt_ddl := IF(@has_log_table > 0 AND @has_log_attempt = 0,
 'ALTER TABLE `biz_deploy_task_log_chunk` ADD COLUMN `attempt_no` INT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE log_attempt_stmt FROM @log_attempt_ddl;
EXECUTE log_attempt_stmt;
DEALLOCATE PREPARE log_attempt_stmt;
