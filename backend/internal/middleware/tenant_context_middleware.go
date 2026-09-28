package middleware

import (
	"errors"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// TenantContextMiddleware resolves the request tenant context from trusted
// sources (contract §3.1) and stores it read-only in the Gin context.
//
// Trusted sources, in priority order:
//  1. X-Tenant-Id header — only when the subject carries a platform-admin role
//     (admin role key alone does not mean cross-tenant; the contract requires
//     an explicit platform permission decision — canary: roleKeys contains
//     the platform ops role AND membership is not required for header use).
//  2. Subject tenant claim — the `tenantId` value stamped into the auth
//     session at issuance (core auth/iam task) and surfaced by
//     TokenAuthMiddleware. A numeric claim > 0 is trusted because only the
//     issuance gate (pkg/tenant.GateSessionIssuance) can stamp it.
//  3. Compat fallback — flag off resolves to global tenant 0.
//
// modeLoader is the cached flag reader; membershipDB resolves memberships
// (nil-safe: treated as "no memberships exist").
func TenantContextMiddleware(modeLoader *tenant.ModeLoader, membershipDB *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		mode := tenant.ModeCompat
		if modeLoader != nil {
			mode = modeLoader.Load()
		}

		subjectTenant := subjectTenantClaim(c)
		headerTenant := strings.TrimSpace(c.GetHeader(tenant.HeaderTenantID))

		var ctx *tenant.Context
		var err error
		if mode == tenant.ModeMulti {
			headerAllowed := false
			if headerTenant != "" {
				headerAllowed = subjectHasPlatformTenantOverride(c, membershipDB)
			}
			ctx, err = tenant.ResolveForCanary(mode, subjectTenant, headerTenant, headerAllowed)
			if err != nil {
				if errors.Is(err, tenant.ErrTenantContextMissing) {
					common.Fail(c, common.CodeError, "tenant.context.missing")
				} else {
					common.Fail(c, common.CodeForbidden, "tenant.forbidden")
				}
				c.Abort()
				return
			}
		} else {
			ctx, _ = tenant.ResolveForCanary(mode, "", "", false)
		}

		tenant.SetGin(c, ctx)
		c.Next()
	}
}

// subjectHasPlatformTenantOverride decides whether the authenticated subject
// may use X-Tenant-Id. Platform ops role only: admin/global roles do NOT
// implicitly gain cross-tenant reach (contract §7).
func subjectHasPlatformTenantOverride(c *gin.Context, membershipDB *gorm.DB) bool {
	roleKeys := common.GetRoleKeys(c)
	for _, roleKey := range roleKeys {
		if strings.TrimSpace(roleKey) == "platform_ops" {
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
