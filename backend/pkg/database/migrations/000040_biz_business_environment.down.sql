-- Drop the bizapp scope environment catalog when rolling back the M+3C
-- promotion. No business records depend on this table.
SET @bs_env_exists = (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'biz_business_environment');
SET @bs_env_stmt = IF(@bs_env_exists > 0, 'DROP TABLE biz_business_environment', 'SELECT 1');
PREPARE bs_env_stmt FROM @bs_env_stmt;
EXECUTE bs_env_stmt;
DEALLOCATE PREPARE bs_env_stmt;