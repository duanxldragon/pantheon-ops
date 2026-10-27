// Package tenant provides the platform-owned tenant context, feature flag,
// and GORM scope helper for the shared-schema multi-tenant MVP (V1 contract).
//
// Contract: docs/contracts/TENANT_CONTRACT_V1.md
// Canary slice: system/config dict resources, gated by the `platform.tenant_mode`
// setting (`compat` default, `multi` opts in per deployment).
//
// Invariants enforced here (contract §2.3/§3):
//   - tenant_id is NOT NULL with 0 reserved for platform/global rows
//   - context resolves once (middleware), is read-only afterwards
//   - business queries against tenant-scoped tables MUST pass through WithTenantScope
//   - missing context in multi mode = deny-by-default (ErrTenantContextMissing)
package tenant

import (
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PlatformGlobalTenantID is reserved for platform/global rows (contract §2.3).
const PlatformGlobalTenantID uint64 = 0

// Mode values for the `platform.tenant_mode` feature flag (contract §6).
const (
	ModeCompat = "compat"
	ModeMulti  = "multi"
)

// Sentinel errors (i18n message keys; classified via errors.Is).
var (
	// ErrTenantContextMissing: multi-mode request without resolvable tenancy —
	// configuration error class, deny-by-default (contract §3.2/§5).
	ErrTenantContextMissing = errors.New("tenant.context.missing")
	// ErrTenantForbidden: subject lacks membership / platform permission for
	// the requested tenant (contract §5).
	ErrTenantForbidden = errors.New("tenant.forbidden")
	// ErrTenantSuspended: tenant master row is suspended (contract §5).
	ErrTenantSuspended = errors.New("tenant.suspended")
	// ErrTenantArchived: tenant master row is archived (contract §5).
	ErrTenantArchived = errors.New("tenant.archived")
)

// ContextKey stores the resolved tenant context in the Gin context.
const ContextKey = "pantheon_tenant_context"

// ResolvedBy values record which trusted source produced the context
// (contract §3.1).
const (
	// ResolvedByHeader: explicit X-Tenant-Id override.
	ResolvedByHeader = "header"
	// ResolvedBySubject: tenant claim stamped into the auth session at login.
	ResolvedBySubject = "subject"
	// ResolvedByCompatFallback: compat mode, everything is platform-global.
	ResolvedByCompatFallback = "compat-fallback"
	// ResolvedByPlatformFallback: platform-scoped route reached by a
	// platform-capable subject with no tenant claim — resolves to the
	// platform-global context (tenant 0) instead of denying.
	ResolvedByPlatformFallback = "platform-fallback"
)

// Context is the request-scoped, read-only tenant context (contract §3.2).
type Context struct {
	// TenantID is the resolved tenancy for the request: >0 tenant-private,
	// 0 platform/global. Both compat mode and the platform-scoped fallback
	// resolve here, and 0 stays a first-class scope in multi mode
	// (contract §2.3/§4).
	TenantID uint64
	// Mode is the active feature-flag mode: "compat" or "multi".
	Mode string
	// ResolvedBy records the resolution source for audit/debug:
	// "header" | "subject" | "compat-fallback" | "platform-fallback".
	ResolvedBy string
}

// IsMulti reports whether the request runs in multi-tenant mode.
func (c *Context) IsMulti() bool { return c != nil && c.Mode == ModeMulti }

// HeaderTenantID is the explicit tenant override header (platform ops only,
// requires platform permission + audit per contract §3.1).
const HeaderTenantID = "X-Tenant-Id"

// ResolveInput carries the caller's tenancy policy for a single request.
// The route side supplies PlatformScoped, the subject side PlatformCapable;
// the two together decide whether a claim-less subject may fall back to the
// platform-global context (contract §3.1/§3.3).
type ResolveInput struct {
	Mode            string
	SubjectTenantID string
	HeaderTenantID  string
	// HeaderAllowed: the subject carries a platform-level grant, so the
	// explicit override may be honoured (contract §3.1 source 1).
	HeaderAllowed bool
	// PlatformScoped: the route group declared the platform-global context as
	// valid for itself (contract §3.3 "平台管理 API 显式声明").
	PlatformScoped bool
	// PlatformCapable: the subject carries a platform-level grant
	// (contract §3.1 "要求平台级权限" / §4 "先全局角色/权限（平台能力）").
	PlatformCapable bool
}

// Resolve builds the request tenant context from trusted sources only, in
// contract §3.1 priority order:
//
//  1. explicit header `X-Tenant-Id` — platform-capable subjects only. The
//     value 0 is accepted here and means "switch to the platform-global
//     scope"; a subject claim never accepts 0 (that would silently widen
//     scope to global).
//  2. subject claim — a numeric tenant > 0 stamped at login.
//  3. platform scope — a route group that declared itself platform-scoped,
//     reached by a platform-capable subject with no claim, resolves to the
//     platform-global context (tenant 0). This is the path that lets tenant
//     subjects and platform/non-tenant subjects share one multi-mode
//     deployment (contract §2.3/§4).
//  4. compat — every request is platform-global.
//
// Anything left over in multi mode is a configuration error and is denied
// (contract §3.2/§5).
func Resolve(in ResolveInput) (*Context, error) {
	if strings.TrimSpace(in.Mode) != ModeMulti {
		return &Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat, ResolvedBy: ResolvedByCompatFallback}, nil
	}

	if raw := strings.TrimSpace(in.HeaderTenantID); raw != "" {
		if !in.HeaderAllowed {
			return nil, ErrTenantForbidden
		}
		id, err := parseTenantID(raw, true)
		if err != nil {
			return nil, ErrTenantForbidden
		}
		return &Context{TenantID: id, Mode: ModeMulti, ResolvedBy: ResolvedByHeader}, nil
	}

	if raw := strings.TrimSpace(in.SubjectTenantID); raw != "" {
		id, err := parseTenantID(raw, false)
		if err != nil {
			return nil, ErrTenantForbidden
		}
		return &Context{TenantID: id, Mode: ModeMulti, ResolvedBy: ResolvedBySubject}, nil
	}

	if in.PlatformScoped && in.PlatformCapable {
		return &Context{TenantID: PlatformGlobalTenantID, Mode: ModeMulti, ResolvedBy: ResolvedByPlatformFallback}, nil
	}

	// multi mode with no trusted tenancy: deny-by-default (contract §3.2).
	return nil, ErrTenantContextMissing
}

