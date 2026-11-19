package service

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// registerServiceReadPermissionRoutes binds existing catalog reads to the page
// and view permissions. Declaring a key does not assign it to any role.
func registerServiceReadPermissionRoutes() error {
	reads := []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/service/applications"},
		{Method: "GET", Path: "/api/v1/business/service/applications/options"},
		{Method: "GET", Path: "/api/v1/business/service/services"},
		{Method: "GET", Path: "/api/v1/business/service/services/options"},
		{Method: "GET", Path: "/api/v1/business/service/services/:id"},
		{Method: "GET", Path: "/api/v1/business/service/instances"},
		{Method: "GET", Path: "/api/v1/business/service/instances/:id"},
	}
	return permissionroutes.Register("business:service", []permissionroutes.Binding{
		{Permission: "business:service:list", Routes: reads},
		{Permission: "business:service:view", Routes: reads},
	})
}
