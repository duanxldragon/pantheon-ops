package middleware

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxBodyBytes is the default maximum request body size (10 MB).
	DefaultMaxBodyBytes int64 = 10 << 20 // 10 MB

	// MaxBodyBytesEnv overrides the default request body size limit (bytes).
	// Large offline deployment packages (JDK/Prometheus tarballs) routinely
	// exceed 10 MB, so deployments must be able to raise the limit.
	MaxBodyBytesEnv = "PANTHEON_MAX_BODY_BYTES"
)

// ResolveMaxBodyBytes returns the configured request body limit: the
// PANTHEON_MAX_BODY_BYTES env override when set, otherwise the 10 MB default.
func ResolveMaxBodyBytes() int64 {
	raw := strings.TrimSpace(os.Getenv(MaxBodyBytesEnv))
	if raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return DefaultMaxBodyBytes
}

// BodySizeLimit returns a Gin middleware that limits the size of request bodies.
// If the Content-Length header exceeds maxBytes or the body read exceeds maxBytes,
// the request is rejected with 413 Request Entity Too Large.
func BodySizeLimit(maxBytes int64) gin.HandlerFunc {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodyBytes
	}

	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"code":    413,
				"message": "request.body_too_large",
			})
			c.Abort()
			return
		}
		// Limit the actual read as well
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
