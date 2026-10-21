package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"

	"github.com/redis/go-redis/v9"
)

// tenantScopePolicyStub satisfies ListAllSessions' policy read; revocation
// paths never consult it.
type tenantScopePolicyStub struct{}

func (tenantScopePolicyStub) GetSessionPolicy() AuthRuntimePolicy {
	return AuthRuntimePolicy{SessionIdleMinutes: 30, SessionRetentionDays: 90}
}

// newTenantScopeHarness builds a full DB+Redis environment with a wired policy
// provider so admin list/count calls work (loader/issuer stay nil — the F02/F03
// paths under test only touch db and Redis).
func newTenantScopeHarness(t *testing.T) (*Service, *redis.Client) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	// ListAllSessions LEFT JOINs system_user for the username column; create
	// a minimal stand-in (session package does not own the user model).
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user (id BIGINT UNSIGNED PRIMARY KEY, username VARCHAR(64), nickname VARCHAR(64))").Error; err != nil {
		t.Fatalf("create system_user fixture: %v", err)
	}
	rdb := testredis.Open(t)
	oldRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = oldRDB })
	return NewService(db, tenantScopePolicyStub{}, nil, nil), rdb
}

// seedTenantRevocationCase inserts one active session row stamped with the
// given tenant plus the matching Redis access/refresh pair (same shape as
// seedRevocationCase, with tenant ownership for F03 scenarios).
func seedTenantRevocationCase(t *testing.T, svc *Service, rdb *redis.Client, sessionID string, userID, tenantID uint64) {
	t.Helper()
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	if err := svc.db.Create(&SystemUserSession{
		SessionID:        sessionID,
		UserID:           userID,
		TenantID:         tenantID,
		RefreshJTI:       "jti-" + sessionID,
		RefreshExpiresAt: now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session %s: %v", sessionID, err)
	}
	ctx := context.Background()
	if err := authtoken.StoreSession(ctx, rdb, "at-"+sessionID, &authtoken.SessionData{
		UserID:    userID,
		Username:  "user",
		SessionID: sessionID,
	}, time.Hour); err != nil {
		t.Fatalf("seed access token for %s: %v", sessionID, err)
	}
	if err := authtoken.StoreRefresh(ctx, rdb, "rt-"+sessionID, userID, sessionID, time.Hour); err != nil {
		t.Fatalf("seed refresh token for %s: %v", sessionID, err)
	}
}

func assertSessionActive(t *testing.T, svc *Service, rdb *redis.Client, sessionID string) {
	t.Helper()
	ctx := context.Background()

	var row SystemUserSession
	if err := svc.db.Where("session_id = ?", sessionID).First(&row).Error; err != nil {
		t.Fatalf("reload session %s: %v", sessionID, err)
	}
	if row.RevokedAt != nil {
		t.Fatalf("expected session %s to stay active", sessionID)
	}
	if val, err := rdb.Get(ctx, authtoken.BlacklistSessionKey(sessionID)).Result(); err == nil && val != "" {
		t.Fatalf("expected no blacklist entry for session %s", sessionID)
	}
	if _, _, err := authtoken.ValidateRefresh(ctx, rdb, "rt-"+sessionID); err != nil {
		t.Fatalf("expected refresh token of session %s to remain valid, got %v", sessionID, err)
	}
}

// F02: self-service revoke must reject a session owned by another user.
func TestRevokeOwnedSession_RejectsForeignUserSession(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "alice-session", 42, 0)

	err := svc.RevokeOwnedSession(7, "user-7-current", "alice-session")
	if err == nil {
		t.Fatal("expected foreign session revocation to be rejected")
	}
	if !errors.Is(err, common.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	assertSessionActive(t, svc, rdb, "alice-session")
}

// F02 documented current-session rule: the self-service path must not revoke
// the current session (logout owns that path).
func TestRevokeOwnedSession_RejectsCurrentSessionTarget(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "my-current-session", 42, 0)

	if err := svc.RevokeOwnedSession(42, "my-current-session", "my-current-session"); err == nil {
		t.Fatal("expected current-session revocation to be rejected on the self-service path")
	}
	assertSessionActive(t, svc, rdb, "my-current-session")
}

