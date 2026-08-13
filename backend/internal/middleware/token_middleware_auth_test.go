package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pantheon-base/pkg/authtoken"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

// TestTokenAuthMiddleware_MissingTokenIsUnauthorized 锁定鉴权语义：
// 缺失 token（无 Cookie、无 Bearer）时必须以 401（业务码）中断且不再调用下游 handler；
// common.Fail 写入 HTTP 200 但响应体中 code=401、message="token.missing"。
// 该路径不依赖 Redis，可在本地确定性执行。
func TestTokenAuthMiddleware_MissingTokenIsUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(TokenAuthMiddleware(nil))

	reached := false
	engine.GET("/secure", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if reached {
		t.Fatal("expected middleware to abort before reaching the handler")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != common.CodeUnauthorized {
		t.Fatalf("expected business code %d in body, got %v", common.CodeUnauthorized, body)
	}
	if msg, ok := body["message"]; !ok || msg != "token.missing" {
		t.Fatalf("expected message %q, got %v", "token.missing", body)
	}
}

// TestTokenAuthMiddleware_InvalidTokenWhenRedisUnavailable 锁定：当 Redis 不可用
// （rdb=nil）且携带了 token 时，必须返回 401（token.invalid）并中断，绝不旁路鉴权。
func TestTokenAuthMiddleware_InvalidTokenWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(TokenAuthMiddleware(nil))

	reached := false
	engine.GET("/secure", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/secure", nil)
	req.Header.Set("Authorization", "Bearer some-invalid-token")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if reached {
		t.Fatal("expected middleware to abort before reaching the handler")
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != common.CodeUnauthorized {
		t.Fatalf("expected business code %d in body, got %v", common.CodeUnauthorized, body)
	}
	if msg, ok := body["message"]; !ok || msg != "token.invalid" {
		t.Fatalf("expected message %q, got %v", "token.invalid", body)
	}
}

// TestApplyTokenContext 直接锁定「写入 context 的键与值」契约，确保重构未改变
// 下游依赖的鉴权上下文结构：userId / username / roleKeys / sessionId 始终写入，
// 且仅当 RoleKeys 非空时额外写入 roleKey=RoleKeys[0]。
// 使用 gin.CreateTestContext 可正确初始化上下文互斥量，无需 HTTP 请求即可安全断言。
func TestApplyTokenContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("full session writes all keys", func(t *testing.T) {
		assertFullTokenContext(t)
	})

	t.Run("empty roleKeys omits roleKey", func(t *testing.T) {
		assertTokenContextWithoutRoleKey(t)
	})
}

func assertFullTokenContext(t *testing.T) {
	t.Helper()
	c := newTokenTestContext()
	applyTokenContext(c, &authtoken.SessionData{
		UserID:    123,
		Username:  "alice",
		RoleKeys:  []string{"admin", "user"},
		SessionID: "sess-abc",
	})

	assertTokenContextValue(t, c, "userId", uint64(123))
	assertTokenContextValue(t, c, "username", "alice")
	assertTokenContextValue(t, c, "sessionId", "sess-abc")
	assertTokenContextValue(t, c, "roleKey", "admin")
	assertTokenRoleKeys(t, c, []string{"admin", "user"})
}

func assertTokenContextWithoutRoleKey(t *testing.T) {
	t.Helper()
	c := newTokenTestContext()
	applyTokenContext(c, &authtoken.SessionData{
		UserID:    456,
		Username:  "bob",
		RoleKeys:  []string{},
		SessionID: "sess-def",
	})

	assertTokenContextValue(t, c, "userId", uint64(456))
	assertTokenContextValue(t, c, "sessionId", "sess-def")
	if _, ok := c.Get("roleKey"); ok {
		t.Fatal("expected roleKey to be absent when RoleKeys is empty")
	}
}

func newTokenTestContext() *gin.Context {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	return c
}

func assertTokenContextValue(t *testing.T, c *gin.Context, key string, want any) {
	t.Helper()
	got, ok := c.Get(key)
	if !ok || got != want {
		t.Fatalf("%s mismatch: ok=%v v=%v", key, ok, got)
	}
}

func assertTokenRoleKeys(t *testing.T, c *gin.Context, want []string) {
	t.Helper()
	value, ok := c.Get("roleKeys")
	if !ok {
		t.Fatalf("roleKeys missing: ok=%v v=%v", ok, value)
	}
	keys, isSlice := value.([]string)
	if !isSlice || len(keys) != len(want) || keys[0] != want[0] {
		t.Fatalf("roleKeys mismatch: ok=%v v=%v", ok, value)
	}
}
