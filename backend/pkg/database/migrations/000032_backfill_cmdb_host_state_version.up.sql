-- Backfill CMDB host state versions so compare-and-swap is enforced
--
-- WHY THIS FILE EXISTS
-- --------------------
-- `biz_cmdb_host.state_version` was created with `DEFAULT 0`, and neither
-- `HostService.Create` nor the import path assigned it. `TransitionState` only
-- enables CAS when the caller sends a positive `ExpectedStateVersion`, so every
-- host created before this fix reported `stateVersion = 0` and silently skipped
-- the compare-and-swap: a stale expectation fell through to lifecycle
-- validation and answered `cmdbhost.state.invalid` instead of
-- `cmdbhost.state.stale`.
--
-- The code fix makes new hosts start at version 1 (creation is version 1, each
-- transition increments). Existing rows still carry 0, which keeps them outside
-- CAS protection until their first transition. This migration moves them to the
-- same starting point, so a client that re-reads a host gets a version that is
-- actually enforced from then on.
--
-- WHAT IT DELIBERATELY DOES NOT DO
-- --------------------------------
--   * It does not touch rows already above 0. Those versions were produced by
--     real transitions and rewriting them could invalidate a client's in-flight
--     expectation.
--   * It does not invent history. The version starts at 1 because that is what
--     a host created under the fixed code reports; it is not a replay of past
--     transitions.
--
-- IDEMPOTENT: the predicate is `state_version = 0`, so a re-run (or a replay
-- after the baseline was rewound) is a no-op once the rows are at 1.
-- DOWN: data is intentionally retained (see the .down.sql).
--
-- GUARDED on table and column existence, like 000025, because business tables
-- are created by module AutoMigrate rather than by this chain: a system-only
-- database reaches this version without ever having `biz_cmdb_host`, and an
-- unguarded UPDATE would turn startup into a failure loop there.

SET @has_host_table := (SELECT COUNT(*) FROM information_schema.tables
                        WHERE table_schema = DATABASE() AND table_name = 'biz_cmdb_host');
SET @has_state_version := (SELECT COUNT(*) FROM information_schema.columns
                           WHERE table_schema = DATABASE() AND table_name = 'biz_cmdb_host' AND column_name = 'state_version');

SET @s := IF(@has_host_table > 0 AND @has_state_version > 0,
  'UPDATE `biz_cmdb_host` SET `state_version` = 1 WHERE `state_version` = 0',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
