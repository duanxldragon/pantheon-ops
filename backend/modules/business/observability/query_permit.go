package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/redis/go-redis/v9"
	"strconv"
	"sync"
	"time"
)

var ErrQueryBusy = errors.New("observability.query_busy")
var queryAdmission = redis.NewScript(`
for _,key in ipairs(KEYS) do
 local kind=redis.call('TYPE',key)['ok']
 if kind~='none' and kind~='zset' then return -1 end
end
local clock=redis.call('TIME')
local now=clock[1]*1000+math.floor(clock[2]/1000)
for _,key in ipairs(KEYS) do redis.call('ZREMRANGEBYSCORE',key,'-inf',now) end
if redis.call('ZCARD',KEYS[1])>=2 or redis.call('ZCARD',KEYS[2])>=8 then return 0 end
for _,key in ipairs(KEYS) do
 redis.call('ZADD',key,now+45000,ARGV[1])
 redis.call('PEXPIRE',key,45000)
end
return 1`)
var queryRelease = redis.NewScript(`
for _,key in ipairs(KEYS) do
 local kind=redis.call('TYPE',key)['ok']
 if kind~='none' and kind~='zset' then return -1 end
end
for _,key in ipairs(KEYS) do redis.call('ZREM',key,ARGV[1]) end
return 1`)

type QueryPermit struct {
	rdb      *redis.Client
	keys     []string
	id       string
	mu       sync.Mutex
	released bool
}

func AcquireQueryPermit(ctx context.Context, rdb *redis.Client, owner QueryOwner) (*QueryPermit, error) {
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 30*time.Second || rdb == nil || owner.ActorID == 0 {
		return nil, ErrWorkspaceDenied
	}
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return nil, ErrQueryBusy
	}
	base := "pantheon:o4:permit:{" + strconv.FormatUint(owner.TenantID, 10) + "}:"
	p := &QueryPermit{rdb: rdb, keys: []string{base + "actor:" + strconv.FormatUint(owner.ActorID, 10), base + "tenant"}, id: hex.EncodeToString(token[:])}
	accepted, err := queryAdmission.Run(ctx, rdb, p.keys, p.id).Int()
	if err != nil || accepted != 1 {
		// Ambiguous admission never starts provider work. Release only our opaque
		// ID; an unavailable store conservatively retains its bounded slot.
		_ = p.Release()
		return nil, ErrQueryBusy
	}
	return p, nil
}

// Release runs only after synchronous provider work has stopped. Its own bounded
// cleanup context survives cancellation of the request and never removes peers.
func (p *QueryPermit) Release() error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.released {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := queryRelease.Run(ctx, p.rdb, p.keys, p.id).Int()
	if err != nil || result != 1 {
		return ErrQueryBusy
	}
	p.released = true
	return nil
}
