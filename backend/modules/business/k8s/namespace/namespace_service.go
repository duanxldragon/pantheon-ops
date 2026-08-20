package namespace

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"

	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const syncTimeout = 30 * time.Second

// NamespaceService manages Kubernetes namespaces through client-go.
//
//nolint:revive // Service names retain the namespace domain prefix for module wiring clarity.
type NamespaceService struct {
	clusterSvc *cluster.ClusterService
	db         *gorm.DB
}

// NewNamespaceService creates a namespace service.
func NewNamespaceService(clusterSvc *cluster.ClusterService, db *gorm.DB) *NamespaceService {
	return &NamespaceService{clusterSvc: clusterSvc, db: db}
}

// Migrate creates or updates namespace binding tables.
func (s *NamespaceService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&NamespaceBinding{})
}

func (s *NamespaceService) binding(clusterID uint64, name string) (*NamespaceBinding, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var item NamespaceBinding
	if err := s.db.Where("cluster_id = ? AND namespace = ?", clusterID, name).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("k8s.namespace.binding_required")
		}
		return nil, err
	}
	return &item, nil
}

// RequireWrite verifies that a namespace has an explicit ownership binding and
// the requested mutation action. Empty action lists deliberately deny writes.
func (s *NamespaceService) RequireWrite(clusterID uint64, name, action string) error {
	binding, err := s.binding(clusterID, name)
	if err != nil {
		return err
	}
	var actions []string
	if err := json.Unmarshal([]byte(binding.AllowedActions), &actions); err != nil {
		return errors.New("k8s.namespace.binding_invalid")
	}
	for _, allowed := range actions {
		if strings.TrimSpace(allowed) == action {
			return nil
		}
	}
	return errors.New("k8s.namespace.action_forbidden")
}

// List returns namespaces visible through a Kubernetes cluster.
func (s *NamespaceService) List(clusterID uint64, dataScope *common.DataScopeReq) (*NamespaceListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("k8s.namespace.list_failed")
	}

	items := make([]NamespaceItem, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		items = append(items, toNamespaceItem(&ns))
	}
	return &NamespaceListResponse{Items: items, Total: len(items)}, nil
}

// Create creates a Kubernetes namespace.
func (s *NamespaceService) Create(clusterID uint64, req CreateNamespaceRequest, dataScope *common.DataScopeReq) (*NamespaceItem, error) {
	if req.BusinessScopeID == 0 || strings.TrimSpace(req.Environment) == "" {
		return nil, errors.New("k8s.namespace.binding_required")
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	actions, _ := json.Marshal(req.AllowedActions)
	binding := NamespaceBinding{ClusterID: clusterID, Namespace: req.Name, BusinessScopeID: req.BusinessScopeID, Environment: req.Environment, AllowedActions: string(actions)}
	if err := s.db.Where("cluster_id = ? AND namespace = ?", clusterID, req.Name).FirstOrCreate(&binding).Error; err != nil {
		return nil, err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		_ = s.db.Delete(&binding).Error
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   req.Name,
			Labels: req.Labels,
		},
	}
	created, err := clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		_ = s.db.Delete(&binding).Error
		return nil, errors.New("k8s.namespace.create_failed")
	}
	item := toNamespaceItem(created)
	return &item, nil
}

// Delete removes a Kubernetes namespace.
func (s *NamespaceService) Delete(clusterID uint64, name, resourceVersion string, dataScope *common.DataScopeReq) error {
	if _, err := s.binding(clusterID, name); err != nil {
		return err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	options := metav1.DeleteOptions{}
	if strings.TrimSpace(resourceVersion) != "" {
		options.Preconditions = &metav1.Preconditions{ResourceVersion: &resourceVersion}
	}
	if err := clientset.CoreV1().Namespaces().Delete(ctx, name, options); err != nil {
		if strings.Contains(err.Error(), "precondition") || strings.Contains(err.Error(), "conflict") {
			return errors.New("k8s.namespace.resource_version_conflict")
		}
		return errors.New("k8s.namespace.delete_failed")
	}
	return s.db.Where("cluster_id = ? AND namespace = ?", clusterID, name).Delete(&NamespaceBinding{}).Error
}

func toNamespaceItem(ns *corev1.Namespace) NamespaceItem {
	created := ""
	if !ns.CreationTimestamp.IsZero() {
		created = ns.CreationTimestamp.Format(time.RFC3339)
	}
	return NamespaceItem{
		Name:              ns.Name,
		Status:            string(ns.Status.Phase),
		Labels:            ns.Labels,
		CreationTimestamp: created,
	}
}
