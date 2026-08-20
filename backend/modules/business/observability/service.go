package observability

import (
	"errors"
	"fmt"
)

// Service provides business logic for observability module.
type Service struct {
	repo *Repository
}

// NewService creates a new observability service.
func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// MetricSource operations

// CreateMetricSource creates a new metric source.
func (s *Service) CreateMetricSource(source *MetricSource) error {
	// Validate
	if source.Name == "" {
		return errors.New("name is required")
	}
	if source.Type == "" {
		return errors.New("type is required")
	}
	if source.Endpoint == "" {
		return errors.New("endpoint is required")
	}

	// Set default status
	if source.Status == "" {
		source.Status = "active"
	}

	return s.repo.CreateMetricSource(source)
}

// GetMetricSource retrieves a metric source by ID.
func (s *Service) GetMetricSource(id uint64) (*MetricSource, error) {
	if id == 0 {
		return nil, errors.New("invalid metric source ID")
	}
	return s.repo.GetMetricSource(id)
}

// ListMetricSources retrieves metric sources with pagination.
func (s *Service) ListMetricSources(page, pageSize int, filters map[string]interface{}) ([]*MetricSource, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListMetricSources(page, pageSize, filters)
}

// UpdateMetricSource updates a metric source.
func (s *Service) UpdateMetricSource(source *MetricSource) error {
	if source.ID == 0 {
		return errors.New("invalid metric source ID")
	}

	// Check existence
	existing, err := s.repo.GetMetricSource(source.ID)
	if err != nil {
		return fmt.Errorf("metric source not found: %w", err)
	}

	// Update fields
	existing.Name = source.Name
	existing.Endpoint = source.Endpoint
	existing.CredentialRef = source.CredentialRef
	existing.Status = source.Status
	existing.Config = source.Config
	existing.Remark = source.Remark
	existing.UpdatedBy = source.UpdatedBy

	return s.repo.UpdateMetricSource(existing)
}

// DeleteMetricSource deletes a metric source.
func (s *Service) DeleteMetricSource(id uint64) error {
	if id == 0 {
		return errors.New("invalid metric source ID")
	}

	// Check if any alert rules depend on this source
	// TODO: Add check logic in V2 Sprint 1 Week 3

	return s.repo.DeleteMetricSource(id)
}

// AlertRule operations

// CreateAlertRule creates a new alert rule.
func (s *Service) CreateAlertRule(rule *AlertRule) error {
	// Validate
	if rule.Name == "" {
		return errors.New("name is required")
	}
	if rule.MetricSourceID == 0 {
		return errors.New("metric source ID is required")
	}
	if rule.PromQL == "" {
		return errors.New("PromQL is required")
	}
	if rule.Severity == "" {
		return errors.New("severity is required")
	}

	// Validate metric source exists
	_, err := s.repo.GetMetricSource(rule.MetricSourceID)
	if err != nil {
		return fmt.Errorf("invalid metric source: %w", err)
	}

	// Set default status
	if rule.Status == "" {
		rule.Status = "enabled"
	}

	// Set default duration
	if rule.Duration == "" {
		rule.Duration = "5m"
	}

	return s.repo.CreateAlertRule(rule)
}

// GetAlertRule retrieves an alert rule by ID.
func (s *Service) GetAlertRule(id uint64) (*AlertRule, error) {
	if id == 0 {
		return nil, errors.New("invalid alert rule ID")
	}
	return s.repo.GetAlertRule(id)
}

// ListAlertRules retrieves alert rules with pagination.
func (s *Service) ListAlertRules(page, pageSize int, filters map[string]interface{}) ([]*AlertRule, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAlertRules(page, pageSize, filters)
}

// UpdateAlertRule updates an alert rule.
func (s *Service) UpdateAlertRule(rule *AlertRule) error {
	if rule.ID == 0 {
		return errors.New("invalid alert rule ID")
	}

	// Check existence
	existing, err := s.repo.GetAlertRule(rule.ID)
	if err != nil {
		return fmt.Errorf("alert rule not found: %w", err)
	}

	// Update fields
	existing.Name = rule.Name
	existing.PromQL = rule.PromQL
	existing.Duration = rule.Duration
	existing.Severity = rule.Severity
	existing.Labels = rule.Labels
	existing.Annotations = rule.Annotations
	existing.NotificationChannelIDs = rule.NotificationChannelIDs
	existing.Status = rule.Status
	existing.Remark = rule.Remark
	existing.UpdatedBy = rule.UpdatedBy

	return s.repo.UpdateAlertRule(existing)
}

// DeleteAlertRule deletes an alert rule.
func (s *Service) DeleteAlertRule(id uint64) error {
	if id == 0 {
		return errors.New("invalid alert rule ID")
	}
	return s.repo.DeleteAlertRule(id)
}

// ValidatePromQL validates a PromQL expression.
// TODO: Implement in V2 Sprint 1 Week 3 - call Prometheus API to validate
func (s *Service) ValidatePromQL(promql string) error {
	if promql == "" {
		return errors.New("PromQL is required")
	}
	// TODO: Call Prometheus /api/v1/query with expr=promql to validate syntax
	return nil
}

// NotificationChannel operations

// CreateNotificationChannel creates a new notification channel.
func (s *Service) CreateNotificationChannel(channel *NotificationChannel) error {
	// Validate
	if channel.Name == "" {
		return errors.New("name is required")
	}
	if channel.Type == "" {
		return errors.New("type is required")
	}
	if channel.Config == nil {
		return errors.New("config is required")
	}

	// Set default status
	if channel.Status == "" {
		channel.Status = "active"
	}

	return s.repo.CreateNotificationChannel(channel)
}

// GetNotificationChannel retrieves a notification channel by ID.
func (s *Service) GetNotificationChannel(id uint64) (*NotificationChannel, error) {
	if id == 0 {
		return nil, errors.New("invalid notification channel ID")
	}
	return s.repo.GetNotificationChannel(id)
}

// ListNotificationChannels retrieves notification channels with pagination.
func (s *Service) ListNotificationChannels(page, pageSize int, filters map[string]interface) ([]*NotificationChannel, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListNotificationChannels(page, pageSize, filters)
}

// UpdateNotificationChannel updates a notification channel.
func (s *Service) UpdateNotificationChannel(channel *NotificationChannel) error {
	if channel.ID == 0 {
		return errors.New("invalid notification channel ID")
	}

	// Check existence
	existing, err := s.repo.GetNotificationChannel(channel.ID)
	if err != nil {
		return fmt.Errorf("notification channel not found: %w", err)
	}

	// Update fields
	existing.Name = channel.Name
	existing.Config = channel.Config
	existing.Status = channel.Status
	existing.Remark = channel.Remark
	existing.UpdatedBy = channel.UpdatedBy

	return s.repo.UpdateNotificationChannel(existing)
}

// DeleteNotificationChannel deletes a notification channel.
func (s *Service) DeleteNotificationChannel(id uint64) error {
	if id == 0 {
		return errors.New("invalid notification channel ID")
	}
	return s.repo.DeleteNotificationChannel(id)
}

// TestNotificationChannel tests a notification channel by sending a test message.
// TODO: Implement in V2 Sprint 1 Week 4
func (s *Service) TestNotificationChannel(id uint64) error {
	channel, err := s.repo.GetNotificationChannel(id)
	if err != nil {
		return fmt.Errorf("channel not found: %w", err)
	}

	// TODO: Send test notification based on channel.Type
	_ = channel

	return errors.New("not implemented yet - V2 Sprint 1 Week 4")
}
