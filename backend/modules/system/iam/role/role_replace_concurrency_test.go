package role

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestConcurrentReplaceRolePermissionsSerialisesOnTheRoleRow covers the
// delete-then-insert residual deterministically.
//
// replaceRolePermissions deletes every grant for a role and re-inserts the
// requested set. Two replacers that interleave (both delete, then both insert)
// leave a merged set or deadlock on the index gap; that interleaving cannot be
// forced through the function itself, because MySQL's own row locks change the
// timing. So the test asserts the mechanism instead: while one replace is in
// flight, the role row is locked, and a second transaction cannot acquire it.
// That lock is what makes the delete-then-insert sequence atomic per role.
func TestConcurrentReplaceRolePermissionsSerialisesOnTheRoleRow(t *testing.T) {
	db := setupRoleTestDB(t)
	service := NewRoleService(db)

	created, err := service.CreateRole(&RoleCreateReq{
		RoleName: "Concurrent Replace",
		RoleKey:  "concurrent_replace",
		Status:   1,
	})
	require.NoError(t, err)

	testmysql.AssertSerialisedWriters(t, db, "system_role", created.ID,
		func(tx *gorm.DB) error {
			return service.replaceRolePermissions(tx, created.ID, []string{"concurrent:first"})
		},
		func(tx *gorm.DB) error {
			return service.replaceRolePermissions(tx, created.ID, []string{"concurrent:second"})
		},
	)

	// The last writer wins with its complete set - no merge, no duplicate.
	var stored []string
	require.NoError(t, db.Table("system_role_permission").
		Where("role_id = ?", created.ID).
		Order("permission_key asc").
		Pluck("permission_key", &stored).Error)
	assert.Equal(t, []string{"concurrent:second"}, stored)
}
