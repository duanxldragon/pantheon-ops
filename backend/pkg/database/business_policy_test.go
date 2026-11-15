package database

import (
	"context"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestBusinessPolicyUsesFreshStoreAndExactNamespace(t *testing.T) {
	ctx := context.Background()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&CasbinRule{}); err != nil {
		t.Fatal(err)
	}
	action := businessauth.Action{Method: "POST", Path: "/api/v1/business/policy/7", RouteTemplate: "/api/v1/business/policy/:id"}
	global := CasbinRule{PType: "p", V0: "admin", V1: "/api/v1/*", V2: "POST"}
	if err := db.Create(&global).Error; err != nil {
		t.Fatal(err)
	}
	policy, err := ReadBusinessPolicy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	local := []businessauth.RoleBinding{{TenantID: 101, RoleKey: "admin"}}
	if _, err := AuthorizeBusinessBindings(ctx, db, policy, local, action); err == nil {
		t.Fatal("tenant key promoted into global grant")
	}
	if err := db.Create(&CasbinRule{PType: "p", V0: tenant.RoleSubject("admin", 101), V1: action.RouteTemplate, V2: "POST"}).Error; err != nil {
		t.Fatal(err)
	}
	fresh, err := ReadBusinessPolicy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthorizeBusinessBindings(ctx, db, fresh, local, action); err != nil {
		t.Fatal("fresh local grant not loaded")
	}
	if err := db.Where("v0 = ?", tenant.RoleSubject("admin", 101)).Delete(&CasbinRule{}).Error; err != nil {
		t.Fatal(err)
	}
	fresh, err = ReadBusinessPolicy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AuthorizeBusinessBindings(ctx, db, fresh, local, action); err == nil {
		t.Fatal("deleted persisted policy still granted")
	}
	if _, err := ReadBusinessPolicy(ctx, nil); err == nil {
		t.Fatal("nil store accepted")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := ReadBusinessPolicy(cancelled, db); err == nil {
		t.Fatal("cancelled policy read accepted")
	}
}
