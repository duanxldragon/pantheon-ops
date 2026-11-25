package cmdb

import (
	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
)

// Register the existing CMDB keys. Base enforces business routes against the
// Gin route template and permissionroutes.Covers lets a :param policy segment
// cover only :param route segments, so a detail grant (GET /hosts/:id) never
// reaches a static sibling such as /hosts/export or /hosts/import-template.
// Detail reads use the seeded host/group detail keys.
func registerHostPermissionRoutes() error {
	return permissionroutes.Register("business:cmdb", []permissionroutes.Binding{
		{Permission: "business:cmdb:host:view", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/hosts"},
		}},
		shared.DetailBinding("business:cmdb:host:detail", "/api/v1/business/cmdb/hosts/:id"),
		{Permission: "business:cmdb:host:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/cmdb/hosts"},
		}},
		{Permission: "business:cmdb:host:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/cmdb/hosts/:id"},
		}},
		{Permission: "business:cmdb:host:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/cmdb/hosts/:id"},
		}},
		{Permission: "business:cmdb:host:collect", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/cmdb/hosts/:id/collect"},
		}},
		{Permission: "business:cmdb:host:status", Routes: []permissionroutes.Route{
			{Method: "PATCH", Path: "/api/v1/business/cmdb/hosts/:id/status"},
			{Method: "PATCH", Path: "/api/v1/business/cmdb/hosts/:id/state"},
		}},
		{Permission: "business:cmdb:host:export", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/hosts/export"},
		}},
		{Permission: "business:cmdb:host:import", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/hosts/import-template"},
			{Method: "POST", Path: "/api/v1/business/cmdb/hosts/import"},
		}},
		{Permission: "business:cmdb:group:view", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/groups"},
		}},
		shared.DetailBinding("business:cmdb:group:detail", "/api/v1/business/cmdb/groups/:id", "/api/v1/business/cmdb/groups/:id/members"),
		{Permission: "business:cmdb:group:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/cmdb/groups"},
		}},
		{Permission: "business:cmdb:group:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/cmdb/groups/:id"},
		}},
		{Permission: "business:cmdb:group:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/cmdb/groups/:id"},
		}},
		{Permission: "business:cmdb:group:export", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/groups/export"},
		}},
		{Permission: "business:cmdb:group:import", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/groups/import-template"},
			{Method: "POST", Path: "/api/v1/business/cmdb/groups/import"},
		}},
		{Permission: "business:cmdb:label:view", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/labels"},
			{Method: "GET", Path: "/api/v1/business/cmdb/labels/options"},
		}},
		{Permission: "business:cmdb:label:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/cmdb/labels"},
		}},
		{Permission: "business:cmdb:label:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/cmdb/labels/:id"},
		}},
		{Permission: "business:cmdb:label:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/cmdb/labels/:id"},
		}},
		{Permission: "business:cmdb:label:export", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/labels/export"},
		}},
		{Permission: "business:cmdb:label:import", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/cmdb/labels/import-template"},
			{Method: "POST", Path: "/api/v1/business/cmdb/labels/import"},
		}},
	})
}
