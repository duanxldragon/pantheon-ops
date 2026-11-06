package contracts

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// routeGroupRefreshStub stands in for modules/system.RefreshSyncMiddleware so
// this package can pin the chain without importing the module that consumes it.
func routeGroupRefreshStub(c *gin.Context) {}

// TestSystemTokenGroupChain pins the token-only chain. The refresh-sync state
// endpoint is authenticated but is not a console: it must NOT gain the
// platform-scope tenant context, which would change how it resolves tenant
// context in multi mode.
func TestSystemTokenGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	names := handlerNames(t, SystemTokenGroup(gin.New().Group("/api/v1"), "/system", nil))

	if len(names) != 1 {
		t.Fatalf("SystemTokenGroup chain length = %d (%v), want 1 (TokenAuth)", len(names), names)
	}
	if !strings.Contains(names[0], "TokenAuthMiddleware") {
		t.Fatalf("SystemTokenGroup[0] = %s, want TokenAuthMiddleware", names[0])
	}
}

// TestSystemAuthGroupChain pins the upload group chain:
// TokenAuth -> PlatformScopeTenantContext and no Casbin.
func TestSystemAuthGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	names := handlerNames(t, SystemAuthGroup(gin.New().Group("/api/v1"), "/system", SystemGroupOptions{}))

	if len(names) != 2 {
		t.Fatalf("SystemAuthGroup chain length = %d (%v), want 2 (TokenAuth, PlatformScopeTenantContext)", len(names), names)
	}
	if !strings.Contains(names[0], "TokenAuthMiddleware") {
		t.Fatalf("SystemAuthGroup[0] = %s, want TokenAuthMiddleware", names[0])
	}
	if !strings.Contains(names[1], "PlatformScopeTenantContextMiddleware") {
		t.Fatalf("SystemAuthGroup[1] = %s, want PlatformScopeTenantContextMiddleware", names[1])
	}
}

// TestSystemProtectedGroupChain pins the three combinations that used to be
// hand-written at every system registration site (audit ARCH-01):
//
//	base:       TokenAuth -> PlatformScopeTenantContext -> Casbin
//	+Capability: ... -> PlatformCapability
//	+DataScope:  ... -> DataScope
//	+RefreshSync: ... -> refresh middleware
//
// Order matters: Casbin authorises the route before DataScope narrows rows, and
// the capability gate runs before either data decision is trusted.
func TestSystemProtectedGroupChain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		opts SystemGroupOptions
		want []string
	}{
		{
			name: "base",
			opts: SystemGroupOptions{},
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware"},
		},
		{
			name: "platform capability",
			opts: SystemGroupOptions{}.WithCapability(),
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware", "PlatformCapabilityMiddleware"},
		},
		{
			name: "data scope",
			opts: SystemGroupOptions{}.WithDataScope(),
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware", "DataScopeMiddleware"},
		},
		{
			name: "refresh sync",
			opts: SystemGroupOptions{RefreshSync: routeGroupRefreshStub},
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware", "routeGroupRefreshStub"},
		},
		{
			name: "data scope and refresh sync",
			opts: SystemGroupOptions{RefreshSync: routeGroupRefreshStub}.WithDataScope(),
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware", "DataScopeMiddleware", "routeGroupRefreshStub"},
		},
		{
			name: "capability and refresh sync",
			opts: SystemGroupOptions{RefreshSync: routeGroupRefreshStub}.WithCapability(),
			want: []string{"TokenAuthMiddleware", "PlatformScopeTenantContextMiddleware", "CasbinMiddleware", "PlatformCapabilityMiddleware", "routeGroupRefreshStub"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			names := handlerNames(t, SystemProtectedGroup(gin.New().Group("/api/v1"), "/system", test.opts))

			if len(names) != len(test.want) {
				t.Fatalf("chain length = %d (%v), want %d (%v)", len(names), names, len(test.want), test.want)
			}
			for i, want := range test.want {
				if !strings.Contains(names[i], want) {
					t.Fatalf("chain[%d] = %s, want %s (chain = %v)", i, names[i], want, names)
				}
			}
		})
	}
}

// TestSystemGroupOptionsReturnsCopies guards the builder: WithCapability /
// WithDataScope are used on shared values, so mutating the receiver would leak
// one module's decision into the next.
func TestSystemGroupOptionsReturnsCopies(t *testing.T) {
	base := SystemGroupOptions{}
	scoped := base.WithDataScope()
	capable := base.WithCapability()

	if base.DataScope || base.Capability {
		t.Fatalf("builder mutated the receiver: %+v", base)
	}
	if !scoped.DataScope || scoped.Capability {
		t.Fatalf("WithDataScope produced %+v", scoped)
	}
	if !capable.Capability || capable.DataScope {
		t.Fatalf("WithCapability produced %+v", capable)
	}
}
