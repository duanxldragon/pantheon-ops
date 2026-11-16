-- Add the per-scope environment catalog promoted in the bizscope M+3C. Each
-- row belongs to one business scope (business_scope_id) and carries an
-- environment type (dev/test/prod). Idempotent: run only when the table is
-- absent.
SET @bs_env_exists = (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'biz_business_environment');
SET @bs_env_cols = IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'biz_business_environment' AND column_name = 'business_scope_id') = 0, 'ADD COLUMN business_scope_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL);
SET @bs_env_create = IF(@bs_env_exists = 0, 'CREATE TABLE biz_business_environment (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  business_scope_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
  code VARCHAR(50) NOT NULL,
  name VARCHAR(255) NOT NULL,
  env_type VARCHAR(50) NOT NULL,
  status VARCHAR(50) NOT NULL,
  remark TEXT NULL,
  created_at DATETIME(3) NULL,
  updated_at DATETIME(3) NULL,
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_biz_business_environment_scope_code (business_scope_id, code),
  KEY idx_biz_business_environment_deleted_at (deleted_at),
  KEY idx_biz_business_environment_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4', 'SELECT 1');
PREPARE bs_env_stmt FROM @bs_env_create;
EXECUTE bs_env_stmt;
DEALLOCATE PREPARE bs_env_stmt;
SET @bs_env_stmt2 = IF(@bs_env_exists > 0 AND @bs_env_cols IS NOT NULL, CONCAT('ALTER TABLE biz_business_environment ', @bs_env_cols), 'SELECT 1');
PREPARE bs_env_stmt2 FROM @bs_env_stmt2;
EXECUTE bs_env_stmt2;
DEALLOCATE PREPARE bs_env_stmt2;