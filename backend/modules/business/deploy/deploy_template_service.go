package deploy

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const templateIDWhereClause = "template_id = ?"

func (s *DeployService) CreateTemplate(req CreateTemplateRequest, actor string) (*TemplateResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Version = strings.TrimSpace(req.Version)
	if req.Name == "" || req.Version == "" {
		return nil, errors.New("deploytemplate.invalid")
	}
	if req.Status == "" {
		req.Status = TemplateStatusEnabled
	}
	if req.ExecutionMode == "" {
		req.ExecutionMode = ExecutionModeFixed
	}
	req.DefaultAction = normalizeTaskAction(req.DefaultAction)
	if req.DefaultAction == "" {
		req.DefaultAction = TaskActionInstall
	}
	if !validTemplateStatus(req.Status) || !validExecutionMode(req.ExecutionMode) || !validTaskAction(req.DefaultAction) {
		return nil, errors.New("deploytemplate.invalid")
	}
	if s.templateExists(req.Name, req.Version, 0) {
		return nil, errors.New("deploytemplate.exists")
	}
	item, steps, err := s.buildTemplateAggregate(req, actor)
	if err != nil {
		return nil, err
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		for index := range steps {
			steps[index].TemplateID = item.ID
		}
		if len(steps) > 0 {
			if err := tx.Create(&steps).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s.GetTemplate(item.ID)
}

func (s *DeployService) ListTemplates(query TemplateQuery) (*TemplateListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := s.db.Model(&DeployTemplate{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR version LIKE ?", like, like)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		db = db.Where(statusWhereClause, status)
	}
	if mode := strings.TrimSpace(query.ExecutionMode); mode != "" {
		db = db.Where("execution_mode = ?", mode)
	}
	if category := strings.TrimSpace(query.Category); category != "" {
		db = db.Where("category = ?", category)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []DeployTemplate
	if err := db.Order(idDescOrder).Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	stepsByTemplate, err := s.loadTemplateSteps(ids)
	if err != nil {
		return nil, err
	}
	items := make([]TemplateResponse, 0, len(rows))
	for i := range rows {
		items = append(items, templateToResponse(&rows[i], stepsByTemplate[rows[i].ID]))
	}
	return &TemplateListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *DeployService) GetTemplate(id uint64) (*TemplateResponse, error) {
	var item DeployTemplate
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("deploytemplate.not_found")
		}
		return nil, err
	}
	stepsByTemplate, err := s.loadTemplateSteps([]uint64{id})
	if err != nil {
		return nil, err
	}
	resp := templateToResponse(&item, stepsByTemplate[id])
	return &resp, nil
}

func (s *DeployService) UpdateTemplate(id uint64, req UpdateTemplateRequest, actor string) (*TemplateResponse, error) {
	current, err := s.loadTemplateForUpdate(id)
	if err != nil {
		return nil, err
	}
	next, err := s.mergeTemplateUpdate(current, req, id)
	if err != nil {
		return nil, err
	}
	item, steps, err := s.buildTemplateAggregate(next, actor)
	if err != nil {
		return nil, err
	}
	item.ID = id
	item.CreatedAt = current.CreatedAt
	item.CreatedBy = current.CreatedBy
	item.UpdatedAt = time.Now()
	item.UpdatedBy = actor
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return persistTemplateUpdate(tx, id, item, steps)
	}); err != nil {
		return nil, err
	}
	return s.GetTemplate(id)
}

func (s *DeployService) loadTemplateForUpdate(id uint64) (DeployTemplate, error) {
	var current DeployTemplate
	if err := s.db.First(&current, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DeployTemplate{}, errors.New("deploytemplate.not_found")
		}
		return DeployTemplate{}, err
	}
	return current, nil
}

func (s *DeployService) mergeTemplateUpdate(current DeployTemplate, req UpdateTemplateRequest, id uint64) (CreateTemplateRequest, error) {
	next := newTemplateRequestFromModel(current)
	applyTemplateFieldOverrides(&next, req)
	steps, err := s.resolveTemplateSteps(req, id)
	if err != nil {
		return next, err
	}
	next.Steps = steps
	next.Name = strings.TrimSpace(next.Name)
	next.Version = strings.TrimSpace(next.Version)
	if next.Name == "" || next.Version == "" {
		return next, errors.New("deploytemplate.invalid")
	}
	next.DefaultAction = normalizeTaskAction(next.DefaultAction)
	if next.DefaultAction == "" {
		next.DefaultAction = TaskActionInstall
	}
	if !validTemplateStatus(next.Status) || !validExecutionMode(next.ExecutionMode) || !validTaskAction(next.DefaultAction) {
		return next, errors.New("deploytemplate.invalid")
	}
	if s.templateExists(next.Name, next.Version, id) {
		return next, errors.New("deploytemplate.exists")
	}
	return next, nil
}

