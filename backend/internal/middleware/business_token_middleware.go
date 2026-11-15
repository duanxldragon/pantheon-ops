package middleware

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// BusinessTokenAuthMiddleware authenticates an opaque token freshly, read-only.
func BusinessTokenAuthMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		data, err := businessauth.ReadSession(c.Request.Context(), rdb, token)
		if err != nil {
			common.Fail(c, common.CodeUnauthorized, "token.invalid")
			c.Abort()
			return
		}
		applyTokenContext(c, data)
		// Authentication identifies the actor; token role snapshots confer no
		// authority on opted routes, including to downstream legacy helpers.
		c.Set("roleKeys", []string{})
		c.Set("roleKey", "")
		c.Set(businessauth.IdentityKey, businessauth.IdentityFromSession(data, token))
		c.Next()
	}
}
