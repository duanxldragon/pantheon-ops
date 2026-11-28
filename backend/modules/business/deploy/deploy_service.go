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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	bizscope "github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	executorprovider "github.com/duanxldragon/pantheon-base/backend/modules/business/deploy/executor"
	deployorchestrator "github.com/duanxldragon/pantheon-base/backend/modules/business/deploy/orchestrator"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"golang.org/x/crypto/ssh"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeployService struct {
	db                       *gorm.DB
	cmdbCapability           cmdb.DeployCMDBCapability
	bizScopeReader           bizcap.BizScopeReader
	hostReader               bizcap.CMDBHostReader
	serviceState             bizcap.ServiceInstanceStateApplier
	serviceStateTransaction  func(*gorm.DB) bizcap.ServiceInstanceStateApplier
	serviceReader            service.Reader
	serviceReaderTransaction func(*gorm.DB) service.Reader
	sshRunnerFactory         func(ctx context.Context, host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error)
	executor                 DeployExecutor
	asyncExecution           bool
	taskCancels              sync.Map
	taskWorkers              sync.Map
	ownership                *executionOwnership
	leaseTiming              *ownershipTiming
	executionRoot            *DeployService
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
	ownership  *executionOwnership
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
	// idAndStatusWhereClause and taskIDAndStatusInWhereClause guard the
	// optimistic state transitions on the task and task-host tables.
	idAndStatusWhereClause       = "id = ? AND status = ?"
	taskIDAndStatusInWhereClause = "task_id = ? AND status IN ?"
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

// SetServiceInstanceStateApplier replaces the owner and clears any old owner's
// transaction factory. Configure an owner/factory pair before starting workers.
func (s *DeployService) SetServiceInstanceStateApplier(applier bizcap.ServiceInstanceStateApplier) {
	s.SetServiceInstanceStateApplierWithTransaction(applier, nil)
}

// SetServiceInstanceStateApplierWithTransaction configures state writeback and
// its same-owner transaction factory together before background reconciliation.
func (s *DeployService) SetServiceInstanceStateApplierWithTransaction(applier bizcap.ServiceInstanceStateApplier, factory func(*gorm.DB) bizcap.ServiceInstanceStateApplier) {
	if applier == nil {
		factory = nil
	} else if nilServiceInstanceState(applier) {
		applier, factory = unboundServiceInstanceState{}, nil
	}
	s.serviceState = applier
	s.serviceStateTransaction = factory
}

func (s *DeployService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&DeployPackage{}, &DeployTemplate{}, &DeployTemplateStep{}, &DeployTask{}, &DeployTaskHost{}, &DeployHostLease{}, &DeployTaskAttempt{}, &DeployCredentialRef{}, &DeployTaskSecret{}, &DeployTaskStep{}, &DeployStepHistory{}, &DeployTaskEvent{}, &DeployTaskLogChunk{})
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

func (s *DeployService) createTask(req CreateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
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
	metadata := action == TaskActionRetire
	if metadata && (req.ExecutorType != ExecutorTypeManual || req.ServiceID == 0 || req.ServiceInstanceID == 0 || retireParamsHaveExecution(req.TemplateParams)) {
		return nil, errors.New(errDeployTaskTemplateInvalid)
	}
	if err := validateDeployStructuralParams(req.TemplateParams); err != nil {
		return nil, err
	}
	if err := validateDeploySecretParams(req.SecretParams); err != nil {
		return nil, err
	}
	var pkg DeployPackage
	if metadata {
		pkg, err = s.loadTaskPackageReference(req.PackageID)
	} else {
		pkg, err = s.loadTaskPackage(req.PackageID, template, req.TemplateParams)
	}
	if err != nil {
		return nil, err
	}
	scopeName, err := s.resolveCreateTaskScopeName(&req, targetIDs, action, dataScope)
	if err != nil {
		return nil, err
	}
	linkTask := serviceLinkTask(req.BusinessScopeID, req.ServiceID, req.ServiceInstanceID, req.TargetType, targetIDs)
	if err := s.validateServiceLink(linkTask, dataScope, nil); err != nil {
		return nil, err
	}
	req.serviceName = linkTask.ServiceName
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
	if normalizeTaskAction(req.Action) != TaskActionRetire && len(req.TemplateParams) == 0 && len(templateDetail.ParameterSchema) > 0 {
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
	scope, err := s.bizScopeReader.GetActive(s.requestContext(), businessScopeID, dataScope)
	if err != nil {
		return "", errors.New(taskCommandScopeInvalid)
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
	var executionSnapshot deployExecutionSnapshot
	if normalizeTaskAction(action) == TaskActionRetire {
		executionSnapshot = retireMetadataExecutionSnapshot(template)
	} else {
		var err error
		executionSnapshot, err = s.buildDeployExecutionSnapshot(pkg, template, action, req.TemplateParams)
		if err != nil {
			return nil, err
		}
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
		ServiceName:       req.serviceName,
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
	db, err := applyTaskTimeRange(db, query)
	if err != nil {
		return nil, err
	}
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}
	if query.ServiceID > 0 {
		db = db.Where("service_id = ?", query.ServiceID)
	}
	if query.ServiceInstanceID > 0 {
		db = db.Where("service_instance_id = ?", query.ServiceInstanceID)
	}
	if query.HostID > 0 {
		db = db.Where("JSON_CONTAINS(target_ids, ?)", strconv.FormatUint(query.HostID, 10))
	}
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
	pageStart := (query.Page - 1) * query.PageSize
	var total int64
	var rows []DeployTask
	if requiresScopedTaskVisibility(dataScope) {
		// NEW-01: resolve visible task ids before paging (ids only plus one batched
		// CMDB scope lookup) instead of loading every task row and paginating in
		// memory. Total stays exact because it counts the visible id set.
		// shortcut: scoped visibility depends on the external CMDB host reader, so
		// candidate ids are still enumerated; upgrade to a joined SQL scope predicate
		// when the CMDB capability exposes one.
		var candidateIDs []uint64
		if err := db.Order(idDescOrder).Pluck("id", &candidateIDs).Error; err != nil {
			return nil, err
		}
		visibleIDs, err := s.filterVisibleTaskIDs(candidateIDs, dataScope)
		if err != nil {
			return nil, err
		}
		total = int64(len(visibleIDs))
		if pageStart < len(visibleIDs) {
			end := pageStart + query.PageSize
			if end > len(visibleIDs) {
				end = len(visibleIDs)
			}
			if err := s.db.Where("id IN ?", visibleIDs[pageStart:end]).Order(idDescOrder).Find(&rows).Error; err != nil {
				return nil, err
			}
		}
	} else {
		// Unscoped callers get a true SQL count plus limit/offset page: no full-table load.
		if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
			return nil, err
		}
		if pageStart < int(total) {
			if err := db.Session(&gorm.Session{}).Order(idDescOrder).Limit(query.PageSize).Offset(pageStart).Find(&rows).Error; err != nil {
				return nil, err
			}
		}
	}
	hostsByTaskID, err := s.loadTaskHostsByTaskIDs(extractTaskIDs(rows), dataScope)
	if err != nil {
		return nil, err
	}
	sequences, err := s.taskCommandSequences(extractTaskIDs(rows))
	if err != nil {
		return nil, err
	}
	items := make([]TaskResponse, 0, len(rows))
	for i := range rows {
		response := taskToResponse(&rows[i], hostsByTaskID[rows[i].ID])
		response.StateVersion = taskCommandVersion(&rows[i], sequences[rows[i].ID])
		items = append(items, response)
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
	hosts, err = s.visibleTaskHosts(hosts, dataScope)
	if err != nil {
		return nil, err
	}
	visibleHostIDs := make([]uint64, 0, len(hosts))
	for _, host := range hosts {
		visibleHostIDs = append(visibleHostIDs, host.ID)
	}
	hostResp := make([]TaskHostResponse, 0, len(hosts))
	for i := range hosts {
		hostResp = append(hostResp, taskHostToResponse(&hosts[i]))
	}
	resp := taskToResponse(task, hostResp)
	sequences, err := s.taskCommandSequences([]uint64{id})
	if err != nil {
		return nil, err
	}
	resp.StateVersion = taskCommandVersion(task, sequences[id])
	resp.SecretPresence = s.deployTaskSecretPresence(task.ID)
	resp.StepHistory, resp.StepHistoryLimited, err = s.loadStepHistory(id, hosts)
	if err != nil {
		return nil, err
	}
	var steps []DeployTaskStep
	if err := s.db.Where(taskIDWhereClause, id).Where("task_host_id IN ?", visibleHostIDs).Order("task_host_id ASC, step_order ASC, id ASC").Find(&steps).Error; err != nil {
		return nil, err
	}
	resp.Steps = make([]TaskStepResponse, 0, len(steps))
	for _, step := range steps {
		resp.Steps = append(resp.Steps, TaskStepResponse{
			ID: step.ID, TaskHostID: step.TaskHostID, StepKey: step.StepKey,
			StepOrder: step.StepOrder, Status: step.Status, AttemptNo: step.AttemptNo,
			StateVersion: step.StateVersion, StartedAt: step.StartedAt, FinishedAt: step.FinishedAt,
		})
	}
	if err := s.db.Model(&DeployTaskLogChunk{}).Select("task_host_id, attempt_no").Where("task_id = ? AND attempt_no > 0", id).Where("task_host_id IN ?", visibleHostIDs).
		Group("task_host_id, attempt_no").Order("task_host_id ASC, attempt_no ASC").Scan(&resp.LogAttempts).Error; err != nil {
		return nil, err
	}
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

func (s *DeployService) updateTask(id uint64, req UpdateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
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
	// Evaluate the final action after template defaults, including indirect
	// attempts to turn an existing executable task into lifecycle metadata.
	if normalizeTaskAction(task.Action) == TaskActionRetire || values.action == TaskActionRetire {
		// Only Create introduces metadata intent. An old executable record cannot
		// become safe merely by changing its action, executor or references.
		if normalizeTaskAction(task.Action) != TaskActionRetire || values.action != TaskActionRetire {
			return nil, errors.New(errDeployTaskTemplateInvalid)
		}
		if _, err := s.resolveTaskExecutionPlan(*task); err != nil {
			return nil, err
		}
		if values.executorType != ExecutorTypeManual || values.serviceID == 0 || values.serviceInstanceID == 0 || retireParamsHaveExecution(values.templateParams) {
			return nil, errors.New(errDeployTaskTemplateInvalid)
		}
	}
	var pkg DeployPackage
	if values.action == TaskActionRetire {
		pkg, err = s.loadTaskPackageReference(values.packageID)
	} else {
		pkg, err = s.loadTaskPackage(values.packageID, template, values.templateParams)
	}
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
	if err := s.validateServiceLink(serviceLinkTask(values.businessScopeID, values.serviceID, values.serviceInstanceID, values.targetType, values.targetIDs), dataScope, nil); err != nil {
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
	if req.Action == nil && values.action != TaskActionRetire {
		values.action = normalizeTaskAction(templateDetail.DefaultAction)
	}
	if values.action != TaskActionRetire && req.TemplateParams == nil && len(templateDetail.ParameterSchema) > 0 {
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
	pkg, err := s.loadTaskPackageReference(packageID)
	if err != nil {
		return pkg, err
	}
	if template == nil {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, templateParams); err != nil {
			return DeployPackage{}, err
		}
	}
	return pkg, nil
}

// A lifecycle metadata reference does not require install parameters or scripts.
func (s *DeployService) loadTaskPackageReference(packageID uint64) (DeployPackage, error) {
	if packageID == 0 {
		return DeployPackage{}, nil
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
		if values.serviceID == 0 && values.serviceInstanceID == 0 {
			values.businessScopeID = 0
		}
		return "", nil
	}
	return s.resolveDeployScopeName(values.businessScopeID, values.targetIDs, values.action, dataScope)
}

func (s *DeployService) persistTaskUpdate(task *DeployTask, values *taskUpdateValues, pkg DeployPackage, template *TemplateResponse, scopeName, actor string) error {
	targetJSON, _ := json.Marshal(values.targetIDs)
	templateParamsJSON, _ := json.Marshal(values.templateParams)
	var executionSnapshot deployExecutionSnapshot
	if values.action == TaskActionRetire {
		// The existing marker was validated above; this only updates draft metadata.
		executionSnapshot = retireMetadataExecutionSnapshot(template)
	} else {
		var err error
		executionSnapshot, err = s.buildDeployExecutionSnapshot(pkg, template, values.action, values.templateParams)
		if err != nil {
			return err
		}
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
		if err := s.fencedTransaction(task.ID, func(tx *gorm.DB, state lockedDeployment) error {
			if state.task.Status != TaskStatusRunning {
				task.Status = state.task.Status
				return nil
			}
			active, err := s.targetTransactionService(tx, s.cmdbCapability).hasActiveTaskAttempt(task.ID, 0, time.Time{})
			if err != nil {
				return err
			}
			if active {
				return nil
			}
			now, err := databaseNow(tx)
			if err != nil {
				return err
			}
			if !state.task.UpdatedAt.Before(now.Add(-5 * time.Minute)) {
				return nil
			}
			for _, lease := range state.leases {
				if lease.TaskID == task.ID && lease.ExpiresAt.After(now) {
					return nil
				}
			}
			for _, attempt := range state.attempts {
				if attempt.Status == TaskStatusRunning {
					if err := tx.Model(&DeployTaskAttempt{}).Where("id = ?", attempt.ID).Updates(map[string]any{"status": "retryable", "lease_expires_at": nil, "finished_at": now, "updated_at": now, "error_message": "worker lease expired"}).Error; err != nil {
						return err
					}
				}
			}
			if err := tx.Model(&DeployTask{}).Where("id = ?", task.ID).Updates(map[string]any{"status": TaskStatusFailed, "finished_at": now, "updated_by": actor, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := appendTaskEventTx(tx, taskEventInput{taskID: task.ID, eventType: "task.reconciled_stale", fromStatus: TaskStatusRunning, toStatus: TaskStatusFailed, actor: actor, reason: "worker lease expired"}); err != nil {
				return err
			}
			task.Status = TaskStatusFailed
			return nil
		}); err != nil {
			return err
		}
	}
	// A task may be cleaned up after it reaches a terminal state. Running and
	// transition states remain protected so a worker cannot race a soft delete.
	if task.Status == TaskStatusRunning || task.Status == TaskStatusPauseRequested ||
		task.Status == TaskStatusCancelRequested {
		return errors.New(errDeployTaskInvalidDeleteState)
	}
	return s.fencedTransaction(task.ID, func(tx *gorm.DB, state lockedDeployment) error {
		if state.task.Status == TaskStatusRunning || state.task.Status == TaskStatusPauseRequested || state.task.Status == TaskStatusCancelRequested {
			return errors.New(errDeployTaskInvalidDeleteState)
		}

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
	return s.StartTaskContext(context.Background(), id, req, actor, dataScope)
}

// StartTaskContext runs the admission and ownership checks for starting a task.
func (s *DeployService) StartTaskContext(ctx context.Context, id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.ownershipTiming().operation)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	reads := s.targetDiscoveryService(ctx)
	task, err := reads.loadVisibleTask(id, dataScope)
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
	if err := validateRetireTaskIntent(*task); err != nil {
		return nil, err
	}
	if task.ExecutorType == ExecutorTypeSSH {
		resolvedReq, version, err := reads.prepareStartCredential(req, *task, dataScope == nil || dataScope.IsAdmin)
		if err != nil {
			return nil, err
		}
		req, credentialVersion = resolvedReq, version
	}
	hosts, err := reads.prepareStartTargets(task, req, dataScope)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	claim := startTaskClaim{
		service: s, dataScope: dataScope,
		task:              task,
		req:               req,
		hosts:             hosts,
		actor:             actor,
		key:               key,
		now:               now,
		timeout:           normalizeDeployExecutionTimeout(req.TimeoutSeconds),
		leaseOwner:        deployHostLeaseOwner(task.ID),
		credentialVersion: credentialVersion,
	}
	var claimErr error
	for discovery := 0; discovery < 3; discovery++ {
		claimErr = s.targetRequestService(ctx).deploymentTransaction(claim.claim)
		if !errors.Is(claimErr, ErrTaskPreviewConflict) || req.RequirePreview || req.IntentVersion != "" {
			break
		}
		current, err := reads.loadVisibleTask(id, dataScope)
		if err != nil {
			return nil, err
		}
		candidates, err := reads.prepareStartTargets(current, req, dataScope)
		if err != nil {
			return nil, err
		}
		claim.task = current
		claim.hosts = candidates
	}
	if claimErr != nil {
		return s.rejectFailedStartClaim(id, claimErr, key, dataScope)
	}
	task, hosts = claim.task, claim.hosts

	if err := s.dispatchStartedTask(task, hosts, req, actor); err != nil {
		return nil, err
	}
	return s.GetTask(id, dataScope)
}

// prepareStartCredential resolves the SSH credential reference of a start
// request into the values persisted on the task.
func (s *DeployService) prepareStartCredential(req StartTaskRequest, task DeployTask, allowGlobal bool) (StartTaskRequest, uint64, error) {
	if err := validateCredentialReferenceRequest(req); err != nil {
		return req, 0, err
	}
	resolvedReq, version, err := s.resolveStartCredential(req, task, allowGlobal)
	if err != nil {
		return req, 0, err
	}
	return resolvedReq, version, nil
}

// prepareStartTargets validates the frozen execution snapshot and resolves the
// hosts the start request targets.
func (s *DeployService) prepareStartTargets(task *DeployTask, req StartTaskRequest, dataScope *common.DataScopeReq) ([]cmdbHostSnapshot, error) {
	if err := s.validateActiveExecutionScope(s.requestContext(), task, dataScope); err != nil {
		return nil, err
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
	if err := s.validateServiceLink(task, dataScope, hosts); err != nil {
		return nil, err
	}
	if task.ExecutorType == ExecutorTypeSSH {
		if err := validateDeploySSHStartRequest(req); err != nil {
			return nil, err
		}
		if err := s.validateTaskExecutionPlan(*task, hosts); err != nil {
			return nil, mapDeployTaskExecutionPlanError(err)
		}
	}
	return hosts, nil
}

// startTaskClaim carries the resolved start request of the claiming
// transaction so the transaction keeps its inputs in one place.
type startTaskClaim struct {
	service              *DeployService
	dataScope            *common.DataScopeReq
	task                 *DeployTask
	req                  StartTaskRequest
	hosts                []cmdbHostSnapshot
	actor                string
	key                  string
	now                  time.Time
	timeout              time.Duration
	leaseOwner           string
	credentialVersion    uint64
	serviceStateSnapshot *deployServiceStateSnapshot
}

// claim flips the task to running, freezes the target snapshot and creates the
// per-host rows together with their leases.
func (c *startTaskClaim) claim(tx *gorm.DB) error {
	var current DeployTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, c.task.ID).Error; err != nil {
		return err
	}
	if current.Status != TaskStatusDraft && current.Status != TaskStatusPending {
		return errors.New(errDeployTaskAlreadyRunning)
	}
	c.task = &current
	sort.Slice(c.hosts, func(i, j int) bool { return c.hosts[i].ID < c.hosts[j].ID })
	// Reserve the task-host keys before physical leases and bound CMDB locks.
	for _, host := range c.hosts {
		row := DeployTaskHost{TaskID: c.task.ID, HostID: host.ID, Status: TaskHostStatusPending}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
			return err
		}
	}
	if _, err := lockDeployment(tx, c.task.ID, true); err != nil {
		return err
	}
	for _, host := range c.hosts {
		if err := acquireHostLease(tx, host.ID, c.task.ID, c.leaseOwner, c.now); err != nil {
			return err
		}
	}
	if err := c.validatePreview(tx); err != nil {
		return err
	}
	inner := c.service.targetTransactionService(tx, c.service.cmdbCapability)
	if c.task.ServiceInstanceID != 0 && inner.serviceState == nil {
		return errors.New(errDeployServiceLinkInvalid)
	}
	stateSnapshot, err := inner.beginServiceStateSnapshot(*c.task, c.actor, c.dataScope)
	if err != nil {
		return err
	}
	c.serviceStateSnapshot = stateSnapshot

	result := tx.Model(&DeployTask{}).
		Where(idWhereClause, c.task.ID).
		Where("status IN ?", []string{TaskStatusDraft, TaskStatusPending}).
		Updates(map[string]interface{}{
			"status":                    TaskStatusRunning,
			"started_at":                &c.now,
			"start_request_key":         c.key,
			"credential_ref_id":         c.req.CredentialRefID,
			"credential_ref_version":    c.credentialVersion,
			"ssh_host_fingerprint":      strings.TrimSpace(c.req.HostFingerprint),
			"execution_timeout_seconds": int(c.timeout.Seconds()),
			"updated_by":                c.actor,
			"updated_at":                c.now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(errDeployTaskAlreadyRunning)
	}

	targetSnapshot := buildDeployTargetSnapshot(c.task, c.hosts, c.now)
	targetSnapshot.ServiceState = c.serviceStateSnapshot
	if c.serviceStateSnapshot != nil {
		targetSnapshot.ExecutionGeneration = 1
	}
	var selectionSnapshot deployTargetSnapshot
	if err := json.Unmarshal(c.task.TargetSnapshot, &selectionSnapshot); err == nil {
		targetSnapshot.Selection = selectionSnapshot.Selection
		targetSnapshot.ScopeVersion = selectionSnapshot.ScopeVersion
		targetSnapshot.Sources = selectionSnapshot.Sources
	}
	targetJSON, _ := json.Marshal(targetSnapshot)
	if err := tx.Model(&DeployTask{}).Where(idWhereClause, c.task.ID).Update("target_snapshot", datatypes.JSON(targetJSON)).Error; err != nil {
		return err
	}

	for _, host := range c.hosts {
		if err := c.createTaskHostRow(tx, host); err != nil {
			return err
		}
	}
	return appendTaskEventTx(tx, taskEventInput{taskID: c.task.ID, eventType: "task.started", fromStatus: c.task.Status, toStatus: TaskStatusRunning, actor: c.actor, requestID: c.key})
}

// createTaskHostRow acquires the host lease and persists the running host row
// with the execution steps needed for checkpoint recovery.
func (c startTaskClaim) createTaskHostRow(tx *gorm.DB, host cmdbHostSnapshot) error {

	initialTrace, _ := json.Marshal([]map[string]any{
		{"at": c.now.Format(time.RFC3339), "phase": "start", "message": "Task host queued"},
	})
	row := DeployTaskHost{
		TaskID:          c.task.ID,
		HostID:          host.ID,
		Hostname:        host.Hostname,
		HostIP:          host.IP,
		SSHPort:         host.SSHPort,
		OS:              host.OS,
		BusinessScopeID: host.BusinessScopeID,
		Status:          TaskHostStatusRunning,
		TraceSteps:      datatypes.JSON(initialTrace),
		StartedAt:       &c.now,
		ResolvedAt:      &c.now,
		UpdatedBy:       c.actor,
	}
	if err := tx.Where("task_id = ? AND host_id = ?", c.task.ID, host.ID).First(&row).Error; err != nil {
		return err
	}
	if err := tx.Model(&row).Updates(map[string]any{"hostname": host.Hostname, "host_ip": host.IP, "ssh_port": host.SSHPort, "os": host.OS, "business_scope_id": host.BusinessScopeID, "status": TaskHostStatusRunning, "trace_steps": datatypes.JSON(initialTrace), "started_at": c.now, "resolved_at": c.now, "updated_by": c.actor}).Error; err != nil {
		return err
	}
	// FZ-03: persist execution steps for checkpoint recovery.
	var executionSnapshot deployExecutionSnapshot
	if err := json.Unmarshal(c.task.ExecutionSnapshot, &executionSnapshot); err == nil && len(executionSnapshot.Steps) > 0 {
		return persistTaskSteps(tx, c.task.ID, row.ID, executionSnapshot.Steps)
	}
	return nil
}

// rejectFailedStartClaim replays an idempotent start that lost the claim race
// and otherwise reports the claim failure.
func (s *DeployService) rejectFailedStartClaim(id uint64, claimErr error, key string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	if strings.TrimSpace(errorText(claimErr)) != errDeployTaskAlreadyRunning {
		return nil, claimErr
	}
	current, loadErr := s.loadVisibleTask(id, dataScope)
	if loadErr == nil && current.Status == TaskStatusRunning && key != "" && current.StartRequestKey == key {
		return s.getTaskReused(id, dataScope)
	}
	return nil, errors.New(errDeployTaskAlreadyRunning)
}

// failStartedTask marks the claimed task and its hosts failed when the service
// instance could not be opened, and releases their leases.
func (s *DeployService) failStartedTask(task *DeployTask, _ []cmdbHostSnapshot, actor string, reason error) {
	if err := s.failQueuedTask(task.ID, actor, reason); err != nil {
		slog.Error("settle failed deploy start", "task_id", task.ID, "reason", reason, "error", err)
	}
}

// dispatchStartedTask hands the claimed task to the async worker or runs it
// inline for the current request.
func (s *DeployService) dispatchStartedTask(task *DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, actor string) error {
	if task.ExecutorType != ExecutorTypeSSH {
		return nil
	}
	if s.asyncExecution {
		// 记录任务开始 metrics
		recordTaskStart(task)
		go s.runAsyncTask(task.ID, actor)
		return nil
	}
	return s.executeSSHTask(*task, hosts, req, actor)
}

type taskWorkerSlot struct {
	mu      sync.Mutex
	retired bool
	pending bool
	done    chan struct{}
}

func (slot *taskWorkerSlot) reserveDispatch() bool {
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if slot.retired {
		return false
	}
	slot.pending = true
	return true
}

func (s *DeployService) reserveTaskWorker(taskID uint64) *taskWorkerSlot {
	for {
		slot := &taskWorkerSlot{done: make(chan struct{})}
		current, loaded := s.taskWorkers.LoadOrStore(taskID, slot)
		if !loaded {
			return slot
		}
		if worker, ok := current.(*taskWorkerSlot); ok && worker.reserveDispatch() {
			return nil
		}
		// A dispatcher can load an old pointer before retirement and acquire
		// its lock after retirement. Retry the current map slot in that case.
	}
}

func (s *DeployService) releaseTaskWorker(taskID uint64, slot *taskWorkerSlot, cancel context.CancelFunc) bool {
	slot.mu.Lock()
	defer slot.mu.Unlock()
	if cancel != nil {
		cancel()
		s.taskCancels.Delete(taskID)
	}
	slot.retired = true
	pending := slot.pending
	s.taskWorkers.CompareAndDelete(taskID, slot)
	return pending
}

func (s *DeployService) runAsyncTask(taskID uint64, actor string) {
	slot := s.reserveTaskWorker(taskID)
	if slot == nil {
		return
	}
	// Reservation retirement precedes final settlement. Signal complete exit
	// only after every deferred cleanup and database operation has returned.
	defer close(slot.done)
	var cancelTask context.CancelFunc
	var pending bool
	releaseWorker := sync.OnceFunc(func() {
		pending = s.releaseTaskWorker(taskID, slot, cancelTask)
	})
	defer releaseWorker()
	workerID := fmt.Sprintf("deploy-worker:%d", time.Now().UnixNano())
	if _, err := s.ReconcileDeployAttempts(time.Now()); err != nil {
		return
	}
	var task DeployTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return
	}
	if err := validateRetireRemoteIntent(task); err != nil {
		slog.Error("reject queued deploy execution intent", "task_id", taskID, "error", err)
		if settlementErr := s.failQueuedTask(taskID, actor, err); settlementErr != nil {
			slog.Error("settle rejected deploy execution intent failed", "task_id", taskID, "reason", err, "error", settlementErr)
		}
		return
	}
	req, err := s.startRequestForTask(task)
	if err != nil {
		if settlementErr := s.failQueuedTask(taskID, actor, err); settlementErr != nil {
			slog.Error("settle queued deploy credential failure", "task_id", taskID, "reason", err, "error", settlementErr)
		}
		return
	}
	var hosts []DeployTaskHost
	if err := s.db.Where(taskIDWhereClause, taskID).Order("id ASC").Find(&hosts).Error; err != nil {
		return
	}

	taskContext, cancelTask := context.WithCancel(context.Background())
	s.taskCancels.Store(taskID, cancelTask)
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
	// Retire the reservation after hosts drain, preserving any Resume dispatch
	// received while an owned completion's pause commit reply was in flight.
	releaseWorker()
	// A pause or cancel can stop dispatch before any host claims an attempt.
	// Acknowledge it after draining through the same fenced completion path.
	if err := s.recomputeTaskStatus(taskID, actor); err != nil {
		slog.Error("settle drained deploy worker failed", "task_id", taskID, "error", err)
		return
	}
	if pending {
		if err := s.dispatchPendingTask(taskID, actor); err != nil {
			slog.Error("dispatch pending deploy worker failed", "task_id", taskID, "error", err)
		}
	}
}

func (s *DeployService) dispatchPendingTask(taskID uint64, actor string) error {
	var dispatch bool
	err := s.fencedTransaction(taskID, func(tx *gorm.DB, state lockedDeployment) error {
		if state.task.Status != TaskStatusRunning || state.task.ExecutorType != ExecutorTypeSSH {
			return nil
		}
		ids := make([]uint64, 0, len(state.hosts))
		for _, host := range state.hosts {
			if host.Status == TaskHostStatusPending || host.Status == TaskHostStatusRunning {
				ids = append(ids, host.HostID)
			}
		}
		if len(ids) == 0 {
			return nil
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		for _, attempt := range state.attempts {
			if attempt.Status == TaskHostStatusRunning && attempt.LeaseExpiresAt != nil && attempt.LeaseExpiresAt.After(now) {
				return nil
			}
		}
		for _, lease := range state.leases {
			if lease.ExpiresAt.After(now) && (lease.TaskID != taskID || lease.Owner != deployHostLeaseOwner(taskID)) {
				return nil
			}
		}
		if state.task.ServiceID != 0 || state.task.ServiceInstanceID != 0 {
			provider, err := s.boundCMDB(tx, ids)
			if err != nil {
				return err
			}
			_, err = s.targetTransactionService(tx, provider).serviceInstanceForCallback(state.task, true, nil)
			if errors.Is(err, errServiceExecutionSuperseded) {
				return nil
			}
			if err != nil {
				return err
			}
		}
		dispatch = true
		return nil
	})
	if err == nil && dispatch {
		go s.runAsyncTask(taskID, actor)
	}
	return err
}

func (s *DeployService) runAsyncTaskHost(taskContext context.Context, task DeployTask, host DeployTaskHost, req StartTaskRequest, actor, workerID string) bool {
	for i := 0; i < deployAttemptMax; i++ {
		// A Resume may already be running while the paused attempt's commit
		// reply drains. The canceled old worker must leave retries to its successor.
		if taskContext.Err() != nil {
			return false
		}
		failed, err := s.runOwnedHost(taskContext, task, host, req, actor, workerID, true)
		if err != nil || !failed {
			return failed
		}
	}
	return true
}

// executeAsyncHostAttempt runs the deploy plan of one host under the execution
// timeout of its task.
func (s *DeployService) executeAsyncHostAttempt(taskContext context.Context, task DeployTask, host DeployTaskHost, req StartTaskRequest, actor string) error {
	target := cmdbHostSnapshot{ID: host.HostID, Hostname: host.Hostname, IP: host.HostIP, SSHPort: host.SSHPort, OS: host.OS, BusinessScopeID: host.BusinessScopeID}
	hostContext, cancelHost := context.WithTimeout(taskContext, deployExecutionTimeout(task))
	defer cancelHost()
	return s.executeTaskHostContext(hostContext, task, target, taskHostToResponse(&host), req, actor)
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

// deployExecutionTimeout returns the per-attempt execution budget, not a
// whole-task deadline: executeAsyncHostAttempt re-arms it independently for
// every attempt (deployAttemptMax retries), so ExecutionTimeoutSeconds=T is
// "one attempt may run up to T", with an effective wall-clock ceiling of
// roughly T×deployAttemptMax when every attempt burns its full budget. A hung
// script is cut off at its own attempt's budget; visible wall-clock time may
// exceed T while later attempts retry. Ruled after round-7 triage
// (2026-11-20-ops-deploy-fixed-template-security): accepted as per-attempt
// semantics, documented, no behavior change.
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
	// A queue worker/restart path carries no caller scope: it replays the
	// snapshot the original start already authorized.
	req, version, err := s.resolveStartCredential(StartTaskRequest{CredentialRefID: task.CredentialRefID, HostFingerprint: task.SSHHostFingerprint, TimeoutSeconds: task.ExecutionTimeoutSeconds}, task, true)
	if err != nil {
		return StartTaskRequest{}, err
	}
	if version != task.CredentialRefVersion {
		return StartTaskRequest{}, errors.New("business.deploy.credential.version_changed")
	}
	return req, nil
}

func (s *DeployService) failQueuedTask(taskID uint64, actor string, reason error) error {
	return s.fencedTransaction(taskID, func(tx *gorm.DB, state lockedDeployment) error {
		if state.task.Status != TaskStatusRunning {
			return nil
		}
		for _, a := range state.attempts {
			if a.Status == TaskHostStatusRunning {
				return errExecutionOwnership
			}
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		if err := tx.Model(&DeployTaskHost{}).Where(taskIDAndStatusInWhereClause, taskID, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]any{"status": TaskHostStatusFailed, "error_message": truncateDeployLog(reason.Error(), 480), "finished_at": now, "updated_at": now, "updated_by": actor}).Error; err != nil {
			return err
		}
		for _, h := range state.hosts {
			if err := releaseHostLease(tx, h.HostID, deployHostLeaseOwner(taskID)); err != nil {
				return err
			}
		}
		return s.recomputeTaskStatusTx(tx, state.task, actor)
	})
}

func (s *DeployService) taskCanceled(taskID uint64) bool {
	status, err := s.executionTaskStatus(taskID)
	return err != nil || status == TaskStatusCanceled
}

func (s *DeployService) taskCancellationRequested(taskID uint64) bool {
	status, err := s.executionTaskStatus(taskID)
	return err != nil || status == TaskStatusCancelRequested || status == TaskStatusCanceled
}

func (s *DeployService) taskExecutionSuspended(taskID uint64) bool {
	status, err := s.executionTaskStatus(taskID)
	return err != nil || status == TaskStatusPauseRequested || status == TaskStatusPaused
}

func (s *DeployService) claimAttempt(taskID, taskHostID uint64, workerID string) *DeployTaskAttempt {
	attempt, err := s.claimOwnedAttempt(taskID, taskHostID, workerID)
	if err != nil {
		return nil
	}
	return attempt
}

// ReconcileDeployAttempts makes abandoned worker claims visible to the next
// worker. It is safe to call periodically or during worker startup.
func (s *DeployService) ReconcileDeployAttempts(_ time.Time) (int64, error) {
	return s.reconcileOwnership()
}

// ReconcileDeployQueue resumes durable running tasks after a worker restart.
// Claims remain protected by attempt leases, so concurrent process startup is
// safe and at most one worker can execute a task host at a time.
func (s *DeployService) ReconcileDeployQueue(actor string) (int, error) {
	if _, err := s.ReconcileDeployAttempts(time.Now()); err != nil {
		return 0, err
	}
	// Requested pause/cancel is acknowledged only after live ownership settles.
	// Include admitted tasks canceled before their first worker claim.
	var requestedTaskIDs []uint64
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	if err := s.db.WithContext(ctx).Model(&DeployTask{}).Where("status IN ?", []string{TaskStatusPauseRequested, TaskStatusCancelRequested}).Order("id ASC").Pluck("id", &requestedTaskIDs).Error; err != nil {
		return 0, err
	}
	for _, id := range requestedTaskIDs {
		if err := s.recomputeTaskStatus(id, actor); err != nil {
			return 0, err
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

func (s *DeployService) finishAttempt(o *executionOwnership, runErr error) {
	status := "success"
	if runErr != nil {
		status = "failed"
	}
	s.finishAttemptWithStatus(o, status, runErr)
}
func (s *DeployService) finishAttemptWithStatus(o *executionOwnership, status string, runErr error) {
	if o == nil {
		return
	}
	_ = s.ownedService(o).completeOwnedAttempt(status, runErr, "system", false)
}

func (s *DeployService) CancelTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	response, err := s.taskControlTransaction(id, func(inner *DeployService) (*TaskResponse, error) {
		return inner.cancelTask(id, actor, dataScope)
	})
	if err == nil {
		if cancel, ok := s.taskCancels.Load(id); ok {
			if cancelFunc, ok := cancel.(context.CancelFunc); ok {
				cancelFunc()
			}
		}
	}
	return response, err
}

func (s *DeployService) cancelTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
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
		TaskStatusCancelRequested: task.ExecutorType == ExecutorTypeManual,
	}
	if !allowed[task.Status] {
		return nil, errors.New(errDeployTaskInvalidCancelState)
	}
	nextStatus := TaskStatusCancelRequested
	// Manual tasks have no worker to acknowledge cancellation. Complete both
	// a new cancel and a legacy stranded request through the existing CAS path.
	if task.Status == TaskStatusPending || task.Status == TaskStatusPaused || task.ExecutorType == ExecutorTypeManual {
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
		Where(idAndStatusWhereClause, task.ID, task.Status).
		Updates(updates)
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidCancelState)
	}
	if err := s.db.Model(&DeployTaskHost{}).Where(taskIDAndStatusInWhereClause, id, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
		"status":      TaskHostStatusSkipped,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error; err != nil {
		return nil, err
	}
	if nextStatus == TaskStatusCanceled {
		if task.Status != TaskStatusPending {
			settled := *task
			settled.Status = TaskStatusCanceled
			if err := s.finishServiceInstanceState(settled, MarkHostResultRequest{}, actor, dataScope); err != nil {
				return nil, err
			}
		}
		var hosts []DeployTaskHost
		if err := s.db.Where(taskIDWhereClause, id).Find(&hosts).Error; err != nil {
			return nil, err
		}
		for _, host := range hosts {
			if err := releaseHostLease(s.db, host.HostID, deployHostLeaseOwner(id)); err != nil {
				return nil, err
			}
		}
	}
	eventType := "task.cancel_requested"
	if nextStatus == TaskStatusCanceled {
		eventType = "task.canceled"
	}
	if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: eventType, fromStatus: task.Status, toStatus: nextStatus, actor: actor}); err != nil {
		return nil, err
	}
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
	nextStatus := TaskStatusPauseRequested
	eventType := "task.pause_requested"
	// Manual execution has no worker to acknowledge a cooperative request.
	if task.ExecutorType == ExecutorTypeManual {
		nextStatus = TaskStatusPaused
		eventType = "task.paused"
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idAndStatusWhereClause, task.ID, TaskStatusRunning).
		Updates(map[string]interface{}{
			"status":     nextStatus,
			"updated_by": actor,
			"updated_at": now,
		})
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidPauseState)
	}
	if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: eventType, fromStatus: TaskStatusRunning, toStatus: nextStatus, actor: actor}); err != nil {
		return nil, err
	}
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
	response, err := s.taskControlTransaction(id, func(inner *DeployService) (*TaskResponse, error) {
		return inner.resumeTask(id, actor, dataScope)
	})
	if err != nil {
		return nil, err
	}
	if s.asyncExecution && response.ExecutorType == ExecutorTypeSSH {
		go s.runAsyncTask(id, actor)
	}
	return response, nil
}

func (s *DeployService) resumeTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusPaused {
		return nil, errors.New(errDeployTaskInvalidResumeState)
	}
	if err := s.validateTaskExecutionAdmission(task, dataScope); err != nil {
		return nil, err
	}
	if task.ServiceID != 0 || task.ServiceInstanceID != 0 {
		if _, err := s.serviceInstanceForCallback(*task, true, dataScope); err != nil {
			return nil, err
		}
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idAndStatusWhereClause, task.ID, TaskStatusPaused).
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
	if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.resumed", fromStatus: TaskStatusPaused, toStatus: TaskStatusRunning, actor: actor}); err != nil {
		return nil, err
	}
	// Re-run the async worker to pick up unfinished hosts from the checkpoint.
	if s.asyncExecution && task.ExecutorType == ExecutorTypeSSH {
		go s.runAsyncTask(task.ID, actor)
	}
	return s.GetTask(id, dataScope)
}

