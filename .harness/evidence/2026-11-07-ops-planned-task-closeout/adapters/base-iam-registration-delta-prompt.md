Read-only CODE review, no tools. Max 160 words. Proved P1/P2 plus minimal fixes; otherwise APPROVE supplied scope only. State omitted source.
Registry source previously reviewed APPROVE. These two helpers supply discovery and role synchronization; role sync calls permissionRequiredAPIsForKey for exact grants and revokes. Review supplied delta only.
"system:module:generate": {"POST /api/v1/lowcode/dynamic-modules/generate"},
"system:generator:datasource:manage": {"POST /api/v1/lowcode/generator/datasources", "PUT /api/v1/lowcode/generator/datasources/:id", "DELETE /api/v1/lowcode/generator/datasources/:id", "POST /api/v1/lowcode/generator/datasources/:id/test"},
}
for _, export := range tenantctx.ReadOnlyExportRoutes {
routes[export.PermissionKey] = append(routes[export.PermissionKey], "POST "+export.Path)
}
return routes
}()

// RequiredAPIRoutesByPermission exposes the permission→route binding table for
// drift checks (tests) without allowing callers to mutate the workbench view.
func RequiredAPIRoutesByPermission() map[string][]RequiredAPIRouteEntry {
out := make(map[string][]RequiredAPIRouteEntry, len(requiredAPIRoutesByPermission))
for key, routes := range requiredAPIRoutesByPermission {
entries := make([]RequiredAPIRouteEntry, 0, len(routes))
for _, route := range routes {
method, path, _ := strings.Cut(route, " ")
entries = append(entries, RequiredAPIRouteEntry{Method: method, Path: path})
}
out[key] = entries
}
for key, routes := range permissionroutes.Snapshot() {
entries := make([]RequiredAPIRouteEntry, 0, len(routes))
for _, route := range routes {
entries = append(entries, RequiredAPIRouteEntry{Method: route.Method, Path: route.Path})
}
out[key] = entries
}
return out
}

func requiredAPIPoliciesByPermissionKey(permissionKey string) []permissionRequiredAPIPolicy {
key := strings.TrimSpace(permissionKey)
routes := requiredAPIRoutesByPermission[key]
policies := make([]permissionRequiredAPIPolicy, 0, len(routes))
for _, route := range routes {
method, path, _ := strings.Cut(route, " ")
policies = append(policies, permissionRequiredAPIPolicy{Path: path, Method: method})
}
for _, route := range permissionroutes.Lookup(key) {
policies = append(policies, permissionRequiredAPIPolicy{Path: route.Path, Method: route.Method})
}
return policies
}

// getRoleMissingAPIPolicies fetches only the data needed to determine missing API policies
// for a single role, without computing the full workbench. This avoids loading overview,
// remediation events, integrity/coverage filters, and unrelated roles.
func (s *PermissionService) getRoleMissingAPIPolicies(roleKey string) (*PermissionWorkbenchRoleResp, error) {
var role permissionWorkbenchRoleRow
// The role belongs to the caller's scope, never to an arbitrary tenant that
// happens to own the same role key. Without the tenant predicate a platform
// remediation could resolve a tenant's `tenant_admin` and then write a dead
// global subject for it.
