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

		latency := time.Since(start)
		requestFields := requestLogFields(c, latency)

		// Inject trace ID from OpenTelemetry context for log correlation
		logger := logging.LogFromContext(c.Request.Context())

		if len(c.Errors) > 0 {
			// 记录错误
			for _, e := range c.Errors.Errors() {
				errorFields := append([]zap.Field{}, requestFields...)
				errorFields = append(errorFields, zap.String("error", logging.SanitizeLogValue(e)))
				logger.Error("HTTP Request Error",
					errorFields...,
				)
			}
		} else {
			// 记录正常请求
			logger.Info("HTTP Request",
				requestFields...,
			)
		}
	}
}

func requestLogFields(c *gin.Context, latency time.Duration) []zap.Field {
	return []zap.Field{
		zap.String("method", logging.SanitizeLogValue(c.Request.Method)),
		zap.String("route", requestRouteLabel(c)),
		zap.Bool("query_present", c.Request.URL.RawQuery != ""),
		zap.Int("status", c.Writer.Status()),
		zap.Duration("latency", latency),
		zap.Bool("client_ip_present", c.ClientIP() != ""),
		zap.Bool("user_agent_present", c.Request.UserAgent() != ""),
	}
}

func requestRouteLabel(c *gin.Context) string {
	route := c.FullPath()
	if route == "" {
		return "unmatched"
	}
	return logging.SanitizeLogValue(route)
}
