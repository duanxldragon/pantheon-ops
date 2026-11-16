package observability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func maintenanceExistingOwner(context.Context, uint64, uint64) (businessauth.OwnerLifecycle, error) {
	return businessauth.OwnerLifecycle{TenantExists: true, ActorExists: true}, nil
}

func TestWorkspaceMaintenanceBoundedExpiryAndSourceOrphans(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "maintenance", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for i := 0; i < 120; i++ {
		row := QueryHistory{TenantID: 11, ActorID: 31, DatasourceID: source.ID, SourceKind: source.Type, Time: []byte(`{"mode":"relative","duration":"15m"}`), Limit: 100, ResultMode: "table", Outcome: "success", ExpiresAt: now.Add(-time.Hour)}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	retained := QueryHistory{TenantID: 11, ActorID: 32, DatasourceID: source.ID, SourceKind: source.Type, Time: []byte(`{"mode":"relative","duration":"15m"}`), Limit: 100, ResultMode: "table", Outcome: "success", ExpiresAt: now.Add(time.Hour)}
	if err := db.Create(&retained).Error; err != nil {
		t.Fatal(err)
	}
	view := SavedView{TenantID: 22, ActorID: 32, DatasourceID: source.ID, Recipe: []byte(`{}`), Revision: 1}
	if err := db.Create(&view).Error; err != nil {
		t.Fatal(err)
	}
	registry := maintenance.NewRegistry()
	registerQueryWorkspaceMaintenance(db, lifecycleReaderFunc(maintenanceExistingOwner), registry)
	result, registered := registry.RunNow(context.Background(), queryWorkspaceMaintenanceName)
	if !registered || result.Outcome != maintenance.OutcomeSucceeded {
		t.Fatalf("maintenance not wired: %v %v", registered, result.Outcome)
	}
	var histories, views int64
	db.Model(&QueryHistory{}).Count(&histories)
	db.Model(&SavedView{}).Count(&views)
	if histories != 21 || views != 0 {
		t.Fatalf("unbounded/incorrect sweep: history%d views%d", histories, views)
	}
	if err := (&queryWorkspaceSweep{db: db, reader: lifecycleReaderFunc(maintenanceExistingOwner)}).run(context.Background()); err != nil {
		t.Fatal(err)
	}
	db.Model(&QueryHistory{}).Count(&histories)
	if histories != 1 {
		t.Fatalf("live metadata lost: %d", histories)
	}
	var guards int64
	db.Model(&QueryOwnerBudget{}).Count(&guards)
	if guards != 2 {
		t.Fatalf("owner guards missing/reclaimed: %d", guards)
	}
	if err := db.Delete(&source).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&queryWorkspaceSweep{db: db, reader: lifecycleReaderFunc(maintenanceExistingOwner)}).run(context.Background()); err != nil {
		t.Fatal(err)
	}
	db.Model(&QueryHistory{}).Count(&histories)
	if histories != 0 {
		t.Fatal("soft-deleted source metadata retained")
	}
}

func TestWorkspaceMaintenanceFailsClosedWithoutRawDatabaseErrors(t *testing.T) {
	if err := SweepQueryWorkspace(context.Background(), nil); !errors.Is(err, ErrWorkspaceMaintenance) {
		t.Fatalf("nil store did not fail safely: %v", err)
	}
	db := testmysql.Open(t)
	if err := (&queryWorkspaceSweep{db: db, reader: lifecycleReaderFunc(maintenanceExistingOwner)}).run(context.Background()); !errors.Is(err, ErrWorkspaceMaintenance) || err.Error() != "observability.history_unavailable" {
		t.Fatalf("missing schema leaked error: %v", err)
	}
}

func TestWorkspaceMaintenanceRechecksExpiryUnderOwnerGuard(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "coalesced-sweep", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	row := QueryHistory{TenantID: 11, ActorID: 31, DatasourceID: source.ID, SourceKind: source.Type, Time: []byte(`{"mode":"relative","duration":"15m"}`), Limit: 100, ResultMode: "table", Outcome: "success", ExpiresAt: time.Now().Add(-time.Hour)}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	owner := QueryOwner{TenantID: 11, ActorID: 31}
	if err := lockQueryOwners(db, []QueryOwner{owner}); err != nil {
		t.Fatal(err)
	}
	tx := db.Begin()
	defer tx.Rollback()
	if err := lockQueryOwners(tx, []QueryOwner{owner}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- (&queryWorkspaceSweep{db: db, reader: lifecycleReaderFunc(maintenanceExistingOwner)}).run(ctx)
	}()
	// Wait for the actual maintenance INSERT to block on this fixture's owner
	// guard, proving its candidate enumeration preceded the simulated poll.
	for {
		var waiting int64
		err := db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.PROCESSLIST WHERE DB=DATABASE() AND STATE='update' AND INFO LIKE ?", "%INSERT INTO `obs_query_owner_budget`%").Scan(&waiting).Error
		if err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		select {
		case err := <-done:
			t.Fatalf("sweep did not wait for owner guard: %v", err)
		case <-ctx.Done():
			t.Fatal("maintenance owner guard barrier timed out")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if err := ownerPredicate(tx.Model(&QueryHistory{}), owner).Where("id = ?", row.ID).Update("expires_at", time.Now().Add(time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var retained QueryHistory
	if err := db.First(&retained, row.ID).Error; err != nil {
		t.Fatalf("freshly extended coalesced history deleted: %v", err)
	}
}
