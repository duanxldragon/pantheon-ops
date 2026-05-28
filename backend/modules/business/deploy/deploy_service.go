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

	"pantheon-ops/backend/modules/business/cmdb"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

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
		db = db.Where("status = ?", strings.TrimSpace(query.Status))
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
	if err := db.Order("id DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
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
	var item DeployPackage
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploypackage.not_found")
		}
		return nil, err
	}
	name := item.Name
	version := item.Version
	updates := map[string]interface{}{"updated_by": actor, "updated_at": time.Now()}
	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
		updates["name"] = name
	}
	if req.Version != nil {
		version = strings.TrimSpace(*req.Version)
		updates["version"] = version
	}
	if name == "" || version == "" {
		return nil, errors.New("deploypackage.invalid")
	}
	if s.packageExists(name, version, id) {
		return nil, errors.New("deploypackage.exists")
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
	if req.ExecutionMode != nil {
		if !validExecutionMode(*req.ExecutionMode) {
			return nil, errors.New("deploypackage.execution_mode_invalid")
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
	if err := validateTemplateDefinition(nextExecutionMode, nextTemplateCode, nextTemplateConfig); err != nil {
		return nil, err
	}
	if req.Status != nil {
		if !validPackageStatus(*req.Status) {
			return nil, errors.New("deploypackage.status_invalid")
		}
		updates["status"] = *req.Status
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(common.NormalizeUint64IDs(req.TargetIDs)) == 0 {
		return nil, errors.New("deploytask.invalid")
	}
	if !validTargetType(req.TargetType) {
		return nil, errors.New("deploytask.target_invalid")
	}
	if req.ExecutorType == "" {
		req.ExecutorType = ExecutorTypeManual
	}
	if !validExecutorType(req.ExecutorType) {
		return nil, errors.New("deploytask.executor_invalid")
	}
	var pkg DeployPackage
	var template *TemplateResponse
	if req.TemplateID > 0 {
		templateDetail, err := s.GetTemplate(req.TemplateID)
		if err != nil {
			return nil, err
		}
		if templateDetail.Status != TemplateStatusEnabled {
			return nil, errors.New("deploytemplate.disabled")
		}
		template = templateDetail
		if req.PackageID == 0 {
			req.PackageID = template.PackageID
			if req.PackageID == 0 && len(template.Steps) > 0 {
				req.PackageID = template.Steps[0].PackageID
			}
		}
		if strings.TrimSpace(req.Action) == "" {
			req.Action = template.DefaultAction
		}
		if len(req.TemplateParams) == 0 && len(template.ParameterSchema) > 0 {
			req.TemplateParams = template.ParameterSchema
		}
	}
	action := normalizeTaskAction(req.Action)
	if !validTaskAction(action) {
		return nil, errors.New("deploytask.action_invalid")
	}
	if req.PackageID == 0 {
		return nil, errors.New("deploytask.invalid")
	}
	if err := s.db.First(&pkg, req.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploypackage.not_found")
		}
		return nil, err
	}
	if pkg.Status != PackageStatusEnabled {
		return nil, errors.New("deploypackage.disabled")
	}
	if template == nil {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, req.TemplateParams); err != nil {
			return nil, err
		}
	}
	targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
	scopeName := ""
	if req.TargetType == TargetTypeHost {
		if req.BusinessScopeID == 0 {
			return nil, errors.New("deploytask.scope_required")
		}
		var count int64
		if err := s.db.Table("biz_business_scope").Where("id = ? AND status = ? AND deleted_at IS NULL", req.BusinessScopeID, "active").Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, errors.New("deploytask.scope_invalid")
		}
		hosts, err := s.cmdbCapability.ResolveDeployTargets(cmdb.DeployHostResolveRequest{
			BusinessScopeID: req.BusinessScopeID,
			TargetType:      TargetTypeHost,
			TargetIDs:       targetIDs,
			DataScope:       dataScope,
		})
		if err != nil {
			return nil, err
		}
		if len(hosts) != len(targetIDs) {
			return nil, errors.New("deploytask.scope_invalid")
		}
		for _, host := range hosts {
			if !hostStatusAllowedForAction(host.Status, action) {
				return nil, errors.New("deploytask.target_invalid")
			}
			scopeName = host.BusinessScopeName
		}
	}
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
		Status:            TaskStatusPending,
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
	return s.GetTask(task.ID)
}

