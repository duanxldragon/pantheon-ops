package deploy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"pantheon-base/modules/business/cmdb"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"golang.org/x/crypto/ssh"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DeployService struct {
	db               *gorm.DB
	cmdbCapability   cmdb.DeployCMDBCapability
	sshRunnerFactory func(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error)
}

type deployExecutionStep struct {
	Index          int
	StepCode       string
	StepName       string
	StepType       string
	Action         string
	Package        DeployPackage
	TemplateParams map[string]any
	StepConfig     map[string]any
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
)

const (
	idDescOrder       = "id DESC"
	taskIDWhereClause = "task_id = ?"
	idWhereClause     = "id = ?"
	statusWhereClause = "status = ?"
)

func NewDeployService(db *gorm.DB, cmdbCapability cmdb.DeployCMDBCapability) *DeployService {
	return &DeployService{
		db:               db,
		cmdbCapability:   cmdbCapability,
		sshRunnerFactory: newDeploySSHRunner,
	}
}

func (s *DeployService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&DeployPackage{}, &DeployTemplate{}, &DeployTemplateStep{}, &DeployTask{}, &DeployTaskHost{})
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
	template, err := s.applyTemplateToCreateRequest(&req)
	if err != nil {
		return nil, err
	}
	action := normalizeTaskAction(req.Action)
	if !validTaskAction(action) {
		return nil, errors.New("business.deploy.task.invalidAction")
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
	var count int64
	if err := s.db.Table("biz_business_scope").Where("id = ? AND status = ? AND deleted_at IS NULL", businessScopeID, "active").Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
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
	scopeName := ""
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
	templateParamsJSON, _ := json.Marshal(req.TemplateParams)
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
		Action:            action,
		TargetType:        req.TargetType,
		TargetIDs:         datatypes.JSON(targetJSON),
		ExecutorType:      req.ExecutorType,
		ExecutionMode:     pkg.ExecutionMode,
		TemplateParams:    datatypes.JSON(templateParamsJSON),
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
	return &resp, nil
}

type taskUpdateValues struct {
	name            string
	templateID      uint64
	packageID       uint64
	businessScopeID uint64
	action          string
	targetType      string
	targetIDs       []uint64
	executorType    string
	templateParams  map[string]any
	remark          string
}

func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusPending && task.Status != TaskStatusDraft {
		return nil, errors.New(errDeployTaskInvalidUpdateState)
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
	return s.GetTask(id, dataScope)
}

