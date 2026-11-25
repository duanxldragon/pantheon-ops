package database

import (
	"context"
	"testing"

	"github.com/casbin/casbin/v2"
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

func TestBusinessEnforcerSeparatesStaticAndParamRoutes(t *testing.T) {
	enforcer, err := casbin.NewEnforcer(loadBusinessCasbinModel())
	if err != nil {
		t.Fatal(err)
	}
	addBusinessRouteMatcher(enforcer)
	if _, err := enforcer.AddPolicy("role:detail", "/api/v1/business/cmdb/hosts/:id", "GET"); err != nil {
		t.Fatal(err)
	}
	allowed, err := enforcer.Enforce("role:detail", "/api/v1/business/cmdb/hosts/export", "GET")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Fatal("detail grant authorised the static export route")
	}
	allowed, err = enforcer.Enforce("role:detail", "/api/v1/business/cmdb/hosts/:id", "GET")
	if err != nil || !allowed {
		t.Fatalf("detail grant did not authorise its own route: %v", err)
	}
	if _, err := enforcer.AddPolicy("admin", "/api/v1/*", "GET"); err != nil {
		t.Fatal(err)
	}
	allowed, err = enforcer.Enforce("admin", "/api/v1/business/cmdb/hosts/export", "GET")
	if err != nil || !allowed {
		t.Fatalf("explicit wildcard did not authorise the route: %v", err)
	}
}

func TestBusinessPolicyDetailGrantDoesNotCoverExportRoute(t *testing.T) {
	ctx := context.Background()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&CasbinRule{}); err != nil {
		t.Fatal(err)
	}
	detail := "/api/v1/business/cmdb/hosts/:id"
	if err := db.Create(&CasbinRule{PType: "p", V0: tenant.RoleSubject("viewer", 101), V1: detail, V2: "GET"}).Error; err != nil {
		t.Fatal(err)
	}
	policy, err := ReadBusinessPolicy(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	bindings := []businessauth.RoleBinding{{TenantID: 101, RoleKey: "viewer"}}
	export := businessauth.Action{Method: "GET", Path: "/api/v1/business/cmdb/hosts/export", RouteTemplate: "/api/v1/business/cmdb/hosts/export"}
	if _, err := AuthorizeBusinessBindings(ctx, db, policy, bindings, export); err == nil {
		t.Fatal("detail grant authorised the export route")
	}
	own := businessauth.Action{Method: "GET", Path: "/api/v1/business/cmdb/hosts/9", RouteTemplate: detail}
	if _, err := AuthorizeBusinessBindings(ctx, db, policy, bindings, own); err != nil {
		t.Fatalf("detail grant did not authorise its own route: %v", err)
	}
}
