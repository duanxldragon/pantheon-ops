package contracts

import (
	"context"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type businessModeStub struct{}

func (*businessModeStub) ReadMode(context.Context) (string, error) { return "multi", nil }

type businessReaderStub struct{}

func (*businessReaderStub) RefreshPrincipal(_ context.Context, id businessauth.Identity) (businessauth.Principal, error) {
	return businessauth.Principal{Identity: id}, nil
}
func (*businessReaderStub) Authorize(context.Context, businessauth.Input) (businessauth.Authorization, error) {
	return businessauth.Authorization{}, nil
}

func TestBusinessGroupFailsClosedAndHasAdditiveChain(t *testing.T) {
	r := gin.New().Group("/api/v1")
	valid := BusinessTenantGroupOptions{DB: &gorm.DB{}, RDB: redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}), ModeReader: &businessModeStub{}, Authorization: &businessReaderStub{}}
	defer valid.RDB.Close()
	for _, field := range []string{"db", "redis", "mode", "auth", "typednil"} {
		opts := valid
		switch field {
		case "db":
			opts.DB = nil
		case "redis":
			opts.RDB = nil
		case "mode":
			opts.ModeReader = nil
		case "auth":
			opts.Authorization = nil
		case "typednil":
			var reader *businessReaderStub
			opts.Authorization = reader
		}
		if _, err := TenantScopedBusinessGroup(r, "/business", opts); err == nil {
			t.Fatal("nil dependency accepted: " + field)
		}
	}
	if _, err := TenantScopedBusinessGroup(nil, "/business", valid); err == nil {
		t.Fatal("nil router accepted")
	}
	routes, err := TenantScopedBusinessGroup(r, "/business", valid)
	if err != nil {
		t.Fatal(err)
	}
	group := routes.(*gin.RouterGroup)
	expected := []string{"BusinessTokenAuthMiddleware", "BusinessTenantContextMiddleware", "BusinessAuthorizationMiddleware"}
	if len(group.Handlers) != len(expected) {
		t.Fatal("wrong opted middleware count")
	}
	for i, handler := range group.Handlers {
		if !strings.Contains(runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name(), expected[i]) {
			t.Fatal("wrong opted chain order")
		}
	}
}

func TestBusinessProviderRegistryStartupOnly(t *testing.T) {
	businessProviders.Lock()
	businessProviders.mode = nil
	businessProviders.authorization = nil
	businessProviders.sealed = false
	businessProviders.Unlock()
	t.Cleanup(func() {
		businessProviders.Lock()
		businessProviders.mode = nil
		businessProviders.authorization = nil
		businessProviders.sealed = false
		businessProviders.Unlock()
	})
	if RegisterBusinessTenantModeReader(nil) == nil || RegisterBusinessAuthorizationReader(nil) == nil || SealBusinessTenantProviders() == nil {
		t.Fatal("nil registration accepted")
	}
	var nilMode *businessModeStub
	if RegisterBusinessTenantModeReader(nilMode) == nil {
		t.Fatal("typed nil mode accepted")
	}
	mode := tenant.ModeReader(&businessModeStub{})
	if err := RegisterBusinessTenantModeReader(mode); err != nil {
		t.Fatal(err)
	}
	if RegisteredBusinessTenantModeReader() != mode {
		t.Fatal("composition seam lost mode")
	}
	if RegisterBusinessTenantModeReader(&businessModeStub{}) == nil {
		t.Fatal("provider replaced")
	}
	if err := RegisterBusinessAuthorizationReader(&businessReaderStub{}); err != nil {
		t.Fatal(err)
	}
	if err := SealBusinessTenantProviders(); err != nil {
		t.Fatal(err)
	}
	if RegisterBusinessAuthorizationReader(&businessReaderStub{}) == nil {
		t.Fatal("late provider registration accepted")
	}
}
