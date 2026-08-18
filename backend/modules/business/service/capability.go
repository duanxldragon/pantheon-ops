package service

import (
	"context"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
)

type ApplicationRef struct {
	ID              uint64
	Code            string
	Name            string
	BusinessScopeID uint64
	DeptID          uint64
	Status          string
}

// TransitionStateCommand exposes the Service-owned state command through the
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

func (m *Manager) HasActiveHostReferences(ctx context.Context, hostID uint64, scope *common.DataScopeReq) (bool, error) {
	var count int64
	err := m.instanceQuery(scope).WithContext(ctx).
		Where("target_type = ? AND host_id = ? AND desired_state <> ?", TargetTypeVM, hostID, DesiredStateRetired).
		Count(&count).Error
	return count > 0, err
}

var _ bizcap.ServiceInstanceReferenceReader = (*Manager)(nil)

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

type InstanceRef struct {
	ID               uint64
	ServiceID        uint64
	ApplicationID    uint64
	BusinessScopeID  uint64
	DeptID           uint64
	Environment      string
	TargetType       string
	HostID           uint64
	K8sClusterID     uint64
	Namespace        string
	WorkloadKind     string
	WorkloadName     string
	DesiredVersion   string
	CurrentVersion   string
	RollbackVersion  string
	DesiredState     string
	ObservedState    string
	HealthState      string
	HealthMessage    string
	HealthRevision   string
	LastTransitionID string
	LifecycleVersion uint64
	Status           string
}

type Reader interface {
	GetApplication(ctx context.Context, id uint64, scope *common.DataScopeReq) (ApplicationRef, error)
	GetService(ctx context.Context, id uint64, scope *common.DataScopeReq) (ServiceRef, error)
	GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (InstanceRef, error)
}

type Command interface {
	CreateInstance(ctx context.Context, req CreateInstanceRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error)
}

type StateCommand interface {
	TransitionState(ctx context.Context, req InstanceStateTransitionRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error)
}

type Dependencies struct {
	BizScopeReader bizcap.BizScopeReader
	CMDBReader     bizcap.CMDBHostReader
	K8sReader      bizcap.K8sTargetReader
}
