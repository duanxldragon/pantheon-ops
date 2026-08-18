package host

import (
	"context"
	"errors"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"gorm.io/gorm"
)

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

	nextLifecycle := row.LifecycleState
	nextConnectivity := row.ConnectivityState
	observedAt := row.ConnectivityObservedAt
	switch strings.TrimSpace(req.Action) {
	case HostTransitionAssign:
		if row.LifecycleState != HostLifecyclePending {
			return nil, errors.New(errHostStateInvalid)
		}
		nextLifecycle = HostLifecycleAssigned
	case HostTransitionMaintenance:
		if row.LifecycleState != HostLifecycleAssigned {
			return nil, errors.New(errHostStateInvalid)
		}
		nextLifecycle = HostLifecycleMaintenance
	case HostTransitionRetire:
		if !oneOfHostState(row.LifecycleState, HostLifecyclePending, HostLifecycleMaintenance) {
			return nil, errors.New(errHostStateInvalid)
		}
		if s.serviceRefs != nil {
			active, err := s.serviceRefs.HasActiveHostReferences(context.Background(), row.ID, dataScope)
			if err != nil {
				return nil, err
			}
			if active {
				return nil, errors.New("cmdbhost.active_service_reference")
			}
		}
		nextLifecycle = HostLifecycleRetired
	case HostTransitionConnectivity:
		if !validHostConnectivity(req.ConnectivityState) {
			return nil, errors.New(errHostStateInvalid)
		}
		nextConnectivity = req.ConnectivityState
		if req.ObservedAt != nil {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.ObservedAt))
			if err != nil {
				return nil, errors.New(errHostStateInvalid)
			}
			observedAt = &parsed
		} else {
			now := time.Now()
			observedAt = &now
		}
	default:
		return nil, errors.New(errHostStateInvalid)
	}

	nextStatus := hostStatusFromState(nextLifecycle, nextConnectivity)
	updates := map[string]any{
		"lifecycle_state":          nextLifecycle,
		"connectivity_state":       nextConnectivity,
		"connectivity_observed_at": observedAt,
		"status":                   nextStatus,
		"state_version":            gorm.Expr("state_version + 1"),
		"updated_by":               actor,
		"updated_at":               time.Now(),
	}
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

func normalizeHostState(row *Host) {
	if row.LifecycleState == "" {
		switch row.Status {
		case "assigned", "online", "offline":
			row.LifecycleState = HostLifecycleAssigned
		case "maintenance":
			row.LifecycleState = HostLifecycleMaintenance
		case "retired":
			row.LifecycleState = HostLifecycleRetired
		default:
			row.LifecycleState = HostLifecyclePending
		}
	}
	if row.ConnectivityState == "" {
		switch row.Status {
		case "online":
			row.ConnectivityState = HostConnectivityReachable
		case "offline":
			row.ConnectivityState = HostConnectivityUnreachable
		default:
			row.ConnectivityState = HostConnectivityUnknown
		}
	}
}

func hostStatusFromState(lifecycle, connectivity string) string {
	switch lifecycle {
	case HostLifecycleMaintenance:
		return "maintenance"
	case HostLifecycleRetired:
		return "retired"
	case HostLifecycleAssigned:
		switch connectivity {
		case HostConnectivityReachable:
			return "online"
		case HostConnectivityUnreachable:
			return "offline"
		default:
			return "assigned"
		}
	default:
		return "pending"
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
