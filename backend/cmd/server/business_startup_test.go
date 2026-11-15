package main

import (
	"os"
	"strings"
	"testing"
)

// Startup ordering is pinned at the real composition root; the providers' live
// behavior and early middleware matching are exercised in their integration tests.
func TestBusinessStartupProviderAndCaptureOrder(t *testing.T) {
	raw, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	source := string(raw)
	ordered := [][]string{
		{"system.InitSystemModule(api", "auth.InitAuthModule(api", "business.InitBusinessModules(api"},
		{"middleware.RequestContextMiddleware(), middleware.OperationLogMiddleware(database.DB)", "middleware.BodySizeLimit(", "middleware.CSRFMiddleware()", "registerAPIRoutes(r)", "permissionroutes.Seal()", "contracts.SealBusinessTenantProviders()", "auditpolicy.ValidateRoutes(r.Routes())", "auditpolicy.Seal()"},
		// Business providers register permission routes from inside
		// registerAPIRoutes, so the seal above always runs after their last
		// declaration.
		{"func registerAPIRoutes(r *gin.Engine) {", "business.InitBusinessModules(api"},
	}
	for _, sequence := range ordered {
		last := -1
		for _, call := range sequence {
			position := strings.Index(source, call)
			if position < 0 || position <= last {
				t.Fatalf("startup order violated at %s", call)
			}
			last = position
		}
	}
}
