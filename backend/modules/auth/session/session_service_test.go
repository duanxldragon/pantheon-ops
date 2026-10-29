package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// switchLoaderStub returns the subject and per-tenant role keys without touching
// the IAM modules. The roles map models what GetUserRolesForTenant would read
// from system_user_role.
type switchLoaderStub struct {
	rolesForTenant map[uint64][]string
}

func (s switchLoaderStub) GetUserByID(userID uint64) (*UserRef, error) {
	return &UserRef{ID: userID, Username: "switcher"}, nil
}

func (s switchLoaderStub) GetUserRoles(uint64) ([]string, error) { return nil, nil }

func (s switchLoaderStub) GetUserRolesForTenant(_, tenantID uint64) ([]string, error) {
	return s.rolesForTenant[tenantID], nil
}

type switchIssuerStub struct{}

func (switchIssuerStub) IssueTokenPairWithContext(_ context.Context, _ uint64, _ string, _ []string, _ *SystemUserSession) (*authtoken.Pair, error) {
	return &authtoken.Pair{AccessToken: "at", RefreshToken: "rt"}, nil
}

// newSwitchHarness seeds the multi-mode switch environment: one active target
// tenant, one active membership for user 42, and one live session for that user.
func newSwitchHarness(t *testing.T) *Service {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}, &tenant.Membership{}, &tenant.Tenant{}); err != nil {
		t.Fatalf("migrate switch tables: %v", err)
	}
	if err := db.Exec(
		"CREATE TABLE IF NOT EXISTS system_setting (" +
			"tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, " +
			"setting_key VARCHAR(128) NOT NULL, " +
			"setting_value VARCHAR(512), " +
			"PRIMARY KEY (tenant_id, setting_key))",
	).Error; err != nil {
		t.Fatalf("create system_setting: %v", err)
	}
	if err := db.Exec(
		"INSERT INTO system_setting (tenant_id, setting_key, setting_value) VALUES (0, ?, ?)",
		tenant.FeatureFlagSettingKey, tenant.ModeMulti,
	).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if err := db.Create(&tenant.Tenant{ID: 202, Code: "beta", Name: "beta", Status: tenant.TenantStatusActive}).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := db.Create(&tenant.Membership{TenantID: 202, UserID: 42, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	now := time.Now().Add(time.Hour)
	if err := db.Create(&SystemUserSession{
		SessionID:        "sess-switch-1",
		UserID:           42,
		TenantID:         0,
		RefreshJTI:       "jti-switch-1",
		RefreshExpiresAt: now,
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return NewService(db, tenantScopePolicyStub{}, switchLoaderStub{}, switchIssuerStub{})
}

// TestSwitchSessionTenant_RefusesTargetWithoutRoles is the W4 refusal (W3
// decision A, TC-01): switching into a tenant where the subject holds a
// membership but no role binding would re-issue a session with an empty role
// set — the blank console. The switch must refuse instead.
func TestSwitchSessionTenant_RefusesTargetWithoutRoles(t *testing.T) {
	svc := newSwitchHarness(t)

	_, err := svc.SwitchSessionTenant(context.Background(), "sess-switch-1", 42, 202)
	if !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("switch without roles err = %v, want ErrTenantForbidden", err)
	}
}

// TestSwitchSessionTenant_AllowsTargetWithRoles pins the positive branch of the
// same predicate: with a role binding present the switch proceeds and re-issues
// the pair against the new claim.
func TestSwitchSessionTenant_AllowsTargetWithRoles(t *testing.T) {
	// Reuse the harness but swap in a loader that resolves roles for 202.
	svc := newSwitchHarness(t)
	svc.loader = switchLoaderStub{rolesForTenant: map[uint64][]string{202: {"tenant_admin"}}}

	pair, err := svc.SwitchSessionTenant(context.Background(), "sess-switch-1", 42, 202)
	if err != nil {
		t.Fatalf("switch with roles: %v", err)
	}
	if pair == nil || pair.AccessToken == "" {
		t.Fatalf("pair = %+v, want re-issued access token", pair)
	}
}
