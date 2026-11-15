package opsinit

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/login"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type startupOldAuthorizationReader struct{}

func (startupOldAuthorizationReader) RefreshPrincipal(context.Context, businessauth.Identity) (businessauth.Principal, error) {
	return businessauth.Principal{}, businessauth.ErrDenied
}
func (startupOldAuthorizationReader) Authorize(context.Context, businessauth.Input) (businessauth.Authorization, error) {
	return businessauth.Authorization{}, businessauth.ErrDenied
}

func TestOwnerLifecycleGuardPrecedesAllOpsDevelopmentAndSeedEffects(t *testing.T) {
	if mode := os.Getenv("PANTHEON_O4_OPS_STARTUP_CASE"); mode != "" {
		db := testmysql.Open(t)
		// The public runtime receives only an owned synthetic settings table.
		// The lifecycle startup guard itself must issue no lifecycle SQL or DDL.
		if err := db.Exec("CREATE TABLE system_setting (id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT, tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, setting_key VARCHAR(128), setting_value TEXT)").Error; err != nil {
			t.Fatal(err)
		}
		pool, _ := db.DB()
		expected := contracts.ErrBusinessProviders
		switch mode {
		case "missing":
		case "incompatible":
			if err := contracts.RegisterBusinessAuthorizationReader(startupOldAuthorizationReader{}); err != nil {
				t.Fatal(err)
			}
		default:
			provider := login.NewBusinessAuthorizationReader(login.NewRuntime(db, nil), nil, nil)
			if _, ok := provider.(businessauth.OwnerLifecycleReader); !ok {
				t.Fatal("actual public lifecycle capability missing")
			}
			if err := contracts.RegisterBusinessAuthorizationReader(provider); err != nil {
				t.Fatal(err)
			}
			expected = observability.ErrWorkspaceMaintenance
		}
		if mode == "pool1" {
			pool.SetMaxOpenConns(1)
		}
		var target *gorm.DB = db
		if mode == "nil" {
			target = nil
		}
		before, err := db.Migrator().GetTables()
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(before)
		if mode == "closed" {
			if err := pool.Close(); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("PANTHEON_AUTO_MIGRATE", "true")
		gin.SetMode(gin.TestMode)
		router := gin.New()
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			Init(router.Group("/api/v1"), target)
		}()
		failure, ok := recovered.(error)
		if !ok || !errors.Is(failure, expected) || failure.Error() != expected.Error() {
			t.Fatalf("Ops did not refuse at required lifecycle guard: %v", recovered)
		}
		if mode == "closed" {
			if len(router.Routes()) != 0 {
				t.Fatal("closed-pool startup registered routes")
			}
			t.Log("actual closed pool refused with exact maintenance error before development bootstrap, seed or routes")
			return
		}
		after, err := db.Migrator().GetTables()
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(after)
		if !reflect.DeepEqual(before, after) || len(router.Routes()) != 0 {
			t.Fatal("refused Ops startup changed development schema, business seed or routes")
		}
		t.Log("exact startup error; opted-in development bootstrap, business schemas/seeds and routes had zero effects")
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"missing", "incompatible", "nil", "closed", "pool1"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, "-test.run=^TestOwnerLifecycleGuardPrecedesAllOpsDevelopmentAndSeedEffects$", "-test.v")
			command.Env = append(os.Environ(), "PANTHEON_O4_OPS_STARTUP_CASE="+mode)
			output, err := command.CombinedOutput()
			t.Log(string(output))
			if err != nil {
				t.Fatalf("isolated actual Ops startup guard failed: %v", err)
			}
		})
	}
}
