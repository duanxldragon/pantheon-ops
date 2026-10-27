package login

import (
	"context"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// ListLoginTenantCandidates lists the tenants a user may explicitly log into
// (multi-membership tenant selection, contract §3.1 source 2: the login
// request may specify the tenant; the session claim must match an active
// membership). Compat mode returns an empty list — selection UI stays hidden
// because login never carries a claim under compat (contract §6).
// Multi-membership users currently denied by discovery are exactly the
// audience of this endpoint. Errors fail closed (empty list, no detail leak).
func (s *Runtime) ListLoginTenantCandidates(ctx context.Context, userID uint64) []tenant.LoginTenantCandidate {
	if s.db == nil || userID == 0 {
		return []tenant.LoginTenantCandidate{}
	}
	if tenant.NormalizeMode(tenant.FeatureFlagSettingKeyReader(s.db)) != tenant.ModeMulti {
		return []tenant.LoginTenantCandidate{}
	}
	candidates, err := tenant.ListActiveMembershipsForUser(s.db.WithContext(ctx), userID)
	if err != nil {
		return []tenant.LoginTenantCandidate{}
	}
	return candidates
}