func newTemplateRequestFromModel(current DeployTemplate) CreateTemplateRequest {
	return CreateTemplateRequest{
		Name:            current.Name,
		Version:         current.Version,
		Description:     current.Description,
		Category:        current.Category,
		ExecutionMode:   current.ExecutionMode,
		DefaultAction:   current.DefaultAction,
		PackageID:       current.PackageID,
		TemplateCode:    current.TemplateCode,
		TemplateConfig:  decodeJSONMap(current.TemplateConfig),
		ParameterSchema: decodeJSONMap(current.ParameterSchema),
		Status:          current.Status,
	}
}

func applyTemplateFieldOverrides(next *CreateTemplateRequest, req UpdateTemplateRequest) {
	if req.Name != nil {
		next.Name = *req.Name
	}
	if req.Version != nil {
		next.Version = *req.Version
	}
	if req.Description != nil {
		next.Description = *req.Description
	}
	if req.Category != nil {
		next.Category = *req.Category
	}
	if req.ExecutionMode != nil {
		next.ExecutionMode = *req.ExecutionMode
	}
	if req.DefaultAction != nil {
		next.DefaultAction = *req.DefaultAction
	}
	if req.PackageID != nil {
		next.PackageID = *req.PackageID
	}
	if req.TemplateCode != nil {
		next.TemplateCode = *req.TemplateCode
	}
	if req.TemplateConfig != nil {
		next.TemplateConfig = *req.TemplateConfig
	}
	if req.ParameterSchema != nil {
		next.ParameterSchema = *req.ParameterSchema
	}
	if req.Status != nil {
		next.Status = *req.Status
	}
}

func (s *DeployService) resolveTemplateSteps(req UpdateTemplateRequest, id uint64) ([]TemplateStepPayload, error) {
	if req.Steps != nil {
		return *req.Steps, nil
	}
	existingSteps, err := s.loadTemplateSteps([]uint64{id})
	if err != nil {
		return nil, err
	}
	steps := make([]TemplateStepPayload, 0, len(existingSteps[id]))
	for _, step := range existingSteps[id] {
		steps = append(steps, TemplateStepPayload{
			StepCode:       step.StepCode,
			StepName:       step.StepName,
			StepType:       step.StepType,
			Action:         step.Action,
			PackageID:      step.PackageID,
			PackageName:    step.PackageName,
			PackageVersion: step.PackageVersion,
			TemplateCode:   step.TemplateCode,
			TemplateParams: step.TemplateParams,
			StepConfig:     step.StepConfig,
			Sort:           step.Sort,
		})
	}
	return steps, nil
}

