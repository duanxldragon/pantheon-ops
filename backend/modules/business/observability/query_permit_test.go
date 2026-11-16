package observability

import (
	"context"
	"github.com/redis/go-redis/v9"
	"testing"
	"time"
)

func TestQueryPermitActualRedisActorAndTenantCaps(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:26316"})
	t.Cleanup(func() { _ = rdb.Close() })
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skip("isolated Redis reference missing")
	}
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
