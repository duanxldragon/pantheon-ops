package bizscope

import (
	"context"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

func TestEnvironmentCRUDAndScopedVisibility(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := NewService(db)
	parents := []BizScope{{Code: "env-a", Name: "Scope A", Environment: "dev", Status: "active", DeptID: 10}, {Code: "env-b", Name: "Scope B", Environment: "prod", Status: "active", DeptID: 20}}
	if err := db.Create(&parents).Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 10}
	req := CreateEnvironmentRequest{BusinessScopeID: parents[0].ID, Code: " dev ", Name: " Development ", EnvType: "dev", Status: "active"}
	item, err := svc.CreateEnvironment(ctx, req, scope)
	if err != nil || item.Code != "dev" || item.Name != "Development" || item.BusinessScopeName != "Scope A" {
		t.Fatalf("create: %+v %v", item, err)
	}
	if _, err := svc.CreateEnvironment(ctx, req, scope); err == nil || err.Error() != environmentCodeExistsKey {
		t.Fatalf("duplicate: %v", err)
	}
	req.BusinessScopeID = parents[1].ID
	if _, err := svc.CreateEnvironment(ctx, req, scope); err == nil || err.Error() != bizScopeNotFoundKey {
		t.Fatalf("foreign scope create: %v", err)
	}
	other, err := svc.CreateEnvironment(ctx, req, nil)
	if err != nil {
		t.Fatalf("same code another scope: %v", err)
	}
	page, err := svc.ListEnvironments(ctx, EnvironmentListQuery{Keyword: "Development"}, scope)
	if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != item.ID {
		t.Fatalf("scoped list: %+v %v", page, err)
	}
	if _, err := svc.GetEnvironment(ctx, other.ID, scope); err == nil || err.Error() != environmentNotFoundKey {
		t.Fatalf("foreign detail: %v", err)
	}
	name := "Updated"
	if _, err := svc.UpdateEnvironment(ctx, other.ID, UpdateEnvironmentRequest{Name: &name}, scope); err == nil || err.Error() != environmentNotFoundKey {
		t.Fatalf("foreign update: %v", err)
	}
	if _, err := svc.UpdateEnvironment(ctx, item.ID, UpdateEnvironmentRequest{BusinessScopeID: &parents[1].ID}, scope); err == nil || err.Error() != bizScopeNotFoundKey {
		t.Fatalf("foreign reassignment: %v", err)
	}
	updated, err := svc.UpdateEnvironment(ctx, item.ID, UpdateEnvironmentRequest{Name: &name}, scope)
	if err != nil || updated.Name != name || updated.BusinessScopeID != parents[0].ID {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if err := svc.DeleteEnvironment(ctx, other.ID, scope); err == nil || err.Error() != environmentNotFoundKey {
		t.Fatalf("foreign delete: %v", err)
	}
	if err := svc.DeleteEnvironment(ctx, item.ID, scope); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.GetEnvironment(ctx, item.ID, scope); err == nil || err.Error() != environmentNotFoundKey {
		t.Fatalf("deleted detail: %v", err)
	}
	req.BusinessScopeID = parents[0].ID
	if _, err := svc.CreateEnvironment(ctx, req, scope); err == nil || err.Error() != environmentCodeExistsKey {
		t.Fatalf("deleted code is reserved: %v", err)
	}
}

func TestEnvironmentValidationAndMissingScope(t *testing.T) {
	for _, test := range []struct {
		name string
		row  BusinessEnvironment
		want string
	}{
		{"scope", BusinessEnvironment{}, environmentScopeRequiredKey},
		{"code", BusinessEnvironment{BusinessScopeID: 1}, environmentCodeRequiredKey},
		{"name", BusinessEnvironment{BusinessScopeID: 1, Code: "dev"}, environmentNameRequiredKey},
		{"type", BusinessEnvironment{BusinessScopeID: 1, Code: "dev", Name: "Dev", EnvType: "invalid"}, environmentTypeInvalidKey},
		{"status", BusinessEnvironment{BusinessScopeID: 1, Code: "dev", Name: "Dev", EnvType: "dev", Status: "invalid"}, environmentStatusInvalidKey},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateEnvironment(&test.row); err == nil || err.Error() != test.want {
				t.Fatalf("got %v want %s", err, test.want)
			}
		})
	}
	db := setupBizScopeTestDB(t)
	if _, err := NewService(db).CreateEnvironment(context.Background(), CreateEnvironmentRequest{BusinessScopeID: 999, Code: "dev", Name: "Dev", EnvType: "dev", Status: "active"}, nil); err == nil || err.Error() != bizScopeNotFoundKey {
		t.Fatalf("missing scope: %v", err)
	}
}

func TestEnvironmentBlocksParentDeletion(t *testing.T) {
	db := setupBizScopeTestDB(t)
	svc := newBizScopeServiceWithCMDB(t, db)
	parent := BizScope{Code: "env-parent", Name: "Parent", Environment: "dev", Status: "active"}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	item, err := svc.CreateEnvironment(context.Background(), CreateEnvironmentRequest{BusinessScopeID: parent.ID, Code: "dev", Name: "Dev", EnvType: "dev", Status: "active"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(parent.ID); err == nil || err.Error() != bizScopeInUseKey {
		t.Fatalf("referenced scope deletion: %v", err)
	}
	if err := svc.DeleteEnvironment(context.Background(), item.ID, nil); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&BizScope{}).Where("id = ?", parent.ID).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("parent lost: %d %v", count, err)
	}
}
