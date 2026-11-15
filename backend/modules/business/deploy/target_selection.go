package deploy

import (
	"context"
	"encoding/json"
	"errors"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"reflect"
	"time"
)

func (s *DeployService) ResolveTargetSelection(ctx context.Context, req cmdb.TargetSelection, scope *common.DataScopeReq) (*cmdb.TargetSelectionResult, error) {
	provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
	if !ok {
		return nil, errors.New("business.cmdb.selection.notConfigured")
	}
	return provider.ResolveTargetSelection(ctx, req, scope, false)
}
func (s *DeployService) CreateTask(req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	return s.CreateTaskContext(context.Background(), req, actor, scope)
}
func (s *DeployService) CreateTaskContext(ctx context.Context, req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.targetRequestService(ctx).createSelectedTask(req, actor, scope)
}
func (s *DeployService) createSelectedTask(req CreateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	if req.TargetSelection == nil {
		return s.createTask(req, actor, scope)
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
	if !ok {
		return nil, errors.New("business.cmdb.selection.notConfigured")
	}
	var taskID uint64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		owner := provider.WithTargetTransaction(tx)
		selection, err := owner.ResolveTargetSelection(tx.Statement.Context, *req.TargetSelection, scope, true)
		if err != nil {
			return err
		}
		if req.ScopeVersion == "" || selection.ScopeVersion != req.ScopeVersion {
			return cmdb.ErrTargetSelectionConflict
		}
		if selection.UnavailableCount > 0 || selection.IneligibleCount > 0 || len(selection.Hosts) == 0 {
			return cmdb.ErrTargetSelectionInvalid
		}
		req.BusinessScopeID = selection.Selection.BusinessScopeID
		req.TargetType = TargetTypeHost
		req.TargetIDs = make([]uint64, 0, len(selection.Hosts))
		for _, host := range selection.Hosts {
			req.TargetIDs = append(req.TargetIDs, host.ID)
		}
		capability, ok := owner.(cmdb.DeployCMDBCapability)
		if !ok {
			return errors.New("business.cmdb.selection.notConfigured")
		}
		inner := s.targetTransactionService(tx, capability)
		task, err := inner.createTask(req, actor, scope)
		if err != nil {
			return err
		}
		taskID = task.ID
		snapshot := deployTargetSnapshot{Selection: &selection.Selection, ScopeVersion: selection.ScopeVersion, Sources: selection.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
		bytes, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		return tx.Model(&DeployTask{}).Where(idWhereClause, taskID).UpdateColumn("target_snapshot", datatypes.JSON(bytes)).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetTask(taskID, scope)
}

func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	return s.UpdateTaskContext(context.Background(), id, req, actor, scope)
}
func (s *DeployService) UpdateTaskContext(ctx context.Context, id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.targetRequestService(ctx).updateSelectedTask(id, req, actor, scope)
}
func (s *DeployService) updateSelectedTask(id uint64, req UpdateTaskRequest, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	if req.TargetSelection == nil {
		current, err := s.loadVisibleTask(id, scope)
		if err != nil {
			return nil, err
		}
		var snapshot deployTargetSnapshot
		if len(current.TargetSnapshot) > 0 && json.Unmarshal(current.TargetSnapshot, &snapshot) != nil {
			return nil, cmdb.ErrTargetSelectionConflict
		}
		if snapshot.Selection != nil && (req.TargetIDs != nil || req.TargetType != nil || req.BusinessScopeID != nil) {
			return nil, cmdb.ErrTargetSelectionInvalid
		}
		return s.updateTask(id, req, actor, scope)
	}
	provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
	if !ok {
		return nil, cmdb.ErrTargetSelectionInvalid
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var current DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
			return err
		}
		owner := provider.WithTargetTransaction(tx)
		result, err := owner.ResolveTargetSelection(tx.Statement.Context, *req.TargetSelection, scope, true)
		if err != nil {
			return err
		}
		if req.ScopeVersion == "" || req.ScopeVersion != result.ScopeVersion {
			return cmdb.ErrTargetSelectionConflict
		}
		if result.UnavailableCount > 0 || result.IneligibleCount > 0 || len(result.Hosts) == 0 {
			return cmdb.ErrTargetSelectionInvalid
		}
		req.BusinessScopeID = &result.Selection.BusinessScopeID
		targetType := TargetTypeHost
		req.TargetType = &targetType
		req.TargetIDs = make([]uint64, 0, len(result.Hosts))
		for _, host := range result.Hosts {
			req.TargetIDs = append(req.TargetIDs, host.ID)
		}
		capability, ok := owner.(cmdb.DeployCMDBCapability)
		if !ok {
			return cmdb.ErrTargetSelectionInvalid
		}
		inner := s.targetTransactionService(tx, capability)
		if _, err := inner.updateTask(id, req, actor, scope); err != nil {
			return err
		}
		snapshot := deployTargetSnapshot{Selection: &result.Selection, ScopeVersion: result.ScopeVersion, Sources: result.Sources, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano), TargetType: TargetTypeHost}
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		return tx.Model(&DeployTask{}).Where(idWhereClause, id).UpdateColumn("target_snapshot", datatypes.JSON(raw)).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetTask(id, scope)
}

