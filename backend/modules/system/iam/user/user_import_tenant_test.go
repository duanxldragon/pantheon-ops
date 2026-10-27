package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const (
	importTenantA = uint64(701)
	importTenantB = uint64(702)
)

// seedUserImportTenantFixtures gives two tenants identical reference data:
// the same department path, post code and role key. Those three values are
// unique per tenant, so an unscoped import lookup can only resolve them to the
// wrong tenant's row.
func seedUserImportTenantFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (101, 701, 'ops', 'Ops A', 1)",
		"INSERT INTO system_role (id, tenant_id, role_key, role_name, status) VALUES (201, 702, 'ops', 'Ops B', 1)",
		"INSERT INTO system_dept (id, tenant_id, parent_id, dept_name) VALUES (11, 701, 0, 'Root')",
		"INSERT INTO system_dept (id, tenant_id, parent_id, dept_name) VALUES (21, 702, 0, 'Root')",
		"INSERT INTO system_post (id, tenant_id, post_code, post_name, dept_id) VALUES (31, 701, 'P1', 'Post A', 11)",
		"INSERT INTO system_post (id, tenant_id, post_code, post_name, dept_id) VALUES (41, 702, 'P1', 'Post B', 21)",
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	require.NoError(t, db.Create(&SystemUser{
		TenantID: importTenantB,
		Username: "shared",
		Password: "x",
		Nickname: "Tenant B Original",
		Status:   1,
	}).Error)
}

func userImportRecords(nickname string) [][]string {
	return [][]string{
		{"username", "password", "nickname", "email", "phone", "deptPath", "postCode", "status", "roleKeys"},
		{"shared", "password123", nickname, "shared@example.com", "13800000000", "Root", "P1", "1", "ops"},
	}
}

// TestImportUsersStaysInsideTheRequestTenant pins the four lookups that used to
// be global: existing users, role keys, department paths and post codes. Tenant
// A's CSV names a username that already exists in tenant B; without the tenant
// scope that row would have been updated, its role resolved to tenant B's role
// and its dept/post to tenant B's ids. The created row must also carry tenant
// A's id and an active membership, otherwise the imported user cannot log in.
func TestImportUsersStaysInsideTheRequestTenant(t *testing.T) {
	db := setupUserTestDB(t)
	seedUserImportTenantFixtures(t, db)
	service := newTenantUserService(db, importTenantA)

	result, err := service.ImportUsers(userImportRecords("Imported A"))
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Updated)

	var imported SystemUser
	require.NoError(t, db.Where("tenant_id = ? AND username = ?", importTenantA, "shared").First(&imported).Error)
	assert.Equal(t, "Imported A", imported.Nickname)
	assert.EqualValues(t, 11, imported.DeptID, "deptPath must resolve inside the importing tenant")
	assert.EqualValues(t, 31, imported.PostID, "postCode must resolve inside the importing tenant")

	var binding struct {
		TenantID uint64 `gorm:"column:tenant_id"`
		RoleID   uint64 `gorm:"column:role_id"`
	}
	require.NoError(t, db.Table("system_user_role").
		Select("tenant_id, role_id").
		Where("user_id = ?", imported.ID).
		Scan(&binding).Error)
	assert.Equal(t, importTenantA, binding.TenantID)
	assert.EqualValues(t, 101, binding.RoleID, "roleKeys must resolve inside the importing tenant")

	row, ok := loadMembership(t, db, importTenantA, imported.ID)
	require.True(t, ok, "the imported user needs a membership or the first login fails")
	assert.Equal(t, "active", row.Status)

	var untouched SystemUser
	require.NoError(t, db.Where("tenant_id = ? AND username = ?", importTenantB, "shared").First(&untouched).Error)
	assert.Equal(t, "Tenant B Original", untouched.Nickname,
		"tenant A's import must not update the same-named user in tenant B")
	assert.EqualValues(t, 0, untouched.DeptID)

	// The second pass exercises the update branch: it must target tenant A's own
	// row, keep the membership, and still leave tenant B alone.
	result, err = service.ImportUsers(userImportRecords("Imported A v2"))
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Updated)

	require.NoError(t, db.Where("tenant_id = ? AND username = ?", importTenantA, "shared").First(&imported).Error)
	assert.Equal(t, "Imported A v2", imported.Nickname)
	require.NoError(t, db.Where("tenant_id = ? AND username = ?", importTenantB, "shared").First(&untouched).Error)
	assert.Equal(t, "Tenant B Original", untouched.Nickname)
	assert.EqualValues(t, 1, countMembershipsForUser(t, db, imported.ID))

	var total int64
	require.NoError(t, db.Model(&SystemUser{}).Where("username = ?", "shared").Count(&total).Error)
	assert.EqualValues(t, 2, total, "each tenant must keep exactly its own user row")
}
