#!/bin/bash
# Tenant Health Check Script
# Usage: ./tenant-health-check.sh [tenant_id]

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-pantheon}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-}"

TENANT_ID="${1:-}"

if [ -z "$TENANT_ID" ]; then
    echo "Usage: $0 <tenant_id>"
    echo "Example: $0 1"
    exit 1
fi

echo "=== Tenant Health Check ==="
echo "Tenant ID: $TENANT_ID"
echo "Database: $DB_NAME@$DB_HOST:$DB_PORT"
echo ""

# Check tenant exists
TENANT_EXISTS=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM tenants WHERE id = $TENANT_ID;
")

if [ "$TENANT_EXISTS" -eq 0 ]; then
    echo "❌ Tenant $TENANT_ID does not exist"
    exit 1
fi

echo "✅ Tenant exists"

# Get tenant info
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -e "
SELECT id, code, name, status, plan, created_at
FROM tenants
WHERE id = $TENANT_ID;
"

echo ""
echo "=== Resource Usage ==="

# Count members
MEMBER_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*)
FROM tenant_memberships
WHERE tenant_id = $TENANT_ID AND status = 'active';
")
echo "Members: $MEMBER_COUNT"

# Count roles
ROLE_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*)
FROM system_role
WHERE tenant_id = $TENANT_ID;
")
echo "Roles: $ROLE_COUNT"

# Count depts
DEPT_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*)
FROM system_dept
WHERE tenant_id = $TENANT_ID;
")
echo "Departments: $DEPT_COUNT"

echo ""
echo "=== Data Integrity Checks ==="

# Check for orphaned users
ORPHANED_USERS=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*)
FROM system_user
WHERE tenant_id = $TENANT_ID
AND id NOT IN (
    SELECT user_id
    FROM tenant_memberships
    WHERE tenant_id = $TENANT_ID AND status = 'active'
);
")

if [ "$ORPHANED_USERS" -gt 0 ]; then
    echo "⚠️  Orphaned users: $ORPHANED_USERS"
else
    echo "✅ No orphaned users"
fi

# Check for cross-tenant data leakage
ISOLATION_VIOLATIONS=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*)
FROM system_role r
LEFT JOIN system_user_role ur ON ur.role_id = r.id
WHERE r.tenant_id = $TENANT_ID
AND ur.tenant_id IS NOT NULL
AND ur.tenant_id != r.tenant_id;
")

if [ "$ISOLATION_VIOLATIONS" -gt 0 ]; then
    echo "❌ Isolation violations: $ISOLATION_VIOLATIONS"
else
    echo "✅ No isolation violations"
fi

echo ""
echo "=== Summary ==="

if [ "$ORPHANED_USERS" -eq 0 ] && [ "$ISOLATION_VIOLATIONS" -eq 0 ]; then
    echo "✅ Tenant health: HEALTHY"
    exit 0
else
    echo "⚠️  Tenant health: ISSUES DETECTED"
    exit 1
fi
