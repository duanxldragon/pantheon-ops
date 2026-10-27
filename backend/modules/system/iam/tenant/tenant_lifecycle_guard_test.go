package tenant

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestTenantStatusTransitionAllowed pins contract §2.1: active <-> suspended,
// active|suspended -> archived, and archived is terminal. The rule is a pure
// function so the state machine can be asserted without a database.
func TestTenantStatusTransitionAllowed(t *testing.T) {
	cases := []struct {
		name     string
		previous string
		next     string
		want     bool
	}{
		{"active to suspended", tenant.TenantStatusActive, tenant.TenantStatusSuspended, true},
		{"active to archived", tenant.TenantStatusActive, tenant.TenantStatusArchived, true},
		{"suspended to active", tenant.TenantStatusSuspended, tenant.TenantStatusActive, true},
		{"suspended to archived", tenant.TenantStatusSuspended, tenant.TenantStatusArchived, true},
		{"archived to active is terminal", tenant.TenantStatusArchived, tenant.TenantStatusActive, false},
		{"archived to suspended is terminal", tenant.TenantStatusArchived, tenant.TenantStatusSuspended, false},
		{"same status is a no-op", tenant.TenantStatusArchived, tenant.TenantStatusArchived, true},
		{"empty next is a no-op", tenant.TenantStatusActive, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tenantStatusTransitionAllowed(tc.previous, tc.next))
		})
	}
}

// TestUpdateTenantRejectsArchivedReactivation drives the real update path: an
// archived tenant stays archived and the rejected write does not touch the name.
func TestUpdateTenantRejectsArchivedReactivation(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "archived-co", Name: "Archived Co"})
	require.NoError(t, err)
	archived := tenant.TenantStatusArchived
	require.NoError(t, service.UpdateTenant(created.ID, UpdateTenantDTO{Status: &archived}))

	active := tenant.TenantStatusActive
	name := "Renamed Co"
	err = service.UpdateTenant(created.ID, UpdateTenantDTO{Status: &active, Name: &name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant.status.terminal")

	row, err := service.GetTenantByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, tenant.TenantStatusArchived, row.Status)
	assert.Equal(t, "Archived Co", row.Name, "a rejected transition must not apply the other fields")
}

// TestRemoveMemberRefusesLastOwner pins contract §2.2 through the real delete
// path against MySQL: the last active owner cannot be removed, a plain member
// still can, and once a second owner exists the first one is removable.
func TestRemoveMemberRefusesLastOwner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "owner-co", Name: "Owner Co"})
	require.NoError(t, err)

	require.NoError(t, service.AddMember(created.ID, 100, "owner"))
	require.NoError(t, service.AddMember(created.ID, 101, "member"))

	assert.ErrorIs(t, service.RemoveMember(created.ID, 100), ErrLastOwner)

	owners := countActiveOwners(t, service, created.ID)
	assert.Equal(t, int64(1), owners, "the refused removal must leave the owner in place")

	// A plain member is not protected.
	require.NoError(t, service.RemoveMember(created.ID, 101))

	// With a second owner the first one is removable again.
	require.NoError(t, service.AddMember(created.ID, 102, "owner"))
	require.NoError(t, service.RemoveMember(created.ID, 100))

	members, total, err := service.ListTenantMembers(created.ID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, uint64(102), members[0].UserID)
	assert.Equal(t, "owner", members[0].Role)
}

func countActiveOwners(t *testing.T, service *Service, tenantID uint64) int64 {
	t.Helper()
	var owners int64
	require.NoError(t, service.db.Model(&Membership{}).
		Where("tenant_id = ? AND role = ? AND status = ?", tenantID, tenant.MembershipRoleOwner, tenant.MembershipActive).
		Count(&owners).Error)
	return owners
}

// TestDeleteTenantArchivedWithMembersSucceeds closes the deadlock between the
// two lifecycle rules: §2.1 makes archiving the documented retirement path and
// §2.2 forbids dropping the last owner, so without an archived-aware delete a
// provisioned tenant could never be removed at all.
func TestDeleteTenantArchivedWithMembersSucceeds(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "retire-co", Name: "Retire Co"})
	require.NoError(t, err)
	require.NoError(t, service.AddMember(created.ID, 201, "owner"))

	// A live tenant with members stays protected.
	assert.ErrorIs(t, service.DeleteTenant(created.ID), ErrTenantHasActiveUsers)

	archived := tenant.TenantStatusArchived
	require.NoError(t, service.UpdateTenant(created.ID, UpdateTenantDTO{Status: &archived}))
	require.NoError(t, service.DeleteTenant(created.ID))

	// Read the tombstone directly: GetTenantByID reports it as not found, which
	// is what stops a deleted tenant from being addressed again.
	var row Tenant
	require.NoError(t, db.Unscoped().Where(condIDEquals, created.ID).Take(&row).Error)
	assert.Equal(t, tenantStatusDeleted, row.Status)
	assert.NotEqual(t, "retire-co", row.Code, "the released code must be archived, not reused in place")

	var active int64
	require.NoError(t, db.Model(&Membership{}).
		Where("tenant_id = ? AND status = ?", created.ID, tenant.MembershipActive).
		Count(&active).Error)
	assert.Zero(t, active, "deleting an archived tenant must disable its remaining memberships")
}

