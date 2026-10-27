package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// platformOpsRoleKey is the platform-operations role key the canary slice
// shipped as its platform-level marker. It stays a valid platform-capability
// signal alongside the Casbin-based check below.
const platformOpsRoleKey = "platform_ops"

// TenantContextMiddleware resolves the request tenant context from trusted
// sources (contract §3.1) and stores it read-only in the Gin context.
//
// Trusted sources, in priority order:
//  1. X-Tenant-Id header — only when the subject carries a platform-level
//     grant. The value 0 is accepted here and means "platform-global".
//  2. Subject tenant claim — the `tenantId` value stamped into the auth
//     session at issuance (core auth/iam task) and surfaced by
//     TokenAuthMiddleware. A numeric claim > 0 is trusted because only the
//     issuance gate (pkg/tenant.GateSessionIssuance) can stamp it.
//  3. Compat fallback — flag off resolves to global tenant 0.
//
// This variant serves tenant-scoped routes: a claim-less subject is denied,
// because the route has not declared the platform-global scope as valid for
// itself (contract §3.2/§3.3). Platform-scoped groups use
// PlatformScopeTenantContextMiddleware instead.
//
// modeLoader is the cached flag reader; membershipDB resolves memberships
// (nil-safe: treated as "no memberships exist").
func TenantContextMiddleware(modeLoader *tenant.ModeLoader, membershipDB *gorm.DB) gin.HandlerFunc {
	return tenantContextMiddleware(modeLoader, membershipDB, false)
}

// PlatformScopeTenantContextMiddleware is the variant for route groups that
// declare themselves platform-scoped (contract §3.3: "平台管理 API（显式声明，
// 要求平台权限，落审计）").
//
// On top of the strict behaviour it adds contract §3.1 source 3: a
// platform-capable subject with no tenant claim resolves to the
// platform-global context (tenant 0) instead of being denied. That is what
// lets non-tenant (platform) operators and tenant subjects share a single
// multi-mode deployment — tenant 0 is a first-class scope in both modes
// (contract §2.3/§4) and the fallback stays narrow: tenant 0 only ever reads
// platform/global rows.
//
// Both conditions are required, so a tenant subject can never fall back: the
// route must have declared the platform scope, AND the subject must carry a
// platform-level grant AND belong to no tenant.
func PlatformScopeTenantContextMiddleware(modeLoader *tenant.ModeLoader, membershipDB *gorm.DB) gin.HandlerFunc {
	return tenantContextMiddleware(modeLoader, membershipDB, true)
}

func tenantContextMiddleware(modeLoader *tenant.ModeLoader, membershipDB *gorm.DB, platformScoped bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := tenant.ModeCompat
		if modeLoader != nil {
			mode = modeLoader.Load()
		}

		headerTenant := strings.TrimSpace(c.GetHeader(tenant.HeaderTenantID))
		subjectTenant := subjectTenantClaim(c)

		// The capability probe costs a Casbin lookup, so only run it on the
		// two paths that can consume it.
		needsCapability := headerTenant != "" || (platformScoped && subjectTenant == "")
		platformCapable := false
		if needsCapability {
			platformCapable = subjectHasPlatformCapability(c)
		}

		// Login-scoped routes (self-service endpoints and the tenant switcher)
		// carry no role policy by design (PERMISSION_MODEL §10.3, contract
		// §2.2): every authenticated subject may call them. A claim-less
		// platform subject therefore has no global policy to prove platform
		// capability with, yet it still needs a resolved context — otherwise
		// /auth/me answers `tenant.context.missing` and the console is
		// unusable for platform operators holding narrow roles. This only
		// relaxes the implicit fallback; the explicit header override keeps
		// requiring a real platform grant.
		fallbackCapable := platformCapable
		if platformScoped && subjectTenant == "" && !fallbackCapable && isLoginScopedRoute(c) {
			fallbackCapable = true
		}

		resolved, err := tenant.Resolve(tenant.ResolveInput{
			Mode:            mode,
			SubjectTenantID: subjectTenant,
			HeaderTenantID:  headerTenant,
			HeaderAllowed:   platformCapable,
			PlatformScoped:  platformScoped,
			PlatformCapable: fallbackCapable && isPlatformPopulationSubject(c, membershipDB),
		})
		if err != nil {
			if errors.Is(err, tenant.ErrTenantContextMissing) {
				common.Fail(c, common.CodeError, "tenant.context.missing")
			} else {
				common.Fail(c, common.CodeForbidden, tenantErrorMessageKey(err))
			}
			c.Abort()
			return
		}

		// Explicit override validation (contract §3.1 source 1, §5): the header
		// names a tenant directly, so it must be a real, active tenant. Without
		// this an operator could pin the scope to an id that does not exist and
		// silently read an empty scope, or pin a suspended tenant.
		if resolved.ResolvedBy == tenant.ResolvedByHeader {
			if err := tenant.ValidateExplicitTenant(membershipDB, resolved.TenantID); err != nil {
				common.Fail(c, common.CodeForbidden, tenantErrorMessageKey(err))
				c.Abort()
				return
			}
		}

		tenant.SetGin(c, resolved)
		c.Next()
	}
}

