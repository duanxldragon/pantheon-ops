package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"gorm.io/gorm"
)

const (
	TransitionActionInstall  = "install"
	TransitionActionStart    = "start"
	TransitionActionHealth   = "health"
	TransitionActionStop     = "stop"
	TransitionActionUpgrade  = "upgrade"
	TransitionActionRollback = "rollback"
	TransitionActionRetire   = "retire"

	errStateInvalid = "business.service.state_invalid"
	errStateStale   = "business.service.state_stale"
)

func (m *Manager) TransitionState(ctx context.Context, req InstanceStateTransitionRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error) {
	row, err := m.findInstance(req.InstanceID, scope)
	if err != nil {
		return InstanceRef{}, err
	}
	correlationID := strings.TrimSpace(req.CorrelationID)
	if correlationID != "" && strings.TrimSpace(row.LastTransitionID) == correlationID {
		if sameRequestedProjection(row, req) {
			return m.GetInstance(ctx, row.ID, scope)
		}
		return InstanceRef{}, errors.New(errStateStale)
	}
	if req.ExpectedLifecycleVersion > 0 && row.LifecycleVersion != req.ExpectedLifecycleVersion {
		if sameRequestedProjection(row, req) {
			return m.GetInstance(ctx, row.ID, scope)
		}
		return InstanceRef{}, errors.New(errStateStale)
	}

	next := *row
	applyRequestedState(&next, req)
	action := strings.TrimSpace(req.Action)
	applyRollbackTarget(row, &next, action)
	if err := validateInstanceTransition(row, &next, action); err != nil {
		return InstanceRef{}, err
	}
	now := time.Now()
	updates := map[string]any{
		"desired_state":      next.DesiredState,
		"observed_state":     next.ObservedState,
		"health_state":       next.HealthState,
		"desired_version":    next.DesiredVersion,
		"current_version":    next.CurrentVersion,
		"rollback_version":   next.RollbackVersion,
		"health_message":     next.HealthMessage,
		"health_revision":    next.HealthRevision,
		"last_transition_id": correlationID,
		"last_transition_at": now,
		"updated_by":         actor,
		"updated_at":         now,
		"lifecycle_version":  gorm.Expr("lifecycle_version + 1"),
	}
	if strings.TrimSpace(req.HealthState) != "" {
		updates["health_observed_at"] = now
	}
	query := m.db.WithContext(ctx).Model(&ServiceInstance{}).Where("id = ?", row.ID)
	if req.ExpectedLifecycleVersion > 0 {
		query = query.Where("lifecycle_version = ?", req.ExpectedLifecycleVersion)
	}
	result := query.Updates(updates)
	if result.Error != nil {
		return InstanceRef{}, result.Error
	}
	if result.RowsAffected != 1 {
		return InstanceRef{}, errors.New(errStateStale)
	}
	return m.GetInstance(ctx, row.ID, scope)
}

func applyRollbackTarget(current, next *ServiceInstance, action string) {
	switch action {
	case TransitionActionUpgrade:
		if current.ObservedState != ObservedStateUpgrading && next.ObservedState == ObservedStateUpgrading &&
			strings.TrimSpace(current.CurrentVersion) != "" && current.CurrentVersion != next.DesiredVersion {
			next.RollbackVersion = current.CurrentVersion
		}
	case TransitionActionRollback:
		if current.ObservedState != ObservedStateUpgrading && next.ObservedState == ObservedStateUpgrading &&
			strings.TrimSpace(current.CurrentVersion) != "" && current.CurrentVersion != next.DesiredVersion {
			next.RollbackVersion = current.CurrentVersion
		}
	}
}

func (m *Manager) ReconcileInstanceState(ctx context.Context, id uint64, req ReconcileInstanceRequest, actor string, scope *common.DataScopeReq) (InstanceRef, error) {
	row, err := m.findInstance(id, scope)
	if err != nil {
		return InstanceRef{}, err
	}
	if req.MaxAgeSeconds <= 0 {
		req.MaxAgeSeconds = 15 * 60
	}
	if row.LastTransitionAt == nil || time.Since(*row.LastTransitionAt) < time.Duration(req.MaxAgeSeconds)*time.Second {
		return m.GetInstance(ctx, id, scope)
	}
	action := ""
	switch row.ObservedState {
	case ObservedStateInstalling:
		action = TransitionActionInstall
	case ObservedStateStarting:
		action = TransitionActionStart
	case ObservedStateStopping:
		action = TransitionActionStop
	case ObservedStateUpgrading:
		action = TransitionActionUpgrade
	default:
		return m.GetInstance(ctx, id, scope)
	}
	return m.TransitionState(ctx, InstanceStateTransitionRequest{
		InstanceID:               id,
		Action:                   action,
		ExpectedLifecycleVersion: row.LifecycleVersion,
		DesiredState:             row.DesiredState,
		ObservedState:            ObservedStateFailed,
		HealthState:              HealthStateDegraded,
		HealthMessage:            "transition timed out; reconciliation required",
		CorrelationID:            fmt.Sprintf("reconcile:%d:%d", id, time.Now().UnixNano()),
	}, actor, scope)
}

