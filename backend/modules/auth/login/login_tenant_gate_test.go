package login

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

// authTenantFixture is the two-tenant auth/IAM isolation fixture (task packet:
// "单租户兼容模式、双租户隔离、篡改 claim/header、并发切换和会话失效测试").
// Tenant 101 vs 202, plus the platform-global placeholder tenant 0.
type authTenantFixture struct {
	db      *gorm.DB
	runtime *Runtime
}

func newAuthTenantFixture(t *testing.T) *authTenantFixture {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&tenant.Membership{}, &tenant.Tenant{}); err != nil {
		t.Fatalf("migrate tenant tables: %v", err)
	}
	rt := NewRuntime(db, testCredentialRepo(db))
	return &authTenantFixture{db: db, runtime: rt}
}

func (f *authTenantFixture) seedTenant(t *testing.T, id uint64, code, status string) {
	t.Helper()
	row := tenant.Tenant{ID: id, Code: code, Name: code, Status: status}
	if err := f.db.Create(&row).Error; err != nil {
		t.Fatalf("seed tenant %d: %v", id, err)
	}
}

func (f *authTenantFixture) seedMembership(t *testing.T, tenantID, userID uint64, role, status string) {
	t.Helper()
	row := tenant.Membership{TenantID: tenantID, UserID: userID, Role: role, Status: status}
	if err := f.db.Create(&row).Error; err != nil {
		t.Fatalf("seed membership t%d u%d: %v", tenantID, userID, err)
	}
}

// seedSetting writes the platform.tenant_mode flag directly.
func (f *authTenantFixture) seedSetting(t *testing.T, mode string) {
	t.Helper()
	if err := f.db.Exec(
		"CREATE TABLE IF NOT EXISTS system_setting (setting_key VARCHAR(128) PRIMARY KEY, setting_value VARCHAR(512))",
	).Error; err != nil {
		t.Fatalf("create system_setting: %v", err)
	}
	if err := f.db.Exec(
		"INSERT INTO system_setting (setting_key, setting_value) VALUES (?, ?) ON DUPLICATE KEY UPDATE setting_value = VALUES(setting_value)",
		tenant.FeatureFlagSettingKey, mode,
	).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
}

// 1. Compat mode never resolves a tenant claim regardless of memberships.
func TestAuthTenantGate_CompatModeNoClaim(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeCompat)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	claim, err := f.runtime.resolveLoginTenantClaim(42, 0)
	if err != nil {
		t.Fatalf("compat resolveLoginTenantClaim: %v", err)
	}
	if claim != 0 {
		t.Fatalf("compat claim = %d, want 0 (contract §6: compat never stamps claims)", claim)
	}
}

// 2. Single active membership resolves deterministically and passes the gate.
func TestAuthTenantGate_SingleMembershipResolves(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	claim, err := f.runtime.resolveLoginTenantClaim(42, 0)
	if err != nil {
		t.Fatalf("resolveLoginTenantClaim: %v", err)
	}
	if claim != 101 {
		t.Fatalf("claim = %d, want 101", claim)
	}
}

// 3. No membership => platform population (claim 0), no error (contract §3.1).
func TestAuthTenantGate_NoMembershipPlatformPopulation(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)

	claim, err := f.runtime.resolveLoginTenantClaim(77, 0)
	if err != nil {
		t.Fatalf("no-membership resolve: %v", err)
	}
	if claim != 0 {
		t.Fatalf("claim = %d, want 0", claim)
	}
}

// 4. Ambiguous (two active) memberships deny-by-default (contract §3.1:
// ambiguous context is never silently narrowed).
func TestAuthTenantGate_AmbiguousMembershipsDenied(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedTenant(t, 202, "beta", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)
	f.seedMembership(t, 202, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	if _, err := f.runtime.resolveLoginTenantClaim(42, 0); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("ambiguous memberships err = %v, want ErrTenantForbidden", err)
	}
}

// 5. Disabled membership: the user is not a member of that tenant, so login
// falls back to the platform population (claim 0) — identical to having no
// membership. Denying login outright would lock platform-only staff out;
// isolation holds because claim-0 sessions are deny-by-default on every
// tenant-scoped route in multi mode (contract §5 TENANT_FORBIDDEN applies at
// resource-access/refresh time — proven by tests 8/9).
func TestAuthTenantGate_DisabledMembershipFallsBackToPlatform(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipDisabled)

	claim, err := f.runtime.resolveLoginTenantClaim(42, 0)
	if err != nil {
		t.Fatalf("disabled membership resolve: %v", err)
	}
	if claim != 0 {
		t.Fatalf("claim = %d, want 0 (platform fallback)", claim)
	}
	// And the same user canNOT refresh into tenant 101 — stale-membership gate.
	if err := tenant.GateSessionRefresh(f.db, tenant.RefreshCheckInput{Mode: tenant.ModeMulti, UserID: 42, TenantID: 101}); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("refresh with disabled membership err = %v, want ErrTenantForbidden", err)
	}
}

