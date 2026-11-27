package argocd

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// RegisterPermissionRoutes declares the REST permissions before Base seals the registry.
func RegisterPermissionRoutes() error {
	const (
		prefix            = "/api/v1/business/cicd/argocd"
		serverDetail      = prefix + "/servers/:id"
		repositoryDetail  = prefix + "/repositories/:id"
		applicationDetail = prefix + "/applications/:id"
	)
	reads := []permissionroutes.Route{
		{Method: "GET", Path: prefix + "/servers"},
		{Method: "GET", Path: serverDetail},
		{Method: "GET", Path: prefix + "/repositories"},
		{Method: "GET", Path: repositoryDetail},
		{Method: "GET", Path: prefix + "/applications"},
		{Method: "GET", Path: applicationDetail},
		{Method: "GET", Path: applicationDetail + "/history"},
	}
	return permissionroutes.Register("business:cicd", []permissionroutes.Binding{
		{Permission: "business:cicd:argocd:list", Routes: reads},
		{Permission: "business:cicd:argocd:view", Routes: reads},
		{Permission: "business:cicd:argocd:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: prefix + "/servers"},
			{Method: "PUT", Path: serverDetail},
			{Method: "POST", Path: serverDetail + "/test"},
			{Method: "POST", Path: prefix + "/repositories"},
			{Method: "PUT", Path: repositoryDetail},
			{Method: "POST", Path: prefix + "/applications"},
			{Method: "PUT", Path: applicationDetail},
			{Method: "POST", Path: applicationDetail + "/sync"},
			{Method: "POST", Path: applicationDetail + "/refresh"},
			{Method: "POST", Path: applicationDetail + "/rollback"},
		}},
		{Permission: "business:cicd:argocd:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: serverDetail},
			{Method: "DELETE", Path: repositoryDetail},
			{Method: "DELETE", Path: applicationDetail},
		}},
	})
}
