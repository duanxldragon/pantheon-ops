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
		method := logging.SanitizeLogValue(c.Request.Method)
		path := logging.SanitizeLogValue(c.Request.URL.Path)
		query := logging.SanitizeLogValue(c.Request.URL.RawQuery)
		clientIP := logging.SanitizeLogValue(c.ClientIP())
		userAgent := logging.SanitizeLogValue(c.Request.UserAgent())

		c.Next()

		end := time.Now()
		latency := end.Sub(start)

		// Inject trace ID from OpenTelemetry context for log correlation
		logger := logging.LogFromContext(c.Request.Context())

		if len(c.Errors) > 0 {
			// 记录错误
			for _, e := range c.Errors.Errors() {
				logger.Error("HTTP Request Error",
					zap.String("method", method),
					zap.String("path", path),
					zap.String("query", query),
					zap.Int("status", c.Writer.Status()),
					zap.Duration("latency", latency),
					zap.String("ip", clientIP),
					zap.String("user_agent", userAgent),
					zap.String("error", logging.SanitizeLogValue(e)),
				)
			}
		} else {
			// 记录正常请求
			logger.Info("HTTP Request",
				zap.String("method", method),
				zap.String("path", path),
				zap.String("query", query),
				zap.Int("status", c.Writer.Status()),
				zap.Duration("latency", latency),
				zap.String("ip", clientIP),
				zap.String("user_agent", userAgent),
			)
		}
	}
}
