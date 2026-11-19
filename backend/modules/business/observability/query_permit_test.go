package observability

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/redis/go-redis/v9"
)

func openQueryTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := strings.TrimSpace(os.Getenv("PANTHEON_OBSERVABILITY_TEST_REDIS_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:26316"
	}
	// This package owns its Redis instance; its cleanup must not flush a generic
	// test database or inherit credentials from the application Redis instance.
	t.Setenv("PANTHEON_TEST_REDIS_REQUIRED", "1")
	t.Setenv("PANTHEON_TEST_REDIS_ADDR", addr)
	t.Setenv("PANTHEON_TEST_REDIS_PASSWORD", os.Getenv("PANTHEON_OBSERVABILITY_TEST_REDIS_PASSWORD"))
	t.Setenv("PANTHEON_REDIS_PASSWORD", "")
	t.Setenv("REDIS_PASSWORD", "")
	return testredis.OpenDB(t, testredis.DBDefault)
}

func TestQueryPermitActualRedisActorAndTenantCaps(t *testing.T) {
	rdb := openQueryTestRedis(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tenant := uint64(time.Now().UnixNano())
	var permits []*QueryPermit
	t.Cleanup(func() {
		for _, p := range permits {
			if err := p.Release(); err != nil {
				t.Error(err)
			}
		}
	})
	for actor := uint64(1); actor <= 4; actor++ {
		for i := 0; i < 2; i++ {
			p, err := AcquireQueryPermit(ctx, rdb, QueryOwner{TenantID: tenant, ActorID: actor})
			if err != nil {
				t.Fatal(err)
			}
			permits = append(permits, p)
		}
	}
	if _, err := AcquireQueryPermit(ctx, rdb, QueryOwner{TenantID: tenant, ActorID: 1}); err == nil {
		t.Fatal("actor cap bypassed")
	}
	if _, err := AcquireQueryPermit(ctx, rdb, QueryOwner{TenantID: tenant, ActorID: 5}); err == nil {
		t.Fatal("tenant cap bypassed")
	}
	if err := permits[0].Release(); err != nil {
		t.Fatal(err)
	}
	p, err := AcquireQueryPermit(ctx, rdb, QueryOwner{TenantID: tenant, ActorID: 5})
	if err != nil {
		t.Fatal(err)
	}
	permits = append(permits, p)
	if err := permits[0].Release(); err != nil {
		t.Fatal("repeat release failed")
	}
	if _, err := AcquireQueryPermit(ctx, rdb, QueryOwner{TenantID: tenant, ActorID: 6}); err == nil {
		t.Fatal("repeat release removed another worker")
	}
	if _, err := AcquireQueryPermit(context.Background(), rdb, QueryOwner{TenantID: tenant, ActorID: 9}); err == nil {
		t.Fatal("unbounded provider lifetime accepted")
	}
}
