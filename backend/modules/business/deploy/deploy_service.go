package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	bizscope "pantheon-base/modules/business/bizscope"
	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/modules/business/cmdb"
	executorprovider "pantheon-base/modules/business/deploy/executor"
	deployorchestrator "pantheon-base/modules/business/deploy/orchestrator"
	"pantheon-base/pkg/common"

	"golang.org/x/crypto/ssh"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeployService struct {
	db               *gorm.DB
	cmdbCapability   cmdb.DeployCMDBCapability
	bizScopeReader   bizcap.BizScopeReader
	hostReader       bizcap.CMDBHostReader
	serviceState     bizcap.ServiceInstanceStateCommand
	sshRunnerFactory func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error)
	executor         DeployExecutor
	asyncExecution   bool
	taskCancels      sync.Map
	taskWorkers      sync.Map
}

const deployAttemptMax = 3

const (
	defaultDeployExecutionTimeout = 30 * time.Minute
	maxDeployExecutionTimeout     = 24 * time.Hour
)

// DeployExecutor is the provider boundary for remote task execution. Providers
// receive a frozen task/host request and must never persist secret material.
//
//nolint:revive // Public API name retained for compatibility.
type DeployExecutor interface {
	ExecuteHost(context.Context, DeployExecutionHostRequest) error
}

// DeployExecutionHostRequest is the persisted execution context supplied to a
// provider. Credential contains decrypted material only in process memory.
//
//nolint:revive // Public API name retained for compatibility.
type DeployExecutionHostRequest struct {
	Task       DeployTask
	TaskHost   TaskHostResponse
	Credential StartTaskRequest
	Actor      string
}

type sshDeployExecutor struct{ service *DeployService }

func (e *sshDeployExecutor) ExecuteHost(ctx context.Context, req DeployExecutionHostRequest) error {
	target := cmdbHostSnapshot{ID: req.TaskHost.HostID, Hostname: req.TaskHost.Hostname, IP: req.TaskHost.HostIP, SSHPort: req.TaskHost.SSHPort, OS: req.TaskHost.OS, BusinessScopeID: req.TaskHost.BusinessScopeID}
	return e.service.executeOrchestratedSSHHost(ctx, req.Task, target, req.TaskHost, req.Credential, req.Actor)
}

type deployExecutionStep struct {
	Index                 int
	StepCode              string
	StepName              string
	StepType              string
	Action                string
	ContinueOnHostFailure bool
	Package               DeployPackage
	TemplateParams        map[string]any
	StepConfig            map[string]any
	Dependencies          []string
	Condition             map[string]any
	FailurePolicy         string
}

type deployExecutionSummary struct {
	InstalledComponents   []cmdb.InstalledComponentUpsert
	RemovedComponentNames []string
}

var (
	errDeployTaskNotFound  = errors.New("business.deploy.task.notFound")
	errDeployTaskForbidden = errors.New("business.deploy.task.forbidden")
)

const (
	errDeployTaskTemplateParamsInvalid    = "business.deploy.task.templateParamsInvalid"
	errDeployTaskTemplateInvalid          = "business.deploy.task.templateInvalid"
	errDeployTaskTemplateNotFound         = "business.deploy.task.templateNotFound"
	errDeployTaskTemplateDisabled         = "business.deploy.task.templateDisabled"
	errDeployTaskInstallCommandRequired   = "business.deploy.task.installCommandRequired"
	errDeployTaskUninstallCommandRequired = "business.deploy.task.uninstallCommandRequired"
	errDeployTaskPackageNotFound          = "business.deploy.task.packageNotFound"
	errDeployTaskPackageSourceMissing     = "business.deploy.task.packageSourceMissing"
	errDeployTaskInvalidUpdateState       = "business.deploy.task.invalidUpdateState"
	errDeployTaskInvalidDeleteState       = "business.deploy.task.invalidDeleteState"
	errDeployTaskSSHHostKeyRequired       = "business.deploy.task.sshHostKeyRequired"
	errDeployTaskSSHHostKeyMismatch       = "business.deploy.task.sshHostKeyMismatch"
	errDeployTaskSSHUserRequired          = "business.deploy.task.sshUserRequired"
	errDeployTaskSSHPasswordRequired      = "business.deploy.task.sshPasswordRequired"
	errDeployTaskSSHPrivateKeyRequired    = "business.deploy.task.sshPrivateKeyRequired"
	errDeployTaskSSHAuthFailed            = "business.deploy.task.sshAuthFailed"
	errDeployTaskSSHConnectFailed         = "business.deploy.task.sshConnectFailed"
	errDeployTaskAlreadyRunning           = "business.deploy.task.alreadyRunning"
	errDeployTaskLeaseConflict            = "business.deploy.task.leaseConflict"
	errDeployTaskSnapshotMissing          = "business.deploy.task.snapshotMissing"
	errDeployTaskHostStaleReport          = "business.deploy.taskHost.staleReport"
	errDeployPackageImmutable             = "business.deploy.package.immutable"
	errDeployTemplateImmutable            = "business.deploy.template.immutable"
	errDeployTaskInvalidPauseState        = "business.deploy.task.invalidPauseState"
	errDeployTaskInvalidResumeState       = "business.deploy.task.invalidResumeState"
	errDeployTaskInvalidRetryState        = "business.deploy.task.invalidRetryState"
	errDeployTaskInvalidCancelState       = "business.deploy.task.invalidCancelState"
)

const (
	defaultHostLeaseDuration = 30 * time.Minute
	observedStateUnknown     = "unknown"
	observedStateStopped     = "stopped"
)

const (
	idDescOrder       = "id DESC"
	taskIDWhereClause = "task_id = ?"
	idWhereClause     = "id = ?"
	statusWhereClause = "status = ?"
)

// NewDeployService creates the deployment service with owner-module capabilities.
func NewDeployService(db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability, readers ...bizcap.BizScopeReader) *DeployService {
	var bizScopeReader bizcap.BizScopeReader
	if len(readers) > 0 {
		bizScopeReader = readers[0]
	}
	if bizScopeReader == nil {
		bizScopeReader = bizscope.NewService(db)
	}
	hostReader, _ := cmdbCapability.(bizcap.CMDBHostReader)
	service := &DeployService{
		db:               db,
		cmdbCapability:   cmdbCapability,
		bizScopeReader:   bizScopeReader,
		hostReader:       hostReader,
		sshRunnerFactory: newDeploySSHRunner,
	}
	service.executor = &sshDeployExecutor{service: service}
	return service
}

// SetAsyncExecution enables the same-process durable worker used by production
// module wiring. Tests can leave it disabled to exercise execution inline.
func (s *DeployService) SetAsyncExecution(enabled bool) { s.asyncExecution = enabled }

// SetExecutor replaces the remote execution provider. A nil provider restores
// the SSH provider used by this module.
func (s *DeployService) SetExecutor(executor DeployExecutor) {
	if executor == nil {
		executor = &sshDeployExecutor{service: s}
	}
	s.executor = executor
}

// SetServiceInstanceStateCommand configures service-instance state writeback.
func (s *DeployService) SetServiceInstanceStateCommand(command bizcap.ServiceInstanceStateCommand) {
	s.serviceState = command
}

func (s *DeployService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&DeployPackage{}, &DeployTemplate{}, &DeployTemplateStep{}, &DeployTask{}, &DeployTaskHost{}, &DeployHostLease{}, &DeployTaskAttempt{}, &DeployCredentialRef{}, &DeployTaskSecret{}, &DeployTaskStep{}, &DeployTaskEvent{}, &DeployTaskLogChunk{})
}

func (s *DeployService) CreatePackage(req CreatePackageRequest, actor string) (*PackageResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Version = strings.TrimSpace(req.Version)
	if req.Name == "" || req.Version == "" {
		return nil, errors.New("deploypackage.invalid")
	}
	if req.Status == "" {
		req.Status = PackageStatusEnabled
	}
	if strings.TrimSpace(req.ExecutionMode) == "" {
		req.ExecutionMode = ExecutionModeFixed
	}
	if !validPackageStatus(req.Status) {
		return nil, errors.New("deploypackage.status_invalid")
	}
	if !validExecutionMode(req.ExecutionMode) {
		return nil, errors.New("deploypackage.execution_mode_invalid")
	}
	if err := validateTemplateDefinition(req.ExecutionMode, req.TemplateCode, req.TemplateConfig); err != nil {
		return nil, err
	}
	if err := validateDeployPackageSource(req.SourceFileName, req.SourceURL); err != nil {
		return nil, err
	}
	if s.packageExists(req.Name, req.Version, 0) {
		return nil, errors.New("deploypackage.exists")
	}
	templateConfigJSON, _ := json.Marshal(req.TemplateConfig)
	item := DeployPackage{
		Name:             req.Name,
		Version:          req.Version,
		Description:      req.Description,
		InstallCommand:   req.InstallCommand,
		UninstallCommand: req.UninstallCommand,
		ExecutionMode:    req.ExecutionMode,
		TemplateCode:     strings.TrimSpace(req.TemplateCode),
		TemplateConfig:   datatypes.JSON(templateConfigJSON),
		SourceObjectKey:  strings.TrimSpace(req.SourceObjectKey),
		SourceFileName:   strings.TrimSpace(req.SourceFileName),
		SourceURL:        strings.TrimSpace(req.SourceURL),
		Status:           req.Status,
		CreatedBy:        actor,
		UpdatedBy:        actor,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	resp := packageToResponse(&item, packageDeploymentStat{})
	return &resp, nil
}

func (s *DeployService) ListPackages(query PackageQuery) (*PackageListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := s.db.Model(&DeployPackage{})
	if strings.TrimSpace(query.Keyword) != "" {
		like := "%" + strings.TrimSpace(query.Keyword) + "%"
		db = db.Where("name LIKE ? OR version LIKE ?", like, like)
	}
	if strings.TrimSpace(query.Status) != "" {
		db = db.Where(statusWhereClause, strings.TrimSpace(query.Status))
	}
	if strings.TrimSpace(query.ExecutionMode) != "" {
		db = db.Where("execution_mode = ?", strings.TrimSpace(query.ExecutionMode))
	}
	if strings.TrimSpace(query.TemplateCode) != "" {
		db = db.Where("template_code = ?", strings.TrimSpace(query.TemplateCode))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []DeployPackage
	if err := db.Order(idDescOrder).Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	packageIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		packageIDs = append(packageIDs, row.ID)
	}
	statsByPackage, err := s.loadPackageDeploymentStats(packageIDs)
	if err != nil {
		return nil, err
	}
	items := make([]PackageResponse, 0, len(rows))
	for i := range rows {
		items = append(items, packageToResponse(&rows[i], statsByPackage[rows[i].ID]))
	}
	return &PackageListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *DeployService) GetPackage(id uint64) (*PackageResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var item DeployPackage
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploypackage.not_found")
		}
		return nil, err
	}
	statsByPackage, err := s.loadPackageDeploymentStats([]uint64{item.ID})
	if err != nil {
		return nil, err
	}
	resp := packageToResponse(&item, statsByPackage[item.ID])
	return &resp, nil
}

func (s *DeployService) UpdatePackage(id uint64, req UpdatePackageRequest, actor string) (*PackageResponse, error) {
	item, err := s.loadPackage(id)
	if err != nil {
		return nil, err
	}
	referenced, err := s.isPackageReferenced(id)
	if err != nil {
		return nil, err
	}
	if referenced && updatePackageChangesExecutionDefinition(req) {
		return nil, errors.New(errDeployPackageImmutable)
	}
	updates := map[string]interface{}{"updated_by": actor, "updated_at": time.Now()}
	name, version := applyPackageScalarUpdates(updates, req, item.Name, item.Version)
	if name == "" || version == "" {
		return nil, errors.New("deploypackage.invalid")
	}
	if s.packageExists(name, version, id) {
		return nil, errors.New("deploypackage.exists")
	}
	nextExecutionMode, nextTemplateCode, nextTemplateConfig, err := applyPackageModeUpdates(updates, req, item)
	if err != nil {
		return nil, err
	}
	if err := validateTemplateDefinition(nextExecutionMode, nextTemplateCode, nextTemplateConfig); err != nil {
		return nil, err
	}
	if err := applyPackageStatusUpdate(updates, req); err != nil {
		return nil, err
	}
	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	statsByPackage, err := s.loadPackageDeploymentStats([]uint64{item.ID})
	if err != nil {
		return nil, err
	}
	resp := packageToResponse(&item, statsByPackage[item.ID])
	return &resp, nil
}

func (s *DeployService) loadPackage(id uint64) (DeployPackage, error) {
	var item DeployPackage
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployPackage{}, errors.New("deploypackage.not_found")
		}
		return DeployPackage{}, err
	}
	return item, nil
}

func applyPackageScalarUpdates(updates map[string]interface{}, req UpdatePackageRequest, name, version string) (string, string) {
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		updates["name"] = name
	}
	if req.Version != nil {
		version = strings.TrimSpace(*req.Version)
		updates["version"] = version
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.InstallCommand != nil {
		updates["install_command"] = *req.InstallCommand
	}
	if req.UninstallCommand != nil {
		updates["uninstall_command"] = *req.UninstallCommand
	}
	return name, version
}

func applyPackageModeUpdates(updates map[string]interface{}, req UpdatePackageRequest, item DeployPackage) (string, string, map[string]any, error) {
	if req.ExecutionMode != nil {
		if !validExecutionMode(*req.ExecutionMode) {
			return "", "", nil, errors.New("deploypackage.execution_mode_invalid")
		}
		updates["execution_mode"] = *req.ExecutionMode
	}
	nextExecutionMode := item.ExecutionMode
	if req.ExecutionMode != nil {
		nextExecutionMode = strings.TrimSpace(*req.ExecutionMode)
	}
	nextTemplateCode := item.TemplateCode
	if req.TemplateCode != nil {
		nextTemplateCode = strings.TrimSpace(*req.TemplateCode)
		updates["template_code"] = nextTemplateCode
	}
	nextTemplateConfig := decodeJSONMap(item.TemplateConfig)
	if req.TemplateConfig != nil {
		nextTemplateConfig = *req.TemplateConfig
		templateConfigJSON, _ := json.Marshal(*req.TemplateConfig)
		updates["template_config"] = datatypes.JSON(templateConfigJSON)
	}
	if req.SourceObjectKey != nil {
		updates["source_object_key"] = strings.TrimSpace(*req.SourceObjectKey)
	}
	if req.SourceFileName != nil {
		updates["source_file_name"] = strings.TrimSpace(*req.SourceFileName)
	}
	if req.SourceURL != nil {
		updates["source_url"] = strings.TrimSpace(*req.SourceURL)
	}
	nextSourceFileName := item.SourceFileName
	if req.SourceFileName != nil {
		nextSourceFileName = strings.TrimSpace(*req.SourceFileName)
	}
	nextSourceURL := item.SourceURL
	if req.SourceURL != nil {
		nextSourceURL = strings.TrimSpace(*req.SourceURL)
	}
	if err := validateDeployPackageSource(nextSourceFileName, nextSourceURL); err != nil {
		return "", "", nil, err
	}
	return nextExecutionMode, nextTemplateCode, nextTemplateConfig, nil
}

func applyPackageStatusUpdate(updates map[string]interface{}, req UpdatePackageRequest) error {
	if req.Status != nil {
		if !validPackageStatus(*req.Status) {
			return errors.New("deploypackage.status_invalid")
		}
		updates["status"] = *req.Status
	}
	return nil
}

