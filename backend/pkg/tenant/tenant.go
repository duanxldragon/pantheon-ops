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

// Context is the request-scoped, read-only tenant context (contract §3.2).
type Context struct {
	// TenantID is the resolved tenancy for the request: >0 tenant-private,
	// 0 platform/global (compat mode always resolves here).
	TenantID uint64
	// Mode is the active feature-flag mode: "compat" or "multi".
	Mode string
	// ResolvedBy records the resolution source for audit/debug:
	// "header" | "subject" | "compat-fallback".
	ResolvedBy string
}

// IsMulti reports whether the request runs in multi-tenant mode.
func (c *Context) IsMulti() bool { return c != nil && c.Mode == ModeMulti }

// HeaderTenantID is the explicit tenant override header (platform ops only,
// requires platform permission + audit per contract §3.1).
const HeaderTenantID = "X-Tenant-Id"

// ResolveForCanary builds the request tenant context from trusted sources only:
// the authenticated subject is authoritative; the explicit header is honored
// only when a platform-scoped resolver grants it. This canary keeps resolution
// minimal: subject claim > compat fallback. The header path exists for
// platform-admin tooling and is validated by the caller-injected checker.
//
// subjectTenantID: tenant claim from the auth session (0 = none).
// headerTenantID: raw X-Tenant-Id value ("" = absent).
// headerAllowed: whether the authenticated subject may use the header override.
func ResolveForCanary(mode, subjectTenantID, headerTenantID string, headerAllowed bool) (*Context, error) {
	normalizedMode := strings.TrimSpace(mode)
	if normalizedMode != ModeMulti {
		// compat: everything is platform-global; no membership checks (contract §6).
		return &Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat, ResolvedBy: "compat-fallback"}, nil
	}

	if raw := strings.TrimSpace(headerTenantID); raw != "" {
		if !headerAllowed {
			return nil, ErrTenantForbidden
		}
		id, err := parseTenantID(raw)
		if err != nil {
			return nil, ErrTenantForbidden
		}
		return &Context{TenantID: id, Mode: ModeMulti, ResolvedBy: "header"}, nil
	}

	if raw := strings.TrimSpace(subjectTenantID); raw != "" {
		id, err := parseTenantID(raw)
		if err != nil {
			return nil, ErrTenantForbidden
		}
		return &Context{TenantID: id, Mode: ModeMulti, ResolvedBy: "subject"}, nil
	}

	// multi mode with no trusted tenancy: deny-by-default (contract §3.2).
	return nil, ErrTenantContextMissing
}

func parseTenantID(raw string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil || id == PlatformGlobalTenantID {
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
	value := strings.TrimSpace(loadFn())
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
//   - ctx nil or compat mode: no filter — global rows only semantics preserved
//     (compat data is exactly the tenant_id=0 population).
//   - multi mode: hard filter tenant_id = ctx.TenantID (0 excluded on purpose:
//     a multi-mode subject never reads platform-global rows through this scope).
func WithTenantScope(ctx *Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !ctx.IsMulti() {
			return db
		}
		return db.Where("tenant_id = ?", ctx.TenantID)
	}
}

// WithTenantWrite stamps tenant_id on create paths (contract §3.3: ownership
// never comes from the request body). No-op under compat (column default 0).
func WithTenantWrite(ctx *Context) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if !ctx.IsMulti() {
			return db
		}
		return db.Set("gorm:tenant_id", ctx.TenantID)
	}
}
