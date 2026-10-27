# Tenant Troubleshooting Guide

## Common Issues and Solutions

### 1. Tenant Creation Fails

**Symptom**: `POST /api/v1/system/tenant` is refused with one of the two code-conflict messages.

Note the envelope shape: the API answers **HTTP 200** with the outcome in the body
(`{"code":400,"message":"tenant.code_taken"}`), so read `message`, not the HTTP status.

**Cause A** — `message: "tenant.code_taken"`: a **live** tenant already holds the code.

**Cause B** — `message: "tenant.code_reserved"`: the code is held by a **soft-deleted**
tenant. Deleted tenants are hidden from the default console list, so the list looks empty
even though the code is taken. Deleting a tenant archives its code to
`__deleted_tenant_<id>` (see `docs/designs/BACKEND.md` §7), so this state should only be
observable for a row written before that rule existed — the startup repair clears it on the
next backend start.

**Solution**:
```bash
# Both cases: inspect what actually holds the code, including deleted rows.
# The console list hides deleted tenants by default; this does not.
mysql -uroot -p -e "SELECT id, code, name, status FROM tenants WHERE code = '<the-code>';"

# Cause A: pick a different code.
# Cause B: also fine to pick a different code — or release the stale reservation:
#   (a) restart the backend, which runs ReleaseDeletedTenantCodes, or
#   (b) archive it directly:
mysql -uroot -p -e "UPDATE tenants SET code = CONCAT('__deleted_tenant_', id) WHERE id = <id>;"
```

A released code becomes immediately usable again, and the deleted tenant stays traceable by
id (its `name` is preserved).

---

### 2. Cannot Delete Tenant

**Symptom**: `DELETE /api/v1/system/tenant/:id` is refused because the tenant still has active members

**Cause**: The tenant is not archived and still has active memberships. Membership is the login
gate, so the API deliberately refuses while any active membership remains (contract §2.2).

**Why "remove the members first" does not work**: the last owner can never be removed
(`tenant.member.last_owner`), so a provisioned tenant has no member-free state on its own.
The supported path is archive-then-delete: an archived tenant is terminal (§2.1) and may be
deleted even while members remain; the delete transaction disables those memberships.

**Solution**:
```bash
# Console: Tenants -> Edit -> Status = Archived -> Save, then Delete.
#
# API equivalent. Both writes use the session cookies and the CSRF token; the
# DELETE is also a secure action and needs X-Operation-Token from
# POST /api/v1/auth/operation-verify:
curl -X PUT http://localhost:8080/api/v1/system/tenant/1 \
  -H "Content-Type: application/json" \
  -H "Cookie: <session cookies>" -H "X-CSRF-Token: <csrf>" \
  -d '{"status":"archived"}'

curl -X DELETE http://localhost:8080/api/v1/system/tenant/1 \
  -H "Cookie: <session cookies>" -H "X-CSRF-Token: <csrf>" -H "X-Operation-Token: <op-token>"
```

> The refusal is reported in the response envelope as `code: 400` with message
> `tenant.has_active_members` (HTTP 200 — this API puts the outcome in the body). Teardown
> scripts should archive the tenant (or purge memberships directly) before calling delete; the
> message key, not the HTTP class, is the reliable signal.
>
> Deleting a tenant also **archives its `code`** to `__deleted_tenant_<id>` in the same
> transaction, so the code is free for reuse afterwards. That is why a deleted tenant no longer
> appears under its original code — match deleted rows by `id` or `name`, not by `code`.

---

### 2b. Hard-deleting a tenant (ops action): purge Casbin policies first

**Context**: `DELETE /system/tenant/:id` is a **soft** delete. Removing a tenant row outright is an
ops action (contract §2.1), and it is the one deletion path the application cannot do for you.

**Why it matters**: `casbin_rule` has **no foreign key** to `tenants` and the application has **no
cleanup path** for tenant-domain policies. Provisioning writes them as
`p.type='p'`, `v0='role:<role_key>@tenant:<id>'` (7 rows per role: 5 methods on `/api/v1/*` plus
whatever the role definition adds). A hard delete that skips them leaves an orphaned grant: if the
`tenants` AUTO_INCREMENT later recycles the id, the new tenant **inherits the old tenant's
authority**. This is a security-relevant leftover, not just row hygiene.

**Procedure** (replace `<id>` with the tenant id):

> **Scope the purge to `<id>`, and to `<id>` only.** The `DELETE` below is a plain
> `LIKE`, so a pattern that is too broad — an empty id, a loop over the wrong
> list, a `%` — will strip the domain policies of **live** tenants too. A tenant
> whose policies are gone still authenticates; it just answers
> `403 permission.denied` on every page (issue 5c). The startup repair
> (`Service.EnsureTenantRolePolicies`) restores them on the next restart, so the
> damage is recoverable, but the tenant is broken until then.