func (s *DeployService) DeletePackage(id uint64) error {
	var count int64
	if err := s.db.Model(&DeployTask{}).Where("package_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("deploypackage.in_use")
	}
	result := s.db.Delete(&DeployPackage{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("deploypackage.not_found")
	}
	return nil
}

func (s *DeployService) CreateTask(req CreateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	targetIDs, err := validateCreateTaskRequest(&req)
	if err != nil {
		return nil, err
	}
	// FZ-02: split write-only secret params out of the persisted template
	// params. Secret values live in biz_deploy_task_secret only.
	req.TemplateParams, req.SecretParams = splitDeploySecretParams(req.TemplateParams)
	mergeDeploySecretParamsForValidation(req.TemplateParams, req.SecretParams)
	template, err := s.applyTemplateToCreateRequest(&req)
	if err != nil {
		return nil, err
	}
	action := normalizeTaskAction(req.Action)
	if !validTaskAction(action) {
		return nil, errors.New("business.deploy.task.invalidAction")
	}
	if err := validateDeployStructuralParams(req.TemplateParams); err != nil {
		return nil, err
	}
	if err := validateDeploySecretParams(req.SecretParams); err != nil {
		return nil, err
	}
	pkg, err := s.loadTaskPackage(req.PackageID, template, req.TemplateParams)
	if err != nil {
		return nil, err
	}
	scopeName, err := s.resolveCreateTaskScopeName(&req, targetIDs, action, dataScope)
	if err != nil {
		return nil, err
	}
	task, err := s.buildCreateTask(&req, pkg, template, targetIDs, action, scopeName, actor)
	if err != nil {
		return nil, err
	}
	if err := s.persistDeployTaskSecrets(task.ID, req.SecretParams); err != nil {
		return nil, err
	}
	return s.GetTask(task.ID, dataScope)
}

func validateCreateTaskRequest(req *CreateTaskRequest) ([]uint64, error) {
	req.Name = strings.TrimSpace(req.Name)
	targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
	if req.Name == "" {
		return nil, errors.New("business.deploy.task.nameRequired")
	}
	if len(targetIDs) == 0 {
		return nil, errors.New("business.deploy.task.targetRequired")
	}
	if !validTargetType(req.TargetType) {
		return nil, errors.New("business.deploy.task.invalidTargetType")
	}
	if req.ExecutorType == "" {
		req.ExecutorType = ExecutorTypeManual
	}
	if !validExecutorType(req.ExecutorType) {
		return nil, errors.New("business.deploy.task.invalidExecutorType")
	}
	return targetIDs, nil
}

func (s *DeployService) applyTemplateToCreateRequest(req *CreateTaskRequest) (*TemplateResponse, error) {
	if req.TemplateID == 0 {
		return nil, nil
	}
	templateDetail, err := s.GetTemplate(req.TemplateID)
	if err != nil {
		return nil, mapDeployTaskTemplateLookupError(err)
	}
	if templateDetail.Status != TemplateStatusEnabled {
		return nil, errors.New(errDeployTaskTemplateDisabled)
	}
	if req.PackageID == 0 {
		req.PackageID = templateDetail.PackageID
		if req.PackageID == 0 && len(templateDetail.Steps) > 0 {
			req.PackageID = templateDetail.Steps[0].PackageID
		}
	}
	if strings.TrimSpace(req.Action) == "" {
		req.Action = templateDetail.DefaultAction
	}
	if len(req.TemplateParams) == 0 && len(templateDetail.ParameterSchema) > 0 {
		req.TemplateParams = templateDetail.ParameterSchema
	}
	return templateDetail, nil
}

func (s *DeployService) resolveCreateTaskScopeName(req *CreateTaskRequest, targetIDs []uint64, action string, dataScope *common.DataScopeReq) (string, error) {
	if req.TargetType != TargetTypeHost {
		return "", nil
	}
	return s.resolveDeployScopeName(req.BusinessScopeID, targetIDs, action, dataScope)
}

func (s *DeployService) resolveDeployScopeName(businessScopeID uint64, targetIDs []uint64, action string, dataScope *common.DataScopeReq) (string, error) {
	if businessScopeID == 0 {
		return "", errors.New("business.deploy.task.scopeRequired")
	}
	if s.bizScopeReader == nil {
		return "", errors.New("business.bizscope.readerNotConfigured")
	}
	scope, err := s.bizScopeReader.GetActive(context.Background(), businessScopeID, dataScope)
	if err != nil {
		return "", errors.New("business.deploy.task.scopeInvalid")
	}
	hosts, err := s.cmdbCapability.ResolveDeployTargets(cmdb.DeployHostResolveRequest{
		BusinessScopeID: businessScopeID,
		TargetType:      TargetTypeHost,
		TargetIDs:       targetIDs,
		DataScope:       dataScope,
	})
	if err != nil {
		return "", err
	}
	if len(hosts) != len(targetIDs) {
		return "", errors.New("business.deploy.task.targetOutOfScope")
	}
	scopeName := scope.Name
	for _, host := range hosts {
		if !hostStatusAllowedForAction(host.Status, action) {
			return "", errors.New("business.deploy.task.targetStatusMismatch")
		}
		scopeName = host.BusinessScopeName
	}
	return scopeName, nil
}

func (s *DeployService) buildCreateTask(req *CreateTaskRequest, pkg DeployPackage, template *TemplateResponse, targetIDs []uint64, action, scopeName, actor string) (*DeployTask, error) {
	targetJSON, _ := json.Marshal(targetIDs)
	// The in-memory view still carries decrypted secrets so required-parameter
	// validation inside the snapshot builder can see them. The snapshot is
	// redacted to sentinels below and the persisted params are stripped.
	executionSnapshot, err := s.buildDeployExecutionSnapshot(pkg, template, action, req.TemplateParams)
	if err != nil {
		return nil, err
	}
	redactExecutionSnapshotSecrets(&executionSnapshot, req.SecretParams)
	persistParams := make(map[string]any, len(req.TemplateParams))
	for key, value := range req.TemplateParams {
		if isDeployTaskSecretParam(key) {
			continue
		}
		persistParams[key] = value
	}
	templateParamsJSON, _ := json.Marshal(persistParams)
	executionSnapshotJSON, _ := json.Marshal(executionSnapshot)
	task := DeployTask{
		Name:              req.Name,
		TemplateID:        req.TemplateID,
		PackageID:         pkg.ID,
		TemplateName:      "",
		TemplateVersion:   "",
		PackageName:       pkg.Name,
		PackageVersion:    pkg.Version,
		BusinessScopeID:   req.BusinessScopeID,
		BusinessScopeName: scopeName,
		ServiceID:         req.ServiceID,
		ServiceInstanceID: req.ServiceInstanceID,
		Action:            action,
		TargetType:        req.TargetType,
		TargetIDs:         datatypes.JSON(targetJSON),
		ExecutorType:      req.ExecutorType,
		ExecutionMode:     pkg.ExecutionMode,
		TemplateParams:    datatypes.JSON(templateParamsJSON),
		ExecutionSnapshot: datatypes.JSON(executionSnapshotJSON),
		Status:            TaskStatusDraft,
		Remark:            req.Remark,
		CreatedBy:         actor,
		UpdatedBy:         actor,
	}
	if template != nil {
		task.TemplateName = template.Name
		task.TemplateVersion = template.Version
		task.ExecutionMode = template.ExecutionMode
	}
	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *DeployService) ListTasks(query TaskQuery, dataScope *common.DataScopeReq) (*TaskListResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := s.db.Model(&DeployTask{})
	if strings.TrimSpace(query.Keyword) != "" {
		like := "%" + strings.TrimSpace(query.Keyword) + "%"
		db = db.Where("name LIKE ? OR package_name LIKE ?", like, like)
	}
	if strings.TrimSpace(query.Status) != "" {
		db = db.Where(statusWhereClause, strings.TrimSpace(query.Status))
	}
	if strings.TrimSpace(query.ExecutorType) != "" {
		db = db.Where("executor_type = ?", strings.TrimSpace(query.ExecutorType))
	}
	var total int64
	var rows []DeployTask
	if err := db.Order(idDescOrder).Find(&rows).Error; err != nil {
		return nil, err
	}
	if dataScope != nil && !dataScope.IsAdmin && strings.TrimSpace(dataScope.Mode) != "" && strings.TrimSpace(dataScope.Mode) != common.DataScopeModeAll {
		filteredRows, err := s.filterVisibleTasks(rows, dataScope)
		if err != nil {
			return nil, err
		}
		rows = filteredRows
	}
	total = int64(len(rows))
	start := (query.Page - 1) * query.PageSize
	if start >= len(rows) {
		return &TaskListResponse{Items: []TaskResponse{}, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
	}
	end := start + query.PageSize
	if end > len(rows) {
		end = len(rows)
	}
	rows = rows[start:end]
	hostsByTaskID, err := s.loadTaskHostsByTaskIDs(extractTaskIDs(rows))
	if err != nil {
		return nil, err
	}
	items := make([]TaskResponse, 0, len(rows))
	for i := range rows {
		items = append(items, taskToResponse(&rows[i], hostsByTaskID[rows[i].ID]))
	}
	return &TaskListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *DeployService) GetTask(id uint64, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	var hosts []DeployTaskHost
	if err := s.db.Where(taskIDWhereClause, id).Order("id ASC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	hostResp := make([]TaskHostResponse, 0, len(hosts))
	for i := range hosts {
		hostResp = append(hostResp, taskHostToResponse(&hosts[i]))
	}
	resp := taskToResponse(task, hostResp)
	resp.SecretPresence = s.deployTaskSecretPresence(task.ID)
	return &resp, nil
}

// getTaskReused returns the task marked as a reused idempotent Start replay.
func (s *DeployService) getTaskReused(id uint64, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	resp, err := s.GetTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	resp.StartRequestReused = true
	return resp, nil
}

type taskUpdateValues struct {
	name              string
	templateID        uint64
	packageID         uint64
	businessScopeID   uint64
	serviceID         uint64
	serviceInstanceID uint64
	action            string
	targetType        string
	targetIDs         []uint64
	executorType      string
	templateParams    map[string]any
	remark            string
}

func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusPending && task.Status != TaskStatusDraft {
		return nil, errors.New(errDeployTaskInvalidUpdateState)
	}
	// FZ-02: task updates may carry write-only secret params while the task is
	// still draft/pending; a started task is rejected above.
	secretParams := map[string]string{}
	if req.SecretParams != nil {
		if err := validateDeploySecretParams(*req.SecretParams); err != nil {
			return nil, err
		}
		secretParams = *req.SecretParams
	}
	if req.TemplateParams != nil {
		stripped, _ := splitDeploySecretParams(*req.TemplateParams)
		req.TemplateParams = &stripped
	}
	values, err := s.buildTaskUpdateValues(task, req)
	if err != nil {
		return nil, err
	}
	template, err := s.applyTemplateToTaskUpdate(&values, req)
	if err != nil {
		return nil, err
	}
	pkg, err := s.loadTaskPackage(values.packageID, template, values.templateParams)
	if err != nil {
		return nil, err
	}
	values.targetIDs = common.NormalizeUint64IDs(values.targetIDs)
	if len(values.targetIDs) == 0 {
		return nil, errors.New("business.deploy.task.targetRequired")
	}
	scopeName, err := s.resolveTaskUpdateScopeName(&values, dataScope)
	if err != nil {
		return nil, err
	}
	if err := s.persistTaskUpdate(task, &values, pkg, template, scopeName, actor); err != nil {
		return nil, err
	}
	if err := s.persistDeployTaskSecrets(task.ID, secretParams); err != nil {
		return nil, err
	}
	return s.GetTask(id, dataScope)
}

func (s *DeployService) buildTaskUpdateValues(task *DeployTask, req UpdateTaskRequest) (taskUpdateValues, error) {
	values := taskUpdateValues{
		name:              task.Name,
		templateID:        task.TemplateID,
		packageID:         task.PackageID,
		businessScopeID:   task.BusinessScopeID,
		serviceID:         task.ServiceID,
		serviceInstanceID: task.ServiceInstanceID,
		action:            normalizeTaskAction(task.Action),
		targetType:        task.TargetType,
		targetIDs:         parseUint64JSON(task.TargetIDs),
		executorType:      task.ExecutorType,
		templateParams:    decodeJSONMap(task.TemplateParams),
		remark:            task.Remark,
	}
	applyTaskSimpleOverrides(&values, req)
	if err := applyTaskNameOverride(&values, req); err != nil {
		return values, err
	}
	if err := applyTaskActionOverride(&values, req); err != nil {
		return values, err
	}
	if err := applyTaskTargetOverride(&values, req); err != nil {
		return values, err
	}
	return values, nil
}

func applyTaskSimpleOverrides(values *taskUpdateValues, req UpdateTaskRequest) {
	if req.TemplateID != nil {
		values.templateID = *req.TemplateID
	}
	if req.PackageID != nil {
		values.packageID = *req.PackageID
	}
	if req.BusinessScopeID != nil {
		values.businessScopeID = *req.BusinessScopeID
	}
	if req.ServiceID != nil {
		values.serviceID = *req.ServiceID
	}
	if req.ServiceInstanceID != nil {
		values.serviceInstanceID = *req.ServiceInstanceID
	}
	if req.TemplateParams != nil {
		values.templateParams = *req.TemplateParams
	}
	if req.Remark != nil {
		values.remark = *req.Remark
	}
}

func applyTaskNameOverride(values *taskUpdateValues, req UpdateTaskRequest) error {
	if req.Name != nil {
		values.name = strings.TrimSpace(*req.Name)
		if values.name == "" {
			return errors.New("business.deploy.task.nameRequired")
		}
	}
	return nil
}

func applyTaskActionOverride(values *taskUpdateValues, req UpdateTaskRequest) error {
	if req.Action != nil {
		values.action = normalizeTaskAction(*req.Action)
		if !validTaskAction(values.action) {
			return errors.New("business.deploy.task.invalidAction")
		}
	}
	return nil
}

func applyTaskTargetOverride(values *taskUpdateValues, req UpdateTaskRequest) error {
	if req.TargetType != nil {
		if !validTargetType(*req.TargetType) {
			return errors.New("business.deploy.task.invalidTargetType")
		}
		values.targetType = *req.TargetType
	}
	if req.TargetIDs != nil {
		values.targetIDs = common.NormalizeUint64IDs(req.TargetIDs)
		if len(values.targetIDs) == 0 {
			return errors.New("business.deploy.task.targetRequired")
		}
	}
	if req.ExecutorType != nil {
		if !validExecutorType(*req.ExecutorType) {
			return errors.New("business.deploy.task.invalidExecutorType")
		}
		values.executorType = *req.ExecutorType
	}
	return nil
}

func (s *DeployService) applyTemplateToTaskUpdate(values *taskUpdateValues, req UpdateTaskRequest) (*TemplateResponse, error) {
	if values.templateID == 0 {
		return nil, nil
	}
	templateDetail, err := s.GetTemplate(values.templateID)
	if err != nil {
		return nil, mapDeployTaskTemplateLookupError(err)
	}
	if templateDetail.Status != TemplateStatusEnabled {
		return nil, errors.New(errDeployTaskTemplateDisabled)
	}
	if req.PackageID == nil {
		values.packageID = templateDetail.PackageID
		if values.packageID == 0 && len(templateDetail.Steps) > 0 {
			values.packageID = templateDetail.Steps[0].PackageID
		}
	}
	if req.Action == nil {
		values.action = normalizeTaskAction(templateDetail.DefaultAction)
	}
	if req.TemplateParams == nil && len(templateDetail.ParameterSchema) > 0 {
		values.templateParams = templateDetail.ParameterSchema
	}
	return templateDetail, nil
}

func (s *DeployService) loadTaskPackage(packageID uint64, template *TemplateResponse, templateParams map[string]any) (DeployPackage, error) {
	if packageID == 0 {
		if template != nil && len(template.Steps) > 0 && templateHasOnlyScriptSteps(template.Steps) {
			return DeployPackage{}, nil
		}
		return DeployPackage{}, errors.New("business.deploy.task.packageRequired")
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, packageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployPackage{}, errors.New(errDeployTaskPackageNotFound)
		}
		return DeployPackage{}, err
	}
	if pkg.Status != PackageStatusEnabled {
		return DeployPackage{}, errors.New("business.deploy.task.packageDisabled")
	}
	if template == nil {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, templateParams); err != nil {
			return DeployPackage{}, err
		}
	}
	return pkg, nil
}

func templateHasOnlyScriptSteps(steps []TemplateStepResponse) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if strings.TrimSpace(step.StepType) != TemplateStepTypeScript {
			return false
		}
	}
	return true
}

func (s *DeployService) resolveTaskUpdateScopeName(values *taskUpdateValues, dataScope *common.DataScopeReq) (string, error) {
	if values.targetType != TargetTypeHost {
		values.businessScopeID = 0
		return "", nil
	}
	return s.resolveDeployScopeName(values.businessScopeID, values.targetIDs, values.action, dataScope)
}

func (s *DeployService) persistTaskUpdate(task *DeployTask, values *taskUpdateValues, pkg DeployPackage, template *TemplateResponse, scopeName, actor string) error {
	targetJSON, _ := json.Marshal(values.targetIDs)
	templateParamsJSON, _ := json.Marshal(values.templateParams)
	executionSnapshot, err := s.buildDeployExecutionSnapshot(pkg, template, values.action, values.templateParams)
	if err != nil {
		return err
	}
	executionSnapshotJSON, _ := json.Marshal(executionSnapshot)
	updates := map[string]interface{}{
		"name":                values.name,
		"template_id":         uint64(0),
		"template_name":       "",
		"template_version":    "",
		"package_id":          pkg.ID,
		"package_name":        pkg.Name,
		"package_version":     pkg.Version,
		"business_scope_id":   values.businessScopeID,
		"business_scope_name": scopeName,
		"service_id":          values.serviceID,
		"service_instance_id": values.serviceInstanceID,
		"action":              values.action,
		"target_type":         values.targetType,
		"target_ids":          datatypes.JSON(targetJSON),
		"executor_type":       values.executorType,
		"execution_mode":      pkg.ExecutionMode,
		"template_params":     datatypes.JSON(templateParamsJSON),
		"execution_snapshot":  datatypes.JSON(executionSnapshotJSON),
		"remark":              values.remark,
		"updated_by":          actor,
		"updated_at":          time.Now(),
	}
	if template != nil {
		updates["template_id"] = template.ID
		updates["template_name"] = template.Name
		updates["template_version"] = template.Version
		updates["execution_mode"] = template.ExecutionMode
	}
	return s.db.Model(task).Updates(updates).Error
}

func (s *DeployService) DeleteTask(id uint64, actor string, dataScope *common.DataScopeReq) error {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return err
	}
	// A process crash can leave a task in running with only expired leases.
	// Reconcile that orphaned execution before applying the normal delete
	// guard; otherwise the task can never be cleaned up from the Web UI.
	if task.Status == TaskStatusRunning && task.UpdatedAt.Before(time.Now().Add(-5*time.Minute)) {
		now := time.Now()
		var activeAttempts int64
		if err := s.db.Model(&DeployTaskAttempt{}).
			Where("task_id = ? AND status = ? AND (lease_expires_at IS NULL OR lease_expires_at > ?)", task.ID, "running", now).
			Count(&activeAttempts).Error; err != nil {
			return err
		}
		var activeLeases int64
		if err := s.db.Model(&DeployHostLease{}).
			Where("task_id = ? AND expires_at > ?", task.ID, now).
			Count(&activeLeases).Error; err != nil {
			return err
		}
		if activeAttempts == 0 && activeLeases == 0 {
			if err := s.db.Model(&DeployTask{}).Where("id = ? AND status = ?", task.ID, TaskStatusRunning).Updates(map[string]any{
				"status": TaskStatusFailed, "finished_at": now, "updated_by": actor, "updated_at": now,
			}).Error; err != nil {
				return err
			}
			task.Status = TaskStatusFailed
			s.appendTaskEvent(task.ID, 0, 0, "task.reconciled_stale", TaskStatusRunning, TaskStatusFailed, actor, "", "worker lease expired")
		}
	}
	// A task may be cleaned up after it reaches a terminal state. Running and
	// transition states remain protected so a worker cannot race a soft delete.
	if task.Status == TaskStatusRunning || task.Status == TaskStatusPauseRequested ||
		task.Status == TaskStatusCancelRequested {
		return errors.New(errDeployTaskInvalidDeleteState)
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(taskIDWhereClause, task.ID).Delete(&DeployTaskHost{}).Error; err != nil {
			return err
		}
		result := tx.Model(task).Updates(map[string]interface{}{
			"updated_by": actor,
			"updated_at": time.Now(),
		}).Delete(task)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errDeployTaskNotFound
		}
		return nil
	})
}

