Read-only CODE review; use no tools. Supplied source only. Prove P1/P2 defects and minimal fix, otherwise APPROVE this bounded scope. Max 220 words; state omitted scope.
Typed registry already reviewed separately. Review actual required API discovery and role grants/revokes using it. No project approval.
"system:module:list": {"GET /api/v1/lowcode/dynamic-modules"},
"system:module:register": {"POST /api/v1/lowcode/dynamic-modules"},
"system:module:unregister": {"DELETE /api/v1/lowcode/dynamic-modules/:name"},
"system:module:delete_record": {"DELETE /api/v1/lowcode/dynamic-modules/:name/record"},
"system:module:purge": {"DELETE /api/v1/lowcode/dynamic-modules/:name/purge"},
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
if err := s.db.Table("system_role").
Where("role_key = ? AND deleted_at IS NULL", roleKey).
Where(permissionTenantClause, s.TenantOwnerID()).
First(&role).Error; err != nil {
return nil, database.TranslateNotFound(err, common.NewBadRequest(errPermissionRoleInvalid))
}

    result := &PermissionWorkbenchRoleResp{
    	ID:                 role.ID,
    	RoleName:           role.RoleName,
    	RoleKey:            role.RoleKey,
    	Status:             role.Status,
    	Menus:              []PermissionWorkbenchMenuResp{},
    	PagePermissions:    []PermissionWorkbenchPermissionResp{},
    	ActionPermissions:  []PermissionWorkbenchPermissionResp{},
    	UnknownPermissions: []PermissionWorkbenchPermissionResp{},

FILE pantheon-base\backend\modules\system\iam\role\role_policy_sync.go
package role

import (
"strings"

    "github.com/casbin/casbin/v2/util"
    "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
    "github.com/duanxldragon/pantheon-base/backend/pkg/database"

    "gorm.io/gorm"
    "gorm.io/gorm/clause"

)

type roleAPIRoute struct {
path string
method string
}

// syncRoleAPIPolicies keeps the Casbin policies of one role in step with the
// permission keys the role currently holds. Only routes declared by the shared
// permission→route binding table are managed here: policies outside that table
// (wildcards written by tenant provisioning, manual grants from the permission
// page) are never touched.
// syncRoleAPIPolicies reconciles one role's Casbin rows with the routes its
// permission keys declare. The three steps are separate functions because each
// one is a different question - what the binding table manages, which rows are
// stale, which grants are missing - and the policy set is only correct when all
// three read the same route maps.
func syncRoleAPIPolicies(tx *gorm.DB, subject string, permissionKeys []string) error {
subject = strings.TrimSpace(subject)
if subject == "" {
return nil
}

    managed, desired := rolePolicyRouteSets(permissionKeys)
    if len(managed) == 0 {
    	return nil
    }

    var existing []database.CasbinRule
    if err := tx.Where("ptype = ? AND v0 = ?", "p", subject).Find(&existing).Error; err != nil {
    	return err
    }

    if err := deleteRetiredRolePolicies(tx, existing, managed, desired); err != nil {
    	return err
    }

    created := missingRolePolicies(subject, desired, existing)
    if len(created) == 0 {
    	return nil
    }
    // Remediation and role saves can race on the same natural key; the unique
    // index keeps one row and the loser simply observes it as already present.
    return tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(created, 100).Error

}

// rolePolicyRouteSets returns every route the shared binding table declares and
// the subset of those routes the given permission keys grant.
func rolePolicyRouteSets(permissionKeys []string) (managed, desired map[roleAPIRoute]struct{}) {
heldKeys := make(map[string]struct{}, len(permissionKeys))
for _, key := range normalizePermissionKeys(permissionKeys) {
heldKeys[key] = struct{}{}
}

    managed = make(map[roleAPIRoute]struct{})
    desired = make(map[roleAPIRoute]struct{})
    for permissionKey, entries := range permission.RequiredAPIRoutesByPermission() {
    	for _, entry := range entries {
    		route := roleAPIRoute{
    			path:   strings.TrimSpace(entry.Path),
    			method: normalizeRolePolicyMethod(entry.Method),
    		}
    		if route.path == "" || route.method == "" {
    			continue
    		}
    		managed[route] = struct{}{}
    		if _, ok := heldKeys[permissionKey]; ok {
    			desired[route] = struct{}{}
    		}
    	}
    }
    return managed, desired

}

// deleteRetiredRolePolicies drops the stored rows that the binding table
// manages but the role no longer grants. Rows outside `managed` are never
// touched, so wildcard and manual grants survive a role save.
func deleteRetiredRolePolicies(
tx *gorm.DB,
existing []database.CasbinRule,
managed, desired map[roleAPIRoute]struct{},
) error {
removedIDs := make([]uint64, 0)
for _, policy := range existing {
route := roleAPIRoute{
path: strings.TrimSpace(policy.V1),
method: normalizeRolePolicyMethod(policy.V2),
}
if _, ok := managed[route]; !ok {
continue
}
if _, ok := desired[route]; !ok {
removedIDs = append(removedIDs, policy.ID)
}
}
if len(removedIDs) == 0 {
return nil
}
return tx.Where("id IN ?", removedIDs).Delete(&database.CasbinRule{}).Error
}

// missingRolePolicies builds the rows to insert for granted routes the role has
// no policy for yet.
func missingRolePolicies(
subject string,
desired map[roleAPIRoute]struct{},
existing []database.CasbinRule,
) []database.CasbinRule {
present := make(map[roleAPIRoute]struct{}, len(existing))
for _, policy := range existing {
route := roleAPIRoute{
path: strings.TrimSpace(policy.V1),
method: normalizeRolePolicyMethod(policy.V2),
}
present[route] = struct{}{}
}

    created := make([]database.CasbinRule, 0, len(desired))
    for route := range desired {
    	if _, ok := present[route]; ok {
    		continue
    	}
    	// A wildcard grant (`/api/v1/*`) already enforces this route; adding an
    	// exact duplicate would only bloat the policy table.
    	if rolePolicyCoversRoute(existing, route) {
    		continue
    	}
    	created = append(created, database.CasbinRule{
    		PType: "p",
    		V0:    subject,
    		V1:    route.path,
    		V2:    route.method,
    	})
    }
    return created

}

func normalizeRolePolicyMethod(method string) string {
return strings.ToUpper(strings.TrimSpace(method))
}

// rolePolicyCoversRoute reports whether an existing policy already authorises
// the route under the Casbin keyMatch2 matcher used by the enforcement model.
func rolePolicyCoversRoute(policies []database.CasbinRule, route roleAPIRoute) bool {
for _, policy := range policies {
if normalizeRolePolicyMethod(policy.V2) != route.method {
continue
}
if util.KeyMatch2(route.path, strings.TrimSpace(policy.V1)) {
return true
}
}
return false
}

FILE pantheon-base\backend\pkg\permissionroutes\registry.go
// Package permissionroutes owns the process-scoped, provider-owned API binding
// registry. Providers register immutable business route bindings during startup;
// IAM consumes copies for role synchronization and workbench coverage.
package permissionroutes

import (
"errors"
"regexp"
"sort"
"strings"
"sync"
)

type Route struct {
Method string
Path string
}
type Binding struct {
Permission string
Routes []Route
}

var ErrInvalid = errors.New("permission_route.invalid")
var ErrConflict = errors.New("permission_route.conflict")
var modulePattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*$`)
var permissionPattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*:[a-z][a-z0-9:-]*$`)
var segmentPattern = regexp.MustCompile(`^(?:[a-zA-Z0-9_-]+|:[a-zA-Z_][a-zA-Z0-9_]*)$`)
var registry = struct {
sync.RWMutex
bindings map[string][]Route
}{bindings: make(map[string][]Route)}

