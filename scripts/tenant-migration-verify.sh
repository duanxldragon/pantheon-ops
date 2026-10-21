#!/bin/bash
# Tenant Data Migration Verification Script
# Validates Phase 1 migration and data integrity before switching to multi mode

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-pantheon}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0

echo "=== Tenant Migration Verification ==="
echo "Database: $DB_NAME@$DB_HOST:$DB_PORT"
echo ""

# Helper function to run SQL query
run_query() {
    mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "$1" 2>/dev/null
}

# Test 1: Check migration 000020 applied
echo "Test 1: Verifying migration 000020 status..."
MIGRATION_STATE=$(run_query "SELECT CONCAT(version, ':', dirty) FROM schema_migrations LIMIT 1;")
MIGRATION_VERSION="${MIGRATION_STATE%%:*}"
MIGRATION_DIRTY="${MIGRATION_STATE##*:}"
if [ -n "$MIGRATION_VERSION" ] && [ "$MIGRATION_VERSION" -ge 20 ] && [ "$MIGRATION_DIRTY" -eq 0 ]; then
    echo -e "${GREEN}✅ Tenant schema migrations applied (version $MIGRATION_VERSION)${NC}"
else
    echo -e "${RED}❌ Tenant schema migration state is not ready (version=${MIGRATION_VERSION:-missing}, dirty=${MIGRATION_DIRTY:-unknown})${NC}"
    echo "   Run: PANTHEON_DSN=... go run ./cmd/tenantmigration up"
    ERRORS=$((ERRORS + 1))
fi

