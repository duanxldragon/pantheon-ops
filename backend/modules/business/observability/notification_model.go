package observability

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// NotificationChannel represents a notification channel configuration.
type NotificationChannel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID    uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Name      string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Type      string         `gorm:"size:32;not null" json:"type"` // email, webhook, slack, dingtalk, feishu
	Enabled   bool           `gorm:"default:true" json:"enabled"`
	Config    datatypes.JSON `json:"config"` // Channel-specific configuration
	Status    string         `gorm:"size:32;not null;default:'active'" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy string         `gorm:"size:64" json:"createdBy"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the obs_notification_channel table name.
func (NotificationChannel) TableName() string {
	return "obs_notification_channel"
}

// NotificationChannelDTO for API requests
type NotificationChannelDTO struct {
	Name    string                 `json:"name" binding:"required"`
	Type    string                 `json:"type" binding:"required,oneof=email webhook slack dingtalk feishu"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config" binding:"required"`
}

// EmailConfig carries the SMTP settings of an email notification channel.
type EmailConfig struct {
	Recipients []string `json:"recipients"`
	SMTPHost   string   `json:"smtpHost"`
	SMTPPort   int      `json:"smtpPort"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	FromAddr   string   `json:"fromAddr"`
	UseTLS     bool     `json:"useTLS"`
}

// WebhookConfig carries the endpoint and delivery options of a webhook notification channel.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"` // POST, PUT
	Headers map[string]string `json:"headers"`
	Timeout int               `json:"timeout"`
}

// SlackConfig carries the webhook used by a Slack notification channel.
type SlackConfig struct {
	WebhookURL string `json:"webhookUrl"`
	Channel    string `json:"channel"`
	Username   string `json:"username"`
}

// DingTalkConfig carries the webhook and signing secret of a DingTalk notification channel.
type DingTalkConfig struct {
	WebhookURL string   `json:"webhookUrl"`
	Secret     string   `json:"secret"`
	AtMobiles  []string `json:"atMobiles"`
	IsAtAll    bool     `json:"isAtAll"`
}

// FeishuConfig carries the webhook of a Feishu notification channel.
type FeishuConfig struct {
	WebhookURL string `json:"webhookUrl"`
	Secret     string `json:"secret"`
}

// NotificationChannelResponse is the notification channel shape returned to clients.
type NotificationChannelResponse struct {
	ID        uint64                 `json:"id"`
	DeptID    uint64                 `json:"deptId"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"createdAt"`
	CreatedBy string                 `json:"createdBy"`
}

// NotificationChannelListResponse is the list of configured notification channels.
type NotificationChannelListResponse struct {
	Items []NotificationChannelResponse `json:"items"`
	Total int64                         `json:"total"`
}

// NotificationLog represents a sent notification record.
type NotificationLog struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID    uint64         `gorm:"column:dept_id;index" json:"deptId"`
	ChannelID uint64         `gorm:"not null;index" json:"channelId"`
	EventID   uint64         `gorm:"not null;index" json:"eventId"`
	RuleID    uint64         `gorm:"not null;index" json:"ruleId"`
	RuleName  string         `gorm:"size:128" json:"ruleName"`
	Severity  string         `gorm:"size:32" json:"severity"`
	Status    string         `gorm:"size:32;not null" json:"status"` // success, failed
	Error     string         `gorm:"type:text" json:"error,omitempty"`
	SentAt    time.Time      `gorm:"not null;index" json:"sentAt"`
	CreatedAt time.Time      `json:"createdAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the obs_notification_log table name.
func (NotificationLog) TableName() string {
	return "obs_notification_log"
}

// NotificationLogResponse is one delivery attempt returned to clients.
type NotificationLogResponse struct {
	ID        uint64    `json:"id"`
	ChannelID uint64    `json:"channelId"`
	EventID   uint64    `json:"eventId"`
	RuleID    uint64    `json:"ruleId"`
	RuleName  string    `json:"ruleName"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	SentAt    time.Time `json:"sentAt"`
	CreatedAt time.Time `json:"createdAt"`
}
