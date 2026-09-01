package cluster

import (
	"time"

	"pantheon-base/modules/business/metrics"
)

// k8sClusterSyncErrors is a shorthand for the cluster sync error counter.
var (
	k8sClusterSyncErrors   = metrics.ClusterSyncErrors
	k8sClusterSyncDuration = metrics.ClusterSyncDuration
)

// UpdateClusterStatusMetrics refreshes the cluster status and capacity gauges
// from the current cluster set. It is cheap to call after list operations.
func UpdateClusterStatusMetrics(clusters []Cluster) {
	status := make(map[string]int)
	for i := range clusters {
		status[metrics.NormalizeLabel(clusters[i].Status)]++
		metrics.SetClusterCapacity(clusters[i].ID, clusters[i].TotalNodes, clusters[i].ReadyNodes, clusters[i].CPUCapacity, clusters[i].MemoryCapacity)
	}
	for state, count := range status {
		metrics.SetClusterStatus(state, float64(count))
	}
}

// RecordClusterMetrics sets the resource gauges for a single cluster.
func RecordClusterMetrics(c *Cluster) {
	if c == nil {
		return
	}
	metrics.SetClusterCapacity(c.ID, c.TotalNodes, c.ReadyNodes, c.CPUCapacity, c.MemoryCapacity)
	metrics.SetClusterStatus(metrics.NormalizeLabel(c.Status), 1)
}

var _ = time.Now