# Test 2: Check all core tables have tenant_id
echo ""
echo "Test 2: Checking tenant_id columns in core tables..."
REQUIRED_TABLES=("system_user" "system_role" "system_menu" "system_dept" "system_setting")
for table in "${REQUIRED_TABLES[@]}"; do
    HAS_COLUMN=$(run_query "
        SELECT COUNT(*)
        FROM information_schema.columns
        WHERE table_schema = '$DB_NAME'
          AND table_name = '$table'
          AND column_name = 'tenant_id';
    ")
    if [ "$HAS_COLUMN" -eq 1 ]; then
        echo -e "${GREEN}✅ $table has tenant_id${NC}"
    else
        echo -e "${RED}❌ $table MISSING tenant_id${NC}"
        ERRORS=$((ERRORS + 1))
    fi
done

# Test 3: Check composite unique indexes
echo ""
echo "Test 3: Verifying composite unique indexes..."
INDEX_CHECKS=(
    "system_user:uk_system_user_tenant_username"
    "system_role:uk_system_role_tenant_key"
    "system_dept:uk_system_dept_tenant_code"
)
for check in "${INDEX_CHECKS[@]}"; do
    table="${check%%:*}"
    index="${check##*:}"
    INDEX_EXISTS=$(run_query "
        SELECT COUNT(*)
        FROM information_schema.statistics
        WHERE table_schema = '$DB_NAME'
          AND table_name = '$table'
          AND index_name = '$index';
    ")
    if [ "$INDEX_EXISTS" -gt 0 ]; then
        echo -e "${GREEN}✅ $table has index $index${NC}"
    else
        echo -e "${YELLOW}⚠️  $table MISSING index $index${NC}"
        WARNINGS=$((WARNINGS + 1))
    fi
done

# Test 4: Check default tenant exists
echo ""
echo "Test 4: Checking default tenant..."
DEFAULT_TENANT_EXISTS=$(run_query "SELECT COUNT(*) FROM tenants WHERE code = 'default';")
if [ "$DEFAULT_TENANT_EXISTS" -eq 1 ]; then
    echo -e "${GREEN}✅ Default tenant exists${NC}"
    run_query "SELECT id, code, name, status FROM tenants WHERE code = 'default';"
else
    echo -e "${YELLOW}⚠️  Default tenant does not exist${NC}"
    echo "   Run: follow Step 1 in docs/migrations/COMPAT_TO_MULTI_UPGRADE.md."
    WARNINGS=$((WARNINGS + 1))
fi

# Test 5: Check for unmapped users (users without tenant membership)
echo ""
echo "Test 5: Checking unmapped users..."
UNMAPPED_USERS=$(run_query "
    SELECT COUNT(*)
    FROM system_user u
    LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
    WHERE tm.id IS NULL;
")
if [ "$UNMAPPED_USERS" -eq 0 ]; then
    echo -e "${GREEN}✅ All users mapped to tenants${NC}"
else
    echo -e "${YELLOW}⚠️  $UNMAPPED_USERS users not mapped to any tenant${NC}"
    echo "   These users will not be able to login in multi mode"
    echo "   Run migration script to assign them to default tenant"
    WARNINGS=$((WARNINGS + 1))
fi

# Test 6: Check data distribution
echo ""
echo "Test 6: Analyzing data distribution..."
echo "Users per tenant:"
run_query "
    SELECT
        CASE WHEN tenant_id = 0 THEN 'compat (0)' ELSE CONCAT('tenant ', tenant_id) END as tenant,
        COUNT(*) as count
    FROM system_user
    GROUP BY tenant_id
    ORDER BY tenant_id;
"

echo ""
echo "Roles per tenant:"
run_query "
    SELECT
        CASE WHEN tenant_id = 0 THEN 'compat (0)' ELSE CONCAT('tenant ', tenant_id) END as tenant,
        COUNT(*) as count
    FROM system_role
    GROUP BY tenant_id
    ORDER BY tenant_id;
"

# Test 7: Check for isolation violations
echo ""
echo "Test 7: Checking data isolation integrity..."
ISOLATION_VIOLATIONS=$(run_query "
    SELECT COUNT(*)
    FROM system_role r
    INNER JOIN system_user_role ur ON ur.role_id = r.id
    WHERE r.tenant_id != ur.tenant_id
      AND r.tenant_id != 0
      AND ur.tenant_id != 0;
")
if [ "$ISOLATION_VIOLATIONS" -eq 0 ]; then
    echo -e "${GREEN}✅ No isolation violations${NC}"
else
    echo -e "${RED}❌ $ISOLATION_VIOLATIONS isolation violations detected${NC}"
    echo "   Cross-tenant data references found"
    ERRORS=$((ERRORS + 1))
fi

# Test 8: Check tenant mode setting
echo ""
echo "Test 8: Checking current tenant mode..."
TENANT_MODE=$(run_query "SELECT setting_value FROM system_setting WHERE setting_key = 'platform.tenant_mode';" | tr -d '"')
if [ "$TENANT_MODE" = "compat" ]; then
    echo -e "${GREEN}✅ Current mode: compat${NC}"
    echo "   Safe to upgrade to multi mode"
elif [ "$TENANT_MODE" = "multi" ]; then
    echo -e "${GREEN}✅ Current mode: multi${NC}"
    echo "   Already in multi-tenant mode"
else
    echo -e "${YELLOW}⚠️  Current mode: $TENANT_MODE (unknown)${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# Test 9: Check for orphaned relationship data
echo ""
echo "Test 9: Checking orphaned relationship data..."
ORPHANED_USER_ROLES=$(run_query "
    SELECT COUNT(*)
    FROM system_user_role ur
    LEFT JOIN system_user u ON u.id = ur.user_id
    LEFT JOIN system_role r ON r.id = ur.role_id
    WHERE u.id IS NULL OR r.id IS NULL;
")
if [ "$ORPHANED_USER_ROLES" -eq 0 ]; then
    echo -e "${GREEN}✅ No orphaned user_role records${NC}"
else
    echo -e "${YELLOW}⚠️  $ORPHANED_USER_ROLES orphaned user_role records${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# Test 10: Check admin user tenant assignment
echo ""
echo "Test 10: Verifying admin user tenant assignment..."
ADMIN_TENANT=$(run_query "
    SELECT tm.tenant_id
    FROM system_user u
    INNER JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
    WHERE u.username = 'admin'
    LIMIT 1;
")
if [ -n "$ADMIN_TENANT" ]; then
    echo -e "${GREEN}✅ Admin user assigned to tenant $ADMIN_TENANT${NC}"
else
    echo -e "${RED}❌ Admin user NOT assigned to any tenant${NC}"
    echo "   Admin will not be able to login in multi mode"
    ERRORS=$((ERRORS + 1))
fi

# Summary
echo ""
echo "=== Verification Summary ==="
if [ "$ERRORS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
    echo -e "${GREEN}✅ All checks passed${NC}"
    echo "   Safe to proceed with compat → multi upgrade"
    exit 0
elif [ "$ERRORS" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  $WARNINGS warnings${NC}"
    echo "   Review warnings before proceeding"
    exit 0
else
    echo -e "${RED}❌ $ERRORS errors, $WARNINGS warnings${NC}"
    echo "   Fix errors before proceeding with upgrade"
    exit 1
fi