func applyRequestedState(row *ServiceInstance, req InstanceStateTransitionRequest) {
	if desired := strings.TrimSpace(req.DesiredState); desired != "" {
		row.DesiredState = desired
	}
	if observed := strings.TrimSpace(req.ObservedState); observed != "" {
		row.ObservedState = observed
	}
	if health := strings.TrimSpace(req.HealthState); health != "" {
		row.HealthState = health
	}
	if version := strings.TrimSpace(req.CurrentVersion); version != "" {
		row.CurrentVersion = version
	}
	if version := strings.TrimSpace(req.DesiredVersion); version != "" {
		row.DesiredVersion = version
	}
	if req.HealthMessage != "" {
		row.HealthMessage = req.HealthMessage
	}
	if req.HealthRevision != "" {
		row.HealthRevision = req.HealthRevision
	}
}

func validateInstanceTransition(current, next *ServiceInstance, action string) error {
	if !validDesiredState(next.DesiredState) || !validObservedState(next.ObservedState) || !validHealthState(next.HealthState) {
		return errors.New(errStateInvalid)
	}
	if action == "" {
		return errors.New(errStateInvalid)
	}
	switch action {
	case TransitionActionInstall:
		if !oneOf(current.ObservedState, ObservedStateUnknown, ObservedStateStopped, ObservedStateFailed, ObservedStateInstalling) ||
			!oneOf(next.ObservedState, ObservedStateInstalling, ObservedStateStopped, ObservedStateFailed) {
			return errors.New(errStateInvalid)
		}
		if next.DesiredState == "" {
			next.DesiredState = DesiredStateStopped
		}
	case TransitionActionStart:
		if !oneOf(current.ObservedState, ObservedStateStopped, ObservedStateFailed, ObservedStateStarting) ||
			!oneOf(next.ObservedState, ObservedStateStarting, ObservedStateRunning, ObservedStateFailed) ||
			strings.TrimSpace(next.CurrentVersion) == "" {
			return errors.New(errStateInvalid)
		}
		next.DesiredState = DesiredStateRunning
	case TransitionActionHealth:
		if current.ObservedState != ObservedStateRunning || next.ObservedState != ObservedStateRunning {
			return errors.New(errStateInvalid)
		}
	case TransitionActionStop:
		if !oneOf(current.ObservedState, ObservedStateRunning, ObservedStateFailed, ObservedStateStopping) ||
			!oneOf(next.ObservedState, ObservedStateStopping, ObservedStateStopped, ObservedStateFailed) {
			return errors.New(errStateInvalid)
		}
		next.DesiredState = DesiredStateStopped
	case TransitionActionUpgrade, TransitionActionRollback:
		if !oneOf(current.ObservedState, ObservedStateRunning, ObservedStateStopped, ObservedStateFailed, ObservedStateUpgrading) ||
			!oneOf(next.ObservedState, ObservedStateUpgrading, ObservedStateRunning, ObservedStateFailed) {
			return errors.New(errStateInvalid)
		}
		if strings.TrimSpace(next.DesiredVersion) == "" {
			return errors.New(errStateInvalid)
		}
		if action == TransitionActionRollback && strings.TrimSpace(current.RollbackVersion) != "" &&
			next.DesiredVersion != current.RollbackVersion {
			return errors.New(errStateInvalid)
		}
		if next.ObservedState == ObservedStateRunning && next.HealthState != HealthStateHealthy {
			return errors.New(errStateInvalid)
		}
	case TransitionActionRetire:
		if !oneOf(current.ObservedState, ObservedStateUnknown, ObservedStateStopped, ObservedStateFailed) ||
			next.ObservedState != ObservedStateRetired {
			return errors.New(errStateInvalid)
		}
		next.DesiredState = DesiredStateRetired
		next.HealthState = HealthStateUnknown
	default:
		return errors.New(errStateInvalid)
	}
	return nil
}

func sameRequestedProjection(row *ServiceInstance, req InstanceStateTransitionRequest) bool {
	if req.DesiredState != "" && req.DesiredState != row.DesiredState {
		return false
	}
	if req.ObservedState != "" && req.ObservedState != row.ObservedState {
		return false
	}
	if req.HealthState != "" && req.HealthState != row.HealthState {
		return false
	}
	if req.CurrentVersion != "" && req.CurrentVersion != row.CurrentVersion {
		return false
	}
	if req.DesiredVersion != "" && req.DesiredVersion != row.DesiredVersion {
		return false
	}
	return true
}

func validDesiredState(value string) bool {
	return oneOf(value, DesiredStateStopped, DesiredStateRunning, DesiredStateRetired)
}

func validObservedState(value string) bool {
	return oneOf(value, ObservedStateUnknown, ObservedStateInstalling, ObservedStateStopped, ObservedStateStarting, ObservedStateRunning, ObservedStateStopping, ObservedStateUpgrading, ObservedStateFailed, ObservedStateRetired)
}

func validHealthState(value string) bool {
	return oneOf(value, HealthStateUnknown, HealthStateHealthy, HealthStateDegraded, HealthStateUnhealthy)
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
