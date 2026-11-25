-- T05: deploy credential references are owned by a business scope (SEC-004).
-- A credential row may only be listed/used by callers whose data scope covers
-- the owning business scope. Existing rows keep 0 = platform/admin-owned;
-- they are never silently treated as a shared pool for scoped roles.
-- Idempotent: reruns are no-ops, and a fresh install that AutoMigrates later
-- simply skips the missing-table branch.

SET @sql := IF(
  (SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref')=0 OR
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='biz_deploy_credential_ref' AND COLUMN_NAME='business_scope_id')>0,
  'SELECT 1',
  'ALTER TABLE biz_deploy_credential_ref ADD COLUMN business_scope_id BIGINT UNSIGNED NOT NULL DEFAULT 0, ADD INDEX idx_deploy_credential_business_scope (business_scope_id)');
PREPARE t05stmt FROM @sql; EXECUTE t05stmt; DEALLOCATE PREPARE t05stmt;
