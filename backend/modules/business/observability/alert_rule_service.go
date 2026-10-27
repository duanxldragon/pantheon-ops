package observability

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AlertRuleService manages alert rules.
type AlertRuleService struct {
	db *gorm.DB
}

// NewAlertRuleService creates a new AlertRuleService.
func NewAlertRuleService(db *gorm.DB) *AlertRuleService {
	return &AlertRuleService{db: db}
}

// Migrate creates or updates database schema.
func (s *AlertRuleService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&AlertRule{}, &AlertEvent{})
}

// Create adds a new alert rule.
func (s *AlertRuleService) Create(dto AlertRuleDTO, createdBy string, scopes ...*common.DataScopeReq) (*AlertRuleResponse, error) {
	scope := firstDataScope(scopes)
	// Verify datasource exists
	var ds PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&ds, dto.DatasourceID).Error; err != nil {
		return nil, fmt.Errorf("datasource.not_found: %w", err)
	}
	if err := s.validateNotificationChannels(dto.NotificationChannels, ds.DeptID, scope); err != nil {
		return nil, err
	}

	labelsJSON, _ := json.Marshal(dto.Labels)
	annotationsJSON, _ := json.Marshal(dto.Annotations)
	channelsJSON, _ := json.Marshal(dto.NotificationChannels)

	record := &AlertRule{
		DeptID:               ds.DeptID,
		Name:                 dto.Name,
		DatasourceID:         dto.DatasourceID,
		Enabled:              dto.Enabled,
		Query:                dto.Query,
		Interval:             dto.Interval,
		ForDuration:          dto.ForDuration,
		Severity:             dto.Severity,
		Labels:               datatypes.JSON(labelsJSON),
		Annotations:          datatypes.JSON(annotationsJSON),
		NotificationChannels: datatypes.JSON(channelsJSON),
		Status:               "active",
		CreatedBy:            createdBy,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return s.toResponse(record, &ds), nil
}

