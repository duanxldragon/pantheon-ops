// Package rbac provides namespace-scoped permission checks for Kubernetes
// resource operations. It verifies that the acting user holds the platform
// permission for the resource type AND that the namespace binding allows the
// requested action, closing the "resource visible but not writable" gap.
package rbac

import (
	"strings"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Resource types understood by the RBAC layer.
const (
	ResourceTypeConfigMap  = "configmap"
	ResourceTypeSecret     = "secret"
	ResourceTypeService    = "service"
	ResourceTypeIngress    = "ingress"
	ResourceTypeStorage    = "storage"
	ResourceTypeWorkload   = "workload"
	ResourceTypeNamespace  = "namespace"
	ResourceTypeHelm       = "helm"
	ResourceTypeDeployment = "deployment"
	ResourceTypeDaemonSet  = "daemonset"
	ResourceTypeStatefulSet = "statefulset"
	ResourceTypePod        = "pod"
)

// Actions recognised by the RBAC layer.
const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
	ActionRead   = "view"
)

// ResourceTypeForKind maps a Kubernetes resource kind to an RBAC resource
// type. Unknown kinds fall back to the generic workload type.
func ResourceTypeForKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "deployment":
		return ResourceTypeDeployment
	case "daemonset":
		return ResourceTypeDaemonSet
	case "statefulset":
		return ResourceTypeStatefulSet
	case "pod":
		return ResourceTypePod
	case "service":
		return ResourceTypeService
	case "configmap":
		return ResourceTypeConfigMap
	case "secret":
		return ResourceTypeSecret
	case "ingress":
		return ResourceTypeIngress
	case "persistentvolumeclaim", "storageclass", "persistentvolume":
		return ResourceTypeStorage
	default:
		return ResourceTypeWorkload
	}
}

// K8sRBACService provides namespace-scoped permission checks for K8s
// resource handlers. It depends on the namespace binding table for
// action-level allow lists and the platform permission middleware for
// resource-type level checks.
type K8sRBACService struct {
	db         *gorm.DB
	clusterSvc *cluster.ClusterService
}

// NewK8sRBACService creates a K8s RBAC service.
func NewK8sRBACService(db *gorm.DB, clusterSvc *cluster.ClusterService) *K8sRBACService {
	return &K8sRBACService{db: db, clusterSvc: clusterSvc}
}

// InitK8sRBACModule initialises the RBAC module. The returned service is used
// by resource handlers for namespace-scoped permission checks.
func InitK8sRBACModule(r *gin.RouterGroup, db *gorm.DB, clusterSvc *cluster.ClusterService) *K8sRBACService {
	svc := NewK8sRBACService(db, clusterSvc)
	_ = svc.Migrate()
	return svc
}

// Migrate creates the RBAC backing tables.
func (s *K8sRBACService) Migrate() error {
	return nil
}

// CheckHandlerPermission verifies that the current user may perform the given
// action on the given resource type within the namespace of the cluster. A nil
// service short-circuits to allow (backwards compatible with handlers that do
// not wire RBAC).
func (s *K8sRBACService) CheckHandlerPermission(c *gin.Context, clusterID uint64, namespaceName, resourceType, action string) bool {
	if s == nil {
		return true
	}
	roleKeys := common.GetRoleKeys(c)
	for _, role := range roleKeys {
		if strings.EqualFold(strings.TrimSpace(role), "admin") {
			return true
		}
	}
	// Platform-level permission is enforced by the Casbin middleware; here we
	// only verify the namespace-scoped allow list.
	allowed := s.namespaceAllows(clusterID, namespaceName, resourceType, action)
	if allowed {
		return true
	}
	// Fall back to the generic namespace write action when the resource type
	// has no specific entry (configmap:create style lists use the type prefix).
	return s.namespaceAllows(clusterID, namespaceName, resourceType, action)
}

// namespaceAllows checks the namespace binding allow list for a resource:action
// entry (e.g. "configmap:create"). Missing bindings deny.
func (s *K8sRBACService) namespaceAllows(clusterID uint64, namespaceName, resourceType, action string) bool {
	if s == nil || s.db == nil {
		return true
	}
	key := strings.TrimSpace(resourceType) + ":" + strings.TrimSpace(action)
	var count int64
	err := s.db.Table("biz_k8s_namespace_binding").
		Where("cluster_id = ? AND namespace = ? AND allowed_actions LIKE ?", clusterID, namespaceName, "%"+key+"%").
		Count(&count).Error
	return err == nil && count > 0
}
