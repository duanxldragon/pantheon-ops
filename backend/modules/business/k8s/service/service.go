// Package service provides live Kubernetes Service resource management.
package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const (
	msgParamInvalid    = "common.param_invalid"
	msgVersionRequired = "k8s.service.resource_version_required"
	msgVersionConflict = "k8s.service.resource_version_conflict"
	opTimeout          = 30 * time.Second
)

// ServicePort describes one port of a Service.
//
//nolint:revive // retained as the public Service port DTO name.
type ServicePort struct {
	Name       string             `json:"name"`
	Port       int32              `json:"port"`
	TargetPort intstr.IntOrString `json:"targetPort"`
	NodePort   int32              `json:"nodePort"`
	Protocol   string             `json:"protocol"`
}

// ServiceItem summarizes a Kubernetes Service.
//
//nolint:revive // retained as the public Service DTO name.
type ServiceItem struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Type            string            `json:"type"`
	ClusterIP       string            `json:"clusterIP"`
	Ports           []ServicePort     `json:"ports"`
	Selector        map[string]string `json:"selector"`
	LoadBalancerIP  string            `json:"loadBalancerIP"`
	ResourceVersion string            `json:"resourceVersion"`
	UID             string            `json:"uid"`
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

// UpdateServiceRequest contains editable routing fields and the read revision.
type UpdateServiceRequest struct {
	Type            string            `json:"type"`
	Ports           []ServicePort     `json:"ports" binding:"required"`
	Selector        map[string]string `json:"selector"`
	ResourceVersion string            `json:"resourceVersion" binding:"required"`
	UID             string            `json:"uid"`
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
	return s.ListWithContext(context.Background(), clusterID, query, dataScope)
}

