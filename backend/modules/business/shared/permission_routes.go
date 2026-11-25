package shared

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// DetailBinding binds GET-only detail routes to one permission key.
func DetailBinding(permission string, paths ...string) permissionroutes.Binding {
	routes := make([]permissionroutes.Route, len(paths))
	for i, path := range paths {
		routes[i] = permissionroutes.Route{Method: "GET", Path: path}
	}
	return permissionroutes.Binding{Permission: permission, Routes: routes}
}
