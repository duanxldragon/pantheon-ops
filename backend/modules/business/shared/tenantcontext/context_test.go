package tenantcontext

import (
	"context"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

func TestWithAndFrom(t *testing.T) {
	value := &tenant.Context{TenantID: 42, Mode: tenant.ModeMulti}
	ctx := With(context.Background(), value)
	if got := From(ctx); got != value {
		t.Fatalf("From(With(ctx, tenant)) = %p, want %p", got, value)
	}
	if got := From(context.Background()); got != nil {
		t.Fatalf("From(unbound ctx) = %v, want nil", got)
	}
}
