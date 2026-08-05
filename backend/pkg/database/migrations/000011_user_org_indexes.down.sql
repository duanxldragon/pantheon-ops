SET @user_post_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'idx_system_user_post_id'
);
SET @user_post_stmt := IF(
  @user_post_idx > 0,
  'DROP INDEX idx_system_user_post_id ON system_user',
  'SELECT 1'
);
PREPARE user_post_stmt FROM @user_post_stmt;
EXECUTE user_post_stmt;
DEALLOCATE PREPARE user_post_stmt;

SET @user_dept_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'idx_system_user_dept_id'
);
SET @user_dept_stmt := IF(
  @user_dept_idx > 0,
  'DROP INDEX idx_system_user_dept_id ON system_user',
  'SELECT 1'
);
PREPARE user_dept_stmt FROM @user_dept_stmt;
EXECUTE user_dept_stmt;
DEALLOCATE PREPARE user_dept_stmt;