func (s *DeployService) ListTasks(query TaskQuery) (*TaskListResponse, error) {
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
		db = db.Where("status = ?", strings.TrimSpace(query.Status))
	}
	if strings.TrimSpace(query.ExecutorType) != "" {
		db = db.Where("executor_type = ?", strings.TrimSpace(query.ExecutorType))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []DeployTask
	if err := db.Order("id DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]TaskResponse, 0, len(rows))
	for i := range rows {
		items = append(items, taskToResponse(&rows[i], nil))
	}
	return &TaskListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *DeployService) GetTask(id uint64) (*TaskResponse, error) {
	var task DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.not_found")
		}
		return nil, err
	}
	var hosts []DeployTaskHost
	if err := s.db.Where("task_id = ?", id).Order("id ASC").Find(&hosts).Error; err != nil {
		return nil, err
	}
	hostResp := make([]TaskHostResponse, 0, len(hosts))
	for i := range hosts {
		hostResp = append(hostResp, taskHostToResponse(&hosts[i]))
	}
	resp := taskToResponse(&task, hostResp)
	return &resp, nil
}

func (s *DeployService) UpdateTask(id uint64, req UpdateTaskRequest, actor string) (*TaskResponse, error) {
	var task DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.not_found")
		}
		return nil, err
	}
	if task.Status != TaskStatusPending && task.Status != TaskStatusDraft {
		return nil, errors.New("deploytask.status_locked")
	}
	updates := map[string]interface{}{"updated_by": actor, "updated_at": time.Now()}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return nil, errors.New("deploytask.invalid")
		}
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.PackageID != nil {
		var pkg DeployPackage
		if err := s.db.First(&pkg, *req.PackageID).Error; err != nil {
			return nil, errors.New("deploypackage.not_found")
		}
		updates["package_id"] = pkg.ID
		updates["package_name"] = pkg.Name
		updates["package_version"] = pkg.Version
	}
	if req.BusinessScopeID != nil {
		updates["business_scope_id"] = *req.BusinessScopeID
		if *req.BusinessScopeID == 0 {
			updates["business_scope_name"] = ""
		} else {
			var scopeName string
			if err := s.db.Table("biz_business_scope").Select("name").Where("id = ? AND deleted_at IS NULL", *req.BusinessScopeID).Limit(1).Pluck("name", &scopeName).Error; err != nil {
				return nil, err
			}
			updates["business_scope_name"] = scopeName
		}
	}
	if req.TargetType != nil {
		if !validTargetType(*req.TargetType) {
			return nil, errors.New("deploytask.target_invalid")
		}
		updates["target_type"] = *req.TargetType
	}
	if req.TargetIDs != nil {
		targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
		if len(targetIDs) == 0 {
			return nil, errors.New("deploytask.invalid")
		}
		targetJSON, _ := json.Marshal(targetIDs)
		updates["target_ids"] = datatypes.JSON(targetJSON)
	}
	if req.ExecutorType != nil {
		if !validExecutorType(*req.ExecutorType) {
			return nil, errors.New("deploytask.executor_invalid")
		}
		updates["executor_type"] = *req.ExecutorType
	}
	if req.Action != nil {
		action := normalizeTaskAction(*req.Action)
		if !validTaskAction(action) {
			return nil, errors.New("deploytask.action_invalid")
		}
		updates["action"] = action
	}
	nextTemplateParams := decodeJSONMap(task.TemplateParams)
	if req.TemplateParams != nil {
		nextTemplateParams = *req.TemplateParams
		templateParamsJSON, _ := json.Marshal(*req.TemplateParams)
		updates["template_params"] = datatypes.JSON(templateParamsJSON)
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, task.PackageID).Error; err == nil {
		if err := validateTemplateParams(pkg.ExecutionMode, pkg.TemplateCode, pkg.TemplateConfig, nextTemplateParams); err != nil {
			return nil, err
		}
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetTask(id)
}