// isLoginScopedRoute reports whether a route is authorized by the login state
// alone rather than by a role policy: the Casbin self-service allowlist plus
// the tenant switcher, whose authority is the membership row (contract §2.2).
// Such routes cannot express "platform capability" as a policy, so the tenant
// fallback must not demand one from a claim-less platform subject.
func isLoginScopedRoute(c *gin.Context) bool {
	if isSelfServiceRoute(c) {
		return true
	}
	switch c.FullPath() {
	case "/api/v1/auth/tenants/switchable", "/api/v1/auth/tenants/current":
		return c.Request.Method == http.MethodGet
	case "/api/v1/auth/tenants/switch":
		return c.Request.Method == http.MethodPost
	default:
		return false
	}
}

// tenantErrorMessageKey maps a tenant sentinel to its i18n message key
// (contract §5).
func tenantErrorMessageKey(err error) string {
	switch {
	case errors.Is(err, tenant.ErrTenantSuspended):
		return "tenant.suspended"
	case errors.Is(err, tenant.ErrTenantArchived):
		return "tenant.archived"
	default:
		return "tenant.forbidden"
	}
}

// isPlatformPopulationSubject reports whether the subject belongs to no tenant,
// which is the claim-less population the issuance gate signs in (contract §3.1
// source 2). It keeps the implicit platform-global fallback off for a subject
// that does hold memberships, so the fallback can never shadow a tenant
// membership; such a subject still reaches the global scope through the
// explicit X-Tenant-Id override. Fails closed when the answer is unavailable.
func isPlatformPopulationSubject(c *gin.Context, membershipDB *gorm.DB) bool {
	if membershipDB == nil {
		// No membership store to consult (canary-style wiring): the capability
		// check alone decides, as it did before this middleware grew a
		// platform-scoped variant.
		return true
	}
	tenantLess, failClosed := tenant.IsPlatformPopulationUser(membershipDB, common.GetUserID(c))
	return !failClosed && tenantLess
}

// PlatformCapabilityMiddleware gates a route group on a platform-level grant
// (contract §3.3: "平台管理 API（显式声明，要求平台权限，落审计）").
//
// It exists for the tenant-management console, which by definition reads and
// writes the `tenants` table across every tenant. The console is declared
// platform-scoped so a tenant-less platform operator can reach it, but that
// declaration alone does not stop a tenant administrator: a tenant admin's
// `role:<key>@tenant:<id>` policy covers `/api/v1/*`, so it would pass Casbin
// and could then enumerate the whole tenant population. This gate closes that
// path — the console stays a platform capability, and a tenant administrator
// keeps the rest of the console, confined to its own tenant.
//
// It is deliberately additive: Casbin still has to allow the route.
func PlatformCapabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !subjectHasPlatformCapability(c) {
			common.Fail(c, common.CodeForbidden, "tenant.platformOnly")
			c.Abort()
			return
		}
		c.Next()
	}
}

// subjectHasPlatformCapability reports whether the authenticated subject holds
// a platform-level grant (contract §3.1 "要求平台级权限"; §4 "先全局角色/权限
// （平台能力），再 domain 内角色"). Two signals count:
//
//  1. the platform ops role key — the marker the canary slice shipped;
//  2. any global (domain-less) Casbin policy that allows the route being
//     requested — a platform grant for exactly this endpoint.
//
// Global policies are the platform namespace by construction: tenant-internal
// authority lives in the `role:<key>@tenant:<id>` subjects (contract §4), so
// matching on the bare role key cannot be satisfied by a tenant-domain grant.
func subjectHasPlatformCapability(c *gin.Context) bool {
	roleKeys := common.GetRoleKeys(c)
	for _, roleKey := range roleKeys {
		if strings.TrimSpace(roleKey) == platformOpsRoleKey {
			return true
		}
	}
	if database.Enforcer == nil {
		return false
	}
	path := c.Request.URL.Path
	method := c.Request.Method
	for _, roleKey := range roleKeys {
		allowed, err := database.Enforcer.Enforce(strings.TrimSpace(roleKey), path, method)
		if err == nil && allowed {
			return true
		}
	}
	return false
}

// subjectTenantClaim extracts the numeric tenant claim stamped into the auth
// session at issuance. Gin stores it as uint64 (token middleware); a missing
// or non-numeric value means "no claim".
func subjectTenantClaim(c *gin.Context) string {
	if v, ok := c.Get("tenantId"); ok {
		switch id := v.(type) {
		case uint64:
			if id > 0 {
				return strconv.FormatUint(id, 10)
			}
		case int:
			if id > 0 {
				return strconv.Itoa(id)
			}
		case string:
			return strings.TrimSpace(id)
		}
	}
	return ""
}
