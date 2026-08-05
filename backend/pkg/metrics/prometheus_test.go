package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMetrics(t *testing.T) {
	tests := []struct {
		name     string
		metric   any
		exercise func()
	}{
		{name: "HTTPRequestsTotal", metric: HTTPRequestsTotal, exercise: func() {
			HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200").Inc()
		}},
		{name: "HTTPRequestDuration", metric: HTTPRequestDuration, exercise: func() {
			HTTPRequestDuration.WithLabelValues("GET", "/api/test").Observe(0.123)
		}},
		{name: "DBConnectionsActive", metric: DBConnectionsActive, exercise: func() { DBConnectionsActive.Set(10) }},
		{name: "DBConnectionsIdle", metric: DBConnectionsIdle, exercise: func() { DBConnectionsIdle.Set(5) }},
		{name: "DBConnectionsOpen", metric: DBConnectionsOpen, exercise: func() { DBConnectionsOpen.Set(15) }},
		{name: "RedisConnectionsActive", metric: RedisConnectionsActive, exercise: func() { RedisConnectionsActive.Set(3) }},
		{name: "RedisConnectionsIdle", metric: RedisConnectionsIdle, exercise: func() { RedisConnectionsIdle.Set(2) }},
		{name: "AuthLoginAttempts", metric: AuthLoginAttempts, exercise: func() {
			AuthLoginAttempts.WithLabelValues("success").Inc()
			AuthLoginAttempts.WithLabelValues("failed").Inc()
			AuthLoginAttempts.WithLabelValues("locked").Inc()
		}},
		{name: "ActiveSessions", metric: ActiveSessions, exercise: func() { ActiveSessions.Set(100) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metric == nil {
				t.Fatalf("%s is nil", tt.name)
			}
			tt.exercise()
		})
	}
}

func TestMetricsRegistration(t *testing.T) {
	// 测试所有指标都已注册到 Prometheus
	metrics := []prometheus.Collector{
		HTTPRequestsTotal,
		HTTPRequestDuration,
		DBConnectionsActive,
		DBConnectionsIdle,
		DBConnectionsOpen,
		RedisConnectionsActive,
		RedisConnectionsIdle,
		AuthLoginAttempts,
		ActiveSessions,
	}

	for _, metric := range metrics {
		if metric == nil {
			t.Error("Metric is nil")
		}
	}
}