// ListWithContext preserves the caller's cancellation and deadline.
func (s *ServiceService) ListWithContext(ctx context.Context, clusterID uint64, query ServiceListQuery, dataScope *common.DataScopeReq) (*ServiceListResponse, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	clientset, err := s.clusterSvc.GetClientsetWithContext(ctx, clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	list, err := clientset.CoreV1().Services(query.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, classifyProviderError(err)
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
	return s.GetWithContext(context.Background(), clusterID, namespace, name, dataScope)
}

// GetWithContext preserves the caller's cancellation and deadline.
func (s *ServiceService) GetWithContext(ctx context.Context, clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	clientset, err := s.clusterSvc.GetClientsetWithContext(ctx, clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	svc, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, classifyProviderError(err)
	}
	item := toItem(svc)
	return &item, nil
}

// Create creates a Service.
func (s *ServiceService) Create(clusterID uint64, namespace string, req CreateServiceRequest, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	return s.CreateWithContext(context.Background(), clusterID, namespace, req, dataScope)
}

// CreateWithContext validates ownership and structured routing before creation.
func (s *ServiceService) CreateWithContext(ctx context.Context, clusterID uint64, namespace string, req CreateServiceRequest, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	if len(validation.IsDNS1035Label(req.Name)) != 0 || len(validation.IsDNS1123Label(namespace)) != 0 {
		return nil, common.NewBadRequest(msgParamInvalid)
	}
	serviceType, ports, err := normalizeRouting(req.Type, req.Ports, req.Selector)
	if err != nil {
		return nil, err
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Type:     serviceType,
			Selector: req.Selector,
			Ports:    ports,
		},
	}
	var item *ServiceItem
	err = s.withWrite(ctx, clusterID, namespace, "service:create", dataScope, func(clientset kubernetes.Interface) error {
		created, err := clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
		if err != nil {
			return classifyProviderError(err)
		}
		result := toItem(created)
		item = &result
		return nil
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

// Update replaces structured routing while retaining provider-owned fields.
func (s *ServiceService) Update(clusterID uint64, namespace, name string, req UpdateServiceRequest, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	return s.UpdateWithContext(context.Background(), clusterID, namespace, name, req, dataScope)
}

// UpdateWithContext uses the caller's read revision for optimistic concurrency.
func (s *ServiceService) UpdateWithContext(ctx context.Context, clusterID uint64, namespace, name string, req UpdateServiceRequest, dataScope *common.DataScopeReq) (*ServiceItem, error) {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	if strings.TrimSpace(req.ResourceVersion) == "" {
		return nil, common.NewBadRequest(msgVersionRequired)
	}
	if len(validation.IsDNS1035Label(name)) != 0 || len(validation.IsDNS1123Label(namespace)) != 0 {
		return nil, common.NewBadRequest(msgParamInvalid)
	}
	var item *ServiceItem
	err := s.withWrite(ctx, clusterID, namespace, "service:update", dataScope, func(clientset kubernetes.Interface) error {
		var err error
		item, err = updateWithClientset(ctx, clientset, namespace, name, req)
		return err
	})
	if err != nil {
		return nil, err
	}
	return item, nil
}

func updateWithClientset(ctx context.Context, clientset kubernetes.Interface, namespace, name string, req UpdateServiceRequest) (*ServiceItem, error) {
	current, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, classifyProviderError(err)
	}
	if current.ResourceVersion != req.ResourceVersion || (req.UID != "" && string(current.UID) != req.UID) {
		return nil, common.NewConflict(msgVersionConflict)
	}
	if req.Type == "" {
		req.Type = string(current.Spec.Type)
	}
	serviceType, ports, err := normalizeRouting(req.Type, req.Ports, req.Selector)
	if err != nil {
		return nil, err
	}
	updated := current.DeepCopy()
	// Names identify named ports; unnamed ports require the same protocol and
	// port number. Preserve fields outside the editable DTO only for that
	// logical port. New ports must not inherit another port's opaque metadata.
	for index := range ports {
		for _, existing := range updated.Spec.Ports {
			namedMatch := ports[index].Name != "" && ports[index].Name == existing.Name
			unnamedMatch := ports[index].Name == "" && existing.Name == "" && ports[index].Protocol == existing.Protocol && ports[index].Port == existing.Port
			if namedMatch || unnamedMatch {
				ports[index].AppProtocol = existing.AppProtocol
				break
			}
		}
	}
	updated.Spec.Type, updated.Spec.Selector, updated.Spec.Ports = serviceType, req.Selector, ports
	result, err := clientset.CoreV1().Services(namespace).Update(ctx, updated, metav1.UpdateOptions{})
	if err != nil {
		return nil, classifyProviderError(err)
	}
	item := toItem(result)
	return &item, nil
}

// Delete refuses unversioned legacy deletion; callers must supply identity
// through DeleteWithResourceVersion.
func (s *ServiceService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	return s.DeleteWithResourceVersion(clusterID, namespace, name, "", "", dataScope)
}

// DeleteWithResourceVersion requires the resource identity captured by the caller.
func (s *ServiceService) DeleteWithResourceVersion(clusterID uint64, namespace, name, resourceVersion, uid string, dataScope *common.DataScopeReq) error {
	return s.DeleteWithResourceVersionAndContext(context.Background(), clusterID, namespace, name, resourceVersion, uid, dataScope)
}

// DeleteWithResourceVersionAndContext refuses stale or recreated resources.
func (s *ServiceService) DeleteWithResourceVersionAndContext(ctx context.Context, clusterID uint64, namespace, name, resourceVersion, uid string, dataScope *common.DataScopeReq) error {
	ctx, cancel := operationContext(ctx)
	defer cancel()
	if strings.TrimSpace(resourceVersion) == "" {
		return common.NewBadRequest(msgVersionRequired)
	}
	if strings.TrimSpace(uid) == "" || len(validation.IsDNS1035Label(name)) != 0 || len(validation.IsDNS1123Label(namespace)) != 0 {
		return common.NewBadRequest(msgParamInvalid)
	}
	identity := types.UID(uid)
	options := metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &resourceVersion, UID: &identity}}
	return s.withWrite(ctx, clusterID, namespace, "service:delete", dataScope, func(clientset kubernetes.Interface) error {
		if err := clientset.CoreV1().Services(namespace).Delete(ctx, name, options); err != nil {
			return classifyProviderError(err)
		}
		return nil
	})
}

func toItem(svc *corev1.Service) ServiceItem {
	ports := make([]ServicePort, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		ports = append(ports, ServicePort{
			Name:       p.Name,
			Port:       p.Port,
			TargetPort: p.TargetPort,
			NodePort:   p.NodePort,
			Protocol:   string(p.Protocol),
		})
	}
	lbIP := ""
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		lbIP = svc.Status.LoadBalancer.Ingress[0].IP
	}
	return ServiceItem{
		Name:            svc.Name,
		Namespace:       svc.Namespace,
		Type:            string(svc.Spec.Type),
		ClusterIP:       svc.Spec.ClusterIP,
		Ports:           ports,
		Selector:        svc.Spec.Selector,
		LoadBalancerIP:  lbIP,
		ResourceVersion: svc.ResourceVersion,
		UID:             string(svc.UID),
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
			TargetPort: p.TargetPort,
		}
		out = append(out, sp)
	}
	return out
}

func operationContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, opTimeout)
}

func (s *ServiceService) withWrite(ctx context.Context, clusterID uint64, namespace, action string, dataScope *common.DataScopeReq, write func(kubernetes.Interface) error) error {
	if s.namespaceSvc == nil {
		return common.NewForbidden("k8s.namespace.binding_required")
	}
	return s.namespaceSvc.WithWriteForClusterWithContext(ctx, clusterID, namespace, action, dataScope, write)
}

func classifyProviderError(err error) error {
	switch {
	case apierrors.IsConflict(err):
		return fmt.Errorf("%w: %w: %s", common.ErrConflict, err, msgVersionConflict)
	case apierrors.IsNotFound(err):
		return fmt.Errorf("%w: %w: k8s.service.not_found", common.ErrNotFound, err)
	case apierrors.IsForbidden(err):
		return fmt.Errorf("%w: %w: k8s.rbac.permission_denied", common.ErrForbidden, err)
	case apierrors.IsInvalid(err) || apierrors.IsBadRequest(err):
		return fmt.Errorf("%w: %w: %s", common.ErrBadRequest, err, msgParamInvalid)
	default:
		return err
	}
}

func normalizeRouting(serviceType string, ports []ServicePort, selector map[string]string) (corev1.ServiceType, []corev1.ServicePort, error) {
	if serviceType == "" {
		serviceType = string(corev1.ServiceTypeClusterIP)
	}
	if serviceType != string(corev1.ServiceTypeClusterIP) && serviceType != string(corev1.ServiceTypeNodePort) && serviceType != string(corev1.ServiceTypeLoadBalancer) {
		return "", nil, common.NewBadRequest(msgParamInvalid)
	}
	for key, value := range selector {
		if len(validation.IsQualifiedName(key)) != 0 || len(validation.IsValidLabelValue(value)) != 0 {
			return "", nil, common.NewBadRequest(msgParamInvalid)
		}
	}
	if len(ports) == 0 {
		return "", nil, common.NewBadRequest(msgParamInvalid)
	}
	normalized := append([]ServicePort(nil), ports...)
	names := make(map[string]bool)
	for index := range normalized {
		port := &normalized[index]
		if port.Protocol == "" {
			port.Protocol = string(corev1.ProtocolTCP)
		}
		if port.Port < 1 || port.Port > 65535 || port.NodePort < 0 || port.NodePort > 65535 || (port.NodePort != 0 && serviceType == string(corev1.ServiceTypeClusterIP)) || (port.Protocol != string(corev1.ProtocolTCP) && port.Protocol != string(corev1.ProtocolUDP) && port.Protocol != string(corev1.ProtocolSCTP)) {
			return "", nil, common.NewBadRequest(msgParamInvalid)
		}
		if (len(normalized) > 1 && port.Name == "") || (port.Name != "" && (len(validation.IsDNS1123Label(port.Name)) != 0 || names[port.Name])) {
			return "", nil, common.NewBadRequest(msgParamInvalid)
		}
		names[port.Name] = true
		if port.TargetPort.Type == intstr.Int {
			if port.TargetPort.IntVal == 0 {
				port.TargetPort = intstr.FromInt32(port.Port)
			}
			if port.TargetPort.IntVal < 1 || port.TargetPort.IntVal > 65535 {
				return "", nil, common.NewBadRequest(msgParamInvalid)
			}
		} else if port.TargetPort.Type != intstr.String || len(validation.IsValidPortName(port.TargetPort.StrVal)) != 0 {
			return "", nil, common.NewBadRequest(msgParamInvalid)
		}
	}
	return corev1.ServiceType(serviceType), toServicePorts(normalized), nil
}
