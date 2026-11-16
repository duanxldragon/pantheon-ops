package observability

import (
	"context"
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func safeMetricRecipe() QueryRecipe {
	return QueryRecipe{Version: 1, SourceKind: "prometheus", Rules: []QueryRule{{Field: "availability", Op: "instant"}}, Time: QueryTime{Mode: "relative", Duration: "15m"}, Limit: 100, ResultMode: "table"}
}
func TestWorkspaceTenantActorAndDeletionAfterRevocation(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}, &AlertRule{}); err != nil {
		t.Fatal(err)
	}
	sourceA := PrometheusDatasource{TenantID: 11, Name: "same", Type: "prometheus", URL: "http://127.0.0.1:1", DeptID: 7}
	sourceB := PrometheusDatasource{TenantID: 22, Name: "same", Type: "prometheus", URL: "http://127.0.0.1:1", DeptID: 8}
	for _, source := range []*PrometheusDatasource{&sourceA, &sourceB} {
		if err := db.Create(source).Error; err != nil {
			t.Fatal(err)
		}
	}
	scope := &common.DataScopeReq{IsAdmin: true}
	owner := QueryOwner{TenantID: 11, ActorID: 31}
	service, err := NewQueryWorkspaceService(db, owner, scope)
	if err != nil {
		t.Fatal(err)
	}
	view, err := service.Save(context.Background(), sourceA.ID, 0, 0, safeMetricRecipe())
	if err != nil {
		t.Fatal(err)
	}
	otherActor, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 11, ActorID: 32}, scope)
	otherTenant, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 22, ActorID: 31}, scope)
	for _, other := range []*QueryWorkspaceService{otherActor, otherTenant} {
		if _, _, err := other.Views(context.Background(), view.ID, 1, 20); !errors.Is(err, ErrWorkspaceDenied) {
			t.Fatalf("foreign actor/tenant view read: %v", err)
		}
	}
	if _, err := service.Save(context.Background(), sourceB.ID, 0, 0, safeMetricRecipe()); err == nil {
		t.Fatal("foreign source accepted")
	}
	if _, err := service.Save(context.Background(), sourceA.ID, view.ID, 0, safeMetricRecipe()); !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("missing revision accepted: %v", err)
	}
	if err := db.Model(&sourceA).Update("dept_id", 8).Error; err != nil {
		t.Fatal(err)
	}
	revoked, _ := NewQueryWorkspaceService(db, owner, &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7})
	if _, _, err := revoked.Views(context.Background(), view.ID, 1, 20); !errors.Is(err, ErrWorkspaceDenied) {
		t.Fatal("revoked source readable")
	}
	if err := revoked.Delete(context.Background(), true, view.ID); err != nil {
		t.Fatal(err)
	}
	var count int64
	db.Model(&SavedView{}).Count(&count)
	if count != 0 {
		t.Fatal("revoked-source private deletion failed")
	}
	var guards int64
	db.Model(&QueryOwnerBudget{}).Count(&guards)
	if guards == 0 {
		t.Fatal("empty owner guard reclaimed")
	}
	legacy := NewDatasourceService(db)
	if result, err := legacy.List(scope); err != nil || result.Total != 0 {
		t.Fatalf("tenant0 widened: %+v %v", result, err)
	}
	if _, err := legacy.WithTenantContext(nil); err == nil {
		t.Fatal("nil tenant context accepted")
	}
	bound, err := legacy.WithTenantContext(&tenant.Context{TenantID: 11, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatal(err)
	}
	if result, err := bound.List(scope); err != nil || result.Total != 1 {
		t.Fatalf("bound tenant mismatch: %+v %v", result, err)
	}
	if _, err := bound.List(); err == nil {
		t.Fatal("opted source nil scope accepted")
	}
}

func TestWorkspaceConcurrentCountBudget(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	names := []string{"one", "two"}
	sources := make([]PrometheusDatasource, 0, len(names))
	for _, name := range names {
		source := PrometheusDatasource{TenantID: 11, Name: name, Type: "prometheus", URL: "http://127.0.0.1:1"}
		if err := db.Create(&source).Error; err != nil {
			t.Fatal(err)
		}
		sources = append(sources, source)
	}
	service, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 11, ActorID: 31}, &common.DataScopeReq{IsAdmin: true})
	var successes, quota atomic.Int64
	var wait sync.WaitGroup
	for i := 0; i < 60; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			_, err := service.Save(context.Background(), sources[i%2].ID, 0, 0, safeMetricRecipe())
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, ErrWorkspaceQuota) {
				quota.Add(1)
			} else {
				t.Errorf("save failed: %v", err)
			}
		}(i)
	}
	wait.Wait()
	if successes.Load() != 50 || quota.Load() != 10 {
		t.Fatalf("quota race: saved%d refused%d", successes.Load(), quota.Load())
	}
	var count int64
	db.Model(&SavedView{}).Count(&count)
	if count != 50 {
		t.Fatalf("stored%d", count)
	}
}

func TestWorkspaceHistoryRetentionAndLateRevision(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "metric", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	service, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 11, ActorID: 31}, &common.DataScopeReq{IsAdmin: true})
	input := HistoryInput{DatasourceID: source.ID, SourceRevision: source.Revision, SourceKind: source.Type, Time: QueryTime{Mode: "relative", Duration: "15m"}, Limit: 100, ResultMode: "table", Outcome: "success", DurationMS: 10}
	for i := 0; i < 502; i++ {
		if _, err := service.AppendHistory(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	rows, total, err := service.History(context.Background(), 0, 1, 100)
	if err != nil || total != 500 || len(rows) != 100 {
		t.Fatalf("retention count: %d %d %v", len(rows), total, err)
	}
	for _, row := range rows {
		if len(row.Recipe) != 0 {
			t.Fatal("raw history invented expression")
		}
	}
	if err := db.Model(&QueryHistory{}).Where("tenant_id = ? AND actor_id = ?", 11, 31).Update("expires_at", time.Now().Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendHistory(context.Background(), input); err != nil {
		t.Fatal(err)
	}
	if _, total, err := service.History(context.Background(), 0, 1, 100); err != nil || total != 1 {
		t.Fatalf("expired history retained: total%d error%v", total, err)
	}
	if err := db.Model(&source).Update("revision", source.Revision+1).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendHistory(context.Background(), input); !errors.Is(err, ErrWorkspaceConflict) {
		t.Fatalf("late source result stored: %v", err)
	}
}
