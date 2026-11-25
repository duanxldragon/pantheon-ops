package filedist

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantschematest"
)

func TestTenantOwnership(t *testing.T) {
	tenantschematest.RequireTenantOwned(t, &FileStorage{}, &FileRecord{}, &DistributionTask{}, &DistributionTarget{})
	tenantschematest.RequireTenantFirstUnique(t, &FileStorage{}, "uk_filedist_storage_name", "tenant_id", "name")
}