func (s *DeployService) buildTaskUpdateValues(task *DeployTask, req UpdateTaskRequest) (taskUpdateValues, error) {
	values := taskUpdateValues{
		name:            task.Name,
		templateID:      task.TemplateID,
		packageID:       task.PackageID,
		businessScopeID: task.BusinessScopeID,
		action:          normalizeTaskAction(task.Action),
		targetType:      task.TargetType,
		targetIDs:       parseUint64JSON(task.TargetIDs),
		executorType:    task.ExecutorType,
		templateParams:  decodeJSONMap(task.TemplateParams),
		remark:          task.Remark,
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
		"action":              values.action,
		"target_type":         values.targetType,
		"target_ids":          datatypes.JSON(targetJSON),
		"executor_type":       values.executorType,
		"execution_mode":      pkg.ExecutionMode,
		"template_params":     datatypes.JSON(templateParamsJSON),
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
	if task.Status != TaskStatusDraft && task.Status != TaskStatusPending {
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
	if task.Status != TaskStatusDraft && task.Status != TaskStatusPending {
		return nil, errors.New("business.deploy.task.invalidStartState")
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, task.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(errDeployTaskPackageNotFound)
		}
		return nil, err
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
		if err := s.validateTaskExecutionPlan(*task, pkg, hosts); err != nil {
			return nil, mapDeployTaskExecutionPlanError(err)
		}
	}
	now := time.Now()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(task).Updates(map[string]interface{}{
			"status":     TaskStatusRunning,
			"started_at": &now,
			"updated_by": actor,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		for _, host := range hosts {
			initialTrace, _ := json.Marshal([]map[string]any{
				{"at": now.Format(time.RFC3339), "phase": "start", "message": "Task host queued"},
			})
			row := DeployTaskHost{
				TaskID:     task.ID,
				HostID:     host.ID,
				Hostname:   host.Hostname,
				HostIP:     host.IP,
				OS:         host.OS,
				Status:     TaskHostStatusRunning,
				TraceSteps: datatypes.JSON(initialTrace),
				StartedAt:  &now,
				UpdatedBy:  actor,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if task.ExecutorType == ExecutorTypeSSH {
		if err := s.executeSSHTask(*task, pkg, hosts, req, actor); err != nil {
			return nil, err
		}
	}
	return s.GetTask(id, dataScope)
}

func (s *DeployService) CancelTask(id uint64, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	now := time.Now()
	task, err := s.loadVisibleTask(id, dataScope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusPending && task.Status != TaskStatusRunning {
		return nil, errors.New("business.deploy.task.invalidCancelState")
	}
	if err := s.db.Model(task).Updates(map[string]interface{}{
		"status":      TaskStatusCanceled,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error; err != nil {
		return nil, err
	}
	_ = s.db.Model(&DeployTaskHost{}).Where("task_id = ? AND status IN ?", id, []string{TaskHostStatusPending, TaskHostStatusRunning}).Updates(map[string]interface{}{
		"status":      TaskHostStatusSkipped,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error
	return s.GetTask(id, dataScope)
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
	now := time.Now()
	updates := map[string]interface{}{
		"status":        req.Status,
		"stdout":        req.Stdout,
		"stderr":        req.Stderr,
		"error_message": req.ErrorMessage,
		"executor_id":   req.ExecutorID,
		"finished_at":   &now,
		"reported_at":   &now,
		"updated_by":    actor,
		"updated_at":    now,
	}
	if err := s.db.Model(&host).Updates(updates).Error; err != nil {
		return nil, err
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
	if err := s.db.First(&host, hostID).Error; err != nil {
		return nil, err
	}
	resp := taskHostToResponse(&host)
	return &resp, nil
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
	var visibleCount int64
	err := s.db.Table("biz_deploy_task_host").
		Joins("JOIN biz_cmdb_host ON biz_cmdb_host.id = biz_deploy_task_host.host_id").
		Where("biz_deploy_task_host.task_id = ? AND biz_cmdb_host.deleted_at IS NULL", taskID).
		Scopes(database.WithDataScope(dataScope)).
		Count(&visibleCount).Error
	if err != nil {
		return false, true, err
	}
	return visibleCount > 0, true, nil
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
	status := TaskStatusSuccess
	if hasFailed {
		status = TaskStatusFailed
	}
	if allSkipped {
		status = TaskStatusCanceled
	}
	now := time.Now()
	return s.db.Model(&DeployTask{}).Where(idWhereClause, taskID).Updates(map[string]interface{}{
		"status":      status,
		"finished_at": &now,
		"updated_by":  actor,
		"updated_at":  now,
	}).Error
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
	default:
		return strings.TrimSpace(action)
	}
}

func validTaskAction(action string) bool {
	return action == TaskActionInstall || action == TaskActionUninstall || action == TaskActionUpgrade || action == TaskActionReinstall
}

func hostStatusForAction(action string) string {
	switch normalizeTaskAction(action) {
	case TaskActionUninstall:
		return "assigned"
	default:
		return "online"
	}
}

func hostStatusAllowedForAction(status, action string) bool {
	switch normalizeTaskAction(action) {
	case TaskActionUninstall, TaskActionUpgrade:
		return strings.TrimSpace(status) == "online"
	case TaskActionReinstall:
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

type deploySSHRunner interface {
	RunScript(script string) (stdout string, stderr string, err error)
	Close() error
}

type deploySSHClient struct {
	client *ssh.Client
}

func (c *deploySSHClient) RunScript(script string) (string, string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	session.Stdin = strings.NewReader(script)
	err = session.Run("/bin/bash -se")
	return stdout.String(), stderr.String(), err
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

func (s *DeployService) executeSSHTask(task DeployTask, pkg DeployPackage, hosts []cmdbHostSnapshot, req StartTaskRequest, actor string) error {
	plan, err := s.resolveTaskExecutionPlan(task, pkg)
	if err != nil {
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

	for _, target := range hosts {
		if err := s.executeTaskHost(task, target, taskHostsByID, plan, req, actor); err != nil {
			return err
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
func (s *DeployService) executeTaskHost(task DeployTask, target cmdbHostSnapshot, taskHostsByID map[uint64]TaskHostResponse, plan []deployExecutionStep, req StartTaskRequest, actor string) error {
	taskHost, ok := taskHostsByID[target.ID]
	if !ok {
		return nil
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
	s.runTaskExecutionSteps(task, target, taskHost, runner, plan, execution)
	return s.finalizeTaskHostExecution(task, taskHost, runner, execution, target.IP, actor)
}

// runTaskExecutionSteps runs the plan steps for one host, stopping at the first
// unrecoverable step failure.
func (s *DeployService) runTaskExecutionSteps(task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, plan []deployExecutionStep, execution *taskHostExecution) {
	for _, step := range plan {
		if s.executeTaskStep(task, target, taskHost, runner, step, execution) {
			break
		}
	}
}

// executeTaskStep runs a single plan step and records its trace. It returns
// true when the host loop must stop on an unrecoverable failure.
func (s *DeployService) executeTaskStep(task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, step deployExecutionStep, execution *taskHostExecution) bool {
	stepLabel := buildDeployStepLabel(step)
	script, renderErr := s.renderExecutionStepScript(step, task, target)
	if renderErr != nil {
		execution.executionErr = mapDeployTaskExecutionPlanError(renderErr)
		s.appendStepTrace(task, taskHost, step, "render_failed", renderErr.Error())
		return true
	}
	s.appendStepTrace(task, taskHost, step, "step_start", fmt.Sprintf("%s started", stepLabel))
	if s.runTaskStepCheck(task, target, taskHost, runner, step, execution, taskStepCheckConfig{configKey: "precheckCommand", phase: "precheck", stepLabel: stepLabel}) {
		return true
	}
	s.appendStepTrace(task, taskHost, step, "script", fmt.Sprintf("%s script rendered", stepLabel))
	stdout, stderr, execErr := runner.RunScript(script)
	execution.recordOutput(step, "script", stdout, stderr)
	if execErr != nil {
		execution.executionErr = execErr
		s.appendStepTrace(task, taskHost, step, "step_failed", execErr.Error())
		return true
	}
	if s.runTaskStepCheck(task, target, taskHost, runner, step, execution, taskStepCheckConfig{configKey: "postcheckCommand", phase: "postcheck", stepLabel: stepLabel}) {
		return true
	}
	s.appendStepTrace(task, taskHost, step, "step_success", fmt.Sprintf("%s completed", stepLabel))
	execution.recordComponent(step, task)
	return false
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
func (s *DeployService) runTaskStepCheck(task DeployTask, target cmdbHostSnapshot, taskHost TaskHostResponse, runner deploySSHRunner, step deployExecutionStep, execution *taskHostExecution, check taskStepCheckConfig) bool {
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
	stdout, stderr, execErr := runner.RunScript(script)
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
	errorMessage := combineTaskExecutionErrors(execution.executionErr, closeErr)
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

func (s *DeployService) resolveTaskExecutionPlan(task DeployTask, fallbackPackage DeployPackage) ([]deployExecutionStep, error) {
	taskParams := decodeJSONMap(task.TemplateParams)
	if task.TemplateID == 0 {
		if err := validateTemplateParams(fallbackPackage.ExecutionMode, fallbackPackage.TemplateCode, fallbackPackage.TemplateConfig, taskParams); err != nil {
			return nil, err
		}
		return []deployExecutionStep{{
			Index:          0,
			StepCode:       "package_default",
			StepName:       fallbackPackage.Name,
			StepType:       TemplateStepTypePackage,
			Action:         normalizeTaskAction(task.Action),
			Package:        fallbackPackage,
			TemplateParams: taskParams,
		}}, nil
	}
	template, err := s.GetTemplate(task.TemplateID)
	if err != nil {
		return nil, err
	}
	steps := template.Steps
	if len(steps) == 0 {
		if err := validateTemplateParams(fallbackPackage.ExecutionMode, fallbackPackage.TemplateCode, fallbackPackage.TemplateConfig, taskParams); err != nil {
			return nil, err
		}
		return []deployExecutionStep{{
			Index:          0,
			StepCode:       "template_default",
			StepName:       template.Name,
			StepType:       TemplateStepTypePackage,
			Action:         normalizeTaskAction(task.Action),
			Package:        fallbackPackage,
			TemplateParams: taskParams,
		}}, nil
	}
	cache := map[uint64]DeployPackage{}
	plan := make([]deployExecutionStep, 0, len(steps))
	for index, step := range steps {
		stepType := strings.TrimSpace(step.StepType)
		if stepType == "" {
			stepType = TemplateStepTypePackage
		}
		if stepType != TemplateStepTypePackage && stepType != TemplateStepTypeScript {
			return nil, errors.New(errDeployTaskTemplateInvalid)
		}
		effectiveAction := normalizeTaskAction(task.Action)
		if effectiveAction == "" {
			effectiveAction = normalizeTaskAction(step.Action)
		}
		params := mergeDeployTemplateParams(step.TemplateParams, taskParams)
		stepConfig := step.StepConfig
		stepPackage := DeployPackage{}
		if stepType == TemplateStepTypePackage {
			resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, fallbackPackage, cache)
			if err != nil {
				return nil, err
			}
			cache[resolvedPackage.ID] = resolvedPackage
			stepPackage = resolvedPackage
			if err := validateTemplateParams(stepPackage.ExecutionMode, stepPackage.TemplateCode, stepPackage.TemplateConfig, params); err != nil {
				return nil, err
			}
		} else if step.PackageID > 0 {
			resolvedPackage, err := s.resolveExecutionPackage(step.PackageID, fallbackPackage, cache)
			if err != nil {
				return nil, err
			}
			cache[resolvedPackage.ID] = resolvedPackage
			stepPackage = resolvedPackage
		}
		plan = append(plan, deployExecutionStep{
			Index:          index,
			StepCode:       step.StepCode,
			StepName:       step.StepName,
			StepType:       stepType,
			Action:         effectiveAction,
			Package:        stepPackage,
			TemplateParams: params,
			StepConfig:     stepConfig,
		})
	}
	return plan, nil
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
		if script == "" {
			return "", errors.New(errDeployTaskTemplateInvalid)
		}
		return renderTemplateSnippet(script, buildDeployRenderContext(step, task, host))
	}
	paramsJSON, _ := json.Marshal(step.TemplateParams)
	stepTask := task
	stepTask.Action = step.Action
	stepTask.PackageID = step.Package.ID
	stepTask.PackageName = step.Package.Name
	stepTask.PackageVersion = step.Package.Version
	stepTask.TemplateParams = datatypes.JSON(paramsJSON)
	return s.resolveInstallScript(step.Package, stepTask)
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

// deployPackageRef names a package for host installed-component bookkeeping.
type deployPackageRef struct {
	name    string
	version string
}

func (s *DeployService) upsertHostInstalledComponent(hostID, taskID uint64, taskName, executorType string, pkg deployPackageRef, actor string, now time.Time) error {
	if strings.TrimSpace(pkg.name) == "" {
		return nil
	}
	var snapshot struct {
		InstalledComponents datatypes.JSON `gorm:"column:installed_components"`
	}
	if err := s.db.Table("biz_cmdb_host").Select("installed_components").Where(idWhereClause, hostID).Take(&snapshot).Error; err != nil {
		return err
	}
	var components []deployInstalledComponent
	payload := snapshot.InstalledComponents
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &components)
	}
	if components == nil {
		components = []deployInstalledComponent{}
	}
	updated := false
	for index := range components {
		if strings.EqualFold(strings.TrimSpace(components[index].Name), strings.TrimSpace(pkg.name)) {
			components[index].Version = pkg.version
			components[index].DeployedAt = now.Format(time.RFC3339)
			components[index].DeployTaskID = taskID
			components[index].DeployTaskName = taskName
			components[index].ExecutorType = executorType
			updated = true
			break
		}
	}
	if !updated {
		components = append(components, deployInstalledComponent{
			Name:           pkg.name,
			Version:        pkg.version,
			DeployedAt:     now.Format(time.RFC3339),
			DeployTaskID:   taskID,
			DeployTaskName: taskName,
			ExecutorType:   executorType,
		})
	}
	nextPayload, _ := json.Marshal(components)
	return s.db.Table("biz_cmdb_host").Where(idWhereClause, hostID).Updates(map[string]interface{}{
		"installed_components": datatypes.JSON(nextPayload),
		"updated_by":           actor,
		"updated_at":           now,
	}).Error
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
	return TaskResponse{
		ID:                task.ID,
		Name:              task.Name,
		TemplateID:        task.TemplateID,
		TemplateName:      task.TemplateName,
		TemplateVersion:   task.TemplateVersion,
		PackageID:         task.PackageID,
		PackageName:       task.PackageName,
		PackageVersion:    task.PackageVersion,
		BusinessScopeID:   task.BusinessScopeID,
		BusinessScopeName: task.BusinessScopeName,
		Action:            normalizeTaskAction(task.Action),
		TargetType:        task.TargetType,
		TargetIDs:         parseUint64JSON(task.TargetIDs),
		ExecutorType:      task.ExecutorType,
		ExecutionMode:     task.ExecutionMode,
		TemplateParams:    decodeJSONMap(task.TemplateParams),
		Status:            task.Status,
		Remark:            task.Remark,
		ExternalTaskID:    task.ExternalTaskID,
		StartedAt:         task.StartedAt,
		FinishedAt:        task.FinishedAt,
		HostCount:         hostCount,
		SuccessCount:      successCount,
		FailedCount:       failedCount,
		RunningCount:      runningCount,
		SkippedCount:      skippedCount,
		DurationSeconds:   durationSeconds,
		CreatedAt:         task.CreatedAt,
		UpdatedAt:         task.UpdatedAt,
		CreatedBy:         task.CreatedBy,
		UpdatedBy:         task.UpdatedBy,
		Hosts:             hosts,
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
		OS:              host.OS,
		Status:          host.Status,
		Stdout:          host.Stdout,
		Stderr:          host.Stderr,
		ErrorMessage:    host.ErrorMessage,
		ExecutorID:      host.ExecutorID,
		TraceSteps:      traceSteps,
		StartedAt:       host.StartedAt,
		FinishedAt:      host.FinishedAt,
		ReportedAt:      host.ReportedAt,
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

func (s *DeployService) validateTaskExecutionPlan(task DeployTask, fallbackPackage DeployPackage, hosts []cmdbHostSnapshot) error {
	plan, err := s.resolveTaskExecutionPlan(task, fallbackPackage)
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
	version := strings.TrimSpace(pkg.Version)
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
