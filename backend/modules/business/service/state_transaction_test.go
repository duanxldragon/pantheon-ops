package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

func TestStateTransactionValidationRevisionAndLifecycle(t *testing.T) {
	m := setupServiceTestDB(t)
	app := Application{Code: "bound", Name: "bound", BusinessScopeID: 10, DeptID: 7}
	if err := m.db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	svc := Service{ApplicationID: app.ID, Code: "bound", Name: "bound", RuntimeType: "vm"}
	if err := m.db.Create(&svc).Error; err != nil {
		t.Fatal(err)
	}
	instance := ServiceInstance{ServiceID: svc.ID, Environment: "test", TargetType: "vm", HostID: 55, DesiredState: "stopped", ObservedState: "unknown", HealthState: "unknown", LifecycleVersion: 7}
	if err := m.db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	rootDB := m.db
	factory := m.WithServiceInstanceStateTransaction
	injected := errors.New("caller rollback")
	var bound bizcap.ServiceInstanceStateApplier
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "owner-command")
	readCount, writeCount, referenceBinds := 0, 0, 0
	err := biztx.ReadCommittedTransaction(ctx, m.db, func(tx *gorm.DB) error {
		// Neither original owner DB nor an independent connection can see these
		// uncommitted validation/revision facts.
		if err := tx.Model(&Application{}).Where("id = ?", app.ID).Update("dept_id", 8).Error; err != nil {
			return err
		}
		if err := tx.Model(&ServiceInstance{}).Where("id = ?", instance.ID).Updates(map[string]any{"observed_state": "installing", "lifecycle_version": 8}).Error; err != nil {
			return err
		}
		checkDB := func(db *gorm.DB) {
			if db.Statement.ConnPool != tx.Statement.ConnPool {
				t.Error("owner operation escaped caller SQL transaction")
			}
			if !db.SkipDefaultTransaction {
				t.Error("owner command enabled independent default write transaction")
			}
			if db.Statement.Context.Value(contextKey{}) != "owner-command" {
				t.Error("owner validation lost command context")
			}
		}
		m.SetDeploymentReferenceReader(&retireReferenceReader{bind: func(db *gorm.DB) bizcap.DeploymentReferenceReader {
			checkDB(db)
			referenceBinds++
			if err := biztx.RequireReadCommittedTransaction(db); err != nil {
				return &retireReferenceReader{err: err}
			}
			return &retireReferenceReader{}
		}})
		if err := m.db.Callback().Query().Before("gorm:query").Register("bound_read", func(db *gorm.DB) { checkDB(db); readCount++ }); err != nil {
			return err
		}
		if err := m.db.Callback().Update().Before("gorm:update").Register("bound_write", func(db *gorm.DB) { checkDB(db); writeCount++ }); err != nil {
			return err
		}
		defer m.db.Callback().Query().Remove("bound_read")
		defer m.db.Callback().Update().Remove("bound_write")
		bound = factory(tx)
		wrongScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
		allowedScope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 8}
		req := bizcap.ServiceInstanceStateTransition{InstanceID: instance.ID, Action: "install", ExpectedLifecycleVersion: 8, ObservedState: "stopped", CurrentVersion: "new", CorrelationID: "bound-install"}
		if err := bound.ApplyServiceInstanceState(ctx, req, "owner", wrongScope); err == nil || err.Error() != errInstanceNotFound {
			t.Fatalf("uncommitted scope authorization: %v", err)
		}
		invalid := req
		invalid.Action = "health"
		if err := bound.ApplyServiceInstanceState(ctx, invalid, "owner", allowedScope); err == nil || err.Error() != errStateInvalid {
			t.Fatalf("state validation bypassed: %v", err)
		}
		stale := req
		stale.ExpectedLifecycleVersion = 6
		if err := bound.ApplyServiceInstanceState(ctx, stale, "owner", allowedScope); err == nil || err.Error() != errStateStale {
			t.Fatalf("revision validation bypassed: %v", err)
		}
		if err := bound.ApplyServiceInstanceState(ctx, req, "owner", allowedScope); err != nil {
			return err
		}
		if err := bound.ApplyServiceInstanceState(ctx, req, "owner", allowedScope); err != nil {
			return err
		}
		retire := bizcap.ServiceInstanceStateTransition{InstanceID: instance.ID, Action: "retire", ExpectedLifecycleVersion: 9, DesiredState: "retired", ObservedState: "retired", CorrelationID: "bound-retire"}
		if err := bound.ApplyServiceInstanceState(ctx, retire, "owner", allowedScope); err != nil {
			return err
		}
		// Use raw SQL to inspect caller-visible state without the read callback.
		var after ServiceInstance
		if err := tx.Raw("SELECT * FROM biz_service_instance WHERE id = ?", instance.ID).Scan(&after).Error; err != nil {
			return err
		}
		if after.ObservedState != "retired" || after.LifecycleVersion != 10 || after.CurrentVersion != "new" {
			t.Fatalf("bound writes/replay: %+v", after)
		}
		return injected
	}, nil)
	if !errors.Is(err, injected) {
		t.Fatalf("outer transaction: %v", err)
	}
	if readCount == 0 || writeCount != 2 {
		t.Fatalf("owner SQL evidence reads=%d writes=%d", readCount, writeCount)
	}
	if referenceBinds != 1 {
		t.Fatalf("Retire reference admission bound %d times, want 1", referenceBinds)
	}
	if m.db != rootDB {
		t.Fatal("binding replaced long-lived owner DB")
	}
	var after ServiceInstance
	if err := m.db.First(&after, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.ObservedState != instance.ObservedState || after.LifecycleVersion != instance.LifecycleVersion || after.CurrentVersion != instance.CurrentVersion {
		t.Fatal("owner independently committed state/retire/revision")
	}
	if err := bound.ApplyServiceInstanceState(ctx, bizcap.ServiceInstanceStateTransition{InstanceID: instance.ID, Action: "install", ObservedState: "installing"}, "owner", nil); !errors.Is(err, sql.ErrTxDone) {
		t.Fatalf("bound capability after caller rollback: %v", err)
	}
	sqlDB, err := m.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB.Stats().InUse != 0 {
		t.Fatal("bound command left a dangling SQL transaction")
	}
}

