package login

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
)

// TestGetUserPermsIgnoresStalePermissionCache is the failed-before regression
// for the CMDB R3 revocation finding: a previously cached permission array must
// never outlive a granted key. The stale entry is seeded in real Redis and the
// grant is then removed from the database.
func TestGetUserPermsIgnoresStalePermissionCache(t *testing.T) {
	db := setupTestDB(t)

	actor := user.SystemUser{Username: "cache_revocation", Password: "unused", Status: 1}
	if err := db.Create(&actor).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	probe := role.SystemRole{RoleKey: "cache_revocation_probe", RoleName: "cache revocation probe", Status: 1}
	if err := db.Create(&probe).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := db.Create(&user.SystemUserRole{UserID: actor.ID, RoleID: probe.ID}).Error; err != nil {
		t.Fatalf("assign role: %v", err)
	}
	if err := db.Create(&role.SystemRolePermission{RoleID: probe.ID, PermissionKey: "system:user:list"}).Error; err != nil {
		t.Fatalf("grant permission: %v", err)
	}

	rdb := testredis.OpenDB(t, testredis.DBLogin)
	previousRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = previousRDB })

	ctx := context.Background()
	cacheKey := fmt.Sprintf("user_perms:%d", actor.ID)
	stale := `["system:user:list","system:user:delete"]`
	if err := rdb.Set(ctx, cacheKey, stale, time.Minute).Err(); err != nil {
		t.Fatalf("seed stale permission cache: %v", err)
	}

	if err := db.Where("role_id = ?", probe.ID).Delete(&role.SystemRolePermission{}).Error; err != nil {
		t.Fatalf("revoke permission: %v", err)
	}

	runtime := &Runtime{db: db}
	perms, err := runtime.GetUserPerms(actor.ID)
	if err != nil {
		t.Fatalf("read current permissions: %v", err)
	}
	if len(perms) != 0 {
		t.Fatalf("revoked permission survived through the cache: %v", perms)
	}

	if got, getErr := rdb.Get(ctx, cacheKey).Result(); getErr != nil || got != stale {
		t.Fatalf("permission read mutated the cache entry: value=%q err=%v", got, getErr)
	}

	if err := db.Create(&role.SystemRolePermission{RoleID: probe.ID, PermissionKey: "system:user:list"}).Error; err != nil {
		t.Fatalf("regrant permission: %v", err)
	}
	perms, err = runtime.GetUserPerms(actor.ID)
	if err != nil {
		t.Fatalf("read regranted permissions: %v", err)
	}
	if len(perms) != 1 || perms[0] != "system:user:list" {
		t.Fatalf("regrant was not read authoritatively: %v", perms)
	}
}
