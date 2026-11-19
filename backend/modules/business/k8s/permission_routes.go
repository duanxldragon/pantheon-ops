package k8s

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// registerK8sServicePermissionRoutes declares complete Service action bindings.
// Only assigned keys grant policies; a read key never includes mutation verbs.
func registerK8sServicePermissionRoutes() error {
	read := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/services"},
		{Method: "GET", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
	}
	return permissionroutes.Register("business:k8s", []permissionroutes.Binding{
		{Permission: "business:k8s:service:list", Routes: read},
		{Permission: "business:k8s:service:view", Routes: read},
		{Permission: "business:k8s:service:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/k8s/clusters/:id/services"},
		}},
		{Permission: "business:k8s:service:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
		}},
		{Permission: "business:k8s:service:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/k8s/clusters/:id/services/:name"},
		}},
	})
}
