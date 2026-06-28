package http

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	HeaderRequestID     = "X-Request-ID"
	HeaderTraceID       = "X-Trace-ID"
	ContextKeyRequestID = "requestId"
	ContextKeyTraceID   = "traceId"
)

func GetRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get(ContextKeyRequestID); ok {
		if requestID, ok := value.(string); ok {
			return strings.TrimSpace(requestID)
		}
	}
	return strings.TrimSpace(c.GetHeader(HeaderRequestID))
}
