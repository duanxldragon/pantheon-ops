package deploy

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// registerDeployPermissionRoutes binds the deploy package keys that must have
// API policies for a scoped (non-admin) role to see and drive the package page.
// Mirror of cmdb's registerHostPermissionRoutes: the released keyMatch2 matcher
// lets a `GET /packages/:id` policy also match `GET /packages/export`, so the
// detail route is deliberately NOT registered under the view key. That detail
// grant is a Base static-route-guard item; until Base ships it, register list
// only to avoid silently widening export to every package viewer.
func registerDeployPermissionRoutes() error {
	return permissionroutes.Register("business:deploy", []permissionroutes.Binding{
		{Permission: "business:deploy:package:view", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/packages"},
		}},
		{Permission: "business:deploy:package:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/packages"},
		}},
		{Permission: "business:deploy:package:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/deploy/packages/:id"},
		}},
		{Permission: "business:deploy:package:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/deploy/packages/:id"},
		}},
		{Permission: "business:deploy:package:export", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/packages/export"},
		}},
		{Permission: "business:deploy:package:import", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/packages/import"},
		}},
	})
}
