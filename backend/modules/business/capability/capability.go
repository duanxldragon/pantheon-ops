package capability

import (
	"context"

	"pantheon-base/pkg/common"

	"gorm.io/datatypes"
)

// BizScopeRef is the minimal business-scope projection shared across business modules.
type BizScopeRef struct {
	ID          uint64
	Code        string
	Name        string
	Environment string
	Status      string
	DeptID      uint64
}

// BizScopeReader reads business-scope data through the owner module contract.
type BizScopeReader interface {
	GetActive(ctx context.Context, id uint64, scope *common.DataScopeReq) (BizScopeRef, error)
	ResolveActiveByCodes(ctx context.Context, codes []string, scope *common.DataScopeReq) (map[string]BizScopeRef, error)
}

// HostRef is the minimal CMDB host projection shared across business modules.
type HostRef struct {
	ID                uint64
	Hostname          string
	IP                string
	SSHPort           int
	OS                string
	Status            string
	BusinessScopeID   uint64
	BusinessScopeCode string
	BusinessScopeName string
	LabelValues       datatypes.JSON
	DeptID            uint64
}

// HostPage contains a paginated host projection.
type HostPage struct {
	Items []HostRef
	Total int64
}

// HostIDsQuery requests hosts by identifier within a data scope.
type HostIDsQuery struct {
	HostIDs   []uint64
	DataScope *common.DataScopeReq
}

// HostScopeQuery requests hosts owned by a business scope.
type HostScopeQuery struct {
	BusinessScopeID uint64
	DataScope       *common.DataScopeReq
}

// AvailableHostQuery requests hosts without a business-scope owner.
type AvailableHostQuery struct {
	DataScope *common.DataScopeReq
}

// BindOwnershipRequest assigns hosts to a business scope.
type BindOwnershipRequest struct {
	BusinessScopeID   uint64
	BusinessScopeCode string
	BusinessScopeName string
	HostIDs           []uint64
	Actor             string
	DataScope         *common.DataScopeReq
}

// UnbindOwnershipRequest removes a host's business-scope ownership.
type UnbindOwnershipRequest struct {
	BusinessScopeID uint64
	HostID          uint64
	Actor           string
	DataScope       *common.DataScopeReq
}

// CMDBHostReader exposes owner-module host read capabilities.
type CMDBHostReader interface {
	GetByIDs(ctx context.Context, req HostIDsQuery) (HostPage, error)
	ListByBusinessScope(ctx context.Context, req HostScopeQuery) (HostPage, error)
	ListAvailable(ctx context.Context, req AvailableHostQuery) (HostPage, error)
	HasBusinessScopeReferences(ctx context.Context, businessScopeID uint64) (bool, error)
}

// CMDBOwnershipCommand exposes owner-module host ownership commands.
type CMDBOwnershipCommand interface {
	Bind(ctx context.Context, req BindOwnershipRequest) error
	Unbind(ctx context.Context, req UnbindOwnershipRequest) error
	WithBusinessScopeOwnershipLock(ctx context.Context, businessScopeID uint64, action func() error) error
}

// K8sTargetQuery identifies a Kubernetes workload target.
type K8sTargetQuery struct {
	ClusterID    uint64
	Namespace    string
	WorkloadKind string
	WorkloadName string
	DataScope    *common.DataScopeReq
}

// K8sTargetRef is the minimal Kubernetes target projection for service instances.
type K8sTargetRef struct {
	ClusterID       uint64
	Namespace       string
	WorkloadKind    string
	WorkloadName    string
	BusinessScopeID uint64
	DeptID          uint64
}

// K8sTargetReader exposes owner-module Kubernetes target reads.
type K8sTargetReader interface {
	ResolveTarget(ctx context.Context, req K8sTargetQuery) (K8sTargetRef, error)
}

// ServiceInstanceStateTransition carries a state transition command for a service instance.
type ServiceInstanceStateTransition struct {
	InstanceID               uint64
	Action                   string
	ExpectedLifecycleVersion uint64
	DesiredState             string
	ObservedState            string
	HealthState              string
	DesiredVersion           string
	CurrentVersion           string
	HealthMessage            string
	HealthRevision           string
	CorrelationID            string
}

// ServiceInstanceStateCommand exposes owner-module service-instance state commands.
type ServiceInstanceStateCommand interface {
	ApplyServiceInstanceState(
		ctx context.Context,
		req ServiceInstanceStateTransition,
		actor string,
		scope *common.DataScopeReq,
	) error
}

// ServiceInstanceReferenceReader exposes active host-reference checks.
type ServiceInstanceReferenceReader interface {
	HasActiveHostReferences(ctx context.Context, hostID uint64, scope *common.DataScopeReq) (bool, error)
}
