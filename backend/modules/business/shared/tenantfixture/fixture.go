package tenantfixture

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

// Fixture holds an isolated test database and its tenant contexts.
type Fixture struct {
	DB      *gorm.DB
	TenantA *tenant.Context
	TenantB *tenant.Context
	Compat  *tenant.Context
}

// Open prepares a disposable MySQL database for tenant isolation tests.
func Open(t *testing.T) Fixture {
	t.Helper()
	return Fixture{
		DB:      testmysql.Open(t),
		TenantA: &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti},
		TenantB: &tenant.Context{TenantID: 202, Mode: tenant.ModeMulti},
		Compat:  &tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat},
	}
}
