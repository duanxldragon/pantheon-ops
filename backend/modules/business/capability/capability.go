package capability

import (
	"context"

	"pantheon-base/pkg/common"

	"gorm.io/datatypes"
)

type BizScopeRef struct {
	ID          uint64
	Code        string
	Name        string
	Environment string
	Status      string
	DeptID      uint64
}

type BizScopeReader interface {
	GetActive(ctx context.Context, id uint64, scope *common.DataScopeReq) (BizScopeRef, error)
	ResolveActiveByCodes(ctx context.Context, codes []string, scope *common.DataScopeReq) (map[string]BizScopeRef, error)
}

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

type HostPage struct {
	Items []HostRef
	Total int64
}

type HostIDsQuery struct {
	HostIDs   []uint64
	DataScope *common.DataScopeReq
}

type HostScopeQuery struct {
	BusinessScopeID uint64
	DataScope       *common.DataScopeReq
}

type AvailableHostQuery struct {
	DataScope *common.DataScopeReq
}

type BindOwnershipRequest struct {
	BusinessScopeID   uint64
	BusinessScopeCode string
	BusinessScopeName string
	HostIDs           []uint64
	Actor             string
	DataScope         *common.DataScopeReq
}

type UnbindOwnershipRequest struct {
	BusinessScopeID uint64
	HostID          uint64
	Actor           string
	DataScope       *common.DataScopeReq
}

type CMDBHostReader interface {
	GetByIDs(ctx context.Context, req HostIDsQuery) (HostPage, error)
	ListByBusinessScope(ctx context.Context, req HostScopeQuery) (HostPage, error)
	ListAvailable(ctx context.Context, req AvailableHostQuery) (HostPage, error)
	HasBusinessScopeReferences(ctx context.Context, businessScopeID uint64) (bool, error)
}

type CMDBOwnershipCommand interface {
	Bind(ctx context.Context, req BindOwnershipRequest) error
	Unbind(ctx context.Context, req UnbindOwnershipRequest) error
	WithBusinessScopeOwnershipLock(ctx context.Context, businessScopeID uint64, action func() error) error
}

type K8sTargetQuery struct {
	ClusterID    uint64
	Namespace    string
	WorkloadKind string
	WorkloadName string
	DataScope    *common.DataScopeReq
}

type K8sTargetRef struct {
	ClusterID       uint64
	Namespace       string
	WorkloadKind    string
	WorkloadName    string
	BusinessScopeID uint64
	DeptID          uint64
}

type K8sTargetReader interface {
	ResolveTarget(ctx context.Context, req K8sTargetQuery) (K8sTargetRef, error)
}

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

type ServiceInstanceStateCommand interface {
	ApplyServiceInstanceState(
		ctx context.Context,
		req ServiceInstanceStateTransition,
		actor string,
		scope *common.DataScopeReq,
	) error
}

type ServiceInstanceReferenceReader interface {
	HasActiveHostReferences(ctx context.Context, hostID uint64, scope *common.DataScopeReq) (bool, error)
}
