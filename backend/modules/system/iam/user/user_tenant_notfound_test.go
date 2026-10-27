package user

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadTenantUserHidesOtherTenants pins contract §5 on the read path: a user
// owned by another tenant must be indistinguishable from a missing one, so a
// guessed id cannot confirm that a row exists somewhere else.
func TestLoadTenantUserHidesOtherTenants(t *testing.T) {
	db := testmysql.Open(t)
	require.NoError(t, db.AutoMigrate(&SystemUser{}))
	require.NoError(t, db.Exec("DELETE FROM system_user").Error)

	owned := SystemUser{TenantID: 101, Username: "alpha-admin", Password: "x", Nickname: "Alpha Admin", Status: 1}
	require.NoError(t, db.Create(&owned).Error)

	self, err := newTenantUserService(db, 101).loadTenantUser(owned.ID)
	require.NoError(t, err)
	assert.Equal(t, "alpha-admin", self.Username)
	assert.Equal(t, uint64(101), self.TenantID)

	_, err = newTenantUserService(db, 202).loadTenantUser(owned.ID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, common.ErrNotFound), "foreign id must resolve to not-found, got %v", err)

	_, err = newTenantUserService(db, 202).loadTenantUser(owned.ID + 999)
	require.Error(t, err)
	assert.True(t, errors.Is(err, common.ErrNotFound), "missing id must resolve to not-found, got %v", err)
}

// TestUpdateUserRejectsForeignTenantUserWithoutWriting pins the write path: the
// tenant-scoped update resolves the row before mutating anything, so another
// tenant's user keeps its nickname and the caller sees not-found.
func TestUpdateUserRejectsForeignTenantUserWithoutWriting(t *testing.T) {
	db := testmysql.Open(t)
	require.NoError(t, db.AutoMigrate(&SystemUser{}))
	require.NoError(t, db.Exec("DELETE FROM system_user").Error)

	owned := SystemUser{TenantID: 101, Username: "alpha-member", Password: "x", Nickname: "Alpha Member", Status: 1}
	require.NoError(t, db.Create(&owned).Error)

	_, err := newTenantUserService(db, 202).UpdateUser(owned.ID, &UserUpdateReq{Nickname: "pwned-by-202"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, common.ErrNotFound), "foreign update must answer not-found, got %v", err)

	var reloaded SystemUser
	require.NoError(t, db.First(&reloaded, owned.ID).Error)
	assert.Equal(t, "Alpha Member", reloaded.Nickname)
}
