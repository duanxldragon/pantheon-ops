package authtoken

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/redis/go-redis/v9"
)

func TestConsumeRefreshHasOneWinnerAcrossClients(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBAuthToken)
	options := *rdb.Options()
	second := redis.NewClient(&options)
	t.Cleanup(func() { second.Close() })
	token := NewRefreshToken()
	ctx := context.Background()
	if err := StoreRefresh(ctx, rdb, token, 42, "consume-session", time.Hour); err != nil {
		t.Fatal(err)
	}
	var won atomic.Int32
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			client := rdb
			if i%2 != 0 {
				client = second
			}
			uid, sid, err := ConsumeRefresh(ctx, client, token)
			if err == nil {
				if uid != 42 || sid != "consume-session" {
					t.Errorf("incorrect claims")
				}
				won.Add(1)
			} else if !errors.Is(err, ErrNotFound) {
				t.Errorf("unexpected consume error: %v", err)
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if won.Load() != 1 {
		t.Fatalf("successful consumers=%d, want one", won.Load())
	}
}

func TestConsumeRefreshPreservesNewerSessionIndex(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBAuthToken)
	ctx := context.Background()
	old, next := NewRefreshToken(), NewRefreshToken()
	for _, tok := range []string{old, next} {
		if err := StoreRefresh(ctx, rdb, tok, 42, "index-session", time.Hour); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := ConsumeRefresh(ctx, rdb, old); err != nil {
		t.Fatal(err)
	}
	if err := RevokeSessionRefresh(ctx, rdb, "index-session"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateRefresh(ctx, rdb, next); !errors.Is(err, ErrNotFound) {
		t.Fatalf("newer token survived revocation: %v", err)
	}
}

func TestConsumeRefreshFailsClosed(t *testing.T) {
	rdb := testredis.OpenDB(t, testredis.DBAuthToken)
	ctx := context.Background()
	if _, _, err := ConsumeRefresh(ctx, nil, "token"); !errors.Is(err, ErrStoreNotInitialized) {
		t.Fatal(err)
	}
	if err := rdb.Set(ctx, RefreshKey("bad-json"), "{", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ConsumeRefresh(ctx, rdb, "bad-json"); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	if _, _, err := ValidateRefresh(ctx, rdb, "bad-json"); !errors.Is(err, ErrNotFound) {
		t.Fatal("invalid consumed token was restored")
	}
	token := NewRefreshToken()
	if err := StoreRefresh(ctx, rdb, token, 42, "failure-session", time.Hour); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := ConsumeRefresh(canceled, rdb, token); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	options := *rdb.Options()
	closed := redis.NewClient(&options)
	closed.Close()
	if _, _, err := ConsumeRefresh(ctx, closed, token); err == nil {
		t.Fatal("closed Redis client returned claims")
	}
	if _, _, err := ConsumeRefresh(ctx, rdb, token); err != nil {
		t.Fatalf("failure before execution unexpectedly consumed token: %v", err)
	}
}