func persistTemplateUpdate(tx *gorm.DB, id uint64, item *DeployTemplate, steps []DeployTemplateStep) error {
	updates := map[string]any{
		"name":             item.Name,
		"version":          item.Version,
		"description":      item.Description,
		"category":         item.Category,
		"execution_mode":   item.ExecutionMode,
		"default_action":   item.DefaultAction,
		"package_id":       item.PackageID,
		"package_name":     item.PackageName,
		"package_version":  item.PackageVersion,
		"template_code":    item.TemplateCode,
		"template_config":  item.TemplateConfig,
		"parameter_schema": item.ParameterSchema,
		"status":           item.Status,
		"updated_at":       item.UpdatedAt,
		"updated_by":       item.UpdatedBy,
	}
	if err := tx.Model(&DeployTemplate{}).Where(idWhereClause, id).Updates(updates).Error; err != nil {
		return err
	}
	if err := tx.Where(templateIDWhereClause, id).Delete(&DeployTemplateStep{}).Error; err != nil {
		return err
	}
	for index := range steps {
		steps[index].TemplateID = id
	}
	if len(steps) > 0 {
		if err := tx.Create(&steps).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *DeployService) DeleteTemplate(id uint64) error {
	var count int64
	if err := s.db.Model(&DeployTask{}).Where(templateIDWhereClause, id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("deploytemplate.in_use")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(templateIDWhereClause, id).Delete(&DeployTemplateStep{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&DeployTemplate{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("deploytemplate.not_found")
		}
		return nil
	})
}

type templateBasePackage struct {
	ID           uint64
	Name         string
	Version      string
	TemplateCode string
}

func (s *DeployService) buildTemplateAggregate(req CreateTemplateRequest, actor string) (*DeployTemplate, []DeployTemplateStep, error) {
	base, err := s.resolveTemplateBasePackage(req)
	if err != nil {
		return nil, nil, err
	}
	item := buildTemplateItem(req, actor, base)
	if len(req.Steps) == 0 && base.ID > 0 {
		req.Steps = []TemplateStepPayload{{
			StepCode:       "default",
			StepName:       base.Name,
			StepType:       "package",
			Action:         req.DefaultAction,
			PackageID:      base.ID,
			PackageName:    base.Name,
			PackageVersion: base.Version,
			TemplateCode:   base.TemplateCode,
			TemplateParams: req.ParameterSchema,
			Sort:           1,
		}}
	}
	steps := make([]DeployTemplateStep, 0, len(req.Steps))
	for index, step := range req.Steps {
		resolved, err := s.resolveTemplateStep(step, req, base, index)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, resolved)
	}
	return item, steps, nil
}

func (s *DeployService) resolveTemplateBasePackage(req CreateTemplateRequest) (templateBasePackage, error) {
	base := templateBasePackage{ID: req.PackageID, TemplateCode: strings.TrimSpace(req.TemplateCode)}
	if base.ID == 0 {
		return base, nil
	}
	var pkg DeployPackage
	if err := s.db.First(&pkg, base.ID).Error; err != nil {
		return templateBasePackage{}, errors.New("deploypackage.not_found")
	}
	base.Name = pkg.Name
	base.Version = pkg.Version
	if base.TemplateCode == "" {
		base.TemplateCode = pkg.TemplateCode
	}
	return base, nil
}

func buildTemplateItem(req CreateTemplateRequest, actor string, base templateBasePackage) *DeployTemplate {
	templateConfigJSON, _ := json.Marshal(req.TemplateConfig)
	parameterSchemaJSON, _ := json.Marshal(req.ParameterSchema)
	return &DeployTemplate{
		Name:            req.Name,
		Version:         req.Version,
		Description:     req.Description,
		Category:        strings.TrimSpace(req.Category),
		ExecutionMode:   req.ExecutionMode,
		DefaultAction:   req.DefaultAction,
		PackageID:       base.ID,
		PackageName:     base.Name,
		PackageVersion:  base.Version,
		TemplateCode:    base.TemplateCode,
		TemplateConfig:  datatypes.JSON(templateConfigJSON),
		ParameterSchema: datatypes.JSON(parameterSchemaJSON),
		Status:          req.Status,
		CreatedBy:       actor,
		UpdatedBy:       actor,
	}
}

func (s *DeployService) resolveTemplateStep(step TemplateStepPayload, req CreateTemplateRequest, base templateBasePackage, index int) (DeployTemplateStep, error) {
	stepCode := strings.TrimSpace(step.StepCode)
	if stepCode == "" {
		stepCode = "step_" + strings.TrimSpace(time.Now().Format("150405")) + "_" + string(rune('a'+index))
	}
	stepType := strings.TrimSpace(step.StepType)
	if stepType == "" {
		stepType = TemplateStepTypePackage
	}
	if stepType != TemplateStepTypePackage && stepType != TemplateStepTypeScript {
		return DeployTemplateStep{}, errors.New("deploytemplate.invalid")
	}
	action := normalizeTaskAction(step.Action)
	if action == "" {
		action = req.DefaultAction
	}
	if !validTaskAction(action) {
		return DeployTemplateStep{}, errors.New("deploytemplate.invalid")
	}
	packageIDForStep, packageNameForStep, packageVersionForStep, templateCodeForStep, err := s.resolveTemplateStepPackage(step, stepType, base)
	if err != nil {
		return DeployTemplateStep{}, err
	}
	if stepType == TemplateStepTypePackage && packageIDForStep == 0 {
		return DeployTemplateStep{}, errors.New("deploytemplate.invalid")
	}
	stepName := strings.TrimSpace(step.StepName)
	if stepName == "" {
		stepName = packageNameForStep
	}
	if stepName == "" {
		stepName = stepCode
	}
	sortValue := step.Sort
	if sortValue <= 0 {
		sortValue = index + 1
	}
	templateParamsJSON, _ := json.Marshal(step.TemplateParams)
	stepConfigJSON, _ := json.Marshal(step.StepConfig)
	return DeployTemplateStep{
		StepCode:       stepCode,
		StepName:       stepName,
		StepType:       stepType,
		Action:         action,
		PackageID:      packageIDForStep,
		PackageName:    packageNameForStep,
		PackageVersion: packageVersionForStep,
		TemplateCode:   templateCodeForStep,
		TemplateParams: datatypes.JSON(templateParamsJSON),
		StepConfig:     datatypes.JSON(stepConfigJSON),
		Sort:           sortValue,
	}, nil
}

func (s *DeployService) resolveTemplateStepPackage(step TemplateStepPayload, stepType string, base templateBasePackage) (uint64, string, string, string, error) {
	packageIDForStep := step.PackageID
	packageNameForStep := strings.TrimSpace(step.PackageName)
	packageVersionForStep := strings.TrimSpace(step.PackageVersion)
	templateCodeForStep := strings.TrimSpace(step.TemplateCode)
	if packageIDForStep > 0 {
		var pkg DeployPackage
		if err := s.db.First(&pkg, packageIDForStep).Error; err != nil {
			return 0, "", "", "", errors.New("deploypackage.not_found")
		}
		packageNameForStep = pkg.Name
		packageVersionForStep = pkg.Version
		if templateCodeForStep == "" {
			templateCodeForStep = pkg.TemplateCode
		}
		return packageIDForStep, packageNameForStep, packageVersionForStep, templateCodeForStep, nil
	}
	if stepType == TemplateStepTypePackage && base.ID > 0 {
		packageIDForStep = base.ID
		packageNameForStep = base.Name
		packageVersionForStep = base.Version
		if templateCodeForStep == "" {
			templateCodeForStep = base.TemplateCode
		}
	}
	return packageIDForStep, packageNameForStep, packageVersionForStep, templateCodeForStep, nil
}

func (s *DeployService) loadTemplateSteps(templateIDs []uint64) (map[uint64][]TemplateStepResponse, error) {
	result := make(map[uint64][]TemplateStepResponse, len(templateIDs))
	if len(templateIDs) == 0 {
		return result, nil
	}
	var rows []DeployTemplateStep
	if err := s.db.Where("template_id IN ?", templateIDs).Order("sort ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.TemplateID] = append(result[row.TemplateID], templateStepToResponse(&row))
	}
	return result, nil
}

func templateToResponse(item *DeployTemplate, steps []TemplateStepResponse) TemplateResponse {
	return TemplateResponse{
		ID:              item.ID,
		Name:            item.Name,
		Version:         item.Version,
		Description:     item.Description,
		Category:        item.Category,
		ExecutionMode:   item.ExecutionMode,
		DefaultAction:   item.DefaultAction,
		PackageID:       item.PackageID,
		PackageName:     item.PackageName,
		PackageVersion:  item.PackageVersion,
		TemplateCode:    item.TemplateCode,
		TemplateConfig:  decodeJSONMap(item.TemplateConfig),
		ParameterSchema: decodeJSONMap(item.ParameterSchema),
		Status:          item.Status,
		StepCount:       len(steps),
		Steps:           steps,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
		CreatedBy:       item.CreatedBy,
		UpdatedBy:       item.UpdatedBy,
	}
}

func templateStepToResponse(step *DeployTemplateStep) TemplateStepResponse {
	return TemplateStepResponse{
		ID:             step.ID,
		TemplateID:     step.TemplateID,
		StepCode:       step.StepCode,
		StepName:       step.StepName,
		StepType:       step.StepType,
		Action:         step.Action,
		PackageID:      step.PackageID,
		PackageName:    step.PackageName,
		PackageVersion: step.PackageVersion,
		TemplateCode:   step.TemplateCode,
		TemplateParams: decodeJSONMap(step.TemplateParams),
		StepConfig:     decodeJSONMap(step.StepConfig),
		Sort:           step.Sort,
		CreatedAt:      step.CreatedAt,
		UpdatedAt:      step.UpdatedAt,
	}
}

func validTemplateStatus(status string) bool {
	return status == TemplateStatusEnabled || status == TemplateStatusDisabled
}

func (s *DeployService) templateExists(name string, version string, excludeID uint64) bool {
	var count int64
	db := s.db.Model(&DeployTemplate{}).Where("name = ? AND version = ?", name, version)
	if excludeID > 0 {
		db = db.Where("id <> ?", excludeID)
	}
	_ = db.Count(&count).Error
	return count > 0
}
