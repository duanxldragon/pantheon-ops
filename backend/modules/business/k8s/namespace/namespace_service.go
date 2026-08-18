package namespace

import (
	"context"
	"errors"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const syncTimeout = 30 * time.Second

type NamespaceService struct {
	clusterSvc *cluster.ClusterService
}

func NewNamespaceService(clusterSvc *cluster.ClusterService) *NamespaceService {
	return &NamespaceService{clusterSvc: clusterSvc}
}

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

func (s *NamespaceService) Create(clusterID uint64, req CreateNamespaceRequest, dataScope *common.DataScopeReq) (*NamespaceItem, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
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
		return nil, errors.New("k8s.namespace.create_failed")
	}
	item := toNamespaceItem(created)
	return &item, nil
}

func (s *NamespaceService) Delete(clusterID uint64, name string, dataScope *common.DataScopeReq) error {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	if err := clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return errors.New("k8s.namespace.delete_failed")
	}
	return nil
}

func toNamespaceItem(ns *corev1.Namespace) NamespaceItem {
	created := ""
	if !ns.CreationTimestamp.IsZero() {
		created = ns.CreationTimestamp.Time.Format(time.RFC3339)
	}
	return NamespaceItem{
		Name:              ns.Name,
		Status:            string(ns.Status.Phase),
		Labels:            ns.Labels,
		CreationTimestamp: created,
	}
}