```sql
-- 1. Inspect what would be removed before removing anything.
SELECT id, ptype, v0, v1, v2, v3 FROM casbin_rule WHERE v0 LIKE CONCAT('role:%@tenant:', <id>);

-- 2. Purge tenant-domain policies (do this FIRST, while the id is still known).
DELETE FROM casbin_rule WHERE v0 LIKE CONCAT('role:%@tenant:', <id>);

-- 3. Purge tenant-scoped rows across the 18 tables carrying tenant_id
--    (authoritative list: docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md).
DELETE FROM tenant_memberships        WHERE tenant_id = <id>;
DELETE FROM system_user               WHERE tenant_id = <id>;
DELETE FROM system_user_role          WHERE tenant_id = <id>;
DELETE FROM system_user_session       WHERE tenant_id = <id>;
DELETE FROM system_role               WHERE tenant_id = <id>;
DELETE FROM system_role_menu          WHERE tenant_id = <id>;
DELETE FROM system_role_permission    WHERE tenant_id = <id>;
DELETE FROM system_role_data_scope    WHERE tenant_id = <id>;
DELETE FROM system_setting            WHERE tenant_id = <id>;
DELETE FROM system_dept               WHERE tenant_id = <id>;
DELETE FROM system_post               WHERE tenant_id = <id>;
DELETE FROM system_dict_type          WHERE tenant_id = <id>;
DELETE FROM system_dict_item          WHERE tenant_id = <id>;
DELETE FROM system_log_login          WHERE tenant_id = <id>;
DELETE FROM system_log_oper           WHERE tenant_id = <id>;
DELETE FROM system_auth_security_event    WHERE tenant_id = <id>;
DELETE FROM system_auth_mfa_challenge     WHERE tenant_id = <id>;
DELETE FROM permission_role_data_scope_policy WHERE tenant_id = <id>;

-- 4. Finally the master row.
DELETE FROM tenants WHERE id = <id>;
```

**Verification**: both queries must return `0` for the deleted id.

```sql
SELECT COUNT(*) FROM casbin_rule WHERE v0 LIKE CONCAT('role:%@tenant:', <id>);
SELECT COUNT(*) FROM tenants WHERE id = <id>;
```

**Deliberately not done**: the application does **not** perform this purge on soft delete, and no
`PurgeTenant` helper exists. The contract assigns hard delete to ops, and the platform deliberately
has no cross-tenant hard-delete endpoint (contract §7 keeps `tenants` CRUD in the platform console).
Any future `PurgeTenant` must run steps 2–4 in one transaction and be platform-capability gated.

---

### 3. User Cannot See Tenant Data

**Symptom**: User gets empty results when querying tenant-scoped resources

**Possible Causes**:
1. User is not a member of any tenant
2. `platform.tenant_mode` is set to `multi` but user has no tenant assignment
3. Tenant is suspended or deleted

**Diagnosis**:
```sql
-- Check user's tenant memberships
SELECT * FROM tenant_memberships WHERE user_id = <user_id> AND status = 'active';

-- Check tenant mode setting
SELECT * FROM system_setting WHERE setting_key = 'platform.tenant_mode';

-- Check tenant status
SELECT * FROM tenants WHERE id = <tenant_id>;
```

**Solutions**:

**Option 1**: Add user to a tenant
```bash
curl -X POST http://localhost:8080/api/v1/tenants/1/members \
  -H "Authorization: Bearer <token>" \
  -d '{"user_id": 100, "role": "member"}'
```

**Option 2**: Switch back to compat mode (temporary)
```sql
UPDATE system_setting 
SET setting_value = '"compat"' 
WHERE setting_key = 'platform.tenant_mode';
```

---

### 4. Cross-Tenant Data Leakage

**Symptom**: User A in Tenant 1 sees data from Tenant 2

**Diagnosis**:
```sql
-- Check if table has tenant_id column
SHOW COLUMNS FROM system_user LIKE 'tenant_id';

-- Check if data has correct tenant_id
SELECT tenant_id, COUNT(*) FROM system_user GROUP BY tenant_id;
```

**Cause**: Table missing `tenant_id` column or queries not using `WithTenantScope`

**Solution**:
1. Ensure migration 000020 has been run:
```bash
# Check migration status
SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 5;

# If 000020 is missing, run migrations
./pantheon-server migrate up
```

2. Verify service code uses tenant scope:
```go
// CORRECT
db.Scopes(tenant.WithTenantScope(ctx)).Find(&users)

// WRONG
db.Find(&users)
```

---

### 5. Login Fails After Enabling Multi-Tenant Mode

**Symptom**: Users cannot log in after switching from `compat` to `multi`

