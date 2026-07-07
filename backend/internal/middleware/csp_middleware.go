package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSPMiddleware 添加 Content-Security-Policy 响应头
func CSPMiddleware() gin.HandlerFunc {
	cspPolicy := buildCSPPolicy()

	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", cspPolicy)
		c.Next()
	}
}

func buildCSPPolicy() string {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("PANTHEON_ENV")))

	// 基础策略
	directives := []string{
		"default-src 'self'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com data:",
		"img-src 'self' data: https: blob:",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}

	// 开发环境：允许 unsafe-eval（Vite HMR 需要）
	if env == "development" || env == "" {
		directives = append(directives, "script-src 'self' 'unsafe-inline' 'unsafe-eval'")
	} else {
		// 生产环境：移除 unsafe-eval，仅保留 unsafe-inline（React 需要）
		directives = append(directives, "script-src 'self' 'unsafe-inline'")
	}

	// CSP 报告端点
	reportURI := os.Getenv("CSP_REPORT_URI")
	if reportURI != "" {
		directives = append(directives, "report-uri "+reportURI)
	}

	return strings.Join(directives, "; ")
}
