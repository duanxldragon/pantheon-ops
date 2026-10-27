package user

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestConcurrentCreateUserMapsLoserToConflict covers the race the username
// pre-check cannot close: both callers see no row and only the UNIQUE index
// decides. The loser must receive the same 409 the pre-check would have
// produced, not a 500 carrying the MySQL duplicate-entry message.
func TestConcurrentCreateUserMapsLoserToConflict(t *testing.T) {
	db := setupUserTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// setupUserTestDB pins one connection for deterministic fixtures; the race
	// needs two transactions in flight at the same time.
	sqlDB.SetMaxOpenConns(4)

	service := NewUserService(db)
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
			_, errs[i] = service.CreateUser(&UserCreateReq{
				Username: "race-user",
				Password: "password123",
				Nickname: fmt.Sprintf("Race %d", i),
				Status:   1,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		require.Truef(t, errors.Is(err, common.ErrConflict),
			"contender %d must fail as a conflict, got %v", i, err)
		assert.Equal(t, "user.create.error.username_exists", common.ErrMessage(err))
	}
	assert.Equal(t, 1, succeeded, "exactly one contender may create the user")

	var rows int64
	require.NoError(t, db.Model(&SystemUser{}).Where("username = ?", "race-user").Count(&rows).Error)
	assert.EqualValues(t, 1, rows, "the unique index must leave exactly one user row")
}

// TestConcurrentReplaceUserRolesSerialisesOnTheUserRow covers the
// delete-then-insert residual deterministically.
//
// replaceUserRoles deletes every binding for a user and re-inserts the requested
// set. Two replacers that interleave (both delete, then both insert) leave a
// merged set or deadlock on the index gap; MySQL's own row locks make that
// interleaving impossible to force through the function. The test therefore
// asserts the mechanism: while one replace is in flight, the user row is
// locked, and a second transaction cannot acquire it.
func TestConcurrentReplaceUserRolesSerialisesOnTheUserRow(t *testing.T) {
	db := setupUserTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(4)

	require.NoError(t, db.Exec(
		"INSERT INTO system_user (tenant_id, username, password, nickname, status) VALUES (0, 'replace-user', 'x', 'Replace', 1)",
	).Error)
	var user SystemUser
	require.NoError(t, db.Where("username = ?", "replace-user").Take(&user).Error)

	testmysql.AssertSerialisedWriters(t, db, "system_user", user.ID,
		func(tx *gorm.DB) error { return replaceUserRoles(tx, user.ID, []uint64{1}) },
		func(tx *gorm.DB) error { return replaceUserRoles(tx, user.ID, []uint64{2}) },
	)

	var stored []uint64
	require.NoError(t, db.Table("system_user_role").
		Where("user_id = ?", user.ID).
		Order("role_id asc").
		Pluck("role_id", &stored).Error)
	assert.Equal(t, []uint64{2}, stored, "the last writer's complete set must survive")
}
