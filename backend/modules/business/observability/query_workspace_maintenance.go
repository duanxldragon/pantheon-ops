package observability

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const queryWorkspaceMaintenanceName = "observability-query-workspace"
const queryWorkspaceMaintenanceBatch = 100

// ErrWorkspaceMaintenance is returned while query history is unavailable.
var ErrWorkspaceMaintenance = errors.New("observability.history_unavailable")

func workspaceMaintenancePool(db *gorm.DB) (*sql.DB, error) {
	if db == nil || db.Config == nil || db.Statement == nil {
		return nil, ErrWorkspaceMaintenance
	}
	pool, err := db.DB()
	if err != nil || pool == nil {
		return nil, ErrWorkspaceMaintenance
	}
	return pool, nil
}

// QueryWorkspaceMaintenanceReady checks the required public capability and pool
// configuration before business bootstrap, seeds or routes can have effects.
// The capacity check is not a reservation or a guarantee of runtime liveness.
func QueryWorkspaceMaintenanceReady(db *gorm.DB) (businessauth.OwnerLifecycleReader, error) {
	reader, err := contracts.RegisteredBusinessOwnerLifecycleReader()
	if err != nil {
		return nil, contracts.ErrBusinessProviders
	}
	pool, err := workspaceMaintenancePool(db)
	if err != nil || pool.Stats().MaxOpenConnections == 1 {
		return nil, ErrWorkspaceMaintenance
	}
	// Stats cannot identify a closed pool. A bounded driver ping checks actual
	// availability without issuing lifecycle SQL or changing global pool limits.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		return nil, ErrWorkspaceMaintenance
	}
	return reader, nil
}

func registerQueryWorkspaceMaintenance(db *gorm.DB, reader businessauth.OwnerLifecycleReader, registry *maintenance.Registry) {
	sweep := &queryWorkspaceSweep{db: db, reader: reader}
	registry.Register(maintenance.Task{Name: queryWorkspaceMaintenanceName, Interval: time.Hour, Run: sweep.run})
}

type workspaceMaintenanceRow struct {
	ID           uint64
	TenantID     uint64
	ActorID      uint64
	DatasourceID uint64
}

type workspaceMaintenanceCursor struct{ afterID, cycleMaxID uint64 }

// Scheduled passes retain their cursors and alternate the first table. No DB
// locks are held between visits; directly overlapping passes fail immediately.
type queryWorkspaceSweep struct {
	db      *gorm.DB
	reader  businessauth.OwnerLifecycleReader
	running atomic.Bool
	cursors [2]workspaceMaintenanceCursor
	first   int
}

func workspaceMaintenanceModel(history bool) any {
	if history {
		return &QueryHistory{}
	}
	return &SavedView{}
}

func workspaceMaintenanceCandidates(db *gorm.DB, history bool, now time.Time) *gorm.DB {
	model := workspaceMaintenanceModel(history)
	liveSources := db.Session(&gorm.Session{NewDB: true}).Model(&PrometheusDatasource{}).
		Select("id")
	if history {
		liveSources = liveSources.Where("tenant_id = obs_query_history.tenant_id")
		return db.Model(model).Where("expires_at <= ? OR datasource_id NOT IN (?)", now, liveSources)
	}
	liveSources = liveSources.Where("tenant_id = obs_saved_view.tenant_id")
	return db.Model(model).Where("datasource_id NOT IN (?)", liveSources)
}

// SweepQueryWorkspace is a maintenance-only one-off pass, never an HTTP route.
// Scheduled maintenance uses the persistent sweep above for finite-cycle retry.
func SweepQueryWorkspace(ctx context.Context, db *gorm.DB) error {
	if _, err := workspaceMaintenancePool(db); err != nil {
		return ErrWorkspaceMaintenance
	}
	reader, err := contracts.RegisteredBusinessOwnerLifecycleReader()
	if err != nil {
		return ErrWorkspaceMaintenance
	}
	return (&queryWorkspaceSweep{db: db, reader: reader}).run(ctx)
}

func (s *queryWorkspaceSweep) run(ctx context.Context) error {
	if ctx == nil || ctx.Err() != nil || s.reader == nil || !s.running.CompareAndSwap(false, true) {
		return ErrWorkspaceMaintenance
	}
	defer s.running.Store(false)
	if _, err := workspaceMaintenancePool(s.db); err != nil {
		return ErrWorkspaceMaintenance
	}
	// SQL diagnostics can contain private owner/source identifiers. Only the
	// registry's fixed aggregate failure escapes this maintenance pass.
	originalDB := s.db
	s.db = s.db.Session(&gorm.Session{Logger: logger.Discard})
	defer func() { s.db = originalDB }()
	bounded, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	first := s.first
	s.first = 1 - first
	var failed bool
	for _, table := range []int{first, 1 - first} {
		if s.scanTable(bounded, table) != nil {
			failed = true
		}
	}
	if failed || bounded.Err() != nil {
		return ErrWorkspaceMaintenance
	}
	return nil
}

