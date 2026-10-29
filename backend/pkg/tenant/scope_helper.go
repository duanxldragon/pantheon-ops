package tenant

import "gorm.io/gorm"

// Scoper is the embeddable per-request tenant context shared by the system
// and auth services (contract §3.3). It replaces the per-service trio — the
// bound shallow clone, the read filter and the write owner — with one tested
// implementation on top of WithTenantScope/WithTenantWrite:
//
//   - nil context: no read filter, owner 0 (platform-internal callers that
//     intentionally want the unfiltered service);
//   - compat: read filter tenant_id = 0, owner 0 (compat data is exactly the
//     tenant_id=0 population; pinning keeps multi-era rows invisible after a
//     downgrade);
//   - multi: read filter tenant_id = ctx.TenantID, owner ctx.TenantID.
//
// Collapsing compat into nil was the exact defect fixed by the 2026-10-24
// packet, so the three contexts must never converge.
//
// Each service keeps its own WithTenantContext because the bound clone must
// return the concrete service type; it sets TenantCtx on a shallow copy so a
// request-scoped view can never mutate the shared receiver.
type Scoper struct {
	// TenantCtx is the per-request tenant context, bound by the service's
	// WithTenantContext from the Gin context. Do not set it ad hoc.
	TenantCtx *Context
}

// Scoped applies the tenant read filter to a query (contract §3.3).
func (s Scoper) Scoped(db *gorm.DB) *gorm.DB {
	return db.Scopes(WithTenantScope(s.TenantCtx))
}

// Scope returns the bare scope function for query builders that compose
// several scopes into one Scopes(...) call.
func (s Scoper) Scope() func(*gorm.DB) *gorm.DB {
	return WithTenantScope(s.TenantCtx)
}

// TenantOwnerID is the tenant a new row belongs to (write path: from the
// context, never from the request body — contract §3.3). Compat/platform => 0.
func (s Scoper) TenantOwnerID() uint64 {
	return OwnerTenantID(s.TenantCtx)
}

// OwnerTenantID is the function form of TenantOwnerID for callers that hold
// the context directly instead of embedding a Scoper.
func OwnerTenantID(ctx *Context) uint64 {
	if ctx == nil || !ctx.IsMulti() {
		return 0
	}
	return ctx.TenantID
}
