package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"pantheon-base/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitAllower abstracts the backing store for rate limit counters.
type RateLimitAllower interface {
	// Allow reports whether the given key is within the limit for the window.
	// Returns true if allowed, false if rate-limited.
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimiterConfig defines the configuration for rate limiting.
type RateLimiterConfig struct {
	MaxRequests int
	Window      time.Duration
	KeyFunc     func(c *gin.Context) string
	Store       RateLimitAllower // nil defaults to in-memory store
}

// RateLimiter returns a Gin middleware that limits request frequency per key.
func RateLimiter(config RateLimiterConfig) gin.HandlerFunc {
	if config.MaxRequests <= 0 {
		config.MaxRequests = 100
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	store := config.Store
	if store == nil {
		store = newMemoryRateLimitStore()
	}

	return func(c *gin.Context) {
		key := config.KeyFunc(c)
		allowed, err := store.Allow(c.Request.Context(), key, config.MaxRequests, config.Window)
		if err != nil {
			slog.Warn("rate limit store error, allowing request", "error", err)
			c.Next()
			return
		}
		if !allowed {
			slog.Warn("rate limit exceeded", "limit", config.MaxRequests)
			c.JSON(http.StatusTooManyRequests, gin.H{"code": 429, "message": "request.too_many"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// NewRedisRateLimitStore returns a RateLimitAllower backed by Redis.
// Uses a Lua script for atomic INCR + EXPIRE in a single round-trip.
func NewRedisRateLimitStore() RateLimitAllower {
	if database.RDB == nil {
		return newMemoryRateLimitStore()
	}
	return &redisRateLimitStore{rdb: database.RDB}
}

// ── Redis store ───────────────────────────────────────────────────────

var incrWithExpireScript = redis.NewScript(`
local key = KEYS[1]
local limit = tonumber(ARGV[2])
local window = tonumber(ARGV[3])
local current = redis.call("INCR", key)
if current == 1 then
	redis.call("PEXPIRE", key, window)
end
if current > limit then
	return 0
end
return 1
`)

type redisRateLimitStore struct {
	rdb *redis.Client
}

func (s *redisRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	redisKey := "rate_limit:" + key
	result, err := incrWithExpireScript.Run(ctx, s.rdb, []string{redisKey}, 1, limit, window.Milliseconds()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ── Memory store (fallback) ───────────────────────────────────────────

type memoryRateLimitStore struct {
	mu      sync.Mutex
	entries map[string]*rateLimiterEntry
}

type rateLimiterEntry struct {
	count    int
	lastSeen time.Time
}

func newMemoryRateLimitStore() *memoryRateLimitStore {
	return &memoryRateLimitStore{entries: make(map[string]*rateLimiterEntry)}
}

func (s *memoryRateLimitStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	entry, exists := s.entries[key]
	if !exists || now.Sub(entry.lastSeen) > window {
		s.entries[key] = &rateLimiterEntry{count: 1, lastSeen: now}
		s.evictStaleLocked(now, window)
		return true, nil
	}

	entry.count++
	if entry.count > limit {
		return false, nil
	}
	return true, nil
}

// evictStaleLocked 在 map 超阈值时清理过期条目，防止无 Redis 部署下
// 按 key（路径+IP）无界增长。调用方须持有 s.mu。
func (s *memoryRateLimitStore) evictStaleLocked(now time.Time, window time.Duration) {
	const maxEntries = 10000
	if len(s.entries) <= maxEntries {
		return
	}
	for key, entry := range s.entries {
		if now.Sub(entry.lastSeen) > window {
			delete(s.entries, key)
		}
	}
}
