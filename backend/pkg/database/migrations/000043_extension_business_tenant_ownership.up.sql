-- T14: extension business tenant ownership (verified table list, see the T14 summary).
-- Adds tenant_id to the 18 extension tables that migration 000041 did not cover and
-- moves the five natural unique keys to tenant-first variants, so two tenants may use
-- the same name. The T01 baseline list was wrong: it named biz_preflight_control (a
-- test-only synthetic table) and missed biz_application and the obs_alert_* /
-- obs_notification_* tables.
-- Idempotent: every statement is guarded by information_schema, so reruns are no-ops
-- and a fresh install that creates the tables afterwards via AutoMigrate simply skips
-- the missing-table branch. Existing rows keep tenant_id = 0 (compat / platform-owned).
-- Soft-delete semantics are unchanged: the new keys are (tenant_id, name) without
-- deleted_at, exactly like the single-column keys they replace.

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_server')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_server' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_argocd_server ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_repository')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_repository' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_argocd_repository ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_application')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_application' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_argocd_application ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_sync_history')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_sync_history' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_argocd_sync_history ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cicd_webhook_event')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cicd_webhook_event' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_cicd_webhook_event ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_helm_release')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_helm_release' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_helm_release ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_install')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_install' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_install ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_release ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_observability_target')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_observability_target' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_observability_target ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_rule')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_rule' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE obs_alert_rule ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_event')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_event' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE obs_alert_event ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_channel')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_channel' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE obs_notification_channel ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_log')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_log' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE obs_notification_log ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_storage')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_storage' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE filedist_storage ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_file')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_file' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE filedist_file ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_target')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_target' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE filedist_target ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_task')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_task' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE filedist_task ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_application')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_application' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_application ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

-- Tenant-first unique keys. The old single-column keys were created by AutoMigrate
-- without a stable name, so they are found by their column set rather than guessed.

SET @old := (SELECT INDEX_NAME FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_server' AND NON_UNIQUE=0 AND INDEX_NAME<>'PRIMARY'
  GROUP BY INDEX_NAME HAVING COUNT(*)=1 AND MAX(COLUMN_NAME)='name' LIMIT 1);
SET @sql := IF(@old IS NULL, 'SELECT 1', CONCAT('ALTER TABLE biz_argocd_server DROP INDEX `', @old, '`'));
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_server')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_argocd_server' AND INDEX_NAME='uk_argocd_server_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_argocd_server ADD UNIQUE INDEX uk_argocd_server_name (tenant_id, name)');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @old := (SELECT INDEX_NAME FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_storage' AND NON_UNIQUE=0 AND INDEX_NAME<>'PRIMARY'
  GROUP BY INDEX_NAME HAVING COUNT(*)=1 AND MAX(COLUMN_NAME)='name' LIMIT 1);
SET @sql := IF(@old IS NULL, 'SELECT 1', CONCAT('ALTER TABLE filedist_storage DROP INDEX `', @old, '`'));
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_storage')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='filedist_storage' AND INDEX_NAME='uk_filedist_storage_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE filedist_storage ADD UNIQUE INDEX uk_filedist_storage_name (tenant_id, name)');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @old := (SELECT INDEX_NAME FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_rule' AND NON_UNIQUE=0 AND INDEX_NAME<>'PRIMARY'
  GROUP BY INDEX_NAME HAVING COUNT(*)=1 AND MAX(COLUMN_NAME)='name' LIMIT 1);
SET @sql := IF(@old IS NULL, 'SELECT 1', CONCAT('ALTER TABLE obs_alert_rule DROP INDEX `', @old, '`'));
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_rule')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_alert_rule' AND INDEX_NAME='uk_obs_alert_rule_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE obs_alert_rule ADD UNIQUE INDEX uk_obs_alert_rule_name (tenant_id, name)');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @old := (SELECT INDEX_NAME FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_channel' AND NON_UNIQUE=0 AND INDEX_NAME<>'PRIMARY'
  GROUP BY INDEX_NAME HAVING COUNT(*)=1 AND MAX(COLUMN_NAME)='name' LIMIT 1);
SET @sql := IF(@old IS NULL, 'SELECT 1', CONCAT('ALTER TABLE obs_notification_channel DROP INDEX `', @old, '`'));
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_channel')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='obs_notification_channel' AND INDEX_NAME='uk_obs_notification_channel_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE obs_notification_channel ADD UNIQUE INDEX uk_obs_notification_channel_name (tenant_id, name)');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release' AND INDEX_NAME='uk_k8s_release_idempotency')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release' AND INDEX_NAME='uk_k8s_release_idempotency' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_k8s_release DROP INDEX uk_k8s_release_idempotency');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_release' AND INDEX_NAME='uk_k8s_release_idempotency' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_k8s_release ADD UNIQUE INDEX uk_k8s_release_idempotency (tenant_id, idempotency_key)');
PREPARE t14stmt FROM @sql; EXECUTE t14stmt; DEALLOCATE PREPARE t14stmt;
