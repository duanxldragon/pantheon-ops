package filedist

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// storageTypeLocal marks a filesystem-backed storage location.
const storageTypeLocal = "local"

// FileStorage represents a file storage location.
type FileStorage struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:128;not null;uniqueIndex" json:"name"`
	Type      string         `gorm:"size:32;not null" json:"type"` // local, s3, oss, minio
	BasePath  string         `gorm:"size:512;not null" json:"basePath"`
	Config    datatypes.JSON `json:"config"`                             // Storage-specific configuration
	MaxSize   int64          `gorm:"default:10737418240" json:"maxSize"` // 10GB default
	UsedSize  int64          `gorm:"default:0" json:"usedSize"`
	FileCount int            `gorm:"default:0" json:"fileCount"`
	IsDefault bool           `gorm:"default:false" json:"isDefault"`
	Status    string         `gorm:"size:32;not null;default:'active'" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	CreatedBy string         `gorm:"size:64" json:"createdBy"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the filedist_storage table name.
func (FileStorage) TableName() string {
	return "filedist_storage"
}

// FileRecord represents an uploaded file.
type FileRecord struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	StorageID   uint64         `gorm:"not null;index" json:"storageId"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Path        string         `gorm:"size:512;not null" json:"path"` // Relative path in storage
	Size        int64          `gorm:"not null" json:"size"`
	MD5         string         `gorm:"size:32;index" json:"md5"`
	SHA256      string         `gorm:"size:64" json:"sha256,omitempty"`
	ContentType string         `gorm:"size:128" json:"contentType"`
	Tags        datatypes.JSON `json:"tags"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	UploadedBy  string         `gorm:"size:64" json:"uploadedBy"`
	UploadedAt  time.Time      `gorm:"not null;index" json:"uploadedAt"`
	CreatedAt   time.Time      `json:"createdAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the filedist_file table name.
func (FileRecord) TableName() string {
	return "filedist_file"
}

// StorageDTO for API requests
type StorageDTO struct {
	Name      string                 `json:"name" binding:"required"`
	Type      string                 `json:"type" binding:"required,oneof=local s3 oss minio"`
	BasePath  string                 `json:"basePath" binding:"required"`
	Config    map[string]interface{} `json:"config"`
	MaxSize   int64                  `json:"maxSize"`
	IsDefault bool                   `json:"isDefault"`
}

// S3Config carries the credentials and endpoint of an S3-compatible storage location.
type S3Config struct {
	Region          string `json:"region"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Endpoint        string `json:"endpoint"`
	Bucket          string `json:"bucket"`
	UseSSL          bool   `json:"useSSL"`
}

// OSSConfig carries the endpoint and credentials of an Aliyun OSS storage location.
type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	Bucket          string `json:"bucket"`
}

// MinioConfig carries the credentials and endpoint of a MinIO storage location.
type MinioConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Bucket          string `json:"bucket"`
	UseSSL          bool   `json:"useSSL"`
}

// StorageResponse is the file storage shape returned to clients; secrets stay redacted.
type StorageResponse struct {
	ID        uint64                 `json:"id"`
	Name      string                 `json:"name"`
	Type      string                 `json:"type"`
	BasePath  string                 `json:"basePath"`
	Config    map[string]interface{} `json:"config,omitempty"`
	MaxSize   int64                  `json:"maxSize"`
	UsedSize  int64                  `json:"usedSize"`
	FileCount int                    `json:"fileCount"`
	IsDefault bool                   `json:"isDefault"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"createdAt"`
	CreatedBy string                 `json:"createdBy"`
}

// StorageListResponse is the list of configured file storage locations.
type StorageListResponse struct {
	Items []StorageResponse `json:"items"`
	Total int64             `json:"total"`
}

// FileRecordDTO for API requests
type FileRecordDTO struct {
	StorageID   uint64   `json:"storageId" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Size        int64    `json:"size" binding:"required"`
	MD5         string   `json:"md5" binding:"required"`
	ContentType string   `json:"contentType"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

// FileRecordResponse is the stored-file shape returned to clients.
type FileRecordResponse struct {
	ID          uint64    `json:"id"`
	StorageID   uint64    `json:"storageId"`
	StorageName string    `json:"storageName,omitempty"`
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	MD5         string    `json:"md5"`
	ContentType string    `json:"contentType"`
	Tags        []string  `json:"tags"`
	Description string    `json:"description,omitempty"`
	UploadedBy  string    `json:"uploadedBy"`
	UploadedAt  time.Time `json:"uploadedAt"`
	// Compatibility aliases for the business overlay UI.
	FileName    string `json:"fileName,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	FilePath    string `json:"filePath,omitempty"`
	FileSize    int64  `json:"fileSize,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	StorageType string `json:"storageType,omitempty"`
	Status      string `json:"status,omitempty"`
	CreatedBy   string `json:"createdBy,omitempty"`
}

// FileRecordListQuery filters and paginates stored files.
type FileRecordListQuery struct {
	StorageID uint64 `form:"storageId"`
	Keyword   string `form:"keyword"`
	Tag       string `form:"tag"`
	Page      int    `form:"page"`
	PageSize  int    `form:"pageSize"`
}

// FileRecordListResponse is the paged list of stored files.
type FileRecordListResponse struct {
	Items    []FileRecordResponse `json:"items"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

// FileUploadResponse for upload completion
type FileUploadResponse struct {
	FileID      uint64 `json:"fileId"`
	DownloadURL string `json:"downloadUrl"`
}
