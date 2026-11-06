package middleware

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"

	"github.com/gin-gonic/gin"
)

// legacySelfServicePaths 登记与 /auth/* 同 handler 的 legacy 认证双入口（AUD-014）。
// 命中这些路径的响应带 Deprecation/Sunset 头；端点本身保留至 2026-12-31，
// 下线约束见 docs/contracts/SYSTEM_AUTH_CONTRACT.md §7.7（en §7.6）与
// docs/runbooks/DECOMMISSION_LEGACY_AUTH_ROUTES_2026-12-31.md。
var legacySelfServicePaths = map[string]bool{
	"/api/v1/system/login":     true,
	"/api/v1/system/refresh":   true,
	"/api/v1/system/logout":    true,
	"/api/v1/system/user/info": true,
}

// legacySunsetHTTPDate 是 2026-12-31 下线日的 RFC 1123 HTTP-date 形式。
const legacySunsetHTTPDate = "Sat, 31 Dec 2026 23:59:59 GMT"

// LegacyDeprecationMiddleware 给 legacy 双入口响应加 Deprecation: true 与
// Sunset 头，使外部集成方仅凭响应即可发现下线计划。须挂在鉴权中间件之前，
// 401/403 响应同样携带标记头。
func LegacyDeprecationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if legacySelfServicePaths[c.Request.URL.Path] {
			c.Header("Deprecation", "true")
			c.Header("Sunset", legacySunsetHTTPDate)
		}
		c.Next()
	}
}

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
