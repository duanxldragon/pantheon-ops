package metrics

// 业务指标辅助函数。本包不依赖任何业务模块，避免 pkg/metrics 与业务包
// 之间的循环引用；业务包各自提供薄包装函数（如 host 包内的
// UpdateHostStatusMetrics、deploy 包内的 recordStepResult）。

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
