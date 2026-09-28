package filedist

import (
	"context"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FileDistModule wires the storage and file distribution services into the
// business router. The module owns the transfer workers used by distribution
// tasks and keeps the public registration surface small for the overlay.
type FileDistModule struct {
	db                  *gorm.DB
	storageService      *StorageService
	distributionService *DistributionService
	transferEngine      *TransferEngine
	handler             *FileDistHandler
}

// InitFileDistModule creates the file distribution module rooted at uploadDir.
// The current storage service persists files under the configured local path;
// uploadDir is retained as the default base path for the first local storage.
func InitFileDistModule(r *gin.RouterGroup, db *gorm.DB, uploadDir string) *FileDistModule {
	_ = r
	if strings.TrimSpace(uploadDir) == "" {
		uploadDir = "./uploads"
	}

	storageService := NewStorageServiceWithBasePath(db, uploadDir)
	transferEngine := NewTransferEngine(db, storageService, 5)
	distributionService := NewDistributionService(db, storageService, transferEngine)

	module := &FileDistModule{
		db:                  db,
		storageService:      storageService,
		distributionService: distributionService,
		transferEngine:      transferEngine,
	}
	module.handler = NewFileDistHandler(storageService, distributionService)

	// Start workers once during server assembly so tasks created through the
	// HTTP API are processed without requiring a second lifecycle hook.
	transferEngine.Start(context.Background())

	return module
}

// Migrate creates the storage, uploaded-file, and distribution-task tables.
func (m *FileDistModule) Migrate(_ *gorm.DB) error {
	if err := m.storageService.Migrate(); err != nil {
		return err
	}
	if err := m.distributionService.Migrate(); err != nil {
		return err
	}
	if err := m.storageService.EnsureDefaultStorage("system"); err != nil {
		return err
	}
	return seedFileDistMenusAndI18n(m.db)
}

// RegisterRoutes registers all file distribution endpoints.
func (m *FileDistModule) RegisterRoutes(r *gin.RouterGroup) {
	m.handler.RegisterRoutes(contracts.DataScopedGroup(r, "/filedist", m.db))
	m.handler.RegisterRoutes(contracts.DataScopedGroup(r, "/business/filedist", m.db))
}

// Stop terminates transfer workers. It is safe to call more than once.
func (m *FileDistModule) Stop() {
	if m != nil && m.transferEngine != nil {
		m.transferEngine.Stop()
	}
}
