#!/bin/bash
# Tenant Data Export Tool
# Exports all tenant-scoped data to JSON format for backup/migration

set -e

DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-3306}"
DB_NAME="${DB_NAME:-pantheon}"
DB_USER="${DB_USER:-root}"
DB_PASS="${DB_PASS:-}"

TENANT_ID="${1:-}"
OUTPUT_DIR="${2:-./tenant-exports}"

if [ -z "$TENANT_ID" ]; then
    echo "Usage: $0 <tenant_id> [output_dir]"
    echo "Example: $0 1 ./exports"
    exit 1
fi

# Create output directory
mkdir -p "$OUTPUT_DIR"
EXPORT_FILE="$OUTPUT_DIR/tenant_${TENANT_ID}_$(date +%Y%m%d_%H%M%S).json"

echo "=== Tenant Data Export ==="
echo "Tenant ID: $TENANT_ID"
echo "Output: $EXPORT_FILE"
echo ""

# Check tenant exists
TENANT_EXISTS=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM tenants WHERE id = $TENANT_ID;
")

if [ "$TENANT_EXISTS" -eq 0 ]; then
    echo "Error: Tenant $TENANT_ID does not exist"
    exit 1
fi

# Start JSON output
cat > "$EXPORT_FILE" << 'HEADER'
{
  "export_version": "1.0",
  "export_timestamp": "TIMESTAMP_PLACEHOLDER",
  "tenant": TENANT_PLACEHOLDER,
  "data": {
HEADER

# Replace placeholders
TIMESTAMP=$(date -Iseconds)
sed -i "s/TIMESTAMP_PLACEHOLDER/$TIMESTAMP/" "$EXPORT_FILE"

# Export tenant info
TENANT_JSON=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_OBJECT(
    'id', id,
    'code', code,
    'name', name,
    'status', status,
    'plan', plan,
    'created_at', created_at
) FROM tenants WHERE id = $TENANT_ID;
")
sed -i "s/TENANT_PLACEHOLDER/$TENANT_JSON/" "$EXPORT_FILE"

# Export users
echo "Exporting users..."
USER_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM system_user WHERE tenant_id = $TENANT_ID;
")
echo "  Found $USER_COUNT users"

mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_ARRAYAGG(JSON_OBJECT(
    'id', id,
    'username', username,
    'nickname', nickname,
    'email', email,
    'phone', phone,
    'dept_id', dept_id,
    'status', status,
    'created_at', created_at
))
FROM system_user WHERE tenant_id = $TENANT_ID;
" | sed 's/NULL/[]/' >> "$EXPORT_FILE"
echo ',' >> "$EXPORT_FILE"

# Export roles
echo "Exporting roles..."
ROLE_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM system_role WHERE tenant_id = $TENANT_ID;
")
echo "  Found $ROLE_COUNT roles"

echo '    "roles": ' >> "$EXPORT_FILE"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_ARRAYAGG(JSON_OBJECT(
    'id', id,
    'role_key', role_key,
    'role_name', role_name,
    'sort', sort,
    'status', status,
    'created_at', created_at
))
FROM system_role WHERE tenant_id = $TENANT_ID;
" | sed 's/NULL/[]/' >> "$EXPORT_FILE"
echo ',' >> "$EXPORT_FILE"

# Export depts
echo "Exporting departments..."
DEPT_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM system_dept WHERE tenant_id = $TENANT_ID;
")
echo "  Found $DEPT_COUNT departments"

echo '    "depts": ' >> "$EXPORT_FILE"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_ARRAYAGG(JSON_OBJECT(
    'id', id,
    'parent_id', parent_id,
    'dept_name', dept_name,
    'dept_code', dept_code,
    'sort', sort,
    'status', status
))
FROM system_dept WHERE tenant_id = $TENANT_ID;
" | sed 's/NULL/[]/' >> "$EXPORT_FILE"
echo ',' >> "$EXPORT_FILE"

# Export menus
echo "Exporting menus..."
MENU_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM system_menu WHERE tenant_id = $TENANT_ID;
")
echo "  Found $MENU_COUNT menus"

echo '    "menus": ' >> "$EXPORT_FILE"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_ARRAYAGG(JSON_OBJECT(
    'id', id,
    'parent_id', parent_id,
    'menu_name', menu_name,
    'path', path,
    'component', component,
    'sort', sort,
    'visible', visible
))
FROM system_menu WHERE tenant_id = $TENANT_ID;
" | sed 's/NULL/[]/' >> "$EXPORT_FILE"
echo ',' >> "$EXPORT_FILE"

# Export memberships
echo "Exporting memberships..."
MEMBER_COUNT=$(mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT COUNT(*) FROM tenant_memberships WHERE tenant_id = $TENANT_ID AND status = 'active';
")
echo "  Found $MEMBER_COUNT members"

echo '    "memberships": ' >> "$EXPORT_FILE"
mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" -D"$DB_NAME" -se "
SELECT JSON_ARRAYAGG(JSON_OBJECT(
    'user_id', user_id,
    'role', role,
    'status', status,
    'created_at', created_at
))
FROM tenant_memberships WHERE tenant_id = $TENANT_ID;
" | sed 's/NULL/[]/' >> "$EXPORT_FILE"

# Close JSON
cat >> "$EXPORT_FILE" << 'FOOTER'
  },
  "statistics": {
    "users": USER_COUNT_PLACEHOLDER,
    "roles": ROLE_COUNT_PLACEHOLDER,
    "depts": DEPT_COUNT_PLACEHOLDER,
    "menus": MENU_COUNT_PLACEHOLDER,
    "members": MEMBER_COUNT_PLACEHOLDER
  }
}
FOOTER

# Replace statistics placeholders
sed -i "s/USER_COUNT_PLACEHOLDER/$USER_COUNT/" "$EXPORT_FILE"
sed -i "s/ROLE_COUNT_PLACEHOLDER/$ROLE_COUNT/" "$EXPORT_FILE"
sed -i "s/DEPT_COUNT_PLACEHOLDER/$DEPT_COUNT/" "$EXPORT_FILE"
sed -i "s/MENU_COUNT_PLACEHOLDER/$MENU_COUNT/" "$EXPORT_FILE"
sed -i "s/MEMBER_COUNT_PLACEHOLDER/$MEMBER_COUNT/" "$EXPORT_FILE"

# Fix JSON formatting (remove trailing comma before closing brace)
sed -i 's/,\([[:space:]]*}\)/\1/g' "$EXPORT_FILE"

echo ""
echo "=== Export Complete ==="
echo "File: $EXPORT_FILE"
echo "Size: $(du -h "$EXPORT_FILE" | cut -f1)"
echo ""
echo "Statistics:"
echo "  Users: $USER_COUNT"
echo "  Roles: $ROLE_COUNT"
echo "  Departments: $DEPT_COUNT"
echo "  Menus: $MENU_COUNT"
echo "  Members: $MEMBER_COUNT"
echo ""
echo "Verify JSON validity:"
echo "  python -m json.tool $EXPORT_FILE > /dev/null && echo 'Valid JSON' || echo 'Invalid JSON'"
