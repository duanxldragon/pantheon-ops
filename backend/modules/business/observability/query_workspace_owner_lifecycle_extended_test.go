package observability

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type lifecycleReaderFunc func(context.Context, uint64, uint64) (businessauth.OwnerLifecycle, error)

func (f lifecycleReaderFunc) ReadOwnerLifecycle(ctx context.Context, tenantID, actorID uint64) (businessauth.OwnerLifecycle, error) {
	return f(ctx, tenantID, actorID)
}

func lifecycleView(t *testing.T, db *gorm.DB, source PrometheusDatasource, actorID uint64) SavedView {
	t.Helper()
	row := SavedView{TenantID: source.TenantID, ActorID: actorID, DatasourceID: source.ID, Recipe: []byte(`{"version":1}`), Revision: 1}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return row
}

func lifecycleCount(t *testing.T, db *gorm.DB, model any, id uint64, expected int64) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where("id = ?", id).Count(&count).Error; err != nil || count != expected {
		t.Fatalf("metadata count=%d expected=%d error=%v", count, expected, err)
	}
}

func lifecycleFixedFailure(t *testing.T, err error) {
	t.Helper()
	if !errors.Is(err, ErrWorkspaceMaintenance) || err.Error() != "observability.history_unavailable" {
		t.Fatalf("maintenance failure was not sanitized: %v", err)
	}
}

func TestWorkspaceLifecycleActualHealthyPoolAndExistenceSemantics(t *testing.T) {
	for _, capacity := range []int{2, 0} {
		t.Run(fmt.Sprintf("pool%d", capacity), func(t *testing.T) {
			db := lifecycleMySQL(t)
			_, reader := actualOwnerLifecycle(t, db)
			if err := db.Exec("INSERT INTO system_user (id,tenant_id,status,deleted_at) VALUES (10,0,1,NOW(3))").Error; err != nil {
				t.Fatal(err)
			}
			legacy := lifecycleSource(t, db, 0, "legacy-data")
			if err := db.Model(&legacy).Updates(map[string]any{"password": "synthetic-retained-ciphertext", "bearer_token": "synthetic-retained-token"}).Error; err != nil {
				t.Fatal(err)
			}
			var legacyBefore PrometheusDatasource
			db.First(&legacyBefore, legacy.ID)
			before := sha256.Sum256([]byte(legacyBefore.Password + "|" + legacyBefore.BearerToken))
			type pair struct {
				history QueryHistory
				view    SavedView
				keep    bool
			}
			var rows []pair
			for i, owner := range []struct {
				tenant, actor uint64
				keep          bool
			}{
				{0, 7, true}, {11, 8, true}, {11, 9, true}, {12, 7, true}, {13, 7, true},
				{11, 999, false}, {14, 7, false}, {22, 7, false}, {11, 10, false},
			} {
				source := lifecycleSource(t, db, owner.tenant, fmt.Sprintf("owner-semantics-%d", i))
				rows = append(rows, pair{lifecycleHistory(t, db, source, owner.actor), lifecycleView(t, db, source, owner.actor), owner.keep})
			}
			pool, _ := db.DB()
			pool.SetMaxOpenConns(capacity)
			if err := (&queryWorkspaceSweep{db: db, reader: reader}).run(context.Background()); err != nil {
				t.Fatal(err)
			}
			for _, row := range rows {
				expected := int64(0)
				if row.keep {
					expected = 1
				}
				lifecycleCount(t, db, &QueryHistory{}, row.history.ID, expected)
				lifecycleCount(t, db, &SavedView{}, row.view.ID, expected)
			}
			var guards int64
			if err := db.Model(&QueryOwnerBudget{}).Count(&guards).Error; err != nil || guards != 4 {
				t.Fatalf("orphan guards were lost or live owners unnecessarily guarded: %d %v", guards, err)
			}
			var legacyAfter PrometheusDatasource
			if err := db.First(&legacyAfter, legacy.ID).Error; err != nil || !reflect.DeepEqual(legacyBefore, legacyAfter) || before != sha256.Sum256([]byte(legacyAfter.Password+"|"+legacyAfter.BearerToken)) {
				t.Fatal("legacy datasource/ciphertext changed")
			}
			t.Logf("actual configured pool=%d: five live owner pairs retained, four absent owner pairs removed; legacy digest and four guards retained", capacity)
		})
	}
}

