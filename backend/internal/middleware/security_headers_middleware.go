package middleware

import "github.com/gin-gonic/gin"

const (
	cspHeader  = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'; frame-ancestors 'none'"
	hstsHeader = "max-age=31536000; includeSubDomains"
	permHeader = "camera=(), microphone=(), geolocation=()"
)

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", cspHeader)
		c.Header("Strict-Transport-Security", hstsHeader)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", permHeader)
		c.Next()
	}
}