func (s *DeployService) StartTask(id uint64, req StartTaskRequest, actor string, dataScope *common.DataScopeReq) (*TaskResponse, error) {
	var task DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.not_found")
		}
		return nil, err
	}
	if task.Status != TaskStatusPending {
		return nil, errors.New("deploytask.status_invalid")
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, task.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploypackage.not_found")
		}
		return nil, err
	}
	hosts, err := s.resolveTaskTargets(&task, dataScope)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, errors.New("deploytask.no_targets")
	}
	now := time.Now()
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&task).Updates(map[string]interface{}{
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
		if err := s.executeSSHTask(task, pkg, hosts, req, actor); err != nil {
			return nil, err
		}
	}
	return s.GetTask(id)
}

func (s *DeployService) CancelTask(id uint64, actor string) (*TaskResponse, error) {
	now := time.Now()
	var task DeployTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, errors.New("deploytask.not_found")
	}
	if task.Status == TaskStatusSuccess || task.Status == TaskStatusFailed || task.Status == TaskStatusCanceled {
		return nil, errors.New("deploytask.status_locked")
	}
	if err := s.db.Model(&task).Updates(map[string]interface{}{
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
	return s.GetTask(id)
}

func (s *DeployService) MarkHostResult(hostID uint64, req MarkHostResultRequest, actor string) (*TaskHostResponse, error) {
	return s.markHostResultWithSummary(hostID, req, actor, deployExecutionSummary{})
}

func (s *DeployService) markHostResultWithSummary(hostID uint64, req MarkHostResultRequest, actor string, summary deployExecutionSummary) (*TaskHostResponse, error) {
	if req.Status != TaskHostStatusSuccess && req.Status != TaskHostStatusFailed && req.Status != TaskHostStatusSkipped {
		return nil, errors.New("deploytask.result_invalid")
	}
	req.Stdout = truncateDeployLog(req.Stdout, 60000)
	req.Stderr = truncateDeployLog(req.Stderr, 60000)
	req.ErrorMessage = truncateDeployLog(req.ErrorMessage, 480)
	var host DeployTaskHost
	if err := s.db.First(&host, hostID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.host_not_found")
		}
		return nil, err
	}
	var task DeployTask
	if err := s.db.First(&task, host.TaskID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.not_found")
		}
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

func (s *DeployService) recomputeTaskStatus(taskID uint64, actor string) error {
	var hosts []DeployTaskHost
	if err := s.db.Where("task_id = ?", taskID).Find(&hosts).Error; err != nil {
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
	return s.db.Model(&DeployTask{}).Where("id = ?", taskID).Updates(map[string]interface{}{
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

func hostStatusAllowedForAction(status string, action string) bool {
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
			return errors.New("deploytask.ssh_host_key_mismatch")
		}
		return nil
	}
}

func newDeploySSHRunner(host cmdbHostSnapshot, req StartTaskRequest) (deploySSHRunner, error) {
	fingerprint := strings.TrimSpace(req.HostFingerprint)
	if fingerprint == "" {
		return nil, errors.New("deploytask.ssh_host_key_required")
	}
	user := strings.TrimSpace(req.SSHUser)
	if user == "" {
		return nil, errors.New("deploytask.ssh_user_required")
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
			return nil, errors.New("deploytask.ssh_auth_failed")
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	default:
		if strings.TrimSpace(req.SSHPassword) == "" {
			return nil, errors.New("deploytask.ssh_password_required")
		}
		config.Auth = []ssh.AuthMethod{ssh.Password(req.SSHPassword)}
	}

	port := host.SSHPort
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.IP, port), config)
	if err != nil {
		return nil, errors.New("deploytask.ssh_connect_failed")
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
		return err
	}
	taskDetail, err := s.GetTask(task.ID)
	if err != nil {
		return err
	}

	taskHostsByID := make(map[uint64]TaskHostResponse, len(taskDetail.Hosts))
	for _, item := range taskDetail.Hosts {
		taskHostsByID[item.HostID] = item
	}

	for _, target := range hosts {
		taskHost, ok := taskHostsByID[target.ID]
		if !ok {
			continue
		}
		runner, runnerErr := s.sshRunnerFactory(target, req)
		if runnerErr != nil {
			if _, err := s.MarkHostResult(taskHost.ID, MarkHostResultRequest{
				Status:       TaskHostStatusFailed,
				ErrorMessage: runnerErr.Error(),
				ExecutorID:   fmt.Sprintf("ssh:%s", target.IP),
			}, actor); err != nil {
				return err
			}
			continue
		}

		_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
			{"at": time.Now().Format(time.RFC3339), "phase": "connect", "message": "SSH connected"},
		})
		stdoutSections := make([]string, 0, len(plan))
		stderrSections := make([]string, 0, len(plan))
		summary := deployExecutionSummary{
			InstalledComponents:   make([]cmdb.InstalledComponentUpsert, 0, len(plan)),
			RemovedComponentNames: make([]string, 0, len(plan)),
		}
		executorID := fmt.Sprintf("ssh:%s", target.IP)
		var executionErr error
		for _, step := range plan {
			stepLabel := buildDeployStepLabel(step)
			renderPhaseOutput := func(phase string, stdout string, stderr string) {
				if strings.TrimSpace(stdout) != "" {
					stdoutSections = append(stdoutSections, decorateStepPhaseOutput(step, phase, stdout))
				}
				if strings.TrimSpace(stderr) != "" {
					stderrSections = append(stderrSections, decorateStepPhaseOutput(step, phase, stderr))
				}
			}
			script, renderErr := s.renderExecutionStepScript(step, task, target)
			if renderErr != nil {
				executionErr = renderErr
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "render_failed",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     renderErr.Error(),
					},
				})
				break
			}
			_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
				{
					"at":          time.Now().Format(time.RFC3339),
					"phase":       "step_start",
					"stepCode":    step.StepCode,
					"stepName":    step.StepName,
					"stepType":    step.StepType,
					"action":      step.Action,
					"packageName": step.Package.Name,
					"message":     fmt.Sprintf("%s started", stepLabel),
				},
			})
			precheckScript, hasPrecheck, precheckErr := renderDeployCheckSnippet(step, task, target, "precheckCommand")
			if precheckErr != nil {
				executionErr = precheckErr
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "precheck_render_failed",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     precheckErr.Error(),
					},
				})
				break
			}
			if hasPrecheck {
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "precheck",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     fmt.Sprintf("%s precheck started", stepLabel),
					},
				})
				stdout, stderr, execErr := runner.RunScript(precheckScript)
				renderPhaseOutput("precheck", stdout, stderr)
				if execErr != nil {
					executionErr = fmt.Errorf("%s precheck failed: %w", stepLabel, execErr)
					_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
						{
							"at":          time.Now().Format(time.RFC3339),
							"phase":       "step_failed",
							"stepCode":    step.StepCode,
							"stepName":    step.StepName,
							"stepType":    step.StepType,
							"action":      step.Action,
							"packageName": step.Package.Name,
							"message":     executionErr.Error(),
						},
					})
					break
				}
			}
			_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
				{
					"at":          time.Now().Format(time.RFC3339),
					"phase":       "script",
					"stepCode":    step.StepCode,
					"stepName":    step.StepName,
					"stepType":    step.StepType,
					"action":      step.Action,
					"packageName": step.Package.Name,
					"message":     fmt.Sprintf("%s script rendered", stepLabel),
				},
			})
			stdout, stderr, execErr := runner.RunScript(script)
			renderPhaseOutput("script", stdout, stderr)
			if execErr != nil {
				executionErr = execErr
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "step_failed",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     execErr.Error(),
					},
				})
				break
			}
			postcheckScript, hasPostcheck, postcheckErr := renderDeployCheckSnippet(step, task, target, "postcheckCommand")
			if postcheckErr != nil {
				executionErr = postcheckErr
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "postcheck_render_failed",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     postcheckErr.Error(),
					},
				})
				break
			}
			if hasPostcheck {
				_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
					{
						"at":          time.Now().Format(time.RFC3339),
						"phase":       "postcheck",
						"stepCode":    step.StepCode,
						"stepName":    step.StepName,
						"stepType":    step.StepType,
						"action":      step.Action,
						"packageName": step.Package.Name,
						"message":     fmt.Sprintf("%s postcheck started", stepLabel),
					},
				})
				stdout, stderr, execErr := runner.RunScript(postcheckScript)
				renderPhaseOutput("postcheck", stdout, stderr)
				if execErr != nil {
					executionErr = fmt.Errorf("%s postcheck failed: %w", stepLabel, execErr)
					_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
						{
							"at":          time.Now().Format(time.RFC3339),
							"phase":       "step_failed",
							"stepCode":    step.StepCode,
							"stepName":    step.StepName,
							"stepType":    step.StepType,
							"action":      step.Action,
							"packageName": step.Package.Name,
							"message":     executionErr.Error(),
						},
					})
					break
				}
			}
			_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
				{
					"at":          time.Now().Format(time.RFC3339),
					"phase":       "step_success",
					"stepCode":    step.StepCode,
					"stepName":    step.StepName,
					"stepType":    step.StepType,
					"action":      step.Action,
					"packageName": step.Package.Name,
					"message":     fmt.Sprintf("%s completed", stepLabel),
				},
			})
			if step.Package.ID == 0 {
				continue
			}
			if normalizeTaskAction(step.Action) == TaskActionUninstall {
				summary.RemovedComponentNames = append(summary.RemovedComponentNames, step.Package.Name)
				continue
			}
			summary.InstalledComponents = append(summary.InstalledComponents, cmdb.InstalledComponentUpsert{
				Name:           step.Package.Name,
				Version:        step.Package.Version,
				DeployedAt:     time.Now(),
				DeployTaskID:   task.ID,
				DeployTaskName: task.Name,
				ExecutorType:   task.ExecutorType,
			})
		}
		closeErr := runner.Close()
		combinedStdout := strings.TrimSpace(strings.Join(stdoutSections, "\n\n"))
		combinedStderr := strings.TrimSpace(strings.Join(stderrSections, "\n\n"))
		if executionErr != nil || closeErr != nil {
			errorMessage := ""
			if executionErr != nil {
				errorMessage = executionErr.Error()
			}
			if closeErr != nil {
				if errorMessage == "" {
					errorMessage = closeErr.Error()
				} else {
					errorMessage = fmt.Sprintf("%s; close: %s", errorMessage, closeErr.Error())
				}
			}
			if _, err := s.markHostResultWithSummary(taskHost.ID, MarkHostResultRequest{
				Status:       TaskHostStatusFailed,
				Stdout:       combinedStdout,
				Stderr:       combinedStderr,
				ErrorMessage: errorMessage,
				ExecutorID:   executorID,
			}, actor, deployExecutionSummary{}); err != nil {
				return err
			}
			_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
				{"at": time.Now().Format(time.RFC3339), "phase": "failed", "message": errorMessage},
			})
			continue
		}
		if _, err := s.markHostResultWithSummary(taskHost.ID, MarkHostResultRequest{
			Status:     TaskHostStatusSuccess,
			Stdout:     combinedStdout,
			Stderr:     combinedStderr,
			ExecutorID: executorID,
		}, actor, summary); err != nil {
			return err
		}
		_ = s.appendTaskHostTrace(task.ID, taskHost.ID, []map[string]any{
			{"at": time.Now().Format(time.RFC3339), "phase": "writeback", "message": fmt.Sprintf("Host marked %s", hostStatusForAction(task.Action))},
		})
	}
	return nil
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
			return nil, errors.New("deploytemplate.invalid")
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

