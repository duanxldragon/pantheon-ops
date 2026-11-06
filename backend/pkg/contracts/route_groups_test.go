package contracts

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// handlerNames returns the fully qualified function names of the middleware
// chain attached to the group returned by a route-group helper.
//
// gin.RouterGroup.Handlers is the merged chain (parent handlers + the ones
// added by Group/Use), and a group obtained from a fresh engine has no parent
// handlers, so the slice is exactly what the helper installed.
func handlerNames(t *testing.T, routes gin.IRoutes) []string {
	t.Helper()

	group, ok := routes.(*gin.RouterGroup)
	if !ok {
		t.Fatalf("route-group helper returned %T, want *gin.RouterGroup", routes)
	}

	names := make([]string, 0, len(group.Handlers))
	for _, handler := range group.Handlers {
		fn := runtime.FuncForPC(reflect.ValueOf(handler).Pointer())
		if fn == nil {
			t.Fatalf("cannot resolve handler %p to a function name", handler)
		}
		names = append(names, fn.Name())
	}
	return names
}

func containsName(names []string, fragment string) bool {
	for _, name := range names {
		if strings.Contains(name, fragment) {
			return true
		}
	}
	return false
}

// assertNoTenantMiddleware encodes TENANT_CONTRACT_V1 §2.6: the business domain
// does not participate in tenant isolation in V1, so none of the route-group
// helpers may install a tenant-context middleware.
//
// This test is the ratchet. Adding a tenant middleware to these helpers is a
// contract change, not a bug fix: it must land together with §2.6 being
// rewritten, tenant_id columns on business tables, a generator change, and a
// dual-repo release. Failing here first is the intended outcome.
func assertNoTenantMiddleware(t *testing.T, helper string, names []string) {
	t.Helper()

	for _, forbidden := range []string{"TenantContextMiddleware", "PlatformScopeTenantContextMiddleware"} {
		if containsName(names, forbidden) {
			t.Fatalf(
				"%s installs %s, but TENANT_CONTRACT_V1 §2.6 states the business domain does not "+
					"participate in tenant isolation in V1; chain = %v",
				helper, forbidden, names,
			)
		}
	}
}

// TestProtectedGroupChain pins the exact middleware chain of ProtectedGroup.
// The chain is Token -> Casbin and nothing else; a fourth entry means someone
// widened the contract without updating §2.6.
func TestProtectedGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	names := handlerNames(t, ProtectedGroup(gin.New().Group("/api/v1"), "/business"))

	if len(names) != 2 {
		t.Fatalf("ProtectedGroup chain length = %d (%v), want 2 (TokenAuth, Casbin)", len(names), names)
	}
	if !strings.Contains(names[0], "TokenAuthMiddleware") {
		t.Fatalf("ProtectedGroup[0] = %s, want TokenAuthMiddleware", names[0])
	}
	if !strings.Contains(names[1], "CasbinMiddleware") {
		t.Fatalf("ProtectedGroup[1] = %s, want CasbinMiddleware", names[1])
	}
	assertNoTenantMiddleware(t, "ProtectedGroup", names)
}

// TestDataScopedGroupChain pins the exact middleware chain of DataScopedGroup:
// Token -> Casbin -> DataScope. DataScope is the IAM data scope (self/dept/all),
// which is orthogonal to tenancy.
func TestDataScopedGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	names := handlerNames(t, DataScopedGroup(gin.New().Group("/api/v1"), "/business", nil))

	if len(names) != 3 {
		t.Fatalf("DataScopedGroup chain length = %d (%v), want 3 (TokenAuth, Casbin, DataScope)", len(names), names)
	}
	if !strings.Contains(names[0], "TokenAuthMiddleware") {
		t.Fatalf("DataScopedGroup[0] = %s, want TokenAuthMiddleware", names[0])
	}
	if !strings.Contains(names[1], "CasbinMiddleware") {
		t.Fatalf("DataScopedGroup[1] = %s, want CasbinMiddleware", names[1])
	}
	if !strings.Contains(names[2], "DataScopeMiddleware") {
		t.Fatalf("DataScopedGroup[2] = %s, want DataScopeMiddleware", names[2])
	}
	assertNoTenantMiddleware(t, "DataScopedGroup", names)
}
