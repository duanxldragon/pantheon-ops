package deploy

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type DeployService struct {
	db *gorm.DB
}

func NewDeployService(db *gorm.DB) *DeployService {
	return &DeployService{db: db}
}

func (s *DeployService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&DeployPackage{}, &DeployTask{}, &DeployTaskHost{})
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
	if !validPackageStatus(req.Status) {
		return nil, errors.New("deploypackage.status_invalid")
	}
	if s.packageExists(req.Name, req.Version, 0) {
		return nil, errors.New("deploypackage.exists")
	}
	item := DeployPackage{
		Name:             req.Name,
		Version:          req.Version,
		Description:      req.Description,
		InstallCommand:   req.InstallCommand,
		UninstallCommand: req.UninstallCommand,
		Status:           req.Status,
		CreatedBy:        actor,
		UpdatedBy:        actor,
	}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	resp := packageToResponse(&item)
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
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []DeployPackage
	if err := db.Order("id DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]PackageResponse, 0, len(rows))
	for i := range rows {
		items = append(items, packageToResponse(&rows[i]))
	}
	return &PackageListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
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
	resp := packageToResponse(&item)
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

func (s *DeployService) CreateTask(req CreateTaskRequest, actor string) (*TaskResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.PackageID == 0 || len(common.NormalizeUint64IDs(req.TargetIDs)) == 0 {
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
	if err := s.db.First(&pkg, req.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploypackage.not_found")
		}
		return nil, err
	}
	if pkg.Status != PackageStatusEnabled {
		return nil, errors.New("deploypackage.disabled")
	}
	targetIDs := common.NormalizeUint64IDs(req.TargetIDs)
	targetJSON, _ := json.Marshal(targetIDs)
	task := DeployTask{
		Name:           req.Name,
		PackageID:      pkg.ID,
		PackageName:    pkg.Name,
		PackageVersion: pkg.Version,
		TargetType:     req.TargetType,
		TargetIDs:      datatypes.JSON(targetJSON),
		ExecutorType:   req.ExecutorType,
		Status:         TaskStatusPending,
		Remark:         req.Remark,
		CreatedBy:      actor,
		UpdatedBy:      actor,
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
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetTask(id)
}

func (s *DeployService) StartTask(id uint64, actor string) (*TaskResponse, error) {
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
	hosts, err := s.resolveTaskTargets(&task)
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
			row := DeployTaskHost{
				TaskID:    task.ID,
				HostID:    host.ID,
				Hostname:  host.Hostname,
				HostIP:    host.IP,
				OS:        host.OS,
				Status:    TaskHostStatusRunning,
				StartedAt: &now,
				UpdatedBy: actor,
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
	if req.Status != TaskHostStatusSuccess && req.Status != TaskHostStatusFailed && req.Status != TaskHostStatusSkipped {
		return nil, errors.New("deploytask.result_invalid")
	}
	var host DeployTaskHost
	if err := s.db.First(&host, hostID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytask.host_not_found")
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
	if err := s.recomputeTaskStatus(host.TaskID, actor); err != nil {
		return nil, err
	}
	if err := s.db.First(&host, hostID).Error; err != nil {
		return nil, err
	}
	resp := taskHostToResponse(&host)
	return &resp, nil
}

func (s *DeployService) resolveTaskTargets(task *DeployTask) ([]cmdbHostSnapshot, error) {
	targetIDs := parseUint64JSON(task.TargetIDs)
	if len(targetIDs) == 0 {
		return nil, nil
	}
	if task.TargetType == TargetTypeHost {
		var hosts []cmdbHostSnapshot
		err := s.db.Table("biz_cmdb_host").Where("id IN ? AND deleted_at IS NULL", targetIDs).Order("id ASC").Scan(&hosts).Error
		return hosts, err
	}
	var selectedGroups []cmdbGroupSnapshot
	if err := s.db.Table("biz_cmdb_group").Where("id IN ? AND deleted_at IS NULL", targetIDs).Scan(&selectedGroups).Error; err != nil {
		return nil, err
	}
	var allGroups []cmdbGroupSnapshot
	if err := s.db.Table("biz_cmdb_group").Where("deleted_at IS NULL").Scan(&allGroups).Error; err != nil {
		return nil, err
	}
	groupsByID := make(map[uint64]cmdbGroupSnapshot, len(allGroups))
	for _, group := range allGroups {
		groupsByID[group.ID] = group
	}
	var hosts []cmdbHostSnapshot
	if err := s.db.Table("biz_cmdb_host").Where("deleted_at IS NULL").Order("id ASC").Scan(&hosts).Error; err != nil {
		return nil, err
	}
	result := make([]cmdbHostSnapshot, 0)
	seen := map[uint64]struct{}{}
	for _, host := range hosts {
		for _, group := range selectedGroups {
			if groupConditionChainMatchesHost(deployConditionChainFromMap(group, groupsByID), host.LabelValues) {
				if _, ok := seen[host.ID]; !ok {
					seen[host.ID] = struct{}{}
					result = append(result, host)
				}
				break
			}
		}
	}
	return result, nil
}

func deployConditionChainFromMap(group cmdbGroupSnapshot, groupsByID map[uint64]cmdbGroupSnapshot) []datatypes.JSON {
	chain := []datatypes.JSON{group.Conditions}
	visited := map[uint64]struct{}{group.ID: {}}
	parentID := group.ParentID
	for parentID != 0 {
		if _, ok := visited[parentID]; ok {
			break
		}
		parent, ok := groupsByID[parentID]
		if !ok {
			break
		}
		visited[parent.ID] = struct{}{}
		chain = append([]datatypes.JSON{parent.Conditions}, chain...)
		parentID = parent.ParentID
	}
	return chain
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

func validTargetType(targetType string) bool {
	return targetType == TargetTypeHost || targetType == TargetTypeGroup
}

func validExecutorType(executorType string) bool {
	return executorType == ExecutorTypeManual || executorType == ExecutorTypeSimulated || executorType == ExecutorTypeAgent || executorType == ExecutorTypeSSH
}

func parseUint64JSON(raw datatypes.JSON) []uint64 {
	var ids []uint64
	_ = json.Unmarshal(raw, &ids)
	return common.NormalizeUint64IDs(ids)
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

func packageToResponse(item *DeployPackage) PackageResponse {
	return PackageResponse{
		ID:               item.ID,
		Name:             item.Name,
		Version:          item.Version,
		Description:      item.Description,
		InstallCommand:   item.InstallCommand,
		UninstallCommand: item.UninstallCommand,
		Status:           item.Status,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
		CreatedBy:        item.CreatedBy,
		UpdatedBy:        item.UpdatedBy,
	}
}

func taskToResponse(task *DeployTask, hosts []TaskHostResponse) TaskResponse {
	return TaskResponse{
		ID:             task.ID,
		Name:           task.Name,
		PackageID:      task.PackageID,
		PackageName:    task.PackageName,
		PackageVersion: task.PackageVersion,
		TargetType:     task.TargetType,
		TargetIDs:      parseUint64JSON(task.TargetIDs),
		ExecutorType:   task.ExecutorType,
		Status:         task.Status,
		Remark:         task.Remark,
		ExternalTaskID: task.ExternalTaskID,
		StartedAt:      task.StartedAt,
		FinishedAt:     task.FinishedAt,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
		CreatedBy:      task.CreatedBy,
		UpdatedBy:      task.UpdatedBy,
		Hosts:          hosts,
	}
}

func taskHostToResponse(host *DeployTaskHost) TaskHostResponse {
	return TaskHostResponse{
		ID:           host.ID,
		TaskID:       host.TaskID,
		HostID:       host.HostID,
		Hostname:     host.Hostname,
		HostIP:       host.HostIP,
		OS:           host.OS,
		Status:       host.Status,
		Stdout:       host.Stdout,
		Stderr:       host.Stderr,
		ErrorMessage: host.ErrorMessage,
		ExecutorID:   host.ExecutorID,
		StartedAt:    host.StartedAt,
		FinishedAt:   host.FinishedAt,
		ReportedAt:   host.ReportedAt,
		UpdatedAt:    host.UpdatedAt,
		UpdatedBy:    host.UpdatedBy,
	}
}

func (s *DeployService) taskQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Model(&DeployTask{}).Scopes(database.WithDataScope(dataScope))
}