// RetryTask creates a new attempt for a failed task. Successful prior host
// results are retained; only failed, canceled, or not-started hosts are
// scheduled again.
func (s *DeployService) RetryTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	response, err := s.taskControlTransaction(id, func(inner *DeployService) (*TaskResponse, error) {
		return inner.retryTask(id, actor, dataScope)
	})
	if err != nil {
		return nil, err
	}
	if s.asyncExecution && response.ExecutorType == ExecutorTypeSSH {
		go s.runAsyncTask(id, actor)
	}
	return response, nil
}

// Direct controls and versioned ControlTask use the same transaction seam.
// The inner service has no worker maps; side effects run after commit.
func (s *DeployService) taskControlTransaction(id uint64, command func(*DeployService) (*TaskResponse, error)) (*TaskResponse, error) {
	var response *TaskResponse
	err := s.deploymentTransaction(func(tx *gorm.DB) error {
		var task DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&task, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errDeployTaskNotFound
			}
			return err
		}
		if _, err := lockDeployment(tx, id); err != nil {
			return err
		}
		provider := s.cmdbCapability
		if owner, ok := provider.(cmdb.TargetSelectionCapability); ok {
			if bound, ok := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability); ok {
				provider = bound
			}
		}
		inner := s.targetTransactionService(tx, provider)
		var err error
		response, err = command(inner)
		return err
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Match claimAttempt's cumulative nonpaused budget before any control writes.
func (s *DeployService) validateRetryAttemptBudget(task *DeployTask) error {
	if task.ExecutorType != ExecutorTypeSSH {
		return nil
	}
	var hosts []DeployTaskHost
	if err := s.db.Clauses(clause.Locking{Strength: "UPDATE"}).Where("task_id = ? AND status <> ?", task.ID, TaskHostStatusSuccess).Order("id ASC").Find(&hosts).Error; err != nil {
		return err
	}
	for _, host := range hosts {
		var count int64
		if err := s.db.Model(&DeployTaskAttempt{}).Where("task_host_id = ? AND status <> ?", host.ID, TaskStatusPaused).Count(&count).Error; err != nil {
			return err
		}
		if count >= deployAttemptMax {
			return errors.New(msgDeployAttemptExhausted)
		}
	}
	return nil
}

