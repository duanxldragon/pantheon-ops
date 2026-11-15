package bizscope

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// New business permissions grant only provider-owned, canonical routes.
func registerBizScopePermissionRoutes() error {
	options := permissionroutes.Route{Method: "GET", Path: "/api/v1/business/bizscope/options"}
	return permissionroutes.Register("business:bizscope", []permissionroutes.Binding{
		// The CMDB list retains its separate business:cmdb:host:view grant.
		{Permission: "business:bizscope:host:view", Routes: []permissionroutes.Route{options}},
		{Permission: "business:bizscope:environment:list", Routes: []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/bizscope/environments"}, options}},
		{Permission: "business:bizscope:environment:view", Routes: []permissionroutes.Route{{Method: "GET", Path: "/api/v1/business/bizscope/environments/:id"}}},
		{Permission: "business:bizscope:environment:create", Routes: []permissionroutes.Route{{Method: "POST", Path: "/api/v1/business/bizscope/environments"}, options}},
		{Permission: "business:bizscope:environment:update", Routes: []permissionroutes.Route{{Method: "PUT", Path: "/api/v1/business/bizscope/environments/:id"}, options}},
		{Permission: "business:bizscope:environment:delete", Routes: []permissionroutes.Route{{Method: "DELETE", Path: "/api/v1/business/bizscope/environments/:id"}}},
	})
}
