package config

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

// Queue-5 upload slice: object keys must embed the canonical tenant identity
// in multi mode (collision isolation across tenants) and keep the legacy
// layout under compat. The tenant segment comes only from the resolved
// context — a spoofable `scope` query value cannot cross tenants.
func TestUploadScope_TenantPrefixedInMultiMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &SettingHandler{}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = mustRequest("POST", "/api/v1/system/upload?scope=avatar")
	tenant.SetGin(c, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})

	got := handler.uploadScope(c)
	if got != "t101/avatar" {
		t.Fatalf("expected t101/avatar, got %q", got)
	}
}

func TestUploadScope_CompatKeepsLegacyLayout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &SettingHandler{}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = mustRequest("POST", "/api/v1/system/upload?scope=avatar")
	tenant.SetGin(c, &tenant.Context{TenantID: 0, Mode: tenant.ModeCompat})

	if got := handler.uploadScope(c); got != "avatar" {
		t.Fatalf("compat must keep legacy scope, got %q", got)
	}
}

func TestUploadScope_DefaultScopeAndNoContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &SettingHandler{}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = mustRequest("POST", "/api/v1/system/upload")

	if got := handler.uploadScope(c); got != "general" {
		t.Fatalf("default scope must be general, got %q", got)
	}
}

func TestUploadScope_SpoofedQueryCannotEscapeTenantNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &SettingHandler{}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	// Even a hostile scope value stays under the tenant's own namespace prefix.
	c.Request = mustRequest("POST", "/api/v1/system/upload?scope=../../202")
	tenant.SetGin(c, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})

	got := handler.uploadScope(c)
	if !strings.HasPrefix(got, "t101/") {
		t.Fatalf("tenant namespace prefix missing: %q", got)
	}
}

func mustRequest(method, target string) *http.Request {
	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		panic(err)
	}
	return req
}