// Preserve provider ownership and transaction-bound scope reads in both paths.
func (s *DeployService) targetTransactionService(tx *gorm.DB, provider cmdb.DeployCMDBCapability) *DeployService {
	reader := s.bizScopeReader
	if owner, ok := reader.(interface {
		WithBizScopeTransaction(*gorm.DB) bizcap.BizScopeReader
	}); ok {
		reader = owner.WithBizScopeTransaction(tx)
	}
	inner := NewDeployService(tx, provider, reader)
	inner.bizScopeReader = reader
	inner.serviceState = s.serviceState
	inner.serviceStateTransaction = s.serviceStateTransaction
	if _, transaction := tx.Statement.ConnPool.(gorm.TxCommitter); transaction && s.serviceState != nil {
		// Unknown injected owners must never escape the transaction through a
		// fallback to their original DB. Refuse at invocation so unlinked tasks
		// retain their existing nil/no-service compatibility.
		inner.serviceState = unboundServiceInstanceState{}
		if s.serviceStateTransaction != nil {
			if bound := s.serviceStateTransaction(tx); !nilServiceInstanceState(bound) {
				inner.serviceState = bound
			}
		}
	}
	inner.ownership = s.ownership
	inner.leaseTiming = s.leaseTiming
	inner.executionRoot = s.executionRoot
	return inner
}

func nilServiceInstanceState(applier bizcap.ServiceInstanceStateApplier) bool {
	if applier == nil {
		return true
	}
	value := reflect.ValueOf(applier)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

type unboundServiceInstanceState struct{}

func (unboundServiceInstanceState) ApplyServiceInstanceState(context.Context, bizcap.ServiceInstanceStateTransition, string, *common.DataScopeReq) error {
	return errors.New("deploy.serviceState.transactionNotSupported")
}

// Request wrappers do not copy worker maps or replace the long-lived dispatcher.
func (s *DeployService) targetRequestService(ctx context.Context) *DeployService {
	if s.db == nil {
		return s
	}
	db := s.db.WithContext(ctx)
	provider := s.cmdbCapability
	if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
		if bound, ok := owner.WithTargetTransaction(db).(cmdb.DeployCMDBCapability); ok {
			provider = bound
		}
	}
	return s.targetTransactionService(db, provider)
}

// Discovery has no ownership locks. WithTargetTransaction enables locking even
// for a request DB, so use the existing read capability here and retain the
// configured provider for final validation inside the admission transaction.
func (s *DeployService) targetDiscoveryService(ctx context.Context) *DeployService {
	if s.db == nil {
		return s
	}
	db := s.db.WithContext(ctx)
	provider := s.cmdbCapability
	if provider != nil {
		provider = cmdb.NewDeployCMDBCapability(db, s.bizScopeReader)
	}
	inner := NewDeployService(db, provider, s.bizScopeReader)
	// Preserve explicit missing-owner failure, rather than constructor fallback.
	inner.bizScopeReader = s.bizScopeReader
	inner.hostReader = s.hostReader
	inner.serviceState = s.serviceState
	inner.serviceStateTransaction = s.serviceStateTransaction
	return inner
}
func (s *DeployService) requestContext() context.Context {
	if s.db != nil && s.db.Statement != nil && s.db.Statement.Context != nil {
		return s.db.Statement.Context
	}
	return context.Background()
}