func mergeDeployTemplateParams(base map[string]any, override map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(override))
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
			return "", errors.New("deploytemplate.invalid")
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
			renderErr = fmt.Errorf("missing template variable: %s", key)
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

func (s *DeployService) upsertHostInstalledComponent(hostID uint64, taskID uint64, taskName string, executorType string, packageName string, packageVersion string, actor string, now time.Time) error {
	if strings.TrimSpace(packageName) == "" {
		return nil
	}
	var snapshot struct {
		InstalledComponents datatypes.JSON `gorm:"column:installed_components"`
	}
	if err := s.db.Table("biz_cmdb_host").Select("installed_components").Where("id = ?", hostID).Take(&snapshot).Error; err != nil {
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
		if strings.EqualFold(strings.TrimSpace(components[index].Name), strings.TrimSpace(packageName)) {
			components[index].Version = packageVersion
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
			Name:           packageName,
			Version:        packageVersion,
			DeployedAt:     now.Format(time.RFC3339),
			DeployTaskID:   taskID,
			DeployTaskName: taskName,
			ExecutorType:   executorType,
		})
	}
	nextPayload, _ := json.Marshal(components)
	return s.db.Table("biz_cmdb_host").Where("id = ?", hostID).Updates(map[string]interface{}{
		"installed_components": datatypes.JSON(nextPayload),
		"updated_by":           actor,
		"updated_at":           now,
	}).Error
}