func (s *DeployService) StartTask(id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	credentialVersion := uint64(0)
	key := strings.TrimSpace(req.IdempotencyKey)

	// Reject or replay non-draft/pending states before attempting any claim.
	switch task.Status {
	case TaskStatusRunning:
		if key != "" && key == task.StartRequestKey {
			return s.getTaskReused(id, dataScope)
		}
		return nil, errors.New(errDeployTaskAlreadyRunning)
	case TaskStatusSuccess, TaskStatusFailed, TaskStatusCanceled,
		TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested:
		return nil, errors.New("business.deploy.task.invalidStartState")
	}
	if task.ExecutorType == ExecutorTypeSSH {
		if err := validateCredentialReferenceRequest(req); err != nil {
			return nil, err
		}
		resolvedReq, version, err := s.resolveStartCredential(req)
		if err != nil {
			return nil, err
		}
		req, credentialVersion = resolvedReq, version
	}

	// Validate the frozen execution snapshot before mutating anything.
	if _, err := s.resolveTaskExecutionPlan(*task); err != nil {
		return nil, mapDeployTaskExecutionPlanError(err)
	}
	hosts, err := s.resolveTaskTargets(task, dataScope)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.New("business.deploy.task.emptyResolvedTargets")
	}
	if task.ExecutorType == ExecutorTypeSSH {
		if err := validateDeploySSHStartRequest(req); err != nil {
			return nil, err
		}
		if err := s.validateTaskExecutionPlan(*task, hosts); err != nil {
			return nil, mapDeployTaskExecutionPlanError(err)
		}
	}

	now := time.Now()
	timeout := normalizeDeployExecutionTimeout(req.TimeoutSeconds)
	leaseOwner := deployHostLeaseOwner(task.ID)
	claimErr := s.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&DeployTask{}).
			Where(idWhereClause, task.ID).
			Where("status IN ?", []string{TaskStatusDraft, TaskStatusPending}).
			Updates(map[string]interface{}{
				"status":                    TaskStatusRunning,
				"started_at":                &now,
				"start_request_key":         key,
				"credential_ref_id":         req.CredentialRefID,
				"credential_ref_version":    credentialVersion,
				"ssh_host_fingerprint":      strings.TrimSpace(req.HostFingerprint),
				"execution_timeout_seconds": int(timeout.Seconds()),
				"updated_by":                actor,
				"updated_at":                now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New(errDeployTaskAlreadyRunning)
		}

		targetSnapshot := buildDeployTargetSnapshot(task, hosts, now)
		targetJSON, _ := json.Marshal(targetSnapshot)
		if err := tx.Model(&DeployTask{}).Where(idWhereClause, task.ID).Update("target_snapshot", datatypes.JSON(targetJSON)).Error; err != nil {
			return err
		}

		for _, host := range hosts {
			if err := acquireHostLease(tx, host.ID, task.ID, leaseOwner, now); err != nil {
				return err
			}
			initialTrace, _ := json.Marshal([]map[string]any{
				{"at": now.Format(time.RFC3339), "phase": "start", "message": "Task host queued"},
			})
			row := DeployTaskHost{
				TaskID:          task.ID,
				HostID:          host.ID,
				Hostname:        host.Hostname,
				HostIP:          host.IP,
				SSHPort:         host.SSHPort,
				OS:              host.OS,
				BusinessScopeID: host.BusinessScopeID,
				Status:          TaskHostStatusRunning,
				TraceSteps:      datatypes.JSON(initialTrace),
				StartedAt:       &now,
				ResolvedAt:      &now,
				UpdatedBy:       actor,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			// FZ-03: persist execution steps for checkpoint recovery.
			var snapshot deployExecutionSnapshot
			if err := json.Unmarshal(task.ExecutionSnapshot, &snapshot); err == nil && len(snapshot.Steps) > 0 {
				if err := persistTaskSteps(tx, task.ID, row.ID, snapshot.Steps); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if claimErr != nil {
		if strings.TrimSpace(errorText(claimErr)) == errDeployTaskAlreadyRunning {
			current, loadErr := s.loadVisibleTask(id, dataScope)
			if loadErr == nil && current.Status == TaskStatusRunning && key != "" && current.StartRequestKey == key {
				return s.getTaskReused(id, dataScope)
			}
			return nil, errors.New(errDeployTaskAlreadyRunning)
		}
		return nil, claimErr
	}
	s.appendTaskEvent(task.ID, 0, 0, "task.started", task.Status, TaskStatusRunning, actor, key, "")
	if err := s.beginServiceInstanceState(*task, actor, dataScope); err != nil {
		now := time.Now()
		_ = s.db.Model(&DeployTask{}).Where(idWhereClause, task.ID).Updates(map[string]interface{}{
			"status": TaskStatusFailed, "finished_at": &now, "updated_by": actor, "updated_at": now,
		}).Error
		_ = s.db.Model(&DeployTaskHost{}).Where("task_id = ? AND status IN ?", task.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
			"status": TaskHostStatusFailed, "error_message": err.Error(), "finished_at": &now, "updated_by": actor, "updated_at": now,
		}).Error
		for _, host := range hosts {
			_ = releaseHostLease(s.db, host.ID, deployHostLeaseOwner(task.ID))
		}
		return nil, err
	}
	if task.ExecutorType == ExecutorTypeSSH {
		if s.asyncExecution {
			// 记录任务开始 metrics
			recordTaskStart(task)
			go s.runAsyncTask(task.ID, actor)
		} else if err := s.executeSSHTask(*task, hosts, req, actor); err != nil {
			return nil, err
		}
	}
	return s.GetTask(id, dataScope)
}

func (s *DeployService) runAsyncTask(taskID uint64, actor string) {
	if _, loaded := s.taskWorkers.LoadOrStore(taskID, struct{}{}); loaded {
		return
	}
	defer s.taskWorkers.Delete(taskID)
	workerID := fmt.Sprintf("deploy-worker:%d", time.Now().UnixNano())
	_, _ = s.ReconcileDeployAttempts(time.Now())
	var task DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return
	}
	req, err := s.startRequestForTask(task)
	if err != nil {
		s.failQueuedTask(taskID, actor, err)
		return
	}
	var hosts []DeployTaskHost
	if err := s.db.Where("task_id = ?", taskID).Order("id ASC").Find(&hosts).Error; err != nil {
		return
	}
	if err := s.ensureTaskHostLeases(task, hosts); err != nil {
		s.failQueuedTask(taskID, actor, err)
		return
	}
	taskContext, cancelTask := context.WithCancel(context.Background())
	s.taskCancels.Store(taskID, cancelTask)
	defer func() {
		cancelTask()
		s.taskCancels.Delete(taskID)
	}()
	maxParallelHosts := deployMaxParallelHosts(task)
	continueOnHostFailure := taskContinueOnHostFailure(task)
	semaphore := make(chan struct{}, maxParallelHosts)
	var waitGroup sync.WaitGroup
	for _, host := range hosts {
		if s.taskCancellationRequested(taskID) || s.taskExecutionSuspended(taskID) {
			break
		}
		waitGroup.Add(1)
		go func(current DeployTaskHost) {
			defer waitGroup.Done()
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-taskContext.Done():
				return
			}
			if failed := s.runAsyncTaskHost(taskContext, task, current, req, actor, workerID); failed && !continueOnHostFailure && !s.taskCancellationRequested(taskID) && !s.taskExecutionSuspended(taskID) {
				cancelTask()
			}
		}(host)
	}
	waitGroup.Wait()
	s.releaseInactiveTaskHostLeases(taskID, hosts)
	if s.taskCancellationRequested(taskID) {
		s.finalizeTaskCancellation(taskID, actor)
	} else if !s.taskExecutionSuspended(taskID) {
		_ = s.recomputeTaskStatus(taskID, actor)
	}
}

func (s *DeployService) runAsyncTaskHost(taskContext context.Context, task DeployTask, host DeployTaskHost, req StartTaskRequest, actor, workerID string) bool {
	for attemptNo := 0; attemptNo < deployAttemptMax; attemptNo++ {
		if s.taskCancellationRequested(task.ID) || s.taskExecutionSuspended(task.ID) {
			return false
		}
		attempt := s.claimAttempt(task.ID, host.ID, workerID)
		if attempt == nil {
			return false
		}
		target := cmdbHostSnapshot{ID: host.HostID, Hostname: host.Hostname, IP: host.HostIP, SSHPort: host.SSHPort, OS: host.OS, BusinessScopeID: host.BusinessScopeID}
		hostContext, cancelHost := context.WithTimeout(taskContext, deployExecutionTimeout(task))
		err := s.executeTaskHostContext(hostContext, task, target, taskHostToResponse(&host), req, actor)
		cancelHost()
		if s.taskExecutionSuspended(task.ID) {
			s.finishAttemptWithStatus(attempt.ID, workerID, "paused", nil)
			return false
		}
		if s.taskCancellationRequested(task.ID) {
			s.finishAttemptWithStatus(attempt.ID, workerID, "canceled", nil)
			return false
		}
		var completed DeployTaskHost
		if loadErr := s.db.First(&completed, host.ID).Error; loadErr != nil {
			err = loadErr
		} else if completed.Status == TaskHostStatusFailed {
			err = errors.New(completed.ErrorMessage)
		} else if completed.Status == TaskHostStatusSkipped {
			err = errors.New("business.deploy.task.canceled")
		}
		if err == nil {
			s.finishAttempt(attempt.ID, workerID, nil)
			return false
		}
		s.finishAttempt(attempt.ID, workerID, err)
		if attemptNo+1 < deployAttemptMax && !s.taskCancellationRequested(task.ID) && !s.taskExecutionSuspended(task.ID) {
			_ = s.db.Model(&DeployTaskHost{}).Where("id = ?", host.ID).Updates(map[string]any{"status": TaskHostStatusRunning, "error_message": "", "finished_at": nil, "reported_at": nil, "updated_at": time.Now()}).Error
			continue
		}
		return true
	}
	return true
}

func (s *DeployService) ensureTaskHostLeases(task DeployTask, hosts []DeployTaskHost) error {
	now := time.Now()
	owner := deployHostLeaseOwner(task.ID)
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, host := range hosts {
			if isTerminalTaskHostStatus(host.Status) {
				continue
			}
			if err := acquireHostLease(tx, host.HostID, task.ID, owner, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *DeployService) releaseInactiveTaskHostLeases(taskID uint64, hosts []DeployTaskHost) {
	owner := deployHostLeaseOwner(taskID)
	for _, host := range hosts {
		active, err := s.hasActiveTaskAttempt(taskID, host.ID, time.Now())
		if err == nil && !active {
			_ = releaseHostLease(s.db, host.HostID, owner)
		}
	}
}

func deployMaxParallelHosts(task DeployTask) int {
	const maxParallelHosts = 16
	value := decodeJSONMap(task.TemplateParams)["maxParallelHosts"]
	parallel := 1
	switch typed := value.(type) {
	case float64:
		parallel = int(typed)
	case int:
		parallel = typed
	}
	if parallel < 1 {
		return 1
	}
	if parallel > maxParallelHosts {
		return maxParallelHosts
	}
	return parallel
}

func taskContinueOnHostFailure(task DeployTask) bool {
	snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
	if err != nil {
		return false
	}
	for _, step := range snapshot.Steps {
		if step.ContinueOnHostFailure || step.FailurePolicy == FailurePolicyContinueHosts {
			return true
		}
	}
	return false
}

func normalizeDeployExecutionTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return defaultDeployExecutionTimeout
	}
	duration := time.Duration(seconds) * time.Second
	if duration > maxDeployExecutionTimeout {
		return maxDeployExecutionTimeout
	}
	return duration
}

func deployExecutionTimeout(task DeployTask) time.Duration {
	return normalizeDeployExecutionTimeout(task.ExecutionTimeoutSeconds)
}

func (s *DeployService) startRequestForTask(task DeployTask) (StartTaskRequest, error) {
	if task.ExecutorType != ExecutorTypeSSH {
		return StartTaskRequest{}, nil
	}
	if task.CredentialRefID == 0 || strings.TrimSpace(task.SSHHostFingerprint) == "" {
		return StartTaskRequest{}, errors.New("business.deploy.task.execution_snapshot_missing")
	}
	req, version, err := s.resolveStartCredential(StartTaskRequest{CredentialRefID: task.CredentialRefID, HostFingerprint: task.SSHHostFingerprint, TimeoutSeconds: task.ExecutionTimeoutSeconds})
	if err != nil {
		return StartTaskRequest{}, err
	}
	if version != task.CredentialRefVersion {
		return StartTaskRequest{}, errors.New("business.deploy.credential.version_changed")
	}
	return req, nil
}

func (s *DeployService) failQueuedTask(taskID uint64, actor string, reason error) {
	now := time.Now()
	_ = s.db.Model(&DeployTaskHost{}).Where("task_id = ? AND status IN ?", taskID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]any{"status": TaskHostStatusFailed, "error_message": truncateDeployLog(reason.Error(), 480), "finished_at": &now, "updated_by": actor, "updated_at": now}).Error
	_ = s.db.Model(&DeployTask{}).Where("id = ? AND status = ?", taskID, TaskStatusRunning).Updates(map[string]any{"status": TaskStatusFailed, "finished_at": &now, "updated_by": actor, "updated_at": now}).Error
	s.appendTaskEvent(taskID, 0, 0, "task.failed", TaskStatusRunning, TaskStatusFailed, actor, "", truncateDeployLog(reason.Error(), 256))
}

func (s *DeployService) taskCanceled(taskID uint64) bool {
	var task DeployTask
	return s.db.Select("status").First(&task, taskID).Error == nil && task.Status == TaskStatusCanceled
}

func (s *DeployService) taskCancellationRequested(taskID uint64) bool {
	var task DeployTask
	if s.db.Select("status").First(&task, taskID).Error != nil {
		return false
	}
	return task.Status == TaskStatusCancelRequested || task.Status == TaskStatusCanceled
}

func (s *DeployService) finalizeTaskCancellation(taskID uint64, actor string) {
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).Where("id = ? AND status = ?", taskID, TaskStatusCancelRequested).Updates(map[string]any{
		"status": TaskStatusCanceled, "finished_at": &now, "updated_by": actor, "updated_at": now,
	})
	if transition.Error == nil && transition.RowsAffected > 0 {
		s.appendTaskEvent(taskID, 0, 0, "task.canceled", TaskStatusCancelRequested, TaskStatusCanceled, actor, "", "")
	}
}

func (s *DeployService) taskExecutionSuspended(taskID uint64) bool {
	var task DeployTask
	if s.db.Select("status").First(&task, taskID).Error != nil {
		return false
	}
	return task.Status == TaskStatusPauseRequested || task.Status == TaskStatusPaused
}

func (s *DeployService) claimAttempt(taskID, taskHostID uint64, workerID string) *DeployTaskAttempt {
	now := time.Now()
	lease := now.Add(5 * time.Minute)
	var attempt DeployTaskAttempt
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var taskHost DeployTaskHost
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&taskHost, taskHostID).Error; err != nil {
			return err
		}
		var active int64
		if err := tx.Model(&DeployTaskAttempt{}).Where("task_host_id = ? AND status = ? AND lease_expires_at > ?", taskHostID, "running", now).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return errors.New("business.deploy.attempt.active")
		}
		var chargeableCount int64
		if err := tx.Model(&DeployTaskAttempt{}).Where("task_host_id = ? AND status <> ?", taskHostID, "paused").Count(&chargeableCount).Error; err != nil {
			return err
		}
		if chargeableCount >= deployAttemptMax {
			return errors.New("business.deploy.attempt.exhausted")
		}
		var maxAttemptNo int
		if err := tx.Model(&DeployTaskAttempt{}).Where("task_host_id = ?", taskHostID).Select("COALESCE(MAX(attempt_no), 0)").Scan(&maxAttemptNo).Error; err != nil {
			return err
		}
		attempt = DeployTaskAttempt{TaskID: taskID, TaskHostID: taskHostID, AttemptNo: maxAttemptNo + 1, Status: "running", WorkerID: workerID, LeaseExpiresAt: &lease, StartedAt: &now, CreatedAt: now, UpdatedAt: now}
		return tx.Create(&attempt).Error
	})
	if err != nil {
		return nil
	}
	return &attempt
}

// ReconcileDeployAttempts makes abandoned worker claims visible to the next
// worker. It is safe to call periodically or during worker startup.
func (s *DeployService) ReconcileDeployAttempts(now time.Time) (int64, error) {
	result := s.db.Model(&DeployTaskAttempt{}).Where("status = ? AND lease_expires_at IS NOT NULL AND lease_expires_at < ?", "running", now).Updates(map[string]any{"status": "retryable", "error_message": "worker lease expired", "finished_at": now, "lease_expires_at": nil, "updated_at": now})
	return result.RowsAffected, result.Error
}

// ReconcileDeployQueue resumes durable running tasks after a worker restart.
// Claims remain protected by attempt leases, so concurrent process startup is
// safe and at most one worker can execute a task host at a time.
func (s *DeployService) ReconcileDeployQueue(actor string) (int, error) {
	if _, err := s.ReconcileDeployAttempts(time.Now()); err != nil {
		return 0, err
	}
	// FZ-03: reconcile pause_requested → paused after worker restart.
	now := time.Now()
	var pausedTaskIDs []uint64
	s.db.Model(&DeployTask{}).Where("status = ?", TaskStatusPauseRequested).Pluck("id", &pausedTaskIDs)
	pauseResult := s.db.Model(&DeployTask{}).
		Where("status = ?", TaskStatusPauseRequested).
		Updates(map[string]interface{}{
			"status":     TaskStatusPaused,
			"updated_by": actor,
			"updated_at": now,
		})
	if pauseResult.Error == nil && pauseResult.RowsAffected > 0 {
		for _, taskID := range pausedTaskIDs {
			s.appendTaskEvent(taskID, 0, 0, "task.paused", TaskStatusPauseRequested, TaskStatusPaused, actor, "", "worker reconciliation")
		}
	}
	var tasks []DeployTask
	if err := s.db.Where("status = ? AND executor_type = ?", TaskStatusRunning, ExecutorTypeSSH).Find(&tasks).Error; err != nil {
		return 0, err
	}
	for _, task := range tasks {
		go s.runAsyncTask(task.ID, actor)
	}
	return len(tasks), nil
}

func (s *DeployService) finishAttempt(id uint64, workerID string, runErr error) {
	status := "success"
	if runErr != nil {
		status = "failed"
	}
	s.finishAttemptWithStatus(id, workerID, status, runErr)
}

func (s *DeployService) finishAttemptWithStatus(id uint64, workerID, status string, runErr error) {
	now := time.Now()
	updates := map[string]any{"status": status, "finished_at": &now, "updated_at": now, "lease_expires_at": nil}
	if runErr != nil {
		updates["error_message"] = truncateDeployLog(runErr.Error(), 480)
	}
	_ = s.db.Model(&DeployTaskAttempt{}).Where("id = ? AND worker_id = ?", id, workerID).Updates(updates).Error
}

func (s *DeployService) CancelTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	now := time.Now()
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	allowed := map[string]bool{
		TaskStatusPending:         true,
		TaskStatusRunning:         true,
		TaskStatusPauseRequested:  true,
		TaskStatusPaused:          true,
		TaskStatusCancelRequested: false,
	}
	if !allowed[task.Status] {
		return nil, errors.New(errDeployTaskInvalidCancelState)
	}
	nextStatus := TaskStatusCancelRequested
	if task.Status == TaskStatusPending || task.Status == TaskStatusPaused {
		nextStatus = TaskStatusCanceled
	}
	updates := map[string]interface{}{
		"status":     nextStatus,
		"updated_by": actor,
		"updated_at": now,
	}
	if nextStatus == TaskStatusCanceled {
		updates["finished_at"] = &now
	}
	transition := s.db.Model(&DeployTask{}).
		Where(idWhereClause+" AND status = ?", task.ID, task.Status).
		Updates(updates)
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidCancelState)
	}
	if err := s.db.Model(&DeployTaskHost{}).Where("task_id = ? AND status IN ?", id, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
		"status":      TaskHostStatusSkipped,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error; err != nil {
		return nil, err
	}
	if cancel, ok := s.taskCancels.Load(id); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
	}
	eventType := "task.cancel_requested"
	if nextStatus == TaskStatusCanceled {
		eventType = "task.canceled"
	}
	s.appendTaskEvent(task.ID, 0, 0, eventType, task.Status, nextStatus, actor, "", "")
	return s.GetTask(id, dataScope)
}

