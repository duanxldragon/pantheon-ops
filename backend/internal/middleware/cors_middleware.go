package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultAllowedOrigins = []string{
	"http://127.0.0.1:5173",
	"http://localhost:5173",
	"http://127.0.0.1:5174",
	"http://localhost:5174",
}

func loadAllowedOrigins() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("PANTHEON_ALLOWED_ORIGINS"))
	items := defaultAllowedOrigins
	if raw != "" {
		items = strings.Split(raw, ",")
	}

	allowed := make(map[string]struct{}, len(items))
	for _, item := range items {
		origin := strings.TrimSpace(item)
		if origin == "" {
			continue
		}
		allowed[origin] = struct{}{}
	}
	return allowed
}

func appendVary(existing, value string) string {
	if existing == "" {
		return value
	}
	parts := strings.Split(existing, ",")
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return existing
		}
	}
	return existing + ", " + value
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))
		if origin != "" {
			if _, ok := loadAllowedOrigins()[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Set("Vary", appendVary(c.Writer.Header().Get("Vary"), "Origin"))
			}
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token, X-Operation-Token, Accept-Language")
		c.Header("Access-Control-Expose-Headers", "X-Request-Id, X-CSRF-Token")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
