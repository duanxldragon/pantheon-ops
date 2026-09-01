package deploy

import "pantheon-base/modules/business/metrics"

// recordStepResult records a deploy task step outcome metric.
func recordStepResult(checkpoint *DeployTaskStep) {
	if checkpoint == nil {
		return
	}
	metrics.DeployStepResult(checkpoint.Status)
}
