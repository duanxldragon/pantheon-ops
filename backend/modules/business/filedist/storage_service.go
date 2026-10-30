package filedist

import (
	"crypto/md5" // #nosec G501 -- MD5 is a non-security integrity fingerprint here; SHA-256 is computed alongside it.
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var storageSecretFields = []string{"secretAccessKey", "accessKeySecret"}

// StorageService manages file storage locations.
type StorageService struct {
	db              *gorm.DB
	defaultBasePath string
}

// NewStorageService creates a new StorageService.
func NewStorageService(db *gorm.DB) *StorageService {
	return &StorageService{db: db}
}

// NewStorageServiceWithBasePath creates a storage service with a configured
// local root used when the caller omits a base path.
func NewStorageServiceWithBasePath(db *gorm.DB, basePath string) *StorageService {
	return &StorageService{db: db, defaultBasePath: basePath}
}

// Migrate creates or updates database schema.
func (s *StorageService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&FileStorage{}, &FileRecord{}); err != nil {
		return err
	}
	return s.encryptStoredConfigs()
}

// normalizeLocalStoragePath rejects parent-directory segments in an
// admin-supplied local storage root before filepath.Clean would silently
// resolve them, so a record path cannot escape the configured root.
func normalizeLocalStoragePath(basePath string) (string, error) {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" {
		return "", errors.New("storage.base_path_required")
	}
	for _, segment := range strings.FieldsFunc(trimmed, func(r rune) bool { return r == '/' || r == os.PathSeparator }) {
		if segment == ".." {
			return "", errors.New("storage.base_path_invalid")
		}
	}
	return filepath.Clean(trimmed), nil
}

