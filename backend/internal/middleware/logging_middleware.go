package middleware

import (
	"time"

	"pantheon-base/pkg/logging"

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

		// Inject request-scoped context into the structured log sink.
		logger := logging.LogFromContext(c.Request.Context())
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			logger.Error("HTTP Request Error", append(fields, zap.Int("error_count", len(c.Errors)))...)
		} else {
			logger.Info("HTTP Request", fields...)
		}
	}
}
