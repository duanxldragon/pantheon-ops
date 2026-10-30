package host

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"gorm.io/gorm"
)

const (
	hostStatusOnline  = "online"
	hostStatusOffline = "offline"
)

// Host lifecycle, connectivity, and transition constants define the CMDB host state machine.
const (
	HostLifecyclePending     = "pending"
	HostLifecycleAssigned    = "assigned"
	HostLifecycleMaintenance = "maintenance"
	HostLifecycleRetired     = "retired"

	HostConnectivityUnknown     = "unknown"
	HostConnectivityReachable   = "reachable"
	HostConnectivityUnreachable = "unreachable"

	HostTransitionAssign       = "assign"
	HostTransitionMaintenance  = "maintenance"
	HostTransitionRetire       = "retire"
	HostTransitionConnectivity = "connectivity"

	errHostStateInvalid = "cmdbhost.state.invalid"
	errHostStateStale   = "cmdbhost.state.stale"
)

// TransitionState applies one validated host lifecycle or connectivity transition.
func (s *HostService) TransitionState(id uint64, req HostStateTransitionRequest, actor string, dataScope *common.DataScopeReq) (*HostResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var row Host
	if err := s.hostQuery(dataScope).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("cmdbhost.not_found")
		}
		return nil, err
	}
	normalizeHostState(&row)
	if req.ExpectedStateVersion > 0 && row.StateVersion != req.ExpectedStateVersion {
		return nil, errors.New(errHostStateStale)
	}

	nextLifecycle, nextConnectivity, observedAt, err := s.resolveHostStateTarget(&row, req, dataScope)
	if err != nil {
		return nil, err
	}

	nextStatus := hostStatusFromState(nextLifecycle, nextConnectivity)
	updates := hostStateTransitionUpdates(nextLifecycle, nextConnectivity, observedAt, nextStatus, actor)
	query := s.hostQuery(dataScope).Where(idWhereClause, id)
	if req.ExpectedStateVersion > 0 {
		query = query.Where("state_version = ?", req.ExpectedStateVersion)
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, errors.New(errHostStateStale)
	}
	return s.GetByID(id, dataScope)
}

func hostStateTransitionUpdates(lifecycle, connectivity string, observedAt *time.Time, status, actor string) map[string]any {
	return map[string]any{
		"lifecycle_state":          lifecycle,
		"connectivity_state":       connectivity,
		"connectivity_observed_at": observedAt,
		"status":                   status,
		"state_version":            gorm.Expr("state_version + 1"),
		"updated_by":               actor,
		"updated_at":               time.Now(),
	}
}

func (s *HostService) resolveHostStateTarget(row *Host, req HostStateTransitionRequest, dataScope *common.DataScopeReq) (string, string, *time.Time, error) {
	if strings.TrimSpace(req.Action) == HostTransitionConnectivity {
		return s.resolveHostConnectivityTarget(row, req)
	}
	lifecycle, err := s.resolveHostLifecycleTarget(row, strings.TrimSpace(req.Action), dataScope)
	if err != nil {
		return "", "", nil, err
	}
	return lifecycle, row.ConnectivityState, row.ConnectivityObservedAt, nil
}

func (s *HostService) resolveHostLifecycleTarget(row *Host, action string, dataScope *common.DataScopeReq) (string, error) {
	switch action {
	case HostTransitionAssign:
		if row.LifecycleState != HostLifecyclePending {
			return "", errors.New(errHostStateInvalid)
		}
		return HostLifecycleAssigned, nil
	case HostTransitionMaintenance:
		if row.LifecycleState != HostLifecycleAssigned {
			return "", errors.New(errHostStateInvalid)
		}
		return HostLifecycleMaintenance, nil
	case HostTransitionRetire:
		if err := s.ensureHostRetirable(row, dataScope); err != nil {
			return "", err
		}
		return HostLifecycleRetired, nil
	default:
		return "", errors.New(errHostStateInvalid)
	}
}

func (s *HostService) resolveHostConnectivityTarget(row *Host, req HostStateTransitionRequest) (string, string, *time.Time, error) {
	if !validHostConnectivity(req.ConnectivityState) {
		return "", "", nil, errors.New(errHostStateInvalid)
	}
	observedAt, err := parseHostConnectivityObservedAt(req)
	if err != nil {
		return "", "", nil, err
	}
	return row.LifecycleState, req.ConnectivityState, observedAt, nil
}

func (s *HostService) ensureHostRetirable(row *Host, dataScope *common.DataScopeReq) error {
	if !oneOfHostState(row.LifecycleState, HostLifecyclePending, HostLifecycleMaintenance) {
		return errors.New(errHostStateInvalid)
	}
	if s.serviceRefs == nil {
		return nil
	}
	active, err := s.serviceRefs.HasActiveHostReferences(context.Background(), row.ID, dataScope)
	if err != nil {
		return err
	}
	if active {
		return errors.New("cmdbhost.active_service_reference")
	}
	return nil
}

func parseHostConnectivityObservedAt(req HostStateTransitionRequest) (*time.Time, error) {
	if req.ObservedAt == nil {
		now := time.Now()
		return &now, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ObservedAt))
	if err != nil {
		return nil, errors.New(errHostStateInvalid)
	}
	return &parsed, nil
}

func normalizeHostState(row *Host) {
	if row.LifecycleState == "" {
		switch row.Status {
		case HostLifecycleAssigned, hostStatusOnline, hostStatusOffline:
			row.LifecycleState = HostLifecycleAssigned
		case HostLifecycleMaintenance:
			row.LifecycleState = HostLifecycleMaintenance
		case HostLifecycleRetired:
			row.LifecycleState = HostLifecycleRetired
		default:
			row.LifecycleState = HostLifecyclePending
		}
	}
	if row.ConnectivityState == "" {
		switch row.Status {
		case hostStatusOnline:
			row.ConnectivityState = HostConnectivityReachable
		case hostStatusOffline:
			row.ConnectivityState = HostConnectivityUnreachable
		default:
			row.ConnectivityState = HostConnectivityUnknown
		}
	}
}

func hostStatusFromState(lifecycle, connectivity string) string {
	switch lifecycle {
	case HostLifecycleMaintenance:
		return HostLifecycleMaintenance
	case HostLifecycleRetired:
		return HostLifecycleRetired
	case HostLifecycleAssigned:
		switch connectivity {
		case HostConnectivityReachable:
			return hostStatusOnline
		case HostConnectivityUnreachable:
			return hostStatusOffline
		default:
			return HostLifecycleAssigned
		}
	default:
		return HostLifecyclePending
	}
}

func validHostConnectivity(value string) bool {
	return oneOfHostState(value, HostConnectivityUnknown, HostConnectivityReachable, HostConnectivityUnreachable)
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func oneOfHostState(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
