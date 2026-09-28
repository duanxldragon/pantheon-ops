package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
)

type tenantSmokeResponse struct {
	TenantID uint64 `json:"tenantId"`
	Mode     string `json:"mode"`
}

// TestTenantContextMiddleware_HostileTwoTenantMatrix exercises the runtime
// boundary used by protected routes. It deliberately sends untrusted headers
// and runs two tenant requests concurrently to catch ambient-context leakage.
func TestTenantContextMiddleware_HostileTwoTenantMatrix(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		mode           string
		subjectTenant  string
		headerTenant   string
		roles          []string
		wantHTTPStatus int
		wantCode       int
		wantTenantID   uint64
		wantMode       string
	}{
		{
			name: "tenant claim resolves tenant 101",
			mode: tenant.ModeMulti, subjectTenant: "101",
			wantHTTPStatus: http.StatusOK, wantCode: common.CodeSuccess,
			wantTenantID: 101, wantMode: tenant.ModeMulti,
		},
		{
			name: "untrusted header cannot switch tenant",
			mode: tenant.ModeMulti, subjectTenant: "101", headerTenant: "202",
			wantHTTPStatus: http.StatusOK, wantCode: common.CodeForbidden,
		},
		{
			name: "platform override can switch explicitly",
			mode: tenant.ModeMulti, subjectTenant: "101", headerTenant: "202",
			roles: []string{"platform_ops"}, wantHTTPStatus: http.StatusOK,
			wantCode: common.CodeSuccess, wantTenantID: 202, wantMode: tenant.ModeMulti,
		},
		{
			name: "missing tenant claim is denied in multi mode",
			mode: tenant.ModeMulti, wantHTTPStatus: http.StatusOK,
			wantCode: common.CodeError,
		},
		{
			name: "compat ignores forged header",
			mode: tenant.ModeCompat, subjectTenant: "101", headerTenant: "202",
			wantHTTPStatus: http.StatusOK, wantCode: common.CodeSuccess,
			wantTenantID: tenant.PlatformGlobalTenantID, wantMode: tenant.ModeCompat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := runTenantSmokeRequest(t, tt.mode, tt.subjectTenant, tt.headerTenant, tt.roles)
			assertTenantSmokeResponse(t, recorder, tt.wantHTTPStatus, tt.wantCode, tt.wantTenantID, tt.wantMode)
		})
	}
}

// runTenantSmokeRequest builds the engine with the same stub/context keys as
// the hostile matrix and issues the protected request.
func runTenantSmokeRequest(t *testing.T, mode, subjectTenant, headerTenant string, roles []string) *httptest.ResponseRecorder {
	t.Helper()
	loader := tenant.NewModeLoader(func() string { return mode }, 0)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		if subjectTenant != "" {
			c.Set("tenantId", subjectTenant)
		}
		if len(roles) > 0 {
			c.Set("roleKeys", roles)
		}
	})
	engine.Use(TenantContextMiddleware(loader, nil))
	engine.GET("/protected", func(c *gin.Context) {
		ctx := tenant.FromGin(c)
		if ctx == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"code": common.CodeError})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": common.CodeSuccess, "data": tenantSmokeResponse{
			TenantID: ctx.TenantID, Mode: ctx.Mode,
		}})
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if headerTenant != "" {
		req.Header.Set(tenant.HeaderTenantID, headerTenant)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

// assertTenantSmokeResponse verifies the envelope and, on success, the exact
// resolved tenant context.
func assertTenantSmokeResponse(t *testing.T, recorder *httptest.ResponseRecorder, wantHTTPStatus, wantCode int, wantTenantID uint64, wantMode string) {
	t.Helper()
	if recorder.Code != wantHTTPStatus {
		t.Fatalf("http status: got %d, want %d; body=%s", recorder.Code, wantHTTPStatus, recorder.Body.String())
	}
	var payload struct {
		Code int                  `json:"code"`
		Data *tenantSmokeResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != wantCode {
		t.Fatalf("response code: got %d, want %d; body=%s", payload.Code, wantCode, recorder.Body.String())
	}
	if wantCode == common.CodeSuccess {
		if payload.Data == nil || payload.Data.TenantID != wantTenantID || payload.Data.Mode != wantMode {
			t.Fatalf("resolved context: got %+v, want tenant=%d mode=%s", payload.Data, wantTenantID, wantMode)
		}
	}
}

func TestTenantContextMiddleware_ConcurrentRequestsKeepTenantBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	loader := tenant.NewModeLoader(func() string { return tenant.ModeMulti }, 0)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("tenantId", c.GetHeader("X-Test-Tenant"))
		c.Next()
	})
	engine.Use(TenantContextMiddleware(loader, nil))
	engine.GET("/protected", func(c *gin.Context) {
		ctx := tenant.FromGin(c)
		c.JSON(http.StatusOK, gin.H{"tenantId": ctx.TenantID})
	})

	type result struct {
		want uint64
		got  uint64
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, id := range []string{"101", "202"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := id
			req := httptest.NewRequest(http.MethodGet, "http://example.com/protected", nil)
			req.Header.Set("X-Test-Tenant", id)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, req)
			var payload struct {
				TenantID uint64 `json:"tenantId"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Errorf("decode concurrent response: %v", err)
				return
			}
			want := uint64(101)
			if id == "202" {
				want = 202
			}
			results <- result{want: want, got: payload.TenantID}
		}()
	}
	wg.Wait()
	close(results)
	for got := range results {
		if got.got != got.want {
			t.Errorf("concurrent request leaked tenant context: got=%d want=%d", got.got, got.want)
		}
	}
}