// Register adds a complete provider declaration. Invalid/conflicting batches
// leave the registry unchanged. Repeating exactly the same binding is harmless.
// This does not grant a role: only assigned keys create desired API policies.
func Register(module string, bindings []Binding) error {
if !modulePattern.MatchString(module) || len(bindings) == 0 || len(bindings) > 10000 {
return ErrInvalid
}
domain := strings.TrimPrefix(module, "business:")
prefix := "/api/v1/business/" + domain
batch := make(map[string][]Route, len(bindings))
for _, binding := range bindings {
if !permissionPattern.MatchString(binding.Permission) || !strings.HasPrefix(binding.Permission, module+":") || len(binding.Routes) == 0 || len(binding.Routes) > 1000 {
return ErrInvalid
}
if _, duplicate := batch[binding.Permission]; duplicate {
return ErrInvalid
}
routes := make([]Route, 0, len(binding.Routes))
seen := make(map[Route]bool)
for _, route := range binding.Routes {
if !(route.Path == prefix || strings.HasPrefix(route.Path, prefix+"/")) || !validPath(route.Path) {
return ErrInvalid
}
switch route.Method {
case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
default:
return ErrInvalid
}
if !seen[route] {
routes = append(routes, route)
seen[route] = true
}
}
sort.Slice(routes, func(i, j int) bool {
if routes[i].Path == routes[j].Path {
return routes[i].Method < routes[j].Method
}
return routes[i].Path < routes[j].Path
})
batch[binding.Permission] = routes
}
registry.Lock()
defer registry.Unlock()
for key, routes := range batch {
if existing, ok := registry.bindings[key]; ok {
if len(existing) != len(routes) {
return ErrConflict
}
for i, route := range routes {
if existing[i] != route {
return ErrConflict
}
}
}
}
for key, routes := range batch {
registry.bindings[key] = routes
}
return nil
}

// Snapshot returns a deep copy; callers cannot mutate the authoritative map.
func Snapshot() map[string][]Route {
registry.RLock()
defer registry.RUnlock()
out := make(map[string][]Route, len(registry.bindings))
for key, routes := range registry.bindings {
out[key] = append([]Route(nil), routes...)
}
return out
}

// Lookup copies only one key for coverage checks of individual permissions.
func Lookup(key string) []Route {
registry.RLock()
defer registry.RUnlock()
return append([]Route(nil), registry.bindings[key]...)
}

func validPath(path string) bool {
if !strings.HasPrefix(path, "/api/v1/business/") {
return false
}
for _, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
if !segmentPattern.MatchString(segment) {
return false
}
}
return true
}
