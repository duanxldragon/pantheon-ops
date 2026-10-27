# Compat to Multi-Tenant Upgrade Guide

## Overview

This guide walks you through upgrading from `compat` mode (single-tenant, tenant_id=0) to `multi` mode (true multi-tenancy) in Pantheon Base.

**Estimated Downtime**: 5-15 minutes (depends on data volume)
**Quick Rollback Time**: under 1 minute for switching back to `compat`; full database restore time depends on database size.
**Prerequisites**: Tenant schema migration version 20 or later applied, default tenant created

---

## Pre-Upgrade Checklist

### 1. Verify Migration Status

```bash
# Confirm tenant schema migration version 20 or later is recorded and clean
mysql -u root -p pantheon -e "SELECT version, dirty FROM schema_migrations;"

# Verify all core tables have tenant_id column
mysql -u root -p pantheon << 'EOF'
SELECT table_name, column_name
FROM information_schema.columns
WHERE table_schema = DATABASE()
  AND column_name = 'tenant_id'
  AND table_name IN ('system_user', 'system_role', 'system_menu', 'system_dept')
ORDER BY table_name;
EOF
```

**Expected Output**: 4 rows (user, role, menu, dept)

### 2. Backup Database

```bash
# Full backup
mysqldump -u root -p pantheon > /backup/pantheon_before_multi_$(date +%Y%m%d_%H%M%S).sql

# Verify backup
ls -lh /backup/pantheon_before_multi_*.sql
```

### 3. Check Current Mode

```bash
mysql -u root -p pantheon -e "SELECT setting_value FROM system_setting WHERE setting_key = 'platform.tenant_mode';"
```

Should return: `"compat"`

---

## Upgrade Steps

### Step 1: Create Default Tenant (if not exists)

```sql
-- Connect to database
mysql -u root -p pantheon

-- Check if default tenant exists
SELECT id, code, name, status FROM tenants WHERE code = 'default';

-- If not exists, create it
INSERT INTO tenants (code, name, status, plan, created_at, updated_at)
VALUES ('default', '默认租户', 'active', 'enterprise', NOW(), NOW());

-- Get tenant ID
SELECT @default_tenant_id := id FROM tenants WHERE code = 'default';
```

### Step 2: Migrate Existing Users to Default Tenant

```sql
-- Assign all users without tenant membership to default tenant
INSERT INTO tenant_memberships (tenant_id, user_id, role, status, created_at, updated_at)
SELECT 
    (SELECT id FROM tenants WHERE code = 'default'),
    u.id,
    CASE WHEN u.username = 'admin' THEN 'owner' ELSE 'member' END,
    'active',
    NOW(),
    NOW()
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
WHERE tm.id IS NULL;

-- Verify migration
SELECT COUNT(*) as unmapped_users
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
WHERE tm.id IS NULL;
```

**Expected**: `unmapped_users = 0`

### Step 3: Update Data tenant_id (compat → tenant 1)

**IMPORTANT**: This step updates all tenant_id=0 rows to belong to the default tenant.

```sql
-- Set default tenant ID variable
SELECT @default_tenant_id := id FROM tenants WHERE code = 'default';

-- Update users
UPDATE system_user SET tenant_id = @default_tenant_id WHERE tenant_id = 0;

-- Update roles
UPDATE system_role SET tenant_id = @default_tenant_id WHERE tenant_id = 0;

-- Update menus
UPDATE system_menu SET tenant_id = @default_tenant_id WHERE tenant_id = 0;

-- Update depts
UPDATE system_dept SET tenant_id = @default_tenant_id WHERE tenant_id = 0;

-- Update settings (except platform.tenant_mode)
UPDATE system_setting 
SET tenant_id = @default_tenant_id 
WHERE tenant_id = 0 
  AND setting_key != 'platform.tenant_mode';

-- Update relationship tables
UPDATE system_user_role SET tenant_id = @default_tenant_id WHERE tenant_id = 0;
UPDATE system_role_menu SET tenant_id = @default_tenant_id WHERE tenant_id = 0;
UPDATE system_role_permission SET tenant_id = @default_tenant_id WHERE tenant_id = 0;

-- Verify no orphaned data (except platform-global settings)
SELECT 
    'users' as table_name, COUNT(*) as compat_rows FROM system_user WHERE tenant_id = 0
UNION ALL
SELECT 'roles', COUNT(*) FROM system_role WHERE tenant_id = 0
UNION ALL
SELECT 'menus', COUNT(*) FROM system_menu WHERE tenant_id = 0
UNION ALL
SELECT 'depts', COUNT(*) FROM system_dept WHERE tenant_id = 0;
```

**Expected**: All counts should be 0

### Step 4: Switch to Multi Mode

```sql
-- Update tenant mode setting
UPDATE system_setting 
SET setting_value = '"multi"' 
WHERE setting_key = 'platform.tenant_mode';

-- Verify
SELECT setting_value FROM system_setting WHERE setting_key = 'platform.tenant_mode';
```

**Expected**: `"multi"`

