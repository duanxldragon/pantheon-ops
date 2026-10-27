package login

import (
	"errors"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// tenantAttributionError exposes the tenant a login attempt belongs to without
// changing the error text. Message-key resolution parses Error(), and tenant
// gate classification relies on errors.Is, so both must survive the wrap.
type tenantAttributionError struct {
	tenantID uint64
	err      error
}

func (e *tenantAttributionError) Error() string    { return e.err.Error() }
func (e *tenantAttributionError) Unwrap() error    { return e.err }
func (e *tenantAttributionError) TenantID() uint64 { return e.tenantID }

// tenantIDFromError reads the attribution stamped by attributeTenant.
func tenantIDFromError(err error) (uint64, bool) {
	var carrier interface{ TenantID() uint64 }
	if errors.As(err, &carrier) {
		return carrier.TenantID(), true
	}
	return 0, false
}

func attributeTenant(err error, tenantID uint64) error {
	if err == nil || tenantID == tenant.PlatformGlobalTenantID {
		return err
	}
	var carrier interface{ TenantID() uint64 }
	if errors.As(err, &carrier) {
		return err
	}
	return &tenantAttributionError{tenantID: tenantID, err: err}
}

// loginAuditTenantID resolves the tenant a login audit row belongs to. The
// subject's owning tenant is only a valid audit scope in multi mode: compat
// reads and writes are pinned to tenant 0 (tenant.WithTenantScope), so
// stamping a tenant id there would write rows no compat admin can read.
func (s *LoginService) loginAuditTenantID(user *authuser.User) uint64 {
	if user == nil || user.TenantID == tenant.PlatformGlobalTenantID {
		return tenant.PlatformGlobalTenantID
	}
	if tenant.NormalizeMode(tenant.FeatureFlagSettingKeyReader(s.db)) != tenant.ModeMulti {
		return tenant.PlatformGlobalTenantID
	}
	return user.TenantID
}

// withLoginAuditTenant returns a facade whose audit writes land in tenantID.
// Tenant 0 keeps the receiver unchanged so compat rows stay tenant 0.
func (s *Runtime) withLoginAuditTenant(tenantID uint64) *Runtime {
	if tenantID == tenant.PlatformGlobalTenantID {
		return s
	}
	return s.WithTenantContext(&tenant.Context{TenantID: tenantID, Mode: tenant.ModeMulti, ResolvedBy: "login-audit"})
}

func (s *Runtime) loginAuditTenantID(user *authuser.User) uint64 {
	return s.loginSvc.loginAuditTenantID(user)
}