// F03: tenant-scoped managers cannot list or count another tenant's sessions.
func TestListAllSessions_IsTenantScoped(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "tenant-a-session", 42, 101)
	seedTenantRevocationCase(t, svc, rdb, "tenant-b-session", 43, 202)

	scoped := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti, ResolvedBy: "subject"})
	page, err := scoped.ListAllSessions(&AdminSessionQuery{})
	if err != nil {
		t.Fatalf("ListAllSessions: %v", err)
	}
	if page.Total != 1 || page.ActiveCount != 1 {
		t.Fatalf("expected only tenant 101 sessions, got total=%d active=%d", page.Total, page.ActiveCount)
	}
	for _, item := range page.Items {
		if item.SessionID != "tenant-a-session" {
			t.Fatalf("tenant scope leaked: saw session %q", item.SessionID)
		}
	}
}

// F03: tenant-scoped managers cannot revoke another tenant's session, and its
// access/refresh tokens must stay fully usable.
func TestRevokeAnySession_IsTenantScoped(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "tenant-a-session", 42, 101)
	seedTenantRevocationCase(t, svc, rdb, "tenant-b-session", 43, 202)

	scoped := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti, ResolvedBy: "subject"})

	// Cross-tenant target: no-op, no leak.
	if err := scoped.RevokeAnySession("tenant-a-current", "tenant-b-session"); err != nil {
		t.Fatalf("cross-tenant revoke must not error loudly, got %v", err)
	}
	assertSessionActive(t, svc, rdb, "tenant-b-session")

	// Own-tenant target still works through the scoped facade.
	if err := scoped.RevokeAnySession("tenant-a-current", "tenant-a-session"); err != nil {
		t.Fatalf("own-tenant revoke: %v", err)
	}
	assertSessionFullyRevoked(t, svc, rdb, "tenant-a-session", 42)
}

// F03: batch revoke only touches the active tenant's sessions and must not
// blacklist another tenant's access tokens.
func TestBatchRevokeSessions_IsTenantScoped(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "tenant-a-batch", 42, 101)
	seedTenantRevocationCase(t, svc, rdb, "tenant-b-batch", 43, 202)

	scoped := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti, ResolvedBy: "subject"})
	revoked, err := scoped.BatchRevokeSessions("tenant-a-current", []string{"tenant-a-batch", "tenant-b-batch"})
	if err != nil {
		t.Fatalf("BatchRevokeSessions: %v", err)
	}
	if revoked != 1 {
		t.Fatalf("expected only 1 in-tenant revocation, got %d", revoked)
	}
	assertSessionFullyRevoked(t, svc, rdb, "tenant-a-batch", 42)
	assertSessionActive(t, svc, rdb, "tenant-b-batch")
}

// F03 explicit platform exception: a compat (platform-global) context keeps
// the unfiltered view and can revoke across tenants.
func TestSessionAdminOps_CompatSeesAllTenants(t *testing.T) {
	svc, rdb := newTenantScopeHarness(t)
	seedTenantRevocationCase(t, svc, rdb, "tenant-a-compat", 42, 101)
	seedTenantRevocationCase(t, svc, rdb, "tenant-b-compat", 43, 202)

	platform := svc.WithTenantContext(&tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat, ResolvedBy: "compat-fallback"})
	page, err := platform.ListAllSessions(&AdminSessionQuery{})
	if err != nil {
		t.Fatalf("ListAllSessions: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("platform operator must see both tenants, got total=%d", page.Total)
	}

	revoked, err := platform.BatchRevokeSessions("platform-current", []string{"tenant-a-compat", "tenant-b-compat"})
	if err != nil {
		t.Fatalf("platform batch revoke: %v", err)
	}
	if revoked != 2 {
		t.Fatalf("platform operator must revoke both tenants, got %d", revoked)
	}
	assertSessionFullyRevoked(t, svc, rdb, "tenant-a-compat", 42)
	assertSessionFullyRevoked(t, svc, rdb, "tenant-b-compat", 43)
}