// PauseTask requests cooperative pause for a running task. The task enters
// pause_requested; the executor must finish the current step, then the worker
// transitions the task to paused.
func (s *DeployService) PauseTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusRunning {
		return nil, errors.New(errDeployTaskInvalidPauseState)
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idWhereClause+" AND status = ?", task.ID, TaskStatusRunning).
		Updates(map[string]interface{}{
			"status":     TaskStatusPauseRequested,
			"updated_by": actor,
			"updated_at": now,
		})
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidPauseState)
	}
	s.appendTaskEvent(task.ID, 0, 0, "task.pause_requested", TaskStatusRunning, TaskStatusPauseRequested, actor, "", "")
	// Propagate cancellation to the current executor step so it finishes
	// promptly; the worker will transition to paused after the step returns.
	if cancel, ok := s.taskCancels.Load(id); ok {
		if cancelFunc, ok := cancel.(context.CancelFunc); ok {
			cancelFunc()
		}
	}
	return s.GetTask(id, dataScope)
}

// ResumeTask resumes a paused task. The task re-enters running and the worker
// re-schedules unfinished hosts from the last checkpoint.
func (s *DeployService) ResumeTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusPaused {
		return nil, errors.New(errDeployTaskInvalidResumeState)
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idWhereClause+" AND status = ?", task.ID, TaskStatusPaused).
		Updates(map[string]interface{}{
			"status":     TaskStatusRunning,
			"updated_by": actor,
			"updated_at": now,
		})
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidResumeState)
	}
	s.appendTaskEvent(task.ID, 0, 0, "task.resumed", TaskStatusPaused, TaskStatusRunning, actor, "", "")
	// Re-run the async worker to pick up unfinished hosts from the checkpoint.
	if s.asyncExecution {
		go s.runAsyncTask(task.ID, actor)
	}
	return s.GetTask(id, dataScope)
}

// RetryTask creates a new attempt for a failed task. Successful prior host
// results are retained; only failed, canceled, or not-started hosts are
// scheduled again.
func (s *DeployService) RetryTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusFailed {
		return nil, errors.New(errDeployTaskInvalidRetryState)
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idWhereClause+" AND status = ?", task.ID, TaskStatusFailed).
		Updates(map[string]interface{}{
			"status":      TaskStatusRunning,
			"finished_at": nil,
			"updated_by":  actor,
			"updated_at":  now,
		})
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidRetryState)
	}
	s.appendTaskEvent(task.ID, 0, 0, "task.retried", TaskStatusFailed, TaskStatusRunning, actor, "", "")
	// Reset failed/canceled hosts to running so the worker re-executes them.
	_ = s.db.Model(&DeployTaskHost{}).
		Where("task_id = ? AND status IN ?", task.ID, []string{TaskHostStatusFailed, TaskHostStatusSkipped}).
		Updates(map[string]interface{}{
			"status":        TaskHostStatusRunning,
			"error_message": "",
			"finished_at":   nil,
			"reported_at":   nil,
			"updated_at":    now,
		})
	if s.asyncExecution {
		go s.runAsyncTask(task.ID, actor)
	}
	return s.GetTask(id, dataScope)
}

// --- FZ-03: Event append and step persistence ---

// appendTaskEvent writes an append-only event record for a task transition.
// The sequence is monotonically increasing per task and never reused.
func (s *DeployService) appendTaskEvent(taskID, taskHostID, taskStepID uint64, eventType, fromStatus, toStatus, actor, requestID, reason string) {
	err := s.withTaskSequence(taskID, &DeployTaskEvent{}, func(sequence uint64, tx *gorm.DB) error {
		return tx.Create(&DeployTaskEvent{
			TaskID: taskID, TaskHostID: taskHostID, TaskStepID: taskStepID,
			Sequence: sequence, EventType: eventType, FromStatus: fromStatus,
			ToStatus: toStatus, Actor: actor, RequestID: requestID,
			Reason: reason, CreatedAt: time.Now(),
		}).Error
	})
	if err != nil {
		slog.Error("append deploy task event failed", "task_id", taskID, "event_type", eventType, "error", err)
	}
}

func (s *DeployService) withTaskSequence(taskID uint64, model any, write func(uint64, *gorm.DB) error) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var task DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&task, taskID).Error; err != nil {
			return err
		}
		var maxSequence uint64
		if err := tx.Model(model).Where("task_id = ?", taskID).Select("COALESCE(MAX(sequence), 0)").Scan(&maxSequence).Error; err != nil {
			return err
		}
		return write(maxSequence+1, tx)
	})
}

// appendTaskEventTx writes an append-only event within an existing transaction.
// persistTaskSteps creates or updates step rows for a task host from the
// frozen execution snapshot. Steps are idempotent: existing steps are not
// overwritten, only missing steps are inserted.
func persistTaskSteps(db *gorm.DB, taskID, taskHostID uint64, steps []deployStepSnapshot) error {
	for i, step := range steps {
		stepKey := step.StepCode
		if stepKey == "" {
			stepKey = fmt.Sprintf("step-%d", i+1)
		}
		var existing DeployTaskStep
		err := db.Where("task_id = ? AND task_host_id = ? AND step_key = ?", taskID, taskHostID, stepKey).First(&existing).Error
		switch {
		case err == nil:
			// Step already exists; do not overwrite.
			continue
		case errors.Is(err, gorm.ErrRecordNotFound):
			dependencyJSON, marshalErr := json.Marshal(step.Dependencies)
			if marshalErr != nil {
				return marshalErr
			}
			conditionJSON, marshalErr := json.Marshal(step.Condition)
			if marshalErr != nil {
				return marshalErr
			}
			failurePolicy := step.FailurePolicy
			if failurePolicy == "" {
				failurePolicy = FailurePolicyFailFast
			}
			newStep := DeployTaskStep{
				TaskID:         taskID,
				TaskHostID:     taskHostID,
				StepKey:        stepKey,
				StepOrder:      i + 1,
				StepType:       step.StepType,
				Action:         step.Action,
				DependencyJSON: datatypes.JSON(dependencyJSON),
				ConditionJSON:  datatypes.JSON(conditionJSON),
				FailurePolicy:  failurePolicy,
				Status:         StepStatusPending,
				StateVersion:   1,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}
			if err := db.Create(&newStep).Error; err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

// loadTaskEvents returns the event log for a task, optionally filtered by sequence.
func (s *DeployService) loadTaskEvents(taskID uint64, afterSequence uint64) ([]DeployTaskEvent, error) {
	var events []DeployTaskEvent
	db := s.db.Where("task_id = ?", taskID)
	if afterSequence > 0 {
		db = db.Where("sequence > ?", afterSequence)
	}
	if err := db.Order("sequence ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// --- FZ-04: Append-only log chunks ---

const maxLogChunkSize = 16384 // 16 KiB

// appendLogChunk writes append-only segments for a task host step.
func (s *DeployService) appendLogChunk(taskID, taskHostID, taskStepID uint64, stream, content string) {
	if len(content) == 0 {
		return
	}
	for len(content) > 0 {
		segment, remaining := splitLogSegment(content)
		err := s.withTaskSequence(taskID, &DeployTaskLogChunk{}, func(sequence uint64, tx *gorm.DB) error {
			return tx.Create(&DeployTaskLogChunk{
				TaskID: taskID, TaskHostID: taskHostID, TaskStepID: taskStepID,
				Sequence: sequence, Stream: stream, Content: segment, CreatedAt: time.Now(),
			}).Error
		})
		if err != nil {
			slog.Error("append deploy task log failed", "task_id", taskID, "stream", stream, "error", err)
			return
		}
		content = remaining
	}
}

func splitLogSegment(content string) (string, string) {
	if len(content) <= maxLogChunkSize {
		return content, ""
	}
	end := maxLogChunkSize
	for end > 0 && !utf8.ValidString(content[:end]) {
		end--
	}
	if end == 0 {
		end = maxLogChunkSize
	}
	return content[:end], content[end:]
}

// loadLogChunks returns log chunks for a task, optionally after a given sequence.
func (s *DeployService) loadLogChunks(taskID uint64, afterSequence uint64) ([]DeployTaskLogChunk, error) {
	var chunks []DeployTaskLogChunk
	db := s.db.Where("task_id = ?", taskID)
	if afterSequence > 0 {
		db = db.Where("sequence > ?", afterSequence)
	}
	if err := db.Order("sequence ASC").Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// WriteLog implements the executor.LogSink interface for persistence.
func (s *DeployService) WriteLog(taskID, taskHostID uint64, stepKey, stream, content string) {
	secrets := s.renderSecrets(taskID)
	redacted := redactDeploySecrets(content, secrets)
	checkpoint, err := s.loadTaskStep(taskID, taskHostID, stepKey)
	if err != nil {
		slog.Error("load deploy task step for log failed", "task_id", taskID, "task_host_id", taskHostID, "step_key", stepKey, "error", err)
		return
	}
	s.appendLogChunk(taskID, taskHostID, checkpoint.ID, stream, redacted)
}

func (s *DeployService) MarkHostResult(hostID uint64, req MarkHostResultRequest, actor string, dataScope *common.DataScopeReq) (*TaskHostResponse, error) {
	return s.markHostResultWithSummary(hostID, req, actor, dataScope, deployExecutionSummary{})
}

func (s *DeployService) markHostResultWithSummary(hostID uint64, req MarkHostResultRequest, actor string, dataScope *common.DataScopeReq, summary deployExecutionSummary) (*TaskHostResponse, error) {
	if req.Status != TaskHostStatusSuccess && req.Status != TaskHostStatusFailed && req.Status != TaskHostStatusSkipped {
		return nil, errors.New("business.deploy.taskHost.invalidResultState")
	}
	req.Stdout = truncateDeployLog(req.Stdout, 60000)
	req.Stderr = truncateDeployLog(req.Stderr, 60000)
	req.ErrorMessage = truncateDeployLog(req.ErrorMessage, 480)
	if req.Status == TaskHostStatusFailed && strings.TrimSpace(req.ErrorMessage) == "" {
		return nil, errors.New("business.deploy.taskHost.markFailed.reasonRequired")
	}
	var host DeployTaskHost
	if err := s.db.First(&host, hostID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("business.deploy.taskHost.notFound")
		}
		return nil, err
	}
	task, err := s.loadVisibleTask(host.TaskID, dataScope)
	if err != nil {
		return nil, err
	}

	reportKey := strings.TrimSpace(req.ReportKey)
	if isTerminalTaskHostStatus(host.Status) {
		if sameTaskHostReport(host, req, reportKey) {
			resp := taskHostToResponse(&host)
			return &resp, nil
		}
		return nil, errors.New(errDeployTaskHostStaleReport)
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":        req.Status,
		"stdout":        req.Stdout,
		"stderr":        req.Stderr,
		"error_message": req.ErrorMessage,
		"executor_id":   req.ExecutorID,
		"report_key":    reportKey,
		"finished_at":   &now,
		"reported_at":   &now,
		"updated_by":    actor,
		"updated_at":    now,
	}
	transition := s.db.Model(&DeployTaskHost{}).
		Where("id = ? AND status IN ?", host.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).
		Updates(updates)
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		var current DeployTaskHost
		if err := s.db.First(&current, host.ID).Error; err != nil {
			return nil, err
		}
		if sameTaskHostReport(current, req, reportKey) {
			resp := taskHostToResponse(&current)
			return &resp, nil
		}
		return nil, errors.New(errDeployTaskHostStaleReport)
	}
	resultTrace := []map[string]any{
		{"at": now.Format(time.RFC3339), "phase": "result", "message": fmt.Sprintf("status=%s", req.Status)},
	}
	if strings.TrimSpace(req.ErrorMessage) != "" {
		resultTrace = append(resultTrace, map[string]any{
			"at": now.Format(time.RFC3339), "phase": "error", "message": strings.TrimSpace(req.ErrorMessage),
		})
	}
	_ = s.appendTaskHostTrace(host.TaskID, host.ID, resultTrace)
	if req.Status == TaskHostStatusSuccess {
		writeback := cmdb.DeployHostWritebackRequest{
			HostID: host.HostID,
			Status: hostStatusForAction(task.Action),
			Actor:  actor,
		}
		if len(summary.RemovedComponentNames) > 0 || len(summary.InstalledComponents) > 0 {
			writeback.RemovedComponentNames = normalizeDeployStringValues(summary.RemovedComponentNames)
			writeback.InstalledComponents = summary.InstalledComponents
		} else if normalizeTaskAction(task.Action) == TaskActionUninstall {
			writeback.RemovedComponentNames = []string{task.PackageName}
		} else {
			writeback.InstalledComponents = []cmdb.InstalledComponentUpsert{{
				Name:           task.PackageName,
				Version:        task.PackageVersion,
				DeployedAt:     now,
				DeployTaskID:   task.ID,
				DeployTaskName: task.Name,
				ExecutorType:   task.ExecutorType,
			}}
		}
		if err := s.cmdbCapability.WriteDeployHostResult(writeback); err != nil {
			return nil, err
		}
	}
	if err := s.recomputeTaskStatus(host.TaskID, actor); err != nil {
		return nil, err
	}
	var refreshedTask DeployTask
	if err := s.db.First(&refreshedTask, host.TaskID).Error; err != nil {
		return nil, err
	}
	if isTerminalTaskStatus(refreshedTask.Status) {
		if err := s.finishServiceInstanceState(refreshedTask, req, actor, dataScope); err != nil {
			return nil, err
		}
	}
	// An async worker persists the host result before it closes its attempt. Keep
	// the host lease until that attempt is finished so a retry cannot race a
	// terminal task status or a competing task claim.
	activeAttempt, attemptErr := s.hasActiveTaskAttempt(host.TaskID, host.ID, time.Now())
	if attemptErr != nil {
		return nil, attemptErr
	}
	if !activeAttempt {
		_ = releaseHostLease(s.db, host.HostID, deployHostLeaseOwner(host.TaskID))
	}
	if err := s.db.First(&host, hostID).Error; err != nil {
		return nil, err
	}
	// 记录主机结果 metrics
	recordHostResult(&host, task.ExecutorType)
	resp := taskHostToResponse(&host)
	return &resp, nil
}

func isTerminalTaskStatus(status string) bool {
	return status == TaskStatusSuccess || status == TaskStatusFailed || status == TaskStatusCanceled
}

func (s *DeployService) beginServiceInstanceState(task DeployTask, actor string, scope *common.DataScopeReq) error {
	if s.serviceState == nil || task.ServiceInstanceID == 0 {
		return nil
	}
	action, observed, desired, ok := deployStateBegin(task.Action)
	if !ok {
		return nil
	}
	return s.serviceState.ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
		InstanceID:     task.ServiceInstanceID,
		Action:         action,
		DesiredState:   desired,
		ObservedState:  observed,
		DesiredVersion: strings.TrimSpace(task.PackageVersion),
		CurrentVersion: "",
		CorrelationID:  fmt.Sprintf("deploy:%d:begin", task.ID),
	}, actor, scope)
}

func (s *DeployService) finishServiceInstanceState(task DeployTask, result MarkHostResultRequest, actor string, scope *common.DataScopeReq) error {
	if s.serviceState == nil || task.ServiceInstanceID == 0 {
		return nil
	}
	action, observed, desired, health, currentVersion, ok := deployStateFinish(task.Action, task.Status, task.PackageVersion, result.HealthState)
	if !ok {
		return nil
	}
	err := s.serviceState.ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
		InstanceID:     task.ServiceInstanceID,
		Action:         action,
		DesiredState:   desired,
		ObservedState:  observed,
		HealthState:    health,
		DesiredVersion: strings.TrimSpace(task.PackageVersion),
		CurrentVersion: currentVersion,
		HealthMessage:  firstNonBlank(result.HealthMessage, stateResultMessage(task)),
		HealthRevision: strings.TrimSpace(result.HealthRevision),
		CorrelationID:  fmt.Sprintf("deploy:%d:finish:%s", task.ID, task.Status),
	}, actor, scope)
	if err != nil || task.Status != TaskStatusSuccess || normalizeTaskAction(task.Action) != TaskActionUninstall {
		return err
	}
	return s.serviceState.ApplyServiceInstanceState(context.Background(), bizcap.ServiceInstanceStateTransition{
		InstanceID:    task.ServiceInstanceID,
		Action:        "retire",
		DesiredState:  "retired",
		ObservedState: "retired",
		HealthState:   "unknown",
		CorrelationID: fmt.Sprintf("deploy:%d:retire", task.ID),
	}, actor, scope)
}

func deployStateBegin(rawAction string) (action, observed, desired string, ok bool) {
	switch normalizeTaskAction(rawAction) {
	case TaskActionInstall, TaskActionReinstall:
		return TaskActionInstall, "installing", observedStateStopped, true
	case TaskActionUpgrade:
		return TaskActionUpgrade, "upgrading", "", true
	case TaskActionRollback:
		return TaskActionRollback, "upgrading", "", true
	case TaskActionStart:
		return TaskActionStart, "starting", TaskStatusRunning, true
	case TaskActionStop, TaskActionUninstall:
		return TaskActionStop, "stopping", observedStateStopped, true
	default:
		return "", "", "", false
	}
}

