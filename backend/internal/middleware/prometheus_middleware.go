package middleware

import (
	"pantheon-ops/backend/pkg/metrics"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// PrometheusMiddleware 记录 HTTP 请求指标
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		path := c.FullPath()
		if path == "" {
			path = "not_found"
		}

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration)
	}
}
