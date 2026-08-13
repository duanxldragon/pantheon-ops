package authtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"pantheon-base/pkg/testredis"
)

func TestValidateSessionRejectsInvalidJSON(t *testing.T) {
	rdb := testredis.Open(t)
	ctx := context.Background()
	token := "bad-session-token"
	if err := rdb.Set(ctx, SessionKey(token), []byte("{"), time.Minute).Err(); err != nil {
		t.Fatalf("seed invalid session token: %v", err)
	}

	_, err := ValidateSession(ctx, rdb, token)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestValidateRefreshRejectsInvalidJSON(t *testing.T) {
	rdb := testredis.Open(t)
	ctx := context.Background()
	token := "bad-refresh-token"
	if err := rdb.Set(ctx, RefreshKey(token), []byte("{"), time.Minute).Err(); err != nil {
		t.Fatalf("seed invalid refresh token: %v", err)
	}

	_, _, err := ValidateRefresh(ctx, rdb, token)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestValidateOperationRejectsInvalidJSON(t *testing.T) {
	rdb := testredis.Open(t)
	ctx := context.Background()
	token := "bad-operation-token"
	if err := rdb.Set(ctx, OperationKey(token), []byte("{"), time.Minute).Err(); err != nil {
		t.Fatalf("seed invalid operation token: %v", err)
	}

	_, err := ValidateOperation(ctx, rdb, token)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestDeleteSessionPairAllowsNilRedis(t *testing.T) {
	if err := DeleteSessionPair(context.Background(), (*redis.Client)(nil), "access", "refresh"); err != nil {
		t.Fatalf("expected nil redis delete to be ignored, got %v", err)
	}
}

func TestRevokeSessionRefreshCascadesTokenDeletion(t *testing.T) {
	rdb := testredis.Open(t)
	ctx := context.Background()
	const sessionID = "sess-cascade-1"
	token := NewRefreshToken()
	if err := StoreRefresh(ctx, rdb, token, 42, sessionID, time.Minute); err != nil {
		t.Fatalf("store refresh: %v", err)
	}
	if _, _, err := ValidateRefresh(ctx, rdb, token); err != nil {
		t.Fatalf("refresh should be valid before revoke: %v", err)
	}

	if err := RevokeSessionRefresh(ctx, rdb, sessionID); err != nil {
		t.Fatalf("revoke session refresh: %v", err)
	}
	if _, _, err := ValidateRefresh(ctx, rdb, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected refresh token to be deleted after cascade, got %v", err)
	}
	// 幂等：重复吊销与无索引会话都应静默成功。
	if err := RevokeSessionRefresh(ctx, rdb, sessionID); err != nil {
		t.Fatalf("second revoke should be a no-op, got %v", err)
	}
	if err := RevokeSessionRefresh(ctx, rdb, "never-issued"); err != nil {
		t.Fatalf("revoking unknown session should succeed, got %v", err)
	}
	if err := RevokeSessionRefresh(ctx, nil, sessionID); err != nil {
		t.Fatalf("nil redis should be ignored, got %v", err)
	}
}

func TestStoreRefreshRotationKeepsIndexOnLatestToken(t *testing.T) {
	rdb := testredis.Open(t)
	ctx := context.Background()
	const sessionID = "sess-rotate-1"
	oldToken := NewRefreshToken()
	newToken := NewRefreshToken()
	if err := StoreRefresh(ctx, rdb, oldToken, 42, sessionID, time.Minute); err != nil {
		t.Fatalf("store old refresh: %v", err)
	}
	if err := StoreRefresh(ctx, rdb, newToken, 42, sessionID, time.Minute); err != nil {
		t.Fatalf("store rotated refresh: %v", err)
	}
	if err := RevokeSessionRefresh(ctx, rdb, sessionID); err != nil {
		t.Fatalf("revoke session refresh: %v", err)
	}
	if _, _, err := ValidateRefresh(ctx, rdb, newToken); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected latest refresh token to be revoked, got %v", err)
	}
}

func TestTokenStoreOperationsHandleNilRedis(t *testing.T) {
	ctx := context.Background()
	if err := StoreSession(ctx, nil, "access", &SessionData{}, time.Minute); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected session store init error, got %v", err)
	}
	if _, err := ValidateSession(ctx, nil, "access"); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected session validate init error, got %v", err)
	}
	if err := DeleteSession(ctx, nil, "access"); err != nil {
		t.Fatalf("expected nil session delete to be ignored, got %v", err)
	}
	if err := StoreRefresh(ctx, nil, "refresh", 1, "session", time.Minute); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected refresh store init error, got %v", err)
	}
	if _, _, err := ValidateRefresh(ctx, nil, "refresh"); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected refresh validate init error, got %v", err)
	}
	if err := DeleteRefresh(ctx, nil, "refresh"); err != nil {
		t.Fatalf("expected nil refresh delete to be ignored, got %v", err)
	}
	if err := StoreOperation(ctx, nil, "operation", &OperationData{}, time.Minute); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected operation store init error, got %v", err)
	}
	if _, err := ValidateOperation(ctx, nil, "operation"); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatalf("expected operation validate init error, got %v", err)
	}
	if err := DeleteOperation(ctx, nil, "operation"); err != nil {
		t.Fatalf("expected nil operation delete to be ignored, got %v", err)
	}
}
