package middleware

import "github.com/gin-gonic/gin"

const (
	hstsHeader = "max-age=31536000; includeSubDomains"
	permHeader = "camera=(), microphone=(), geolocation=()"
)

// SecurityHeadersMiddleware 设置通用安全响应头。
// 注意：Content-Security-Policy 由 CSPMiddleware 统一负责（单一来源），此处不再设置。
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Strict-Transport-Security", hstsHeader)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", permHeader)
		c.Next()
	}
}
