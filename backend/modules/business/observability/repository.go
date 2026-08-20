package observability

import (
	"gorm.io/gorm"
)

// Repository provides data access for observability module.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new observability repository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// MetricSource operations

// CreateMetricSource creates a new metric source.
func (r *Repository) CreateMetricSource(source *MetricSource) error {
	return r.db.Create(source).Error
}

// GetMetricSource retrieves a metric source by ID.
func (r *Repository) GetMetricSource(id uint64) (*MetricSource, error) {
	var source MetricSource
	err := r.db.First(&source, id).Error
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// ListMetricSources retrieves metric sources with pagination.
func (r *Repository) ListMetricSources(page, pageSize int, filters map[string]interface{}) ([]*MetricSource, int64, error) {
	var sources []*MetricSource
	var total int64

	query := r.db.Model(&MetricSource{})

	// Apply filters
	if businessScopeID, ok := filters["business_scope_id"].(uint64); ok && businessScopeID > 0 {
		query = query.Where("business_scope_id = ?", businessScopeID)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}
	if sourceType, ok := filters["type"].(string); ok && sourceType != "" {
		query = query.Where("type = ?", sourceType)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&sources).Error; err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// UpdateMetricSource updates a metric source.
func (r *Repository) UpdateMetricSource(source *MetricSource) error {
	return r.db.Save(source).Error
}

// DeleteMetricSource soft-deletes a metric source.
func (r *Repository) DeleteMetricSource(id uint64) error {
	return r.db.Delete(&MetricSource{}, id).Error
}

// AlertRule operations

// CreateAlertRule creates a new alert rule.
func (r *Repository) CreateAlertRule(rule *AlertRule) error {
	return r.db.Create(rule).Error
}

// GetAlertRule retrieves an alert rule by ID.
func (r *Repository) GetAlertRule(id uint64) (*AlertRule, error) {
	var rule AlertRule
	err := r.db.First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListAlertRules retrieves alert rules with pagination.
func (r *Repository) ListAlertRules(page, pageSize int, filters map[string]interface{}) ([]*AlertRule, int64, error) {
	var rules []*AlertRule
	var total int64

	query := r.db.Model(&AlertRule{})

	// Apply filters
	if businessScopeID, ok := filters["business_scope_id"].(uint64); ok && businessScopeID > 0 {
		query = query.Where("business_scope_id = ?", businessScopeID)
	}
	if environment, ok := filters["environment"].(string); ok && environment != "" {
		query = query.Where("environment = ?", environment)
	}
	if severity, ok := filters["severity"].(string); ok && severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	return rules, total, nil
}

// UpdateAlertRule updates an alert rule.
func (r *Repository) UpdateAlertRule(rule *AlertRule) error {
	return r.db.Save(rule).Error
}

// DeleteAlertRule soft-deletes an alert rule.
func (r *Repository) DeleteAlertRule(id uint64) error {
	return r.db.Delete(&AlertRule{}, id).Error
}

// AlertRecord operations

// CreateAlertRecord creates a new alert record.
func (r *Repository) CreateAlertRecord(record *AlertRecord) error {
	return r.db.Create(record).Error
}

// GetAlertRecord retrieves an alert record by ID.
func (r *Repository) GetAlertRecord(id uint64) (*AlertRecord, error) {
	var record AlertRecord
	err := r.db.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListAlertRecords retrieves alert records with pagination.
func (r *Repository) ListAlertRecords(page, pageSize int, filters map[string]interface{}) ([]*AlertRecord, int64, error) {
	var records []*AlertRecord
	var total int64

	query := r.db.Model(&AlertRecord{})

	// Apply filters
	if alertRuleID, ok := filters["alert_rule_id"].(uint64); ok && alertRuleID > 0 {
		query = query.Where("alert_rule_id = ?", alertRuleID)
	}
	if severity, ok := filters["severity"].(string); ok && severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if resolved, ok := filters["resolved"].(bool); ok {
		if resolved {
			query = query.Where("resolved_at IS NOT NULL")
		} else {
			query = query.Where("resolved_at IS NULL")
		}
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("fired_at DESC").Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// UpdateAlertRecord updates an alert record.
func (r *Repository) UpdateAlertRecord(record *AlertRecord) error {
	return r.db.Save(record).Error
}

// NotificationChannel operations

// CreateNotificationChannel creates a new notification channel.
func (r *Repository) CreateNotificationChannel(channel *NotificationChannel) error {
	return r.db.Create(channel).Error
}

// GetNotificationChannel retrieves a notification channel by ID.
func (r *Repository) GetNotificationChannel(id uint64) (*NotificationChannel, error) {
	var channel NotificationChannel
	err := r.db.First(&channel, id).Error
	if err != nil {
		return nil, err
	}
	return &channel, nil
}

// ListNotificationChannels retrieves notification channels with pagination.
func (r *Repository) ListNotificationChannels(page, pageSize int, filters map[string]interface{}) ([]*NotificationChannel, int64, error) {
	var channels []*NotificationChannel
	var total int64

	query := r.db.Model(&NotificationChannel{})

	// Apply filters
	if businessScopeID, ok := filters["business_scope_id"].(uint64); ok && businessScopeID > 0 {
		query = query.Where("business_scope_id = ?", businessScopeID)
	}
	if channelType, ok := filters["type"].(string); ok && channelType != "" {
		query = query.Where("type = ?", channelType)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// UpdateNotificationChannel updates a notification channel.
func (r *Repository) UpdateNotificationChannel(channel *NotificationChannel) error {
	return r.db.Save(channel).Error
}

// DeleteNotificationChannel soft-deletes a notification channel.
func (r *Repository) DeleteNotificationChannel(id uint64) error {
	return r.db.Delete(&NotificationChannel{}, id).Error
}

// LogSource operations

// CreateLogSource creates a new log source.
func (r *Repository) CreateLogSource(source *LogSource) error {
	return r.db.Create(source).Error
}

// GetLogSource retrieves a log source by ID.
func (r *Repository) GetLogSource(id uint64) (*LogSource, error) {
	var source LogSource
	err := r.db.First(&source, id).Error
	if err != nil {
		return nil, err
	}
	return &source, nil
}

// ListLogSources retrieves log sources with pagination.
func (r *Repository) ListLogSources(page, pageSize int, filters map[string]interface{}) ([]*LogSource, int64, error) {
	var sources []*LogSource
	var total int64

	query := r.db.Model(&LogSource{})

	// Apply filters
	if businessScopeID, ok := filters["business_scope_id"].(uint64); ok && businessScopeID > 0 {
		query = query.Where("business_scope_id = ?", businessScopeID)
	}
	if sourceType, ok := filters["type"].(string); ok && sourceType != "" {
		query = query.Where("type = ?", sourceType)
	}
	if status, ok := filters["status"].(string); ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Paginate
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&sources).Error; err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// UpdateLogSource updates a log source.
func (r *Repository) UpdateLogSource(source *LogSource) error {
	return r.db.Save(source).Error
}

// DeleteLogSource soft-deletes a log source.
func (r *Repository) DeleteLogSource(id uint64) error {
	return r.db.Delete(&LogSource{}, id).Error
}
