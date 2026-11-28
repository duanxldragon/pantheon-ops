package deploy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/service"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const stateStaleKey = "business.service.state_stale"

const errDeployServiceLinkInvalid = "business.deploy.task.serviceLinkInvalid"

// Only a valid owner binding superseded by a later lifecycle command gets this
// sentinel. Cancellation may settle its local execution without changing that
// newer owner; malformed snapshots and owner/transaction failures still refuse.
var errServiceExecutionSuperseded = errors.New(stateStaleKey)

// Manual reports bind to the admitted execution, not the mutable task command
// version. The snapshot survives terminal replay and changes on every retry.
func taskExecutionVersion(task *DeployTask) string {
	if task.ServiceID == 0 && task.ServiceInstanceID == 0 {
		return ""
	}
	var frozen deployTargetSnapshot
	if task.ServiceID == 0 || task.ServiceInstanceID == 0 || json.Unmarshal(task.TargetSnapshot, &frozen) != nil || frozen.ServiceState == nil || frozen.ExecutionGeneration == 0 {
		return ""
	}
	raw, _ := json.Marshal([]any{"linked-deploy-execution-v1", task.ID, task.ServiceID, task.ServiceInstanceID, frozen.ExecutionGeneration, frozen.ServiceState, frozen.Hosts})
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// SetServiceReader configures the Service owner and its transaction seam before
// workers start. An owner without the seam may perform discovery but cannot admit
// linked task writes.
func (s *DeployService) SetServiceReader(reader service.Reader) {
	if nilServiceCapability(reader) {
		reader = nil
	}
	s.serviceReader = reader
	s.serviceReaderTransaction = nil
	if owner, ok := reader.(interface{ WithReaderTransaction(*gorm.DB) service.Reader }); ok {
		s.serviceReaderTransaction = owner.WithReaderTransaction
	}
}

type unboundServiceReader struct{}

func (unboundServiceReader) GetApplication(context.Context, uint64, *common.DataScopeReq) (service.ApplicationRef, error) {
	return service.ApplicationRef{}, errors.New(errDeployServiceLinkInvalid)
}
func (unboundServiceReader) GetService(context.Context, uint64, *common.DataScopeReq) (service.ServiceRef, error) {
	return service.ServiceRef{}, errors.New(errDeployServiceLinkInvalid)
}
func (unboundServiceReader) GetInstance(context.Context, uint64, *common.DataScopeReq) (service.InstanceRef, error) {
	return service.InstanceRef{}, errors.New(errDeployServiceLinkInvalid)
}

// Service hides invisible owners behind these documented not-found keys.
// Preserve unexpected reader/database/transaction errors for diagnostics and
// server error classification instead of calling them invalid input.
func serviceLinkReadError(err error) error {
	switch err.Error() {
	case "business.service.instance_not_found", "business.service.service_not_found", "business.service.application_not_found":
		return errors.New(errDeployServiceLinkInvalid)
	default:
		return err
	}
}

func (s *DeployService) validateServiceLink(task *DeployTask, scope *common.DataScopeReq, resolved []cmdbHostSnapshot) error {
	if task.ServiceID == 0 && task.ServiceInstanceID == 0 {
		return nil
	}
	invalid := errors.New(errDeployServiceLinkInvalid)
	if task.ServiceID == 0 || task.ServiceInstanceID == 0 || nilServiceCapability(s.serviceReader) {
		return invalid
	}
	// Resolve/lock CMDB targets before Service owner rows, including group
	// mutations, to preserve the completion path's CMDB -> Service order.
	if resolved == nil {
		var err error
		resolved, err = s.resolveTaskTargets(task, scope)
		if err != nil {
			return err
		}
	}
	ctx := s.requestContext()
	instance, err := s.serviceReader.GetInstance(ctx, task.ServiceInstanceID, scope)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return serviceLinkReadError(err)
	}
	if instance.ID != task.ServiceInstanceID || instance.ServiceID != task.ServiceID ||
		instance.BusinessScopeID != task.BusinessScopeID || instance.Status != service.InstanceStatusActive ||
		instance.DesiredState == service.DesiredStateRetired || instance.ObservedState == service.ObservedStateRetired {
		return invalid
	}
	svc, err := s.serviceReader.GetService(ctx, task.ServiceID, scope)
	if err != nil {
		return serviceLinkReadError(err)
	}
	app, err := s.serviceReader.GetApplication(ctx, svc.ApplicationID, scope)
	if err != nil {
		return serviceLinkReadError(err)
	}
	if svc.ID != task.ServiceID || svc.ApplicationID != instance.ApplicationID || svc.Status != service.StatusActive ||
		app.ID != svc.ApplicationID || app.Status != service.StatusActive || app.BusinessScopeID != task.BusinessScopeID ||
		svc.BusinessScopeID != task.BusinessScopeID {
		return invalid
	}
	// Deploy currently has only a host executor. One instance cannot account for
	// a multi-host task, and a Kubernetes target cannot be treated as a VM.
	if instance.TargetType != service.TargetTypeVM || instance.HostID == 0 || instance.K8sClusterID != 0 ||
		instance.Namespace != "" || instance.WorkloadKind != "" || instance.WorkloadName != "" {
		return invalid
	}
	if len(resolved) != 1 || resolved[0].ID != instance.HostID || resolved[0].BusinessScopeID != task.BusinessScopeID {
		return invalid
	}
	// Snapshot the owner name here so task detail and CSV exports keep showing
	// it after the service row is renamed.
	task.ServiceName = svc.Name
	return nil
}

