package middleware

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"

	"github.com/gin-gonic/gin"
)

func resetTokenSessionCacheForTest(t *testing.T, ttl time.Duration) {
	t.Helper()

	previousTTL := tokenSessionCacheTTL
	t.Cleanup(func() { tokenSessionCacheTTL = previousTTL })
	tokenSessionCacheTTL = ttl
}

func seedStaleActivitySession(token string) authtoken.SessionData {
	return authtoken.SessionData{
		UserID:         7,
		Username:       "race",
		RoleKeys:       []string{"admin", "auditor"},
		SessionID:      "sid-race",
		LastActivityAt: time.Now().Add(-2 * time.Minute).Unix(),
	}
}

// TestTokenSessionCacheHandsOutCopies is the deterministic half of the guard: a
// caller must never be able to reach the cached entry through the value it was
// given. It fails without -race, so the invariant is checked in every CI run,
// not only in the race job.
func TestTokenSessionCacheHandsOutCopies(t *testing.T) {
	resetTokenSessionCacheForTest(t, time.Minute)
	token := "copy-guard-token"
	seed := seedStaleActivitySession(token)
	storeTokenSessionCache(token, &seed, time.Now().Add(time.Hour))

	first, ok := lookupTokenSessionCache(token)
	if !ok {
		t.Fatal("expected a cache hit for the freshly stored token")
	}
	first.data.LastActivityAt = 424242
	first.data.RoleKeys[0] = "hacked"

	second, ok := lookupTokenSessionCache(token)
	if !ok {
		t.Fatal("expected a cache hit on the second lookup")
	}
	if second.data.LastActivityAt != seed.LastActivityAt {
		t.Fatalf("mutating a returned session changed the cached one: LastActivityAt = %d, want %d",
			second.data.LastActivityAt, seed.LastActivityAt)
	}
	if second.data.RoleKeys[0] != "admin" {
		t.Fatalf("mutating a returned RoleKeys slice changed the cached one: RoleKeys[0] = %q, want %q",
			second.data.RoleKeys[0], "admin")
	}

	// The caller that stored the session must not keep a handle either: a later
	// write to the original struct must not show up in the cache.
	seed.RoleKeys[1] = "mutated-after-store"
	third, _ := lookupTokenSessionCache(token)
	if third.data.RoleKeys[1] != "auditor" {
		t.Fatalf("cache kept a reference to the caller's slice: RoleKeys[1] = %q, want %q",
			third.data.RoleKeys[1], "auditor")
	}
}

// TestTokenSessionCacheConcurrentSameTokenIsRaceFree drives the real hot path
// (resolve -> refresh) from several goroutines holding the same token. Before
// the fix this reports DATA RACE: every request received the same
// *SessionData, and refreshTokenActivity wrote LastActivityAt to it outside the
// cache lock. Run with -race; the assertions alone would pass on broken code,
// which is why the manifest requires the race reading.
func TestTokenSessionCacheConcurrentSameTokenIsRaceFree(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBMiddleware)
	ctx := context.Background()

	resetTokenSessionCacheForTest(t, time.Minute)
	token := "concurrent-same-token"
	// LastActivityAt is two minutes old, so every call takes the write path
	// inside refreshTokenActivity instead of racing once and then going quiet.
	seeded := seedStaleActivitySession(token)
	storeTokenSessionCache(token, &seeded, time.Now().Add(time.Hour))

	const goroutines = 8
	const iterations = 50

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan string, goroutines*iterations)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/api/v1/ping", nil)
			<-start
			for j := 0; j < iterations; j++ {
				sessionData, aborted := resolveTokenSession(ctx, c, rdb, token)
				if aborted {
					errs <- "resolveTokenSession aborted on a cached session"
					return
				}
				if sessionData.UserID != 7 {
					errs <- "resolved session lost its identity"
					return
				}
				refreshTokenActivity(ctx, rdb, token, sessionData)
				_ = sessionData.RoleKeys
			}
		}()
	}

	close(start)
	wg.Wait()
	close(errs)

	for msg := range errs {
		t.Error(msg)
	}

	final, ok := lookupTokenSessionCache(token)
	if !ok {
		t.Fatal("expected the session to still be cached after the concurrent run")
	}
	if final.data.LastActivityAt <= 0 {
		t.Fatalf("activity timestamp was never refreshed: %d", final.data.LastActivityAt)
	}
}
