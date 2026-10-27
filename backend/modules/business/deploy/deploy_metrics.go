package deploy

import "github.com/duanxldragon/pantheon-base/backend/modules/business/metrics"

// recordStepResult records a deploy task step outcome metric.
func recordStepResult(checkpoint *DeployTaskStep) {
	if checkpoint == nil {
		return
	}
	metrics.DeployStepResult(checkpoint.Status)
}
