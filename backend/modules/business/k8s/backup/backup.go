package backup

import (
	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/namespace"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitBackupModule initializes the Velero Backup module.
func InitBackupModule(r *gin.RouterGroup, db *gorm.DB, clusterService *cluster.ClusterService, namespaceService *namespace.NamespaceService) *BackupService {
	service := NewBackupService(db, clusterService, namespaceService)
	handler := NewBackupHandler(service)

	// Migrate database
	if err := service.Migrate(); err != nil {
		panic(err)
	}

	// Register routes
	handler.RegisterRoutes(r)

	return service
}
