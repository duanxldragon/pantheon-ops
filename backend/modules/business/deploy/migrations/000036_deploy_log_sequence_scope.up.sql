-- Sequence allocation is serialized per task; retain all existing output and IDs.
-- Correct the legacy globally unique sequence index with one atomic ALTER.
SET @deploy_log_table_exists = (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'biz_deploy_task_log_chunk');
SET @deploy_log_sequence_columns = (SELECT GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX SEPARATOR ',') FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'biz_deploy_task_log_chunk' AND INDEX_NAME = 'uk_deploy_task_log_chunk_seq');
SET @deploy_log_sequence_ddl = IF(@deploy_log_table_exists = 0 OR @deploy_log_sequence_columns = 'task_id,sequence', 'SELECT 1', IF(@deploy_log_sequence_columns = 'sequence', 'ALTER TABLE biz_deploy_task_log_chunk DROP INDEX uk_deploy_task_log_chunk_seq, ADD UNIQUE INDEX uk_deploy_task_log_chunk_seq (task_id, sequence)', 'ALTER TABLE biz_deploy_task_log_chunk ADD UNIQUE INDEX uk_deploy_task_log_chunk_seq (task_id, sequence)'));
PREPARE deploy_log_sequence_stmt FROM @deploy_log_sequence_ddl;
EXECUTE deploy_log_sequence_stmt;
DEALLOCATE PREPARE deploy_log_sequence_stmt;
