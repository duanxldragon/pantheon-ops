package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSPMiddleware 添加 Content-Security-Policy 响应头
func CSPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := GetCSPNonce(c)
		cspPolicy := buildCSPPolicy(nonce)
		c.Header("Content-Security-Policy", cspPolicy)
		c.Next()
	}
}

func buildCSPPolicy(nonce string) string {
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

	// 构建 script-src，使用 nonce 替代 unsafe-inline（生产环境）
	scriptSrc := "script-src 'self'"
	if nonce != "" {
		scriptSrc += " 'nonce-" + nonce + "'"
	}
	// 开发环境：允许 unsafe-eval（Vite HMR 需要）和 unsafe-inline（兼容性）
	if env == "development" || env == "" {
		scriptSrc += " 'unsafe-inline' 'unsafe-eval'"
	}
	// 生产环境：如果没有 nonce，保留 unsafe-inline 作为降级（不应该发生）
	if (env != "development" && env != "") && nonce == "" {
		scriptSrc += " 'unsafe-inline'"
	}
	directives = append(directives, scriptSrc)

	// CSP 报告端点
	reportURI := os.Getenv("CSP_REPORT_URI")
	if reportURI != "" {
		directives = append(directives, "report-uri "+reportURI)
	}

	return strings.Join(directives, "; ")
}