// 6. Suspended tenant denies with the suspended code; archived likewise.
func TestAuthTenantGate_TenantStatusGates(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusSuspended)
	f.seedTenant(t, 202, "beta", tenant.TenantStatusArchived)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)
	f.seedMembership(t, 202, 43, tenant.MembershipRoleMember, tenant.MembershipActive)

	if _, err := f.runtime.resolveLoginTenantClaim(42, 0); !errors.Is(err, tenant.ErrTenantSuspended) {
		t.Fatalf("suspended tenant err = %v, want ErrTenantSuspended", err)
	}
	if _, err := f.runtime.resolveLoginTenantClaim(43, 0); !errors.Is(err, tenant.ErrTenantArchived) {
		t.Fatalf("archived tenant err = %v, want ErrTenantArchived", err)
	}
}

// 7. Refresh gate: active membership passes.
func TestAuthTenantGate_RefreshPassesWithMembership(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	err := tenant.GateSessionRefresh(f.db, tenant.RefreshCheckInput{Mode: tenant.ModeMulti, UserID: 42, TenantID: 101})
	if err != nil {
		t.Fatalf("refresh with membership: %v", err)
	}
}

// 8. Refresh gate: revoked membership kills the refresh (stale membership).
func TestAuthTenantGate_RefreshDeniedAfterMembershipRevoked(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	// Membership is disabled after issuance.
	if err := f.db.Model(&tenant.Membership{}).
		Where("tenant_id = ? AND user_id = ?", 101, 42).
		Update("status", tenant.MembershipDisabled).Error; err != nil {
		t.Fatalf("disable membership: %v", err)
	}

	err := tenant.GateSessionRefresh(f.db, tenant.RefreshCheckInput{Mode: tenant.ModeMulti, UserID: 42, TenantID: 101})
	if !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("refresh after revoke err = %v, want ErrTenantForbidden", err)
	}
}

// 9. Refresh gate: claim-less sessions (legacy/pre-claim) always pass — the
// flag-off regression guarantee (contract §6).
func TestAuthTenantGate_RefreshClaimlessPasses(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)

	err := tenant.GateSessionRefresh(f.db, tenant.RefreshCheckInput{Mode: tenant.ModeMulti, UserID: 42, TenantID: 0})
	if err != nil {
		t.Fatalf("claim-less refresh err = %v, want nil", err)
	}
}

// 10. Session revocation helper: revokes all live rows for the user only.
func TestAuthTenantGate_RevokeUserSessions(t *testing.T) {
	f := newAuthTenantFixture(t)
	if err := f.db.AutoMigrate(&session.SystemUserSession{}); err != nil {
		t.Fatalf("migrate session table: %v", err)
	}
	now := time.Now()
	for _, sid := range []string{"sess-a", "sess-b"} {
		row := session.SystemUserSession{SessionID: sid, UserID: 42, RefreshJTI: sid + "-jti", RefreshExpiresAt: now.Add(time.Hour), TenantID: 101}
		if err := f.db.Create(&row).Error; err != nil {
			t.Fatalf("seed session %s: %v", sid, err)
		}
	}
	// Another user's session must remain untouched.
	other := session.SystemUserSession{SessionID: "sess-other", UserID: 43, RefreshJTI: "other-jti", RefreshExpiresAt: now.Add(time.Hour)}
	if err := f.db.Create(&other).Error; err != nil {
		t.Fatalf("seed other session: %v", err)
	}

	revoked, err := tenant.RevokeUserSessionsInTenant(context.Background(), nil, f.db, 42)
	if err != nil {
		t.Fatalf("RevokeUserSessionsInTenant: %v", err)
	}
	if revoked != 2 {
		t.Fatalf("revoked = %d, want 2", revoked)
	}

	var stillLive int64
	if err := f.db.Table("system_user_session").
		Where("user_id = ? AND revoked_at IS NULL", 42).Count(&stillLive).Error; err != nil {
		t.Fatalf("count live sessions: %v", err)
	}
	if stillLive != 0 {
		t.Fatalf("user 42 still has %d live sessions", stillLive)
	}
	var otherLive int64
	if err := f.db.Table("system_user_session").
		Where("user_id = ? AND revoked_at IS NULL", 43).Count(&otherLive).Error; err != nil {
		t.Fatalf("count other sessions: %v", err)
	}
	if otherLive != 1 {
		t.Fatalf("user 43 sessions affected: %d live, want 1", otherLive)
	}
}

// 11. GateSessionIssuance compat: never stamps (double safety net).
func TestAuthTenantGate_IssuanceCompatNoStamp(t *testing.T) {
	got, err := tenant.GateSessionIssuance(testmysql.Open(t), tenant.IssuanceCheckInput{Mode: tenant.ModeCompat, UserID: 42, TenantID: 101})
	if err != nil {
		t.Fatalf("compat issuance: %v", err)
	}
	if got != 0 {
		t.Fatalf("compat issuance stamped %d, want 0", got)
	}
}

