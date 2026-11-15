package filedist

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FileDistributionAuthorizer is a composition port for current actor policy and
// source-file permission. It must refresh authority on every call, including
// scheduled execution. Creation-time scopes and stored usernames are not grants.
type FileDistributionAuthorizer interface {
	AuthorizeFileDistribution(context.Context, uint64, uint64, string) (*common.DataScopeReq, error)
}

func (s *DistributionService) SetTargetSelectionCapability(provider cmdb.TargetSelectionCapability) {
	s.targetSelection = provider
}

func (s *DistributionService) SetFileDistributionAuthorizer(authorizer FileDistributionAuthorizer) {
	s.authorizer = authorizer
}

// CreateAuthorizedTask accepts an actor ID only from the verified request context.
func (s *DistributionService) CreateAuthorizedTask(ctx context.Context, dto DistributionTaskDTO, actorID uint64, createdBy string) (*DistributionTaskResponse, error) {
	var task DistributionTask
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		targets, scope, err := s.resolveAuthorizedTargets(ctx, tx, actorID, dto.FileID, "create", dto.Selection, dto.ScopeVersion, dto.Targets)
		if err != nil {
			return err
		}
		snapshot := append([]TargetConfig(nil), targets...)
		for i := range snapshot {
			snapshot[i].Password, snapshot[i].PrivateKey = "", ""
		}
		targetJSON, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		selectionJSON, err := json.Marshal(dto.Selection)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		task = DistributionTask{Name: dto.Name, FileID: dto.FileID, Targets: datatypes.JSON(targetJSON), Status: statusPending, TotalTargets: len(targets), ScheduledAt: dto.ScheduledAt, CreatedBy: createdBy, OwnerID: actorID, DeptID: scope.DeptID, Selection: datatypes.JSON(selectionJSON), ScopeVersion: dto.ScopeVersion, ResolvedAt: &now}
		if err := tx.Create(&task).Error; err != nil {
			return err
		}
		for _, target := range targets {
			password, err := secrets.Encrypt(target.Password)
			if err != nil {
				return err
			}
			privateKey, err := secrets.Encrypt(target.PrivateKey)
			if err != nil {
				return err
			}
			record := DistributionTarget{TaskID: task.ID, HostID: target.HostID, StateVersion: target.StateVersion, Host: target.Host, Port: target.Port, TargetPath: target.TargetPath, Username: target.Username, Password: password, PrivateKey: privateKey, HostFingerprint: target.HostFingerprint, Status: statusPending, MaxRetries: target.MaxRetries}
			if err := tx.Create(&record).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if dto.ScheduledAt == nil || !dto.ScheduledAt.After(time.Now()) {
		go s.executeTask(task.ID)
	}
	file, _ := s.storageService.GetFileByID(dto.FileID)
	return s.toTaskResponse(&task, file), nil
}

func fileOwnerVisible(file FileRecord, scope *common.DataScopeReq) bool {
	if scope == nil || scope.UserID == 0 || file.OwnerID == 0 {
		return false
	}
	if scope.IsAdmin || strings.TrimSpace(scope.Mode) == common.DataScopeModeAll {
		return true
	}
	switch scope.Mode {
	case common.DataScopeModeSelf:
		return file.OwnerID == scope.UserID
	case common.DataScopeModeDept:
		return file.DeptID != 0 && file.DeptID == scope.DeptID
	case common.DataScopeModeDeptAndChildren, common.DataScopeModeCustom:
		if file.DeptID == 0 {
			return false
		}
		for _, id := range scope.DeptIDs {
			if file.DeptID == id {
				return true
			}
		}
	}
	return false
}

func (s *DistributionService) resolveAuthorizedTargets(ctx context.Context, tx *gorm.DB, actorID, fileID uint64, action string, selection *cmdb.TargetSelection, version string, targets []TargetConfig) ([]TargetConfig, *common.DataScopeReq, error) {
	denied := common.NewForbidden("permission.denied")
	if ctx == nil || ctx.Err() != nil || tx == nil || actorID == 0 || s.authorizer == nil || s.targetSelection == nil || selection == nil || version == "" || len(targets) == 0 {
		return nil, nil, denied
	}
	scope, err := s.authorizer.AuthorizeFileDistribution(ctx, actorID, fileID, action)
	if err != nil || scope == nil || scope.UserID != actorID {
		return nil, nil, denied
	}
	var file FileRecord
	if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&file, fileID).Error; err != nil || !fileOwnerVisible(file, scope) {
		return nil, nil, denied
	}
	provider := s.targetSelection.WithTargetTransaction(tx)
	if provider == nil {
		return nil, nil, denied
	}
	resolved, err := provider.ResolveTargetSelection(ctx, *selection, scope, true)
	if err != nil || resolved == nil || resolved.ScopeVersion != version || resolved.UnavailableCount != 0 || resolved.IneligibleCount != 0 || len(resolved.Hosts) != len(targets) {
		return nil, nil, cmdb.ErrTargetSelectionConflict
	}
	// The existing provider owns SSH ports as well as addresses. Never query
	// another module's tables or let the client override the endpoint.
	endpointReader, ok := provider.(interface {
		ResolveDeployTargets(cmdb.DeployHostResolveRequest) ([]cmdb.DeployHostTarget, error)
	})
	if !ok {
		return nil, nil, denied
	}
	ids := make([]uint64, len(resolved.Hosts))
	for i, host := range resolved.Hosts {
		ids[i] = host.ID
	}
	endpoints, err := endpointReader.ResolveDeployTargets(cmdb.DeployHostResolveRequest{BusinessScopeID: selection.BusinessScopeID, TargetType: "host", TargetIDs: ids, DataScope: scope})
	if err != nil || len(endpoints) != len(ids) {
		return nil, nil, cmdb.ErrTargetSelectionConflict
	}
	byID := make(map[uint64]cmdb.DeployHostTarget, len(endpoints))
	for _, endpoint := range endpoints {
		byID[endpoint.ID] = endpoint
	}
	seen := make(map[uint64]bool, len(targets))
	canonical := append([]TargetConfig(nil), targets...)
	for i, target := range canonical {
		host, ok := byID[target.HostID]
		port := host.SSHPort
		if port == 0 {
			port = 22
		}
		if !ok || target.HostID == 0 || seen[target.HostID] || target.StateVersion != host.StateVersion || (target.Host != "" && target.Host != host.IP) || (target.Port != 0 && target.Port != port) || strings.TrimSpace(host.IP) == "" {
			return nil, nil, cmdb.ErrTargetSelectionConflict
		}
		seen[target.HostID] = true
		canonical[i].Host, canonical[i].Port = host.IP, port
	}
	return canonical, scope, ctx.Err()
}

