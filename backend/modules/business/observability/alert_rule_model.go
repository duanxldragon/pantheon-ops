package observability

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AlertRule represents an alert rule configuration.
type AlertRule struct {
	ID                   uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID               uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Name                 string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	DatasourceID         uint64         `gorm:"not null;index" json:"datasourceId"`
	Enabled              bool           `gorm:"default:true" json:"enabled"`
	Query                string         `gorm:"type:text;not null" json:"query"`         // PromQL query
	Interval             int            `gorm:"not null;default:60" json:"interval"`     // seconds
	ForDuration          int            `gorm:"not null;default:300" json:"forDuration"` // seconds (5 min default)
	Severity             string         `gorm:"size:32;not null" json:"severity"`        // critical, warning, info
	Labels               datatypes.JSON `json:"labels"`
	Annotations          datatypes.JSON `json:"annotations"`
	NotificationChannels datatypes.JSON `json:"notificationChannels"`                            // array of channel IDs
	Status               string         `gorm:"size:32;not null;default:'active'" json:"status"` // active, paused, error
	LastEvaluation       *time.Time     `json:"lastEvaluation,omitempty"`
	LastFired            *time.Time     `json:"lastFired,omitempty"`
	FiredCount           int            `gorm:"default:0" json:"firedCount"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	CreatedBy            string         `gorm:"size:64" json:"createdBy"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AlertRule) TableName() string {
	return "obs_alert_rule"
}

// AlertRuleDTO for API requests
type AlertRuleDTO struct {
	Name                 string            `json:"name" binding:"required"`
	DatasourceID         uint64            `json:"datasourceId" binding:"required"`
	Enabled              bool              `json:"enabled"`
	Query                string            `json:"query" binding:"required"`
	Interval             int               `json:"interval" binding:"required,min=10"`
	ForDuration          int               `json:"forDuration" binding:"required,min=0"`
	Severity             string            `json:"severity" binding:"required,oneof=critical warning info"`
	Labels               map[string]string `json:"labels"`
	Annotations          map[string]string `json:"annotations"`
	NotificationChannels []uint64          `json:"notificationChannels"`
}

type AlertRuleResponse struct {
	ID                   uint64            `json:"id"`
	DeptID               uint64            `json:"deptId"`
	Name                 string            `json:"name"`
	DatasourceID         uint64            `json:"datasourceId"`
	DatasourceName       string            `json:"datasourceName,omitempty"`
	Enabled              bool              `json:"enabled"`
	Query                string            `json:"query"`
	Interval             int               `json:"interval"`
	ForDuration          int               `json:"forDuration"`
	Severity             string            `json:"severity"`
	Labels               map[string]string `json:"labels"`
	Annotations          map[string]string `json:"annotations"`
	NotificationChannels []uint64          `json:"notificationChannels"`
	Status               string            `json:"status"`
	LastEvaluation       *time.Time        `json:"lastEvaluation,omitempty"`
	LastFired            *time.Time        `json:"lastFired,omitempty"`
	FiredCount           int               `json:"firedCount"`
	CreatedAt            time.Time         `json:"createdAt"`
	CreatedBy            string            `json:"createdBy"`
}

type AlertRuleListQuery struct {
	Keyword      string `form:"keyword"`
	DatasourceID uint64 `form:"datasourceId"`
	Severity     string `form:"severity"`
	Status       string `form:"status"`
	Enabled      *bool  `form:"enabled"`
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
}

type AlertRuleListResponse struct {
	Items    []AlertRuleResponse `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

// AlertEvent represents a fired alert event.
type AlertEvent struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID      uint64         `gorm:"column:dept_id;index" json:"deptId"`
	RuleID      uint64         `gorm:"not null;index" json:"ruleId"`
	RuleName    string         `gorm:"size:128;not null" json:"ruleName"`
	Severity    string         `gorm:"size:32;not null" json:"severity"`
	Status      string         `gorm:"size:32;not null" json:"status"` // firing, resolved
	Value       float64        `json:"value"`
	Query       string         `gorm:"type:text" json:"query"`
	Labels      datatypes.JSON `json:"labels"`
	Annotations datatypes.JSON `json:"annotations"`
	FiredAt     time.Time      `gorm:"not null;index" json:"firedAt"`
	ResolvedAt  *time.Time     `json:"resolvedAt,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (AlertEvent) TableName() string {
	return "obs_alert_event"
}

type AlertEventResponse struct {
	ID          uint64            `json:"id"`
	DeptID      uint64            `json:"deptId"`
	RuleID      uint64            `json:"ruleId"`
	RuleName    string            `json:"ruleName"`
	Severity    string            `json:"severity"`
	Status      string            `json:"status"`
	Value       float64           `json:"value"`
	Query       string            `json:"query"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	FiredAt     time.Time         `json:"firedAt"`
	ResolvedAt  *time.Time        `json:"resolvedAt,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
}

type AlertEventListQuery struct {
	RuleID   uint64 `form:"ruleId"`
	Severity string `form:"severity"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type AlertEventListResponse struct {
	Items    []AlertEventResponse `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}
