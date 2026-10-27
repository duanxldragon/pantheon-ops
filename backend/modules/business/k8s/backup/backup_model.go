package backup

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// VeleroBackup 表示 Velero 备份记录（对应 K8s Backup CRD）
type VeleroBackup struct {
	ID                   uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                 string         `gorm:"size:128;not null;index:idx_cluster_name" json:"name"`
	ClusterID            uint64         `gorm:"not null;index:idx_cluster_name" json:"clusterId"`
	BackupType           string         `gorm:"size:32;not null;default:full" json:"backupType"` // full, namespace, resource
	IncludedNamespaces   datatypes.JSON `gorm:"type:json" json:"includedNamespaces,omitempty"`
	ExcludedNamespaces   datatypes.JSON `gorm:"type:json" json:"excludedNamespaces,omitempty"`
	IncludedResources    datatypes.JSON `gorm:"type:json" json:"includedResources,omitempty"`
	ExcludedResources    datatypes.JSON `gorm:"type:json" json:"excludedResources,omitempty"`
	LabelSelector        string         `gorm:"size:256" json:"labelSelector,omitempty"`
	IncludePVs           bool           `gorm:"default:true" json:"includePvs"`
	TTL                  int            `gorm:"default:720" json:"ttl"`                     // 保留时长（小时）
	Status               string         `gorm:"size:32;not null;default:New" json:"status"` // New, InProgress, Completed, Failed
	Phase                string         `gorm:"size:32" json:"phase,omitempty"`             // Velero phase
	StartTime            *time.Time     `json:"startTime,omitempty"`
	CompletionTime       *time.Time     `json:"completionTime,omitempty"`
	Expiration           *time.Time     `json:"expiration,omitempty"`
	Errors               int            `gorm:"default:0" json:"errors"`
	Warnings             int            `gorm:"default:0" json:"warnings"`
	BackupSize           int64          `json:"backupSize,omitempty"`
	StorageLocation      string         `gorm:"size:128" json:"storageLocation,omitempty"`
	VolumeSnapshotsCount int            `gorm:"default:0" json:"volumeSnapshotsCount"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	CreatedBy            string         `gorm:"size:64" json:"createdBy"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_velero_backup table name.
func (VeleroBackup) TableName() string {
	return "biz_velero_backup"
}

// VeleroSchedule 表示 Velero 定时备份（对应 K8s Schedule CRD）
type VeleroSchedule struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string         `gorm:"size:128;not null;index:idx_cluster_name" json:"name"`
	ClusterID        uint64         `gorm:"not null;index:idx_cluster_name" json:"clusterId"`
	CronExpression   string         `gorm:"size:64;not null" json:"cronExpression"` // 0 2 * * *
	Enabled          bool           `gorm:"default:true" json:"enabled"`
	BackupTemplate   datatypes.JSON `gorm:"type:json;not null" json:"backupTemplate"`
	LastBackupTime   *time.Time     `json:"lastBackupTime,omitempty"`
	NextBackupTime   *time.Time     `json:"nextBackupTime,omitempty"`
	LastBackupStatus string         `gorm:"size:32" json:"lastBackupStatus,omitempty"`
	Phase            string         `gorm:"size:32" json:"phase,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	CreatedBy        string         `gorm:"size:64" json:"createdBy"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_velero_schedule table name.
func (VeleroSchedule) TableName() string {
	return "biz_velero_schedule"
}

// VeleroRestore 表示 Velero 恢复记录（对应 K8s Restore CRD）
type VeleroRestore struct {
	ID                 uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name               string         `gorm:"size:128;not null;index:idx_cluster_name" json:"name"`
	ClusterID          uint64         `gorm:"not null;index:idx_cluster_name" json:"clusterId"`
	BackupName         string         `gorm:"size:128;not null" json:"backupName"`
	NamespaceMapping   datatypes.JSON `gorm:"type:json" json:"namespaceMapping,omitempty"`
	IncludedNamespaces datatypes.JSON `gorm:"type:json" json:"includedNamespaces,omitempty"`
	ExcludedNamespaces datatypes.JSON `gorm:"type:json" json:"excludedNamespaces,omitempty"`
	IncludedResources  datatypes.JSON `gorm:"type:json" json:"includedResources,omitempty"`
	ExcludedResources  datatypes.JSON `gorm:"type:json" json:"excludedResources,omitempty"`
	RestorePVs         bool           `gorm:"default:true" json:"restorePvs"`
	Status             string         `gorm:"size:32;not null;default:New" json:"status"`
	Phase              string         `gorm:"size:32" json:"phase,omitempty"`
	StartTime          *time.Time     `json:"startTime,omitempty"`
	CompletionTime     *time.Time     `json:"completionTime,omitempty"`
	Errors             int            `gorm:"default:0" json:"errors"`
	Warnings           int            `gorm:"default:0" json:"warnings"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
	CreatedBy          string         `gorm:"size:64" json:"createdBy"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_velero_restore table name.
func (VeleroRestore) TableName() string {
	return "biz_velero_restore"
}

// VeleroStorageLocation 表示备份存储位置（对应 BackupStorageLocation CRD）
type VeleroStorageLocation struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:128;not null;unique" json:"name"`
	Provider        string         `gorm:"size:32;not null" json:"provider"` // aws, azure, gcp, minio
	Bucket          string         `gorm:"size:128;not null" json:"bucket"`
	Region          string         `gorm:"size:64" json:"region,omitempty"`
	Endpoint        string         `gorm:"size:256" json:"endpoint,omitempty"` // MinIO/S3 endpoint
	AccessKeyID     string         `gorm:"size:128" json:"-"`                  // 加密存储，不返回
	SecretAccessKey string         `gorm:"size:256" json:"-"`                  // 加密存储，不返回
	Config          datatypes.JSON `gorm:"type:json" json:"config,omitempty"`
	IsDefault       bool           `gorm:"default:false" json:"isDefault"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the biz_velero_storage_location table name.
func (VeleroStorageLocation) TableName() string {
	return "biz_velero_storage_location"
}

// Constants
const (
	// Backup Type
	BackupTypeFull      = "full"
	BackupTypeNamespace = "namespace"
	BackupTypeResource  = "resource"

	// Status
	StatusNew        = "New"
	StatusInProgress = "InProgress"
	StatusCompleted  = "Completed"
	StatusFailed     = "Failed"

	// Provider
	ProviderAWS   = "aws"
	ProviderAzure = "azure"
	ProviderGCP   = "gcp"
	ProviderMinio = "minio"
)