// CreateStorage adds a new storage location.
func (s *StorageService) CreateStorage(dto StorageDTO, createdBy string) (*StorageResponse, error) {
	if dto.Type == storageTypeLocal {
		if strings.TrimSpace(dto.BasePath) == "" {
			dto.BasePath = s.defaultBasePath
		}
		basePath, err := normalizeLocalStoragePath(dto.BasePath)
		if err != nil {
			return nil, err
		}
		dto.BasePath = basePath
	}
	// If marking as default, clear other defaults
	if dto.IsDefault {
		_ = s.db.Model(&FileStorage{}).Where("is_default = ?", true).Update("is_default", false).Error
	}

	protectedConfig, err := secrets.EncryptConfigFields(dto.Config, storageSecretFields...)
	if err != nil {
		return nil, err
	}
	configJSON, err := json.Marshal(protectedConfig)
	if err != nil {
		return nil, err
	}

	record := &FileStorage{
		Name:      dto.Name,
		Type:      dto.Type,
		BasePath:  dto.BasePath,
		Config:    datatypes.JSON(configJSON),
		MaxSize:   dto.MaxSize,
		IsDefault: dto.IsDefault,
		Status:    "active",
		CreatedBy: createdBy,
	}

	if record.MaxSize == 0 {
		record.MaxSize = 10737418240 // 10GB default
	}

	// Create base directory for local storage
	if dto.Type == storageTypeLocal {
		// codeql go/path-injection: this is a false positive: the base path is
		// the configuration of a storage location, reachable only through the
		// permission-gated storage CRUD API, and normalizeLocalStoragePath
		// rejects traversal segments.
		if err := os.MkdirAll(dto.BasePath, 0750); err != nil {
			return nil, fmt.Errorf("create_directory_failed: %w", err)
		}
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	return s.toStorageResponse(record), nil
}

// EnsureDefaultStorage creates a local default storage for a fresh database.
// Existing deployments are left untouched.
func (s *StorageService) EnsureDefaultStorage(createdBy string) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	var count int64
	if err := s.db.Model(&FileStorage{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	basePath := s.defaultBasePath
	if strings.TrimSpace(basePath) == "" {
		basePath = "./uploads"
	}
	_, err := s.CreateStorage(StorageDTO{
		Name:      "local-default",
		Type:      storageTypeLocal,
		BasePath:  basePath,
		IsDefault: true,
	}, createdBy)
	return err
}

// ListStorages returns all storage locations.
func (s *StorageService) ListStorages() (*StorageListResponse, error) {
	var records []FileStorage
	if err := s.db.Order("is_default DESC, created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]StorageResponse, len(records))
	for i, r := range records {
		items[i] = *s.toStorageResponse(&r)
	}

	return &StorageListResponse{
		Items: items,
		Total: int64(len(items)),
	}, nil
}

// GetStorageByID retrieves a storage by ID.
func (s *StorageService) GetStorageByID(id uint64) (*StorageResponse, error) {
	var record FileStorage
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toStorageResponse(&record), nil
}

// UpdateStorage modifies an existing storage.
func (s *StorageService) UpdateStorage(id uint64, dto StorageDTO) error {
	var record FileStorage
	if err := s.db.First(&record, id).Error; err != nil {
		return err
	}
	var existingConfig map[string]interface{}
	if err := json.Unmarshal(record.Config, &existingConfig); err != nil {
		return err
	}
	config := secrets.PreserveStoredFields(dto.Config, existingConfig, storageSecretFields...)
	protectedConfig, err := secrets.EncryptConfigFields(config, storageSecretFields...)
	if err != nil {
		return err
	}
	configJSON, err := json.Marshal(protectedConfig)
	if err != nil {
		return err
	}

	if dto.Type == storageTypeLocal {
		basePath, err := normalizeLocalStoragePath(dto.BasePath)
		if err != nil {
			return err
		}
		dto.BasePath = basePath
	}

	updates := map[string]interface{}{
		"name":      dto.Name,
		"type":      dto.Type,
		"base_path": dto.BasePath,
		"config":    configJSON,
		"max_size":  dto.MaxSize,
	}

	if dto.IsDefault {
		_ = s.db.Model(&FileStorage{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false).Error
		updates["is_default"] = true
	}

	return s.db.Model(&FileStorage{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteStorage removes a storage location.
func (s *StorageService) DeleteStorage(id uint64) error {
	// Check if there are files in this storage
	var count int64
	if err := s.db.Model(&FileRecord{}).Where("storage_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("storage_has_files: %d files exist", count)
	}

	return s.db.Delete(&FileStorage{}, id).Error
}

// resolveWithinRoot joins rel to a storage root and proves the result stays inside that root.
// Every filesystem operation in this module goes through it, so a stored record path or a
// caller-supplied name cannot reach a file outside the configured storage location.
func resolveWithinRoot(root, rel string) (string, error) {
	base, err := normalizeLocalStoragePath(root)
	if err != nil {
		return "", err
	}
	target := filepath.Clean(filepath.Join(base, filepath.FromSlash(rel)))
	if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
		return "", errors.New("storage.path_escapes_base_path")
	}
	return target, nil
}

// sanitizeUploadFileName reduces an uploaded name to a single safe path element. Multipart
// file names are attacker-controlled, so directory parts, traversal segments, empty results and
// control characters are rejected instead of being joined into a stored path.
func sanitizeUploadFileName(name string) (string, error) {
	cleaned := strings.TrimSpace(name)
	if cleaned == "" {
		return "", errors.New("file.name_required")
	}
	// Multipart file names are attacker-controlled, and browser-supplied Windows paths arrive with
	// backslashes even when this process runs on Linux. Treat both separators as directory parts
	// before taking the final element, otherwise a stored name could keep path structure on Linux.
	cleaned = filepath.Base(filepath.FromSlash(strings.ReplaceAll(cleaned, `\`, "/")))
	if cleaned == "." || cleaned == ".." || cleaned == string(filepath.Separator) || strings.Contains(cleaned, "..") {
		return "", errors.New("file.name_invalid")
	}
	if strings.ContainsAny(cleaned, "\r\n\x00") || filepath.Base(cleaned) != cleaned {
		return "", errors.New("file.name_invalid")
	}
	return cleaned, nil
}

// UploadFile stores a file and creates a record.
func (s *StorageService) UploadFile(storageID uint64, fileName string, reader io.Reader, size int64, uploadedBy string) (*FileRecordResponse, error) {
	// Get storage
	var storage FileStorage
	if err := s.db.First(&storage, storageID).Error; err != nil {
		return nil, fmt.Errorf("storage.not_found: %w", err)
	}

	// Check storage capacity
	if storage.UsedSize+size > storage.MaxSize {
		return nil, fmt.Errorf("storage_full: used %d, max %d", storage.UsedSize, storage.MaxSize)
	}

	// Generate unique path. The caller-supplied name is reduced to a single safe path
	// element and the joined result is proven to stay inside the storage root.
	safeName, err := sanitizeUploadFileName(fileName)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	relativePath := filepath.Join(
		now.Format("2006/01/02"),
		fmt.Sprintf("%d-%s", now.Unix(), safeName),
	)
	fullPath, err := resolveWithinRoot(storage.BasePath, relativePath)
	if err != nil {
		return nil, err
	}

	// Create directory
	if err := os.MkdirAll(filepath.Dir(fullPath), 0750); err != nil {
		return nil, fmt.Errorf("create_directory_failed: %w", err)
	}

	// Write file and calculate MD5 + SHA-256
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("create_file_failed: %w", err)
	}

	md5Hash := md5.New() // #nosec G401 -- MD5 is kept as a non-security integrity fingerprint; SHA-256 accompanies it.
	sha256Hash := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, md5Hash, sha256Hash), reader)
	if err != nil {
		_ = file.Close()
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("write_file_failed: %w", err)
	}
	// Closing flushes the stored bytes; if it fails the record must not claim a
	// file that never reached disk.
	if err := file.Close(); err != nil {
		_ = os.Remove(fullPath)
		return nil, fmt.Errorf("close_file_failed: %w", err)
	}

	md5sum := hex.EncodeToString(md5Hash.Sum(nil))
	sha256sum := hex.EncodeToString(sha256Hash.Sum(nil))

	// Create file record
	record := &FileRecord{
		StorageID:  storageID,
		Name:       safeName,
		Path:       relativePath,
		Size:       written,
		MD5:        md5sum,
		SHA256:     sha256sum,
		UploadedBy: uploadedBy,
		UploadedAt: time.Now(),
	}

	if err := s.db.Create(record).Error; err != nil {
		_ = os.Remove(fullPath)
		return nil, err
	}

	// Update storage statistics
	_ = s.db.Model(&storage).Updates(map[string]interface{}{
		"used_size":  gorm.Expr("used_size + ?", written),
		"file_count": gorm.Expr("file_count + 1"),
	}).Error

	return s.toFileRecordResponse(record, &storage), nil
}

// ListFiles returns files with filtering.
func (s *StorageService) ListFiles(query FileRecordListQuery) (*FileRecordListResponse, error) {
	db := s.db.Model(&FileRecord{})

	if query.StorageID > 0 {
		db = db.Where("storage_id = ?", query.StorageID)
	}
	if query.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+query.Keyword+"%")
	}
	if query.Tag != "" {
		db = db.Where("JSON_CONTAINS(tags, ?)", fmt.Sprintf("\"%s\"", query.Tag))
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	var records []FileRecord
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("uploaded_at DESC").Offset(offset).Limit(query.PageSize).Find(&records).Error; err != nil {
		return nil, err
	}

	// Load storage names
	storageMap := make(map[uint64]*FileStorage)
	for _, r := range records {
		if _, exists := storageMap[r.StorageID]; !exists {
			var storage FileStorage
			if err := s.db.First(&storage, r.StorageID).Error; err == nil {
				storageMap[r.StorageID] = &storage
			}
		}
	}

	items := make([]FileRecordResponse, len(records))
	for i, r := range records {
		items[i] = *s.toFileRecordResponse(&r, storageMap[r.StorageID])
	}

	return &FileRecordListResponse{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetFileByID retrieves a file record by ID.
func (s *StorageService) GetFileByID(id uint64) (*FileRecordResponse, error) {
	var record FileRecord
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}

	var storage FileStorage
	_ = s.db.First(&storage, record.StorageID).Error

	return s.toFileRecordResponse(&record, &storage), nil
}

// DeleteFile removes a file record and physical file.
func (s *StorageService) DeleteFile(id uint64) error {
	var record FileRecord
	if err := s.db.First(&record, id).Error; err != nil {
		return err
	}

	var storage FileStorage
	if err := s.db.First(&storage, record.StorageID).Error; err != nil {
		return err
	}

	// Delete physical file. A stored path that no longer resolves inside its storage root is
	// refused instead of being removed, so a tampered or migrated record cannot delete files
	// outside the storage location.
	fullPath, err := resolveWithinRoot(storage.BasePath, record.Path)
	if err != nil {
		return err
	}
	_ = os.Remove(fullPath)

	// Delete record
	if err := s.db.Delete(&record).Error; err != nil {
		return err
	}

	// Update storage statistics
	_ = s.db.Model(&storage).Updates(map[string]interface{}{
		"used_size":  gorm.Expr("used_size - ?", record.Size),
		"file_count": gorm.Expr("file_count - 1"),
	}).Error

	return nil
}

// GetFilePath returns the full path to a file.
func (s *StorageService) GetFilePath(id uint64) (string, error) {
	var record FileRecord
	if err := s.db.First(&record, id).Error; err != nil {
		return "", err
	}

	var storage FileStorage
	if err := s.db.First(&storage, record.StorageID).Error; err != nil {
		return "", err
	}

	return resolveWithinRoot(storage.BasePath, record.Path)
}

func (s *StorageService) toStorageResponse(r *FileStorage) *StorageResponse {
	var config map[string]interface{}
	_ = json.Unmarshal(r.Config, &config)
	config = secrets.RedactConfigFields(config, storageSecretFields...)

	return &StorageResponse{
		ID:        r.ID,
		Name:      r.Name,
		Type:      r.Type,
		BasePath:  r.BasePath,
		Config:    config,
		MaxSize:   r.MaxSize,
		UsedSize:  r.UsedSize,
		FileCount: r.FileCount,
		IsDefault: r.IsDefault,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		CreatedBy: r.CreatedBy,
	}
}

func (s *StorageService) encryptStoredConfigs() error {
	var records []FileStorage
	if err := s.db.Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		var config map[string]interface{}
		if err := json.Unmarshal(records[i].Config, &config); err != nil {
			return err
		}
		protected, err := secrets.EncryptConfigFields(config, storageSecretFields...)
		if err != nil {
			return err
		}
		encoded, err := json.Marshal(protected)
		if err != nil {
			return err
		}
		if err := s.db.Model(&FileStorage{}).Where("id = ?", records[i].ID).Update("config", encoded).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *StorageService) toFileRecordResponse(r *FileRecord, storage *FileStorage) *FileRecordResponse {
	var tags []string
	_ = json.Unmarshal(r.Tags, &tags)

	resp := &FileRecordResponse{
		ID:          r.ID,
		StorageID:   r.StorageID,
		Name:        r.Name,
		Path:        r.Path,
		Size:        r.Size,
		MD5:         r.MD5,
		SHA256:      r.SHA256,
		ContentType: r.ContentType,
		Tags:        tags,
		Description: r.Description,
		UploadedBy:  r.UploadedBy,
		UploadedAt:  r.UploadedAt,
		FileName:    r.Name,
		DisplayName: r.Name,
		FilePath:    r.Path,
		FileSize:    r.Size,
		Status:      "active",
		CreatedBy:   r.UploadedBy,
	}

	if storage != nil {
		resp.StorageName = storage.Name
		resp.StorageType = storage.Type
	}

	return resp
}