// List returns alert rules with filtering.
func (s *AlertRuleService) List(query AlertRuleListQuery, scopes ...*common.DataScopeReq) (*AlertRuleListResponse, error) {
	scope := firstDataScope(scopes)
	db := s.db.Model(&AlertRule{}).Scopes(database.WithDataScope(scope))

	if query.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+query.Keyword+"%")
	}
	if query.DatasourceID > 0 {
		db = db.Where("datasource_id = ?", query.DatasourceID)
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Enabled != nil {
		db = db.Where("enabled = ?", *query.Enabled)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	var records []AlertRule
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	// Load datasource names
	dsMap := make(map[uint64]*PrometheusDatasource)
	for _, r := range records {
		if _, exists := dsMap[r.DatasourceID]; !exists {
			var ds PrometheusDatasource
			if err := s.db.Scopes(database.WithDataScope(scope)).First(&ds, r.DatasourceID).Error; err == nil {
				dsMap[r.DatasourceID] = &ds
			}
		}
	}

	items := make([]AlertRuleResponse, len(records))
	for i, r := range records {
		items[i] = *s.toResponse(&r, dsMap[r.DatasourceID])
	}

	return &AlertRuleListResponse{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetByID retrieves an alert rule by ID.
func (s *AlertRuleService) GetByID(id uint64, scopes ...*common.DataScopeReq) (*AlertRuleResponse, error) {
	scope := firstDataScope(scopes)
	var record AlertRule
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
		return nil, err
	}

	var ds PrometheusDatasource
	_ = s.db.Scopes(database.WithDataScope(scope)).First(&ds, record.DatasourceID).Error

	return s.toResponse(&record, &ds), nil
}

// Update modifies an existing alert rule.
func (s *AlertRuleService) Update(id uint64, dto AlertRuleDTO, scopes ...*common.DataScopeReq) error {
	scope := firstDataScope(scopes)
	var record AlertRule
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&record, id).Error; err != nil {
		return err
	}
	var datasource PrometheusDatasource
	if err := s.db.Scopes(database.WithDataScope(scope)).First(&datasource, dto.DatasourceID).Error; err != nil {
		return err
	}
	if err := s.validateNotificationChannels(dto.NotificationChannels, datasource.DeptID, scope); err != nil {
		return err
	}
	labelsJSON, _ := json.Marshal(dto.Labels)
	annotationsJSON, _ := json.Marshal(dto.Annotations)
	channelsJSON, _ := json.Marshal(dto.NotificationChannels)

	updates := map[string]interface{}{
		"name":                  dto.Name,
		"datasource_id":         dto.DatasourceID,
		"enabled":               dto.Enabled,
		"query":                 dto.Query,
		"interval":              dto.Interval,
		"for_duration":          dto.ForDuration,
		"severity":              dto.Severity,
		"labels":                labelsJSON,
		"annotations":           annotationsJSON,
		"notification_channels": channelsJSON,
	}

	updates["dept_id"] = datasource.DeptID
	return s.db.Model(&record).Updates(updates).Error
}

func (s *AlertRuleService) validateNotificationChannels(channelIDs []uint64, deptID uint64, scope *common.DataScopeReq) error {
	if len(channelIDs) == 0 {
		return nil
	}
	var count int64
	if err := s.db.Model(&NotificationChannel{}).
		Scopes(database.WithDataScope(scope)).
		Where("id IN ? AND dept_id = ?", channelIDs, deptID).
		Count(&count).Error; err != nil {
		return err
	}
	if count != int64(len(channelIDs)) {
		return errors.New("observability.notification_channel_forbidden")
	}
	return nil
}

// Delete removes an alert rule.
func (s *AlertRuleService) Delete(id uint64, scopes ...*common.DataScopeReq) error {
	result := s.db.Scopes(database.WithDataScope(firstDataScope(scopes))).Where("id = ?", id).Delete(&AlertRule{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ToggleEnabled enables or disables an alert rule.
func (s *AlertRuleService) ToggleEnabled(id uint64, enabled bool, scopes ...*common.DataScopeReq) error {
	result := s.db.Model(&AlertRule{}).Scopes(database.WithDataScope(firstDataScope(scopes))).Where("id = ?", id).Update("enabled", enabled)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// RecordEvent records an alert event.
func (s *AlertRuleService) RecordEvent(ruleID uint64, value float64, status string) error {
	var rule AlertRule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		return err
	}

	event := &AlertEvent{
		DeptID:      rule.DeptID,
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		Status:      status,
		Value:       value,
		Query:       rule.Query,
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		FiredAt:     time.Now(),
	}

	if err := s.db.Create(event).Error; err != nil {
		return err
	}

	// Update rule statistics
	updates := map[string]interface{}{
		"last_evaluation": time.Now(),
	}
	if status == "firing" {
		updates["last_fired"] = time.Now()
		updates["fired_count"] = gorm.Expr("fired_count + 1")
	}

	return s.db.Model(&rule).Updates(updates).Error
}

// ListEvents returns alert events.
func (s *AlertRuleService) ListEvents(query AlertEventListQuery, scopes ...*common.DataScopeReq) (*AlertEventListResponse, error) {
	db := s.db.Model(&AlertEvent{}).Scopes(database.WithDataScope(firstDataScope(scopes)))

	if query.RuleID > 0 {
		db = db.Where("rule_id = ?", query.RuleID)
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}

	var records []AlertEvent
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("fired_at DESC").Offset(offset).Limit(query.PageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]AlertEventResponse, len(records))
	for i, r := range records {
		items[i] = *s.toEventResponse(&r)
	}

	return &AlertEventListResponse{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *AlertRuleService) toResponse(r *AlertRule, ds *PrometheusDatasource) *AlertRuleResponse {
	var labels map[string]string
	var annotations map[string]string
	var channels []uint64

	_ = json.Unmarshal(r.Labels, &labels)
	_ = json.Unmarshal(r.Annotations, &annotations)
	_ = json.Unmarshal(r.NotificationChannels, &channels)

	resp := &AlertRuleResponse{
		ID:                   r.ID,
		DeptID:               r.DeptID,
		Name:                 r.Name,
		DatasourceID:         r.DatasourceID,
		Enabled:              r.Enabled,
		Query:                r.Query,
		Interval:             r.Interval,
		ForDuration:          r.ForDuration,
		Severity:             r.Severity,
		Labels:               labels,
		Annotations:          annotations,
		NotificationChannels: channels,
		Status:               r.Status,
		LastEvaluation:       r.LastEvaluation,
		LastFired:            r.LastFired,
		FiredCount:           r.FiredCount,
		CreatedAt:            r.CreatedAt,
		CreatedBy:            r.CreatedBy,
	}

	if ds != nil {
		resp.DatasourceName = ds.Name
	}

	return resp
}

func (s *AlertRuleService) toEventResponse(e *AlertEvent) *AlertEventResponse {
	var labels map[string]string
	var annotations map[string]string

	_ = json.Unmarshal(e.Labels, &labels)
	_ = json.Unmarshal(e.Annotations, &annotations)

	return &AlertEventResponse{
		ID:          e.ID,
		DeptID:      e.DeptID,
		RuleID:      e.RuleID,
		RuleName:    e.RuleName,
		Severity:    e.Severity,
		Status:      e.Status,
		Value:       e.Value,
		Query:       e.Query,
		Labels:      labels,
		Annotations: annotations,
		FiredAt:     e.FiredAt,
		ResolvedAt:  e.ResolvedAt,
		CreatedAt:   e.CreatedAt,
	}
}
