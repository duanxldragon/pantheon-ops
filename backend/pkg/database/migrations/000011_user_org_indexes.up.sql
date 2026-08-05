-- Indexes on system_user org columns. The dept/post delete guards count users
-- via WHERE dept_id = ? / post_id IN ? (dept_service.DeleteDept,
-- post_service.ensurePostsNotAssignedToUsers); without these indexes each
-- guard is a full scan of system_user.
-- Guarded via information_schema so the migration is idempotent and safe on
-- bootstrapped/partial schemas (same pattern as 000009).

SET @user_dept_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'idx_system_user_dept_id'
);
SET @user_table_exists := (
  SELECT COUNT(*)
  FROM information_schema.tables
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
);
SET @user_dept_stmt := IF(
  @user_table_exists > 0 AND @user_dept_idx = 0,
  'ALTER TABLE system_user ADD INDEX idx_system_user_dept_id (dept_id)',
  'SELECT 1'
);
PREPARE user_dept_stmt FROM @user_dept_stmt;
EXECUTE user_dept_stmt;
DEALLOCATE PREPARE user_dept_stmt;

SET @user_post_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user'
    AND index_name = 'idx_system_user_post_id'
);
SET @user_post_stmt := IF(
  @user_table_exists > 0 AND @user_post_idx = 0,
  'ALTER TABLE system_user ADD INDEX idx_system_user_post_id (post_id)',
  'SELECT 1'
);
PREPARE user_post_stmt FROM @user_post_stmt;
EXECUTE user_post_stmt;
DEALLOCATE PREPARE user_post_stmt;
