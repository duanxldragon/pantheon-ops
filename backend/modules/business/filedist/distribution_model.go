package filedist

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Lifecycle statuses shared by distribution tasks and their per-target rows.
const (
	statusPending      = "pending"
	statusRunning      = "running"
	statusTransferring = "transferring"
	statusRetrying     = "retrying"
	statusCompleted    = "completed"
	statusFailed       = "failed"
)

// DistributionTask represents a file distribution task.
type DistributionTask struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string         `gorm:"size:128;not null" json:"name"`
	FileID       uint64         `gorm:"not null;index" json:"fileId"`
	Targets      datatypes.JSON `json:"targets"`                              // Array of target configurations
	Status       string         `gorm:"size:32;not null;index" json:"status"` // pending, running, completed, failed, paused
	Progress     int            `gorm:"default:0" json:"progress"`            // 0-100
	TotalTargets int            `gorm:"default:0" json:"totalTargets"`
	SuccessCount int            `gorm:"default:0" json:"successCount"`
	FailedCount  int            `gorm:"default:0" json:"failedCount"`
	Error        string         `gorm:"type:text" json:"error,omitempty"`
	ScheduledAt  *time.Time     `json:"scheduledAt,omitempty"` // For scheduled tasks
	StartedAt    *time.Time     `json:"startedAt,omitempty"`
	FinishedAt   *time.Time     `json:"finishedAt,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	CreatedBy    string         `gorm:"size:64" json:"createdBy"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the filedist_task table name.
func (DistributionTask) TableName() string {
	return "filedist_task"
}

// DistributionTarget represents a single target server.
type DistributionTarget struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID          uint64         `gorm:"not null;index" json:"taskId"`
	Host            string         `gorm:"size:128;not null" json:"host"`
	Port            int            `gorm:"default:22" json:"port"`
	TargetPath      string         `gorm:"size:512;not null" json:"targetPath"`
	Username        string         `gorm:"size:64" json:"username"`
	Password        string         `gorm:"size:512" json:"-"`                    // encrypted
	PrivateKey      string         `gorm:"type:text" json:"-"`                   // encrypted
	HostFingerprint string         `gorm:"size:128" json:"-"`                    // required SSH host key fingerprint (SHA256)
	Status          string         `gorm:"size:32;not null;index" json:"status"` // pending, transferring, completed, failed, retrying
	Progress        int            `gorm:"default:0" json:"progress"`
	Error           string         `gorm:"type:text" json:"error,omitempty"`
	RetryCount      int            `gorm:"default:0" json:"retryCount"`
	MaxRetries      int            `gorm:"not null" json:"maxRetries"`
	StartedAt       *time.Time     `json:"startedAt,omitempty"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the filedist_target table name.
func (DistributionTarget) TableName() string {
	return "filedist_target"
}

// DistributionTaskDTO for API requests
type DistributionTaskDTO struct {
	Name        string         `json:"name" binding:"required"`
	FileID      uint64         `json:"fileId" binding:"required"`
	Targets     []TargetConfig `json:"targets" binding:"required,min=1"`
	ScheduledAt *time.Time     `json:"scheduledAt"`
}

// TargetConfig describes where a distribution task pushes a file on one target.
type TargetConfig struct {
	Host            string `json:"host" binding:"required"`
	Port            int    `json:"port"`
	TargetPath      string `json:"targetPath" binding:"required"`
	Username        string `json:"username" binding:"required"`
	Password        string `json:"password"`
	PrivateKey      string `json:"privateKey"`
	HostFingerprint string `json:"hostFingerprint" binding:"required"` // SSH host key fingerprint (SHA256)
	MaxRetries      int    `json:"maxRetries"`
}

// DistributionTaskResponse is the distribution task shape returned to clients.
type DistributionTaskResponse struct {
	ID           uint64     `json:"id"`
	Name         string     `json:"name"`
	FileID       uint64     `json:"fileId"`
	FileName     string     `json:"fileName,omitempty"`
	Status       string     `json:"status"`
	Progress     int        `json:"progress"`
	TotalTargets int        `json:"totalTargets"`
	SuccessCount int        `json:"successCount"`
	FailedCount  int        `json:"failedCount"`
	Error        string     `json:"error,omitempty"`
	ScheduledAt  *time.Time `json:"scheduledAt,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	CreatedBy    string     `json:"createdBy"`
}

// DistributionTargetResponse is the per-target progress shape returned to clients.
type DistributionTargetResponse struct {
	ID         uint64     `json:"id"`
	TaskID     uint64     `json:"taskId"`
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	TargetPath string     `json:"targetPath"`
	Username   string     `json:"username"`
	Status     string     `json:"status"`
	Progress   int        `json:"progress"`
	Error      string     `json:"error,omitempty"`
	RetryCount int        `json:"retryCount"`
	MaxRetries int        `json:"maxRetries"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// TaskListQuery filters and paginates distribution tasks.
type TaskListQuery struct {
	Status   string `form:"status"`
	FileID   uint64 `form:"fileId"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// TaskListResponse is the paged list of distribution tasks.
type TaskListResponse struct {
	Items    []DistributionTaskResponse `json:"items"`
	Total    int64                      `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"pageSize"`
}