func (s *DistributionService) revalidateTask(ctx context.Context, tx *gorm.DB, task DistributionTask, targets []DistributionTarget, action string) error {
	var selection cmdb.TargetSelection
	if task.OwnerID == 0 || task.ScopeVersion == "" || len(task.Selection) == 0 || json.Unmarshal(task.Selection, &selection) != nil {
		return common.NewForbidden("permission.denied")
	}
	configs := make([]TargetConfig, len(targets))
	for i, target := range targets {
		configs[i] = TargetConfig{HostID: target.HostID, StateVersion: target.StateVersion, Host: target.Host, Port: target.Port}
	}
	_, _, err := s.resolveAuthorizedTargets(ctx, tx, task.OwnerID, task.FileID, action, &selection, task.ScopeVersion, configs)
	return err
}

func (s *DistributionService) authorizeQueuedTarget(ctx context.Context, taskID uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil || task.Status != statusRunning {
			return common.NewForbidden("permission.denied")
		}
		var targets []DistributionTarget
		if err := tx.Where(whereTaskByID, taskID).Find(&targets).Error; err != nil {
			return err
		}
		return s.revalidateTask(ctx, tx, task, targets, "execute")
	})
}

func (s *DistributionService) validateTaskAction(taskID uint64, action string) error {
	ctx := context.Background()
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, taskID).Error; err != nil {
			return err
		}
		var targets []DistributionTarget
		if err := tx.Where(whereTaskByID, taskID).Find(&targets).Error; err != nil {
			return err
		}
		return s.revalidateTask(ctx, tx, task, targets, action)
	})
}

func safeDistributionFailure(err error) string {
	if errors.Is(err, cmdb.ErrTargetSelectionConflict) {
		return "business.cmdb.selection.conflict"
	}
	return "permission.denied"
}
