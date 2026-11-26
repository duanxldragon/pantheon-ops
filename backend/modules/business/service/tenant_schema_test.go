package service

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantschematest"
)

func TestTenantOwnership(t *testing.T) {
	tenantschematest.RequireTenantOwned(t, &Application{})
}
