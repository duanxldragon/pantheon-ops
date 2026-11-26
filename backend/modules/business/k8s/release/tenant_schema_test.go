package release

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantschematest"
)

func TestTenantOwnership(t *testing.T) {
	tenantschematest.RequireTenantOwned(t, &Release{})
	tenantschematest.RequireTenantFirstUnique(t, &Release{}, "uk_k8s_release_idempotency", "tenant_id", "idempotency_key")
}
