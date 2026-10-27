package tenant

import (
	"errors"
	"fmt"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// tenantTestDDL declares the cross-module tables the tenant service writes.
//
// This package only owns `tenants` and `tenant_memberships` (AutoMigrated from
// Tenant/Membership below). Provisioning, however, also writes `system_role`,
// `system_user`, `system_user_role`, `system_role_permission`, `system_setting`
// and `system_dept` — tables owned by other modules. They are declared here
// explicitly, mirroring the production DDL, so a test failure means the service
// is wrong rather than the fixture.
//
// Statements are separate because the MySQL driver runs with
// multiStatements=false, so one Exec can only carry one statement.
var tenantTestDDL = []string{
	`CREATE TABLE system_role (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		role_name VARCHAR(64) NOT NULL,
		role_key VARCHAR(64) NOT NULL,
		sort BIGINT DEFAULT 0,
		data_scope VARCHAR(32) DEFAULT 'all',
		status BIGINT DEFAULT 1,
		remark VARCHAR(255) DEFAULT '',
		deleted_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_system_role_tenant_key (tenant_id, role_key)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_user (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		username VARCHAR(64) NOT NULL,
		password VARCHAR(255) NOT NULL,
		nickname VARCHAR(64) NULL,
		status BIGINT DEFAULT 1,
		deleted_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_system_user_tenant_username (tenant_id, username)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_user_role (
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		user_id BIGINT UNSIGNED NOT NULL,
		role_id BIGINT UNSIGNED NOT NULL,
		PRIMARY KEY (tenant_id, user_id, role_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_role_permission (
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		role_id BIGINT UNSIGNED NOT NULL,
		permission_key VARCHAR(128) NOT NULL,
		PRIMARY KEY (tenant_id, role_id, permission_key)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_setting (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		setting_key VARCHAR(128) NOT NULL,
		setting_value TEXT NULL,
		group_key VARCHAR(32) NOT NULL,
		module VARCHAR(64) NOT NULL DEFAULT 'system',
		label_key VARCHAR(128) DEFAULT '',
		description VARCHAR(255) DEFAULT '',
		value_type VARCHAR(16) NOT NULL DEFAULT 'string',
		sort INT DEFAULT 0,
		is_public BIGINT DEFAULT 0,
		is_encrypted BIGINT DEFAULT 0,
		remark VARCHAR(255) NULL,
		is_readonly TINYINT DEFAULT 0,
		created_at DATETIME(3) NULL,
		updated_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		UNIQUE KEY uk_system_setting_tenant_key (tenant_id, setting_key)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_dept (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		parent_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		ancestors VARCHAR(255) DEFAULT '',
		is_root BIGINT DEFAULT 0,
		dept_name VARCHAR(64) NOT NULL,
		sort INT DEFAULT 0,
		leader_user_id BIGINT UNSIGNED DEFAULT 0,
		leader VARCHAR(64) DEFAULT '',
		phone VARCHAR(20) DEFAULT '',
		email VARCHAR(128) DEFAULT '',
		status BIGINT DEFAULT 1,
		deleted_at DATETIME(3) NULL,
		created_at DATETIME(3) NULL,
		updated_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		KEY idx_system_dept_tenant_id (tenant_id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_dict_type (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		dict_code VARCHAR(64) NOT NULL,
		dict_name VARCHAR(64) NOT NULL,
		module VARCHAR(64) NOT NULL DEFAULT 'system',
		status TINYINT DEFAULT 1,
		remark VARCHAR(255) DEFAULT '',
		created_at DATETIME(3) NULL,
		updated_at DATETIME(3) NULL,
		deleted_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		UNIQUE KEY idx_dict_type_tenant_code (tenant_id, dict_code)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	`CREATE TABLE system_dict_item (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		dict_code VARCHAR(64) NOT NULL,
		item_label_key VARCHAR(128) NOT NULL,
		item_value VARCHAR(255) NOT NULL,
		item_color VARCHAR(32) DEFAULT '',
		sort INT DEFAULT 0,
		status TINYINT DEFAULT 1,
		remark VARCHAR(255) DEFAULT '',
		tag_type VARCHAR(32) DEFAULT '',
		css_class VARCHAR(64) DEFAULT '',
		created_at DATETIME(3) NULL,
		updated_at DATETIME(3) NULL,
		deleted_at DATETIME(3) NULL,
		PRIMARY KEY (id),
		KEY idx_dict_item_tenant_code_value (tenant_id, dict_code, item_value)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
}

// setupTestDB opens an isolated MySQL database for a single test.
//
// The suite runs against real MySQL (pkg/testmysql) like every other
// MySQL-backed package in this repo, rather than an in-memory SQLite file.
// Two reasons:
//
//   - CreateTenant provisions the tenant, which writes system_role,
//     system_user, system_user_role and system_setting. Under SQLite the
//     fixture only ever created two tables, so the suite silently stopped
//     exercising the provisioning path that production actually runs.
//   - The reserved placeholder is adopted onto id = 0 (contract §2.3), and
//     MySQL's AUTO_INCREMENT semantics for an explicit 0 differ from SQLite's.
//     Only MySQL exercises the real idiom.
//
// mustLoadTenantRow reads a tenant row straight from storage, tombstone
// included. GetTenantByID deliberately reports a deleted tenant as not found,
// so a test that inspects the post-delete row has to go around the service.
func mustLoadTenantRow(t *testing.T, db *gorm.DB, id uint64) Tenant {
	t.Helper()
	var row Tenant
	require.NoError(t, db.Unscoped().Where(condIDEquals, id).Take(&row).Error)
	return row
}

// It skips when no test DSN is configured, so the suite degrades to a no-op
// instead of failing on a machine without MySQL.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	// Plan definitions are part of the fixture: Bootstrap seeds them and every
	// quota read resolves against this table.
	require.NoError(t, db.AutoMigrate(&Tenant{}, &Membership{}, &Plan{}))
	for _, stmt := range tenantTestDDL {
		require.NoError(t, db.Exec(stmt).Error)
	}
	// Provisioning seeds tenant-scoped role-menu grants, so the navigation
	// tables are part of the provisioning fixture, not an optional extra.
	require.NoError(t, db.Exec(tenantMenuTestDDL).Error)
	require.NoError(t, db.Exec(tenantRoleMenuTestDDL).Error)
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

	// Re-adding a non-owner row is idempotent and updates the role: user
	// create in a tenant viewpoint already inserts the `member` row
	// server-side, and the console's join submit must not fail on it.
	err = service.AddMember(tenant.ID, 100, "member")
	require.NoError(t, err)

	members, total, err := service.ListTenantMembers(tenant.ID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "member", members[0].Role)
}

func TestAddMember_OwnerIsProtected(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenant, err := service.CreateTenant(CreateTenantDTO{
		Code: "test-tenant",
		Name: "Test Tenant",
	})
	require.NoError(t, err)

	require.NoError(t, service.AddMember(tenant.ID, 100, "owner"))
	// Re-adding owner is a no-op; a non-owner role can never demote the owner.
	require.NoError(t, service.AddMember(tenant.ID, 100, "owner"))
	assert.Equal(t, ErrMembershipExists, service.AddMember(tenant.ID, 100, "member"))

	members, _, err := service.ListTenantMembers(tenant.ID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, "owner", members[0].Role)
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

	// Verify status changed to deleted. A deleted tenant is no longer
	// addressable through the service, so the row is read directly.
	deleted := mustLoadTenantRow(t, db, tenant.ID)
	assert.Equal(t, "deleted", deleted.Status)
}

// TestCreateTenant_AdminPasswordHonoursConfiguredPolicy pins the contract that
// the tenant administrator's password is measured against the deployment's
// configurable policy (`security.password_min_length`), the same setting that
// governs user creation and password change.
//
// It used to be a hardcoded `min=8` in the request binding while the console
// form validated 6, so a 6-character password the form accepted came back as a
// bare `param.invalid` with no indication of which field was wrong.
func TestCreateTenant_AdminPasswordHonoursConfiguredPolicy(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	// At the seeded default (6) a 6-character password is acceptable.
	_, err := service.CreateTenant(CreateTenantDTO{
		Code:          "policy-default",
		Name:          "Policy Default",
		AdminUsername: "admin",
		AdminPassword: "abc123",
	})
	require.NoError(t, err, "6 characters must be accepted at the default policy")

	// Below the policy is a bad request, not an opaque failure.
	_, err = service.CreateTenant(CreateTenantDTO{
		Code:          "policy-too-short",
		Name:          "Policy Too Short",
		AdminUsername: "admin",
		AdminPassword: "abc12",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, common.ErrBadRequest), "expected a bad request, got %v", err)

	// Raise the platform policy: the same 6 characters must now be rejected,
	// which is what proves the value is read rather than hardcoded.
	require.NoError(t, db.Exec(
		`INSERT INTO system_setting (tenant_id, setting_key, setting_value, group_key)
		 VALUES (?, ?, ?, ?)`,
		platformGlobalTenantID, "security.password_min_length", "10", "security",
	).Error)

	_, err = service.CreateTenant(CreateTenantDTO{
		Code:          "policy-raised-ok",
		Name:          "Policy Raised OK",
		AdminUsername: "admin",
		AdminPassword: "abcdefghij",
	})
	require.NoError(t, err, "10 characters must be accepted at a 10-character policy")

	_, err = service.CreateTenant(CreateTenantDTO{
		Code:          "policy-raised-short",
		Name:          "Policy Raised Short",
		AdminUsername: "admin",
		AdminPassword: "abc123",
	})
	require.Error(t, err, "6 characters must be rejected at a 10-character policy")

	// A tenant created without an administrator is not password-constrained:
	// the password is never used.
	_, err = service.CreateTenant(CreateTenantDTO{
		Code:          "policy-no-admin",
		Name:          "Policy No Admin",
		AdminPassword: "x",
	})
	require.NoError(t, err)
}
