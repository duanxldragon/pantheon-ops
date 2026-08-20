package observability

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MetricSource represents a metrics backend (Prometheus, VictoriaMetrics, etc.)
type MetricSource struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null" json:"name"`
	Type            string         `gorm:"size:64;not null;index" json:"type"` // prometheus, victoria-metrics
	Endpoint        string         `gorm:"size:512;not null" json:"endpoint"`
	CredentialRef   string         `gorm:"size:255" json:"credentialRef"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Status          string         `gorm:"size:32;not null;default:active;index" json:"status"`
	Config          datatypes.JSON `gorm:"type:json" json:"config"`
	Remark          string         `gorm:"type:text" json:"remark"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the metric source table name.
func (MetricSource) TableName() string {
	return "biz_observability_metric_source"
}

// AlertRule represents an alert rule configuration.
type AlertRule struct {
	ID                     uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	MetricSourceID         uint64         `gorm:"column:metric_source_id;not null;index" json:"metricSourceId"`
	Name                   string         `gorm:"size:255;not null" json:"name"`
	BusinessScopeID        uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	DeptID                 uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Environment            string         `gorm:"size:32;index" json:"environment"` // prod, test, dev
	PromQL                 string         `gorm:"type:text;not null" json:"promql"`
	Duration               string         `gorm:"size:32" json:"duration"` // 5m, 10m, 1h
	Severity               string         `gorm:"size:32;not null;index" json:"severity"` // critical, warning, info
	Labels                 datatypes.JSON `gorm:"type:json" json:"labels"`
	Annotations            datatypes.JSON `gorm:"type:json" json:"annotations"`
	NotificationChannelIDs datatypes.JSON `gorm:"type:json" json:"notificationChannelIds"` // [1, 2, 3]
	Status                 string         `gorm:"size:32;not null;default:enabled;index" json:"status"` // enabled, disabled
	Remark                 string         `gorm:"type:text" json:"remark"`
	CreatedAt              time.Time      `json:"createdAt"`
	UpdatedAt              time.Time      `json:"updatedAt"`
	CreatedBy              string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy              string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the alert rule table name.
func (AlertRule) TableName() string {
	return "biz_observability_alert_rule"
}

// AlertRecord represents an alert firing history record.
type AlertRecord struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	AlertRuleID      uint64         `gorm:"column:alert_rule_id;not null;index" json:"alertRuleId"`
	AlertRuleName    string         `gorm:"size:255" json:"alertRuleName"`
	FiredAt          time.Time      `gorm:"not null;index" json:"firedAt"`
	ResolvedAt       *time.Time     `gorm:"index" json:"resolvedAt"`
	Severity         string         `gorm:"size:32;not null;index" json:"severity"`
	Labels           datatypes.JSON `gorm:"type:json" json:"labels"`
	Annotations      datatypes.JSON `gorm:"type:json" json:"annotations"`
	NotificationsSent datatypes.JSON `gorm:"type:json" json:"notificationsSent"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the alert record table name.
func (AlertRecord) TableName() string {
	return "biz_observability_alert_record"
}

// NotificationChannel represents a notification delivery channel.
type NotificationChannel struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null" json:"name"`
	Type            string         `gorm:"size:64;not null;index" json:"type"` // email, dingtalk, wechat, slack
	Config          datatypes.JSON `gorm:"type:json;not null" json:"config"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Status          string         `gorm:"size:32;not null;default:active;index" json:"status"`
	Remark          string         `gorm:"type:text" json:"remark"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the notification channel table name.
func (NotificationChannel) TableName() string {
	return "biz_observability_notification_channel"
}

// LogSource represents a log aggregation backend (Loki, Elasticsearch, etc.)
type LogSource struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null" json:"name"`
	Type            string         `gorm:"size:64;not null;index" json:"type"` // loki, elasticsearch
	Endpoint        string         `gorm:"size:512;not null" json:"endpoint"`
	CredentialRef   string         `gorm:"size:255" json:"credentialRef"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Status          string         `gorm:"size:32;not null;default:active;index" json:"status"`
	Config          datatypes.JSON `gorm:"type:json" json:"config"`
	Remark          string         `gorm:"type:text" json:"remark"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the log source table name.
func (LogSource) TableName() string {
	return "biz_observability_log_source"
}
