package host

import "pantheon-base/pkg/metrics"

// UpdateHostStatusMetrics refreshes the lifecycle and connectivity gauges from
// the current host set. It is cheap to call after list operations.
func UpdateHostStatusMetrics(hosts []Host) {
	projected := make([]metrics.HostStatusMetric, len(hosts))
	for i, h := range hosts {
		projected[i] = metrics.HostStatusMetric{
			LifecycleState:    h.LifecycleState,
			ConnectivityState: h.ConnectivityState,
		}
	}
	metrics.SetHostStatusMetrics(projected)
}

// RecordHostOperation records a CMDB host write operation outcome.
func RecordHostOperation(operation, status string) {
	metrics.RecordHostOperation(operation, status)
}

// RecordHostMetrics sets the resource attribute gauges for a single host.
func RecordHostMetrics(h *Host) {
	if h == nil {
		return
	}
	metrics.SetHostAttributes(h.IP, h.CPUCores, h.MemoryGB, h.DiskGB)
}
