package bizscope

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	envCodeExistsKey  = "business.bizscope.environment.codeExists"
	envNotFoundKey    = "business.bizscope.environment.notFound"
	envStatusInvalid  = "business.bizscope.environment.statusInvalid"
	envCodeInvalid    = "business.bizscope.environment.codeRequired"
	envNameRequired   = "business.bizscope.environment.nameRequired"
	envStatusRequired = "business.bizscope.environment.statusRequired"
	envTypeInvalid    = "business.bizscope.environment.envTypeInvalid"
	envTypeRequired   = "business.bizscope.environment.envTypeRequired"
)

var validEnvTypes = map[string]bool{"dev": true, "test": true, "prod": true}

// ListEnvironments returns a filtered, paged set of environments for a scope.
func (s *Service) ListEnvironments(scopeID uint64, query *EnvironmentListQuery) (*EnvironmentListPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query == nil {
		query = &EnvironmentListQuery{}
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	if err := s.requireScope(scopeID); err != nil {
		return nil, err
	}

	db := s.db.Model(&BusinessEnvironment{}).Where("business_scope_id = ?", scopeID)
	if query.Code != "" {
		db = db.Where("code LIKE ?", "%"+query.Code+"%")
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []BusinessEnvironment
	if err := db.Order("id asc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]EnvironmentListResp, len(rows))
	for index, row := range rows {
		items[index] = toEnvironmentResp(row)
	}
	return &EnvironmentListPageResp{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

// ListEnvironmentOptions returns active environment definitions for a scope,
// keyed by code, so callers can restrict deployments to defined environments.
func (s *Service) ListEnvironmentOptions(scopeID uint64, activeOnly bool) ([]EnvironmentOptionItem, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	db := s.db.Model(&BusinessEnvironment{}).Where("business_scope_id = ?", scopeID).Order("id asc")
	if activeOnly {
		db = db.Where("status = ?", "active")
	}
	var rows []BusinessEnvironment
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]EnvironmentOptionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, EnvironmentOptionItem{Label: row.Name, Value: row.Code, Code: row.Code, Name: row.Name})
	}
	return items, nil
}

// CreateEnvironment validates and creates an environment for a business scope.
func (s *Service) CreateEnvironment(scopeID uint64, req *CreateEnvironmentRequest) (*EnvironmentListResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if req == nil {
		return nil, errors.New("param.invalid")
	}
	if err := s.requireScope(scopeID); err != nil {
		return nil, err
	}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	envType := strings.TrimSpace(req.EnvType)
	status := strings.TrimSpace(req.Status)
	if code == "" {
		return nil, errors.New(envCodeInvalid)
	}
	if name == "" {
		return nil, errors.New(envNameRequired)
	}
	if envType == "" {
		return nil, errors.New(envTypeRequired)
	}
	if !validEnvTypes[envType] {
		return nil, errors.New(envTypeInvalid)
	}
	if status == "" {
		return nil, errors.New(envStatusRequired)
	}
	if !validEnvironmentStatus(status) {
		return nil, errors.New(envStatusInvalid)
	}
	if s.environmentCodeExists(scopeID, code, 0) {
		return nil, errors.New(envCodeExistsKey)
	}
	row := BusinessEnvironment{
		BusinessScopeID: scopeID,
		Code:            code,
		Name:            name,
		EnvType:         envType,
		Status:          status,
		Remark:          strings.TrimSpace(req.Remark),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp := toEnvironmentResp(row)
	return &resp, nil
}

// UpdateEnvironment updates the provided fields of an existing environment.
func (s *Service) UpdateEnvironment(scopeID uint64, id uint64, req *UpdateEnvironmentRequest) (*EnvironmentListResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if req == nil {
		return nil, errors.New("param.invalid")
	}
	var row BusinessEnvironment
	if err := s.db.Where("business_scope_id = ? AND id = ?", scopeID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(envNotFoundKey)
		}
		return nil, err
	}
	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			return nil, errors.New(envCodeInvalid)
		}
		if s.environmentCodeExists(scopeID, code, id) {
			return nil, errors.New(envCodeExistsKey)
		}
		row.Code = code
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New(envNameRequired)
		}
		row.Name = name
	}
	if req.EnvType != nil {
		envType := strings.TrimSpace(*req.EnvType)
		if envType == "" {
			return nil, errors.New(envTypeRequired)
		}
		if !validEnvTypes[envType] {
			return nil, errors.New(envTypeInvalid)
		}
		row.EnvType = envType
	}
	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if status == "" {
			return nil, errors.New(envStatusRequired)
		}
		if !validEnvironmentStatus(status) {
			return nil, errors.New(envStatusInvalid)
		}
		row.Status = status
	}
	if req.Remark != nil {
		row.Remark = strings.TrimSpace(*req.Remark)
	}
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	resp := toEnvironmentResp(row)
	return &resp, nil
}

// DeleteEnvironment removes an environment; it fails if none exists.
func (s *Service) DeleteEnvironment(scopeID uint64, id uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	result := s.db.Where("business_scope_id = ? AND id = ?", scopeID, id).Delete(&BusinessEnvironment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(envNotFoundKey)
	}
	return nil
}

// GetEnvironment returns a single environment for a business scope.
func (s *Service) GetEnvironment(scopeID uint64, id uint64) (*EnvironmentListResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var row BusinessEnvironment
	if err := s.db.Where("business_scope_id = ? AND id = ?", scopeID, id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(envNotFoundKey)
		}
		return nil, err
	}
	resp := toEnvironmentResp(row)
	return &resp, nil
}

func (s *Service) requireScope(scopeID uint64) error {
	if _, err := s.GetActive(context.Background(), scopeID, nil); err != nil {
		if errors.Is(err, errors.New(bizScopeNotFoundKey)) {
			return err
		}
		return err
	}
	return nil
}

func (s *Service) environmentCodeExists(scopeID uint64, code string, excludeID uint64) bool {
	if s.db == nil {
		return false
	}
	var count int64
	query := s.db.Model(&BusinessEnvironment{}).Where("business_scope_id = ? AND code = ?", scopeID, code)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	_ = query.Count(&count).Error
	return count > 0
}

func validEnvironmentStatus(status string) bool {
	return status == "active" || status == "inactive"
}

func toEnvironmentResp(row BusinessEnvironment) EnvironmentListResp {
	return EnvironmentListResp{
		ID:              row.ID,
		BusinessScopeID: row.BusinessScopeID,
		Code:            row.Code,
		Name:            row.Name,
		EnvType:         row.EnvType,
		Status:          row.Status,
		Remark:          row.Remark,
		CreatedAt:       row.CreatedAt.Format(time.RFC3339),
	}
}
