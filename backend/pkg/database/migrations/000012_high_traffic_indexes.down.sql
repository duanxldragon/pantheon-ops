-- Rollback high-traffic indexes (guarded, mirrors the up file).
--
-- The previous version used `DROP INDEX IF EXISTS` (not valid MySQL DDL) and
-- dropped idx_security_event_severity_ack_created, which the up file never
-- created (it is explicitly deferred). Both defects are fixed here so the
-- rollback path actually executes: each drop is guarded by information_schema
-- and only touches the two indexes the up file creates.

SET @s1 := 'DROP INDEX idx_user_session_user_revoked_expires ON system_user_session';
SET @c1 := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_session'
    AND index_name = 'idx_user_session_user_revoked_expires'
);
SET @s1 := IF(@c1 > 0, @s1, 'SELECT 1');
PREPARE stmt1 FROM @s1;
EXECUTE stmt1;
DEALLOCATE PREPARE stmt1;

SET @s2 := 'DROP INDEX idx_casbin_ptype_v0_v1 ON casbin_rule';
SET @c2 := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'casbin_rule'
    AND index_name = 'idx_casbin_ptype_v0_v1'
);
SET @s2 := IF(@c2 > 0, @s2, 'SELECT 1');
PREPARE stmt2 FROM @s2;
EXECUTE stmt2;
DEALLOCATE PREPARE stmt2;
