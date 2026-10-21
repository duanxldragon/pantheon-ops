# Tenant Troubleshooting Guide

## Common Issues and Solutions

### 1. Tenant Creation Fails

**Symptom**: `POST /api/v1/tenants` returns 409 Conflict

**Cause**: Tenant code already exists

**Solution**:
```bash
# Check existing tenants
curl -X GET http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <token>"

# Use a different tenant code
```

---

### 2. Cannot Delete Tenant

**Symptom**: `DELETE /api/v1/tenants/:id` returns 400 "tenant has active members"

**Cause**: Tenant still has active members

**Solution**:
```bash
# List tenant members
curl -X GET http://localhost:8080/api/v1/tenants/1/members \
  -H "Authorization: Bearer <token>"

# Remove all members first
curl -X DELETE http://localhost:8080/api/v1/tenants/1/members/100 \
  -H "Authorization: Bearer <token>"

# Then delete tenant
curl -X DELETE http://localhost:8080/api/v1/tenants/1 \
  -H "Authorization: Bearer <token>"
```

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
