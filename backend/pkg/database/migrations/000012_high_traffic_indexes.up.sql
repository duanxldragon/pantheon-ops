-- High-traffic index optimizations for pantheon-base
-- Addresses N+1 query patterns and slow dashboard aggregates
--
-- Guarded via information_schema + PREPARE (same compat pattern as 000009/000011)
-- so this file is idempotent on re-run. It was previously unguarded, which was
-- survivable only while nothing ever replayed it; the 2026-09-24 marker change
-- (system_i18n.updated_by) makes the bootstrap rewind schema_migrations to v7
-- and replay 8..latest on existing databases, and an unguarded CREATE INDEX
-- died on the duplicate-key error at replay.
--
-- NOTE: Security event index (severity, acknowledged_at, created_at) deferred
-- because system_auth_security_event table in 000001 is missing those columns.
-- The model (security_model.go) defines them, indicating a schema drift.
-- This index should be added in a future migration after the columns are added.

SET @session_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'system_user_session'
    AND index_name = 'idx_user_session_user_revoked_expires'
);
SET @session_stmt := IF(
  @session_idx = 0,
  'CREATE INDEX idx_user_session_user_revoked_expires ON system_user_session(user_id, revoked_at, refresh_expires_at)',
  'SELECT 1'
);
PREPARE session_stmt FROM @session_stmt;
EXECUTE session_stmt;
DEALLOCATE PREPARE session_stmt;

SET @casbin_idx := (
  SELECT COUNT(*)
  FROM information_schema.statistics
  WHERE table_schema = DATABASE()
    AND table_name = 'casbin_rule'
    AND index_name = 'idx_casbin_ptype_v0_v1'
);
SET @casbin_stmt := IF(
  @casbin_idx = 0,
  'CREATE INDEX idx_casbin_ptype_v0_v1 ON casbin_rule(ptype, v0, v1)',
  'SELECT 1'
);
PREPARE casbin_stmt FROM @casbin_stmt;
EXECUTE casbin_stmt;
DEALLOCATE PREPARE casbin_stmt;
