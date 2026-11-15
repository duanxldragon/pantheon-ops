package service

import (
	"context"
	"errors"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	validation "k8s.io/apimachinery/pkg/util/validation"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const msgServiceConflict = "business.k8s.service.resourceVersionConflict"

// UpdateServiceRequest changes only mutable routing fields on a reviewed Service.
// Omitted fields retain the provider's values. UID and version prevent changing
// a replaced Service or silently overwriting concurrent changes.
type UpdateServiceRequest struct {
	UID             string             `json:"uid" binding:"required"`
	ResourceVersion string             `json:"resourceVersion" binding:"required"`
	Ports           *[]ServicePort     `json:"ports"`
	Selector        *map[string]string `json:"selector"`
}

func validateServiceUpdate(req UpdateServiceRequest) error {
	if strings.TrimSpace(req.UID) == "" || strings.TrimSpace(req.ResourceVersion) == "" || len(req.UID) > 128 || len(req.ResourceVersion) > 128 || (req.Ports == nil && req.Selector == nil) {
		return errors.New(msgParamInvalid)
	}
	if req.Ports != nil {
		if len(*req.Ports) == 0 || len(*req.Ports) > 64 {
			return errors.New(msgParamInvalid)
		}
		names := map[string]bool{}
		for _, port := range *req.Ports {
			if port.Port < 1 || port.Port > 65535 || port.TargetPort < 0 || port.TargetPort > 65535 || port.NodePort < 0 || port.NodePort > 65535 {
				return errors.New(msgParamInvalid)
			}
			if port.TargetPortName != "" && (port.TargetPort != 0 || len(validation.IsValidPortName(port.TargetPortName)) > 0) {
				return errors.New(msgParamInvalid)
			}
			if port.Protocol != "" && port.Protocol != "TCP" && port.Protocol != "UDP" && port.Protocol != "SCTP" {
				return errors.New(msgParamInvalid)
			}
			if port.Name != "" && (names[port.Name] || len(validation.IsDNS1123Label(port.Name)) > 0) {
				return errors.New(msgParamInvalid)
			}
			if len(*req.Ports) > 1 && port.Name == "" {
				return errors.New(msgParamInvalid)
			}
			names[port.Name] = true
		}
	}
	if req.Selector != nil {
		if len(*req.Selector) > 64 {
			return errors.New(msgParamInvalid)
		}
		for key, value := range *req.Selector {
			if len(validation.IsQualifiedName(key)) > 0 || len(validation.IsValidLabelValue(value)) > 0 {
				return errors.New(msgParamInvalid)
			}
		}
	}
	return nil
}

// UpdateContext updates with the API server's resourceVersion compare-and-swap.
func (s *ServiceService) UpdateContext(ctx context.Context, clusterID uint64, namespace, name string, req UpdateServiceRequest, scope *common.DataScopeReq) (*ServiceItem, error) {
	if err := validateServiceUpdate(req); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	api, err := s.mutationClient(ctx, clusterID, namespace, "service:update", scope)
	if err != nil {
		return nil, err
	}
	return updateService(ctx, api, name, req)
}

func (s *ServiceService) mutationClient(ctx context.Context, clusterID uint64, namespace, action string, scope *common.DataScopeReq) (typedcorev1.ServiceInterface, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, errors.New(msgParamInvalid)
	}
	if s.namespaceSvc == nil {
		return nil, errors.New("k8s.namespace.binding_required")
	}
	client, err := s.clusterSvc.GetClientsetWithContext(ctx, clusterID, scope)
	if err != nil {
		return nil, err
	}
	if err := s.namespaceSvc.RequireWriteWithContext(ctx, clusterID, namespace, action); err != nil {
		return nil, err
	}
	return client.CoreV1().Services(namespace), nil
}

func updateService(ctx context.Context, api typedcorev1.ServiceInterface, name string, req UpdateServiceRequest) (*ServiceItem, error) {
	current, err := api.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, serviceProviderError(ctx, err, "business.k8s.service.updateFailed")
	}
	if string(current.UID) != req.UID || current.ResourceVersion != req.ResourceVersion {
		return nil, errors.New(msgServiceConflict)
	}
	desired := current.DeepCopy()
	if req.Selector != nil {
		desired.Spec.Selector = *req.Selector
	}
	if req.Ports != nil {
		desired.Spec.Ports = toServicePorts(*req.Ports)
	}
	updated, err := api.Update(ctx, desired, metav1.UpdateOptions{})
	if err != nil {
		return nil, serviceProviderError(ctx, err, "business.k8s.service.updateFailed")
	}
	item := toItem(updated)
	return &item, nil
}

// DeleteContext deletes the exact UID/version selected by the caller.
func (s *ServiceService) DeleteContext(ctx context.Context, clusterID uint64, namespace, name, uid, version string, scope *common.DataScopeReq) error {
	if strings.TrimSpace(uid) == "" || strings.TrimSpace(version) == "" {
		return errors.New(msgParamInvalid)
	}
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	api, err := s.mutationClient(ctx, clusterID, namespace, "service:delete", scope)
	if err != nil {
		return err
	}
	return deleteService(ctx, api, name, uid, version)
}

func deleteService(ctx context.Context, api typedcorev1.ServiceInterface, name, uid, version string) error {
	identity := types.UID(uid)
	err := api.Delete(ctx, name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &identity, ResourceVersion: &version}})
	if err != nil {
		return serviceProviderError(ctx, err, "k8s.service.delete_failed")
	}
	return nil
}

func serviceProviderError(ctx context.Context, err error, fallback string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if apierrors.IsConflict(err) {
		return errors.New(msgServiceConflict)
	}
	if apierrors.IsNotFound(err) {
		return errors.New("k8s.service.not_found")
	}
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
		return errors.New("k8s.rbac.permission_denied")
	}
	if apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) {
		return errors.New(msgParamInvalid)
	}
	return errors.New(fallback)
}
