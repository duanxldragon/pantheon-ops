package deploy

import "pantheon-base/pkg/metrics"

// recordStepResult records a deploy task step outcome metric.
func recordStepResult(checkpoint *DeployTaskStep) {
	if checkpoint == nil {
		return
	}
	metrics.DeployStepResult(checkpoint.Status)
}
