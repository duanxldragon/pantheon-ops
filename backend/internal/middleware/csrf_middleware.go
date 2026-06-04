package middleware

import (
	"strings"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

var safeMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
}

var csrfExemptPaths = []string{
	"/api/v1/auth/login",
	"/api/v1/auth/refresh",
	"/api/v1/auth/mfa/verify",
}

func isCSRFExempt(path string) bool {
	for _, prefix := range csrfExemptPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethods[c.Request.Method] || isCSRFExempt(c.Request.URL.Path) {
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie(common.CookieCSRFToken)
		if err != nil || csrfCookie == "" {
			common.Fail(c, common.CodeForbidden, "csrf.missing")
			c.Abort()
			return
		}

		csrfHeader := c.GetHeader("X-CSRF-Token")
		if csrfHeader == "" {
			common.Fail(c, common.CodeForbidden, "csrf.missing")
			c.Abort()
			return
		}

		if csrfCookie != csrfHeader {
			common.Fail(c, common.CodeForbidden, "csrf.mismatch")
			c.Abort()
			return
		}

		c.Next()
	}
}