func (s *queryWorkspaceSweep) scanTable(ctx context.Context, table int) error {
	if ctx.Err() != nil {
		return ErrWorkspaceMaintenance
	}
	q := s.db.WithContext(ctx).Model(workspaceMaintenanceModel(table == 0))
	cursor := s.cursors[table]
	if cursor.cycleMaxID == 0 {
		if err := q.Select("COALESCE(MAX(id),0)").Scan(&cursor.cycleMaxID).Error; err != nil {
			return ErrWorkspaceMaintenance
		}
	}
	var rows []workspaceMaintenanceRow
	if err := q.Select("id,tenant_id,actor_id,datasource_id").
		Where("id > ? AND id <= ?", cursor.afterID, cursor.cycleMaxID).
		Order("id").Limit(queryWorkspaceMaintenanceBatch).Find(&rows).Error; err != nil {
		// Even a captured ceiling is not committed when enumeration cannot run.
		return ErrWorkspaceMaintenance
	}
	var failed bool
	for _, row := range rows {
		if ctx.Err() != nil {
			return ErrWorkspaceMaintenance
		}
		recordCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if s.visit(recordCtx, table == 0, row) != nil {
			failed = true
		}
		cancel()
		cursor.afterID = row.ID
		s.cursors[table] = cursor
	}
	if len(rows) < queryWorkspaceMaintenanceBatch || cursor.afterID >= cursor.cycleMaxID {
		s.cursors[table] = workspaceMaintenanceCursor{}
	}
	if failed {
		return ErrWorkspaceMaintenance
	}
	return nil
}

func workspaceMaintenanceRemoval(db *gorm.DB, history bool, row workspaceMaintenanceRow) (bool, error) {
	var count int64
	err := ownerPredicate(workspaceMaintenanceCandidates(db, history, time.Now().UTC()),
		QueryOwner{TenantID: row.TenantID, ActorID: row.ActorID}).Where("id = ?", row.ID).Count(&count).Error
	return count != 0, err
}

func (s *queryWorkspaceSweep) visit(ctx context.Context, history bool, row workspaceMaintenanceRow) error {
	facts, err := s.reader.ReadOwnerLifecycle(ctx, row.TenantID, row.ActorID)
	if err != nil || ctx.Err() != nil {
		return ErrWorkspaceMaintenance
	}
	if facts.TenantExists && facts.ActorExists {
		remove, err := workspaceMaintenanceRemoval(s.db.WithContext(ctx), history, row)
		if err != nil {
			return ErrWorkspaceMaintenance
		}
		if !remove {
			return nil
		}
	}
	pool, err := workspaceMaintenancePool(s.db)
	if err != nil {
		return ErrWorkspaceMaintenance
	}
	stats := pool.Stats()
	if stats.MaxOpenConnections != 0 && stats.MaxOpenConnections-stats.InUse < 2 {
		return ErrWorkspaceMaintenance
	}
	owner := QueryOwner{TenantID: row.TenantID, ActorID: row.ActorID}
	service := &QueryWorkspaceService{db: s.db, owner: owner}
	err = service.ownedTransaction(ctx, []uint64{row.DatasourceID}, func(tx *gorm.DB) error {
		var current workspaceMaintenanceRow
		q := ownerPredicate(tx.Model(workspaceMaintenanceModel(history)), owner).Where("id = ?", row.ID)
		if err := q.Select("id,tenant_id,actor_id,datasource_id").Clauses(clause.Locking{Strength: "UPDATE"}).Take(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		if current != row {
			// A changed source/owner needs a later pass with the correct lock order.
			return nil
		}
		remove, err := workspaceMaintenanceRemoval(tx, history, current)
		if err != nil {
			return err
		}
		// The spare-capacity observation was not a reservation. This second
		// public read is fresh after all locks and may time out: rollback keeps
		// the whole record, even when expired or a source orphan.
		fresh, err := s.reader.ReadOwnerLifecycle(ctx, row.TenantID, row.ActorID)
		if err != nil || ctx.Err() != nil {
			return ErrWorkspaceMaintenance
		}
		if fresh.TenantExists && fresh.ActorExists && !remove {
			return nil
		}
		return ownerPredicate(tx.Model(workspaceMaintenanceModel(history)), owner).
			Where("id = ?", row.ID).Delete(workspaceMaintenanceModel(history)).Error
	})
	if err != nil {
		return ErrWorkspaceMaintenance
	}
	return nil
}
