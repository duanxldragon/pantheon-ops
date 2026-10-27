package service

import (
	"context"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

// ApplicationRef is the minimal application projection exposed to other modules.
type ApplicationRef struct {
	ID              uint64
	Code            string
	Name            string
	BusinessScopeID uint64
	DeptID          uint64
	Status          string
}

// ApplyServiceInstanceState exposes the Service-owned state command through the
// narrow business capability contract consumed by Deploy.
func (m *Manager) ApplyServiceInstanceState(ctx context.Context, req bizcap.ServiceInstanceStateTransition, actor string, scope *common.DataScopeReq) error {
	_, err := m.TransitionState(ctx, InstanceStateTransitionRequest{
		InstanceID:               req.InstanceID,
		Action:                   req.Action,
		ExpectedLifecycleVersion: req.ExpectedLifecycleVersion,
		DesiredState:             req.DesiredState,
		ObservedState:            req.ObservedState,
		HealthState:              req.HealthState,
		DesiredVersion:           req.DesiredVersion,
		CurrentVersion:           req.CurrentVersion,
		HealthMessage:            req.HealthMessage,
		HealthRevision:           req.HealthRevision,
		CorrelationID:            req.CorrelationID,
	}, actor, scope)
	return err
}

var _ bizcap.ServiceInstanceStateCommand = (*Manager)(nil)

// HasActiveHostReferences reports whether a host is referenced by an active service instance.
func (m *Manager) HasActiveHostReferences(ctx context.Context, hostID uint64, scope *common.DataScopeReq) (bool, error) {
	var count int64
	err := m.instanceQuery(scope).WithContext(ctx).
		Where("target_type = ? AND host_id = ? AND desired_state <> ?", TargetTypeVM, hostID, DesiredStateRetired).
		Count(&count).Error
	return count > 0, err
}

var _ bizcap.ServiceInstanceReferenceReader = (*Manager)(nil)

// ServiceRef is the minimal service projection exposed to other modules.
//
//nolint:revive // retained as the domain projection name in the service package.
type ServiceRef struct {
	ID              uint64
	ApplicationID   uint64
	Code            string
	Name            string
	RuntimeType     string
	BusinessScopeID uint64
	DeptID          uint64
	Status          string
}

// InstanceRef is the minimal service-instance projection exposed to other modules.
type InstanceRef struct {
	ID               uint64 `json:"id"`
	ServiceID        uint64 `json:"serviceId"`
	ApplicationID    uint64 `json:"applicationId"`
	BusinessScopeID  uint64 `json:"businessScopeId"`
	DeptID           uint64 `json:"deptId"`
	Environment      string `json:"environment"`
	TargetType       string `json:"targetType"`
	HostID           uint64 `json:"hostId"`
	K8sClusterID     uint64 `json:"k8sClusterId"`
	Namespace        string `json:"namespace"`
	WorkloadKind     string `json:"workloadKind"`
	WorkloadName     string `json:"workloadName"`
	DesiredVersion   string `json:"desiredVersion"`
	CurrentVersion   string `json:"currentVersion"`
	RollbackVersion  string `json:"rollbackVersion"`
	DesiredState     string `json:"desiredState"`
	ObservedState    string `json:"observedState"`
	HealthState      string `json:"healthState"`
	HealthMessage    string `json:"healthMessage"`
	HealthRevision   string `json:"healthRevision"`
	LastTransitionID string `json:"lastTransitionId"`
	LifecycleVersion uint64 `json:"lifecycleVersion"`
	Status           string `json:"status"`
}

// Reader exposes service-owned read capabilities to other business modules.
type Reader interface {
	GetApplication(ctx context.Context, id uint64, scope *common.DataScopeReq) (ApplicationRef, error)
	GetService(ctx context.Context, id uint64, scope *common.DataScopeReq) (ServiceRef, error)
	GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (InstanceRef, error)
}

// Command exposes service-owned write capabilities to other business modules.
type Command interface {
	CreateInstance(ctx context.Context, req CreateInstanceRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error)
}

// StateCommand exposes service-instance state transitions to other modules.
type StateCommand interface {
	TransitionState(ctx context.Context, req InstanceStateTransitionRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error)
}

// Dependencies contains owner-module readers required by the service module.
type Dependencies struct {
	BizScopeReader bizcap.BizScopeReader
	CMDBReader     bizcap.CMDBHostReader
	K8sReader      bizcap.K8sTargetReader
}