func deployStateFinish(rawAction, taskStatus, packageVersion, reportedHealth string) (action, observed, desired, health, currentVersion string, ok bool) {
	success := taskStatus == TaskStatusSuccess
	switch normalizeTaskAction(rawAction) {
	case TaskActionInstall, TaskActionReinstall:
		if success {
			return TaskActionInstall, observedStateStopped, observedStateStopped, observedStateUnknown, strings.TrimSpace(packageVersion), true
		}
		return TaskActionInstall, TaskStatusFailed, observedStateStopped, observedStateUnknown, "", true
	case TaskActionUpgrade:
		if success {
			health = strings.TrimSpace(reportedHealth)
			if health == "" {
				health = observedStateUnknown
			}
			return TaskActionUpgrade, TaskStatusRunning, TaskStatusRunning, health, strings.TrimSpace(packageVersion), true
		}
		return TaskActionUpgrade, TaskStatusFailed, "", observedStateUnknown, "", true
	case TaskActionRollback:
		if success {
			health = strings.TrimSpace(reportedHealth)
			if health == "" {
				health = observedStateUnknown
			}
			return TaskActionRollback, TaskStatusRunning, TaskStatusRunning, health, strings.TrimSpace(packageVersion), true
		}
		return TaskActionRollback, TaskStatusFailed, "", observedStateUnknown, "", true
	case TaskActionStart:
		if success {
			return TaskActionStart, TaskStatusRunning, TaskStatusRunning, observedStateUnknown, "", true
		}
		return TaskActionStart, TaskStatusFailed, TaskStatusRunning, observedStateUnknown, "", true
	case TaskActionHealth:
		if success {
			health = strings.TrimSpace(reportedHealth)
			if health == "" {
				health = observedStateUnknown
			}
			return TaskActionHealth, TaskStatusRunning, "", health, "", true
		}
		return TaskActionHealth, TaskStatusRunning, "", "unhealthy", "", true
	case TaskActionStop, TaskActionUninstall:
		if success {
			return TaskActionStop, observedStateStopped, observedStateStopped, observedStateUnknown, "", true
		}
		return TaskActionStop, TaskStatusFailed, "stopped", observedStateUnknown, "", true
	case TaskActionRetire:
		if success {
			return TaskActionRetire, "retired", "retired", observedStateUnknown, "", true
		}
		return "", "", "", "", "", false
	default:
		return "", "", "", "", "", false
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func stateResultMessage(task DeployTask) string {
	if task.Status == TaskStatusFailed {
		return fmt.Sprintf("deploy task %d failed", task.ID)
	}
	return ""
}

func (s *DeployService) resolveTaskTargets(task *DeployTask, dataScope *common.DataScopeReq) ([]cmdbHostSnapshot, error) {
	targetIDs := parseUint64JSON(task.TargetIDs)
	if len(targetIDs) == 0 {
		return nil, nil
	}
	rows, err := s.cmdbCapability.ResolveDeployTargets(cmdb.DeployHostResolveRequest{
		BusinessScopeID: task.BusinessScopeID,
		TargetType:      task.TargetType,
		TargetIDs:       targetIDs,
		DataScope:       dataScope,
	})
	if err != nil {
		return nil, err
	}
	result := make([]cmdbHostSnapshot, 0, len(rows))
	for _, row := range rows {
		result = append(result, cmdbHostSnapshot{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			SSHPort:           row.SSHPort,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeName: row.BusinessScopeName,
			LabelValues:       row.LabelValues,
			DeptID:            row.DeptID,
		})
	}
	return result, nil
}

func (s *DeployService) loadVisibleTask(id uint64, dataScope *common.DataScopeReq) (*DeployTask, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var task DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errDeployTaskNotFound
		}
		return nil, err
	}
	visible, err := s.taskVisible(&task, dataScope)
	if err != nil {
		return nil, err
	}
	if !visible {
		return nil, errDeployTaskForbidden
	}
	return &task, nil
}

func isDeployTaskNotFound(err error) bool {
	return errors.Is(err, errDeployTaskNotFound) || strings.TrimSpace(errorText(err)) == errDeployTaskNotFound.Error()
}

func isDeployTaskForbidden(err error) bool {
	return errors.Is(err, errDeployTaskForbidden) || strings.TrimSpace(errorText(err)) == errDeployTaskForbidden.Error()
}

func mapDeployTaskTemplateLookupError(err error) error {
	if err == nil {
		return nil
	}
	switch strings.TrimSpace(errorText(err)) {
	case "deploytemplate.not_found":
		return errors.New(errDeployTaskTemplateNotFound)
	case "deploytemplate.disabled":
		return errors.New(errDeployTaskTemplateDisabled)
	default:
		return err
	}
}

func mapDeployTaskExecutionPlanError(err error) error {
	if err == nil {
		return nil
	}
	switch strings.TrimSpace(errorText(err)) {
	case "deploytemplate.not_found":
		return errors.New(errDeployTaskTemplateNotFound)
	case "deploytemplate.disabled":
		return errors.New(errDeployTaskTemplateDisabled)
	case "deploytemplate.invalid":
		return errors.New(errDeployTaskTemplateInvalid)
	case "deploypackage.not_found":
		return errors.New(errDeployTaskPackageNotFound)
	case "deploypackage.disabled":
		return errors.New("business.deploy.task.packageDisabled")
	case "deploypackage.source_missing":
		return errors.New(errDeployTaskPackageSourceMissing)
	default:
		return err
	}
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *DeployService) filterVisibleTasks(rows []DeployTask, dataScope *common.DataScopeReq) ([]DeployTask, error) {
	if !requiresScopedTaskVisibility(dataScope) {
		return rows, nil
	}
	filtered := make([]DeployTask, 0, len(rows))
	for i := range rows {
		visible, err := s.taskVisible(&rows[i], dataScope)
		if err != nil {
			return nil, err
		}
		if visible {
			filtered = append(filtered, rows[i])
		}
	}
	return filtered, nil
}

func (s *DeployService) taskVisible(task *DeployTask, dataScope *common.DataScopeReq) (bool, error) {
	if task == nil {
		return false, nil
	}
	if !requiresScopedTaskVisibility(dataScope) {
		return true, nil
	}
	hasVisibleTaskHost, hasTaskHosts, err := s.taskHasVisibleTaskHost(task.ID, dataScope)
	if err != nil {
		return false, err
	}
	if hasTaskHosts {
		return hasVisibleTaskHost, nil
	}
	hosts, err := s.resolveTaskTargets(task, dataScope)
	if err != nil {
		return false, err
	}
	return len(hosts) > 0, nil
}

func (s *DeployService) taskHasVisibleTaskHost(taskID uint64, dataScope *common.DataScopeReq) (bool, bool, error) {
	var taskHostCount int64
	if err := s.db.Model(&DeployTaskHost{}).Where(taskIDWhereClause, taskID).Count(&taskHostCount).Error; err != nil {
		return false, false, err
	}
	if taskHostCount == 0 {
		return false, false, nil
	}
	if s.hostReader == nil {
		return false, true, errors.New("business.cmdb.readerNotConfigured")
	}
	var taskHosts []DeployTaskHost
	if err := s.db.Where(taskIDWhereClause, taskID).Find(&taskHosts).Error; err != nil {
		return false, true, err
	}
	ids := make([]uint64, 0, len(taskHosts))
	for _, taskHost := range taskHosts {
		ids = append(ids, taskHost.HostID)
	}
	page, err := s.hostReader.GetByIDs(context.Background(), bizcap.HostIDsQuery{HostIDs: ids, DataScope: dataScope})
	if err != nil {
		return false, true, err
	}
	return page.Total > 0, true, nil
}

func requiresScopedTaskVisibility(dataScope *common.DataScopeReq) bool {
	if dataScope == nil || dataScope.IsAdmin {
		return false
	}
	mode := strings.TrimSpace(dataScope.Mode)
	return mode != "" && mode != common.DataScopeModeAll
}

func (s *DeployService) recomputeTaskStatus(taskID uint64, actor string) error {
	var hosts []DeployTaskHost
	if err := s.db.Where(taskIDWhereClause, taskID).Find(&hosts).Error; err != nil {
		return err
	}
	if len(hosts) == 0 {
		return nil
	}
	allDone := true
	hasFailed := false
	allSkipped := true
	for _, host := range hosts {
		if host.Status == TaskHostStatusPending || host.Status == TaskHostStatusRunning {
			allDone = false
		}
		if host.Status == TaskHostStatusFailed {
			hasFailed = true
		}
		if host.Status != TaskHostStatusSkipped {
			allSkipped = false
		}
	}
	if !allDone {
		return nil
	}
	activeAttempt, err := s.hasActiveTaskAttempt(taskID, 0, time.Now())
	if err != nil {
		return err
	}
	if activeAttempt {
		return nil
	}
	status := TaskStatusSuccess
	if hasFailed {
		status = TaskStatusFailed
	}
	if allSkipped {
		status = TaskStatusCanceled
	}
	now := time.Now()
	err = s.db.Model(&DeployTask{}).Where(idWhereClause, taskID).Updates(map[string]interface{}{
		"status":      status,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error
	if err == nil {
		s.appendTaskEvent(taskID, 0, 0, "task.completed", TaskStatusRunning, status, actor, "", "")
		// 记录任务完成 metrics
		var task DeployTask
		if err := s.db.First(&task, taskID).Error; err == nil {
			if task.StartedAt != nil && task.FinishedAt != nil {
				duration := task.FinishedAt.Sub(*task.StartedAt)
				recordTaskComplete(&task, duration)
			}
			if status == TaskStatusFailed {
				recordTaskFailure(&task, "execution_failed")
				// Trigger auto-rollback if enabled
				if task.RollbackEnabled && task.RollbackStrategy == RollbackStrategyAuto {
					ctx := context.Background()
					if rollbackErr := s.TriggerAutoRollback(ctx, &task); rollbackErr != nil {
						// Log error but don't fail the original task status update
						s.appendTaskEvent(taskID, 0, 0, "rollback.auto_trigger_failed", "", "", "system", "", rollbackErr.Error())
					} else {
						s.appendTaskEvent(taskID, 0, 0, "rollback.auto_triggered", "", "", "system", "", "")
					}
				}
			}
		}
	}
	return err
}

// hasActiveTaskAttempt reports whether a worker currently owns a non-expired
// attempt. A zero taskHostID checks every host under the task.
func (s *DeployService) hasActiveTaskAttempt(taskID, taskHostID uint64, now time.Time) (bool, error) {
	db := s.db.Model(&DeployTaskAttempt{}).
		Where("task_id = ? AND status = ? AND (lease_expires_at IS NULL OR lease_expires_at > ?)", taskID, "running", now)
	if taskHostID > 0 {
		db = db.Where("task_host_id = ?", taskHostID)
	}
	var active int64
	if err := db.Count(&active).Error; err != nil {
		return false, err
	}
	return active > 0, nil
}

func (s *DeployService) packageExists(name string, version string, excludeID uint64) bool {
	var count int64
	db := s.db.Model(&DeployPackage{}).Where("name = ? AND version = ?", name, version)
	if excludeID > 0 {
		db = db.Where("id <> ?", excludeID)
	}
	_ = db.Count(&count).Error
	return count > 0
}

func validPackageStatus(status string) bool {
	return status == PackageStatusEnabled || status == PackageStatusDisabled
}

func (s *DeployService) isPackageReferenced(packageID uint64) (bool, error) {
	var count int64
	if err := s.db.Model(&DeployTask{}).Where("package_id = ?", packageID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *DeployService) isTemplateReferenced(templateID uint64) (bool, error) {
	var count int64
	if err := s.db.Model(&DeployTask{}).Where(templateIDWhereClause, templateID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func updatePackageChangesExecutionDefinition(req UpdatePackageRequest) bool {
	return req.Name != nil || req.Version != nil || req.InstallCommand != nil ||
		req.UninstallCommand != nil || req.ExecutionMode != nil || req.TemplateCode != nil ||
		req.TemplateConfig != nil || req.SourceObjectKey != nil || req.SourceFileName != nil ||
		req.SourceURL != nil
}

func updateTemplateChangesExecutionDefinition(req UpdateTemplateRequest) bool {
	return req.Name != nil || req.Version != nil || req.ExecutionMode != nil ||
		req.DefaultAction != nil || req.PackageID != nil || req.TemplateCode != nil ||
		req.TemplateConfig != nil || req.ParameterSchema != nil || req.Steps != nil
}

func validExecutionMode(mode string) bool {
	return mode == ExecutionModeFixed || mode == ExecutionModeOrchestrated
}

func normalizeTaskAction(action string) string {
	switch strings.TrimSpace(action) {
	case "", TaskActionInstall:
		return TaskActionInstall
	case TaskActionUninstall:
		return TaskActionUninstall
	case TaskActionUpgrade:
		return TaskActionUpgrade
	case TaskActionReinstall:
		return TaskActionReinstall
	case TaskActionStart:
		return TaskActionStart
	case TaskActionHealth:
		return TaskActionHealth
	case TaskActionStop:
		return TaskActionStop
	case TaskActionRollback:
		return TaskActionRollback
	case TaskActionRetire:
		return TaskActionRetire
	default:
		return strings.TrimSpace(action)
	}
}

func validTaskAction(action string) bool {
	switch action {
	case TaskActionInstall, TaskActionUninstall, TaskActionUpgrade, TaskActionReinstall,
		TaskActionStart, TaskActionHealth, TaskActionStop, TaskActionRollback, TaskActionRetire:
		return true
	default:
		return false
	}
}

func hostStatusForAction(action string) string {
	switch normalizeTaskAction(action) {
	case TaskActionUninstall, TaskActionRetire:
		return "assigned"
	default:
		return "online"
	}
}

func hostStatusAllowedForAction(status, action string) bool {
	switch normalizeTaskAction(action) {
	case TaskActionUninstall, TaskActionUpgrade, TaskActionStart, TaskActionHealth, TaskActionStop, TaskActionRollback:
		return strings.TrimSpace(status) == "online"
	case TaskActionReinstall, TaskActionRetire:
		return strings.TrimSpace(status) == "assigned" || strings.TrimSpace(status) == "online"
	default:
		return strings.TrimSpace(status) == "assigned" || strings.TrimSpace(status) == "online"
	}
}

func validTargetType(targetType string) bool {
	return targetType == TargetTypeHost || targetType == TargetTypeGroup
}

func validExecutorType(executorType string) bool {
	return executorType == ExecutorTypeManual || executorType == ExecutorTypeSimulated || executorType == ExecutorTypeAgent || executorType == ExecutorTypeSSH
}

func deployHostLeaseOwner(taskID uint64) string {
	return fmt.Sprintf("task:%d", taskID)
}

func buildDeployTargetSnapshot(task *DeployTask, hosts []cmdbHostSnapshot, now time.Time) deployTargetSnapshot {
	snapshot := deployTargetSnapshot{
		TargetType: task.TargetType,
		ResolvedAt: now.Format(time.RFC3339),
		Hosts:      make([]deployTargetHostSnapshot, 0, len(hosts)),
	}
	for _, host := range hosts {
		snapshot.Hosts = append(snapshot.Hosts, deployTargetHostSnapshot{
			HostID:            host.ID,
			Hostname:          host.Hostname,
			IP:                host.IP,
			SSHPort:           host.SSHPort,
			OS:                host.OS,
			BusinessScopeID:   host.BusinessScopeID,
			BusinessScopeName: host.BusinessScopeName,
			DeptID:            host.DeptID,
			Status:            host.Status,
		})
	}
	return snapshot
}

func acquireHostLease(tx *gorm.DB, hostID, taskID uint64, owner string, now time.Time) error {
	expiresAt := now.Add(defaultHostLeaseDuration)
	lease := DeployHostLease{HostID: hostID, TaskID: taskID, Owner: owner, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now}
	if err := tx.Create(&lease).Error; err == nil {
		return nil
	}
	var existing DeployHostLease
	if err := tx.Where("host_id = ?", hostID).First(&existing).Error; err != nil {
		return err
	}
	if existing.ExpiresAt.After(now) && existing.TaskID == taskID && existing.Owner == owner {
		return tx.Model(&DeployHostLease{}).Where("host_id = ? AND task_id = ? AND owner = ?", hostID, taskID, owner).Updates(map[string]any{"expires_at": expiresAt, "updated_at": now}).Error
	}
	if existing.ExpiresAt.After(now) {
		return errors.New(errDeployTaskLeaseConflict)
	}
	result := tx.Model(&DeployHostLease{}).
		Where("host_id = ? AND expires_at <= ?", hostID, now).
		Updates(map[string]interface{}{"task_id": taskID, "owner": owner, "expires_at": expiresAt, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(errDeployTaskLeaseConflict)
	}
	return nil
}

func releaseHostLease(db *gorm.DB, hostID uint64, owner string) error {
	return db.Where("host_id = ? AND owner = ?", hostID, owner).Delete(&DeployHostLease{}).Error
}

func isTerminalTaskHostStatus(status string) bool {
	return status == TaskHostStatusSuccess || status == TaskHostStatusFailed || status == TaskHostStatusSkipped
}

func sameTaskHostReport(host DeployTaskHost, req MarkHostResultRequest, reportKey string) bool {
	if reportKey != "" {
		return reportKey == host.ReportKey
	}
	return req.Status == host.Status
}

type deploySSHRunner interface {
	RunScript(ctx context.Context, script string) (stdout string, stderr string, err error)
	Close() error
}

type deploySSHClient struct {
	client *ssh.Client
}

func (c *deploySSHClient) RunScript(ctx context.Context, script string) (string, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer func() { _ = session.Close() }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	session.Stdin = strings.NewReader(script)
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ssh session goroutine panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
				done <- fmt.Errorf("ssh session panic: %v", r)
			}
		}()
		done <- session.Run("/bin/bash -se")
	}()
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = session.Close()
		err = ctx.Err()
	}
	return stdout.String(), stderr.String(), err
}

func runDeployScript(ctx context.Context, runner deploySSHRunner, script string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	return runner.RunScript(ctx, script)
}

func (c *deploySSHClient) Close() error {
	return c.client.Close()
}

func deployHostKeyCallback(expectedFingerprint string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if strings.TrimSpace(ssh.FingerprintSHA256(key)) != strings.TrimSpace(expectedFingerprint) {
			return errors.New(errDeployTaskSSHHostKeyMismatch)
		}
		return nil
	}
}

func newDeploySSHRunner(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
	fingerprint := strings.TrimSpace(req.HostFingerprint)
	if fingerprint == "" {
		return nil, errors.New(errDeployTaskSSHHostKeyRequired)
	}
	user := strings.TrimSpace(req.SSHUser)
	if user == "" {
		return nil, errors.New(errDeployTaskSSHUserRequired)
	}
	authMode := strings.TrimSpace(req.AuthMode)
	if authMode == "" {
		authMode = "password"
	}
	config := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: deployHostKeyCallback(fingerprint),
		Timeout:         10 * time.Second,
	}
	switch authMode {
	case "private_key":
		signer, err := ssh.ParsePrivateKey([]byte(req.SSHPrivateKey))
		if err != nil {
			return nil, errors.New(errDeployTaskSSHAuthFailed)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		if strings.TrimSpace(req.SSHPassword) == "" {
			return nil, errors.New(errDeployTaskSSHPasswordRequired)
		}
		config.Auth = []ssh.AuthMethod{ssh.Password(req.SSHPassword)}
	}

	port := host.SSHPort
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.IP, port), config)
	if err != nil {
		if strings.Contains(strings.TrimSpace(err.Error()), errDeployTaskSSHHostKeyMismatch) {
			return nil, errors.New(errDeployTaskSSHHostKeyMismatch)
		}
		return nil, errors.New(errDeployTaskSSHConnectFailed)
	}
	return &deploySSHClient{client: client}, nil
}

func parseUint64JSON(raw datatypes.JSON) []uint64 {
	var ids []uint64
	_ = json.Unmarshal(raw, &ids)
	return common.NormalizeUint64IDs(ids)
}

type deployInstalledComponent struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	DeployedAt     string `json:"deployedAt,omitempty"`
	DeployTaskID   uint64 `json:"deployTaskId,omitempty"`
	DeployTaskName string `json:"deployTaskName,omitempty"`
	ExecutorType   string `json:"executorType,omitempty"`
}