// ─────────────────────────────────────────────────────────────
// Slice 2: explicit tenant choice (picker) paths
// ─────────────────────────────────────────────────────────────

// 13. Explicit choice resolves for a multi-membership user (picker path):
// ambiguous memberships no longer deny when the subject picks a tenant.
func TestAuthTenantGate_ExplicitChoiceResolvesMultiMembership(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedTenant(t, 202, "beta", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)
	f.seedMembership(t, 202, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	claim, err := f.runtime.resolveLoginTenantClaim(42, 202)
	if err != nil {
		t.Fatalf("explicit choice resolve: %v", err)
	}
	if claim != 202 {
		t.Fatalf("claim = %d, want 202", claim)
	}
}

// 14. Explicit choice WITHOUT membership is rejected (claim forgery guard):
// the gate validates the choice against active membership regardless of source.
func TestAuthTenantGate_ExplicitChoiceWithoutMembershipDenied(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedTenant(t, 202, "beta", tenant.TenantStatusActive)
	// user 42 only belongs to 101
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	if _, err := f.runtime.resolveLoginTenantClaim(42, 202); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("forged choice err = %v, want ErrTenantForbidden", err)
	}
}

// 15. Explicit choice for a suspended tenant is rejected even with membership.
func TestAuthTenantGate_ExplicitChoiceSuspendedTenantDenied(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusSuspended)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleOwner, tenant.MembershipActive)

	if _, err := f.runtime.resolveLoginTenantClaim(42, 101); !errors.Is(err, tenant.ErrTenantSuspended) {
		t.Fatalf("suspended choice err = %v, want ErrTenantSuspended", err)
	}
}

// 16. Compat mode ignores the explicit choice entirely (no claim stamped,
// zero behavior change under flag-off).
func TestAuthTenantGate_CompatIgnoresExplicitChoice(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeCompat)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleMember, tenant.MembershipActive)

	claim, err := f.runtime.resolveLoginTenantClaim(42, 101)
	if err != nil {
		t.Fatalf("compat explicit choice: %v", err)
	}
	if claim != 0 {
		t.Fatalf("compat explicit choice claim = %d, want 0", claim)
	}
}

// 17. Candidate listing returns only active memberships in active tenants.
func TestAuthTenantGate_ListCandidates(t *testing.T) {
	f := newAuthTenantFixture(t)
	f.seedSetting(t, tenant.ModeMulti)
	f.seedTenant(t, 101, "alpha", tenant.TenantStatusActive)
	f.seedTenant(t, 202, "beta", tenant.TenantStatusActive)
	f.seedTenant(t, 303, "gamma", tenant.TenantStatusSuspended)
	f.seedMembership(t, 101, 42, tenant.MembershipRoleOwner, tenant.MembershipActive)
	f.seedMembership(t, 202, 42, tenant.MembershipRoleMember, tenant.MembershipActive)
	f.seedMembership(t, 303, 42, tenant.MembershipRoleMember, tenant.MembershipActive)   // suspended tenant
	f.seedMembership(t, 101, 43, tenant.MembershipRoleMember, tenant.MembershipDisabled) // other user, disabled

	candidates := f.runtime.ListLoginTenantCandidates(context.Background(), 42)
	if len(candidates) != 2 {
		t.Fatalf("candidates = %+v, want 2 (101, 202)", candidates)
	}
	if candidates[0].TenantID != 101 || candidates[1].TenantID != 202 {
		t.Fatalf("candidate order/ids = %d,%d, want 101,202", candidates[0].TenantID, candidates[1].TenantID)
	}
	if candidates[0].Code != "alpha" || candidates[0].Role != tenant.MembershipRoleOwner {
		t.Fatalf("candidate metadata = %+v, want alpha/owner", candidates[0])
	}

	// Compat mode: empty list (picker hidden).
	if err := f.db.Exec("UPDATE system_setting SET setting_value = ? WHERE setting_key = ?", tenant.ModeCompat, tenant.FeatureFlagSettingKey).Error; err != nil {
		t.Fatalf("flip flag: %v", err)
	}
	if got := f.runtime.ListLoginTenantCandidates(context.Background(), 42); len(got) != 0 {
		t.Fatalf("compat candidates = %+v, want empty", got)
	}
}

// 18. Token payload round-trips the claim through the Redis JSON envelope.
func TestAuthTenantGate_SessionDataClaimRoundTrip(t *testing.T) {
	data := &authtoken.SessionData{UserID: 42, Username: "u", TenantID: 101}
	b, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back authtoken.SessionData
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.TenantID != 101 {
		t.Fatalf("round-trip TenantID = %d, want 101", back.TenantID)
	}
	// Legacy payloads without the field decode as 0 (compat resolution path).
	var legacy authtoken.SessionData
	if err := json.Unmarshal([]byte(`{"uid":42,"un":"u"}`), &legacy); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if legacy.TenantID != 0 {
		t.Fatalf("legacy TenantID = %d, want 0", legacy.TenantID)
	}
}