func (s *DeployService) retryTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusFailed {
		return nil, errors.New(errDeployTaskInvalidRetryState)
	}
	if err := s.validateTaskExecutionAdmission(task, dataScope); err != nil {
		return nil, err
	}
	if err := s.validateRetryAttemptBudget(task); err != nil {
		return nil, err
	}
	snapshot := task.TargetSnapshot
	if task.ServiceID != 0 || task.ServiceInstanceID != 0 {
		var frozen deployTargetSnapshot
		if json.Unmarshal(task.TargetSnapshot, &frozen) != nil || len(frozen.Hosts) != 1 || frozen.ExecutionGeneration == 0 || frozen.ExecutionGeneration == ^uint64(0) {
			return nil, errors.New(errDeployServiceLinkInvalid)
		}
		state, err := s.beginServiceStateSnapshot(*task, actor, dataScope)
		if err != nil {
			return nil, err
		}
		frozen.ServiceState = state
		frozen.ExecutionGeneration++
		raw, err := json.Marshal(frozen)
		if err != nil {
			return nil, err
		}
		snapshot = raw
	}
	now := time.Now()
	transition := s.db.Model(&DeployTask{}).
		Where(idAndStatusWhereClause, task.ID, TaskStatusFailed).
		Updates(map[string]interface{}{
			"target_snapshot": snapshot,
			"status":          TaskStatusRunning,
			"finished_at":     nil,
			"updated_by":      actor,
			"updated_at":      now,
		})
	if transition.Error != nil {
		return nil, transition.Error
	}
	if transition.RowsAffected == 0 {
		return nil, errors.New(errDeployTaskInvalidRetryState)
	}
	if err := s.appendTaskEvent(taskEventInput{taskID: task.ID, eventType: "task.retried", fromStatus: TaskStatusFailed, toStatus: TaskStatusRunning, actor: actor}); err != nil {
		return nil, err
	}
	// Reset failed/canceled hosts to running so the worker re-executes them.
	if err := s.db.Model(&DeployTaskHost{}).
		Where(taskIDAndStatusInWhereClause, task.ID, []string{TaskHostStatusFailed, TaskHostStatusSkipped}).
		Updates(map[string]interface{}{
			"status":        TaskHostStatusRunning,
			"error_message": "",
			"finished_at":   nil,
			"reported_at":   nil,
			"updated_at":    now,
		}).Error; err != nil {
		return nil, err
	}
	if s.asyncExecution && task.ExecutorType == ExecutorTypeSSH {
		go s.runAsyncTask(task.ID, actor)
	}
	return s.GetTask(id, dataScope)
}