type deployGroupCondition struct {
	Operator string                     `json:"operator"`
	Rules    []deployGroupConditionRule `json:"rules"`
}

type deployGroupConditionRule struct {
	Key string `json:"key"`
	Op  string `json:"op"`
	Val string `json:"val"`
}

type deployLabelEntry struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

func groupMatchesHost(conditionJSON datatypes.JSON, labelJSON datatypes.JSON) bool {
	var condition deployGroupCondition
	if err := json.Unmarshal(conditionJSON, &condition); err != nil || len(condition.Rules) == 0 {
		return false
	}
	var labels []deployLabelEntry
	_ = json.Unmarshal(labelJSON, &labels)
	labelMap := make(map[string]string, len(labels))
	for _, label := range labels {
		labelMap[label.Key] = label.Val
	}
	operator := strings.ToUpper(strings.TrimSpace(condition.Operator))
	if operator == "" {
		operator = "AND"
	}
	matched := operator == "AND"
	for _, rule := range condition.Rules {
		ok := deployRuleMatches(labelMap[rule.Key], rule)
		if operator == "OR" && ok {
			return true
		}
		if operator == "AND" && !ok {
			return false
		}
		matched = ok
	}
	return matched
}

func groupConditionChainMatchesHost(conditionChain []datatypes.JSON, labelJSON datatypes.JSON) bool {
	for _, conditionJSON := range conditionChain {
		if !groupMatchesHost(conditionJSON, labelJSON) {
			return false
		}
	}
	return len(conditionChain) > 0
}

