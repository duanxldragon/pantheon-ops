// Package metrics holds the Ops business overlay Prometheus metric helpers.
// It was migrated out of the base-owned pkg/metrics package (which must stay
// platform-only): CMDB host, deploy task, and Kubernetes cluster gauges and
// counters are business semantics and belong in the business overlay.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HostOperationsTotal 主机（CMDB）写操作计数。
	HostOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_host_operations_total",
			Help: "Total number of CMDB host operations",
		},
		[]string{"operation", "status"},
	)

	// HostsTotal 主机生命周期状态分布。
	HostsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pantheon_hosts_total",
			Help: "Number of CMDB hosts by lifecycle state",
		},
		[]string{"state"},
	)

	// HostConnectivity 主机连通性状态分布。
	HostConnectivity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pantheon_host_connectivity",
			Help: "Number of CMDB hosts by connectivity state",
		},
		[]string{"connectivity"},
	)

	// HostAttributes 主机资源量（按 IP 区分，避免高基数标签）。
	HostAttributes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pantheon_host_attributes",
			Help: "CMDB host resource attributes (cpu cores, memory GB, disk GB)",
		},
		[]string{"ip", "attribute"},
	)

	// DeployTaskStepsTotal 部署步骤执行结果分布。
	DeployTaskStepsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_deploy_task_steps_total",
			Help: "Total number of deploy task step executions",
		},
		[]string{"status"},
	)

	// DeployTasksTotal 部署任务终态分布。
	DeployTasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_deploy_tasks_total",
			Help: "Total number of deploy tasks reaching a terminal state",
		},
		[]string{"status"},
	)

	// DeployTaskHostsTotal 部署任务主机结果分布。
	DeployTaskHostsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_deploy_task_hosts_total",
			Help: "Total number of deploy task host results",
		},
		[]string{"status"},
	)

	// ClusterStatusTotal 集群状态分布。
	ClusterStatusTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pantheon_k8s_cluster_status",
			Help: "Number of Kubernetes clusters by status",
		},
		[]string{"status"},
	)

	// ClusterCapacity 集群容量指标。
	ClusterCapacity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "pantheon_k8s_cluster_capacity",
			Help: "Kubernetes cluster node and capacity metrics",
		},
		[]string{"metric"},
	)

	// ClusterSyncErrors 集群同步失败计数。
	ClusterSyncErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_k8s_cluster_sync_errors_total",
			Help: "Kubernetes cluster sync failures by cluster and phase",
		},
		[]string{"cluster", "phase"},
	)

	// ClusterSyncDuration 集群同步耗时分布。
	ClusterSyncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pantheon_k8s_cluster_sync_duration_seconds",
			Help:    "Kubernetes cluster sync duration in seconds",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"cluster", "result"},
	)
)

// RecordHostOperation records a CMDB host write operation outcome.
func RecordHostOperation(operation, status string) {
	HostOperationsTotal.WithLabelValues(normalizeLabel(operation), normalizeLabel(status)).Inc()
}

// SetHostAttributes sets the resource attribute gauges for a single host.
func SetHostAttributes(ip string, cpuCores int, memoryGB, diskGB float64) {
	HostAttributes.WithLabelValues(ip, "cpu_cores").Set(float64(cpuCores))
	HostAttributes.WithLabelValues(ip, "memory_gb").Set(memoryGB)
	HostAttributes.WithLabelValues(ip, "disk_gb").Set(diskGB)
}

// SetHostStatusMetrics refreshes the lifecycle and connectivity gauges from
// the current host set. It is cheap to call after list operations.
func SetHostStatusMetrics(hosts []HostStatusMetric) {
	lifecycle := make(map[string]int)
	connectivity := make(map[string]int)
	for _, h := range hosts {
		lifecycle[normalizeLabel(h.LifecycleState)]++
		connectivity[normalizeLabel(h.ConnectivityState)]++
	}
	for state, count := range lifecycle {
		HostsTotal.WithLabelValues(state).Set(float64(count))
	}
	for state, count := range connectivity {
		HostConnectivity.WithLabelValues(state).Set(float64(count))
	}
}

// HostStatusMetric is the minimal host projection consumed by metrics.
type HostStatusMetric struct {
	LifecycleState    string
	ConnectivityState string
}

// DeployStepResult records a deploy task step outcome.
func DeployStepResult(status string) {
	DeployTaskStepsTotal.WithLabelValues(normalizeLabel(status)).Inc()
}

// DeployTaskTerminal records a deploy task reaching a terminal state.
func DeployTaskTerminal(status string) {
	DeployTasksTotal.WithLabelValues(normalizeLabel(status)).Inc()
}

// DeployTaskStarted records a deploy task start.
func DeployTaskStarted(executorType string) {
	DeployTasksTotal.WithLabelValues("started").Inc()
}

// DeployTaskCompleted records a deploy task completion.
func DeployTaskCompleted(executorType string) {
	DeployTasksTotal.WithLabelValues("completed").Inc()
}

// DeployTaskFailed records a deploy task failure.
func DeployTaskFailed(executorType string) {
	DeployTasksTotal.WithLabelValues("failed").Inc()
}

// DeployHostResult records a host-level deploy result.
func DeployHostResult(executorType, status string) {
	DeployTaskHostsTotal.WithLabelValues(normalizeLabel(status)).Inc()
}

// SetClusterStatus records the cluster status distribution gauge.
func SetClusterStatus(status string, count float64) {
	ClusterStatusTotal.WithLabelValues(normalizeLabel(status)).Set(count)
}

// SetClusterCapacity records the node and capacity gauges for one cluster.
func SetClusterCapacity(clusterID uint64, totalNodes, readyNodes int, cpuCapacity, memoryCapacity float64) {
	ClusterCapacity.WithLabelValues("total_nodes").Set(float64(totalNodes))
	ClusterCapacity.WithLabelValues("ready_nodes").Set(float64(readyNodes))
	ClusterCapacity.WithLabelValues("cpu_capacity").Set(cpuCapacity)
	ClusterCapacity.WithLabelValues("memory_capacity_gb").Set(memoryCapacity)
}

// NormalizeLabel normalizes an empty label value.
func NormalizeLabel(s string) string {
	return normalizeLabel(s)
}

func normalizeLabel(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
