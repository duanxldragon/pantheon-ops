-- Role data-scope policies: backfill the tenant dimension (contract §2.3/§4)
--
-- WHY THIS FILE EXISTS
-- --------------------
-- `system_role_data_scope` has carried `tenant_id` and
-- `UNIQUE KEY uk_role_data_scope_tenant_role (tenant_id, role_key)` since
-- 000024, but until now no read or write path used the column: every policy was
-- selected and upserted by `role_key` alone. The runtime therefore treated
-- `role_key` as a single global namespace shared by all tenants, and because
-- `resolveDataScopeMode` takes the widest matching row, one tenant's policy
-- decided how many rows another tenant saw (acceptance F7/F10).
--
-- The code fix (W1) makes every policy read/write tenant-scoped. That change
-- alone would orphan the existing population: the console wrote policies with
-- `tenant_id` left at its default 0, so after the fix a tenant-local role's
-- policy would no longer be found, the fail-closed lookup would trip, and every
-- request from a user holding that role would answer 500.
--
-- This migration copies each platform-scoped (tenant_id = 0) policy onto the
-- tenants that actually own the role key, so the post-fix read finds the same
-- policy the pre-fix read effectively used.
--
-- WHAT IT DELIBERATELY DOES NOT DO
-- --------------------------------
--   * It does not copy a policy that exists only under another tenant. That row
--     is another tenant's authority, and copying it would preserve exactly the
--     cross-tenant leak F10 describes.
--   * It does not invent a policy for a role that has none anywhere. Pre-fix,
--     such a role already fell through the fail-closed lookup; adding an `all`
--     row here would silently widen a role nobody ever configured.
--   * It does not rewrite `dept_ids`. The copied set is the same set the tenant
--     was already filtered by, and the row filter is tenant-scoped, so a
--     department id belonging to a different tenant can only narrow the result
--     to zero rows — never widen it. New writes are validated against the
--     caller's tenant by `ensureDeptIDsBelongToTenant` (F11).
--
-- IDEMPOTENT: `INSERT IGNORE` against the (tenant_id, role_key) unique key makes
-- a re-run a no-op, and an existing tenant-owned row is never overwritten.
-- DOWN: data is intentionally retained (see the .down.sql).
--
-- GUARDED, like 000024, and for the same reason: a database can carry
-- schema_migrations = 24 while 000024's DDL never actually ran (that is the
-- whole reason 000024 exists). On such a database `tenant_id` is absent from
-- both tables, and an unguarded INSERT would turn startup into a hard failure
-- loop. The guard degrades to a no-op instead — the columns are what 000024
-- adds, and in a normal upgrade 000024 has already run by the time this file is
-- reached, so the guard only fires on the pathological case.

SET @has_role_tenant := (SELECT COUNT(*) FROM information_schema.columns
                         WHERE table_schema = DATABASE() AND table_name = 'system_role' AND column_name = 'tenant_id');
SET @has_scope_tenant := (SELECT COUNT(*) FROM information_schema.columns
                          WHERE table_schema = DATABASE() AND table_name = 'system_role_data_scope' AND column_name = 'tenant_id');

SET @s := IF(@has_role_tenant > 0 AND @has_scope_tenant > 0,
  'INSERT IGNORE INTO `system_role_data_scope` (`tenant_id`, `role_key`, `mode`, `dept_ids`)
   SELECT r.`tenant_id`, r.`role_key`, p.`mode`, p.`dept_ids`
   FROM `system_role` AS r
   JOIN `system_role_data_scope` AS p
     ON p.`role_key` = r.`role_key`
    AND p.`tenant_id` = 0
   WHERE r.`tenant_id` <> 0
     AND r.`deleted_at` IS NULL
     AND TRIM(r.`role_key`) <> ''''',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
