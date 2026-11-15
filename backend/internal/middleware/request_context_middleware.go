package middleware

import (
	"context"
	"strings"
	"time"
	"unicode"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const maxRequestIDLength = 64

// MetadataAwareLogger preserves Gin's legacy logger on unmatched routes, while
// opted routes log only registered action, status and the server correlation ID.
func MetadataAwareLogger() gin.HandlerFunc {
	legacy := gin.Logger()
	return func(c *gin.Context) {
		if route, protected := auditpolicy.Match(c); protected {
			c.Next()
			logging.LogFromContext(c.Request.Context()).Info("business request",
				zap.String("method", logging.SanitizeLogValue(route.Method)), zap.String("route", logging.SanitizeLogValue(route.Template)), zap.Int("status", c.Writer.Status()))
			return
		}
		legacy(c)
	}
}

// MetadataAwareRecovery avoids Gin's raw request/error panic dump for the
// metadata strategy. The legacy recovery behavior stays unchanged elsewhere.
func MetadataAwareRecovery() gin.HandlerFunc {
	legacy := gin.Recovery()
	return func(c *gin.Context) {
		if _, protected := auditpolicy.Match(c); protected {
			defer func() {
				if recover() != nil {
					logging.LogFromContext(c.Request.Context()).Error("business request panic", zap.String("outcome", "unknown"))
					auditpolicy.SetOutcome(c, auditpolicy.OutcomeUnknown)
					c.AbortWithStatus(500)
				}
			}()
			c.Next()
			return
		}
		legacy(c)
	}
}

// RequestContextMiddleware injects request and trace IDs into the request context.
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := ""
		if _, protected := auditpolicy.Match(c); !protected {
			requestID = normalizeIncomingRequestID(c.GetHeader(common.HeaderRequestID), c.GetHeader(common.HeaderTraceID))
		}
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(common.ContextKeyRequestID, requestID)
		c.Set(common.ContextKeyTraceID, requestID)
		c.Header(common.HeaderRequestID, requestID)
		c.Header(common.HeaderTraceID, requestID)

		ctx := logging.WithRequestID(c.Request.Context(), requestID)
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

func normalizeIncomingRequestID(values ...string) string {
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		if len(trimmed) > maxRequestIDLength {
			trimmed = trimmed[:maxRequestIDLength]
		}
		if isSafeRequestID(trimmed) {
			return trimmed
		}
	}
	return ""
}

func isSafeRequestID(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
		case r == '-', r == '_', r == '.', r == ':':
		default:
			return false
		}
	}
	return true
}
