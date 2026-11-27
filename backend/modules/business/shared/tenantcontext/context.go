package tenantcontext

import (
	"context"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

type contextKey struct{}

func With(ctx context.Context, tenantContext *tenant.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if tenantContext == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, tenantContext)
}

func From(ctx context.Context) *tenant.Context {
	if ctx == nil {
		return nil
	}
	tenantContext, _ := ctx.Value(contextKey{}).(*tenant.Context)
	return tenantContext
}
