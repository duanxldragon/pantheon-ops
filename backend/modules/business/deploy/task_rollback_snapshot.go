package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// frozenRollbackPlan supports the inverse already defined for a single package.
// Scripts, dependency graphs and conditional inverses need a separate policy;
// changing their action alone must never replay the original forward commands.
func frozenRollbackPlan(original *DeployTask) (deployExecutionSnapshot, error) {
	snapshot, err := decodeExecutionSnapshot(original.ExecutionSnapshot)
	if err != nil {
		return snapshot, err
	}
	unsupported := errors.New("business.deploy.task.rollbackNotSupported")
	if DetermineRollbackAction(original.Action) != TaskActionUninstall || len(snapshot.Steps) != 1 {
		return snapshot, unsupported
	}
	step := &snapshot.Steps[0]
	if step.StepType != TemplateStepTypePackage || len(step.Dependencies) > 0 || len(step.Condition) > 0 || len(step.StepConfig) > 0 {
		return snapshot, unsupported
	}
	if !validExecutionMode(step.Package.ExecutionMode) || DetermineRollbackAction(step.Action) != TaskActionUninstall {
		return snapshot, unsupported
	}
	if step.Package.ExecutionMode == ExecutionModeFixed && strings.TrimSpace(step.Package.TemplateCode) != "" {
		if err := validateFixedTemplateDefinition(step.Package.TemplateCode); err != nil {
			return snapshot, unsupported
		}
	} else if strings.TrimSpace(step.Package.UninstallCommand) == "" {
		return snapshot, unsupported
	}
	step.Action = TaskActionUninstall
	if _, exists := step.TemplateParams["action"]; exists {
		step.TemplateParams["action"] = TaskActionUninstall
	}
	return snapshot, nil
}

func (s *DeployService) createFrozenRollback(ctx context.Context, taskID uint64, actor string, scope *common.DataScopeReq) (*TaskResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var rollbackID uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		inner := s.targetTransactionService(tx, s.cmdbCapability).targetRequestService(ctx)
		var original DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&original, taskID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDeployTaskNotFound
			}
			return err
		}
		visible, err := inner.taskVisible(&original, scope)
		if err != nil {
			return err
		}
		if !visible {
			return errDeployTaskForbidden
		}
		if original.Status != TaskStatusSuccess && original.Status != TaskStatusFailed {
			return errors.New("business.deploy.task.rollbackInvalidState")
		}
		snapshot, err := frozenRollbackPlan(&original)
		if err != nil {
			return err
		}
		targets := common.NormalizeUint64IDs(parseUint64JSON(original.TargetIDs))
		if original.TargetType != TargetTypeHost || len(targets) == 0 {
			return errors.New("business.deploy.task.rollbackNotSupported")
		}
		scopeName, err := inner.resolveDeployScopeName(original.BusinessScopeID, targets, TaskActionUninstall, scope)
		if err != nil {
			return err
		}
		secrets, complete, err := inner.loadDeployTaskSecretMaterial(original.ID)
		if err != nil {
			return err
		}
		if !complete {
			return errors.New(errDeployTaskSecretParamInvalid)
		}
		redactExecutionSnapshotSecrets(&snapshot, secrets)
		encoded, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		params := decodeJSONMap(original.TemplateParams)
		if _, exists := params["action"]; exists {
			params["action"] = TaskActionUninstall
		}
		encodedParams, err := json.Marshal(params)
		if err != nil {
			return err
		}
		now := time.Now()
		rollback := DeployTask{
			Name:       fmt.Sprintf("%s-rollback", strings.TrimSuffix(original.Name, "-rollback")),
			TemplateID: original.TemplateID, TemplateName: original.TemplateName, TemplateVersion: original.TemplateVersion,
			PackageID: original.PackageID, PackageName: original.PackageName, PackageVersion: original.PackageVersion,
			BusinessScopeID: original.BusinessScopeID, BusinessScopeName: scopeName,
			ServiceID: original.ServiceID, ServiceInstanceID: original.ServiceInstanceID,
			ServiceName: original.ServiceName, ServiceInstanceName: original.ServiceInstanceName,
			Action: TaskActionUninstall, TargetType: TargetTypeHost, TargetIDs: original.TargetIDs,
			ExecutorType: original.ExecutorType, ExecutionMode: original.ExecutionMode,
			TemplateParams: encodedParams, ExecutionSnapshot: encoded,
			Status: TaskStatusDraft, RollbackTaskID: &original.ID, SourceTaskID: &original.ID,
			Remark: fmt.Sprintf("rollback of task %d", taskID), CreatedAt: now, UpdatedAt: now, CreatedBy: actor, UpdatedBy: actor,
		}
		if err := tx.Create(&rollback).Error; err != nil {
			return err
		}
		if err := inner.persistDeployTaskSecrets(rollback.ID, secrets); err != nil {
			return err
		}
		rollbackID = rollback.ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.targetRequestService(ctx).GetTask(rollbackID, scope)
}
