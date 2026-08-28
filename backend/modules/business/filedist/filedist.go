// Package filedist provides the file distribution module: uploading and
// distributing binary/archive artifacts for the offline deployment scenario.
// Files are stored under a configurable uploads directory and referenced by
// stable UUID keys so inner-network hosts can fetch them without external
// registries.
package filedist

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FileRecord is a distributed file record in the database.
type FileRecord struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	FileKey       string    `gorm:"size:64;not null;uniqueIndex:uk_filedist_key" json:"fileKey"`
	FileName      string    `gorm:"size:255;not null" json:"fileName"`
	SizeBytes     int64     `gorm:"not null" json:"sizeBytes"`
	SHA256        string    `gorm:"size:64;not null" json:"sha256"`
	ContentType   string    `gorm:"size:128" json:"contentType"`
	StoragePath   string    `gorm:"size:512;not null" json:"-"`
	DownloadCount int64     `gorm:"not null;default:0" json:"downloadCount"`
	Status        string    `gorm:"size:32;default:active" json:"status"`
	Remark        string    `gorm:"size:512" json:"remark"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	CreatedBy     string    `gorm:"size:64" json:"createdBy"`
}

// TableName returns the file record table name.
func (FileRecord) TableName() string { return "biz_filedist_record" }

// FileDistModule is the filedist module facade.
type FileDistModule struct {
	db        *gorm.DB
	uploadDir string
}

// InitFileDistModule creates the file distribution module rooted at uploadDir.
func InitFileDistModule(r *gin.RouterGroup, db *gorm.DB, uploadDir string) *FileDistModule {
	m := &FileDistModule{db: db, uploadDir: uploadDir}
	if strings.TrimSpace(m.uploadDir) == "" {
		m.uploadDir = "./uploads"
	}
	_ = os.MkdirAll(m.uploadDir, 0o755)
	return m
}

// Migrate creates the file record table.
func (m *FileDistModule) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&FileRecord{})
}

// RegisterRoutes registers filedist endpoints.
func (m *FileDistModule) RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/business/filedist")
	group.GET("/files", m.ListFiles)
	group.POST("/files", m.UploadFile)
	group.GET("/files/:key/download", m.DownloadFile)
	group.DELETE("/files/:key", m.DeleteFile)
}

// ListFiles returns uploaded file records.
func (m *FileDistModule) ListFiles(c *gin.Context) {
	var records []FileRecord
	if err := m.db.Order("id DESC").Limit(200).Find(&records).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "filedist.list_failed")
		return
	}
	common.Success(c, gin.H{"items": records, "total": len(records)})
}

// UploadFile stores an uploaded artifact and records its metadata.
func (m *FileDistModule) UploadFile(c *gin.Context) {
	common.SetAuditMetadata(c, "filedist.file.upload", common.BusinessInsert)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "filedist.file_required")
		return
	}
	defer func() { _ = file.Close() }()

	content, err := io.ReadAll(io.LimitReader(file, 512<<20)) // 512MB cap
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "filedist.read_failed")
		return
	}
	if len(content) == 0 {
		common.Fail(c, common.CodeParamInvalid, "filedist.empty_file")
		return
	}

	sum := sha256.Sum256(content)
	fileKey := uuid.NewString()
	storageName := fileKey + filepath.Ext(strings.TrimSpace(header.Filename))
	storagePath := filepath.Join(m.uploadDir, storageName)
	if err := os.WriteFile(storagePath, content, 0o644); err != nil {
		common.FailWithError(c, common.CodeError, err, "filedist.store_failed")
		return
	}

	record := FileRecord{
		FileKey:     fileKey,
		FileName:    header.Filename,
		SizeBytes:   int64(len(content)),
		SHA256:      hex.EncodeToString(sum[:]),
		ContentType: header.Header.Get("Content-Type"),
		StoragePath: storagePath,
		Status:      "active",
		CreatedBy:   strings.TrimSpace(c.GetHeader("X-User-ID")),
	}
	if record.CreatedBy == "" {
		record.CreatedBy = "0"
	}
	if err := m.db.Create(&record).Error; err != nil {
		_ = os.Remove(storagePath)
		common.FailWithError(c, common.CodeError, err, "filedist.record_failed")
		return
	}
	common.Success(c, record)
}

// DownloadFile streams a stored file and increments its download counter.
func (m *FileDistModule) DownloadFile(c *gin.Context) {
	key := strings.TrimSpace(c.Param("key"))
	var record FileRecord
	if err := m.db.Where("file_key = ? AND status = ?", key, "active").First(&record).Error; err != nil {
		common.FailWithError(c, common.CodeError, errors.New("filedist.file_not_found"), "filedist.file_not_found")
		return
	}
	if _, err := os.Stat(record.StoragePath); err != nil {
		common.FailWithError(c, common.CodeError, errors.New("filedist.file_missing"), "filedist.file_missing")
		return
	}
	_ = m.db.Model(&FileRecord{}).Where("id = ?", record.ID).
		UpdateColumn("download_count", gorm.Expr("download_count + 1"))
	c.Header("Content-Disposition", "attachment; filename=\""+record.FileName+"\"")
	c.File(record.StoragePath)
}

// DeleteFile removes a distributed file and its metadata.
func (m *FileDistModule) DeleteFile(c *gin.Context) {
	common.SetAuditMetadata(c, "filedist.file.delete", common.BusinessDelete)
	key := strings.TrimSpace(c.Param("key"))
	var record FileRecord
	if err := m.db.Where("file_key = ?", key).First(&record).Error; err != nil {
		common.FailWithError(c, common.CodeError, errors.New("filedist.file_not_found"), "filedist.file_not_found")
		return
	}
	if err := m.db.Delete(&record).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "filedist.delete_failed")
		return
	}
	_ = os.Remove(record.StoragePath)
	common.Success(c, gin.H{"deleted": true})
}
