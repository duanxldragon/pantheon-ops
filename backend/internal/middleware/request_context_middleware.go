package middleware

import (
	"context"
	"strings"
	"time"
	"unicode"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/logging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxRequestIDLength = 64

// RequestContextMiddleware injects request and trace IDs into the request context.
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := normalizeIncomingRequestID(
			c.GetHeader(common.HeaderRequestID),
			c.GetHeader(common.HeaderTraceID),
		)
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