**Diagnosis**:
```sql
-- Check if users are assigned to tenants
SELECT u.id, u.username, tm.tenant_id
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
WHERE tm.id IS NULL;
```

**Solution**: Migrate existing users to default tenant
```sql
-- Create default tenant if not exists
INSERT INTO tenants (code, name, status, created_at, updated_at)
VALUES ('default', '默认租户', 'active', NOW(), NOW())
ON DUPLICATE KEY UPDATE id=id;

-- Assign all unmapped users to default tenant
INSERT INTO tenant_memberships (tenant_id, user_id, role, status, created_at, updated_at)
SELECT 
  (SELECT id FROM tenants WHERE code = 'default'),
  u.id,
  'member',
  'active',
  NOW(),
  NOW()
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
WHERE tm.id IS NULL;
```

---

### 5b. Login Succeeds but Every Page Fails with `tenant.context.missing`

**Symptom**: A tenant admin signs in successfully (HTTP 200 on `/auth/login`), then almost
every request fails with `{"code":500,"message":"tenant.context.missing"}`. The console looks
signed-in but is broken everywhere — the worst failure shape to self-diagnose, because login
reports success.

**Cause**: soft-deleting a tenant does **not** delete the `system_user` rows it provisioned.
Those rows survive as orphans, and because the global credential lookup tie-breaks on the
lowest id, an orphan can shadow a live tenant's same-named user. Login then succeeds as the
orphan — whose tenant is gone, so no claim resolves — and the resulting claim-less session is
deny-by-default on every tenant-scoped route.

**Diagnosis** — find the shadowing pair:
```sql
SELECT u.id, u.tenant_id, u.username, t.status AS tenant_status
FROM system_user u
JOIN tenants t ON t.id = u.tenant_id
WHERE u.username = '<the username that fails>'
ORDER BY u.id;
```
If an `id` with `tenant_status = 'deleted'` sorts **before** the live row, that is the orphan.
Confirm which row login resolves to:
```sql
SELECT id, tenant_id FROM system_user
WHERE username = '<username>'
  AND (tenant_id = 0 OR tenant_id IN (SELECT id FROM tenants WHERE status <> 'deleted'))
ORDER BY CASE WHEN tenant_id = 0 THEN 0 ELSE 1 END, id ASC;
```
The first row is the login subject. A `tenant_id` whose tenant is `deleted` means the fix is
not in place.

**Fix**: upgrade to the build that (a) excludes deleted-tenant subjects from the credential
lookup and (b) refuses a claim-less tenant-scoped subject at login with `403 tenant.forbidden`
instead of issuing a broken session. No data repair is required — the exclusion is evaluated at
read time, so pre-existing orphans stop shadowing immediately.

**Optional cleanup** (the orphan no longer causes harm once the fix is in place, so this is
hygiene, not a fix): delete the orphan rows for the deleted tenants.
```sql
-- Inspect first: these are rows whose tenant is soft-deleted.
SELECT u.id, u.tenant_id, u.username
FROM system_user u JOIN tenants t ON t.id = u.tenant_id
WHERE t.status = 'deleted';
-- Then delete by explicit id, never by username.
DELETE FROM system_user WHERE id IN (<ids>);
```

**Why not fix it at delete time?** Archiving the orphan's username on tenant delete would leave
rows created *before* that change still shadowing, and it mutates user records that audits may
need. Excluding them from the login candidate set is the read-time equivalent of the same rule
and heals history too.

---

### 5c. Tenant Sign-In Succeeds but Every Page Answers `permission.denied`

**Symptom**: the tenant administrator signs in successfully, then every authorized route
answers `{"code":403,"message":"permission.denied"}`. Unlike issue 5b this is not a session
problem — the session carries the right tenant claim, the role is correct, the permission keys
and menus are all bound. Only the Casbin rules are missing.

**Cause**: the tenant's `role:<key>@tenant:<id>` policies are gone. `ProvisionTenant` writes
them, but `casbin_rule` has no foreign key to `tenants` and the sanctioned hard-delete procedure
(§2b) purges those rows **by hand** — so one over-broad purge, or a tenant created before that
provisioning step existed, leaves a tenant that authenticates but holds no permissions.

**Diagnosis** — the decisive query is the last one:
```sql
-- Roles and bindings should look healthy (that is what makes this confusing).
SELECT id, role_key, status FROM system_role WHERE tenant_id = <id>;
SELECT user_id, role_id, tenant_id FROM system_user_role WHERE tenant_id = <id>;
SELECT COUNT(*) FROM system_role_permission WHERE role_id = <a tenant role id>;
SELECT COUNT(*) FROM system_role_menu       WHERE role_id = <a tenant role id>;

-- The actual gap: expect 6 rows (5 methods for tenant_admin + GET for
-- tenant_member). Zero here is the defect.
SELECT v0, v1, v2 FROM casbin_rule WHERE v0 LIKE CONCAT('role:%@tenant:', <id>);
```

