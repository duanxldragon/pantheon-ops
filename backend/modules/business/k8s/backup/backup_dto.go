package backup

import "time"

// --- 存储位置 ---

// CreateStorageLocationRequest 创建存储位置请求
type CreateStorageLocationRequest struct {
	Name            string                 `json:"name" binding:"required,max=128"`
	Provider        string                 `json:"provider" binding:"required,oneof=aws azure gcp minio"`
	Bucket          string                 `json:"bucket" binding:"required,max=128"`
	Region          string                 `json:"region,omitempty"`
	Endpoint        string                 `json:"endpoint,omitempty"`
	AccessKeyID     string                 `json:"accessKeyId,omitempty"`
	SecretAccessKey string                 `json:"secretAccessKey,omitempty"`
	Config          map[string]interface{} `json:"config,omitempty"`
	IsDefault       bool                   `json:"isDefault"`
}

// StorageLocationResponse 存储位置响应
type StorageLocationResponse struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	Bucket    string    `json:"bucket"`
	Region    string    `json:"region,omitempty"`
	Endpoint  string    `json:"endpoint,omitempty"`
	IsDefault bool      `json:"isDefault"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// --- 备份 ---

// CreateBackupRequest 创建备份请求
type CreateBackupRequest struct {
	Name               string   `json:"name" binding:"required,max=128"`
	ClusterID          uint64   `json:"clusterId" binding:"required"`
	BackupType         string   `json:"backupType" binding:"required,oneof=full namespace resource"`
	IncludedNamespaces []string `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string `json:"includedResources,omitempty"`
	ExcludedResources  []string `json:"excludedResources,omitempty"`
	LabelSelector      string   `json:"labelSelector,omitempty"`
	IncludePVs         bool     `json:"includePvs"`
	TTL                int      `json:"ttl"` // 小时
	StorageLocation    string   `json:"storageLocation,omitempty"`
}

// BackupResponse 备份响应
//
//nolint:revive // the exported DTO name keeps the backup domain prefix for API clarity.
type BackupResponse struct {
	ID                   uint64     `json:"id"`
	Name                 string     `json:"name"`
	ClusterID            uint64     `json:"clusterId"`
	BackupType           string     `json:"backupType"`
	IncludedNamespaces   []string   `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces   []string   `json:"excludedNamespaces,omitempty"`
	IncludedResources    []string   `json:"includedResources,omitempty"`
	ExcludedResources    []string   `json:"excludedResources,omitempty"`
	LabelSelector        string     `json:"labelSelector,omitempty"`
	IncludePVs           bool       `json:"includePvs"`
	TTL                  int        `json:"ttl"`
	Status               string     `json:"status"`
	Phase                string     `json:"phase,omitempty"`
	StartTime            *time.Time `json:"startTime,omitempty"`
	CompletionTime       *time.Time `json:"completionTime,omitempty"`
	Expiration           *time.Time `json:"expiration,omitempty"`
	Errors               int        `json:"errors"`
	Warnings             int        `json:"warnings"`
	BackupSize           int64      `json:"backupSize,omitempty"`
	StorageLocation      string     `json:"storageLocation,omitempty"`
	VolumeSnapshotsCount int        `json:"volumeSnapshotsCount"`
	CreatedAt            time.Time  `json:"createdAt"`
	CreatedBy            string     `json:"createdBy"`
}

// BackupListQuery 备份列表查询
//
//nolint:revive // the exported DTO name keeps the backup domain prefix for API clarity.
type BackupListQuery struct {
	ClusterID  uint64 `form:"clusterId" binding:"required"`
	Status     string `form:"status"`
	BackupType string `form:"backupType"`
	Name       string `form:"name"`
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=20"`
}

