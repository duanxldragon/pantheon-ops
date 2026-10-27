-- Session pagination indexes (task 2026-09-22-export-and-session-pagination).
-- ListAllSessions now pushes the revoked_at status filter, the active-scope
-- refresh_expires_at predicate, and the created_at DESC ordering into SQL
-- (COUNT + LIMIT/OFFSET). Without these indexes MySQL full-scans
-- system_user_session and filesorts on every admin page request.
--
-- Same guarded pattern as 000017: the ALTER only runs when the index is
-- absent, so re-running or partially-provisioned dev databases stay safe.

SET @schema := DATABASE();

SET @t := (SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = @schema AND table_name = 'system_user_session');

SET @c1 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_revoked_at');
SET @s1 := IF(@t>0 AND @c1=0,'ALTER TABLE `system_user_session` ADD INDEX `idx_system_user_session_revoked_at` (`revoked_at`)','SELECT 1');
PREPARE stmt1 FROM @s1; EXECUTE stmt1; DEALLOCATE PREPARE stmt1;

SET @c2 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_created_at');
SET @s2 := IF(@t>0 AND @c2=0,'ALTER TABLE `system_user_session` ADD INDEX `idx_system_user_session_created_at` (`created_at`)','SELECT 1');
PREPARE stmt2 FROM @s2; EXECUTE stmt2; DEALLOCATE PREPARE stmt2;

SET @c3 := (SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @schema AND table_name = 'system_user_session' AND index_name = 'idx_system_user_session_refresh_expires_at');
SET @s3 := IF(@t>0 AND @c3=0,'ALTER TABLE `system_user_session` ADD INDEX `idx_system_user_session_refresh_expires_at` (`refresh_expires_at`)','SELECT 1');
PREPARE stmt3 FROM @s3; EXECUTE stmt3; DEALLOCATE PREPARE stmt3;
