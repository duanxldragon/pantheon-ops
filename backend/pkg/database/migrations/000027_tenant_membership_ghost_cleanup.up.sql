-- Tenant memberships: deactivate the rows of soft-deleted users (contract §2.2)
--
-- WHY THIS FILE EXISTS
-- --------------------
-- DeleteUser soft-deletes system_user and rewrites the username to
-- `__deleted_user_<id>`, but before W2 it never touched tenant_memberships.
-- The membership stayed `active`, and the member list joins system_user, so
-- every deleted user kept showing up as a `__deleted_user_*` member that the
-- console cannot remove (acceptance F4). The W2 code fix deactivates the
-- membership in the same transaction; this migration repairs the rows that
-- were already left behind.
--
-- Deactivation, not deletion: a membership is the audit record of a past
-- relationship, and the login gate already treats any non-active status as a
-- denial. Deleting the rows would erase that history for no benefit.
--
-- IDEMPOTENT: the predicate only matches active memberships whose user row is
-- already soft-deleted, so a re-run matches nothing.
--
-- GUARDED: tenant_memberships only exists on tenant-era schemas (000022/000023
-- created it), and a database can carry a later schema_migrations version
-- while an earlier DDL never ran. If either table or the deleted_at column is
-- absent the statement degrades to a no-op instead of failing startup.
--
-- DOWN: data is intentionally retained (see the .down.sql).

SET @has_memberships := (SELECT COUNT(*) FROM information_schema.tables
                         WHERE table_schema = DATABASE() AND table_name = 'tenant_memberships');
SET @has_user_deleted_at := (SELECT COUNT(*) FROM information_schema.columns
                             WHERE table_schema = DATABASE() AND table_name = 'system_user' AND column_name = 'deleted_at');

SET @s := IF(@has_memberships > 0 AND @has_user_deleted_at > 0,
  'UPDATE `tenant_memberships` AS tm
   JOIN `system_user` AS u ON u.`id` = tm.`user_id`
   SET tm.`status` = ''disabled''
   WHERE tm.`status` = ''active''
     AND u.`deleted_at` IS NOT NULL',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