func TestStateTransactionRejectsMissingTransactionAndCanceledContext(t *testing.T) {
	m := setupServiceTestDB(t)
	for _, db := range []*gorm.DB{nil, m.db} {
		if err := m.WithServiceInstanceStateTransaction(db).ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{}, "owner", nil); err == nil {
			t.Fatal("nontransaction DB accepted")
		}
	}
	if err := m.db.Transaction(func(tx *gorm.DB) error {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := m.WithServiceInstanceStateTransaction(tx).ApplyServiceInstanceState(ctx, bizcap.ServiceInstanceStateTransition{}, "owner", nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled owner command: %v", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestStateTransactionOptimisticWriteConflictRollsBack(t *testing.T) {
	m := setupServiceTestDB(t)
	app := Application{Code: "cas", Name: "cas", BusinessScopeID: 10, DeptID: 7}
	if err := m.db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	svc := Service{ApplicationID: app.ID, Code: "cas", Name: "cas", RuntimeType: "vm"}
	if err := m.db.Create(&svc).Error; err != nil {
		t.Fatal(err)
	}
	instance := ServiceInstance{ServiceID: svc.ID, Environment: "test", TargetType: "vm", HostID: 55, DesiredState: "stopped", ObservedState: "installing", HealthState: "unknown", LifecycleVersion: 8}
	if err := m.db.Create(&instance).Error; err != nil {
		t.Fatal(err)
	}
	err := m.db.Transaction(func(tx *gorm.DB) error {
		if err := m.db.Callback().Update().Before("gorm:update").Register("owner_revision_conflict", func(db *gorm.DB) {
			if db.Statement.Table == "biz_service_instance" {
				if err := db.Exec("UPDATE biz_service_instance SET lifecycle_version = lifecycle_version + 1 WHERE id = ?", instance.ID).Error; err != nil {
					db.AddError(err)
				}
			}
		}); err != nil {
			return err
		}
		defer m.db.Callback().Update().Remove("owner_revision_conflict")
		return m.WithServiceInstanceStateTransaction(tx).ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{InstanceID: instance.ID, Action: "install", ObservedState: "stopped", ExpectedLifecycleVersion: 8}, "owner", nil)
	})
	if err == nil || err.Error() != errStateStale {
		t.Fatalf("conditional revision write did not refuse: %v", err)
	}
	var after ServiceInstance
	if err := m.db.First(&after, instance.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.LifecycleVersion != instance.LifecycleVersion || after.ObservedState != instance.ObservedState {
		t.Fatal("revision conflict did not roll back")
	}
}
