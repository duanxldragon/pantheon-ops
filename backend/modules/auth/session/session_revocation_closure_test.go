package session

import (
	"context"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"

	"github.com/redis/go-redis/v9"
)

// newRevocationClosureHarness builds a full DB+Redis test environment for the
// revocation-closure regression suite. policy/loader/issuer are nil-safe for
// the revoke paths under test (they only touch s.db and Redis).
func newRevocationClosureHarness(t *testing.T) (*Service, *redis.Client) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	rdb := testredis.Open(t)
	oldRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = oldRDB })
	return NewService(db, nil, nil, nil), rdb
}

// seedRevocationCase inserts one active session row and issues the matching
// Redis access/refresh token pair so a revoke path can be verified end-to-end.
func seedRevocationCase(t *testing.T, svc *Service, rdb *redis.Client, sessionID string, userID uint64) {
	t.Helper()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	if err := svc.db.Create(&SystemUserSession{
		SessionID:        sessionID,
		UserID:           userID,
		RefreshJTI:       "jti-" + sessionID,
		RefreshExpiresAt: now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session %s: %v", sessionID, err)
	}
	ctx := context.Background()
	if err := authtoken.StoreSession(ctx, rdb, "at-"+sessionID, &authtoken.SessionData{
		UserID:    userID,
		Username:  "alice",
		SessionID: sessionID,
	}, time.Hour); err != nil {
		t.Fatalf("seed access token for %s: %v", sessionID, err)
	}
	if err := authtoken.StoreRefresh(ctx, rdb, "rt-"+sessionID, userID, sessionID, time.Hour); err != nil {
		t.Fatalf("seed refresh token for %s: %v", sessionID, err)
	}
}

// assertSessionFullyRevoked locks the unified invalidation contract: after a
// revoke path returns success, the DB row is revoked, the refresh token no
// longer validates, and the access token is rejected via the per-session
// blacklist checked by TokenAuthMiddleware.
func assertSessionFullyRevoked(t *testing.T, svc *Service, rdb *redis.Client, sessionID string, userID uint64) {
	t.Helper()
	ctx := context.Background()

	var row SystemUserSession
	if err := svc.db.Where("session_id = ?", sessionID).First(&row).Error; err != nil {
		t.Fatalf("reload session %s: %v", sessionID, err)
	}
	if row.RevokedAt == nil {
		t.Fatalf("expected DB revoked_at set for session %s", sessionID)
	}

	if _, _, err := authtoken.ValidateRefresh(ctx, rdb, "rt-"+sessionID); err == nil {
		t.Fatalf("expected refresh token for session %s to be revoked", sessionID)
	}

	val, err := rdb.Get(ctx, authtoken.BlacklistSessionKey(sessionID)).Result()
	if err != nil || val == "" {
		t.Fatalf("expected session blacklist key for %s, got val=%q err=%v", sessionID, val, err)
	}
	ttl, err := rdb.TTL(ctx, authtoken.BlacklistSessionKey(sessionID)).Result()
	if err != nil || ttl < authtoken.AccessTokenTTL {
		t.Fatalf("expected blacklist TTL >= access token TTL for %s, got %v err=%v", sessionID, ttl, err)
	}

	// The middleware consults the blacklist by session ID; the seeded access
	// token must still exist in Redis (it is rejected by the blacklist, not by
	// absence) so the test proves the blacklist is the blocking factor.
	if _, err := authtoken.ValidateSession(ctx, rdb, "at-"+sessionID); err != nil {
		t.Fatalf("expected access token payload to still exist (blacklist is the blocker): %v", err)
	}
}

func TestRevokeAnySession_InvalidatesAccessAndRefreshTokens(t *testing.T) {
	svc, rdb := newRevocationClosureHarness(t)
	seedRevocationCase(t, svc, rdb, "admin-target", 42)

	if err := svc.RevokeAnySession("admin-current", "admin-target"); err != nil {
		t.Fatalf("RevokeAnySession: %v", err)
	}
	assertSessionFullyRevoked(t, svc, rdb, "admin-target", 42)
}

func TestRevokeAnySession_CurrentSessionProtectionUnchanged(t *testing.T) {
	svc, rdb := newRevocationClosureHarness(t)
	seedRevocationCase(t, svc, rdb, "self-session", 42)

	if err := svc.RevokeAnySession("self-session", "self-session"); err == nil {
		t.Fatal("expected current-session revocation to be rejected")
	}

	ctx := context.Background()
	var row SystemUserSession
	if err := svc.db.Where("session_id = ?", "self-session").First(&row).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if row.RevokedAt != nil {
		t.Fatal("expected current session to stay active in DB")
	}
	if val, err := rdb.Get(ctx, authtoken.BlacklistSessionKey("self-session")).Result(); err == nil && val != "" {
		t.Fatal("expected no blacklist entry for the protected current session")
	}
	if _, _, err := authtoken.ValidateRefresh(ctx, rdb, "rt-self-session"); err != nil {
		t.Fatalf("expected current session refresh token to remain valid, got %v", err)
	}
}

func TestRevokeOwnedSession_InvalidatesAccessAndRefreshTokens(t *testing.T) {
	svc, rdb := newRevocationClosureHarness(t)
	seedRevocationCase(t, svc, rdb, "owned-target", 42)

	if err := svc.RevokeOwnedSession(42, "owned-current", "owned-target"); err != nil {
		t.Fatalf("RevokeOwnedSession: %v", err)
	}
	assertSessionFullyRevoked(t, svc, rdb, "owned-target", 42)
}

func TestBatchRevokeSessions_InvalidatesAllRevokedSessions(t *testing.T) {
	svc, rdb := newRevocationClosureHarness(t)
	seedRevocationCase(t, svc, rdb, "batch-a", 42)
	seedRevocationCase(t, svc, rdb, "batch-b", 42)

	revoked, err := svc.BatchRevokeSessions("batch-current", []string{"batch-a", "batch-b"})
	if err != nil {
		t.Fatalf("BatchRevokeSessions: %v", err)
	}
	if revoked != 2 {
		t.Fatalf("expected 2 revoked, got %d", revoked)
	}
	assertSessionFullyRevoked(t, svc, rdb, "batch-a", 42)
	assertSessionFullyRevoked(t, svc, rdb, "batch-b", 42)
}

func TestRevokeSession_InvalidatesAccessAndRefreshTokens(t *testing.T) {
	svc, rdb := newRevocationClosureHarness(t)
	seedRevocationCase(t, svc, rdb, "logout-target", 42)

	if err := svc.RevokeSession("logout-target"); err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	assertSessionFullyRevoked(t, svc, rdb, "logout-target", 42)
}

// TestRevokeAnySession_RedisFailureReturnsError 锁定 Redis 失败语义：Redis
// 已配置但写入失败时，撤销必须报错，不允许静默形成"已撤销仍可用"的安全
// 假象。DB 侧 revoked_at 已写入（非事务路径），错误把失败显式暴露给调用方。
// RDB == nil 保持既有 no-op 契约：无 Redis 部署下所有 token 校验本身就不可能
// 成功，不存在假象窗口。
func TestRevokeAnySession_RedisFailureReturnsError(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemUserSession{}); err != nil {
		t.Fatalf("migrate sessions: %v", err)
	}
	// 模拟 Redis 故障：指向无监听端口的客户端，写入必然 dial 失败。
	brokenRDB := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: time.Second})
	t.Cleanup(func() { _ = brokenRDB.Close() })
	oldRDB := database.RDB
	database.RDB = brokenRDB
	t.Cleanup(func() { database.RDB = oldRDB })

	svc := NewService(db, nil, nil, nil)
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	if err := svc.db.Create(&SystemUserSession{
		SessionID:        "redis-down",
		UserID:           42,
		RefreshJTI:       "jti",
		RefreshExpiresAt: now.Add(time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	if err := svc.RevokeAnySession("current", "redis-down"); err == nil {
		t.Fatal("expected revoke to fail when Redis blacklist write fails")
	}

	var row SystemUserSession
	if err := svc.db.Where("session_id = ?", "redis-down").First(&row).Error; err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if row.RevokedAt == nil {
		t.Fatal("expected DB revoked_at to be written before the Redis failure surfaced")
	}
}

func TestRevokeSessionArtifacts_EmptySessionIDIsNoop(t *testing.T) {
	if err := RevokeSessionArtifacts(""); err != nil {
		t.Fatalf("expected empty session id noop, got %v", err)
	}
}
