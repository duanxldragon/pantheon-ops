package middleware

import (
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

// CasbinMiddleware 权限校验中间件
func CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSelfServiceRoute(c) {
			c.Next()
			return
		}

		// 1. 获取用户角色集合 (在 Token 中间件之后，应已设置)
		roleKeys := common.GetRoleKeys(c)
		if len(roleKeys) == 0 {
			roleKeys = []string{"guest"}
		}

		if database.Enforcer == nil {
			failPermissionCheck(c, "permission.engine.not_initialized")
			return
		}

		// Tenant domain expansion (contract §4): consult the global subject
		// first, then the tenant-scoped subject role:<key>@tenant:<id> for the
		// resolved request context. Compat contexts resolve to global subjects
		// only — zero behavior change when the flag is off.
		tenantCtx := tenant.FromGin(c)
		allowed, err := authorizeRoleKeys(roleKeys, c.Request.URL.Path, c.Request.Method, casbinTenantEnforce(tenantCtx))
		if err != nil || !allowed {
			failPermissionCheck(c, "permission.denied")
			return
		}

		c.Next()
	}
}

func failPermissionCheck(c *gin.Context, messageKey string) {
	common.Fail(c, common.CodeForbidden, messageKey)
	c.Abort()
}

func authorizeRoleKeys(
	roleKeys []string,
	obj, act string,
	enforce func(roleKey, obj, act string) (bool, error),
) (bool, error) {
	for _, roleKey := range roleKeys {
		allowed, err := enforce(roleKey, obj, act)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

// casbinTenantEnforce returns an enforce function that consults every policy
// subject variant (global + tenant-scoped) for the resolved request context.
func casbinTenantEnforce(tenantCtx *tenant.Context) func(roleKey, obj, act string) (bool, error) {
	return func(roleKey, obj, act string) (bool, error) {
		return enforceAnySubject(tenant.CasbinDomainPolicySubjects(roleKey, tenantCtx), obj, act)
	}
}

// enforceAnySubject reports whether any subject variant is allowed for obj/act.
func enforceAnySubject(subjects []string, obj, act string) (bool, error) {
	for _, subject := range subjects {
		ok, err := database.Enforcer.Enforce(subject, obj, act)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func isSelfServiceRoute(c *gin.Context) bool {
	return isSelfServiceRouteBySignature(c.FullPath(), c.Request.Method, c.Query("scope"))
}

func isSelfServiceRouteBySignature(fullPath, method, scope string) bool {
	switch fullPath {
	case "/api/v1/system/logout":
		return method == "POST"
	case "/api/v1/auth/logout":
		return method == "POST"
	case "/api/v1/auth/activity":
		return method == "POST"
	case "/api/v1/system/user/info":
		return method == "GET"
	case "/api/v1/auth/me":
		return method == "GET"
	case "/api/v1/auth/security":
		return method == "GET"
	case "/api/v1/system/profile":
		return method == "GET" || method == "PUT"
	case "/api/v1/system/profile/password":
		return method == "PUT"
	case "/api/v1/system/menu/tree":
		return method == "GET" && strings.ToLower(strings.TrimSpace(scope)) != "manage"
	case "/api/v1/auth/password":
		return method == "PUT"
	case "/api/v1/auth/sessions":
		return method == "GET"
	case "/api/v1/auth/sessions/:id":
		return method == "DELETE"
	case "/api/v1/auth/login-logs":
		return method == "GET"
	default:
		return false
	}
}

func readRoleKeysFromContext(c *gin.Context) []string {
	return common.GetRoleKeys(c)
}