func TestWorkspaceLifecycleUnknownRetainsWholeExpiredAndSourceOrphanRows(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "unknown-owner")
	history := lifecycleHistory(t, db, source, 999)
	view := lifecycleView(t, db, source, 999)
	if err := db.Model(&history).Updates(map[string]any{"expires_at": time.Now().Add(-40 * 24 * time.Hour), "recipe": []byte(`{"version":1}`)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&source).Error; err != nil {
		t.Fatal(err)
	}
	var beforeHistory, afterHistory QueryHistory
	var beforeView, afterView SavedView
	db.First(&beforeHistory, history.ID)
	db.First(&beforeView, view.ID)
	_, actual := actualOwnerLifecycle(t, db)
	reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		facts, err := actual.ReadOwnerLifecycle(ctx, tenant, actor)
		if err != nil {
			return facts, err
		}
		return businessauth.OwnerLifecycle{TenantExists: true}, errors.New("synthetic-private-provider-error")
	})
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if err := db.First(&afterHistory, history.ID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&afterView, view.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeHistory, afterHistory) || !reflect.DeepEqual(beforeView, afterView) {
		t.Fatal("unknown owner changed whole retained metadata")
	}
	sweep.reader = actual
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	lifecycleCount(t, db, &QueryHistory{}, history.ID, 0)
	lifecycleCount(t, db, &SavedView{}, view.ID, 0)
}

func TestWorkspaceLifecycleFreshLockedFactsRetainRevivedOrUnknownOwner(t *testing.T) {
	for _, outcome := range []string{"live", "unknown"} {
		t.Run(outcome, func(t *testing.T) {
			db := lifecycleMySQL(t)
			source := lifecycleSource(t, db, 11, "fresh-lock-"+outcome)
			row := lifecycleHistory(t, db, source, 999)
			_, actual := actualOwnerLifecycle(t, db)
			calls := 0
			reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
				calls++
				if calls == 2 {
					if outcome == "unknown" {
						return businessauth.OwnerLifecycle{}, businessauth.ErrOwnerLifecycleUnavailable
					}
					if err := db.WithContext(ctx).Exec("INSERT INTO system_user (id,tenant_id,status) VALUES (999,0,1)").Error; err != nil {
						return businessauth.OwnerLifecycle{}, err
					}
				}
				return actual.ReadOwnerLifecycle(ctx, tenant, actor)
			})
			err := (&queryWorkspaceSweep{db: db, reader: reader}).run(context.Background())
			if outcome == "unknown" {
				lifecycleFixedFailure(t, err)
			} else if err != nil {
				t.Fatal(err)
			}
			if calls != 2 {
				t.Fatalf("fresh public read missing: %d", calls)
			}
			lifecycleCount(t, db, &QueryHistory{}, row.ID, 1)
		})
	}
}

func TestWorkspaceLifecycleCurrentSourceIdentityIsRechecked(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "old-source")
	newSource := lifecycleSource(t, db, 11, "new-source")
	row := lifecycleHistory(t, db, source, 999)
	_, actual := actualOwnerLifecycle(t, db)
	calls := 0
	reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		calls++
		facts, err := actual.ReadOwnerLifecycle(ctx, tenant, actor)
		if calls == 1 && err == nil {
			err = db.WithContext(ctx).Model(&QueryHistory{}).Where("id = ?", row.ID).Update("datasource_id", newSource.ID).Error
		}
		return facts, err
	})
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("changed source was used under the wrong source lock")
	}
	lifecycleCount(t, db, &QueryHistory{}, row.ID, 1)
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	lifecycleCount(t, db, &QueryHistory{}, row.ID, 0)
}

