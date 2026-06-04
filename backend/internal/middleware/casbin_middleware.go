package middleware

import (
	"pantheon-platform/backend/pkg/common"
	"pantheon-platform/backend/pkg/database"
	"strings"

	"github.com/gin-gonic/gin"
)

// CasbinMiddleware 权限校验中间件
func CasbinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSelfServiceRoute(c) {
			c.Next()
			return
		}

		// 1. 获取用户角色集合 (在 JWT 中间件之后，应已设置)
		roleKeys := readRoleKeysFromContext(c)
		if len(roleKeys) == 0 {
			roleKeys = []string{"guest"}
		}

		if database.Enforcer == nil {
			failPermissionCheck(c, "permission.engine.not_initialized")
			return
		}

		allowed, err := authorizeRoleKeys(roleKeys, c.Request.URL.Path, c.Request.Method, func(roleKey, obj, act string) (bool, error) {
			return database.Enforcer.Enforce(roleKey, obj, act)
		})
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

func isSelfServiceRoute(c *gin.Context) bool {
	return isSelfServiceRouteBySignature(c.FullPath(), c.Request.Method, c.Query("scope"))
}

func isSelfServiceRouteBySignature(fullPath string, method string, scope string) bool {
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
	roleKeysValue, ok := c.Get("roleKeys")
	if ok {
		if roleKeys, ok := roleKeysValue.([]string); ok {
			return roleKeys
		}
	}
	roleKeyValue, ok := c.Get("roleKey")
	if ok {
		if roleKey, ok := roleKeyValue.(string); ok && roleKey != "" {
			return []string{roleKey}
		}
	}
	return nil
}