**Fix**: restart the backend. `Service.EnsureTenantRolePolicies` runs at startup and re-derives
the policies for every **active** tenant, logging what it restored:
```
INFO tenant policy repair restored missing Casbin domain policies {"tenants": 2, "policiesRestored": 12}
```
No manual SQL is needed, and the repair is idempotent and additive — it never removes a policy.

**Related**: if the tenant is deliberately meant to be locked out, use **suspension**
(`PUT /system/tenant/:id` with `status=suspended`), not policy deletion. The login gate refuses a
suspended tenant, and the repair respects that by only touching `active` tenants.

---

### 6. OIDC Login Not Showing Tenant Picker

**Symptom**: Users with multiple tenant memberships are auto-logged into one tenant without choice

**Cause**: OAuth callback handler not calling `ListLoginTenantCandidates`

**Status**: Known issue, fix pending (Phase 2.2)

**Workaround**: Use password login for users needing tenant selection

---

### 7. Performance Degradation After Phase 1 Migration

**Symptom**: Queries slower after adding `tenant_id` columns

**Diagnosis**:
```sql
-- Check if indexes exist
SHOW INDEX FROM system_user WHERE Key_name LIKE '%tenant%';
SHOW INDEX FROM system_role WHERE Key_name LIKE '%tenant%';
```

**Solution**: Ensure composite indexes were created
```sql
-- Phase 1 migration should have created these
SHOW CREATE TABLE system_user;  -- Should show uk_system_user_tenant_username
SHOW CREATE TABLE system_role;  -- Should show uk_system_role_tenant_key
```

If indexes are missing, re-run migration 000020.

---

### 8. Tenant Quota Not Enforced

**Symptom**: Users can create unlimited resources despite quota settings

**Status**: Known limitation - Phase 4 not yet implemented

**Workaround**: Monitor tenant resource usage manually
```sql
SELECT 
  t.id,
  t.code,
  COUNT(DISTINCT tm.user_id) as user_count,
  COUNT(DISTINCT r.id) as role_count
FROM tenants t
LEFT JOIN tenant_memberships tm ON tm.tenant_id = t.id AND tm.status = 'active'
LEFT JOIN system_role r ON r.tenant_id = t.id
WHERE t.status = 'active'
GROUP BY t.id, t.code;
```

---

## Diagnostic Queries

### Check Tenant Health

```sql
-- Tenant overview
SELECT 
  t.id,
  t.code,
  t.name,
  t.status,
  COUNT(DISTINCT tm.user_id) as member_count,
  t.created_at
FROM tenants t
LEFT JOIN tenant_memberships tm ON tm.tenant_id = t.id AND tm.status = 'active'
GROUP BY t.id;
```

### Find Orphaned Data

```sql
-- Users without tenant assignment (in multi mode)
SELECT u.id, u.username
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
WHERE tm.id IS NULL;

-- Roles with tenant_id=0 (platform global)
SELECT id, role_key, tenant_id FROM system_role WHERE tenant_id = 0;

-- Credential rows whose tenant has been soft-deleted ("orphans").
-- Harmless once the read-time exclusion is in place, but they are the shape
-- behind issue 5b, so check for them when a login resolves oddly.
SELECT u.id, u.tenant_id, u.username, t.code
FROM system_user u JOIN tenants t ON t.id = u.tenant_id
WHERE t.status = 'deleted';
```

### Verify Tenant Isolation

```sql
-- Check tenant_id distribution in core tables
SELECT 'users' as table_name, tenant_id, COUNT(*) FROM system_user GROUP BY tenant_id
UNION ALL
SELECT 'roles', tenant_id, COUNT(*) FROM system_role GROUP BY tenant_id
UNION ALL
SELECT 'depts', tenant_id, COUNT(*) FROM system_dept GROUP BY tenant_id;
```

---

## Emergency Procedures

### Rollback to Compat Mode

```sql
-- 1. Switch mode
UPDATE system_setting 
SET setting_value = '"compat"' 
WHERE setting_key = 'platform.tenant_mode';

-- 2. Restart application
# Application restart required to reload setting
```

### Rollback Phase 1 Migration

```bash
# Rollback to version 19 (before Phase 1)
./pantheon-server migrate down 20

# WARNING: This removes tenant_id columns and data isolation
# Only use in emergency if Phase 1 causes critical issues
```

---

## Support Contacts

- **Documentation**: `docs/contracts/TENANT_CONTRACT_V1.md`
- **GitHub Issues**: https://github.com/your-org/pantheon-base/issues
- **Migration Runbook**: `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md`
