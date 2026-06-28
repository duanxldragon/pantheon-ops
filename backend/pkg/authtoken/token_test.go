package authtoken

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"pantheon-ops/backend/pkg/testredis"
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
