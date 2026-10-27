package tenant

import (
	"fmt"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests pin the `code` lifecycle: it is UNIQUE (contract §2.1), so a
// soft-deleted row would keep it reserved forever unless the identifier is
// archived first. That is the rule docs/designs/BACKEND.md sets for every
// soft-delete + unique-key entity, and the state the console exposed as an
// unexplained `tenant.conflict` in front of an empty tenant list.

// TestDeleteTenant_ReleasesCodeForReuse is the operator-facing promise: after a
// tenant is deleted, its code can be used again.
func TestDeleteTenant_ReleasesCodeForReuse(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	first, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme", Plan: "free"})
	require.NoError(t, err)

	require.NoError(t, service.DeleteTenant(first.ID))

	// The same code must now be creatable. Before the fix this returned
	// ErrTenantCodeExists, because the UNIQUE index still held the value on the
	// hidden row.
	second, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme Reloaded", Plan: "free"})
	require.NoError(t, err)
	require.NotNil(t, second)

	// A fresh row, not a resurrected one: reusing the id would silently
	// reattach the deleted tenant's sessions and audit trail to the new one.
	assert.NotEqual(t, first.ID, second.ID)
	assert.Equal(t, "acme", second.Code)

	// The deleted row survives under an archived code, so history stays
	// addressable by id at the storage layer.
	deleted := mustLoadTenantRow(t, db, first.ID)
	assert.Equal(t, tenantStatusDeleted, deleted.Status)
	assert.Equal(t, fmt.Sprintf("%s%d", deletedTenantCodePrefix, first.ID), deleted.Code)
}

// TestDeleteTenant_ArchivedCodeIsOutsideTheOperatorNamespace documents why the
// archived form is safe to hand out: the prefix starts with an underscore, which
// the contract's code grammar (`^[a-z][a-z0-9-]{1,62}$`, §2.1) can never
// produce, so an archived value is not something an operator would type.
func TestDeleteTenant_ArchivedCodeIsOutsideTheOperatorNamespace(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "beta", Name: "Beta", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	archived := fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID)
	assert.True(t, len(archived) > 0 && archived[0] == '_', "archived code must start with '_': %s", archived)
}

// TestAllocateDeletedTenantCode_FallsBackWhenNameIsTaken covers the one way the
// deterministic candidate can be unavailable — some other row already holds it —
// which must degrade to a unique code rather than fail the delete.
func TestAllocateDeletedTenantCode_FallsBackWhenNameIsTaken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	const squattedID = 7
	const squatterID = 99
	candidate := fmt.Sprintf("%s%d", deletedTenantCodePrefix, squattedID)
	require.NoError(t, db.Exec(
		"INSERT INTO tenants (id, code, name, status, plan) VALUES (?, ?, ?, ?, ?)",
		squatterID, candidate, "Squatter", tenant.TenantStatusActive, "free",
	).Error)

	var got string
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		var err error
		got, err = service.allocateDeletedTenantCode(tx, squattedID)
		return err
	}))

	assert.NotEqual(t, candidate, got)
	assert.True(t, strings.HasPrefix(got, deletedTenantCodePrefix), "got %q", got)
}

// TestCreateTenant_CodeHeldByLiveTenantIsTaken keeps the two conflict causes
// apart: a live tenant holding the code is a plain duplicate, not a reserved
// one, because the remedy differs.
func TestCreateTenant_CodeHeldByLiveTenantIsTaken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	_, err := service.CreateTenant(CreateTenantDTO{Code: "gamma", Name: "Gamma", Plan: "free"})
	require.NoError(t, err)

	_, err = service.CreateTenant(CreateTenantDTO{Code: "gamma", Name: "Gamma Again", Plan: "free"})
	assert.Equal(t, ErrTenantCodeExists, err)
}

// TestCreateTenant_CodeHeldByDeletedTenantIsReserved reproduces the exact state
// an upgrade produces: rows soft-deleted by the old code path, which left the
// original `code` in place. The create must say "reserved" (so the operator
// learns a hidden row is in the way) rather than the generic duplicate.
func TestCreateTenant_CodeHeldByDeletedTenantIsReserved(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy", Plan: "free"})
	require.NoError(t, err)

	// Write the pre-fix state directly: status flipped, code untouched.
	require.NoError(t, db.Model(&Tenant{}).Where("id = ?", created.ID).
		Update("status", tenantStatusDeleted).Error)

	_, err = service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy Again", Plan: "free"})
	assert.Equal(t, ErrTenantCodeReserved, err)
}

// TestReleaseDeletedTenantCodes_RepairsLegacyRows covers the startup repair
// that docs/designs/BACKEND.md requires for historical rows.
func TestReleaseDeletedTenantCodes_RepairsLegacyRows(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, db.Model(&Tenant{}).Where("id = ?", created.ID).
		Update("status", tenantStatusDeleted).Error)

	require.NoError(t, service.ReleaseDeletedTenantCodes())

	repaired := mustLoadTenantRow(t, db, created.ID)
	assert.Equal(t, fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID), repaired.Code)

	// ... and the code is usable again.
	_, err = service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy Again", Plan: "free"})
	require.NoError(t, err)
}