func TestWorkspaceLifecycleActualSlotStolenAfterCapacityObservation(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "occupied-race")
	row := lifecycleHistory(t, db, source, 999)
	owner := QueryOwner{TenantID: 11, ActorID: 999}
	if err := lockQueryOwners(db, []QueryOwner{owner}); err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	_, actual := actualOwnerLifecycle(t, db)
	calls := 0
	reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		calls++
		if calls != 2 {
			return actual.ReadOwnerLifecycle(ctx, tenant, actor)
		}
		// This callback is reached only after consumer transaction, source and
		// metadata locks: acquire the formerly spare second pool slot here.
		connection, err := pool.Conn(ctx)
		if err != nil {
			return businessauth.OwnerLifecycle{}, err
		}
		defer connection.Close()
		if pool.Stats().InUse != 2 {
			return businessauth.OwnerLifecycle{}, errors.New("slot barrier missing")
		}
		return actual.ReadOwnerLifecycle(ctx, tenant, actor)
	})
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	started := time.Now()
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if calls != 2 || time.Since(started) > 4*time.Second {
		t.Fatal("occupied locked-reader race not bounded")
	}
	lifecycleCount(t, db, &QueryHistory{}, row.ID, 1)
	// Actual SQL lock acquisition, rather than deadline completion alone, proves
	// the consumer released its transaction, connection and owner/source locks.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := (&QueryWorkspaceService{db: db, owner: owner}).ownedTransaction(ctx, []uint64{source.ID}, func(tx *gorm.DB) error {
		return tx.Raw("SELECT id FROM obs_query_history WHERE id = ? FOR UPDATE", row.ID).Scan(new(uint64)).Error
	}); err != nil {
		t.Fatal("locks/connections not released after rollback")
	}
	if pool.Stats().InUse != 0 {
		t.Fatal("consumer connection leaked")
	}
	sweep.reader = actual
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	lifecycleCount(t, db, &QueryHistory{}, row.ID, 0)
	var guards int64
	db.Model(&QueryOwnerBudget{}).Count(&guards)
	if guards != 1 {
		t.Fatal("retained guard changed")
	}
	t.Log("actual pool2 slot-steal timed out/rolled back; SQL owner/source/metadata lock probe succeeded; recovery wrap removed orphan")
}

func lifecycleFairRows(t *testing.T, db *gorm.DB, source PrometheusDatasource) (QueryHistory, SavedView) {
	t.Helper()
	for i := 0; i < 120; i++ {
		lifecycleHistory(t, db, source, 7)
		lifecycleView(t, db, source, 7)
	}
	return lifecycleHistory(t, db, source, 999), lifecycleView(t, db, source, 999)
}

func TestWorkspaceLifecycleActualFullOccupancyThenRecovery(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "full-occupancy")
	orphan, orphanView := lifecycleFairRows(t, db, source)
	_, reader := actualOwnerLifecycle(t, db)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	before := sweep.cursors
	if before[0].afterID != 100 || before[1].afterID != 100 {
		t.Fatal("initial bounded cursor missing")
	}
	one, err := pool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer one.Close()
	two, err := pool.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer two.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	lifecycleFixedFailure(t, sweep.run(ctx))
	cancel()
	if sweep.cursors != before {
		t.Fatal("full occupancy fabricated enumeration progress")
	}
	if err := one.Close(); err != nil {
		t.Fatal(err)
	}
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if sweep.cursors != ([2]workspaceMaintenanceCursor{}) {
		t.Fatal("partial release did not visit/wrap tail including capacity-unavailable orphan")
	}
	lifecycleCount(t, db, &QueryHistory{}, orphan.ID, 1)
	lifecycleCount(t, db, &SavedView{}, orphanView.ID, 1)
	if err := two.Close(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := sweep.run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	lifecycleCount(t, db, &QueryHistory{}, orphan.ID, 0)
	lifecycleCount(t, db, &SavedView{}, orphanView.ID, 0)
	t.Log("all slots held: cursor unchanged; one slot released: visits/wrap retained orphan; all released: both later orphans reached/deleted within two finite passes")
}

