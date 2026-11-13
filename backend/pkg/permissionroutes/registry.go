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

// Route is one provider-declared HTTP method and path binding.
type Route struct {
	Method string
	Path   string
}

// Binding is a permission key with its complete, deduplicated route list.
type Binding struct {
	Permission string
	Routes     []Route
}

// ErrInvalid reports a declaration that violates the registry contract.
var ErrInvalid = errors.New("permission_route.invalid")

// ErrConflict reports a declaration that disagrees with an existing binding.
var ErrConflict = errors.New("permission_route.conflict")

// ErrSealed reports a declaration attempted after the startup seal point.
var ErrSealed = errors.New("permission_route.sealed")

var modulePattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*$`)
var permissionPattern = regexp.MustCompile(`^business:[a-z][a-z0-9-]*:[a-z][a-z0-9:_-]*$`)
var segmentPattern = regexp.MustCompile(`^(?:[a-zA-Z0-9_-]+|:[a-zA-Z_][a-zA-Z0-9_]*)$`)
var registry = struct {
	sync.RWMutex
	bindings map[string][]Route
	sealed   bool
}{bindings: make(map[string][]Route)}

// Register adds a complete provider declaration. Invalid/conflicting batches
// leave the registry unchanged. Repeating exactly the same binding is harmless.
// This does not grant a role: only assigned keys create desired API policies.
// After Seal, every declaration is rejected with ErrSealed.
func Register(module string, bindings []Binding) error {
	if !modulePattern.MatchString(module) || len(bindings) == 0 || len(bindings) > 10000 {
		return ErrInvalid
	}
	domain := strings.TrimPrefix(module, "business:")
	prefix := "/api/v1/business/" + domain
	batch := make(map[string][]Route, len(bindings))
	for _, binding := range bindings {
		if _, duplicate := batch[binding.Permission]; duplicate {
			return ErrInvalid
		}
		routes, err := normalizeBindingRoutes(module, prefix, binding)
		if err != nil {
			return err
		}
		batch[binding.Permission] = routes
	}
	registry.Lock()
	defer registry.Unlock()
	if registry.sealed {
		return ErrSealed
	}
	if batchConflicts(batch) {
		return ErrConflict
	}
	for key, routes := range batch {
		registry.bindings[key] = routes
	}
	return nil
}

func normalizeBindingRoutes(module, prefix string, binding Binding) ([]Route, error) {
	if !permissionPattern.MatchString(binding.Permission) || !strings.HasPrefix(binding.Permission, module+":") || len(binding.Routes) == 0 || len(binding.Routes) > 1000 {
		return nil, ErrInvalid
	}
	routes := make([]Route, 0, len(binding.Routes))
	seen := make(map[Route]bool)
	for _, route := range binding.Routes {
		if !validRoutePath(prefix, route) || !validRouteMethod(route.Method) {
			return nil, ErrInvalid
		}
		if !seen[route] {
			routes = append(routes, route)
			seen[route] = true
		}
	}
	sortRoutes(routes)
	return routes, nil
}

func validRoutePath(prefix string, route Route) bool {
	return (route.Path == prefix || strings.HasPrefix(route.Path, prefix+"/")) && validPath(route.Path)
}

func validRouteMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return true
	default:
		return false
	}
}

func sortRoutes(routes []Route) {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
}

func batchConflicts(batch map[string][]Route) bool {
	for key, routes := range batch {
		existing, ok := registry.bindings[key]
		if !ok {
			continue
		}
		if len(existing) != len(routes) {
			return true
		}
		for i, route := range routes {
			if existing[i] != route {
				return true
			}
		}
	}
	return false
}

// Seal freezes the registry for the rest of the process. It is called from the
// same startup composition seam as the other compiled registries, so a running
// process cannot change the permission keys IAM reconciles against. Sealing is
// idempotent and never affects Snapshot or Lookup.
func Seal() error {
	registry.Lock()
	defer registry.Unlock()
	registry.sealed = true
	return nil
}

// Snapshot returns a deep copy; callers cannot mutate the authoritative map.
// Read paths stay available after Seal.
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
