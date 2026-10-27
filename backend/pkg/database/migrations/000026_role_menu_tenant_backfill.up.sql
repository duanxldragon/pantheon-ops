-- Role→menu grants: backfill the tenant dimension (contract §2.3/§4)
--
-- WHY THIS FILE EXISTS
-- --------------------
-- `system_role_menu` has carried `tenant_id` and `PRIMARY KEY (tenant_id,
-- role_id, menu_id)` since 000020, but `InsertRoleMenus` kept inserting only
-- `(role_id, menu_id)`. Every grant written from a tenant console therefore
-- landed in the global namespace (`tenant_id = 0`), while `loadRoleMenus`
-- reads under the request tenant scope. The grant was unreadable the moment it
-- was saved, and re-saving an existing tenant role deleted the rows it then
-- failed to rewrite (acceptance F3).
--
-- The code fix (W3) writes the grant into the tenant that owns the role. This
-- migration moves the historical rows to the same place so already-configured
-- tenant roles keep their navigation access instead of coming back empty.
--
-- WHY IT CAN BE AUTOMATIC (unlike the data-scope backfill)
-- -------------------------------------------------------
-- `system_role.id` is globally unique, so a grant row references exactly one
-- role, and that role's `tenant_id` is authoritative: there is no ambiguity to
-- defer to an operator. Platform roles keep `tenant_id = 0`.
--
-- IDEMPOTENT: the predicate matches only rows still at 0 whose role belongs to
-- a tenant, so a re-run matches nothing. `UPDATE IGNORE` drops a row whose
-- tenant-owned copy already exists (duplicate), leaving the existing grant
-- untouched.
--
-- GUARDED, like 000024/000025: a database can record a later schema_migrations
-- version while an earlier DDL never actually ran. If either tenant_id column
-- is absent the statement degrades to a no-op instead of turning startup into
-- a failure loop.
--
-- DOWN: data is intentionally retained (see the .down.sql).

SET @has_role_menu_tenant := (SELECT COUNT(*) FROM information_schema.columns
                              WHERE table_schema = DATABASE() AND table_name = 'system_role_menu' AND column_name = 'tenant_id');
SET @has_role_tenant := (SELECT COUNT(*) FROM information_schema.columns
                         WHERE table_schema = DATABASE() AND table_name = 'system_role' AND column_name = 'tenant_id');

SET @s := IF(@has_role_menu_tenant > 0 AND @has_role_tenant > 0,
  'UPDATE IGNORE `system_role_menu` AS rm
   JOIN `system_role` AS r ON r.`id` = rm.`role_id`
   SET rm.`tenant_id` = r.`tenant_id`
   WHERE rm.`tenant_id` = 0
     AND r.`tenant_id` <> 0
     AND r.`deleted_at` IS NULL',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;

-- If a tenant-owned copy already existed, `UPDATE IGNORE` leaves the global
-- row at 0 instead of overwriting the real grant. Drop that phantom only when
-- the tenant-owned copy is demonstrably there, so the statement can never
-- delete the sole grant. On a legacy (role_id, menu_id) primary key the copy
-- cannot exist alongside the 0 row, so this is a no-op there as well.
SET @s := IF(@has_role_menu_tenant > 0 AND @has_role_tenant > 0,
  'DELETE rm FROM `system_role_menu` AS rm
   JOIN `system_role` AS r ON r.`id` = rm.`role_id`
   JOIN (SELECT `tenant_id`, `role_id`, `menu_id` FROM `system_role_menu` WHERE `tenant_id` <> 0) AS owned
     ON owned.`role_id` = rm.`role_id`
    AND owned.`menu_id` = rm.`menu_id`
    AND owned.`tenant_id` = r.`tenant_id`
   WHERE rm.`tenant_id` = 0
     AND r.`tenant_id` <> 0
     AND r.`deleted_at` IS NULL',
  'SELECT 1');
PREPARE st FROM @s; EXECUTE st; DEALLOCATE PREPARE st;