func TestWorkspaceLifecycleActualRuntimePoolReductionAndRecovery(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "pool-reduction")
	orphan, orphanView := lifecycleFairRows(t, db, source)
	_, reader := actualOwnerLifecycle(t, db)
	pool, _ := db.DB()
	pool.SetMaxOpenConns(2)
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(1)
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if pool.Stats().MaxOpenConnections != 1 || sweep.cursors != ([2]workspaceMaintenanceCursor{}) {
		t.Fatal("runtime pool1 changed capacity or failed visited-tail wrap")
	}
	lifecycleCount(t, db, &QueryHistory{}, orphan.ID, 1)
	lifecycleCount(t, db, &SavedView{}, orphanView.ID, 1)
	pool.SetMaxOpenConns(2)
	for i := 0; i < 2; i++ {
		if err := sweep.run(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	lifecycleCount(t, db, &QueryHistory{}, orphan.ID, 0)
	lifecycleCount(t, db, &SavedView{}, orphanView.ID, 0)
	t.Log("actual owned pool2->1 retained rows and wrapped visited tails without self-starvation; pool1->2 recovered both later orphans within two finite passes")
}

func TestWorkspaceLifecycleFiniteCeilingUnknownPrefixAndContinuousInserts(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "finite-cycle")
	for i := 0; i < 120; i++ {
		actor := uint64(7)
		if i%2 == 0 {
			actor = 8
		}
		lifecycleHistory(t, db, source, actor)
		lifecycleView(t, db, source, actor)
	}
	orphan := lifecycleHistory(t, db, source, 999)
	orphanView := lifecycleView(t, db, source, 999)
	_, actual := actualOwnerLifecycle(t, db)
	reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		facts, err := actual.ReadOwnerLifecycle(ctx, tenant, actor)
		if err == nil && actor == 8 {
			return facts, businessauth.ErrOwnerLifecycleUnavailable
		}
		return facts, err
	})
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if sweep.cursors[0] != (workspaceMaintenanceCursor{100, 121}) || sweep.cursors[1] != (workspaceMaintenanceCursor{100, 121}) || sweep.first != 1 {
		t.Fatal("100-visit limit/finite ceiling/table rotation missing")
	}
	var later []QueryHistory
	for i := 0; i < 5; i++ {
		later = append(later, lifecycleHistory(t, db, source, 999))
		lifecycleView(t, db, source, 999)
	}
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if sweep.cursors != ([2]workspaceMaintenanceCursor{}) {
		t.Fatal("new inserts postponed wrap beyond captured121 ceiling")
	}
	lifecycleCount(t, db, &QueryHistory{}, orphan.ID, 0)
	lifecycleCount(t, db, &SavedView{}, orphanView.ID, 0)
	for _, row := range later {
		lifecycleCount(t, db, &QueryHistory{}, row.ID, 1)
	}
	// Sparse/deleted IDs do not stall keyset progress. The new cycle still has
	// a finite126 ceiling while more inserts arrive during its100-row prefix.
	if err := db.Where("id = ?", 10).Delete(&QueryHistory{}).Error; err != nil {
		t.Fatal(err)
	}
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	if sweep.cursors[0].cycleMaxID != 126 || sweep.cursors[1].cycleMaxID != 126 {
		t.Fatal("next cycle failed to capture finite current maximum")
	}
	lifecycleHistory(t, db, source, 999)
	lifecycleView(t, db, source, 999)
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	for _, row := range later {
		lifecycleCount(t, db, &QueryHistory{}, row.ID, 0)
	}
	if sweep.cursors != ([2]workspaceMaintenanceCursor{}) {
		t.Fatal("continuous insert second cycle did not wrap")
	}
	t.Log("both tables:100 live/unknown prefix visits,121 ceiling then wrap despite126 inserts; sparse ID skipped and126 ceiling wrapped despite127 inserts")
}

func TestWorkspaceLifecycleTableRotationDeadlineAndOverlap(t *testing.T) {
	db := lifecycleMySQL(t)
	source := lifecycleSource(t, db, 11, "rotation-deadline")
	history := lifecycleHistory(t, db, source, 999)
	laterHistory := lifecycleHistory(t, db, source, 1001)
	view := lifecycleView(t, db, source, 1000)
	_, actual := actualOwnerLifecycle(t, db)
	entered, release := make(chan struct{}), make(chan struct{})
	reader := lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		if actor == 999 {
			close(entered)
			select {
			case <-release:
			case <-ctx.Done():
			}
			return businessauth.OwnerLifecycle{}, businessauth.ErrOwnerLifecycleUnavailable
		}
		return actual.ReadOwnerLifecycle(ctx, tenant, actor)
	})
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- sweep.run(ctx) }()
	<-entered
	lifecycleFixedFailure(t, sweep.run(context.Background()))
	lifecycleFixedFailure(t, <-done)
	close(release)
	if sweep.cursors[0].afterID != history.ID || sweep.cursors[1] != (workspaceMaintenanceCursor{}) || sweep.first != 1 {
		t.Fatal("deadline skipped unvisited table or lost visited cursor")
	}
	var order []uint64
	sweep.reader = lifecycleReaderFunc(func(ctx context.Context, tenant, actor uint64) (businessauth.OwnerLifecycle, error) {
		order = append(order, actor)
		return actual.ReadOwnerLifecycle(ctx, tenant, actor)
	})
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(order) == 0 || order[0] != 1000 {
		t.Fatal("next pass did not start deadline-starved saved-view table")
	}
	lifecycleCount(t, db, &SavedView{}, view.ID, 0)
	lifecycleCount(t, db, &QueryHistory{}, laterHistory.ID, 0)
	// A completed tail wraps on the next pass; the visited history now retries.
	if err := sweep.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	lifecycleCount(t, db, &QueryHistory{}, history.ID, 0)
}

