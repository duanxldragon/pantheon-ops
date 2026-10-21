package tenant

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Create tables
	err = db.AutoMigrate(&Tenant{}, &Membership{})
	require.NoError(t, err)

	return db
}

func TestCreateTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	dto := CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
		Plan: "free",
	}

	tenant, err := service.CreateTenant(dto)
	require.NoError(t, err)
	assert.Equal(t, "test-tenant", tenant.Code)
	assert.Equal(t, "Test Tenant", tenant.Name)
	assert.Equal(t, "active", tenant.Status)
	assert.Equal(t, "free", tenant.Plan)
}

func TestCreateTenant_DuplicateCode(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	dto := CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	}

	_, err := service.CreateTenant(dto)
	require.NoError(t, err)

	// Try to create duplicate
	_, err = service.CreateTenant(dto)
	assert.Equal(t, ErrTenantCodeExists, err)
}

func TestGetTenantByID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Create tenant
	dto := CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	}
	created, err := service.CreateTenant(dto)
	require.NoError(t, err)

	// Get by ID
	tenant, err := service.GetTenantByID(created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, tenant.ID)
	assert.Equal(t, "test-tenant", tenant.Code)
}

func TestGetTenantByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	_, err := service.GetTenantByID(99999)
	assert.Equal(t, ErrTenantNotFound, err)
}

func TestListTenants(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Create multiple tenants
	for i := 1; i <= 5; i++ {
		_, err := service.CreateTenant(CreateTenantDTO{
			Code: fmt.Sprintf("tenant-%d", i),
			Name: fmt.Sprintf("Tenant %d", i),
			Plan: "free",
		})
		require.NoError(t, err)
	}

	// List with pagination
	filter := Filter{
		Page: 1,
		Size: 3,
	}
	response, err := service.ListTenants(filter)
	require.NoError(t, err)
	assert.Equal(t, int64(5), response.Total)
	assert.Len(t, response.Items, 3)
}

func TestUpdateTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Create tenant
	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	// Update
	newName := "Updated Tenant"
	newStatus := "suspended"
	err = service.UpdateTenant(tenant.ID, UpdateTenantDTO{
		Name:   &newName,
		Status: &newStatus,
	})
	require.NoError(t, err)

	// Verify
	updated, err := service.GetTenantByID(tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Tenant", updated.Name)
	assert.Equal(t, "suspended", updated.Status)
}

func TestAddMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// Create tenant
	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	// Add member
	err = service.AddMember(tenant.ID, 100, "admin")
	require.NoError(t, err)

	// Verify
	members, total, err := service.ListTenantMembers(tenant.ID, 1, 20)
	require.NoError(t, err)
	assert.Len(t, members, 1)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, tenant.ID, members[0].TenantID)
	assert.Equal(t, uint64(100), members[0].UserID)
	assert.Equal(t, "admin", members[0].Role)
}

func TestAddMember_Duplicate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	err = service.AddMember(tenant.ID, 100, "admin")
	require.NoError(t, err)

	// Try to add same user again
	err = service.AddMember(tenant.ID, 100, "member")
	assert.Equal(t, ErrMembershipExists, err)
}

func TestRemoveMember(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	err = service.AddMember(tenant.ID, 100, "admin")
	require.NoError(t, err)

	// Remove member
	err = service.RemoveMember(tenant.ID, 100)
	require.NoError(t, err)

	// Verify
	members, total, err := service.ListTenantMembers(tenant.ID, 1, 20)
	require.NoError(t, err)
	assert.Len(t, members, 0)
	assert.Equal(t, int64(0), total)
}

// TestListTenantMembers_PaginationContract pins the bounded listing contract:
// total counts the full membership, pages are bounded, and out-of-range or
// oversized requests are clamped instead of materializing the whole table.
func TestListTenantMembers_PaginationContract(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "paged-tenant",
		Name: "Paged Tenant",
	})
	require.NoError(t, err)

	const memberCount = 7
	for i := 1; i <= memberCount; i++ {
		require.NoError(t, service.AddMember(tenant.ID, uint64(200+i), "member"))
	}

	members, total, err := service.ListTenantMembers(tenant.ID, 1, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(memberCount), total)
	assert.Len(t, members, 3)

	members, _, err = service.ListTenantMembers(tenant.ID, 3, 3)
	require.NoError(t, err)
	assert.Len(t, members, 1)

	// Oversized pageSize is clamped instead of loading everything.
	members, _, err = service.ListTenantMembers(tenant.ID, 1, 100000)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(members), maxTenantMemberPageSize)

	// Non-positive page/pageSize fall back to safe defaults.
	members, total, err = service.ListTenantMembers(tenant.ID, 0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(memberCount), total)
	assert.LessOrEqual(t, len(members), defaultTenantMemberPageSize)

	// Role lookups stay bounded and scoped to a single membership.
	role, isMember, err := service.ActiveMembershipRole(tenant.ID, 201)
	require.NoError(t, err)
	assert.True(t, isMember)
	assert.Equal(t, "member", role)

	role, isMember, err = service.ActiveMembershipRole(tenant.ID, 999999)
	require.NoError(t, err)
	assert.False(t, isMember)
	assert.Empty(t, role)

	rolesByTenant, err := service.ActiveMembershipRolesByTenant(202, []uint64{tenant.ID, 424242})
	require.NoError(t, err)
	assert.Equal(t, "member", rolesByTenant[tenant.ID])
	assert.NotContains(t, rolesByTenant, uint64(424242))
}

func TestDeleteTenant_WithActiveMembers(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	err = service.AddMember(tenant.ID, 100, "admin")
	require.NoError(t, err)

	// Try to delete - should fail
	err = service.DeleteTenant(tenant.ID)
	assert.Equal(t, ErrTenantHasActiveUsers, err)
}

func TestDeleteTenant_Success(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	// Delete empty tenant
	err = service.DeleteTenant(tenant.ID)
	require.NoError(t, err)

	// Verify status changed to deleted
	deleted, err := service.GetTenantByID(tenant.ID)
	require.NoError(t, err)
	assert.Equal(t, "deleted", deleted.Status)
}