// TestReleaseDeletedTenantCodes_IsIdempotent pins that the repair can run on
// every startup without rewriting rows it has already handled.
func TestReleaseDeletedTenantCodes_IsIdempotent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	require.NoError(t, service.ReleaseDeletedTenantCodes())
	afterFirst := mustLoadTenantRow(t, db, created.ID)

	require.NoError(t, service.ReleaseDeletedTenantCodes())
	afterSecond := mustLoadTenantRow(t, db, created.ID)

	assert.Equal(t, afterFirst.Code, afterSecond.Code)
	assert.Equal(t, fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID), afterSecond.Code)
}

// TestReleaseDeletedTenantCodes_NeverTouchesTheReservedPlaceholder is the
// safety-critical case: the platform/global row (contract §2.3) is not a tenant,
// so the repair must ignore it even if it were somehow marked deleted.
func TestReleaseDeletedTenantCodes_NeverTouchesTheReservedPlaceholder(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	seedReservedPlaceholder(t, db)

	// Force the pathological state: the placeholder marked deleted.
	require.NoError(t, db.Exec(
		"UPDATE tenants SET status = ? WHERE id = ?", tenantStatusDeleted, platformGlobalTenantID,
	).Error)

	require.NoError(t, service.ReleaseDeletedTenantCodes())

	var code string
	require.NoError(t, db.Raw(
		"SELECT code FROM tenants WHERE id = ?", platformGlobalTenantID,
	).Scan(&code).Error)
	assert.Equal(t, "__global__", code, "the reserved placeholder must keep its code")
}

// TestBootstrap_RunsTheRepair pins that the repair is actually reachable from
// the startup path, not just callable in isolation.
func TestBootstrap_RunsTheRepair(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, db.Model(&Tenant{}).Where("id = ?", created.ID).
		Update("status", tenantStatusDeleted).Error)

	require.NoError(t, service.Bootstrap())

	repaired := mustLoadTenantRow(t, db, created.ID)
	assert.Equal(t, fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID), repaired.Code)
}

// TestCreateTenant_ReusedCodeIsFullyProvisioned checks that reuse is not just a
// row insert: the recreated tenant must be usable, i.e. provisioned like any
// other create.
//
// The first tenant is created without an administrator on purpose. A provisioned
// tenant owns an active membership, and contract §2.2 forbids deleting a tenant
// that still has active members — so a deletable fixture is one that was never
// provisioned. That asymmetry is itself worth pinning: it is why the console's
// delete button is only reachable for empty tenants.
func TestCreateTenant_ReusedCodeIsFullyProvisioned(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	first, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(first.ID))

	second, err := service.CreateTenant(CreateTenantDTO{
		Code: "acme", Name: "Acme Reloaded", Plan: "free",
		AdminUsername: "acme-admin", AdminPassword: "abc12345",
	})
	require.NoError(t, err)

	// Provisioning must have run for the new tenant: roles + an owner
	// membership for its administrator.
	var roleCount int64
	require.NoError(t, db.Table("system_role").Where("tenant_id = ?", second.ID).Count(&roleCount).Error)
	assert.Greater(t, roleCount, int64(0))

	var membershipCount int64
	require.NoError(t, db.Model(&Membership{}).
		Where("tenant_id = ? AND status = ?", second.ID, "active").Count(&membershipCount).Error)
	assert.Greater(t, membershipCount, int64(0))

	// The deleted tenant must not have been revived by the reuse.
	var staleCount int64
	require.NoError(t, db.Model(&Membership{}).
		Where("tenant_id = ? AND status = ?", first.ID, "active").Count(&staleCount).Error)
	assert.Equal(t, int64(0), staleCount)
}

// TestListTenants_DeletedRowKeepsItsName makes sure the archived code does not
// erase the only human-readable handle on a deleted tenant.
func TestListTenants_DeletedRowKeepsItsName(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme Corp", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	view, err := service.ListTenants(Filter{Status: tenantStatusDeleted})
	require.NoError(t, err)
	require.Len(t, view.Items, 1)
	assert.Equal(t, "Acme Corp", view.Items[0].Name)
	assert.Equal(t, fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID), view.Items[0].Code)
}

// TestGetTenantByCode_ResolvesOnlyLiveCodes pins that a released code no longer
// resolves to the deleted row, so lookups by code cannot silently land on a
// tombstone.
func TestGetTenantByCode_ResolvesOnlyLiveCodes(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	_, err = service.GetTenantByCode("acme")
	assert.Equal(t, ErrTenantNotFound, err)
}

// TestTenantStatusDeletedValueMatchesContract keeps the marker literal honest:
// the repair and the list filter both key off this exact string.
func TestTenantStatusDeletedValueMatchesContract(t *testing.T) {
	assert.Equal(t, "deleted", tenantStatusDeleted)
	assert.NotEqual(t, tenant.TenantStatusActive, tenantStatusDeleted)
}
