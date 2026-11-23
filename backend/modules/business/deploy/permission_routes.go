package deploy

import "github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"

// registerDeployPermissionRoutes binds the deploy package/template/task/credential
// keys that must have API policies for a scoped (non-admin) role to see and drive
// the deploy pages. Mirror of cmdb's registerHostPermissionRoutes: the released
// keyMatch2 matcher lets a `GET /packages/:id` policy also match `GET /packages/export`,
// so every detail route (`/packages/:id`, `/templates/:id`, `/tasks/:id`) is
// deliberately NOT registered under its read key. That detail grant is a Base
// static-route-guard item; until Base ships it, register static/list routes only to
// avoid silently widening export to every viewer.
//
// Detail-family reads (task detail/logs/events/rollback) also stay unbound for the
// same matcher gap; scoped roles are denied by default there. `business:deploy:task:export`
// is registered by log_download.go (export + log-download) and MUST NOT be repeated
// here — a second Register with a different route list would fail the module seal.
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
		{Permission: "business:deploy:template:list", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/templates"},
		}},
		{Permission: "business:deploy:template:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/templates"},
		}},
		{Permission: "business:deploy:template:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/deploy/templates/:id"},
		}},
		{Permission: "business:deploy:template:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/deploy/templates/:id"},
		}},
		{Permission: "business:deploy:template:export", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/templates/export"},
		}},
		{Permission: "business:deploy:template:import", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/templates/import"},
		}},
		{Permission: "business:deploy:task:view", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/tasks"},
			// Resolving targets is an in-scope read used by the target picker.
			{Method: "POST", Path: "/api/v1/business/deploy/targets/resolve"},
		}},
		{Permission: "business:deploy:task:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks"},
		}},
		{Permission: "business:deploy:task:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/deploy/tasks/:id"},
		}},
		{Permission: "business:deploy:task:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/deploy/tasks/:id"},
		}},
		{Permission: "business:deploy:task:start", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/start"},
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/preview"},
		}},
		{Permission: "business:deploy:task:pause", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/pause"},
		}},
		{Permission: "business:deploy:task:resume", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/resume"},
		}},
		{Permission: "business:deploy:task:retry", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/retry"},
		}},
		{Permission: "business:deploy:task:cancel", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/tasks/:id/cancel"},
		}},
		{Permission: "business:deploy:task:mark-result", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/task-hosts/:id/result"},
			{Method: "POST", Path: "/api/v1/business/deploy/task-hosts/:id/report"},
		}},
		{Permission: "business:deploy:credential:list", Routes: []permissionroutes.Route{
			{Method: "GET", Path: "/api/v1/business/deploy/credentials"},
		}},
		{Permission: "business:deploy:credential:create", Routes: []permissionroutes.Route{
			{Method: "POST", Path: "/api/v1/business/deploy/credentials"},
		}},
		{Permission: "business:deploy:credential:update", Routes: []permissionroutes.Route{
			{Method: "PUT", Path: "/api/v1/business/deploy/credentials/:id"},
		}},
		{Permission: "business:deploy:credential:delete", Routes: []permissionroutes.Route{
			{Method: "DELETE", Path: "/api/v1/business/deploy/credentials/:id"},
		}},
	})
}
