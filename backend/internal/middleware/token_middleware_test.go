package middleware

import (
	"testing"
	"time"

	"pantheon-base/pkg/authtoken"
)

func TestLoadTokenSessionCacheTTL(t *testing.T) {
	t.Setenv("PANTHEON_TOKEN_CACHE_TTL_SECONDS", "0")
	if got := loadTokenSessionCacheTTL(); got != 0 {
		t.Fatalf("expected zero ttl when env is 0, got %s", got)
	}
}

func TestTokenSessionCacheDisabledByZeroTTL(t *testing.T) {
	previousTTL := tokenSessionCacheTTL
	previousCache := tokenSessionCache
	defer func() {
		tokenSessionCacheTTL = previousTTL
		tokenSessionCache = previousCache
	}()

	tokenSessionCacheTTL = 0
	tokenSessionCache = make(map[string]*tokenCachedSession)

	storeTokenSessionCache("token", &authtoken.SessionData{}, time.Now().Add(time.Hour))
	if len(tokenSessionCache) != 0 {
		t.Fatalf("expected cache to stay empty when ttl is zero, got %d entries", len(tokenSessionCache))
	}

	tokenSessionCache["token"] = &tokenCachedSession{
		data:      &authtoken.SessionData{},
		cachedAt:  time.Now(),
		expiresAt: time.Now().Add(time.Hour),
	}
	if _, ok := lookupTokenSessionCache("token"); ok {
		t.Fatal("expected cache lookup to miss when ttl is zero")
	}
}
