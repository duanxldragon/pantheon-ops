-- Add trusted numeric ownership and stable-ID selection facts without inferring
-- ownership from historical usernames. Missing FileDist tables remain missing;
-- the module's existing schema bootstrap remains responsible for initial tables.
SET @fd_task_exists = (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'filedist_task');
SET @fd_task_cols = CONCAT_WS(', ',
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_task' AND column_name = 'owner_id') = 0, 'ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_task' AND column_name = 'dept_id') = 0, 'ADD COLUMN dept_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_task' AND column_name = 'selection') = 0, 'ADD COLUMN selection JSON NULL', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_task' AND column_name = 'scope_version') = 0, 'ADD COLUMN scope_version VARCHAR(64) NULL', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_task' AND column_name = 'resolved_at') = 0, 'ADD COLUMN resolved_at DATETIME(3) NULL', NULL));
SET @fd_stmt = IF(@fd_task_exists > 0 AND @fd_task_cols <> '', CONCAT('ALTER TABLE filedist_task ', @fd_task_cols), 'SELECT 1');
PREPARE fd_migration FROM @fd_stmt;
EXECUTE fd_migration;
DEALLOCATE PREPARE fd_migration;

SET @fd_file_exists = (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'filedist_file');
SET @fd_file_cols = CONCAT_WS(', ',
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_file' AND column_name = 'owner_id') = 0, 'ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_file' AND column_name = 'dept_id') = 0, 'ADD COLUMN dept_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL));
SET @fd_stmt = IF(@fd_file_exists > 0 AND @fd_file_cols <> '', CONCAT('ALTER TABLE filedist_file ', @fd_file_cols), 'SELECT 1');
PREPARE fd_migration FROM @fd_stmt;
EXECUTE fd_migration;
DEALLOCATE PREPARE fd_migration;

SET @fd_target_exists = (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'filedist_target');
SET @fd_target_cols = CONCAT_WS(', ',
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_target' AND column_name = 'host_id') = 0, 'ADD COLUMN host_id BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL),
 IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'filedist_target' AND column_name = 'state_version') = 0, 'ADD COLUMN state_version BIGINT UNSIGNED NOT NULL DEFAULT 0', NULL));
SET @fd_stmt = IF(@fd_target_exists > 0 AND @fd_target_cols <> '', CONCAT('ALTER TABLE filedist_target ', @fd_target_cols), 'SELECT 1');
PREPARE fd_migration FROM @fd_stmt;
EXECUTE fd_migration;
DEALLOCATE PREPARE fd_migration;
