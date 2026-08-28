package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Namespace is the Prometheus namespace for all pantheon metrics
const Namespace = "pantheon"

var (
	// HTTP 请求总数
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	// HTTP 请求延迟
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "pantheon_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets, // [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
		},
		[]string{"method", "path"},
	)

	// 数据库连接池
	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_db_connections_open",
			Help: "Number of open database connections",
		},
	)

	// Redis 连接
	RedisConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_redis_connections_active",
			Help: "Number of active Redis connections",
		},
	)

	RedisConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_redis_connections_idle",
			Help: "Number of idle Redis connections",
		},
	)

	// 登录指标
	AuthLoginAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "pantheon_auth_login_attempts_total",
			Help: "Total number of login attempts",
		},
		[]string{"status"}, // success, failed, locked
	)

	// 活跃会话数
	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_active_sessions",
			Help: "Number of active user sessions",
		},
	)

	// OperationLogDroppedTotal 操作日志异步队列满时丢弃的日志条数。
	OperationLogDroppedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "pantheon_operation_log_dropped_total",
			Help: "Operation log entries dropped because the async queue was full",
		},
	)

	// OperationLogQueueDepth 操作日志异步队列当前深度。
	OperationLogQueueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pantheon_operation_log_queue_depth",
			Help: "Current depth of the operation log async queue",
		},
	)

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