func (s *DeployService) executeSSHTask(task DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, actor string) error {
	if _, err := s.resolveTaskExecutionPlan(task); err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	taskDetail, err := s.GetTask(task.ID, nil)
	if err != nil {
		return err
	}

	taskHostsByID := make(map[uint64]TaskHostResponse, len(taskDetail.Hosts))
	for _, item := range taskDetail.Hosts {
		taskHostsByID[item.HostID] = item
	}

	continueOnHostFailure := taskContinueOnHostFailure(task)
	for _, target := range hosts {
		if err := s.executeTaskHostContext(context.Background(), task, target, taskHostsByID[target.ID], req, actor); err != nil {
			return err
		}
		var completed DeployTaskHost
		if err := s.db.First(&completed, taskHostsByID[target.ID].ID).Error; err != nil {
			return err
		}
		if completed.Status == TaskHostStatusFailed && !continueOnHostFailure {
			if strings.TrimSpace(completed.ErrorMessage) == "" {
				return errors.New("business.deploy.task.hostFailed")
			}
			return errors.New(completed.ErrorMessage)
		}
	}
	return nil
}

// taskHostExecution accumulates stdout/stderr sections, the component summary,
// and the first unrecoverable execution error for a single target host.
type taskHostExecution struct {
	stdoutSections []string
	stderrSections []string
	summary        deployExecutionSummary
	executionErr   error
}

func (e *taskHostExecution) recordOutput(step deployExecutionStep, phase, stdout, stderr string) {
	if strings.TrimSpace(stdout) != "" {
		e.stdoutSections = append(e.stdoutSections, decorateStepPhaseOutput(step, phase, stdout))
	}
	if strings.TrimSpace(stderr) != "" {
		e.stderrSections = append(e.stderrSections, decorateStepPhaseOutput(step, phase, stderr))
	}
}

func (e *taskHostExecution) recordComponent(step deployExecutionStep, task DeployTask) {
	if step.Package.ID == 0 {
		return
	}
	if normalizeTaskAction(step.Action) == TaskActionUninstall {
		e.summary.RemovedComponentNames = append(e.summary.RemovedComponentNames, step.Package.Name)
		return
	}
	e.summary.InstalledComponents = append(e.summary.InstalledComponents, cmdb.InstalledComponentUpsert{
		Name:           step.Package.Name,
		Version:        step.Package.Version,
		DeployedAt:     time.Now(),
		DeployTaskID:   task.ID,
		DeployTaskName: task.Name,
		ExecutorType:   task.ExecutorType,
	})
}

// executeTaskHost executes the full plan against one target host and persists
// the host result, returning a fatal error only when result persistence fails.
func (s *DeployService) executeTaskHostContext(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, req StartTaskRequest, actor string) error {
	if s.executor == nil {
		s.executor = &sshDeployExecutor{service: s}
	}
	return s.executor.ExecuteHost(ctx, DeployExecutionHostRequest{Task: task, TaskHost: taskHost, Credential: req, Actor: actor})
}

type sshStepProvider struct {
	service   *DeployService
	task      DeployTask
	target    cmdbHostSnapshot
	taskHost  TaskHostResponse
	runner    deploySSHRunner
	execution *taskHostExecution
	steps     map[string]deployExecutionStep
}

func (p *sshStepProvider) Type() string { return ExecutorTypeSSH }

func (p *sshStepProvider) Validate(_ context.Context, plan executorprovider.ExecutionPlan) error {
	if strings.TrimSpace(plan.TargetIP) == "" || len(plan.Steps) == 0 {
		return errors.New(errDeployTaskSnapshotMissing)
	}
	return nil
}

func (p *sshStepProvider) ExecuteStep(ctx context.Context, req executorprovider.StepRequest, logSink executorprovider.LogSink) executorprovider.StepResult {
	step, exists := p.steps[req.StepKey]
	if !exists {
		return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: errDeployTaskSnapshotMissing}
	}
	checkpoint, err := p.service.loadTaskStep(p.task.ID, p.taskHost.ID, req.StepKey)
	if err != nil {
		return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: err.Error()}
	}
	if checkpoint.Status == StepStatusSuccess || checkpoint.Status == StepStatusSkipped {
		return executorprovider.StepResult{Status: checkpoint.Status, Changed: checkpoint.Changed}
	}
	if err := p.service.updateTaskStepStatus(&checkpoint, StepStatusRunning, "", ""); err != nil {
		return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: err.Error()}
	}
	p.service.appendTaskEvent(p.task.ID, p.taskHost.ID, checkpoint.ID, "step.running", checkpoint.Status, StepStatusRunning, req.Actor, "", "")

	stdoutStart := len(p.execution.stdoutSections)
	stderrStart := len(p.execution.stderrSections)
	previousExecutionErr := p.execution.executionErr
	stop, changed := p.service.executeTaskStep(ctx, p.task, p.target, p.taskHost, p.runner, step, p.execution)
	writeNewExecutionLogs(logSink, req, p.execution, stdoutStart, stderrStart)
	if !stop {
		checkpoint.Changed = changed
		if err := p.service.updateTaskStepStatus(&checkpoint, StepStatusSuccess, "", ""); err != nil {
			return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: err.Error()}
		}
		p.service.appendTaskEvent(p.task.ID, p.taskHost.ID, checkpoint.ID, "step.success", StepStatusRunning, StepStatusSuccess, req.Actor, "", "")
		return executorprovider.StepResult{Status: StepStatusSuccess, Changed: changed}
	}

	if p.service.taskExecutionSuspended(p.task.ID) {
		p.execution.executionErr = previousExecutionErr
		_ = p.service.updateTaskStepStatus(&checkpoint, StepStatusPending, "", "")
		return executorprovider.StepResult{Status: StepStatusCanceled}
	}
	status := StepStatusFailed
	if p.service.taskCanceled(p.task.ID) || errors.Is(ctx.Err(), context.Canceled) {
		status = StepStatusCanceled
	}
	errorMessage := combineTaskExecutionErrors(p.execution.executionErr, nil)
	errorMessage = redactDeploySecrets(errorMessage, p.service.renderSecrets(p.task.ID))
	if err := p.service.updateTaskStepStatus(&checkpoint, status, "", errorMessage); err != nil {
		return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: err.Error()}
	}
	p.service.appendTaskEvent(p.task.ID, p.taskHost.ID, checkpoint.ID, "step."+status, StepStatusRunning, status, req.Actor, "", truncateDeployLog(errorMessage, 256))
	return executorprovider.StepResult{Status: status, ErrorMessage: errorMessage}
}

func (p *sshStepProvider) Cancel(_ context.Context, handle executorprovider.ExecutionHandle) error {
	if handle.CancelFunc != nil {
		handle.CancelFunc()
	}
	return nil
}

func writeNewExecutionLogs(logSink executorprovider.LogSink, req executorprovider.StepRequest, execution *taskHostExecution, stdoutStart, stderrStart int) {
	if logSink == nil {
		return
	}
	if stdoutStart < len(execution.stdoutSections) {
		logSink.WriteLog(req.TaskID, req.TaskHostID, req.StepKey, "stdout", strings.Join(execution.stdoutSections[stdoutStart:], "\n"))
	}
	if stderrStart < len(execution.stderrSections) {
		logSink.WriteLog(req.TaskID, req.TaskHostID, req.StepKey, "stderr", strings.Join(execution.stderrSections[stderrStart:], "\n"))
	}
}

func (s *DeployService) loadTaskStep(taskID, taskHostID uint64, stepKey string) (DeployTaskStep, error) {
	var checkpoint DeployTaskStep
	err := s.db.Where("task_id = ? AND task_host_id = ? AND step_key = ?", taskID, taskHostID, stepKey).First(&checkpoint).Error
	return checkpoint, err
}

func (s *DeployService) updateTaskStepStatus(checkpoint *DeployTaskStep, status, errorCode, errorMessage string) error {
	now := time.Now()
	updates := map[string]any{"status": status, "error_code": errorCode, "error_message": truncateDeployLog(errorMessage, 1024), "changed": checkpoint.Changed, "updated_at": now, "state_version": gorm.Expr("state_version + 1")}
	if status == StepStatusRunning {
		updates["started_at"] = &now
		updates["finished_at"] = nil
		updates["attempt_no"] = gorm.Expr("attempt_no + 1")
	} else if status == StepStatusSuccess || status == StepStatusFailed || status == StepStatusSkipped || status == StepStatusCanceled {
		updates["finished_at"] = &now
	}
	err := s.db.Model(&DeployTaskStep{}).Where("id = ?", checkpoint.ID).Updates(updates).Error
	if err == nil {
		// 更新内存中的 checkpoint 状态以供 metrics 使用
		checkpoint.Status = status
		// 记录步骤结果 metrics
		recordStepResult(checkpoint)
	}
	return err
}

func (s *DeployService) executeOrchestratedSSHHost(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, req StartTaskRequest, actor string) error {
	steps, err := s.resolveTaskExecutionPlan(task)
	if err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	runner, runnerErr := s.sshRunnerFactory(target, req)
	if runnerErr != nil {
		if _, err := s.MarkHostResult(taskHost.ID, MarkHostResultRequest{Status: TaskHostStatusFailed, ErrorMessage: runnerErr.Error(), ExecutorID: fmt.Sprintf("ssh:%s", target.IP)}, actor, nil); err != nil {
			return err
		}
		return nil
	}
	execution := &taskHostExecution{stdoutSections: make([]string, 0, len(steps)), stderrSections: make([]string, 0, len(steps)), summary: deployExecutionSummary{InstalledComponents: make([]cmdb.InstalledComponentUpsert, 0, len(steps)), RemovedComponentNames: make([]string, 0, len(steps))}}
	_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{{"at": time.Now().Format(time.RFC3339), "phase": "connect", "message": "SSH connected"}})
	provider := &sshStepProvider{service: s, task: task, target: target, taskHost: taskHost, runner: runner, execution: execution, steps: make(map[string]deployExecutionStep, len(steps))}
	plan := executorprovider.ExecutionPlan{TaskID: task.ID, TaskHostID: taskHost.ID, TaskName: task.Name, Action: task.Action, ExecutorType: ExecutorTypeSSH, TargetIP: target.IP, TargetPort: target.SSHPort, TargetOS: target.OS, MaxParallelSteps: 1, Actor: actor}
	for index, step := range steps {
		stepKey := step.StepCode
		if stepKey == "" {
			stepKey = fmt.Sprintf("step-%d", index+1)
			step.StepCode = stepKey
		}
		provider.steps[stepKey] = step
		checkpoint, loadErr := s.loadTaskStep(task.ID, taskHost.ID, stepKey)
		if loadErr != nil {
			_ = runner.Close()
			return loadErr
		}
		plan.Steps = append(plan.Steps, executorprovider.StepPlan{StepKey: stepKey, StepName: step.StepName, StepType: step.StepType, Action: step.Action, ContinueOnHostFailure: step.ContinueOnHostFailure, PackageName: step.Package.Name, PackageVersion: step.Package.Version, TemplateParams: step.TemplateParams, Dependencies: step.Dependencies, Condition: conditionType(step.Condition), ConditionParams: step.Condition, FailurePolicy: step.FailurePolicy, InitialStatus: checkpoint.Status})
	}
	registry := executorprovider.NewRegistry()
	registry.Register(provider)
	result := deployorchestrator.New(registry).RunHostSteps(ctx, plan, s)
	if err := s.persistSkippedTaskSteps(task, taskHost, result, actor); err != nil {
		_ = runner.Close()
		return err
	}
	if result.Status == deployorchestrator.StatusCanceled && s.taskExecutionSuspended(task.ID) {
		_ = runner.Close()
		now := time.Now()
		transition := s.db.Model(&DeployTask{}).Where("id = ? AND status = ?", task.ID, TaskStatusPauseRequested).Updates(map[string]any{"status": TaskStatusPaused, "updated_by": actor, "updated_at": now})
		if transition.Error != nil {
			return transition.Error
		}
		if transition.RowsAffected > 0 {
			s.appendTaskEvent(task.ID, 0, 0, "task.paused", TaskStatusPauseRequested, TaskStatusPaused, actor, "", "")
		}
		return nil
	}
	if result.Status == deployorchestrator.StatusCanceled && s.taskCancellationRequested(task.ID) {
		_ = runner.Close()
		return nil
	}
	if result.Status == deployorchestrator.StatusFailed && execution.executionErr == nil {
		execution.executionErr = orchestrationExecutionError(result)
	}
	if result.Status == deployorchestrator.StatusCanceled && execution.executionErr == nil {
		execution.executionErr = ctx.Err()
		if execution.executionErr == nil {
			execution.executionErr = context.Canceled
		}
	}
	return s.finalizeTaskHostExecution(task, taskHost, runner, execution, target.IP, actor)
}

func (s *DeployService) persistSkippedTaskSteps(task DeployTask, taskHost TaskHostResponse, result deployorchestrator.HostExecution, actor string) error {
	for _, step := range result.Steps {
		if step.Status != StepStatusSkipped {
			continue
		}
		checkpoint, err := s.loadTaskStep(task.ID, taskHost.ID, step.StepKey)
		if err != nil {
			return err
		}
		if checkpoint.Status == StepStatusSkipped {
			continue
		}
		if err := s.updateTaskStepStatus(&checkpoint, StepStatusSkipped, "", ""); err != nil {
			return err
		}
		s.appendTaskEvent(task.ID, taskHost.ID, checkpoint.ID, "step.skipped", checkpoint.Status, StepStatusSkipped, actor, "", "condition evaluated false")
	}
	return nil
}

func orchestrationExecutionError(result deployorchestrator.HostExecution) error {
	for _, step := range result.Steps {
		if step.Status == deployorchestrator.StatusFailed && strings.TrimSpace(step.Error) != "" {
			return errors.New(step.Error)
		}
	}
	return errors.New("business.deploy.task.orchestrationFailed")
}

func conditionType(condition map[string]any) string {
	conditionType, _ := condition["type"].(string)
	return strings.TrimSpace(conditionType)
}

func (s *DeployService) executeSSHHost(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, req StartTaskRequest, actor string) error {
	plan, err := s.resolveTaskExecutionPlan(task)
	if err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	runner, runnerErr := s.sshRunnerFactory(target, req)
	if runnerErr != nil {
		if _, err := s.MarkHostResult(taskHost.ID, MarkHostResultRequest{
			Status:       TaskHostStatusFailed,
			ErrorMessage: runnerErr.Error(),
			ExecutorID:   fmt.Sprintf("ssh:%s", target.IP),
		}, actor, nil); err != nil {
			return err
		}
		return nil
	}

	_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
		{"at": time.Now().Format(time.RFC3339), "phase": "connect", "message": "SSH connected"},
	})
	execution := &taskHostExecution{
		stdoutSections: make([]string, 0, len(plan)),
		stderrSections: make([]string, 0, len(plan)),
		summary: deployExecutionSummary{
			InstalledComponents:   make([]cmdb.InstalledComponentUpsert, 0, len(plan)),
			RemovedComponentNames: make([]string, 0, len(plan)),
		},
	}
	s.runTaskExecutionSteps(ctx, task, target, taskHost, runner, plan, execution)
	return s.finalizeTaskHostExecution(task, taskHost, runner, execution, target.IP, actor)
}

// runTaskExecutionSteps runs the plan steps for one host, stopping at the first
// unrecoverable step failure.
func (s *DeployService) runTaskExecutionSteps(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, plan []deployExecutionStep, execution *taskHostExecution) {
	for _, step := range plan {
		stop, _ := s.executeTaskStep(ctx, task, target, taskHost, runner, step, execution)
		if stop {
			break
		}
	}
}

// executeTaskStep runs a single plan step and records its trace. It returns
// true when the host loop must stop on an unrecoverable failure.
func (s *DeployService) executeTaskStep(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, step deployExecutionStep, execution *taskHostExecution) (bool, bool) {
	stepLabel := buildDeployStepLabel(step)
	var script string
	var probeCommand string
	if isComponentType(step.StepType) {
		plan, err := s.resolveComponentPlan(step, task, target)
		if err != nil {
			execution.executionErr = mapDeployTaskExecutionPlanError(err)
			s.appendStepTrace(task, taskHost, step, "render_failed", err.Error())
			return true, false
		}
		if normalizeTaskAction(step.Action) == TaskActionUninstall {
			script = plan.UninstallScript
			probeCommand = plan.UninstallHealthCheckCommand
		} else {
			script = plan.InstallScript
			probeCommand = plan.HealthCheckCommand
		}
		if strings.TrimSpace(probeCommand) != "" {
			s.appendStepTrace(task, taskHost, step, "healthcheck", fmt.Sprintf("%s healthcheck started", stepLabel))
			stdout, stderr, probeErr := runDeployScript(ctx, runner, probeCommand)
			execution.recordOutput(step, "healthcheck", stdout, stderr)
			if probeErr == nil {
				// Already in the desired state → idempotent ok, no change.
				s.appendStepTrace(task, taskHost, step, "step_unchanged", fmt.Sprintf("%s already in desired state", stepLabel))
				execution.recordComponent(step, task)
				return false, false
			}
		}
	} else {
		rendered, renderErr := s.renderExecutionStepScript(step, task, target)
		if renderErr != nil {
			execution.executionErr = mapDeployTaskExecutionPlanError(renderErr)
			s.appendStepTrace(task, taskHost, step, "render_failed", renderErr.Error())
			return true, false
		}
		script = rendered
	}
	s.appendStepTrace(task, taskHost, step, "step_start", fmt.Sprintf("%s started", stepLabel))
	if s.runTaskStepCheck(ctx, task, target, taskHost, runner, step, execution, taskStepCheckConfig{configKey: "precheckCommand", phase: "precheck", stepLabel: stepLabel}) {
		return true, false
	}
	s.appendStepTrace(task, taskHost, step, "script", fmt.Sprintf("%s script rendered", stepLabel))
	stdout, stderr, execErr := runDeployScript(ctx, runner, script)
	execution.recordOutput(step, "script", stdout, stderr)
	if execErr != nil {
		execution.executionErr = execErr
		s.appendStepTrace(task, taskHost, step, "step_failed", execErr.Error())
		return true, false
	}
	if s.runTaskStepCheck(ctx, task, target, taskHost, runner, step, execution, taskStepCheckConfig{configKey: "postcheckCommand", phase: "postcheck", stepLabel: stepLabel}) {
		return true, false
	}
	s.appendStepTrace(task, taskHost, step, "step_success", fmt.Sprintf("%s completed", stepLabel))
	execution.recordComponent(step, task)
	return false, true
}

// taskStepCheckConfig bundles the per-check rendering parameters shared by the
// precheck and postcheck passes of a host step.
type taskStepCheckConfig struct {
	configKey string
	phase     string
	stepLabel string
}

// runTaskStepCheck renders and runs an optional precheck/postcheck snippet,
// returning true when the host loop must stop.
func (s *DeployService) runTaskStepCheck(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, step deployExecutionStep, execution *taskHostExecution, check taskStepCheckConfig) bool {
	script, hasCheck, checkErr := renderDeployCheckSnippet(step, task, target, check.configKey)
	if checkErr != nil {
		execution.executionErr = mapDeployTaskExecutionPlanError(checkErr)
		s.appendStepTrace(task, taskHost, step, check.phase+"_render_failed", checkErr.Error())
		return true
	}
	if !hasCheck {
		return false
	}
	s.appendStepTrace(task, taskHost, step, check.phase, fmt.Sprintf("%s %s started", check.stepLabel, check.phase))
	stdout, stderr, execErr := runDeployScript(ctx, runner, script)
	execution.recordOutput(step, check.phase, stdout, stderr)
	if execErr != nil {
		execution.executionErr = fmt.Errorf("%s %s failed: %w", check.stepLabel, check.phase, execErr)
		s.appendStepTrace(task, taskHost, step, "step_failed", execution.executionErr.Error())
		return true
	}
	return false
}

