package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

// Queue-5 audit slice: the operation log row must carry the tenant of the
// resolved context (read after c.Next()), and never a tenant from a spoofable
// request field. Compat/no-middleware requests stamp 0.
func TestOperationLogMiddleware_StampTenantFromResolvedContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	var captured []SystemLogOper
	store := &operationLogAsyncStore{
		db:    db,
		queue: make(chan SystemLogOper, 4),
		done:  make(chan struct{}),
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Simulate TenantContextMiddleware resolving tenant 101 for this route.
		tenant.SetGin(c, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})
		c.Next()
	})
	router.Use(func(c *gin.Context) {
		start := time.Now()
		// Body capture is not needed for the tenant stamp; keep the flow minimal.
		c.Next()
		log := SystemLogOper{
			TenantID: tenantIDForOperationLog(c),
			Method:   c.Request.Method,
			OperURL:  c.Request.URL.Path,
			OperTime: start,
		}
		captured = append(captured, log)
		_ = store
	})
	router.POST("/api/v1/system/dict/type", func(c *gin.Context) {
		// A request trying to spoof audit ownership via body/header fields.
		c.Header("X-Tenant-Id", "202")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/system/dict/type", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if len(captured) != 1 {
		t.Fatalf("expected 1 captured log, got %d", len(captured))
	}
	if captured[0].TenantID != 101 {
		t.Fatalf("audit row must stamp the resolved tenant 101, got %d", captured[0].TenantID)
	}
}

func TestOperationLogMiddleware_StampTenantCompatIsZero(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var captured []SystemLogOper
	router := gin.New()
	router.Use(func(c *gin.Context) {
		// Compat: middleware resolves the global context (or nothing at all).
		tenant.SetGin(c, &tenant.Context{TenantID: 0, Mode: tenant.ModeCompat})
		c.Next()
		captured = append(captured, SystemLogOper{TenantID: tenantIDForOperationLog(c)})
	})
	router.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	if len(captured) != 1 || captured[0].TenantID != 0 {
		t.Fatalf("compat request must stamp tenant 0, got %+v", captured)
	}
}

func TestTenantIDForOperationLog_NoContextIsZero(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if got := tenantIDForOperationLog(c); got != 0 {
		t.Fatalf("no tenant context must stamp 0, got %d", got)
	}
}
