package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Exercise the existing login-tenants activity path and the independent Redis
// validation primitive used by fresh authorization. This does not implement or
// claim coverage of the planned business middleware/provider integration.
func TestLegacyTokenActivityCannotResurrectBeforeFreshValidation(t *testing.T) {
	for _, mode := range []string{"deleted", "expired"} {
		t.Run(mode, func(t *testing.T) {
			runLegacyTokenActivityCase(t, mode)
		})
	}
}

func runLegacyTokenActivityCase(t *testing.T, mode string) {
	t.Helper()
	rdb := testredis.OpenDB(t, testredis.DBMiddleware)
	ctx := context.Background()
	resetTokenSessionCacheForTest(t, time.Minute)
	token := authtoken.NewAccessToken()
	t.Cleanup(func() { invalidateTokenSessionCache(token) })
	seed := seedStaleActivitySession(token)
	if err := authtoken.StoreSession(ctx, rdb, token, &seed, time.Minute); err != nil {
		t.Fatal(err)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/api/v1/auth/login-tenants", nil)
	stale, aborted := resolveTokenSession(ctx, c, rdb, token)
	if aborted {
		t.Fatal("legacy request did not read the valid session")
	}
	if cached, ok := lookupTokenSessionCache(token); !ok || cached.data.LastActivityAt != seed.LastActivityAt {
		t.Fatal("legacy request did not populate its existing cache")
	}
	invalidateTokenActivitySession(ctx, t, rdb, token, mode)
	refreshTokenActivity(ctx, rdb, token, stale)
	if cached, ok := lookupTokenSessionCache(token); !ok || cached.data.LastActivityAt != seed.LastActivityAt {
		t.Error("failed activity refresh advanced the legacy cache")
	}
	// Both a subsequent entry check and a provider-result release check
	// must read Redis independently, even while the legacy cache is warm.
	assertFreshValidationDeniesContinuations(ctx, t, rdb, token)
	if n, err := rdb.Exists(ctx, authtoken.SessionKey(token)).Result(); err != nil || n != 0 {
		t.Error("legacy activity recreated the deleted or expired key")
	}
}

func invalidateTokenActivitySession(ctx context.Context, t *testing.T, rdb *redis.Client, token, mode string) {
	t.Helper()
	var err error
	if mode == "deleted" {
		err = authtoken.DeleteSession(ctx, rdb, token)
	} else {
		err = rdb.PExpireAt(ctx, authtoken.SessionKey(token), time.Unix(1, 0)).Err()
	}
	if err != nil {
		t.Fatal(err)
	}
}

func assertFreshValidationDeniesContinuations(ctx context.Context, t *testing.T, rdb *redis.Client, token string) {
	t.Helper()
	providerCalls, releaseCalls := 0, 0
	for _, continuation := range []*int{&providerCalls, &releaseCalls} {
		_, err := authtoken.ValidateSession(ctx, rdb, token)
		if err == nil {
			(*continuation)++
		}
		if !errors.Is(err, authtoken.ErrNotFound) {
			t.Error("fresh validation accepted the stale legacy session")
		}
	}
	if providerCalls != 0 || releaseCalls != 0 {
		t.Error("a provider-like continuation ran after session invalidation")
	}
}
