package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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
)
