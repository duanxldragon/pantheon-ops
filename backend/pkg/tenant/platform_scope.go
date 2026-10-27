package tenant

import "strings"

// This file classifies the console surfaces that only a platform-capable
// subject may use. The route groups for menu authoring, the tenant console and
// i18n management are gated by PlatformCapabilityMiddleware (see
// modules/system/system_modules.go); a tenant subject that is shown those menus
// can open the page and then every request answers 403 tenant.platformOnly.
//
// The classification lives next to the tenant context, not inside the menu
// module, because three independent boundaries consume it: navigation/manage
// menu reads, tenant provisioning (which copies platform bindings), and the
// login permission loader (which must drop stale copied keys). Keeping one
// definition is what stops those boundaries from drifting apart.

// PlatformOnlyMenuModuleTenant is the module key of the platform tenancy menus.
const PlatformOnlyMenuModuleTenant = "system.tenant"

// platformOnlyMenuPaths are the seeded pages whose route groups require a
// platform capability. Keep aligned with the PlatformCapabilityMiddleware call
// sites in modules/system/system_modules.go.
//
// The tenant-ops pages already classify by module below, so listing their paths
// is deliberate redundancy: a menu tree that reaches a console without the
// `module` column filled (an older row, a gateway-cached payload) must still be
// hidden rather than render a page every request of which answers 403.
var platformOnlyMenuPaths = []string{
	"/system/tenant",
	"/system/tenant-ops/overview",
	"/system/tenant-ops/member",
	"/system/menu",
	"/system/i18n",
}

// IsPlatformOnlyMenu reports whether a menu row belongs to a platform-only
// console surface.
func IsPlatformOnlyMenu(module, path string) bool {
	if strings.TrimSpace(module) == PlatformOnlyMenuModuleTenant {
		return true
	}
	path = strings.TrimSpace(path)
	for _, candidate := range platformOnlyMenuPaths {
		if path == candidate {
			return true
		}
	}
	return false
}

// IsPlatformOnlyPermissionKey reports whether a permission key authorises a
// platform-only surface.
//
// `system:menu:list` is deliberately NOT in the set. Tenant role administration
// renders the manage-scoped menu tree but must not author the shared tree, so
// the read key stays assignable while the write keys do not.
func IsPlatformOnlyPermissionKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	if strings.HasPrefix(key, "system:tenant:") || strings.HasPrefix(key, "system:i18n:") {
		return true
	}
	switch key {
	case "system:menu:create", "system:menu:update", "system:menu:delete":
		return true
	default:
		return false
	}
}

// FilterPlatformOnlyPermissionKeys drops platform-only keys from a permission
// list, preserving the order of the remaining keys.
func FilterPlatformOnlyPermissionKeys(keys []string) []string {
	filtered := make([]string, 0, len(keys))
	for _, key := range keys {
		if IsPlatformOnlyPermissionKey(key) {
			continue
		}
		filtered = append(filtered, key)
	}
	return filtered
}

// IsTenantScoped reports whether a resolved request context is a tenant-private
// view. Compat mode and the platform fallback both resolve to the platform
// global tenant (0) and are not tenant-scoped, so a platform operator who
// explicitly pins a tenant through X-Tenant-Id is treated as that tenant's view.
func IsTenantScoped(ctx *Context) bool {
	return ctx != nil && ctx.IsMulti() && ctx.TenantID != PlatformGlobalTenantID
}