// BackupListResponse 备份列表响应
//
//nolint:revive // the exported DTO name keeps the backup domain prefix for API clarity.
type BackupListResponse struct {
	Items    []BackupResponse `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}

// --- 定时备份 ---

// CreateScheduleRequest 创建定时备份请求
type CreateScheduleRequest struct {
	Name           string              `json:"name" binding:"required,max=128"`
	ClusterID      uint64              `json:"clusterId" binding:"required"`
	CronExpression string              `json:"cronExpression" binding:"required"`
	Enabled        bool                `json:"enabled"`
	BackupTemplate CreateBackupRequest `json:"backupTemplate" binding:"required"`
}

// ScheduleResponse 定时备份响应
type ScheduleResponse struct {
	ID               uint64         `json:"id"`
	Name             string         `json:"name"`
	ClusterID        uint64         `json:"clusterId"`
	CronExpression   string         `json:"cronExpression"`
	Enabled          bool           `json:"enabled"`
	BackupTemplate   map[string]any `json:"backupTemplate"`
	LastBackupTime   *time.Time     `json:"lastBackupTime,omitempty"`
	NextBackupTime   *time.Time     `json:"nextBackupTime,omitempty"`
	LastBackupStatus string         `json:"lastBackupStatus,omitempty"`
	Phase            string         `json:"phase,omitempty"`
	CreatedAt        time.Time      `json:"createdAt"`
	CreatedBy        string         `json:"createdBy"`
}

// ScheduleListQuery 定时备份列表查询
type ScheduleListQuery struct {
	ClusterID uint64 `form:"clusterId" binding:"required"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"pageSize,default=20"`
}

// ScheduleListResponse 定时备份列表响应
type ScheduleListResponse struct {
	Items    []ScheduleResponse `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// --- 恢复 ---

// CreateRestoreRequest 创建恢复请求
type CreateRestoreRequest struct {
	Name               string            `json:"name" binding:"required,max=128"`
	ClusterID          uint64            `json:"clusterId" binding:"required"`
	BackupName         string            `json:"backupName" binding:"required"`
	NamespaceMapping   map[string]string `json:"namespaceMapping,omitempty"`
	IncludedNamespaces []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string          `json:"includedResources,omitempty"`
	ExcludedResources  []string          `json:"excludedResources,omitempty"`
	RestorePVs         bool              `json:"restorePvs"`
}

// RestoreResponse 恢复响应
type RestoreResponse struct {
	ID                 uint64            `json:"id"`
	Name               string            `json:"name"`
	ClusterID          uint64            `json:"clusterId"`
	BackupName         string            `json:"backupName"`
	NamespaceMapping   map[string]string `json:"namespaceMapping,omitempty"`
	IncludedNamespaces []string          `json:"includedNamespaces,omitempty"`
	ExcludedNamespaces []string          `json:"excludedNamespaces,omitempty"`
	IncludedResources  []string          `json:"includedResources,omitempty"`
	ExcludedResources  []string          `json:"excludedResources,omitempty"`
	RestorePVs         bool              `json:"restorePvs"`
	Status             string            `json:"status"`
	Phase              string            `json:"phase,omitempty"`
	StartTime          *time.Time        `json:"startTime,omitempty"`
	CompletionTime     *time.Time        `json:"completionTime,omitempty"`
	Errors             int               `json:"errors"`
	Warnings           int               `json:"warnings"`
	CreatedAt          time.Time         `json:"createdAt"`
	CreatedBy          string            `json:"createdBy"`
}

// RestoreListQuery 恢复列表查询
type RestoreListQuery struct {
	ClusterID uint64 `form:"clusterId" binding:"required"`
	Status    string `form:"status"`
	Page      int    `form:"page,default=1"`
	PageSize  int    `form:"pageSize,default=20"`
}

// RestoreListResponse 恢复列表响应
type RestoreListResponse struct {
	Items    []RestoreResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// VeleroInstallRequest 安装 Velero 请求
type VeleroInstallRequest struct {
	ClusterID       uint64 `json:"clusterId" binding:"required"`
	StorageLocation string `json:"storageLocation" binding:"required"`
	Image           string `json:"image,omitempty"` // 默认 velero/velero:v1.12.0
}

// VeleroStatusResponse Velero 状态响应
type VeleroStatusResponse struct {
	Installed        bool   `json:"installed"`
	Version          string `json:"version,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
	RunningPods      int    `json:"runningPods"`
	StorageLocations int    `json:"storageLocations"`
}