// --- FZ-03: Event append and step persistence ---

// taskEventInput is the payload of an append-only task event record. Omitted
// fields carry their zero value.
type taskEventInput struct {
	taskID     uint64
	taskHostID uint64
	taskStepID uint64
	eventType  string
	fromStatus string
	toStatus   string
	actor      string
	requestID  string
	reason     string
}

// appendTaskEvent writes an append-only event record for a task transition.
// The sequence is monotonically increasing per task and never reused.
func (s *DeployService) appendTaskEvent(event taskEventInput) error {
	if s.ownership != nil && (event.taskID != s.ownership.taskID || (event.taskHostID != 0 && event.taskHostID != s.ownership.taskHostID)) {
		s.ownership.fail(errExecutionOwnership)
		return errExecutionOwnership
	}
	err := s.withTaskSequence(event.taskID, &DeployTaskEvent{}, func(sequence uint64, tx *gorm.DB) error {
		if event.taskHostID > 0 {
			if err := s.requireHostWriter(tx, event.taskID, event.taskHostID); err != nil {
				return err
			}
		}
		return tx.Create(&DeployTaskEvent{
			TaskID: event.taskID, TaskHostID: event.taskHostID, TaskStepID: event.taskStepID,
			Sequence: sequence, EventType: event.eventType, FromStatus: event.fromStatus,
			ToStatus: event.toStatus, Actor: event.actor, RequestID: event.requestID,
			Reason: event.reason, CreatedAt: time.Now(),
		}).Error
	})
	if err != nil {
		slog.Error("append deploy task event failed", "task_id", event.taskID, "event_type", event.eventType, "error", err)
	}
	return err
}

