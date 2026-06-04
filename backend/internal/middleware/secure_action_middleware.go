package middleware

import (
	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// SecureActionMiddleware 敏感操作二次验证中间件
// 校验请求头中的 X-Operation-Token
func SecureActionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Operation-Token")
		if token == "" {
			common.FailWithCode(c, 403, "auth.operation.verification_required")
			c.Abort()
			return
		}

		claims, err := common.ParseOperationToken(token)
		if err != nil {
			common.FailWithCode(c, 403, "auth.operation.verification_expired")
			c.Abort()
			return
		}

		// 校验令牌所属用户是否为当前用户
		currentUserID := common.GetUserID(c)
		if claims.UserID != currentUserID {
			common.FailWithCode(c, 403, "auth.operation.verification_mismatch")
			c.Abort()
			return
		}
		currentSessionID := c.GetString("sessionId")
		if claims.SessionID == "" || currentSessionID == "" || claims.SessionID != currentSessionID {
			common.FailWithCode(c, 403, "auth.operation.verification_mismatch")
			c.Abort()
			return
		}

		c.Next()
	}
}
