package middleware

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimiterEntry tracks request count and last reset time for a single key.
type rateLimiterEntry struct {
	count    int
	lastSeen time.Time
}

func evictExpiredEntries(entries map[string]*rateLimiterEntry, now time.Time, window time.Duration) {
	for k, v := range entries {
		if now.Sub(v.lastSeen) > 2*window {
			delete(entries, k)
		}
	}
}

// RateLimiterConfig defines the configuration for rate limiting.
type RateLimiterConfig struct {
	// MaxRequests is the maximum number of requests allowed within the window.
	MaxRequests int
	// Window is the time window for the rate limit.
	Window time.Duration
	// KeyFunc extracts the rate limit key from the request (default: client IP).
	KeyFunc func(c *gin.Context) string
}

// RateLimiter returns a Gin middleware that limits request frequency per key.
// Uses an in-memory sliding window counter with lazy eviction.
func RateLimiter(config RateLimiterConfig) gin.HandlerFunc {
	if config.MaxRequests <= 0 {
		config.MaxRequests = 100
	}
	if config.Window <= 0 {
		config.Window = time.Minute
	}
	if config.KeyFunc == nil {
		config.KeyFunc = func(c *gin.Context) string {
			return c.ClientIP()
		}
	}

	var mu sync.Mutex
	entries := make(map[string]*rateLimiterEntry)

	// Lazy eviction: periodically clean up stale entries
	var lastCleanup time.Time

	return func(c *gin.Context) {
		key := config.KeyFunc(c)

		mu.Lock()
		now := time.Now()

		// Lazy eviction: clean up entries older than 2x window every minute
		if now.Sub(lastCleanup) > time.Minute {
			evictExpiredEntries(entries, now, config.Window)
			lastCleanup = now
		}

		entry, exists := entries[key]
		if !exists || now.Sub(entry.lastSeen) > config.Window {
			entries[key] = &rateLimiterEntry{count: 1, lastSeen: now}
			mu.Unlock()
			c.Next()
			return
		}

		entry.count++
		if entry.count > config.MaxRequests {
			mu.Unlock()
			slog.Warn("rate limit exceeded", "key", key, "path", c.Request.URL.Path, "limit", config.MaxRequests, "window", config.Window)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    429,
				"message": "too many requests",
			})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}