func (s *DeployService) withTaskSequence(taskID uint64, model any, write func(uint64, *gorm.DB) error) error {
	return s.fencedTransaction(taskID, func(tx *gorm.DB, _ lockedDeployment) error {
		var maxSequence uint64
		if err := tx.Model(model).Where(taskIDWhereClause, taskID).Select("COALESCE(MAX(sequence), 0)").Scan(&maxSequence).Error; err != nil {
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
func (s *DeployService) loadTaskEvents(taskID, afterSequence uint64) ([]DeployTaskEvent, error) {
	var events []DeployTaskEvent
	db := s.db.Where(taskIDWhereClause, taskID)
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
func (s *DeployService) appendLogChunk(taskID, taskHostID, taskStepID uint64, attemptNo int, stream, content string) {
	if s.ownership != nil && (taskID != s.ownership.taskID || taskHostID != s.ownership.taskHostID) {
		s.ownership.fail(errExecutionOwnership)
		return
	}
	if len(content) == 0 {
		return
	}
	for len(content) > 0 {
		segment, remaining := splitLogSegment(content)
		err := s.withTaskSequence(taskID, &DeployTaskLogChunk{}, func(sequence uint64, tx *gorm.DB) error {
			if err := s.requireHostWriter(tx, taskID, taskHostID); err != nil {
				return err
			}
			return tx.Create(&DeployTaskLogChunk{
				TaskID: taskID, TaskHostID: taskHostID, TaskStepID: taskStepID,
				Sequence: sequence, AttemptNo: attemptNo, Stream: stream, Content: segment, CreatedAt: time.Now(),
			}).Error
		})
		if err != nil {
			if s.ownership != nil {
				s.ownership.fail(err)
			}
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

type taskLogFilter struct {
	visibleHostIDs []uint64
	hostID         uint64
	stepID         uint64
	attemptNo      uint64
	firstError     bool
}

// loadLogChunks bounds each authorized provider-source page before it reaches the UI.
func (s *DeployService) loadLogChunks(taskID, afterSequence uint64, filter taskLogFilter) ([]DeployTaskLogChunk, error) {
	var chunks []DeployTaskLogChunk
	db := s.db.Where(taskIDWhereClause, taskID)
	if filter.visibleHostIDs != nil {
		db = db.Where("task_host_id IN ?", filter.visibleHostIDs)
	}
	if afterSequence > 0 {
		db = db.Where("sequence > ?", afterSequence)
	}
	if filter.hostID > 0 {
		db = db.Where("task_host_id = ?", filter.hostID)
	}
	if filter.visibleHostIDs != nil || filter.stepID > 0 || filter.attemptNo > 0 {
		steps := s.db.Model(&DeployTaskStep{}).Select("id").Where(taskIDWhereClause, taskID)
		// Correlation also excludes a chunk mislabeled with another host's step.
		steps = steps.Where("task_host_id = biz_deploy_task_log_chunk.task_host_id")
		if filter.stepID > 0 {
			steps = steps.Where("id = ?", filter.stepID)
		}
		if filter.attemptNo > 0 {
			db = db.Where("attempt_no = ?", filter.attemptNo)
		}
		if filter.stepID == 0 && filter.attemptNo == 0 {
			db = db.Where("task_step_id = 0 OR task_step_id IN (?)", steps)
		} else {
			db = db.Where("task_step_id IN (?)", steps)
		}
	}
	limit := 500
	if filter.firstError {
		db = db.Where("stream = ?", "stderr")
		limit = 1
	}
	if err := db.Order("sequence ASC").Limit(limit).Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// WriteLog implements the executor.LogWriter interface for persistence.
func (s *DeployService) WriteLog(taskID, taskHostID uint64, stepKey, stream, content string) {
	secrets := s.renderSecrets(taskID)
	redacted := redactDeploySecrets(content, secrets)
	checkpoint, err := s.loadTaskStep(taskID, taskHostID, stepKey)
	if err != nil {
		if s.ownership != nil {
			s.ownership.fail(err)
		}
		slog.Error("load deploy task step for log failed", "task_id", taskID, "task_host_id", taskHostID, "step_key", stepKey, "error", err)
		return
	}
	s.appendLogChunk(taskID, taskHostID, checkpoint.ID, checkpoint.AttemptNo, stream, redacted)
}

func (s *DeployService) MarkHostResult(hostID uint64, req MarkHostResultRequest, actor string, dataScope *common.DataScopeReq) (*TaskHostResponse, error) {
	return s.markHostResultWithSummary(hostID, req, actor, dataScope, deployExecutionSummary{})
}

func (s *DeployService) markHostResultWithSummary(hostID uint64, req MarkHostResultRequest, actor string, dataScope *common.DataScopeReq, summary deployExecutionSummary) (*TaskHostResponse, error) {
	if err := normalizeMarkHostResultRequest(&req); err != nil {
		return nil, err
	}
	if s.ownership != nil {
		if hostID != s.ownership.taskHostID {
			return nil, errExecutionOwnership
		}
		err := s.fencedTransaction(s.ownership.taskID, func(_ *gorm.DB, state lockedDeployment) error {
			// Final result and its projection are committed with finish/retry below.
			s.ownership.mu.Lock()
			defer s.ownership.mu.Unlock()
			if s.ownership.result != nil {
				return errExecutionOwnership
			}
			s.ownership.result = &req
			s.ownership.summary = summary
			return nil
		})
		if err != nil {
			return nil, err
		}
		return &TaskHostResponse{ID: hostID, Status: req.Status}, nil
	}
	var response *TaskHostResponse
	var finished DeployTask
	var settled bool
	err := s.deploymentTransaction(func(tx *gorm.DB) error {
		var hint DeployTaskHost
		if err := tx.First(&hint, hostID).Error; err != nil {
			return err
		}
		state, err := lockDeployment(tx, hint.TaskID, true)
		if err != nil {
			return err
		}
		if req.Status == TaskHostStatusSuccess {
			if err := validateRetireTaskIntent(state.task); err != nil {
				return err
			}
			if normalizeTaskAction(state.task.Action) == TaskActionRetire {
				if err := validateRetireResult(state.task, summary); err != nil {
					return err
				}
			}
		}
		if err := s.requireHostWriter(tx, hint.TaskID, hostID); err != nil {
			return err
		}
		leaseNow, err := databaseNow(tx)
		if err != nil {
			return err
		}
		for _, lease := range state.leases {
			if lease.HostID == hint.HostID && lease.ExpiresAt.After(leaseNow) && (lease.TaskID != hint.TaskID || lease.Owner != deployHostLeaseOwner(hint.TaskID)) {
				return errExecutionOwnership
			}
		}
		var host DeployTaskHost
		for _, h := range state.hosts {
			if h.ID == hostID {
				host = h
			}
		}
		provider, err := s.boundCMDB(tx, []uint64{host.HostID})
		if err != nil {
			return err
		}
		inner := s.targetTransactionService(tx, provider)
		visible, err := inner.taskVisible(&state.task, dataScope)
		if err != nil {
			return err
		}
		if !visible {
			return errDeployTaskForbidden
		}
		// Public/manual reports cannot impersonate a durable worker.
		if state.task.ServiceID != 0 || state.task.ServiceInstanceID != 0 {
			version := taskExecutionVersion(&state.task)
			if version == "" || req.ExecutionVersion != version {
				return errors.New(errDeployTaskHostStaleReport)
			}
		}
		for _, a := range state.attempts {
			if a.TaskHostID == hostID && a.Status == TaskHostStatusRunning {
				return errExecutionOwnership
			}
		}
		if isTerminalTaskHostStatus(host.Status) {
			response, err = s.hostResultReplayOrStale(host, req, strings.TrimSpace(req.ReportKey))
			if response != nil {
				response.ExecutionVersion = taskExecutionVersion(&state.task)
			}
			return err
		}
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		if err := inner.commitHostResult(tx, host, &state.task, req, actor, summary, now); err != nil {
			return err
		}
		if err := inner.recomputeTaskStatusTx(tx, state.task, actor, req); err != nil {
			return err
		}
		var finalTask DeployTask
		if err := tx.First(&finalTask, state.task.ID).Error; err != nil {
			return err
		}
		finished = finalTask
		settled = isTerminalTaskStatus(finalTask.Status) && finalTask.Status != state.task.Status

		if err := releaseHostLease(tx, host.HostID, deployHostLeaseOwner(host.TaskID)); err != nil {
			return err
		}
		if err := tx.First(&host, hostID).Error; err != nil {
			return err
		}
		r := taskHostToResponse(&host)
		r.ExecutionVersion = taskExecutionVersion(&state.task)
		response = &r
		return nil
	})
	if err == nil && response != nil {
		h := DeployTaskHost{ID: response.ID, Status: response.Status}
		recordHostResult(&h, finished.ExecutorType)
		if settled {
			s.recordTaskFinishMetrics(finished.ID, finished.Status)
		}
	}
	return response, err
}

// normalizeMarkHostResultRequest rejects unsupported host results, truncates the
// reported text and requires a reason for a failure.
func normalizeMarkHostResultRequest(req *MarkHostResultRequest) error {
	if req.Status != TaskHostStatusSuccess && req.Status != TaskHostStatusFailed && req.Status != TaskHostStatusSkipped {
		return errors.New("business.deploy.taskHost.invalidResultState")
	}
	req.Stdout = truncateDeployLog(req.Stdout, 60000)
	req.Stderr = truncateDeployLog(req.Stderr, 60000)
	req.ErrorMessage = truncateDeployLog(req.ErrorMessage, 480)
	if req.Status == TaskHostStatusFailed && strings.TrimSpace(req.ErrorMessage) == "" {
		return errors.New("business.deploy.taskHost.markFailed.reasonRequired")
	}
	return nil
}

// applyMarkedHostResult persists the reported result while the host is still
// pending or running. It returns false when a competing report settled it.
func (s *DeployService) applyMarkedHostResult(host DeployTaskHost, req MarkHostResultRequest, reportKey, actor string, now time.Time) (bool, error) {
	transition := s.db.Model(&DeployTaskHost{}).
		Where("id = ? AND status IN ?", host.ID, []string{TaskHostStatusPending, TaskHostStatusRunning}).
		Updates(map[string]interface{}{
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
		})
	if transition.Error != nil {
		return false, transition.Error
	}
	return transition.RowsAffected > 0, nil
}

// hostResultReplayOrStale replays the persisted result of an already settled
// host, and reports a stale report when the submission differs.
func (s *DeployService) hostResultReplayOrStale(host DeployTaskHost, req MarkHostResultRequest, reportKey string) (*TaskHostResponse, error) {
	if sameTaskHostReport(host, req, reportKey) {
		resp := taskHostToResponse(&host)
		return &resp, nil
	}
	return nil, errors.New(errDeployTaskHostStaleReport)
}

// replayLostHostResult reloads a host whose result transition did not apply and
// replays its persisted result.
func (s *DeployService) replayLostHostResult(hostID uint64, req MarkHostResultRequest, reportKey string) (*TaskHostResponse, error) {
	var current DeployTaskHost
	if err := s.db.First(&current, hostID).Error; err != nil {
		return nil, err
	}
	return s.hostResultReplayOrStale(current, req, reportKey)
}

// writebackMarkedHostResult publishes the installed or removed components of a
// successful host result to CMDB.
func (s *DeployService) writebackMarkedHostResult(host DeployTaskHost, task *DeployTask, req MarkHostResultRequest, actor string, summary deployExecutionSummary, now time.Time) error {
	if req.Status != TaskHostStatusSuccess {
		return nil
	}
	if _, tx := s.db.Statement.ConnPool.(gorm.TxCommitter); !tx || s.requestContext().Value(deploymentTxKey{}) != true {
		return errors.New("deploy.projection.transactionIsolationUnknown")
	}
	if err := validateRetireTaskIntent(*task); err != nil {
		return err
	}
	if normalizeTaskAction(task.Action) == TaskActionRetire {
		if s.ownership != nil {
			return errors.New(errDeployTaskTemplateInvalid)
		}
		return validateRetireResult(*task, summary)
	}
	writeback := cmdb.DeployHostWritebackRequest{
		HostID: host.HostID,
		Status: hostStatusForAction(task.Action),
		Actor:  actor,
	}
	switch {
	case len(summary.RemovedComponentNames) > 0 || len(summary.InstalledComponents) > 0:
		writeback.RemovedComponentNames = normalizeDeployStringValues(summary.RemovedComponentNames)
		writeback.InstalledComponents = summary.InstalledComponents
	case normalizeTaskAction(task.Action) == TaskActionUninstall:
		writeback.RemovedComponentNames = []string{task.PackageName}
	default:
		writeback.InstalledComponents = []cmdb.InstalledComponentUpsert{{
			Name:           task.PackageName,
			Version:        task.PackageVersion,
			DeployedAt:     now,
			DeployTaskID:   task.ID,
			DeployTaskName: task.Name,
			ExecutorType:   task.ExecutorType,
		}}
	}
	return s.cmdbCapability.WriteDeployHostResult(writeback)
}

func isTerminalTaskStatus(status string) bool {
	return status == TaskStatusSuccess || status == TaskStatusFailed || status == TaskStatusCanceled
}

func (s *DeployService) beginServiceInstanceState(task DeployTask, actor string, scope *common.DataScopeReq) error {
	if task.ServiceID == 0 && task.ServiceInstanceID == 0 {
		return nil
	}
	instance, err := s.serviceInstanceForCallback(task, false, scope)
	if err != nil {
		return err
	}
	if normalizeTaskAction(task.Action) == TaskActionRetire && (instance.DesiredState != observedStateStopped || instance.ObservedState != observedStateStopped) {
		// Stopped-only is the conservative source supported by this coordinator.
		return errors.New("business.service.state_invalid")
	}
	action, observed, desired, ok := deployStateBegin(task.Action)
	if !ok {
		return nil
	}
	return s.serviceState.ApplyServiceInstanceState(s.requestContext(), bizcap.ServiceInstanceStateTransition{
		InstanceID:               task.ServiceInstanceID,
		ExpectedLifecycleVersion: instance.LifecycleVersion,
		Action:                   action,
		DesiredState:             desired,
		ObservedState:            observed,
		DesiredVersion:           strings.TrimSpace(task.PackageVersion),
		CurrentVersion:           "",
		CorrelationID:            fmt.Sprintf("deploy:%d:begin:%d", task.ID, instance.LifecycleVersion),
	}, actor, scope)
}

func (s *DeployService) finishServiceInstanceState(task DeployTask, result MarkHostResultRequest, actor string, scope *common.DataScopeReq) error {
	if task.ServiceID == 0 && task.ServiceInstanceID == 0 {
		return nil
	}
	instance, err := s.serviceInstanceForCallback(task, true, scope)
	if task.Status == TaskStatusCanceled && errors.Is(err, errServiceExecutionSuperseded) {
		return nil
	}
	if err != nil {
		return err
	}
	if normalizeTaskAction(task.Action) == TaskActionRetire && task.Status == TaskStatusSuccess && (instance.DesiredState != observedStateStopped || instance.ObservedState != observedStateStopped) {
		return errors.New("business.service.state_invalid")
	}
	action, observed, desired, health, currentVersion, ok := deployStateFinish(task.Action, task.Status, task.PackageVersion, result.HealthState)
	if !ok {
		return nil
	}
	desiredVersion := strings.TrimSpace(task.PackageVersion)
	if normalizeTaskAction(task.Action) == TaskActionRetire {
		desiredVersion = ""
	}
	err = s.serviceState.ApplyServiceInstanceState(s.requestContext(), bizcap.ServiceInstanceStateTransition{
		InstanceID:               task.ServiceInstanceID,
		ExpectedLifecycleVersion: instance.LifecycleVersion,
		Action:                   action,
		DesiredState:             desired,
		ObservedState:            observed,
		HealthState:              health,
		DesiredVersion:           desiredVersion,
		CurrentVersion:           currentVersion,
		HealthMessage:            firstNonBlank(result.HealthMessage, stateResultMessage(task)),
		HealthRevision:           strings.TrimSpace(result.HealthRevision),
		CorrelationID:            fmt.Sprintf("deploy:%d:finish:%s", task.ID, task.Status),
	}, actor, scope)
	if err != nil || task.Status != TaskStatusSuccess || normalizeTaskAction(task.Action) != TaskActionUninstall {
		return err
	}
	instance, err = s.serviceInstanceForCallback(task, false, scope)
	if err != nil {
		return err
	}
	return s.serviceState.ApplyServiceInstanceState(s.requestContext(), bizcap.ServiceInstanceStateTransition{
		InstanceID:               task.ServiceInstanceID,
		ExpectedLifecycleVersion: instance.LifecycleVersion,
		Action:                   "retire",
		DesiredState:             "retired",
		ObservedState:            "retired",
		HealthState:              "unknown",
		CorrelationID:            fmt.Sprintf("deploy:%d:retire", task.ID),
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

// deployStateFinishResult is the service-instance state a finished deploy
// action leaves behind.
type deployStateFinishResult struct {
	action, observed, desired, health, currentVersion string
}

func deployStateFinish(rawAction, taskStatus, packageVersion, reportedHealth string) (action, observed, desired, health, currentVersion string, ok bool) {
	result, ok := resolveDeployStateFinish(normalizeTaskAction(rawAction), taskStatus == TaskStatusSuccess, packageVersion, reportedHealth)
	return result.action, result.observed, result.desired, result.health, result.currentVersion, ok
}

// resolveDeployStateFinish maps a normalized deploy action and its terminal task
// status onto the service-instance state.
func resolveDeployStateFinish(rawAction string, success bool, packageVersion, reportedHealth string) (deployStateFinishResult, bool) {
	switch rawAction {
	case TaskActionInstall, TaskActionReinstall:
		return finishComponentInstallState(success, packageVersion), true
	case TaskActionUpgrade:
		return finishVersionedServiceState(TaskActionUpgrade, success, packageVersion, reportedHealth), true
	case TaskActionRollback:
		return finishVersionedServiceState(TaskActionRollback, success, packageVersion, reportedHealth), true
	case TaskActionStart:
		return finishServiceStartState(success), true
	case TaskActionHealth:
		return finishHealthProbeState(success, reportedHealth), true
	case TaskActionStop, TaskActionUninstall:
		return finishServiceStopState(success), true
	case TaskActionRetire:
		return finishServiceRetireState(success)
	default:
		return deployStateFinishResult{}, false
	}
}

// finishComponentInstallState reports the stopped instance left by an install,
// reinstall or failed attempt of one.
func finishComponentInstallState(success bool, packageVersion string) deployStateFinishResult {
	if !success {
		return deployStateFinishResult{action: TaskActionInstall, observed: TaskStatusFailed, desired: observedStateStopped, health: observedStateUnknown}
	}
	return deployStateFinishResult{action: TaskActionInstall, observed: observedStateStopped, desired: observedStateStopped, health: observedStateUnknown, currentVersion: strings.TrimSpace(packageVersion)}
}

// finishVersionedServiceState reports the running instance an upgrade or
// rollback leaves behind, with the health the host reported.
func finishVersionedServiceState(action string, success bool, packageVersion, reportedHealth string) deployStateFinishResult {
	if !success {
		return deployStateFinishResult{action: action, observed: TaskStatusFailed, desired: "", health: observedStateUnknown}
	}
	return deployStateFinishResult{action: action, observed: TaskStatusRunning, desired: TaskStatusRunning, health: reportedHealthOrUnknown(reportedHealth), currentVersion: strings.TrimSpace(packageVersion)}
}

// finishServiceStartState reports the instance after a start action.
func finishServiceStartState(success bool) deployStateFinishResult {
	if !success {
		return deployStateFinishResult{action: TaskActionStart, observed: TaskStatusFailed, desired: TaskStatusRunning, health: observedStateUnknown}
	}
	return deployStateFinishResult{action: TaskActionStart, observed: TaskStatusRunning, desired: TaskStatusRunning, health: observedStateUnknown}
}

// finishHealthProbeState reports the instance after a health-only action; the
// desired state stays empty because the probe changes nothing.
func finishHealthProbeState(success bool, reportedHealth string) deployStateFinishResult {
	if !success {
		return deployStateFinishResult{action: TaskActionHealth, observed: TaskStatusRunning, desired: "", health: "unhealthy"}
	}
	return deployStateFinishResult{action: TaskActionHealth, observed: TaskStatusRunning, desired: "", health: reportedHealthOrUnknown(reportedHealth)}
}

// finishServiceStopState reports the stopped instance left by a stop or
// uninstall action.
func finishServiceStopState(success bool) deployStateFinishResult {
	if !success {
		return deployStateFinishResult{action: TaskActionStop, observed: TaskStatusFailed, desired: "stopped", health: observedStateUnknown}
	}
	return deployStateFinishResult{action: TaskActionStop, observed: observedStateStopped, desired: observedStateStopped, health: observedStateUnknown}
}

// finishServiceRetireState reports the retired instance; a failed retire leaves
// the state untouched.
func finishServiceRetireState(success bool) (deployStateFinishResult, bool) {
	if !success {
		return deployStateFinishResult{}, false
	}
	return deployStateFinishResult{action: TaskActionRetire, observed: "retired", desired: "retired", health: observedStateUnknown}, true
}

// reportedHealthOrUnknown falls back to the unknown probe state when the host
// did not report a health state.
func reportedHealthOrUnknown(reportedHealth string) string {
	health := strings.TrimSpace(reportedHealth)
	if health == "" {
		return observedStateUnknown
	}
	return health
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
			StateVersion: row.StateVersion, UpdatedAt: row.UpdatedAt, InstalledComponents: row.InstalledComponents,
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

// filterVisibleTaskIDs resolves which candidate tasks are visible under the
// caller's data scope. Host-linked tasks are decided by a single CMDB reader
// call over all their hosts; tasks with no host rows fall back to target
// resolution, matching taskVisible.
func (s *DeployService) filterVisibleTaskIDs(taskIDs []uint64, dataScope *common.DataScopeReq) ([]uint64, error) {
	if len(taskIDs) == 0 {
		return nil, nil
	}
	var taskHosts []DeployTaskHost
	if err := s.db.Where("task_id IN ?", taskIDs).Find(&taskHosts).Error; err != nil {
		return nil, err
	}
	hostsByTask := make(map[uint64][]uint64, len(taskIDs))
	hostIDs := make([]uint64, 0, len(taskHosts))
	for _, taskHost := range taskHosts {
		hostsByTask[taskHost.TaskID] = append(hostsByTask[taskHost.TaskID], taskHost.HostID)
		hostIDs = append(hostIDs, taskHost.HostID)
	}
	visibleHosts := make(map[uint64]bool, len(hostIDs))
	if len(hostIDs) > 0 {
		if s.hostReader == nil {
			return nil, errors.New("business.cmdb.readerNotConfigured")
		}
		page, err := s.hostReader.GetByIDs(s.requestContext(), bizcap.HostIDsQuery{HostIDs: hostIDs, DataScope: dataScope})
		if err != nil {
			return nil, err
		}
		for _, host := range page.Items {
			visibleHosts[host.ID] = true
		}
	}
	visible := make(map[uint64]bool, len(taskIDs))
	var noHostIDs []uint64
	for _, id := range taskIDs {
		hostIDs, hasHosts := hostsByTask[id]
		if !hasHosts {
			noHostIDs = append(noHostIDs, id)
			continue
		}
		for _, hostID := range hostIDs {
			if visibleHosts[hostID] {
				visible[id] = true
				break
			}
		}
	}
	if len(noHostIDs) > 0 {
		var tasks []DeployTask
		if err := s.db.Where("id IN ?", noHostIDs).Find(&tasks).Error; err != nil {
			return nil, err
		}
		byID := make(map[uint64]*DeployTask, len(tasks))
		for i := range tasks {
			byID[tasks[i].ID] = &tasks[i]
		}
		for _, id := range noHostIDs {
			task := byID[id]
			if task == nil {
				continue
			}
			hosts, err := s.resolveTaskTargets(task, dataScope)
			if err != nil {
				return nil, err
			}
			if len(hosts) > 0 {
				visible[id] = true
			}
		}
	}
	result := make([]uint64, 0, len(taskIDs))
	for _, id := range taskIDs {
		if visible[id] {
			result = append(result, id)
		}
	}
	return result, nil
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
	page, err := s.hostReader.GetByIDs(s.requestContext(), bizcap.HostIDsQuery{HostIDs: ids, DataScope: dataScope})
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
	var finished DeployTask
	var settled bool
	err := s.fencedTransaction(taskID, func(tx *gorm.DB, state lockedDeployment) error {
		if err := s.recomputeTaskStatusTx(tx, state.task, actor); err != nil {
			return err
		}
		if err := tx.First(&finished, taskID).Error; err != nil {
			return err
		}
		settled = isTerminalTaskStatus(finished.Status) && finished.Status != state.task.Status
		return nil
	})
	if err == nil && settled {
		root := s.executionRoot
		if root == nil {
			root = s
		}
		root.recordTaskFinishMetrics(taskID, finished.Status)
	}
	return err
}

// terminalTaskStatus derives the task status implied by the host rows. An
// empty result means at least one host is still pending or running.
func terminalTaskStatus(hosts []DeployTaskHost) string {
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
		return ""
	}
	status := TaskStatusSuccess
	if hasFailed {
		status = TaskStatusFailed
	}
	if allSkipped {
		status = TaskStatusCanceled
	}
	return status
}

// recordTaskFinishMetrics reports completion/failure metrics for a task that
// just reached a terminal status and triggers the configured auto-rollback.
func (s *DeployService) recordTaskFinishMetrics(taskID uint64, status string) {
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	var task DeployTask
	if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return
	}
	if task.StartedAt != nil && task.FinishedAt != nil {
		recordTaskComplete(&task, task.FinishedAt.Sub(*task.StartedAt))
	}
	if status != TaskStatusFailed {
		return
	}
	recordTaskFailure(&task, "execution_failed")
	if !task.RollbackEnabled || task.RollbackStrategy != RollbackStrategyAuto {
		return
	}
	if rollbackErr := s.TriggerAutoRollback(ctx, &task); rollbackErr != nil {
		s.appendTaskEvent(taskEventInput{taskID: taskID, eventType: "rollback.auto_trigger_failed", actor: "system", reason: rollbackErr.Error()})
		return
	}
	s.appendTaskEvent(taskEventInput{taskID: taskID, eventType: "rollback.auto_triggered", actor: "system"})
}

// hasActiveTaskAttempt reports whether a worker currently owns a non-expired
// attempt. A zero taskHostID checks every host under the task.
func (s *DeployService) hasActiveTaskAttempt(taskID, taskHostID uint64, _ time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	db := s.db.WithContext(ctx).Table("biz_deploy_task_attempt AS a").Where("a.task_id = ? AND a.status = 'running' AND a.worker_id <> '' AND a.lease_expires_at > CURRENT_TIMESTAMP(6)", taskID).
		Where("NOT EXISTS (SELECT 1 FROM biz_deploy_task_attempt newer WHERE newer.task_host_id = a.task_host_id AND (newer.attempt_no > a.attempt_no OR (newer.attempt_no = a.attempt_no AND newer.id > a.id)))").
		Where("EXISTS (SELECT 1 FROM biz_deploy_task_host h JOIN biz_deploy_host_lease l ON l.host_id = h.host_id WHERE h.id = a.task_host_id AND h.task_id = a.task_id AND l.task_id = a.task_id AND l.owner = ? AND l.expires_at > CURRENT_TIMESTAMP(6))", deployHostLeaseOwner(taskID))
	if taskHostID > 0 {
		db = db.Where("a.task_host_id = ?", taskHostID)
	}
	var count int64
	err := db.Count(&count).Error
	if s.ownership != nil {
		s.ownership.fail(err)
	}
	return count > 0, err
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

func acquireHostLease(tx *gorm.DB, hostID, taskID uint64, owner string, _ time.Time) error {
	// Duplicate-safe insertion materializes the unique physical key before its
	// locking read; competing processes cannot both pass a check-then-create.
	now, err := databaseNow(tx)
	if err != nil {
		return err
	}
	lease := DeployHostLease{HostID: hostID, TaskID: taskID, Owner: owner, ExpiresAt: now, CreatedAt: now, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&lease).Error; err != nil {
		return err
	}
	var current DeployHostLease
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("host_id = ?", hostID).First(&current).Error; err != nil {
		return err
	}
	now, err = databaseNow(tx)
	if err != nil {
		return err
	}
	if current.ExpiresAt.After(now) && (current.TaskID != taskID || current.Owner != owner) {
		return errors.New(errDeployTaskLeaseConflict)
	}
	return tx.Model(&DeployHostLease{}).Where("host_id = ?", hostID).Updates(map[string]any{"task_id": taskID, "owner": owner, "expires_at": now.Add(defaultHostLeaseDuration), "updated_at": now}).Error
}

func releaseHostLease(db *gorm.DB, hostID uint64, owner string) error {
	if _, tx := db.Statement.ConnPool.(gorm.TxCommitter); !tx || db.Statement.Context.Value(deploymentTxKey{}) != true {
		return errors.New("deploy.lease.requiresSerializedTransaction")
	}
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
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	done := make(chan error, 1)
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ssh session goroutine panic recovered",
					"panic", r,
					"stack", string(debug.Stack()),
				)
				done <- fmt.Errorf("ssh session panic: %v", r)
			}
		}()
		session, err := c.client.NewSession()
		if err != nil {
			done <- err
			return
		}
		defer func() { _ = session.Close() }()
		session.Stdout = &stdout
		session.Stderr = &stderr
		session.Stdin = strings.NewReader(script)
		done <- session.Run("/bin/bash -se")
	}()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		// Each runner owns one host connection. Closing it also unblocks a
		// pending channel-open request; session.Close alone cannot do that.
		_ = c.client.Close()
		<-done
		err = ctx.Err()
	}
	<-joined
	// Run waits for its stdout/stderr copy goroutines. Join it before reading
	// the buffers, including when cancellation closes the connection.
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

func newDeploySSHRunner(ctx context.Context, host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	connectCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(connectCtx, "tcp", net.JoinHostPort(host.IP, fmt.Sprint(port)))
	if err == nil {
		// ssh.NewClientConn has no context parameter. Bound handshake I/O with
		// the same host context and existing ten-second connection budget.
		cancelJoined := make(chan struct{})
		stopCancel := context.AfterFunc(connectCtx, func() { defer close(cancelJoined); _ = connection.Close() })
		sshConnection, channels, requests, handshakeErr := ssh.NewClientConn(connection, net.JoinHostPort(host.IP, fmt.Sprint(port)), config)
		err = handshakeErr
		if !stopCancel() {
			<-cancelJoined
			err = connectCtx.Err()
		}
		if err == nil {
			return &deploySSHClient{client: ssh.NewClient(sshConnection, channels, requests)}, nil
		}
		_ = connection.Close()
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	if strings.Contains(strings.TrimSpace(err.Error()), errDeployTaskSSHHostKeyMismatch) {
		return nil, errors.New(errDeployTaskSSHHostKeyMismatch)
	}
	return nil, errors.New(errDeployTaskSSHConnectFailed)
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
	if err := validateRetireRemoteIntent(task); err != nil {
		return err
	}
	if _, err := s.resolveTaskExecutionPlan(task); err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	worker := fmt.Sprintf("deploy-inline:%d", time.Now().UnixNano())
	for _, target := range hosts {
		var host DeployTaskHost
		if err := s.db.Where("task_id = ? AND host_id = ?", task.ID, target.ID).First(&host).Error; err != nil {
			return err
		}
		failed, err := s.runOwnedHost(context.Background(), task, host, req, actor, worker, false)
		if err != nil {
			return err
		}
		if failed && !taskContinueOnHostFailure(task) {
			if err := s.db.First(&host, host.ID).Error; err != nil {
				return err
			}
			if strings.TrimSpace(host.ErrorMessage) != "" {
				return errors.New(host.ErrorMessage)
			}
			return errors.New("business.deploy.task.hostFailed")
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
	if normalizeTaskAction(task.Action) == TaskActionRetire || normalizeTaskAction(step.Action) == TaskActionRetire {
		return
	}
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
	if err := validateRetireRemoteIntent(task); err != nil {
		return err
	}
	if s.ownership == nil {
		return errExecutionOwnership
	}
	if err := s.validateExecutionHosts(ctx, &task, []uint64{taskHost.HostID}, nil); err != nil {
		if recordErr := s.markSSHConnectFailure(task, taskHost, err, target.IP, actor); recordErr != nil {
			return recordErr
		}
		return err
	}
	if s.executor == nil {
		s.executor = &sshDeployExecutor{service: s}
	}
	if err := s.ownership.check(); err != nil {
		return err
	}
	return s.executor.ExecuteHost(ctx, DeployExecutionHostRequest{Task: task, TaskHost: taskHost, Credential: req, Actor: actor, ownership: s.ownership})
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

func (p *sshStepProvider) ExecuteStep(ctx context.Context, req executorprovider.StepRequest, logWriter executorprovider.LogWriter) executorprovider.StepResult {
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
	p.service.appendTaskEvent(taskEventInput{taskID: p.task.ID, taskHostID: p.taskHost.ID, taskStepID: checkpoint.ID, eventType: "step.running", fromStatus: checkpoint.Status, toStatus: StepStatusRunning, actor: req.Actor})

	stdoutStart := len(p.execution.stdoutSections)
	stderrStart := len(p.execution.stderrSections)
	previousExecutionErr := p.execution.executionErr
	stop, changed := p.service.executeTaskStep(ctx, taskStepHostRun{task: p.task, target: p.target, taskHost: p.taskHost, runner: p.runner, execution: p.execution}, step)
	writeNewExecutionLogs(logWriter, req, p.execution, stdoutStart, stderrStart)
	if !stop {
		checkpoint.Changed = changed
		if err := p.service.updateTaskStepStatus(&checkpoint, StepStatusSuccess, "", ""); err != nil {
			return executorprovider.StepResult{Status: StepStatusFailed, ErrorMessage: err.Error()}
		}
		p.service.appendTaskEvent(taskEventInput{taskID: p.task.ID, taskHostID: p.taskHost.ID, taskStepID: checkpoint.ID, eventType: "step.success", fromStatus: StepStatusRunning, toStatus: StepStatusSuccess, actor: req.Actor})
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
	p.service.appendTaskEvent(taskEventInput{taskID: p.task.ID, taskHostID: p.taskHost.ID, taskStepID: checkpoint.ID, eventType: "step." + status, fromStatus: StepStatusRunning, toStatus: status, actor: req.Actor, reason: truncateDeployLog(errorMessage, 256)})
	return executorprovider.StepResult{Status: status, ErrorMessage: errorMessage}
}

func (p *sshStepProvider) Cancel(_ context.Context, handle executorprovider.ExecutionHandle) error {
	if handle.CancelFunc != nil {
		handle.CancelFunc()
	}
	return nil
}

func writeNewExecutionLogs(logWriter executorprovider.LogWriter, req executorprovider.StepRequest, execution *taskHostExecution, stdoutStart, stderrStart int) {
	if logWriter == nil {
		return
	}
	if stdoutStart < len(execution.stdoutSections) {
		logWriter.WriteLog(req.TaskID, req.TaskHostID, req.StepKey, "stdout", strings.Join(execution.stdoutSections[stdoutStart:], "\n"))
	}
	if stderrStart < len(execution.stderrSections) {
		logWriter.WriteLog(req.TaskID, req.TaskHostID, req.StepKey, "stderr", strings.Join(execution.stderrSections[stderrStart:], "\n"))
	}
}

func (s *DeployService) loadTaskStep(taskID, taskHostID uint64, stepKey string) (DeployTaskStep, error) {
	if s.ownership != nil && (taskID != s.ownership.taskID || taskHostID != s.ownership.taskHostID) {
		s.ownership.fail(errExecutionOwnership)
		return DeployTaskStep{}, errExecutionOwnership
	}
	var checkpoint DeployTaskStep
	ctx, cancel := context.WithTimeout(s.requestContext(), s.ownershipTiming().operation)
	defer cancel()
	err := s.db.WithContext(ctx).Where("task_id = ? AND task_host_id = ? AND step_key = ?", taskID, taskHostID, stepKey).First(&checkpoint).Error
	if s.ownership != nil {
		s.ownership.fail(err)
	}
	return checkpoint, err
}

func (s *DeployService) updateTaskStepStatus(checkpoint *DeployTaskStep, status, errorCode, errorMessage string) error {
	now := time.Now()
	updates := map[string]any{"status": status, "error_code": errorCode, "error_message": truncateDeployLog(errorMessage, 1024), "changed": checkpoint.Changed, "updated_at": now, "state_version": gorm.Expr("state_version + 1")}
	switch status {
	case StepStatusRunning:
		updates["started_at"] = &now
		updates["finished_at"] = nil
		updates["attempt_no"] = gorm.Expr("attempt_no + 1")
	case StepStatusSuccess, StepStatusFailed, StepStatusSkipped, StepStatusCanceled:
		updates["finished_at"] = &now
	}
	err := s.fencedTransaction(checkpoint.TaskID, func(tx *gorm.DB, _ lockedDeployment) error {
		if err := s.requireHostWriter(tx, checkpoint.TaskID, checkpoint.TaskHostID); err != nil {
			return err
		}
		var current DeployTaskStep
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, checkpoint.ID).Error; err != nil {
			return err
		}
		if current.TaskID != checkpoint.TaskID || current.TaskHostID != checkpoint.TaskHostID || (s.ownership != nil && current.TaskHostID != s.ownership.taskHostID) {
			return errExecutionOwnership
		}
		if err := tx.Model(&DeployTaskStep{}).Where(idWhereClause, checkpoint.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.First(&current, checkpoint.ID).Error; err != nil {
			return err
		}
		return appendStepCheckpointHistory(tx, current)
	})
	if err == nil {
		// 更新内存中的 checkpoint 状态以供 metrics 使用
		checkpoint.Status = status
		// 记录步骤结果 metrics
		recordStepResult(checkpoint)
	}
	return err
}

func (s *DeployService) executeOrchestratedSSHHost(ctx context.Context, task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, req StartTaskRequest, actor string) error {
	if err := validateRetireRemoteIntent(task); err != nil {
		return err
	}
	if s.ownership == nil {
		return errExecutionOwnership
	}
	if err := s.validateExecutionHosts(ctx, &task, []uint64{taskHost.HostID}, nil); err != nil {
		return s.markSSHConnectFailure(task, taskHost, err, target.IP, actor)
	}
	steps, err := s.resolveTaskExecutionPlan(task)
	if err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	if err := s.assertCommandOwnership(); err != nil {
		return err
	}
	runner, runnerErr := s.sshRunnerFactory(ctx, target, req)
	if runnerErr != nil {
		return s.markSSHConnectFailure(task, taskHost, runnerErr, target.IP, actor)
	}
	defer func() { _ = runner.Close() }()
	runner = &executionScopeSSHRunner{deploySSHRunner: runner, service: s, task: task, hostID: taskHost.HostID}
	execution := &taskHostExecution{stdoutSections: make([]string, 0, len(steps)), stderrSections: make([]string, 0, len(steps)), summary: deployExecutionSummary{InstalledComponents: make([]cmdb.InstalledComponentUpsert, 0, len(steps)), RemovedComponentNames: make([]string, 0, len(steps))}}
	_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{{"at": time.Now().Format(time.RFC3339), "phase": "connect", "message": "SSH connected"}})
	provider := &sshStepProvider{service: s, task: task, target: target, taskHost: taskHost, runner: runner, execution: execution, steps: make(map[string]deployExecutionStep, len(steps))}
	plan := executorprovider.ExecutionPlan{TaskID: task.ID, TaskHostID: taskHost.ID, TaskName: task.Name, Action: task.Action, ExecutorType: ExecutorTypeSSH, TargetIP: target.IP, TargetPort: target.SSHPort, TargetOS: target.OS, MaxParallelSteps: 1, Actor: actor}
	if err := s.registerOrchestrationSteps(steps, provider, &plan, task.ID, taskHost.ID); err != nil {
		_ = runner.Close()
		return err
	}
	registry := executorprovider.NewRegistry()
	registry.Register(provider)
	result := deployorchestrator.New(registry).RunHostSteps(ctx, plan, s)
	if err := s.persistSkippedTaskSteps(task, taskHost, result, actor); err != nil {
		_ = runner.Close()
		return err
	}
	if result.Status == deployorchestrator.StatusCanceled && s.ownership != nil {
		return context.Cause(ctx)
	}
	applyOrchestrationOutcome(ctx, result, execution)
	return s.finalizeTaskHostExecution(task, taskHost, runner, execution, target.IP, actor)
}

// markSSHConnectFailure records a host result for an SSH runner that never came
// up; a nil error means the host row was marked as failed.
func (s *DeployService) markSSHConnectFailure(task DeployTask, taskHost TaskHostResponse, runnerErr error, ip, actor string) error {
	// A command that interrupts connection setup follows the same pause/cancel
	// handling as an interrupted step; it is not a remote host failure.
	if errors.Is(runnerErr, context.Canceled) {
		if s.taskExecutionSuspended(task.ID) {
			return s.pauseTaskAfterSuspend(task, actor)
		}
		if s.taskCancellationRequested(task.ID) {
			return nil
		}
	}
	_, err := s.MarkHostResult(taskHost.ID, MarkHostResultRequest{
		Status:       TaskHostStatusFailed,
		ErrorMessage: runnerErr.Error(),
		ExecutorID:   fmt.Sprintf("ssh:%s", ip),
	}, actor, nil)
	return err
}

// registerOrchestrationSteps seeds the provider step index and the orchestrator
// plan with the checkpoint status of every plan step.
func (s *DeployService) registerOrchestrationSteps(steps []deployExecutionStep, provider *sshStepProvider, plan *executorprovider.ExecutionPlan, taskID, taskHostID uint64) error {
	for index, step := range steps {
		stepKey := step.StepCode
		if stepKey == "" {
			stepKey = fmt.Sprintf("step-%d", index+1)
			step.StepCode = stepKey
		}
		provider.steps[stepKey] = step
		checkpoint, loadErr := s.loadTaskStep(taskID, taskHostID, stepKey)
		if loadErr != nil {
			return loadErr
		}
		plan.Steps = append(plan.Steps, executorprovider.StepPlan{StepKey: stepKey, StepName: step.StepName, StepType: step.StepType, Action: step.Action, ContinueOnHostFailure: step.ContinueOnHostFailure, PackageName: step.Package.Name, PackageVersion: step.Package.Version, TemplateParams: step.TemplateParams, Dependencies: step.Dependencies, Condition: conditionType(step.Condition), ConditionParams: step.Condition, FailurePolicy: step.FailurePolicy, InitialStatus: checkpoint.Status})
	}
	return nil
}

// interruptOrchestratedHost reports whether a canceled orchestration result was
// caused by a pause or a cancel request; in that case the returned error is the
// caller's return value and the SSH runner is already closed.
func (s *DeployService) interruptOrchestratedHost(task DeployTask, result deployorchestrator.HostExecution, runner deploySSHRunner, actor string) (bool, error) {
	if result.Status != deployorchestrator.StatusCanceled {
		return false, nil
	}
	if s.taskExecutionSuspended(task.ID) {
		_ = runner.Close()
		return true, s.pauseTaskAfterSuspend(task, actor)
	}
	if s.taskCancellationRequested(task.ID) {
		_ = runner.Close()
		return true, nil
	}
	return false, nil
}

// pauseTaskAfterSuspend moves a task from pause-requested to paused.
func (s *DeployService) pauseTaskAfterSuspend(task DeployTask, actor string) error {
	if s.ownership != nil {
		return nil
	}
	return s.recomputeTaskStatus(task.ID, actor)
}

// applyOrchestrationOutcome records the error implied by a failed or canceled
// orchestration result when the step itself reported none.
func applyOrchestrationOutcome(ctx context.Context, result deployorchestrator.HostExecution, execution *taskHostExecution) {
	if result.Status == deployorchestrator.StatusFailed && execution.executionErr == nil {
		execution.executionErr = orchestrationExecutionError(result)
	}
	if result.Status != deployorchestrator.StatusCanceled || execution.executionErr != nil {
		return
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		execution.executionErr = ctxErr
		return
	}
	execution.executionErr = context.Canceled
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
		s.appendTaskEvent(taskEventInput{taskID: task.ID, taskHostID: taskHost.ID, taskStepID: checkpoint.ID, eventType: "step.skipped", fromStatus: checkpoint.Status, toStatus: StepStatusSkipped, actor: actor, reason: "condition evaluated false"})
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
	if err := validateRetireRemoteIntent(task); err != nil {
		return err
	}
	if s.ownership == nil {
		return errExecutionOwnership
	}
	if err := s.validateExecutionHosts(ctx, &task, []uint64{taskHost.HostID}, nil); err != nil {
		return s.markSSHConnectFailure(task, taskHost, err, target.IP, actor)
	}
	plan, err := s.resolveTaskExecutionPlan(task)
	if err != nil {
		return mapDeployTaskExecutionPlanError(err)
	}
	if err := s.assertCommandOwnership(); err != nil {
		return err
	}
	runner, runnerErr := s.sshRunnerFactory(ctx, target, req)
	if runnerErr != nil {
		return s.markSSHConnectFailure(task, taskHost, runnerErr, target.IP, actor)
	}
	defer func() { _ = runner.Close() }()
	runner = &executionScopeSSHRunner{deploySSHRunner: runner, service: s, task: task, hostID: taskHost.HostID}

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
	run := taskStepHostRun{task: task, target: target, taskHost: taskHost, runner: runner, execution: execution}
	for _, step := range plan {
		if stop, _ := s.executeTaskStep(ctx, run, step); stop {
			break
		}
	}
	return s.finalizeTaskHostExecution(task, taskHost, runner, execution, target.IP, actor)
}

// taskStepHostRun carries the host-scoped state shared by every plan step of a
// single host run.
type taskStepHostRun struct {
	task      DeployTask
	target    cmdbHostSnapshot
	taskHost  TaskHostResponse
	runner    deploySSHRunner
	execution *taskHostExecution
}

// executeTaskStep runs a single plan step and records its trace. It returns
// true when the host loop must stop on an unrecoverable failure.
func (s *DeployService) executeTaskStep(ctx context.Context, run taskStepHostRun, step deployExecutionStep) (bool, bool) {
	task, target := run.task, run.target
	taskHost, runner, execution := run.taskHost, run.runner, run.execution
	if normalizeTaskAction(task.Action) == TaskActionRetire || normalizeTaskAction(step.Action) == TaskActionRetire {
		execution.executionErr = errors.New(errDeployTaskTemplateInvalid)
		return true, false
	}
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
	if s.runTaskStepCheck(ctx, run, step, taskStepCheckConfig{configKey: "precheckCommand", phase: "precheck", stepLabel: stepLabel}) {
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
	if s.runTaskStepCheck(ctx, run, step, taskStepCheckConfig{configKey: "postcheckCommand", phase: "postcheck", stepLabel: stepLabel}) {
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
func (s *DeployService) runTaskStepCheck(ctx context.Context, run taskStepHostRun, step deployExecutionStep, check taskStepCheckConfig) bool {
	task, taskHost, execution := run.task, run.taskHost, run.execution
	if normalizeTaskAction(task.Action) == TaskActionRetire || normalizeTaskAction(step.Action) == TaskActionRetire {
		execution.executionErr = errors.New(errDeployTaskTemplateInvalid)
		return true
	}
	script, hasCheck, checkErr := renderDeployCheckSnippet(step, task, run.target, check.configKey)
	if checkErr != nil {
		execution.executionErr = mapDeployTaskExecutionPlanError(checkErr)
		s.appendStepTrace(task, taskHost, step, check.phase+"_render_failed", checkErr.Error())
		return true
	}
	if !hasCheck {
		return false
	}
	s.appendStepTrace(task, taskHost, step, check.phase, fmt.Sprintf("%s %s started", check.stepLabel, check.phase))
	stdout, stderr, execErr := runDeployScript(ctx, run.runner, script)
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
	if err := validateRetireSnapshot(task, snapshot); err != nil {
		return nil, err
	}
	if snapshot.Kind == deploySnapshotMetadataOnly {
		return []deployExecutionStep{}, nil
	}
	if len(snapshot.Steps) == 0 {
		return nil, errors.New(errDeployTaskSnapshotMissing)
	}
	return executionSnapshotToPlan(snapshot), nil
}

const deploySnapshotMetadataOnly = "metadata_only"

func retireMetadataExecutionSnapshot(template *TemplateResponse) deployExecutionSnapshot {
	snapshot := deployExecutionSnapshot{Kind: deploySnapshotMetadataOnly, Steps: []deployStepSnapshot{}}
	if template != nil {
		snapshot.TemplateID, snapshot.TemplateName, snapshot.TemplateVersion = template.ID, template.Name, template.Version
	}
	return snapshot
}

func retireParamsHaveExecution(params map[string]any) bool {
	for _, key := range []string{"script", "platformScripts", "installCommand", "uninstallCommand", "precheckCommand", "postcheckCommand", "healthCheckCommand", "uninstallHealthCheckCommand"} {
		if value, exists := params[key]; exists && value != nil && strings.TrimSpace(anyToString(value)) != "" {
			return true
		}
	}
	return false
}

func validateRetireSnapshot(task DeployTask, snapshot deployExecutionSnapshot) error {
	if normalizeTaskAction(task.Action) == TaskActionRetire || snapshot.Kind != "" {
		if normalizeTaskAction(task.Action) != TaskActionRetire || task.ExecutorType != ExecutorTypeManual || task.ServiceID == 0 || task.ServiceInstanceID == 0 || snapshot.Kind != deploySnapshotMetadataOnly || snapshot.Steps == nil || len(snapshot.Steps) != 0 || retireParamsHaveExecution(decodeJSONMap(task.TemplateParams)) {
			return errors.New(errDeployTaskTemplateInvalid)
		}
		return nil
	}
	for _, step := range snapshot.Steps {
		if normalizeTaskAction(step.Action) == TaskActionRetire {
			return errors.New(errDeployTaskTemplateInvalid)
		}
	}
	return nil
}

func validateRetireTaskIntent(task DeployTask) error {
	snapshot, err := decodeExecutionSnapshot(task.ExecutionSnapshot)
	if err != nil {
		return err
	}
	return validateRetireSnapshot(task, snapshot)
}

// Metadata can reserve a host for manual coordination, but never claim a worker.
// Ordinary legacy tasks retain their existing missing-snapshot worker contract.
func validateRetireRemoteIntent(task DeployTask) error {
	if normalizeTaskAction(task.Action) == TaskActionRetire {
		return errors.New(errDeployTaskTemplateInvalid)
	}
	return validateRetireTaskIntent(task)
}

func validateRetireResult(task DeployTask, summary deployExecutionSummary) error {
	if err := validateRetireTaskIntent(task); err != nil {
		return err
	}
	if len(summary.InstalledComponents) != 0 || len(summary.RemovedComponentNames) != 0 {
		return errors.New(errDeployTaskTemplateInvalid)
	}
	return nil
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
	if normalizedAction == TaskActionRetire {
		return deployExecutionSnapshot{}, errors.New(errDeployTaskTemplateInvalid)
	}
	if template == nil || template.ID == 0 {
		steps, err := s.defaultExecutionSnapshotSteps(pkg, deployStepSnapshotPackageDefault, pkg.Name, normalizedAction, taskParams)
		if err != nil {
			return deployExecutionSnapshot{}, err
		}
		snapshot.Steps = steps
		return snapshot, nil
	}
	if len(template.Steps) == 0 {
		steps, err := s.defaultExecutionSnapshotSteps(pkg, deployStepSnapshotTemplateDefault, template.Name, normalizedAction, taskParams)
		if err != nil {
			return deployExecutionSnapshot{}, err
		}
		snapshot.Steps = steps
		return snapshot, nil
	}
	cache := map[uint64]DeployPackage{}
	steps := make([]deployStepSnapshot, 0, len(template.Steps))
	for _, step := range template.Steps {
		snapshotStep, err := s.buildTemplateStepSnapshot(step, pkg, normalizedAction, taskParams, cache)
		if err != nil {
			return deployExecutionSnapshot{}, err
		}
		steps = append(steps, snapshotStep)
	}
	if err := validateDeploySnapshotDAG(steps); err != nil {
		return deployExecutionSnapshot{}, err
	}
	snapshot.Steps = steps
	return snapshot, nil
}

// Step codes of the fallback snapshots used when a task has no executable
// template steps: the package itself, then the template definition.
const (
	deployStepSnapshotPackageDefault  = "package_default"
	deployStepSnapshotTemplateDefault = "template_default"
)

// defaultExecutionSnapshotSteps builds the single fallback step and validates
// the request parameters against the resolved package.
func (s *DeployService) defaultExecutionSnapshotSteps(pkg DeployPackage, stepCode, stepName, normalizedAction string, taskParams map[string]any) ([]deployStepSnapshot, error) {
	// The task action is the effective action; parameter maps do not always
	// repeat it, so validate with it injected before freezing the snapshot.
	validationParams := cloneDeployConfig(taskParams)
	validationParams["action"] = normalizedAction
	if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, validationParams); err != nil {
		return nil, err
	}
	return []deployStepSnapshot{{
		StepCode:       stepCode,
		StepName:       stepName,
		StepType:       TemplateStepTypePackage,
		Action:         normalizedAction,
		Package:        packageToSnapshot(pkg),
		TemplateParams: taskParams,
	}}, nil
}

// buildTemplateStepSnapshot freezes a single template step: its type, effective
// action, resolved package and validated parameters.
func (s *DeployService) buildTemplateStepSnapshot(step TemplateStepResponse, pkg DeployPackage, normalizedAction string, taskParams map[string]any, cache map[uint64]DeployPackage) (deployStepSnapshot, error) {
	if normalizeTaskAction(step.Action) == TaskActionRetire || normalizedAction == TaskActionRetire {
		return deployStepSnapshot{}, errors.New(errDeployTaskTemplateInvalid)
	}
	stepType := strings.TrimSpace(step.StepType)
	if stepType == "" {
		stepType = TemplateStepTypePackage
	}
	if !validStepType(stepType) {
		return deployStepSnapshot{}, errors.New(errDeployTaskTemplateInvalid)
	}
	action := normalizedAction
	if action == "" {
		action = normalizeTaskAction(step.Action)
	}
	params := mergeDeployTemplateParams(step.TemplateParams, taskParams)
	stepPackage, err := s.resolveStepSnapshotPackage(step, stepType, pkg, cache)
	if err != nil {
		return deployStepSnapshot{}, err
	}
	if err := validateStepSnapshotParams(stepType, stepPackage, params, action); err != nil {
		return deployStepSnapshot{}, err
	}
	return deployStepSnapshot{
		StepCode:              step.StepCode,
		StepName:              step.StepName,
		StepType:              stepType,
		Action:                action,
		ContinueOnHostFailure: step.ContinueOnHostFailure,
		Package:               packageToSnapshot(stepPackage),
		TemplateParams:        params,
		StepConfig:            step.StepConfig,
		Dependencies:          stringSliceConfig(step.StepConfig, "dependencies"),
		Condition:             mapConfig(step.StepConfig, "condition"),
		FailurePolicy:         effectiveFailurePolicy(stringConfig(step.StepConfig, "failurePolicy"), step.ContinueOnHostFailure),
	}, nil
}

// resolveStepSnapshotPackage returns the package a frozen step needs, reusing
// the per-plan lookup cache. A step that is neither a package nor a component
// type and carries no package reference keeps the zero-value package.
func (s *DeployService) resolveStepSnapshotPackage(step TemplateStepResponse, stepType string, pkg DeployPackage, cache map[uint64]DeployPackage) (DeployPackage, error) {
	if stepType != TemplateStepTypePackage && !isComponentType(stepType) && step.PackageID == 0 {
		return DeployPackage{}, nil
	}
	resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, pkg, cache)
	if err != nil {
		return DeployPackage{}, err
	}
	cache[resolvedPackage.ID] = resolvedPackage
	return resolvedPackage, nil
}

// validateStepSnapshotParams applies the parameter contract of the step type:
// package steps validate against the resolved package, component steps against
// their fixed schema, other step types have no parameter contract.
func validateStepSnapshotParams(stepType string, stepPackage DeployPackage, params map[string]any, action string) error {
	if stepType == TemplateStepTypePackage {
		validationParams := cloneDeployConfig(params)
		validationParams["action"] = action
		return validateTemplateParams(stepPackage.ExecutionMode, stepPackage.TemplateCode, stepPackage.TemplateConfig, validationParams)
	}
	if isComponentType(stepType) {
		componentParams := cloneDeployConfig(params)
		componentParams["action"] = action
		return validateFixedTemplateParams(componentTypeToTemplateCode[stepType], componentParams)
	}
	return nil
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
	if normalizeTaskAction(task.Action) == TaskActionRetire || normalizeTaskAction(step.Action) == TaskActionRetire {
		return "", errors.New(errDeployTaskTemplateInvalid)
	}
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
		if s.ownership != nil {
			s.ownership.fail(err)
		}
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
	if normalizeTaskAction(task.Action) == TaskActionRetire || normalizeTaskAction(step.Action) == TaskActionRetire {
		return "", false, errors.New(errDeployTaskTemplateInvalid)
	}
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
	if s.ownership != nil && (taskID != s.ownership.taskID || hostID != s.ownership.physicalHostID) {
		s.ownership.fail(errExecutionOwnership)
		return errExecutionOwnership
	}
	project := func(tx *gorm.DB) error {
		provider, err := s.boundCMDB(tx, []uint64{hostID})
		if err != nil {
			return err
		}
		return provider.WriteDeployHostResult(cmdb.DeployHostWritebackRequest{HostID: hostID, Actor: actor, InstalledComponents: []cmdb.InstalledComponentUpsert{{Name: pkg.name, Version: pkg.version, DeployedAt: now, DeployTaskID: taskID, DeployTaskName: taskName, ExecutorType: executorType}}})
	}
	if s.ownership != nil {
		return s.fencedTransaction(taskID, func(tx *gorm.DB, _ lockedDeployment) error { return project(tx) })
	}
	return s.deploymentTransaction(func(tx *gorm.DB) error {
		// Retained legacy component reporting may reference a historical task
		// absent here. Present task rows precede all host/result locks.
		var tasks []DeployTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", taskID).Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) > 0 {
			state, err := lockDeployment(tx, taskID)
			if err != nil {
				return err
			}
			// Do not acquire a new, potentially lower physical key after the
			// task's ordered set. Legacy tasks without hosts use the single key.
			if len(state.hosts) > 0 {
				found := false
				for _, h := range state.hosts {
					found = found || h.HostID == hostID
				}
				if !found {
					return errExecutionOwnership
				}
			}
		}
		if err := s.requireHostWriter(tx, taskID, 0); err != nil {
			return err
		}
		// Even a missing historical task must serialize with new admissions on
		// the unique physical key before locking/projecting the CMDB host.
		now, err := databaseNow(tx)
		if err != nil {
			return err
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&DeployHostLease{HostID: hostID, ExpiresAt: now, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			return err
		}
		var lease DeployHostLease
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("host_id = ?", hostID).First(&lease).Error; err != nil {
			return err
		}
		now, err = databaseNow(tx)
		if err != nil {
			return err
		}
		if lease.ExpiresAt.After(now) && (lease.TaskID != taskID || lease.Owner != deployHostLeaseOwner(taskID)) {
			return errExecutionOwnership
		}
		return project(tx)
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
	for i := range hosts {
		hosts[i].ExecutionVersion = taskExecutionVersion(task)
	}
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

	var targetSnapshot deployTargetSnapshot
	_ = json.Unmarshal(task.TargetSnapshot, &targetSnapshot)
	return TaskResponse{
		TargetSelection: targetSnapshot.Selection, ScopeVersion: targetSnapshot.ScopeVersion,
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
	if normalizeTaskAction(task.Action) == TaskActionRetire {
		return "", errors.New(errDeployTaskTemplateInvalid)
	}
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
			serviceName = defaultNginxServiceName
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
	if s.ownership != nil && (taskID != s.ownership.taskID || taskHostID != s.ownership.taskHostID) {
		s.ownership.fail(errExecutionOwnership)
		return errExecutionOwnership
	}
	if len(steps) == 0 {
		return nil
	}
	return s.fencedTransaction(taskID, func(tx *gorm.DB, _ lockedDeployment) error {
		if err := s.requireHostWriter(tx, taskID, taskHostID); err != nil {
			return err
		}
		return appendHostTraceTx(tx, taskID, taskHostID, steps)
	})
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

func (s *DeployService) loadTaskHostsByTaskIDs(taskIDs []uint64, dataScope *common.DataScopeReq) (map[uint64][]TaskHostResponse, error) {
	result := make(map[uint64][]TaskHostResponse, len(taskIDs))
	if len(taskIDs) == 0 {
		return result, nil
	}
	var hosts []DeployTaskHost
	if err := s.db.Where("task_id IN ?", taskIDs).Order("task_id ASC, id ASC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	// SEC-002: one task can contain hosts outside the caller's scope; never expose
	// their IP/stdout/stderr just because another host in the task is visible.
	visible, err := s.visibleTaskHosts(hosts, dataScope)
	if err != nil {
		return nil, err
	}
	for _, host := range visible {
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
