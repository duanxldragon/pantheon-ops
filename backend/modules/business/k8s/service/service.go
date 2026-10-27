// Package service provides live Kubernetes Service resource management.
package service

import (
	"context"
	"errors"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	msgParamInvalid = "common.param_invalid"
	opTimeout       = 30 * time.Second
)

// ServicePort describes one port of a Service.
//
//nolint:revive // retained as the public Service port DTO name.
type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort int32  `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
	Protocol   string `json:"protocol"`
}

// ServiceItem summarizes a Kubernetes Service.
//
//nolint:revive // retained as the public Service DTO name.
type ServiceItem struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Type           string            `json:"type"`
	ClusterIP      string            `json:"clusterIP"`
	Ports          []ServicePort     `json:"ports"`
	Selector       map[string]string `json:"selector"`
	LoadBalancerIP string            `json:"loadBalancerIP"`
}

// ServiceListResponse contains Service summaries.
//
//nolint:revive // retained as the public response DTO name.
type ServiceListResponse struct {
	Items []ServiceItem `json:"items"`
	Total int           `json:"total"`
}

// ServiceListQuery filters Services.
//
//nolint:revive // retained as the public query DTO name.
type ServiceListQuery struct {
	Namespace string `form:"namespace"`
	Name      string `form:"name"`
}

// CreateServiceRequest contains Service creation fields.
//
//nolint:revive // retained as the public request DTO name.
type CreateServiceRequest struct {
	Name     string            `json:"name" binding:"required"`
	Type     string            `json:"type"`
	Ports    []ServicePort     `json:"ports" binding:"required"`
	Selector map[string]string `json:"selector"`
}

// ServiceService manages Kubernetes Services through client-go.
//
//nolint:revive // retained as the public service name for this package.
type ServiceService struct {
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewServiceService creates a Service service.
func NewServiceService(clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *ServiceService {
	return &ServiceService{clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// Migrate is a no-op for the live resource type.
func (s *ServiceService) Migrate() error {
	return nil
}

// List returns Services in a namespace.
func (s *ServiceService) List(clusterID uint64, query ServiceListQuery, dataScope *common.DataScopeReq) (*ServiceListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().Services(query.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("k8s.service.list_failed")
	}
	items := make([]ServiceItem, 0, len(list.Items))
	for i := range list.Items {
		svc := &list.Items[i]
		if query.Name != "" && svc.Name != query.Name {
			continue
		}
		items = append(items, toItem(svc))
	}
	return &ServiceListResponse{Items: items, Total: len(items)}, nil
}

// Get returns one Service.
func (s *ServiceService) Get(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("k8s.service.not_found")
	}
	item := toItem(svc)
	return &item, nil
}

// Create creates a Service.
func (s *ServiceService) Create(clusterID uint64, namespace string, req CreateServiceRequest, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	if s.namespaceSvc == nil {
		return nil, errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "service:create"); err != nil {
		return nil, err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceType(req.Type),
			Selector: req.Selector,
			Ports:    toServicePorts(req.Ports),
		},
	}
	created, err := clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.New("k8s.service.create_failed")
	}
	item := toItem(created)
	return &item, nil
}

// Delete removes a Service.
func (s *ServiceService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	if s.namespaceSvc == nil {
		return errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "service:delete"); err != nil {
		return err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	if err := clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return errors.New("k8s.service.delete_failed")
	}
	return nil
}

func toItem(svc *corev1.Service) ServiceItem {
	ports := make([]ServicePort, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		ports = append(ports, ServicePort{
			Name: p.Name,
			Port: p.Port,
			// IntVal is already int32, so no narrowing conversion happens here; a
			// named (string) target port has no numeric value and stays 0.
			TargetPort: p.TargetPort.IntVal,
			NodePort:   p.NodePort,
			Protocol:   string(p.Protocol),
		})
	}
	lbIP := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		lbIP = svc.Status.LoadBalancer.Ingress[0].IP
	}
	return ServiceItem{
		Name:           svc.Name,
		Namespace:      svc.Namespace,
		Type:           string(svc.Spec.Type),
		ClusterIP:      svc.Spec.ClusterIP,
		Ports:          ports,
		Selector:       svc.Spec.Selector,
		LoadBalancerIP: lbIP,
	}
}

func toServicePorts(ports []ServicePort) []corev1.ServicePort {
	out := make([]corev1.ServicePort, 0, len(ports))
	for _, p := range ports {
		sp := corev1.ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			NodePort:   p.NodePort,
			Protocol:   corev1.Protocol(p.Protocol),
			TargetPort: intstr.FromInt32(p.TargetPort),
		}
		out = append(out, sp)
	}
	return out
}
