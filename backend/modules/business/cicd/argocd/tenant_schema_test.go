package argocd

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantschematest"
)

func TestTenantOwnership(t *testing.T) {
	tenantschematest.RequireTenantOwned(t, &ArgoCDServer{}, &ArgoCDRepository{}, &ArgoCDApplication{}, &ArgoCDSyncHistory{}, &WebhookEvent{})
	tenantschematest.RequireTenantFirstUnique(t, &ArgoCDServer{}, "uk_argocd_server_name", "tenant_id", "name")
}
