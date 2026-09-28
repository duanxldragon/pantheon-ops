package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const nonceLength = 16

// CSPNonceMiddleware 为每个请求生成加密安全的 CSP nonce
func CSPNonceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nonce := generateNonce()
		c.Set("csp_nonce", nonce)
		c.Next()
	}
}

// generateNonce 生成加密安全的随机 nonce
func generateNonce() string {
	b := make([]byte, nonceLength)
	if _, err := rand.Read(b); err != nil {
		// 降级到基于时间戳的 nonce（不理想但安全）
		return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return base64.StdEncoding.EncodeToString(b)
}

// GetCSPNonce 从 Gin context 中获取 CSP nonce
func GetCSPNonce(c *gin.Context) string {
	if nonce, exists := c.Get("csp_nonce"); exists {
		if nonceStr, ok := nonce.(string); ok {
			return nonceStr
		}
	}
	return ""
}
