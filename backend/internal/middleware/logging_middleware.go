package middleware

import (
	"time"

	"pantheon-ops/backend/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// StructuredLoggingMiddleware 结构化日志中间件
func StructuredLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		// Inject trace ID from OpenTelemetry context for log correlation
		logger := logging.LogFromContext(c.Request.Context())

		if len(c.Errors) > 0 {
			logger.Error("HTTP Request Error",
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.Int("error_count", len(c.Errors)),
			)
			return
		}

		logger.Info("HTTP Request",
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
		)
	}
}