### Step 5: Restart Application

```bash
# Restart the application to reload settings
systemctl restart pantheon-server

# Or if using Docker
docker-compose restart pantheon-server

# Check application logs
tail -f /var/log/pantheon/server.log | grep -i tenant
```

Look for: `Tenant mode: multi`

---

## Post-Upgrade Verification

### 1. Verify Tenant Mode

```bash
curl http://localhost:8080/api/v1/system/settings/platform.tenant_mode \
  -H "Authorization: Bearer <admin-token>"
```

**Expected**: `{"value": "multi"}`

### 2. Test User Login

```bash
# Admin should still be able to login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123",
    "tenant_id": 1
  }'
```

**Expected**: 200 OK with JWT token

### 3. Verify Tenant Isolation

```sql
-- Connect as admin to tenant 1
-- Should see only tenant 1 data

SELECT tenant_id, COUNT(*) FROM system_user GROUP BY tenant_id;
SELECT tenant_id, COUNT(*) FROM system_role GROUP BY tenant_id;
```

### 4. Test Tenant API

```bash
# List tenants (requires platform_ops role)
curl http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer <admin-token>"

# Get my tenants
curl http://localhost:8080/api/v1/tenants/my \
  -H "Authorization: Bearer <admin-token>"
```

### 5. Run Health Check

```bash
./scripts/tenant-health-check.sh 1
```

**Expected**: `✅ Tenant health: HEALTHY`

---

## Rollback Procedure

If issues occur, rollback to compat mode:

### Quick Rollback (< 1 minute)

```sql
-- Switch back to compat mode
UPDATE system_setting 
SET setting_value = '"compat"' 
WHERE setting_key = 'platform.tenant_mode';

-- Restart application
systemctl restart pantheon-server
```

**Note**: Data remains assigned to the created default tenant, but compat mode ignores tenant_id filtering. This is a temporary application-mode rollback, not a schema rollback.

### Full Rollback (restore from backup)

A full restore replaces all database contents with the pre-upgrade snapshot and takes time proportional to database size. Stop all application writers first and verify the backup file before restoring. Do not use an invented `migrate down` command.

```bash
systemctl stop pantheon-server
mysql -u root -p pantheon < "$PANTHEON_BACKUP_FILE"
systemctl start pantheon-server
```

---

## Troubleshooting

### Issue 1: Users Cannot Login After Upgrade

**Symptom**: `401 Unauthorized` or `tenant not found`

**Diagnosis**:
```sql
-- Check if users have tenant memberships
SELECT u.id, u.username, tm.tenant_id
FROM system_user u
LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active';
```

**Fix**: Re-run Step 2 (Migrate Users)

### Issue 2: Data Not Visible in Multi Mode

**Symptom**: Empty results when querying resources

**Diagnosis**:
```sql
-- Check data distribution
SELECT tenant_id, COUNT(*) FROM system_user GROUP BY tenant_id;
```

**Fix**: If all data still has tenant_id=0, re-run Step 3 (Update Data)

### Issue 3: Performance Degradation

**Symptom**: Slow queries after upgrade

**Diagnosis**:
```sql
-- Check if indexes exist
SHOW INDEX FROM system_user WHERE Key_name LIKE '%tenant%';
```

**Fix**: Apply the current embedded migration set (version 20 or later) using the migration runner; do not replay one migration manually.

---

## Post-Upgrade Tasks

### 1. Update Application Configuration

Edit `config.yaml`:
```yaml
platform:
  tenant_mode: multi  # Update from compat
```

### 2. Train Platform Operators

- How to create new tenants
- How to assign users to tenants
- How to monitor tenant quotas

### 3. Setup Monitoring

```bash
# Add Prometheus metrics
curl http://localhost:8080/metrics | grep tenant_count
```

### 4. Document Tenant Policies

Create internal documentation:
- Tenant creation approval process
- Tenant suspension criteria
- Data retention policies

---

## FAQ

**Q: Can I test multi mode without affecting production?**  
A: Yes, create a staging environment with a copy of production data and test there first.

**Q: How long does the upgrade take?**  
A: Typically 5-15 minutes. Step 3 (data migration) is the longest, proportional to data volume.

**Q: Can I switch back and forth between compat and multi?**  
A: Yes, but only switch to compat temporarily for troubleshooting. Don't create new data in compat after going multi.

**Q: Will existing JWT tokens still work?**  
A: Yes, if they have tenant_id=0 in claims. But users should re-login to get tenant_id=1 tokens.

**Q: What happens to scheduled jobs/cron?**  
A: Review job code to ensure they specify a tenant context or run in platform-global mode.

**Q: Can I create a second tenant immediately after upgrade?**  
A: Yes, use `POST /api/v1/tenants` API.

---

## Support

- **Troubleshooting**: `docs/runbooks/TENANT_TROUBLESHOOTING.md`
- **API Reference**: `docs/api/TENANT_API.md`
- **Contract**: `docs/contracts/TENANT_CONTRACT_V1.md`
