package middleware

import (
	"net/http"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	idempotencyHeader = "X-Idempotency-Key"
	idempotencyTTL    = 24 * time.Hour
	idempotencyPrefix = "pantheon:idem:"
)

// IdempotencyMiddleware rejects duplicate write requests that carry the same
// X-Idempotency-Key header within the TTL window, protecting start/execute
// endpoints from client retries that would otherwise double-apply.
type IdempotencyMiddleware struct {
	redis *redis.Client
}

// NewIdempotencyMiddleware creates the idempotency middleware.
func NewIdempotencyMiddleware(redisClient *redis.Client) *IdempotencyMiddleware {
	return &IdempotencyMiddleware{redis: redisClient}
}

// Handle returns the Gin handler. Requests without the header pass through;
// requests with the header are deduplicated via Redis SET NX.
func (m *IdempotencyMiddleware) Handle() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead || c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}
		key := strings.TrimSpace(c.GetHeader(idempotencyHeader))
		if key == "" || m.redis == nil {
			c.Next()
			return
		}
		redisKey := idempotencyPrefix + key
		ok, err := m.redis.SetNX(c.Request.Context(), redisKey, "1", idempotencyTTL).Result()
		if err != nil {
			// Redis failure: fail closed so duplicates are never double-applied.
			common.Fail(c, common.CodeError, "common.idempotency_unavailable")
			c.Abort()
			return
		}
		if !ok {
			common.Fail(c, http.StatusConflict, "common.idempotency_conflict")
			c.Abort()
			return
		}
		c.Next()
	}
}
