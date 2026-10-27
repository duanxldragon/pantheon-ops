-- Credential refs are retained on rollback to avoid deleting encrypted
-- material that may still be referenced by an older cluster record.
SET @has_index = (
  SELECT COUNT(*) FROM information_schema.statistics
  WHERE table_schema = DATABASE() AND table_name = 'biz_k8s_cluster'
    AND index_name = 'idx_k8s_cluster_credential_ref'
);
SET @sql = IF(@has_index > 0,
  'DROP INDEX idx_k8s_cluster_credential_ref ON biz_k8s_cluster',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_column = (
  SELECT COUNT(*) FROM information_schema.columns
  WHERE table_schema = DATABASE() AND table_name = 'biz_k8s_cluster'
    AND column_name = 'kubeconfig_credential_ref_id'
);
SET @sql = IF(@has_column > 0,
  'ALTER TABLE biz_k8s_cluster DROP COLUMN kubeconfig_credential_ref_id',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
