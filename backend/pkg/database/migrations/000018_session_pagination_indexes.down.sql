-- Down migration for 000018: drop the session pagination indexes.
SET @schema := DATABASE();

SET @s1 := 'ALTER TABLE `system_user_session` DROP INDEX `idx_system_user_session_revoked_at`';
SET @c1 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_revoked_at');
SET @s1 := IF(@c1>0, @s1, 'SELECT 1');
PREPARE stmt1 FROM @s1; EXECUTE stmt1; DEALLOCATE PREPARE stmt1;

SET @s2 := 'ALTER TABLE `system_user_session` DROP INDEX `idx_system_user_session_created_at`';
SET @c2 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_created_at');
SET @s2 := IF(@c2>0, @s2, 'SELECT 1');
PREPARE stmt2 FROM @s2; EXECUTE stmt2; DEALLOCATE PREPARE stmt2;

SET @s3 := 'ALTER TABLE `system_user_session` DROP INDEX `idx_system_user_session_refresh_expires_at`';
SET @c3 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_refresh_expires_at');
SET @s3 := IF(@c3>0, @s3, 'SELECT 1');
PREPARE stmt3 FROM @s3; EXECUTE stmt3; DEALLOCATE PREPARE stmt3;
