package argocd

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// RegisterPermissionRoutes declares the REST permissions before Base seals the registry.
func RegisterPermissionRoutes() error {
	const prefix = "/api/v1/business/cicd/argocd"
	reads := []permissionroutes.Route{
		{Method: "GET", Path: prefix + "/servers"},
		{Method: "GET", Path: prefix + "/servers/:id"},
		{Method: "GET", Path: prefix + "/repositories"},
		{Method: "GET", Path: prefix + "/repositories/:id"},
		{Method: "GET", Path: prefix + "/applications"},
		{Method: "GET", Path: prefix + "/applications/:id"},
		{Method: "GET", Path: prefix + "/applications/:id/history"},
	}
	return permissionroutes.Register("business:cicd", []permissionroutes.Binding{
		{Permission: "business:cicd:argocd:list", Routes: reads},
		{Permission: "business:cicd:argocd:view", Routes: reads},
		{Permission: "business:cicd:argocd:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: prefix + "/servers"},
			{Method: "PUT", Path: prefix + "/servers/:id"},
			{Method: "POST", Path: prefix + "/servers/:id/test"},
			{Method: "POST", Path: prefix + "/repositories"},
			{Method: "PUT", Path: prefix + "/repositories/:id"},
			{Method: "POST", Path: prefix + "/applications"},
			{Method: "PUT", Path: prefix + "/applications/:id"},
			{Method: "POST", Path: prefix + "/applications/:id/sync"},
			{Method: "POST", Path: prefix + "/applications/:id/refresh"},
			{Method: "POST", Path: prefix + "/applications/:id/rollback"},
		}},
		{Permission: "business:cicd:argocd:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: prefix + "/servers/:id"},
			{Method: "DELETE", Path: prefix + "/repositories/:id"},
			{Method: "DELETE", Path: prefix + "/applications/:id"},
		}},
	})
}
