package argocd

import (
	"bytes"
	"time"
)

const (
	appStatusProgressing   = "Progressing"
	syncHistoryRunning     = "Running"
	syncHistoryFailed      = "Failed"
	syncHistorySucceeded   = "Succeeded"
	syncHistoryUnknown     = "Unknown"
	syncObservationTimeout = 10 * time.Minute
)

type applicationOutcome struct {
	status        string
	historyStatus string
	finished      bool
}

// observedApplicationOutcome keeps configuration sync separate from deployment
// readiness: a successful operation is not successful while health is degraded.
func observedApplicationOutcome(status *ArgoCDApplicationStatus) applicationOutcome {
	unknown := applicationOutcome{status: AppStatusUnknown, historyStatus: syncHistoryRunning}
	if status == nil {
		return unknown
	}
	if status.OperationState != nil && (status.OperationState.Phase == "Running" || status.OperationState.Phase == "Terminating") {
		// Health can describe the old rollout while the new operation runs.
		return applicationOutcome{status: appStatusProgressing, historyStatus: syncHistoryRunning}
	}
	if status.Health.Status == HealthDegraded || status.Health.Status == HealthMissing {
		return applicationOutcome{status: AppStatusError, historyStatus: syncHistoryFailed, finished: true}
	}
	if status.OperationState == nil {
		return unknown
	}
	switch status.OperationState.Phase {
	case "Failed", "Error":
		return applicationOutcome{status: AppStatusError, historyStatus: syncHistoryFailed, finished: true}
	case "Succeeded":
		if status.Sync.Status == AppStatusSynced && status.Health.Status == HealthHealthy {
			return applicationOutcome{status: AppStatusSynced, historyStatus: syncHistorySucceeded, finished: true}
		}
		return applicationOutcome{status: appStatusProgressing, historyStatus: syncHistoryRunning}
	case "Running", "Terminating":
		return applicationOutcome{status: appStatusProgressing, historyStatus: syncHistoryRunning}
	default:
		return unknown
	}
}

func historyMatchesObservation(history *ArgoCDSyncHistory, status *ArgoCDApplicationStatus) bool {
	if history == nil || status == nil || status.OperationState == nil || status.OperationState.StartedAt == nil || status.OperationState.StartedAt.Before(history.StartedAt) {
		return false
	}
	if history.Revision == "" {
		return true
	}
	observedRevision := status.OperationState.Operation.Sync.Revision
	if observedRevision == "" {
		observedRevision = status.Sync.Revision
	}
	return observedRevision == history.Revision
}

func sameApplicationSource(current, observed *ArgoCDApplication) bool {
	return current.ID == observed.ID && current.TenantID == observed.TenantID && current.ServerID == observed.ServerID && current.Name == observed.Name && current.Project == observed.Project &&
		current.SourceRepoURL == observed.SourceRepoURL && current.SourcePath == observed.SourcePath && current.SourceTargetRev == observed.SourceTargetRev &&
		current.DestinationServer == observed.DestinationServer && current.DestinationNS == observed.DestinationNS && bytes.Equal(current.SyncPolicy, observed.SyncPolicy)
}
