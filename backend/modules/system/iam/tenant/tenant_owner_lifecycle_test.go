package tenant

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type lifecycleWriterKey struct{}

// Pause the actual service after its status read. An unlocked implementation
// lets retirement commit before the stale activation; a locked implementation
// must serialize retirement behind the activation. Both must leave a tombstone.
func TestTenantOwnerLifecycleDelayedActivationCannotRevive(t *testing.T) {
	for _, writers := range []int{1, 6} {
		t.Run(fmt.Sprintf("writers_%d", writers), func(t *testing.T) {
			runDelayedActivationCannotRevive(t, writers)
		})
	}
}

func runDelayedActivationCannotRevive(t *testing.T, writers int) {
	t.Helper()
	db := setupTestDB(t)
	row := Tenant{Code: "lifecycle-race", Name: "Lifecycle Race", Status: "suspended", Metadata: `{"retained":true}`}
	require.NoError(t, db.Create(&row).Error)
	loaded, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	var wg sync.WaitGroup
	t.Cleanup(func() { unblock(); wg.Wait() })
	registerLifecycleReadPause(t, db, loaded, release)
	active, archived := "active", "archived"
	updater := NewService(db.WithContext(context.WithValue(context.Background(), lifecycleWriterKey{}, true)))
	results := make(chan error, writers)
	wg.Add(1)
	go func() { defer wg.Done(); results <- updater.UpdateTenant(row.ID, UpdateTenantDTO{Status: &active}) }()
	select {
	case <-loaded:
	case <-time.After(10 * time.Second):
		t.Fatal("updater did not reach controlled status-read barrier")
	}
	for i := 1; i < writers; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- updater.UpdateTenant(row.ID, UpdateTenantDTO{Status: &active}) }()
	}
	retire := func() error {
		service := NewService(db)
		if err := service.UpdateTenant(row.ID, UpdateTenantDTO{Status: &archived}); err != nil {
			return err
		}
		return service.DeleteTenant(row.ID)
	}
	// Probe the real InnoDB row, without a timing-based lock assumption.
	probe := probeTenantRowForUpdate(db, row.ID)
	if probe == nil {
		// Regression path: archive and delete while the stale read is held.
		require.NoError(t, retire())
		unblock()
	} else {
		requireLifecycleRowLocked(t, probe, db, row.ID)
		done := make(chan error, 1)
		wg.Add(1)
		go func() { defer wg.Done(); done <- retire() }()
		unblock()
		require.NoError(t, <-done)
	}
	requireLifecycleActivatorsDenied(t, results, writers)
	var tombstone Tenant
	require.NoError(t, db.Where("id = ?", row.ID).Take(&tombstone).Error)
	require.Equal(t, tenantStatusDeleted, tombstone.Status, "delayed activation revived a deleted tenant")
	require.NotEqual(t, row.Code, tombstone.Code)
	require.Equal(t, row.Metadata, tombstone.Metadata)
	_, err := NewService(db).GetTenantByID(row.ID)
	require.ErrorIs(t, err, ErrTenantNotFound)
}

func registerLifecycleReadPause(t *testing.T, db *gorm.DB, loaded, release chan struct{}) {
	t.Helper()
	var once sync.Once
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:pause-lifecycle-read", func(tx *gorm.DB) {
		if tx.Statement.Table == "tenants" && tx.Statement.Context.Value(lifecycleWriterKey{}) == true && tx.Error == nil {
			once.Do(func() { close(loaded); <-release })
		}
	}))
}

func probeTenantRowForUpdate(db *gorm.DB, id uint64) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var probedID uint64
		return tx.Table("tenants").Select("id").Where("id = ?", id).
			Clauses(clause.Locking{Strength: "UPDATE", Options: "NOWAIT"}).Scan(&probedID).Error
	})
}

func requireLifecycleRowLocked(t *testing.T, probeErr error, db *gorm.DB, id uint64) {
	t.Helper()
	var sqlErr *mysqlDriver.MySQLError
	require.True(t, errors.As(probeErr, &sqlErr))
	require.Equal(t, uint16(3572), sqlErr.Number)
	testmysql.AssertRowLocked(t, db, "tenants", id)
}

func requireLifecycleActivatorsDenied(t *testing.T, results <-chan error, writers int) {
	t.Helper()
	for i := 0; i < writers; i++ {
		err := <-results
		if err != nil {
			require.True(t, errors.Is(err, ErrTenantNotFound) || errors.Is(err, common.ErrConflict), "unexpected update error: %v", err)
		}
	}
}

func TestTenantOwnerLifecycleNoOpAndAtomicUpdate(t *testing.T) {
	db := setupTestDB(t)
	row := Tenant{Code: "lifecycle-atomic", Name: "Lifecycle Atomic", Status: "active", Metadata: `{"retained":true}`}
	require.NoError(t, db.Create(&row).Error)
	service := NewService(db)
	for _, dto := range []UpdateTenantDTO{{}, {Status: &row.Status}, {Name: &row.Name, Metadata: &row.Metadata}} {
		require.NoError(t, service.UpdateTenant(row.ID, dto), "valid no-op must succeed")
	}
	// A late field failure must not leave an earlier name/status write committed.
	injected := errors.New("test field failure")
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:fail-lifecycle-metadata", func(tx *gorm.DB) {
		if tx.Statement.Table == "tenants" {
			if fields, ok := tx.Statement.Dest.(map[string]interface{}); ok {
				if _, ok := fields["metadata"]; ok {
					tx.AddError(injected)
				}
			}
		}
	}))
	name, status, metadata := "Changed Name", "suspended", "changed"
	require.ErrorIs(t, service.UpdateTenant(row.ID, UpdateTenantDTO{Name: &name, Status: &status, Metadata: &metadata}), injected)
	var unchanged Tenant
	require.NoError(t, db.Where("id = ?", row.ID).Take(&unchanged).Error)
	require.Equal(t, row.Name, unchanged.Name)
	require.Equal(t, row.Status, unchanged.Status)
	require.Equal(t, row.Metadata, unchanged.Metadata)
}

func TestTenantOwnerLifecyclePlanUpdateWithOneConnection(t *testing.T) {
	db := setupTestDB(t)
	row := Tenant{Code: "lifecycle-plan", Name: "Lifecycle Plan", Status: "active", Plan: "free"}
	require.NoError(t, db.Create(&row).Error)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	plan := "basic"
	require.NoError(t, NewService(db.WithContext(ctx)).UpdateTenant(row.ID, UpdateTenantDTO{Plan: &plan}))
	var changed Tenant
	require.NoError(t, db.Where("id = ?", row.ID).Take(&changed).Error)
	require.Equal(t, plan, changed.Plan)
}