// TestCreateTenantRollsBackWhenProvisioningFails pins the compensation path: a
// tenant row that never became usable must not survive, because it would hold
// its code and have no owner (§2.2), leaving no repair path except manual SQL.
func TestCreateTenantRollsBackWhenProvisioningFails(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	const callbackName = "test:fail-system-user-create"
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "system_user" {
			tx.AddError(errors.New("injected provisioning failure"))
		}
	}))
	t.Cleanup(func() { _ = db.Callback().Create().Remove(callbackName) })

	_, err := service.CreateTenant(CreateTenantDTO{
		Code:          "rollback-co",
		Name:          "Rollback Co",
		AdminUsername: "rollback-admin",
		AdminPassword: "admin-pass-123",
	})
	require.Error(t, err)

	var stranded int64
	require.NoError(t, db.Unscoped().Model(&Tenant{}).
		Where("code = ?", "rollback-co").
		Count(&stranded).Error)
	assert.Zero(t, stranded, "a failed provisioning must not leave the tenant row behind")

	// The compensation is what makes the retry a plain re-create: the code is
	// free again as soon as the injected failure is removed.
	require.NoError(t, db.Callback().Create().Remove(callbackName))
	retried, err := service.CreateTenant(CreateTenantDTO{Code: "rollback-co", Name: "Rollback Co"})
	require.NoError(t, err)
	assert.NotZero(t, retried.ID)
}

// TestDeletedTenantCannotBeRevivedByUpdate pins the fourth state that the
// contract only implies. `deleted` is a tombstone written by DeleteTenant, not
// a lifecycle state, and it has to be terminal: the row keeps a synthetic code
// (`__deleted_tenant_<id>`) that no operator can type, so reviving it would
// produce a live tenant nobody can look up by code, with its memberships still
// disabled.
//
// Found while writing the V1.3 clarification: `tenantStatusTransitionAllowed`
// fell through to `default: return true` for an unknown previous status, and
// UpdateTenant loads the row without filtering the tombstone, so a PUT with
// `{"status":"active"}` brought a deleted tenant back.
func TestDeletedTenantCannotBeRevivedByUpdate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "tombstone-co", Name: "Tombstone Co"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	active := tenant.TenantStatusActive
	err = service.UpdateTenant(created.ID, UpdateTenantDTO{Status: &active})
	require.Error(t, err, "a deleted tenant must not be reactivated")
	assert.ErrorIs(t, err, ErrTenantNotFound)

	var row Tenant
	require.NoError(t, db.Unscoped().Where(condIDEquals, created.ID).Take(&row).Error)
	assert.Equal(t, tenantStatusDeleted, row.Status, "the tombstone must survive the rejected update")

	// The state machine itself has to say so, not only the guard around it.
	assert.False(t, tenantStatusTransitionAllowed(tenantStatusDeleted, tenant.TenantStatusActive))
	assert.False(t, tenantStatusTransitionAllowed(tenantStatusDeleted, tenant.TenantStatusArchived))
}

// TestDeletedTenantRefusesNewMembers is the same rule on the membership path: a
// tombstone cannot accumulate members, because a membership is the login gate
// (contract §2.2) and the tenant is gone.
func TestDeletedTenantRefusesNewMembers(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "tombstone-members", Name: "Tombstone Members"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	err = service.AddMember(created.ID, 777, "member")
	assert.ErrorIs(t, err, ErrTenantNotFound)

	var memberships int64
	require.NoError(t, db.Model(&Membership{}).Where("tenant_id = ?", created.ID).Count(&memberships).Error)
	assert.Zero(t, memberships)
}

// TestDeletedTenantCannotBeReprovisioned keeps the repair entry point honest:
// provisioning a tombstone would re-create roles and an administrator for a
// tenant that no longer exists.
func TestDeletedTenantCannotBeReprovisioned(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	created, err := service.CreateTenant(CreateTenantDTO{Code: "tombstone-provision", Name: "Tombstone Provision"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	_, err = service.ProvisionTenant(ProvisionInput{TenantID: created.ID})
	assert.ErrorIs(t, err, ErrTenantNotFound)
}