// appendStepTrace records a per-step task-host trace entry with a common shape.
func (s *DeployService) appendStepTrace(task DeployTask, taskHost TaskHostResponse, step deployExecutionStep, phase, message string) {
	_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
		{
			"at":          time.Now().Format(time.RFC3339),
			"phase":       phase,
			"stepCode":    step.StepCode,
			"stepName":    step.StepName,
			"stepType":    step.StepType,
			"action":      step.Action,
			"packageName": step.Package.Name,
			"message":     message,
		},
	})
}

// finalizeTaskHostExecution closes the runner and persists the host result.
func (s *DeployService) finalizeTaskHostExecution(task DeployTask, taskHost TaskHostResponse, runner deploySSHRunner, execution *taskHostExecution, hostIP, actor string) error {
	closeErr := runner.Close()
	combinedStdout := strings.TrimSpace(strings.Join(execution.stdoutSections, "\n\n"))
	combinedStderr := strings.TrimSpace(strings.Join(execution.stderrSections, "\n\n"))
	// FZ-02: central redaction — output and error text never contain secret
	// material from the task secret store or the resolved SSH credential.
	secrets := s.renderSecrets(task.ID)
	combinedStdout = redactDeploySecrets(combinedStdout, secrets)
	combinedStderr = redactDeploySecrets(combinedStderr, secrets)
	errorMessage := redactDeploySecrets(combineTaskExecutionErrors(execution.executionErr, closeErr), secrets)
	if errorMessage != "" {
		if _, err := s.markHostResultWithSummary(taskHost.ID, MarkHostResultRequest{
			Status:       TaskHostStatusFailed,
			Stdout:       combinedStdout,
			Stderr:       combinedStderr,
			ErrorMessage: errorMessage,
			ExecutorID:   fmt.Sprintf("ssh:%s", hostIP),
		}, actor, nil, deployExecutionSummary{}); err != nil {
			return err
		}
		_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
			{"at": time.Now().Format(time.RFC3339), "phase": "failed", "message": errorMessage},
		})
		return nil
	}
	if _, err := s.markHostResultWithSummary(taskHost.ID, MarkHostResultRequest{
		Status:     TaskHostStatusSuccess,
		Stdout:     combinedStdout,
		Stderr:     combinedStderr,
		ExecutorID: fmt.Sprintf("ssh:%s", hostIP),
	}, actor, nil, execution.summary); err != nil {
		return err
	}
	_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
		{"at": time.Now().Format(time.RFC3339), "phase": "writeback", "message": fmt.Sprintf("Host marked %s", hostStatusForAction(task.Action))},
	})
	return nil
}

// combineTaskExecutionErrors merges the step execution error and runner close
// error into a single message (empty when both are nil).
func combineTaskExecutionErrors(executionErr, closeErr error) string {
	switch {
	case executionErr == nil && closeErr == nil:
		return ""
	case executionErr != nil && closeErr != nil:
		return fmt.Sprintf("%s; close: %s", executionErr.Error(), closeErr.Error())
	case executionErr != nil:
		return executionErr.Error()
	default:
		return closeErr.Error()
	}
}

// resolveTaskExecutionPlan reconstructs the execution steps from the frozen
// execution snapshot. It never reads live package/template rows, so a task's
// historical execution intent cannot drift after Start.
func (s *DeployService) resolveTaskExecutionPlan(task DeployTask) ([]deployExecutionStep, error) {
	snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
	if err != nil {
		return nil, err
	}
	if len(snapshot.Steps) == 0 {
		return nil, errors.New(errDeployTaskSnapshotMissing)
	}
	return executionSnapshotToPlan(snapshot), nil
}

// buildDeployExecutionSnapshot resolves the live package/template definition and
// serializes it as an immutable execution snapshot. It is called only at task
// creation and draft/pending update, never during execution.
func (s *DeployService) buildDeployExecutionSnapshot(pkg DeployPackage, template *TemplateResponse, action string, taskParams map[string]any) (deployExecutionSnapshot, error) {
	snapshot := deployExecutionSnapshot{}
	if template != nil {
		snapshot.TemplateID = template.ID
		snapshot.TemplateName = template.Name
		snapshot.TemplateVersion = template.Version
	}
	normalizedAction := normalizeTaskAction(action)
	if template == nil || template.ID == 0 {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, taskParams); err != nil {
			return deployExecutionSnapshot{}, err
		}
		snapshot.Steps = []deployStepSnapshot{{
			StepCode:       "package_default",
			StepName:       pkg.Name,
			StepType:       TemplateStepTypePackage,
			Action:         normalizedAction,
			Package:        packageToSnapshot(pkg),
			TemplateParams: taskParams,
		}}
		return snapshot, nil
	}
	if len(template.Steps) == 0 {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, taskParams); err != nil {
			return deployExecutionSnapshot{}, err
		}
		snapshot.Steps = []deployStepSnapshot{{
			StepCode:       "template_default",
			StepName:       template.Name,
			StepType:       TemplateStepTypePackage,
			Action:         normalizedAction,
			Package:        packageToSnapshot(pkg),
			TemplateParams: taskParams,
		}}
		return snapshot, nil
	}
	cache := map[uint64]DeployPackage{}
	steps := make([]deployStepSnapshot, 0, len(template.Steps))
	for _, step := range template.Steps {
		stepType := strings.TrimSpace(step.StepType)
		if stepType == "" {
			stepType = TemplateStepTypePackage
		}
		if !validStepType(stepType) {
			return deployExecutionSnapshot{}, errors.New(errDeployTaskTemplateInvalid)
		}
		effectiveAction := normalizedAction
		if effectiveAction == "" {
			effectiveAction = normalizeTaskAction(step.Action)
		}
		params := mergeDeployTemplateParams(step.TemplateParams, taskParams)
		stepPackage := DeployPackage{}
		if stepType == TemplateStepTypePackage {
			resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, pkg, cache)
			if err != nil {
				return deployExecutionSnapshot{}, err
			}
			cache[resolvedPackage.ID] = resolvedPackage
			stepPackage = resolvedPackage
			if err := validateTemplateParams(stepPackage.ExecutionMode, stepPackage.TemplateCode, stepPackage.TemplateConfig, params); err != nil {
				return deployExecutionSnapshot{}, err
			}
		} else if isComponentType(stepType) {
			resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, pkg, cache)
			if err != nil {
				return deployExecutionSnapshot{}, err
			}
			cache[resolvedPackage.ID] = resolvedPackage
			stepPackage = resolvedPackage
			componentParams := cloneDeployConfig(params)
			componentParams["action"] = effectiveAction
			if err := validateFixedTemplateParams(componentTypeToTemplateCode[stepType], componentParams); err != nil {
				return deployExecutionSnapshot{}, err
			}
		} else if step.PackageID > 0 {
			resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, pkg, cache)
			if err != nil {
				return deployExecutionSnapshot{}, err
			}
			cache[resolvedPackage.ID] = resolvedPackage
			stepPackage = resolvedPackage
		}
		steps = append(steps, deployStepSnapshot{
			StepCode:              step.StepCode,
			StepName:              step.StepName,
			StepType:              stepType,
			Action:                effectiveAction,
			ContinueOnHostFailure: step.ContinueOnHostFailure,
			Package:               packageToSnapshot(stepPackage),
			TemplateParams:        params,
			StepConfig:            step.StepConfig,
			Dependencies:          stringSliceConfig(step.StepConfig, "dependencies"),
			Condition:             mapConfig(step.StepConfig, "condition"),
			FailurePolicy:         effectiveFailurePolicy(stringConfig(step.StepConfig, "failurePolicy"), step.ContinueOnHostFailure),
		})
	}
	if err := validateDeploySnapshotDAG(steps); err != nil {
		return deployExecutionSnapshot{}, err
	}
	snapshot.Steps = steps
	return snapshot, nil
}

func validateDeploySnapshotDAG(steps []deployStepSnapshot) error {
	nodes := make([]deployorchestrator.DAGNode, 0, len(steps))
	for index, step := range steps {
		stepKey := strings.TrimSpace(step.StepCode)
		if stepKey == "" {
			stepKey = fmt.Sprintf("step-%d", index+1)
		}
		condition := conditionType(step.Condition)
		switch condition {
		case "", deployorchestrator.ConditionAlways, deployorchestrator.ConditionOnSuccess, deployorchestrator.ConditionOnFailure, deployorchestrator.ConditionParamEquals, deployorchestrator.ConditionOnChanged:
		default:
			return errors.New(errDeployTaskTemplateInvalid)
		}
		failurePolicy := strings.TrimSpace(step.FailurePolicy)
		switch failurePolicy {
		case "", FailurePolicyFailFast, FailurePolicyContinueHosts, FailurePolicyContinueSteps:
		default:
			return errors.New(errDeployTaskTemplateInvalid)
		}
		nodes = append(nodes, deployorchestrator.DAGNode{StepKey: stepKey, Dependencies: step.Dependencies, Condition: condition, FailurePolicy: failurePolicy, StepOrder: index})
	}
	validation := deployorchestrator.ValidateDAG(nodes)
	if !validation.Valid {
		return fmt.Errorf("%s: %s", errDeployTaskTemplateInvalid, strings.Join(validation.Errors, "; "))
	}
	return nil
}

func stringConfig(config map[string]any, key string) string {
	value, _ := config[key].(string)
	return strings.TrimSpace(value)
}

func effectiveFailurePolicy(policy string, continueOnHostFailure bool) string {
	policy = strings.TrimSpace(policy)
	if continueOnHostFailure {
		return FailurePolicyContinueHosts
	}
	return policy
}

func stringSliceConfig(config map[string]any, key string) []string {
	values, ok := config[key].([]any)
	if !ok {
		if typed, typedOK := config[key].([]string); typedOK {
			return append([]string(nil), typed...)
		}
		return nil
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if text, textOK := value.(string); textOK && strings.TrimSpace(text) != "" {
			result = append(result, strings.TrimSpace(text))
		}
	}
	return result
}

func mapConfig(config map[string]any, key string) map[string]any {
	value, _ := config[key].(map[string]any)
	return value
}

func decodeExecutionSnapshot(raw datatypes.JSON) (deployExecutionSnapshot, error) {
	if len(raw) == 0 {
		return deployExecutionSnapshot{}, nil
	}
	var snapshot deployExecutionSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return deployExecutionSnapshot{}, err
	}
	return snapshot, nil
}

func executionSnapshotToPlan(snapshot deployExecutionSnapshot) []deployExecutionStep {
	plan := make([]deployExecutionStep, 0, len(snapshot.Steps))
	for index, step := range snapshot.Steps {
		plan = append(plan, deployExecutionStep{
			Index:                 index,
			StepCode:              step.StepCode,
			StepName:              step.StepName,
			StepType:              step.StepType,
			Action:                step.Action,
			ContinueOnHostFailure: step.ContinueOnHostFailure,
			Package:               snapshotToPackage(step.Package),
			TemplateParams:        step.TemplateParams,
			StepConfig:            step.StepConfig,
			Dependencies:          append([]string(nil), step.Dependencies...),
			Condition:             step.Condition,
			FailurePolicy:         step.FailurePolicy,
		})
	}
	return plan
}

func packageToSnapshot(pkg DeployPackage) deployPackageSnapshot {
	return deployPackageSnapshot{
		ID:               pkg.ID,
		Name:             pkg.Name,
		Version:          pkg.Version,
		InstallCommand:   pkg.InstallCommand,
		UninstallCommand: pkg.UninstallCommand,
		ExecutionMode:    pkg.ExecutionMode,
		TemplateCode:     pkg.TemplateCode,
		TemplateConfig:   decodeJSONMap(pkg.TemplateConfig),
		SourceObjectKey:  pkg.SourceObjectKey,
		SourceFileName:   pkg.SourceFileName,
		SourceURL:        pkg.SourceURL,
	}
}

func snapshotToPackage(snapshot deployPackageSnapshot) DeployPackage {
	templateConfigJSON, _ := json.Marshal(snapshot.TemplateConfig)
	return DeployPackage{
		ID:               snapshot.ID,
		Name:             snapshot.Name,
		Version:          snapshot.Version,
		InstallCommand:   snapshot.InstallCommand,
		UninstallCommand: snapshot.UninstallCommand,
		ExecutionMode:    snapshot.ExecutionMode,
		TemplateCode:     snapshot.TemplateCode,
		TemplateConfig:   datatypes.JSON(templateConfigJSON),
		SourceObjectKey:  snapshot.SourceObjectKey,
		SourceFileName:   snapshot.SourceFileName,
		SourceURL:        snapshot.SourceURL,
	}
}

func (s *DeployService) resolveExecutionPackage(packageID uint64, fallbackPackage DeployPackage, cache map[uint64]DeployPackage) (DeployPackage, error) {
	if packageID == 0 {
		if fallbackPackage.ID == 0 {
			return DeployPackage{}, errors.New("deploypackage.not_found")
		}
		if fallbackPackage.Status != PackageStatusEnabled {
			return DeployPackage{}, errors.New("deploypackage.disabled")
		}
		return fallbackPackage, nil
	}
	if cached, ok := cache[packageID]; ok {
		if cached.Status != PackageStatusEnabled {
			return DeployPackage{}, errors.New("deploypackage.disabled")
		}
		return cached, nil
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, packageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployPackage{}, errors.New("deploypackage.not_found")
		}
		return DeployPackage{}, err
	}
	if pkg.Status != PackageStatusEnabled {
		return DeployPackage{}, errors.New("deploypackage.disabled")
	}
	return pkg, nil
}

func mergeDeployTemplateParams(base, override map[string]any) map[string]any {
	result := make(map[string]any)
	for key, value := range base {
		result[key] = value
	}
	for key, value := range override {
		result[key] = value
	}
	return result
}

func (s *DeployService) renderExecutionStepScript(step deployExecutionStep, task DeployTask, host cmdbHostSnapshot) (string, error) {
	if step.StepType == TemplateStepTypeScript {
		script := readDeployConfigString(step.StepConfig, "script")
		if platformScripts := deployPlatformScripts(step.StepConfig); len(platformScripts) > 0 {
			if selected := selectDeployPlatformScript(platformScripts, host.OS); selected != "" {
				script = selected
			}
		}
		if script == "" {
			return "", errors.New(errDeployTaskTemplateInvalid)
		}
		return renderTemplateSnippet(script, buildDeployRenderContext(step, task, host))
	}
	if isComponentType(step.StepType) {
		plan, err := s.resolveComponentPlan(step, task, host)
		if err != nil {
			return "", err
		}
		if normalizeTaskAction(step.Action) == TaskActionUninstall {
			return plan.UninstallScript, nil
		}
		return plan.InstallScript, nil
	}
	if step.Package.ID == 0 {
		return "", errors.New(errDeployTaskTemplateInvalid)
	}
	// FZ-02: resolve secret reference sentinels from the encrypted store into
	// the in-memory render view. The resolved values never leave this process.
	renderParams := resolveStepParamsSecretSentinels(step.TemplateParams, s.renderSecrets(task.ID))
	paramsJSON, _ := json.Marshal(renderParams)
	stepTask := task
	stepTask.Action = step.Action
	stepTask.PackageID = step.Package.ID
	stepTask.PackageName = step.Package.Name
	stepTask.PackageVersion = step.Package.Version
	stepTask.TemplateParams = datatypes.JSON(paramsJSON)
	return s.resolveInstallScript(step.Package, stepTask)
}

func deployPlatformScripts(config map[string]any) map[string]string {
	value, ok := config["platformScripts"]
	if !ok {
		return nil
	}
	result := map[string]string{}
	if values, ok := value.(map[string]any); ok {
		for key, raw := range values {
			if text := strings.TrimSpace(anyToString(raw)); text != "" {
				result[strings.ToLower(strings.TrimSpace(key))] = text
			}
		}
	}
	return result
}

func selectDeployPlatformScript(scripts map[string]string, hostOS string) string {
	os := strings.ToLower(strings.TrimSpace(hostOS))
	for _, key := range []string{os, "linux"} {
		if value := strings.TrimSpace(scripts[key]); value != "" {
			return value
		}
	}
	for _, family := range []string{"ubuntu", "debian", "redhat", "centos", "rhel", "alpine", "windows"} {
		if strings.Contains(os, family) {
			if value := strings.TrimSpace(scripts[family]); value != "" {
				return value
			}
		}
	}
	return ""
}

// renderSecrets lazily loads the decrypted secret parameters for a task for
// render-time use. Load failures degrade to an empty map so rendering fails on
// the sentinel validation instead of leaking unhandled errors.
func (s *DeployService) renderSecrets(taskID uint64) map[string]string {
	secrets, err := s.loadDeployTaskSecrets(taskID)
	if err != nil {
		return map[string]string{}
	}
	return secrets
}

func renderTemplateSnippet(template string, context map[string]string) (string, error) {
	pattern := regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)
	var renderErr error
	rendered := pattern.ReplaceAllStringFunc(template, func(token string) string {
		if renderErr != nil {
			return token
		}
		match := pattern.FindStringSubmatch(token)
		if len(match) != 2 {
			return token
		}
		key := strings.TrimSpace(match[1])
		value, ok := context[key]
		if !ok {
			renderErr = errors.New(errDeployTaskTemplateParamsInvalid)
			return token
		}
		return value
	})
	if renderErr != nil {
		return "", renderErr
	}
	return strings.TrimSpace(rendered), nil
}

func buildDeployRenderContext(step deployExecutionStep, task DeployTask, host cmdbHostSnapshot) map[string]string {
	context := map[string]string{}
	for key, value := range step.TemplateParams {
		context[key] = strings.TrimSpace(anyToString(value))
	}
	context["action"] = normalizeTaskAction(step.Action)
	context["stepCode"] = strings.TrimSpace(step.StepCode)
	context["stepName"] = strings.TrimSpace(step.StepName)
	context["stepType"] = strings.TrimSpace(step.StepType)
	context["taskName"] = strings.TrimSpace(task.Name)
	context["taskAction"] = normalizeTaskAction(task.Action)
	context["businessScopeName"] = strings.TrimSpace(task.BusinessScopeName)
	context["hostName"] = strings.TrimSpace(host.Hostname)
	context["hostIp"] = strings.TrimSpace(host.IP)
	context["hostOs"] = strings.TrimSpace(host.OS)
	context["hostStatus"] = strings.TrimSpace(host.Status)
	context["packageName"] = strings.TrimSpace(step.Package.Name)
	context["packageVersion"] = strings.TrimSpace(step.Package.Version)
	context["packageSourceUrl"] = strings.TrimSpace(step.Package.SourceURL)
	context["packageSourceFileName"] = strings.TrimSpace(step.Package.SourceFileName)
	context["task.name"] = context["taskName"]
	context["task.action"] = context["taskAction"]
	context["step.code"] = context["stepCode"]
	context["step.name"] = context["stepName"]
	context["step.type"] = context["stepType"]
	context["host.name"] = context["hostName"]
	context["host.ip"] = context["hostIp"]
	context["host.os"] = context["hostOs"]
	context["host.status"] = context["hostStatus"]
	context["package.name"] = context["packageName"]
	context["package.version"] = context["packageVersion"]
	context["package.sourceUrl"] = context["packageSourceUrl"]
	context["package.sourceFileName"] = context["packageSourceFileName"]
	return context
}

