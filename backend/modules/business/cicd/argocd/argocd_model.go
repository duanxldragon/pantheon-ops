package argocd

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ArgoCDServer represents an ArgoCD server configuration.
//
//nolint:revive // the exported model name keeps the Argo CD domain prefix for API clarity.
type ArgoCDServer struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	ClusterID   uint64         `gorm:"index" json:"clusterId"`
	Endpoint    string         `gorm:"size:256;not null" json:"endpoint"`
	AuthToken   string         `gorm:"size:512" json:"-"`                                // encrypted, never returned
	CACert      string         `gorm:"type:text" json:"-"`                               // PEM bundle for private CAs; verification is never disabled
	Status      string         `gorm:"size:32;not null;default:'unknown'" json:"status"` // connected, disconnected, unknown
	Version     string         `gorm:"size:64" json:"version,omitempty"`
	Installed   bool           `gorm:"default:false" json:"installed"`
	LastChecked *time.Time     `json:"lastChecked,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	CreatedBy   string         `gorm:"size:64" json:"createdBy"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_argocd_server table name.
func (ArgoCDServer) TableName() string {
	return "biz_argocd_server"
}

// ArgoCDRepository represents a Git repository configured in ArgoCD.
//
//nolint:revive // the exported model name keeps the Argo CD domain prefix for API clarity.
type ArgoCDRepository struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ServerID  uint64         `gorm:"not null;index:idx_server_repo" json:"serverId"`
	RepoURL   string         `gorm:"size:512;not null;index:idx_server_repo" json:"repoUrl"`
	Name      string         `gorm:"size:128" json:"name"`
	Branch    string         `gorm:"size:128" json:"branch"`
	Type      string         `gorm:"size:32;not null;default:'git'" json:"type"` // git, helm
	Username  string         `gorm:"size:128" json:"username,omitempty"`
	Password  string         `gorm:"size:512" json:"-"`  // encrypted
	SSHKey    string         `gorm:"size:4096" json:"-"` // encrypted
	Status    string         `gorm:"size:32;not null;default:'unknown'" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy string         `gorm:"size:64" json:"createdBy"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_argocd_repository table name.
func (ArgoCDRepository) TableName() string {
	return "biz_argocd_repository"
}

// ArgoCDApplication represents an ArgoCD application.
//
//nolint:revive // the exported model name keeps the Argo CD domain prefix for API clarity.
type ArgoCDApplication struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ServerID          uint64         `gorm:"not null;index:idx_server_name" json:"serverId"`
	Name              string         `gorm:"size:128;not null;index:idx_server_name" json:"name"`
	Project           string         `gorm:"size:128;not null;default:'default'" json:"project"`
	SourceRepoURL     string         `gorm:"size:512" json:"sourceRepoUrl"`
	SourcePath        string         `gorm:"size:256" json:"sourcePath"`
	SourceTargetRev   string         `gorm:"size:128" json:"sourceTargetRev"`
	DestinationServer string         `gorm:"size:256" json:"destinationServer"`
	DestinationNS     string         `gorm:"size:128" json:"destinationNamespace"`
	SyncPolicy        datatypes.JSON `gorm:"type:json" json:"syncPolicy,omitempty"`
	Status            string         `gorm:"size:32;not null;default:'Unknown'" json:"status"` // Synced, OutOfSync, Unknown, Error
	HealthStatus      string         `gorm:"size:32" json:"healthStatus,omitempty"`            // Healthy, Degraded, Progressing, Missing, Unknown
	SyncStatus        string         `gorm:"size:32" json:"syncStatus,omitempty"`
	LastError         string         `gorm:"type:text" json:"lastError,omitempty"`
	LastSyncedAt      *time.Time     `json:"lastSyncedAt,omitempty"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	CreatedBy         string         `gorm:"size:64" json:"createdBy"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_argocd_application table name.
func (ArgoCDApplication) TableName() string {
	return "biz_argocd_application"
}

// ArgoCDSyncHistory records each sync event for an application.
//
//nolint:revive // the exported model name keeps the Argo CD domain prefix for API clarity.
type ArgoCDSyncHistory struct {
	ID            uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ApplicationID uint64     `gorm:"not null;index" json:"applicationId"`
	Revision      string     `gorm:"size:128" json:"revision"`
	Status        string     `gorm:"size:32;not null" json:"status"` // Succeeded, Failed, Error
	Message       string     `gorm:"type:text" json:"message,omitempty"`
	StartedAt     time.Time  `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
}

// TableName returns the biz_argocd_sync_history table name.
func (ArgoCDSyncHistory) TableName() string {
	return "biz_argocd_sync_history"
}

// Constants
const (
	ArgoCDStatusConnected    = "connected"
	ArgoCDStatusDisconnected = "disconnected"
	ArgoCDStatusUnknown      = "unknown"

	AppStatusSynced    = "Synced"
	AppStatusOutOfSync = "OutOfSync"
	AppStatusUnknown   = "Unknown"
	AppStatusError     = "Error"

	HealthHealthy     = "Healthy"
	HealthDegraded    = "Degraded"
	HealthProgressing = "Progressing"
	HealthMissing     = "Missing"
	HealthUnknown     = "Unknown"
)