func serviceLinkTask(scopeID, serviceID, instanceID uint64, targetType string, targetIDs []uint64) *DeployTask {
	raw, _ := json.Marshal(targetIDs)
	return &DeployTask{BusinessScopeID: scopeID, ServiceID: serviceID, ServiceInstanceID: instanceID, TargetType: targetType, TargetIDs: datatypes.JSON(raw)}
}

// Completion uses the admitted snapshot rather than rediscovering a group that
// may have changed since start. State commands remain bound to the locked owner.
func (s *DeployService) serviceInstanceForCallback(task DeployTask, requireSnapshot bool, scope *common.DataScopeReq) (service.InstanceRef, error) {
	if task.ServiceID == 0 || task.ServiceInstanceID == 0 || nilServiceCapability(s.serviceReader) || nilServiceInstanceState(s.serviceState) {
		return service.InstanceRef{}, errors.New(errDeployServiceLinkInvalid)
	}
	if _, unbound := s.serviceState.(unboundServiceInstanceState); unbound {
		return service.InstanceRef{}, errors.New("deploy.serviceState.transactionNotSupported")
	}
	instance, err := s.serviceReader.GetInstance(s.requestContext(), task.ServiceInstanceID, scope)
	if err != nil {
		return instance, err
	}
	if instance.ID != task.ServiceInstanceID || instance.ServiceID != task.ServiceID || instance.BusinessScopeID != task.BusinessScopeID || instance.TargetType != service.TargetTypeVM {
		return instance, errors.New(errDeployServiceLinkInvalid)
	}
	if requireSnapshot {
		var snapshot deployTargetSnapshot
		if json.Unmarshal(task.TargetSnapshot, &snapshot) != nil || len(snapshot.Hosts) != 1 || snapshot.Hosts[0].HostID != instance.HostID || snapshot.Hosts[0].BusinessScopeID != task.BusinessScopeID {
			return instance, errors.New(errDeployServiceLinkInvalid)
		}
		if snapshot.ServiceState == nil || snapshot.ServiceState.InstanceID != instance.ID || snapshot.ExecutionGeneration == 0 {
			return instance, errors.New(errDeployServiceLinkInvalid)
		}
		if snapshot.ServiceState.LifecycleVersion != instance.LifecycleVersion || snapshot.ServiceState.TransitionID != instance.LastTransitionID {
			return instance, errServiceExecutionSuperseded
		}
	}
	return instance, nil
}

func (s *DeployService) beginServiceStateSnapshot(task DeployTask, actor string, scope *common.DataScopeReq) (*deployServiceStateSnapshot, error) {
	if task.ServiceID == 0 && task.ServiceInstanceID == 0 {
		return nil, nil
	}
	if err := s.beginServiceInstanceState(task, actor, scope); err != nil {
		return nil, err
	}
	instance, err := s.serviceInstanceForCallback(task, false, scope)
	if err != nil {
		return nil, err
	}
	return &deployServiceStateSnapshot{InstanceID: instance.ID, LifecycleVersion: instance.LifecycleVersion, TransitionID: instance.LastTransitionID}, nil
}