func renderDeployCheckSnippet(step deployExecutionStep, task DeployTask, host cmdbHostSnapshot, key string) (string, bool, error) {
	command := readDeployConfigString(step.StepConfig, key)
	if command == "" {
		return "", false, nil
	}
	rendered, err := renderTemplateSnippet(command, buildDeployRenderContext(step, task, host))
	if err != nil {
		return "", false, err
	}
	if rendered == "" {
		return "", false, nil
	}
	return "set -e\n" + rendered + "\n", true, nil
}

func readDeployConfigString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(anyToString(value))
}

func buildDeployStepLabel(step deployExecutionStep) string {
	name := strings.TrimSpace(step.StepName)
	if name == "" {
		name = strings.TrimSpace(step.Package.Name)
	}
	if name == "" {
		name = strings.TrimSpace(step.StepCode)
	}
	return fmt.Sprintf("[%d] %s (%s)", step.Index+1, name, normalizeTaskAction(step.Action))
}

func decorateStepOutput(step deployExecutionStep, content string) string {
	return fmt.Sprintf("=== %s ===\n%s", buildDeployStepLabel(step), strings.TrimSpace(content))
}

func decorateStepPhaseOutput(step deployExecutionStep, phase string, content string) string {
	phaseLabel := strings.TrimSpace(phase)
	if phaseLabel == "" {
		return decorateStepOutput(step, content)
	}
	return fmt.Sprintf("=== %s / %s ===\n%s", buildDeployStepLabel(step), phaseLabel, strings.TrimSpace(content))
}

func normalizeDeployStringValues(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

type deployPackageRef struct {
	name    string
	version string
}

func (s *DeployService) upsertHostInstalledComponent(hostID, taskID uint64, taskName, executorType string, pkg deployPackageRef, actor string, now time.Time) error {
	if strings.TrimSpace(pkg.name) == "" {
		return nil
	}
	return s.cmdbCapability.WriteDeployHostResult(cmdb.DeployHostWritebackRequest{
		HostID: hostID,
		Actor:  actor,
		InstalledComponents: []cmdb.InstalledComponentUpsert{{
			Name:           pkg.name,
			Version:        pkg.version,
			DeployedAt:     now,
			DeployTaskID:   taskID,
			DeployTaskName: taskName,
			ExecutorType:   executorType,
		}},
	})
}

type packageDeploymentStat struct {
	LatestDeployedAt   time.Time
	LatestTaskID       uint64
	LatestTaskName     string
	LatestTaskStatus   string
	LatestHostCount    int
	LatestSuccessCount int
}

func packageToResponse(item *DeployPackage, stat packageDeploymentStat) PackageResponse {
	var latestDeployedAt *time.Time
	if !stat.LatestDeployedAt.IsZero() {
		value := stat.LatestDeployedAt
		latestDeployedAt = &value
	}
	return PackageResponse{
		ID:                 item.ID,
		Name:               item.Name,
		Version:            item.Version,
		Description:        item.Description,
		InstallCommand:     item.InstallCommand,
		UninstallCommand:   item.UninstallCommand,
		ExecutionMode:      item.ExecutionMode,
		TemplateCode:       item.TemplateCode,
		TemplateConfig:     decodeJSONMap(item.TemplateConfig),
		SourceObjectKey:    item.SourceObjectKey,
		SourceFileName:     item.SourceFileName,
		SourceURL:          item.SourceURL,
		Status:             item.Status,
		LatestDeployedAt:   latestDeployedAt,
		LatestTaskID:       stat.LatestTaskID,
		LatestTaskName:     stat.LatestTaskName,
		LatestTaskStatus:   stat.LatestTaskStatus,
		LatestHostCount:    stat.LatestHostCount,
		LatestSuccessCount: stat.LatestSuccessCount,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
		CreatedBy:          item.CreatedBy,
		UpdatedBy:          item.UpdatedBy,
	}
}

func taskToResponse(task *DeployTask, hosts []TaskHostResponse) TaskResponse {
	hostCount := len(hosts)
	successCount := 0
	failedCount := 0
	runningCount := 0
	skippedCount := 0
	for _, host := range hosts {
		switch host.Status {
		case TaskHostStatusSuccess:
			successCount++
		case TaskHostStatusFailed:
			failedCount++
		case TaskHostStatusRunning, TaskHostStatusPending:
			runningCount++
		case TaskHostStatusSkipped:
			skippedCount++
		}
	}
	durationSeconds := computeDurationSeconds(task.StartedAt, task.FinishedAt)

	// 计算是否可以回滚
	canRollback := (task.Status == TaskStatusFailed || task.Status == TaskStatusCanceled) &&
		task.SourceTaskID == nil &&
		DetermineRollbackAction(task.Action) != ""

	return TaskResponse{
		ID:                   task.ID,
		Name:                 task.Name,
		TemplateID:           task.TemplateID,
		TemplateName:         task.TemplateName,
		TemplateVersion:      task.TemplateVersion,
		PackageID:            task.PackageID,
		PackageName:          task.PackageName,
		PackageVersion:       task.PackageVersion,
		BusinessScopeID:      task.BusinessScopeID,
		BusinessScopeName:    task.BusinessScopeName,
		ServiceID:            task.ServiceID,
		ServiceInstanceID:    task.ServiceInstanceID,
		ServiceName:          task.ServiceName,
		ServiceInstanceName:  task.ServiceInstanceName,
		Action:               normalizeTaskAction(task.Action),
		TargetType:           task.TargetType,
		TargetIDs:            parseUint64JSON(task.TargetIDs),
		ExecutorType:         task.ExecutorType,
		ExecutionMode:        task.ExecutionMode,
		CredentialRefID:      task.CredentialRefID,
		CredentialRefVersion: task.CredentialRefVersion,
		TemplateParams:       decodeJSONMap(task.TemplateParams),
		Status:               task.Status,
		Remark:               task.Remark,
		ExternalTaskID:       task.ExternalTaskID,
		RollbackEnabled:      task.RollbackEnabled,
		RollbackTaskID:       task.RollbackTaskID,
		SourceTaskID:         task.SourceTaskID,
		RollbackStrategy:     task.RollbackStrategy,
		CanRollback:          canRollback,
		StartedAt:            task.StartedAt,
		FinishedAt:           task.FinishedAt,
		HostCount:            hostCount,
		SuccessCount:         successCount,
		FailedCount:          failedCount,
		RunningCount:         runningCount,
		SkippedCount:         skippedCount,
		DurationSeconds:      durationSeconds,
		CreatedAt:            task.CreatedAt,
		UpdatedAt:            task.UpdatedAt,
		CreatedBy:            task.CreatedBy,
		UpdatedBy:            task.UpdatedBy,
		Hosts:                hosts,
	}
}

func taskHostToResponse(host *DeployTaskHost) TaskHostResponse {
	var traceSteps []map[string]any
	if len(host.TraceSteps) > 0 {
		_ = json.Unmarshal(host.TraceSteps, &traceSteps)
	}
	return TaskHostResponse{
		ID:              host.ID,
		TaskID:          host.TaskID,
		HostID:          host.HostID,
		Hostname:        host.Hostname,
		HostIP:          host.HostIP,
		SSHPort:         host.SSHPort,
		OS:              host.OS,
		BusinessScopeID: host.BusinessScopeID,
		Status:          host.Status,
		Stdout:          host.Stdout,
		Stderr:          host.Stderr,
		ErrorMessage:    host.ErrorMessage,
		ExecutorID:      host.ExecutorID,
		ReportKey:       host.ReportKey,
		TraceSteps:      traceSteps,
		StartedAt:       host.StartedAt,
		FinishedAt:      host.FinishedAt,
		ReportedAt:      host.ReportedAt,
		ResolvedAt:      host.ResolvedAt,
		DurationSeconds: computeDurationSeconds(host.StartedAt, host.FinishedAt),
		UpdatedAt:       host.UpdatedAt,
		UpdatedBy:       host.UpdatedBy,
	}
}

func decodeJSONMap(raw datatypes.JSON) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func validateTemplateDefinition(executionMode string, templateCode string, templateConfig map[string]any) error {
	mode := strings.TrimSpace(executionMode)
	code := strings.TrimSpace(templateCode)
	if mode == ExecutionModeOrchestrated {
		return nil
	}
	_ = templateConfig
	return validateFixedTemplateDefinition(code)
}

func validateTemplateParams(executionMode string, templateCode string, templateConfigRaw datatypes.JSON, templateParams map[string]any) error {
	return validateTemplateParamsForCode(executionMode, templateCode, templateConfigRaw, templateParams)
}

func validateDeploySSHStartRequest(req StartTaskRequest) error {
	if strings.TrimSpace(req.HostFingerprint) == "" {
		return errors.New(errDeployTaskSSHHostKeyRequired)
	}
	if strings.TrimSpace(req.SSHUser) == "" {
		return errors.New(errDeployTaskSSHUserRequired)
	}
	if strings.TrimSpace(req.AuthMode) == "private_key" {
		if strings.TrimSpace(req.SSHPrivateKey) == "" {
			return errors.New(errDeployTaskSSHPrivateKeyRequired)
		}
		return nil
	}
	if strings.TrimSpace(req.SSHPassword) == "" {
		return errors.New(errDeployTaskSSHPasswordRequired)
	}
	return nil
}

func (s *DeployService) validateTaskExecutionPlan(task DeployTask, hosts []cmdbHostSnapshot) error {
	plan, err := s.resolveTaskExecutionPlan(task)
	if err != nil {
		return err
	}
	if len(hosts) == 0 {
		return nil
	}
	sampleHost := hosts[0]
	for _, step := range plan {
		if _, err := s.renderExecutionStepScript(step, task, sampleHost); err != nil {
			return err
		}
		if _, _, err := renderDeployCheckSnippet(step, task, sampleHost, "precheckCommand"); err != nil {
			return err
		}
		if _, _, err := renderDeployCheckSnippet(step, task, sampleHost, "postcheckCommand"); err != nil {
			return err
		}
	}
	return nil
}

func (s *DeployService) resolveInstallScript(pkg DeployPackage, task DeployTask) (string, error) {
	if strings.TrimSpace(pkg.TemplateCode) != "" && pkg.ExecutionMode == ExecutionModeFixed {
		return renderFixedTemplateScript(pkg, task)
	}
	if normalizeTaskAction(task.Action) == TaskActionUninstall {
		if strings.TrimSpace(pkg.UninstallCommand) == "" {
			return "", errors.New(errDeployTaskUninstallCommandRequired)
		}
		return pkg.UninstallCommand, nil
	}
	if strings.TrimSpace(pkg.InstallCommand) == "" {
		return "", errors.New(errDeployTaskInstallCommandRequired)
	}
	return pkg.InstallCommand, nil
}

func renderNginxSystemdScript(pkg DeployPackage, task DeployTask) (string, error) {
	params := decodeJSONMap(task.TemplateParams)
	action := normalizeTaskAction(task.Action)
	installRoot := strings.TrimSpace(anyToString(params["installRoot"]))
	serviceName := strings.TrimSpace(anyToString(params["serviceName"]))
	sourceObjectKey := strings.TrimSpace(pkg.SourceObjectKey)
	sourceURL := strings.TrimSpace(pkg.SourceURL)
	if action != TaskActionUninstall && (installRoot == "" || serviceName == "") {
		return "", errors.New(errDeployTaskTemplateParamsInvalid)
	}
	if action == TaskActionUninstall {
		if serviceName == "" {
			serviceName = "nginx"
		}
		if installRoot == "" {
			installRoot = "/data/nginx"
		}
		return fmt.Sprintf(`set -e
SERVICE_NAME="%s"
INSTALL_ROOT="%s"
systemctl stop "${SERVICE_NAME}" || true
systemctl disable "${SERVICE_NAME}" || true
rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
systemctl daemon-reload
rm -rf "$INSTALL_ROOT"
id nginx >/dev/null 2>&1 && userdel nginx || true
echo "Uninstall completed for ${SERVICE_NAME}"
`, serviceName, installRoot), nil
	}
	if action == TaskActionReinstall {
		action = TaskActionUpgrade
	}
	version := templateParamString(params, "version", pkg.Version)
	if version == "" {
		return "", errors.New(errDeployTaskTemplateInvalid)
	}
	sourceDownload := ""
	if sourceObjectKey != "" {
		if sourceURL == "" {
			return "", errors.New(errDeployTaskPackageSourceMissing)
		}
		sourceDownload = fmt.Sprintf(`SOURCE_URL="%s"
curl -fsSL "$SOURCE_URL" -o "$PKG_DIR/$TARBALL"
`, sourceURL)
	} else {
		sourceDownload = fmt.Sprintf(`if [ ! -f "$PKG_DIR/$TARBALL" ]; then
  curl -fsSL "https://nginx.org/download/$TARBALL" -o "$PKG_DIR/$TARBALL"
fi
`)
	}
	script := fmt.Sprintf(`set -e
NGINX_VERSION="%s"
INSTALL_ROOT="%s"
SERVICE_NAME="%s"
PKG_DIR="/tmp/nginx-${NGINX_VERSION}"
TARBALL="nginx-${NGINX_VERSION}.tar.gz"
SRC_DIR="$PKG_DIR/nginx-${NGINX_VERSION}"
mkdir -p "$INSTALL_ROOT" "$PKG_DIR"
%s
rm -rf "$SRC_DIR"
tar -xzf "$PKG_DIR/$TARBALL" -C "$PKG_DIR"
cd "$SRC_DIR"
if ! id nginx >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /sbin/nologin nginx
fi
if command -v apt-get >/dev/null 2>&1; then
  apt-get update
  DEBIAN_FRONTEND=noninteractive apt-get install -y build-essential libpcre3 libpcre3-dev zlib1g zlib1g-dev libssl-dev curl
elif command -v yum >/dev/null 2>&1; then
  yum install -y gcc make pcre pcre-devel zlib zlib-devel openssl openssl-devel curl
fi
./configure --prefix="$INSTALL_ROOT" --with-http_ssl_module --with-http_stub_status_module
make -j"$(nproc)"
make install
mkdir -p "$INSTALL_ROOT/conf/conf.d" "$INSTALL_ROOT/logs" "$INSTALL_ROOT/client_body_temp"
cat > /etc/systemd/system/${SERVICE_NAME}.service <<EOF
[Unit]
Description=nginx web server
After=network.target

[Service]
Type=forking
PIDFile=%s/logs/nginx.pid
ExecStartPre=%s/sbin/nginx -t
ExecStart=%s/sbin/nginx
ExecReload=%s/sbin/nginx -s reload
ExecStop=%s/sbin/nginx -s quit
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
if [ "%s" = "upgrade" ]; then
  systemctl stop "${SERVICE_NAME}" || true
fi
systemctl restart "${SERVICE_NAME}"
systemctl status "${SERVICE_NAME}" --no-pager
rm -rf "$PKG_DIR"
`, version, installRoot, serviceName, sourceDownload, installRoot, installRoot, installRoot, installRoot, installRoot, action)
	return script, nil
}

func (s *DeployService) appendTaskHostTrace(taskID uint64, taskHostID uint64, steps []map[string]any) error {
	if len(steps) == 0 {
		return nil
	}
	var host DeployTaskHost
	if err := s.db.Select("trace_steps").Where("id = ? AND task_id = ?", taskHostID, taskID).Take(&host).Error; err != nil {
		return err
	}
	var current []map[string]any
	if len(host.TraceSteps) > 0 {
		_ = json.Unmarshal(host.TraceSteps, &current)
	}
	current = append(current, steps...)
	payload, err := json.Marshal(current)
	if err != nil {
		return err
	}
	return s.db.Model(&DeployTaskHost{}).Where("id = ? AND task_id = ?", taskHostID, taskID).Update("trace_steps", datatypes.JSON(payload)).Error
}

func anyToString(value any) string {
	if value == nil {
		return ""
	}
	switch current := value.(type) {
	case string:
		return current
	default:
		return fmt.Sprint(current)
	}
}

func truncateDeployLog(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	suffix := "\n...[truncated]"
	limit := maxBytes - len(suffix)
	if limit <= 0 {
		return suffix[:maxBytes]
	}
	return value[:limit] + suffix
}

func computeDurationSeconds(startedAt, finishedAt *time.Time) int64 {
	if startedAt == nil || finishedAt == nil {
		return 0
	}
	seconds := int64(finishedAt.Sub(*startedAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func extractTaskIDs(rows []DeployTask) []uint64 {
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func (s *DeployService) loadTaskHostsByTaskIDs(taskIDs []uint64) (map[uint64][]TaskHostResponse, error) {
	result := make(map[uint64][]TaskHostResponse, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}
	var hosts []DeployTaskHost
	if err := s.db.Where("task_id IN ?", taskIDs).Order("task_id ASC, id ASC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	for _, host := range hosts {
		result[host.TaskID] = append(result[host.TaskID], taskHostToResponse(&host))
	}
	return result, nil
}

func (s *DeployService) loadPackageDeploymentStats(packageIDs []uint64) (map[uint64]packageDeploymentStat, error) {
	result := make(map[uint64]packageDeploymentStat, len(packageIDs))
	if len(packageIDs) == 0 {
		return result, nil
	}
	var tasks []DeployTask
	if err := s.db.Where("package_id IN ?", packageIDs).Order(idDescOrder).Find(&tasks).Error; err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if _, exists := result[task.PackageID]; exists {
			continue
		}
		var hostRows []DeployTaskHost
		if err := s.db.Where(taskIDWhereClause, task.ID).Find(&hostRows).Error; err != nil {
			return nil, err
		}
		successCount := 0
		for _, host := range hostRows {
			if host.Status == TaskHostStatusSuccess {
				successCount++
			}
		}
		latestTime := task.FinishedAt
		if latestTime == nil {
			latestTime = task.StartedAt
		}
		stat := packageDeploymentStat{
			LatestTaskID:       task.ID,
			LatestTaskName:     task.Name,
			LatestTaskStatus:   task.Status,
			LatestHostCount:    len(hostRows),
			LatestSuccessCount: successCount,
		}
		if latestTime != nil {
			stat.LatestDeployedAt = *latestTime
		}
		result[task.PackageID] = stat
	}
	return result, nil
}
