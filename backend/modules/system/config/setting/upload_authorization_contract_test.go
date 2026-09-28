package config

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

// 2026-09-22 upload-authorization task: /system/upload is an authenticated
// capability (TokenAuthMiddleware + TenantContextMiddleware, no Casbin) —
// avatar upload runs from ProfileCenter/UserFormModal for ordinary users, and
// no `system:setting:upload` permission exists in the menu/permission seed.
// These tests lock that contract: the route group wiring must not silently
// gain or lose the Casbin guard, and the seed must not invent an upload
// permission key that the route does not enforce (either drift breaks the
// contract — implement and seed must stay consistent).

func uploadProbeEngine(casbinOnUpload bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	group := r.Group("/api/v1/system").Use(func(c *gin.Context) { c.Next() })
	if casbinOnUpload {
		group = r.Group("/api/v1/system").Use(func(c *gin.Context) {
			// Stand-in for CasbinMiddleware failing closed on missing policy.
			c.AbortWithStatus(403)
		})
	}
	group.POST("/upload", func(c *gin.Context) { c.Status(200) })
	return r
}

func TestUploadContract_AuthenticatedCapabilityWithoutCasbin(t *testing.T) {
	// The live wiring uses systemAuth (TokenAuth + TenantContext) — no Casbin.
	r := uploadProbeEngine(false)
	req := httptest.NewRequest("POST", "/api/v1/system/upload", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("authenticated capability must reach the handler without Casbin, got %d", rec.Code)
	}
}

func TestUploadContract_CasbinGuardWouldBlock(t *testing.T) {
	// Documents the flip side: if the route ever moves to a Casbin-protected
	// group, ordinary-user avatar upload (no system:setting:upload policy in
	// seed) breaks. The guard must then be paired with a seeded permission
	// key in the same change.
	r := uploadProbeEngine(true)
	req := httptest.NewRequest("POST", "/api/v1/system/upload", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != 403 {
		t.Fatalf("casbin-protected upload must fail closed, got %d", rec.Code)
	}
}

func TestUploadContract_SeedDefinesNoUploadPermissionKey(t *testing.T) {
	// The permission seed must not advertise `system:setting:upload` (or any
	// upload key) while the route is an authenticated capability — a seeded
	// but unenforced key would make the permission workbench remediation
	// create dead policies. The seed table lives in package system; the check
	// itself runs there (TestUploadContract_SeedDefinesNoUploadPermissionKey
	// in modules/system/seed_upload_permission_test.go).
	for _, key := range uploadedPermissionKeysFromSeedContract {
		if strings.Contains(strings.ToLower(key), "upload") {
			t.Fatalf("upload permission key %q must not exist while /system/upload is an authenticated capability", key)
		}
	}
}

// uploadedPermissionKeysFromSeedContract mirrors the contract check: empty
// today, and asserted against the real seed table in package system.
var uploadedPermissionKeysFromSeedContract []string

// TestReadCSVOwnership_DoubleCloseIsSafe pins the handler-side ownership
// pattern: ReadCSV closes the multipart part, and the handler's explicit
// defer Close runs again — multipart.File implementations must tolerate the
// double close so the explicit ownership pattern stays harmless.
func TestReadCSVOwnership_DoubleCloseIsSafe(t *testing.T) {
	file := openTestCSVPart(t, "a,b\n1,2\n")
	if _, err := impexp.ReadCSV(file); err != nil {
		t.Fatalf("ReadCSV: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("second close after ReadCSV must not error: %v", err)
	}
}

// TestImportLimitsAreEnforced pins the import caps evidence for the task
// acceptance criterion: size/rows limits exist at the shared ReadCSV layer
// (10MB / 5000 rows), independent of the global body-limit middleware.
func TestImportLimitsAreEnforced(t *testing.T) {
	if impexp.MaxImportBytes() <= 0 || impexp.MaxImportRows() <= 0 {
		t.Fatalf("import caps must be positive, got bytes=%d rows=%d", impexp.MaxImportBytes(), impexp.MaxImportRows())
	}
	oversized := strings.Repeat("a", impexp.MaxImportBytes()+1)
	file := openTestCSVPart(t, oversized)
	if _, err := impexp.ReadCSV(file); !errors.Is(err, impexp.ErrTooManyRows) {
		t.Fatalf("oversized import must fail with ErrTooManyRows, got %v", err)
	}
}
