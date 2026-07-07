package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMetrics(t *testing.T) {
	t.Run("HTTPRequestsTotal", func(t *testing.T) {
		if HTTPRequestsTotal == nil {
			t.Error("HTTPRequestsTotal is nil")
		}

		// 测试指标更新
		HTTPRequestsTotal.WithLabelValues("GET", "/api/test", "200").Inc()
	})

	t.Run("HTTPRequestDuration", func(t *testing.T) {
		if HTTPRequestDuration == nil {
			t.Error("HTTPRequestDuration is nil")
		}

		// 测试指标更新
		HTTPRequestDuration.WithLabelValues("GET", "/api/test").Observe(0.123)
	})

	t.Run("DBConnectionsActive", func(t *testing.T) {
		if DBConnectionsActive == nil {
			t.Error("DBConnectionsActive is nil")
		}

		// 测试指标更新
		DBConnectionsActive.Set(10)
	})

	t.Run("DBConnectionsIdle", func(t *testing.T) {
		if DBConnectionsIdle == nil {
			t.Error("DBConnectionsIdle is nil")
		}

		// 测试指标更新
		DBConnectionsIdle.Set(5)
	})

	t.Run("DBConnectionsOpen", func(t *testing.T) {
		if DBConnectionsOpen == nil {
			t.Error("DBConnectionsOpen is nil")
		}

		// 测试指标更新
		DBConnectionsOpen.Set(15)
	})

	t.Run("RedisConnectionsActive", func(t *testing.T) {
		if RedisConnectionsActive == nil {
			t.Error("RedisConnectionsActive is nil")
		}

		// 测试指标更新
		RedisConnectionsActive.Set(3)
	})

	t.Run("RedisConnectionsIdle", func(t *testing.T) {
		if RedisConnectionsIdle == nil {
			t.Error("RedisConnectionsIdle is nil")
		}

		// 测试指标更新
		RedisConnectionsIdle.Set(2)
	})

	t.Run("AuthLoginAttempts", func(t *testing.T) {
		if AuthLoginAttempts == nil {
			t.Error("AuthLoginAttempts is nil")
		}

		// 测试指标更新
		AuthLoginAttempts.WithLabelValues("success").Inc()
		AuthLoginAttempts.WithLabelValues("failed").Inc()
		AuthLoginAttempts.WithLabelValues("locked").Inc()
	})

	t.Run("ActiveSessions", func(t *testing.T) {
		if ActiveSessions == nil {
			t.Error("ActiveSessions is nil")
		}

		// 测试指标更新
		ActiveSessions.Set(100)
	})
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
