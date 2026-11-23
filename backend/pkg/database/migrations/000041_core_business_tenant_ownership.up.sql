-- T04: core business tenant ownership (frozen list: T01 baseline classification).
-- Adds tenant_id to the 32 core tenant-owned business tables and moves the eight
-- natural-name unique keys to tenant-first variants so two tenants may use the
-- same name. Idempotent: every statement is guarded by information_schema, so
-- reruns are no-ops and a fresh install that creates tables afterwards via
-- AutoMigrate simply skips the missing-table branch.

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_package ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_template ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template_step')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template_step' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_template_step ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_host')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_host' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_host ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_host_lease')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_host_lease' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_host_lease ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_attempt')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_attempt' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_attempt ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_credential_ref ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_step')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_step' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_step ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_event')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_event' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_event ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_log_chunk')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_log_chunk' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_log_chunk ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_secret')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_task_secret' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_task_secret ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_step_history')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_step_history' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_step_history ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_host ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_group')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_group' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_group ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_label_schema ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_cluster ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster_credential_ref')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster_credential_ref' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_cluster_credential_ref ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_namespace_binding')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_namespace_binding' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_namespace_binding ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_pv')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_pv' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_pv ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_pvc')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_pvc' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_pvc ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_storage_class')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_storage_class' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_storage_class ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_ingress')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_ingress' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_k8s_ingress ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_backup')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_backup' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_velero_backup ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_schedule')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_schedule' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_velero_schedule ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_restore')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_restore' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_velero_restore ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_velero_storage_location ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_drill')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_drill' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_velero_drill ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_service')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_service' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_service ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_service_instance')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_service_instance' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_service_instance ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_business_scope ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_environment')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_environment' AND COLUMN_NAME='tenant_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_business_environment ADD COLUMN tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0 AFTER id');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

-- Tenant-first unique key variants (drop the global-name index, then add the
-- tenant-scoped one; each step is independently guarded).

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package' AND INDEX_NAME='uk_deploy_package_name_version_deleted')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package' AND INDEX_NAME='uk_deploy_package_name_version_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_package DROP INDEX uk_deploy_package_name_version_deleted');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_package' AND INDEX_NAME='uk_deploy_package_name_version_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_package ADD UNIQUE INDEX uk_deploy_package_name_version_deleted (tenant_id, name, version, deleted_at)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template' AND INDEX_NAME='uk_deploy_template_name_version_deleted')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template' AND INDEX_NAME='uk_deploy_template_name_version_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_template DROP INDEX uk_deploy_template_name_version_deleted');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_template' AND INDEX_NAME='uk_deploy_template_name_version_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_template ADD UNIQUE INDEX uk_deploy_template_name_version_deleted (tenant_id, name, version, deleted_at)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND INDEX_NAME='uk_deploy_credential_name_deleted')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND INDEX_NAME='uk_deploy_credential_name_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_credential_ref DROP INDEX uk_deploy_credential_name_deleted');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND INDEX_NAME='uk_deploy_credential_name_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_deploy_credential_ref ADD UNIQUE INDEX uk_deploy_credential_name_deleted (tenant_id, name, deleted_at)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host' AND INDEX_NAME='uk_ip_deleted')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host' AND INDEX_NAME='uk_ip_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_host DROP INDEX uk_ip_deleted');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_host' AND INDEX_NAME='uk_ip_deleted' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_host ADD UNIQUE INDEX uk_ip_deleted (tenant_id, ip, deleted_at)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema' AND INDEX_NAME='uk_cmdb_label_schema_key')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema' AND INDEX_NAME='uk_cmdb_label_schema_key' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_label_schema DROP INDEX uk_cmdb_label_schema_key');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_cmdb_label_schema' AND INDEX_NAME='uk_cmdb_label_schema_key' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_cmdb_label_schema ADD UNIQUE INDEX uk_cmdb_label_schema_key (tenant_id, `key`)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster' AND INDEX_NAME='uk_k8s_cluster_code')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster' AND INDEX_NAME='uk_k8s_cluster_code' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_k8s_cluster DROP INDEX uk_k8s_cluster_code');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_k8s_cluster' AND INDEX_NAME='uk_k8s_cluster_code' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_k8s_cluster ADD UNIQUE INDEX uk_k8s_cluster_code (tenant_id, code, deleted_at)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope' AND INDEX_NAME='idx_biz_business_scope_code')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope' AND INDEX_NAME='idx_biz_business_scope_code' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_business_scope DROP INDEX idx_biz_business_scope_code');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_business_scope' AND INDEX_NAME='idx_biz_business_scope_code' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_business_scope ADD UNIQUE INDEX idx_biz_business_scope_code (tenant_id, code)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location' AND INDEX_NAME='uni_biz_velero_storage_location_name')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location' AND INDEX_NAME='uni_biz_velero_storage_location_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_velero_storage_location DROP INDEX uni_biz_velero_storage_location_name');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;
SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location')=0 OR
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_velero_storage_location' AND INDEX_NAME='uni_biz_velero_storage_location_name' AND SEQ_IN_INDEX=1 AND COLUMN_NAME='tenant_id')=1,
  'SELECT 1',
  'ALTER TABLE biz_velero_storage_location ADD UNIQUE INDEX uni_biz_velero_storage_location_name (tenant_id, name)');
PREPARE t04stmt FROM @sql; EXECUTE t04stmt; DEALLOCATE PREPARE t04stmt;

