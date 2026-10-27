package tenant

import (
	"testing"

	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// TestTenantRoleDefinitionsAreRegisteredAsTenantScoped 守护角色集与豁免集同步：
// DataScopeMiddleware 依据 pkg/tenant.TenantScopedRoleKeys 豁免租户内角色，使其
// 不受平台级 dept 行级策略约束。若新增一个租户角色却忘记登记，该角色的每个请求都会
// 落入 fail-closed 分支并返回 500。这里把「provisioning 集 ⊆ 豁免集」钉成测试，
// 让遗漏在 CI 就失败，而不是等到运行期。
func TestTenantRoleDefinitionsAreRegisteredAsTenantScoped(t *testing.T) {
	if len(tenantRoleDefinitions) == 0 {
		t.Fatal("tenantRoleDefinitions must not be empty")
	}
	for _, def := range tenantRoleDefinitions {
		if !tenantctx.IsTenantScopedRoleKey(def.Key) {
			t.Fatalf("tenant role %q is provisioned but not registered in pkg/tenant.TenantScopedRoleKeys; "+
				"its requests would fail closed with a 500", def.Key)
		}
	}
}
