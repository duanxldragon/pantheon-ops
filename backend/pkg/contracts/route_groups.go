package contracts

import (
	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ProtectedGroup is the standard entry point for a business route group:
// the chain is TokenAuth -> Casbin and nothing else.
//
// It deliberately does NOT install a tenant-context middleware. That is a
// contract requirement, not an oversight: TENANT_CONTRACT_V1 §2.6 records that
// the business domain (modules/business/*) does not participate in tenant
// isolation in V1 — business tables carry no tenant_id column, the generator
// explicitly rejects the "tenant" data scope (internal/scaffold/contract.go),
// and tenant.WithTenantScope(nil) would filter nothing even if it ran.
//
// Do not "fix" this by appending middleware.TenantContextMiddleware here: it
// isolates nothing, and in multi mode it turns every business request into
// TENANT_CONTEXT_MISSING. Changing this chain is a contract change, not a bug
// fix; route_groups_test.go pins the chain and fails first.
func ProtectedGroup(r *gin.RouterGroup, path string) gin.IRoutes {
	return ProtectedGroupWithRedis(r, path, database.RDB)
}

// ProtectedGroupWithRedis is ProtectedGroup with an explicit Redis client for
// token-revocation and session-cache lookups. Same chain: TokenAuth -> Casbin.
//
// Same contract boundary as ProtectedGroup: no tenant-context middleware by
// design (TENANT_CONTRACT_V1 §2.6). DataScopedGroup builds on this function, so
// the boundary covers it as well.
func ProtectedGroupWithRedis(r *gin.RouterGroup, path string, rdb *redis.Client) gin.IRoutes {
	return r.Group(path).
		Use(middleware.TokenAuthMiddleware(rdb)).
		Use(middleware.CasbinMiddleware())
}

// DataScopedGroup appends the IAM data-scope middleware to ProtectedGroup:
// TokenAuth -> Casbin -> DataScope.
//
// DataScope is orthogonal to tenancy: it narrows rows by self/department scope,
// not by tenant. The group still installs no tenant-context middleware, per
// TENANT_CONTRACT_V1 §2.6, and business tables have no tenant_id to filter on.
func DataScopedGroup(r *gin.RouterGroup, path string, db *gorm.DB) gin.IRoutes {
	return ProtectedGroup(r, path).
		Use(middleware.DataScopeMiddleware(db))
}

// SystemGroupOptions declares the optional middleware of an authenticated
// /system route group.
//
// The mandatory prefix is always TokenAuth -> PlatformScopeTenantContext ->
// Casbin. On top of it, the system consoles only ever needed three variations,
// which were hand-written at every registration site in
// modules/system/system_modules.go before audit ARCH-01:
//
//	Capability:  platform-console gate (tenant console, i18n console, menu writes)
//	DataScope:   IAM row-level scope (user list/export, dept-scoped rows)
//	RefreshSync: cross-instance refresh signal on write paths
//
// The options are explicit at the call site on purpose: an omitted option is a
// permission decision (no DataScope means unscoped rows), so it must be
// readable where the route is declared rather than inherited from the previous
// block.
type SystemGroupOptions struct {
	// TenantModeLoader resolves platform.tenant_mode for the tenant-context
	// middleware; nil is tolerated by the middleware (compat behaviour).
	TenantModeLoader *tenant.ModeLoader
	// DB backs both the tenant-scope and the data-scope lookups.
	DB *gorm.DB
	// RDB overrides the global Redis client for token lookups (tests).
	RDB *redis.Client
	// Capability requires a platform capability in addition to Casbin.
	Capability bool
	// DataScope applies the IAM self/department/all row filter.
	DataScope bool
	// RefreshSync is the module's refresh middleware; nil installs nothing.
	RefreshSync gin.HandlerFunc
}

// WithCapability returns a copy that also requires a platform capability.
func (o SystemGroupOptions) WithCapability() SystemGroupOptions {
	o.Capability = true
	return o
}

// WithDataScope returns a copy that also applies the IAM row-level data scope.
func (o SystemGroupOptions) WithDataScope() SystemGroupOptions {
	o.DataScope = true
	return o
}

// redisClient falls back to the process-wide client so call sites do not have
// to pass database.RDB explicitly.
func (o SystemGroupOptions) redisClient() *redis.Client {
	if o.RDB != nil {
		return o.RDB
	}
	return database.RDB
}

// SystemTokenGroup installs TokenAuth only.
//
// The refresh-sync state endpoint needs to be authenticated but is not a
// console: it deliberately gets no tenant-context and no Casbin middleware, and
// widening this chain would change how it resolves tenant context in multi
// mode.
func SystemTokenGroup(r *gin.RouterGroup, path string, rdb *redis.Client) gin.IRoutes {
	if rdb == nil {
		rdb = database.RDB
	}
	return r.Group(path).Use(middleware.TokenAuthMiddleware(rdb))
}

// SystemAuthGroup installs TokenAuth -> PlatformScopeTenantContext, without
// Casbin. Used by the upload paths: they resolve a tenant context (the object
// key prefix is derived from it) but are not route-authorised per operation.
func SystemAuthGroup(r *gin.RouterGroup, path string, opts SystemGroupOptions) gin.IRoutes {
	return r.Group(path).
		Use(middleware.TokenAuthMiddleware(opts.redisClient())).
		Use(middleware.PlatformScopeTenantContextMiddleware(opts.TenantModeLoader, opts.DB))
}

// SystemProtectedGroup installs the full system console chain:
// TokenAuth -> PlatformScopeTenantContext -> Casbin, then the optional
// Capability / DataScope gates, then the optional RefreshSync middleware, in
// that order.
//
// Order is a security property, not a style choice: Casbin authorises the route
// before DataScope narrows rows, and RefreshSync runs last so a refused request
// never broadcasts a refresh signal.
func SystemProtectedGroup(r *gin.RouterGroup, path string, opts SystemGroupOptions) gin.IRoutes {
	group := SystemAuthGroup(r, path, opts).
		Use(middleware.CasbinMiddleware())
	if opts.Capability {
		group = group.Use(middleware.PlatformCapabilityMiddleware())
	}
	if opts.DataScope {
		group = group.Use(middleware.DataScopeMiddleware(opts.DB))
	}
	if opts.RefreshSync != nil {
		group = group.Use(opts.RefreshSync)
	}
	return group
}