type lifecycleOldStartupReader struct{}

func (*lifecycleOldStartupReader) RefreshPrincipal(context.Context, businessauth.Identity) (businessauth.Principal, error) {
	return businessauth.Principal{}, businessauth.ErrDenied
}
func (*lifecycleOldStartupReader) Authorize(context.Context, businessauth.Input) (businessauth.Authorization, error) {
	return businessauth.Authorization{}, businessauth.ErrDenied
}

func TestWorkspaceLifecycleStartupCapabilityAndPoolGuard(t *testing.T) {
	if mode := os.Getenv("PANTHEON_O4_LIFECYCLE_STARTUP_CASE"); mode != "" {
		db := lifecycleMySQL(t)
		pool, _ := db.DB()
		expected := contracts.ErrBusinessProviders
		switch mode {
		case "missing":
		case "incompatible":
			if err := contracts.RegisterBusinessAuthorizationReader(&lifecycleOldStartupReader{}); err != nil {
				t.Fatal(err)
			}
		case "typednil":
			var reader *lifecycleOldStartupReader
			if err := contracts.RegisterBusinessAuthorizationReader(reader); !errors.Is(err, contracts.ErrBusinessProviders) {
				t.Fatal("typed nil provider accepted")
			}
		default:
			provider, _ := actualOwnerLifecycle(t, db)
			if err := contracts.RegisterBusinessAuthorizationReader(provider); err != nil {
				t.Fatal(err)
			}
			expected = ErrWorkspaceMaintenance
		}
		var store = *db
		switch mode {
		case "pool1":
			pool.SetMaxOpenConns(1)
		case "closed":
			if err := pool.Close(); err != nil {
				t.Fatal(err)
			}
		case "nil":
			store = gorm.DB{}
		case "invalid":
			store = gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{}}
		case "pool2":
			pool.SetMaxOpenConns(2)
			expected = nil
		case "pool0":
			pool.SetMaxOpenConns(0)
			expected = nil
		}
		target := &store
		if mode == "nil" {
			target = nil
		}
		reader, err := QueryWorkspaceMaintenanceReady(target)
		if expected == nil {
			if err != nil || reader == nil {
				t.Fatal("supported pool rejected")
			}
			return
		}
		if !errors.Is(err, expected) || err.Error() != expected.Error() {
			t.Fatalf("wrong startup branch: %v", err)
		}
		if mode == "closed" {
			gin.SetMode(gin.TestMode)
			router := gin.New()
			var recovered any
			func() {
				defer func() { recovered = recover() }()
				InitObservabilityModule(router.Group("/api/v1"), target, nil)
			}()
			failure, ok := recovered.(error)
			if !ok || !errors.Is(failure, expected) || len(router.Routes()) != 0 {
				t.Fatal("closed-pool startup did not fail before routes")
			}
			return
		}
		before, err := db.Migrator().GetTables()
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(before)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			InitObservabilityModule(router.Group("/api/v1"), target, nil)
		}()
		failure, ok := recovered.(error)
		if !ok || !errors.Is(failure, expected) {
			t.Fatalf("module startup did not reject before readiness/seed: %v", recovered)
		}
		after, err := db.Migrator().GetTables()
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(after)
		if !reflect.DeepEqual(before, after) || len(router.Routes()) != 0 {
			t.Fatal("rejected startup wrote schema/routes")
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "incompatible", "typednil", "nil", "invalid", "closed", "pool1", "pool2", "pool0"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, "-test.run=^TestWorkspaceLifecycleStartupCapabilityAndPoolGuard$", "-test.v")
			command.Env = append(os.Environ(), "PANTHEON_O4_LIFECYCLE_STARTUP_CASE="+mode)
			output, err := command.CombinedOutput()
			t.Log(string(output))
			if err != nil {
				t.Fatalf("isolated public startup failed: %v", err)
			}
		})
	}
}
