package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultMaxBodyBytes is the default maximum request body size (10 MB).
	DefaultMaxBodyBytes int64 = 10 << 20 // 10 MB
)

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
				"message": "request body too large",
			})
			c.Abort()
			return
		}
		// Limit the actual read as well
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}