// ResolveForCanary keeps the original canary signature: a tenant claim or the
// compat fallback, with no platform-scope declaration. It is retained so the
// canary slice (system/config dict) and its tests stay valid; new call sites
// use Resolve.
func ResolveForCanary(mode, subjectTenantID, headerTenantID string, headerAllowed bool) (*Context, error) {
	return Resolve(ResolveInput{
		Mode:            mode,
		SubjectTenantID: subjectTenantID,
		HeaderTenantID:  headerTenantID,
		HeaderAllowed:   headerAllowed,
	})
}

// parseTenantID parses an explicit tenant id. allowGlobal permits 0, which is
// reserved for platform/global rows (contract §2.3); callers pass true only
// for the explicit header override.
func parseTenantID(raw string, allowGlobal bool) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, errors.New("tenant.id.invalid")
	}
	if id == PlatformGlobalTenantID && !allowGlobal {
		// 0 is not a valid explicit tenant: it would silently widen scope to global.
		return 0, errors.New("tenant.id.invalid")
	}
	return id, nil
}

// FromGin returns the resolved context, or nil when absent.
func FromGin(c *gin.Context) *Context {
	if c == nil {
		return nil
	}
	if val, ok := c.Get(ContextKey); ok {
		if ctx, ok := val.(*Context); ok {
			return ctx
		}
	}
	return nil
}

// SetGin stores the resolved context (middleware-only; business code reads).
func SetGin(c *gin.Context, ctx *Context) { c.Set(ContextKey, ctx) }

// ModeLoader reads the current `platform.tenant_mode` flag value.
// Loaded at most once per TTL to keep the request path free of extra queries.
type ModeLoader struct {
	mu           sync.RWMutex
	cached       string
	loaded       bool
	loadFn       func() string
	nowNano      func() int64
	ttlNano      int64
	lastLoadNano int64
}

// NewModeLoader builds a caching loader; loadFn returns the raw flag value.
func NewModeLoader(loadFn func() string, ttlNano int64) *ModeLoader {
	return &ModeLoader{loadFn: loadFn, nowNano: defaultNowNano, ttlNano: ttlNano}
}

