package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestBusinessTokenFreshReadOnly(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBMiddleware)
	ctx := context.Background()
	token := authtoken.NewAccessToken()
	data := &authtoken.SessionData{UserID: 7, SessionID: "test-session", TenantID: 101, RoleKeys: []string{"admin"}, LastActivityAt: time.Now().Add(-2 * time.Minute).Unix()}
	if err := authtoken.StoreSession(ctx, rdb, token, data, time.Minute); err != nil {
		t.Fatal("seed session")
	}
	before, err := rdb.Get(ctx, authtoken.SessionKey(token)).Result()
	if err != nil {
		t.Fatal(err)
	}
	expiry, err := rdb.Do(ctx, "PEXPIRETIME", authtoken.SessionKey(token)).Int64()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	engine := gin.New()
	engine.Use(BusinessTokenAuthMiddleware(rdb))
	engine.POST("/business", func(c *gin.Context) { calls++; c.Status(204) })
	request := func() {
		req := httptest.NewRequest("POST", "/business", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		engine.ServeHTTP(httptest.NewRecorder(), req)
	}
	request()
	after, _ := rdb.Get(ctx, authtoken.SessionKey(token)).Result()
	expiryAfter, _ := rdb.Do(ctx, "PEXPIRETIME", authtoken.SessionKey(token)).Int64()
	if calls != 1 || before != after || expiry != expiryAfter {
		t.Fatal("fresh auth wrote payload or expiration")
	}
	assertBusinessTokenRevokedDespiteWarmCache(ctx, t, rdb, token, data, request, &calls)
	assertBusinessTokenBlacklistMarkers(ctx, t, rdb, token, data, request, &calls)
	assertBusinessTokenPersistentInvalidClosed(ctx, t, rdb, token, data, request, &calls)
}

func assertBusinessTokenRevokedDespiteWarmCache(ctx context.Context, t *testing.T, rdb *redis.Client, token string, data *authtoken.SessionData, request func(), calls *int) {
	t.Helper()
	// A warm legacy cache must not grant a deleted opaque session.
	storeTokenSessionCache(token, data, time.Now().Add(time.Minute))
	t.Cleanup(func() { invalidateTokenSessionCache(token) })
	if err := rdb.Del(ctx, authtoken.SessionKey(token)).Err(); err != nil {
		t.Fatal(err)
	}
	request()
	if *calls != 1 {
		t.Fatal("legacy cache granted a revoked token")
	}
}

func assertBusinessTokenBlacklistMarkers(ctx context.Context, t *testing.T, rdb *redis.Client, token string, data *authtoken.SessionData, request func(), calls *int) {
	t.Helper()
	for _, key := range []string{authtoken.BlacklistUserKey(7), authtoken.BlacklistSessionKey(data.SessionID)} {
		if err := authtoken.StoreSession(ctx, rdb, token, data, time.Minute); err != nil {
			t.Fatal(err)
		}
		if err := rdb.Set(ctx, key, "", time.Minute).Err(); err != nil {
			t.Fatal(err)
		}
		request()
		if *calls != 1 {
			t.Fatal("empty blacklist marker allowed")
		}
		if err := rdb.Del(ctx, key).Err(); err != nil {
			t.Fatal(err)
		}
		if err := rdb.LPush(ctx, key, "synthetic-marker").Err(); err != nil {
			t.Fatal(err)
		}
		request()
		if *calls != 1 {
			t.Fatal("independent blacklist read error allowed")
		}
		if err := rdb.Del(ctx, key).Err(); err != nil {
			t.Fatal(err)
		}
	}
}

func assertBusinessTokenPersistentInvalidClosed(ctx context.Context, t *testing.T, rdb *redis.Client, token string, data *authtoken.SessionData, request func(), calls *int) {
	t.Helper()
	if err := authtoken.StoreSession(ctx, rdb, token, data, 0); err != nil {
		t.Fatal(err)
	}
	request()
	if *calls != 1 {
		t.Fatal("persistent token accepted")
	}
	if err := rdb.Set(ctx, authtoken.SessionKey(token), `{"uid":-1,"sid":"forged","tn":-1}`, time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	request()
	if *calls != 1 {
		t.Fatal("invalid claim accepted")
	}
	if err := rdb.Close(); err != nil {
		t.Fatal(err)
	}
	request()
	if *calls != 1 {
		t.Fatal("closed store accepted")
	}
}

func TestBusinessTokenNilAndInvalidHeaders(t *testing.T) {
	engine := gin.New()
	engine.Use(BusinessTokenAuthMiddleware(nil))
	engine.POST("/business", func(c *gin.Context) { t.Fatal("nil store authorized") })
	for _, header := range []string{"", "Bearer", "Basic synthetic", "Bearer forged.jwt.token"} {
		req := httptest.NewRequest("POST", "/business", nil)
		req.Header.Set("Authorization", header)
		engine.ServeHTTP(httptest.NewRecorder(), req)
	}
}
