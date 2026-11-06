-- Preflight for 000031_drop_permission_role_data_scope_policy.
--
-- READ ONLY. Every statement below is a SELECT; nothing here writes, locks for
-- update, or drops. Run it against the target database BEFORE the deployment
-- that applies 000031, then keep the output with the change record.
--
-- Why this exists: 000031.up.sql is `DROP TABLE`, and its down migration
-- rebuilds the STRUCTURE ONLY — rows are gone for good. The migration cannot
-- ask "is this table still holding data?" by itself, so the question is asked
-- here, by a human, before the deploy.
--
-- Statements are separated by a lone `;` on its own line so the file can be fed
-- to `mysql` as a script or executed statement-by-statement by a test
-- (see migrate_preflight_test.go).

-- 1. Which schema version is this database on?
--    31 means 000031 has already been applied and statements 2-4 will fail
--    because the table is gone — that is expected, skip to step 5.
SELECT version, dirty FROM schema_migrations
;

-- 2. Does the table this migration drops still exist?
--    Expect 1 before the deploy, 0 after it.
SELECT COUNT(*) AS dead_table_exists
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = 'permission_role_data_scope_policy'
;

-- 3. How many rows would be destroyed?
--    Non-zero means: export first (see the runbook), then decide whether those
--    rows are genuinely disposable. Zero means nothing is lost.
SELECT COUNT(*) AS rows_to_be_dropped
FROM permission_role_data_scope_policy
;

-- 4. Is the replacement table present?
--    Dropping the old table is only safe when the live policy actually lives in
--    system_role_data_scope. If this returns 0 while step 3 returns > 0, stop:
--    the rows in the old table may be the only copy.
SELECT COUNT(*) AS replacement_table_exists
FROM information_schema.tables
WHERE table_schema = DATABASE()
  AND table_name = 'system_role_data_scope'
;

-- 5. Is the replacement table actually carrying the policy now?
--    0 here together with a non-zero step 3 is the dangerous combination.
SELECT COUNT(*) AS replacement_rows
FROM system_role_data_scope
;

-- 6. Eyeball the rows that are about to disappear, if any.
SELECT id, tenant_id, role_key, mode, created_at
FROM permission_role_data_scope_policy
ORDER BY id
LIMIT 20
;
