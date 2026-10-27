package role

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestConcurrentCreateRoleMapsLoserToConflict covers the race the role-key
// pre-check cannot close: both callers pass the check and the UNIQUE index
// decides. The loser must receive the 409 the pre-check would have produced,
// not a 500 with the driver's duplicate-entry message.
func TestConcurrentCreateRoleMapsLoserToConflict(t *testing.T) {
	db := setupRoleTestDB(t)
	service := NewRoleService(db)

	const contenders = 4
	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
		errs  = make([]error, contenders)
	)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = service.CreateRole(&RoleCreateReq{
				RoleName: fmt.Sprintf("Race %d", i),
				RoleKey:  "race-role",
				Status:   1,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	requireSingleRoleKeyWinner(t, db, errs, "race-role")
}

// TestConcurrentRenameRoleMapsLoserToConflict exercises the same contract on
// the update path: renaming two roles to one key must yield a conflict, not a
// driver error.
func TestConcurrentRenameRoleMapsLoserToConflict(t *testing.T) {
	db := setupRoleTestDB(t)
	service := NewRoleService(db)

	first, err := service.CreateRole(&RoleCreateReq{RoleName: "First", RoleKey: "first-role", Status: 1})
	require.NoError(t, err)
	second, err := service.CreateRole(&RoleCreateReq{RoleName: "Second", RoleKey: "second-role", Status: 1})
	require.NoError(t, err)

	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
		errs  = make([]error, 2)
	)
	renames := []uint64{first.ID, second.ID}
	for i, roleID := range renames {
		wg.Add(1)
		go func(i int, roleID uint64) {
			defer wg.Done()
			<-start
			_, errs[i] = service.UpdateRole(roleID, &RoleUpdateReq{
				RoleName: fmt.Sprintf("Renamed %d", i),
				RoleKey:  "renamed-role",
				Status:   1,
			})
		}(i, roleID)
	}
	close(start)
	wg.Wait()

	requireSingleRoleKeyWinner(t, db, errs, "renamed-role")
}

// requireSingleRoleKeyWinner pins the shared contract of both races: one
// writer wins, every loser is a 409 with the pre-check's message, and the
// unique index leaves exactly one row on the contested key.
func requireSingleRoleKeyWinner(t *testing.T, db *gorm.DB, errs []error, roleKey string) {
	t.Helper()
	succeeded := 0
	for i, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		require.Truef(t, errors.Is(err, common.ErrConflict),
			"contender %d must fail as a conflict, got %v", i, err)
		assert.Equal(t, "role.key.exists", common.ErrMessage(err))
	}
	assert.Equal(t, 1, succeeded, "exactly one contender may take role key %q", roleKey)

	var rows int64
	require.NoError(t, db.Model(&SystemRole{}).Where("role_key = ?", roleKey).Count(&rows).Error)
	assert.EqualValues(t, 1, rows, "the unique index must leave exactly one role with key %q", roleKey)
}
