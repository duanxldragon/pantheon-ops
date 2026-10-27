package tenant

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests cover Service.EnsureTenantRootDepts and provisionTenantRootDept,
// the provisioning + startup repair for a tenant's root department.
//
// Why this has to exist: DeptService.CreateDept rejects `parent_id = 0`
// (`dept.parent.required`) by design — departments must hang under the
// organisation root or an existing department — so a tenant with no root
// department can never create its first one, and therefore never a post either
// (PostService.ensurePostDeptID requires a non-root tenant department). That is
// the state tenant 253 was found in: `system_dept` held rows for tenant 0 only,
// GET /system/dept/tree answered `[]`, and every create attempt failed with no
// way to self-heal from the UI.

// rootDeptRow is the minimal projection the assertions need.
type rootDeptRow struct {
	ID       uint64 `gorm:"column:id"`
	TenantID uint64 `gorm:"column:tenant_id"`
	ParentID uint64 `gorm:"column:parent_id"`
	IsRoot   int    `gorm:"column:is_root"`
	DeptName string `gorm:"column:dept_name"`
}

// rootDeptFor returns the tenant's root department, or gorm.ErrRecordNotFound.
func rootDeptFor(t *testing.T, db *gorm.DB, tenantID uint64) (rootDeptRow, error) {
	t.Helper()
	var row rootDeptRow
	err := db.Table("system_dept").
		Where("tenant_id = ? AND is_root = ?", tenantID, common.StatusFlagYes).
		Order("id asc").
		First(&row).Error
	return row, err
}

// A tenant provisioned without an admin (roles + settings only) must still
// receive its root department, otherwise it can never create one.
func TestProvisionTenant_CreatesRootDept(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 601, "rootdept-provision", tenantctx.TenantStatusActive)

	result, err := service.ProvisionTenant(ProvisionInput{TenantID: 601})
	require.NoError(t, err)
	require.NotZero(t, result.RootDeptID, "provisioning must report the created root dept")

	root, err := rootDeptFor(t, db, 601)
	require.NoError(t, err)
	require.Equal(t, result.RootDeptID, root.ID)
	require.Equal(t, uint64(601), root.TenantID)
	require.Equal(t, uint64(0), root.ParentID, "a root dept has no parent")
	require.Equal(t, common.StatusFlagYes, root.IsRoot)
	require.Equal(t, tenantRootDeptName, root.DeptName)
}

// Re-running provisioning must reuse the existing root, not duplicate it — the
// repair path calls straight into the same helper.
func TestProvisionTenant_RootDeptIsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 602, "rootdept-idempotent", tenantctx.TenantStatusActive)

	first, err := service.ProvisionTenant(ProvisionInput{TenantID: 602})
	require.NoError(t, err)
	second, err := service.ProvisionTenant(ProvisionInput{TenantID: 602})
	require.NoError(t, err)

	require.Equal(t, first.RootDeptID, second.RootDeptID, "second provision must reuse the root")

	var count int64
	require.NoError(t, db.Table("system_dept").
		Where("tenant_id = ? AND is_root = ?", 602, common.StatusFlagYes).
		Count(&count).Error)
	require.EqualValues(t, 1, count, "exactly one root dept per tenant")
}

// The startup repair creates a root for every active tenant that lacks one.
func TestEnsureTenantRootDepts_CreatesMissingRootsForActiveTenants(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 603, "rootdept-repair", tenantctx.TenantStatusActive)

	_, err := rootDeptFor(t, db, 603)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound, "fixture must start with no root dept")

	require.NoError(t, service.EnsureTenantRootDepts())

	root, err := rootDeptFor(t, db, 603)
	require.NoError(t, err)
	require.Equal(t, uint64(603), root.TenantID)
	require.Equal(t, common.StatusFlagYes, root.IsRoot)
}

// Running the repair twice must not add a second root.
func TestEnsureTenantRootDepts_IsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 604, "rootdept-repair-idem", tenantctx.TenantStatusActive)

	require.NoError(t, service.EnsureTenantRootDepts())
	require.NoError(t, service.EnsureTenantRootDepts())

	var count int64
	require.NoError(t, db.Table("system_dept").
		Where("tenant_id = ? AND is_root = ?", 604, common.StatusFlagYes).
		Count(&count).Error)
	require.EqualValues(t, 1, count)
}

// Suspension/archival is the supported way to lock a tenant out, so the repair
// must not provision for a non-active tenant — mirroring EnsureTenantRolePolicies.
func TestEnsureTenantRootDepts_SkipsNonActiveTenants(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedRepairTenant(t, db, 605, "rootdept-suspended", tenantctx.TenantStatusSuspended)
	seedRepairTenant(t, db, 606, "rootdept-archived", tenantctx.TenantStatusArchived)
	seedRepairTenant(t, db, 607, "rootdept-deleted", tenantStatusDeleted)
	seedRepairTenant(t, db, 608, "rootdept-active", tenantctx.TenantStatusActive)

	require.NoError(t, service.EnsureTenantRootDepts())

	for _, id := range []uint64{605, 606, 607} {
		_, err := rootDeptFor(t, db, id)
		require.ErrorIsf(t, err, gorm.ErrRecordNotFound, "tenant %d must not receive a root", id)
	}
	_, err := rootDeptFor(t, db, 608)
	require.NoError(t, err, "the active tenant must be repaired")
}

// The reserved platform/global placeholder (id 0) owns the platform tree, whose
// root is created by the dept module — this repair must not touch it.
func TestEnsureTenantRootDepts_NeverTouchesTheReservedPlaceholder(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	seedReservedPlaceholder(t, db)

	require.NoError(t, service.EnsureTenantRootDepts())

	_, err := rootDeptFor(t, db, tenantctx.PlatformGlobalTenantID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
