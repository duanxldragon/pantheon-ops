Independent CODE review. Read-only, no tools. Supplied source only. Report proved P0/P1/P2 defects and minimal fix; otherwise APPROVE. Max 180 words. State scope limits. No release, production or project-completion approval. Current/frozen version binding was previously tested; do not infer missing helpers are defective.

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