func deployRuleMatches(actual string, rule deployGroupConditionRule) bool {
	values := strings.Split(rule.Val, ",")
	contains := false
	for _, value := range values {
		if strings.TrimSpace(value) == actual {
			contains = true
			break
		}
	}
	switch rule.Op {
	case "eq":
		return actual == rule.Val
	case "neq":
		return actual != rule.Val
	case "in":
		return contains
	case "notIn":
		return !contains
	default:
		return false
	}
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

func (s *DeployService) resolveInstallScript(pkg DeployPackage, task DeployTask) (string, error) {
	if strings.TrimSpace(pkg.TemplateCode) != "" && pkg.ExecutionMode == ExecutionModeFixed {
		return renderFixedTemplateScript(pkg, task)
	}
	if normalizeTaskAction(task.Action) == TaskActionUninstall {
		if strings.TrimSpace(pkg.UninstallCommand) == "" {
			return "", errors.New("deploytask.uninstall_command_required")
		}
		return pkg.UninstallCommand, nil
	}
	if strings.TrimSpace(pkg.InstallCommand) == "" {
		return "", errors.New("deploytask.install_command_required")
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
		return "", errors.New("deploytask.template_params_invalid")
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
		return "", errors.New("deploypackage.invalid")
	}
	sourceDownload := ""
	if sourceObjectKey != "" {
		if sourceURL == "" {
			return "", errors.New("deploypackage.source_missing")
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

func computeDurationSeconds(startedAt *time.Time, finishedAt *time.Time) int64 {
	if startedAt == nil || finishedAt == nil {
		return 0
	}
	seconds := int64(finishedAt.Sub(*startedAt).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}

func (s *DeployService) loadPackageDeploymentStats(packageIDs []uint64) (map[uint64]packageDeploymentStat, error) {
	result := make(map[uint64]packageDeploymentStat, len(packageIDs))
	if len(packageIDs) == 0 {
		return result, nil
	}
	var tasks []DeployTask
	if err := s.db.Where("package_id IN ?", packageIDs).Order("id DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}
	for _, task := range tasks {
		if _, exists := result[task.PackageID]; exists {
			continue
		}
		var hostRows []DeployTaskHost
		if err := s.db.Where("task_id = ?", task.ID).Find(&hostRows).Error; err != nil {
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

func (s *DeployService) taskQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Model(&DeployTask{}).Scopes(database.WithDataScope(dataScope))
}
