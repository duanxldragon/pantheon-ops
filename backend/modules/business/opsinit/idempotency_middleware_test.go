package opsinit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func TestO4ExactIdempotencyBypassDoesNotReadOrCacheHeaders(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:26316"})
	defer rdb.Close()
	if rdb.Ping(context.Background()).Err() != nil {
		t.Fatal("required isolated Redis unavailable")
	}
	router := gin.New()
	router.Use(NewIdempotencyMiddleware(rdb).Handle())
	for _, route := range observability.ProtectedQueryRoutes() {
		router.Handle(route.Method, route.Template, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	}
	key := fmt.Sprintf("o4-exact-header-sentinel-%d", time.Now().UnixNano())
	t.Cleanup(func() { rdb.Del(context.Background(), idempotencyPrefix+key) })
	for _, route := range observability.ProtectedQueryRoutes() {
		for i := 0; i < 2; i++ {
			body := &neverReadBody{t: t}
			req := httptest.NewRequest(route.Method, strings.ReplaceAll(strings.ReplaceAll(route.Template, ":id", "1"), ":traceId", "abc"), body)
			req.Header.Set(idempotencyHeader, key)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusNoContent {
				t.Fatalf("exact route failed: %s %s", route.Method, route.Template)
			}
		}
	}
	if rdb.Exists(context.Background(), idempotencyPrefix+key).Val() != 0 {
		t.Fatal("protected route persisted incoming header")
	}
}

type neverReadBody struct{ t *testing.T }

func (r *neverReadBody) Read([]byte) (int, error) {
	r.t.Error("idempotency read protected body")
	return 0, fmt.Errorf("protected-body-sentinel")
}

func TestO4IdempotencyNearMissAndUnavailableKeepLegacyBehavior(t *testing.T) {
	for _, path := range []string{"/api/v1/business/observability/datasources/1/query-extra", "/api/v1/observability/datasources/1/query-extra", "/api/v1/business/observability/datasources/1/query"} {
		rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:26316"})
		if rdb.Ping(context.Background()).Err() != nil {
			t.Fatal("required isolated Redis unavailable")
		}
		key := fmt.Sprintf("o4-near-miss-%d", time.Now().UnixNano())
		router := gin.New()
		router.Use(NewIdempotencyMiddleware(rdb).Handle())
		router.PATCH(path, func(c *gin.Context) { c.Status(http.StatusNoContent) })
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPatch, path, nil)
			req.Header.Set(idempotencyHeader, key)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			if i == 0 && recorder.Code != http.StatusNoContent {
				t.Fatal("unmatched first write blocked")
			}
			if i == 1 && !strings.Contains(recorder.Body.String(), "common.idempotency_conflict") {
				t.Fatal("unmatched duplicate protection removed")
			}
		}
		rdb.Del(context.Background(), idempotencyPrefix+key)
		_ = rdb.Close()
		req := httptest.NewRequest(http.MethodPatch, path, nil)
		req.Header.Set(idempotencyHeader, key)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if !strings.Contains(recorder.Body.String(), "common.idempotency_unavailable") {
			t.Fatal("unmatched store error did not fail closed")
		}
	}
}