// Load returns the current mode, cached for the loader TTL. Fail-safe: any
// error/empty from loadFn falls back to compat (flag-off = flag-on-free).
func (l *ModeLoader) Load() string {
	l.mu.RLock()
	cached, loaded, fresh := l.cached, l.loaded, l.isFresh()
	l.mu.RUnlock()
	if loaded && fresh {
		return cached
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.loaded && l.isFresh() {
		return l.cached
	}
	loadFn := l.loadFn
	if loadFn == nil {
		value := ModeCompat
		l.cached = value
		l.loaded = true
		l.lastLoadNano = l.nowNano()
		return l.cached
	}
	value := NormalizeFlagValue(loadFn())
	if value != ModeMulti {
		value = ModeCompat
	}
	l.cached = value
	l.loaded = true
	l.lastLoadNano = l.nowNano()
	return l.cached
}

// Invalidate clears the cached mode (config change / tests).
func (l *ModeLoader) Invalidate() {
	l.mu.Lock()
	l.loaded = false
	l.mu.Unlock()
}

func (l *ModeLoader) isFresh() bool {
	return l.ttlNano <= 0 || l.nowNano()-l.lastLoadNano < l.ttlNano
}
func defaultNowNano() int64 { return timeNowNano() }

// WithTenantScope returns the GORM scope injecting `tenant_id = ctx.TenantID`
// for tenant-scoped tables (contract §3.3). Behavior:
//   - ctx nil: no filter — reserved for platform-internal paths (bootstrap,
//     maintenance, background jobs) that run outside any request context.
//   - compat mode: hard filter tenant_id = ctx.TenantID, which Resolve pins to
//     0 for every compat request (compat fallback). Pinning 0 instead of
//     leaving the query unfiltered keeps rows created while the deployment ran
//     in multi mode out of the global view after a downgrade; compat data is
//     exactly the tenant_id=0 population by construction.
//   - multi mode: hard filter tenant_id = ctx.TenantID (0 excluded on purpose:
//     a multi-mode subject never reads platform-global rows through this scope).
//
// The filter column is qualified with the statement's primary table (or its
// alias) whenever one is known. A bare `tenant_id` is ambiguous the moment the
// query joins another tenant-scoped table — e.g. system_user_session JOIN
// system_user, or system_user JOIN system_user_role — so joined read paths must
// be qualified or MySQL rejects them with error 1052.
func WithTenantScope(ctx *Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if ctx == nil {
			return db
		}
		table := ""
		if db != nil && db.Statement != nil {
			table = db.Statement.Table
		}
		return db.Where(qualifyTenantColumn(table)+" = ?", ctx.TenantID)
	}
}

// qualifyTenantColumn resolves the column the tenant filter should target.
// It prefers the statement's explicit table (`.Table("t")`, `.Table("t AS a")`
// or `.Table("t a")`) and falls back to the bare `tenant_id` when the statement
// is model-based (no table resolved yet) or the table is not a plain identifier
// (subquery / raw SQL). The fallback preserves the previous single-table
// behavior, so only joined queries change — and they change from broken to
// correct.
func qualifyTenantColumn(table string) string {
	table = strings.TrimSpace(table)
	if table == "" || strings.ContainsAny(table, ",()") {
		return "tenant_id"
	}
	switch fields := strings.Fields(table); len(fields) {
	case 2:
		table = fields[1] // "system_user u"
	case 3:
		if strings.EqualFold(fields[1], "AS") {
			table = fields[2] // "system_user AS u"
		}
	}
	if table == "" || strings.ContainsAny(table, " \t,()") {
		return "tenant_id"
	}
	return table + ".tenant_id"
}

// WithTenantWrite stamps tenant_id on create paths (contract §3.3: ownership
// never comes from the request body). It mirrors WithTenantScope: a nil
// context is a platform-internal write (no stamp); every request-bound write
// stamps ctx.TenantID — under compat that is always 0, the same value the
// column default produces, so the pin is explicit and downgrade-proof.
func WithTenantWrite(ctx *Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if ctx == nil {
			return db
		}
		return db.Set("gorm:tenant_id", ctx.TenantID)
	}
}
