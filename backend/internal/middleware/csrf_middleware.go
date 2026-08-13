package middleware

import (
	"pantheon-base/pkg/common"
	commonhttp "pantheon-base/pkg/common/http"

	"github.com/gin-gonic/gin"
)

var safeMethods = map[string]bool{
	"GET":     true,
	"HEAD":    true,
	"OPTIONS": true,
}

// csrfExemptPaths 使用精确路径匹配（非前缀匹配），
// 防止未来在这些前缀下新增的 POST 路由被静默豁免 CSRF 校验。
var csrfExemptPaths = map[string]bool{
	"/api/v1/auth/login":      true,
	"/api/v1/auth/refresh":    true,
	"/api/v1/auth/mfa/verify": true,
	// legacy 双入口（与 /auth/* 同 handler），保留至旧路由下线。
	"/api/v1/system/login":   true,
	"/api/v1/system/refresh": true,
}

func isCSRFExempt(path string) bool {
	return csrfExemptPaths[path]
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if safeMethods[c.Request.Method] || isCSRFExempt(c.Request.URL.Path) {
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie(commonhttp.CookieCSRFToken)
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
