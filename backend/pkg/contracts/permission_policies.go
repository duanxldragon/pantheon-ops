package contracts

import "strings"

type PermissionAPIPolicy struct {
	Path   string
	Method string
}

func RequiredAPIPoliciesByPermissionKey(permissionKey string) []PermissionAPIPolicy {
	switch strings.TrimSpace(permissionKey) {
	case "system:security-event:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/security-event/list", Method: "GET"},
		}
	case "system:module:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules", Method: "GET"},
		}
	case "system:module:register":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules", Method: "POST"},
		}
	case "system:module:unregister":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name", Method: "DELETE"},
		}
	case "system:module:delete_record":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name/record", Method: "DELETE"},
		}
	case "system:module:purge":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/:name/purge", Method: "DELETE"},
		}
	case "system:module:generate":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/dynamic-modules/generate", Method: "POST"},
		}
	case "system:generator:datasource:manage":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/system/generator/datasources", Method: "POST"},
			{Path: "/api/v1/system/generator/datasources/:id", Method: "PUT"},
			{Path: "/api/v1/system/generator/datasources/:id", Method: "DELETE"},
			{Path: "/api/v1/system/generator/datasources/:id/test", Method: "POST"},
		}
	case "business:cmdb:host:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts", Method: "GET"},
		}
	case "business:cmdb:host:detail":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts/:id", Method: "GET"},
		}
	case "business:cmdb:host:create":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts", Method: "POST"},
		}
	case "business:cmdb:host:update":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts/:id", Method: "PUT"},
		}
	case "business:cmdb:host:delete":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts/:id", Method: "DELETE"},
		}
	case "business:cmdb:host:collect":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts/:id/collect", Method: "POST"},
		}
	case "business:cmdb:host:status":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/hosts/:id/status", Method: "PATCH"},
		}
	case "business:cmdb:group:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/groups", Method: "GET"},
		}
	case "business:cmdb:group:detail":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/groups/:id", Method: "GET"},
			{Path: "/api/v1/business/cmdb/groups/:id/members", Method: "GET"},
		}
	case "business:cmdb:group:create":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/groups", Method: "POST"},
		}
	case "business:cmdb:group:update":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/groups/:id", Method: "PUT"},
		}
	case "business:cmdb:group:delete":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/groups/:id", Method: "DELETE"},
		}
	case "business:cmdb:label:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/labels", Method: "GET"},
		}
	case "business:cmdb:label:create":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/labels", Method: "POST"},
		}
	case "business:cmdb:label:update":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/labels/:id", Method: "PUT"},
		}
	case "business:cmdb:label:delete":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/cmdb/labels/:id", Method: "DELETE"},
		}
	case "business:deploy:package:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/packages", Method: "GET"},
		}
	case "business:deploy:package:create":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/packages", Method: "POST"},
		}
	case "business:deploy:package:update":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/packages/:id", Method: "PUT"},
		}
	case "business:deploy:package:delete":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/packages/:id", Method: "DELETE"},
		}
	case "business:deploy:task:list":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks", Method: "GET"},
		}
	case "business:deploy:task:detail":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks/:id", Method: "GET"},
		}
	case "business:deploy:task:create":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks", Method: "POST"},
		}
	case "business:deploy:task:update":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks/:id", Method: "PUT"},
		}
	case "business:deploy:task:start":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks/:id/start", Method: "POST"},
		}
	case "business:deploy:task:cancel":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/tasks/:id/cancel", Method: "POST"},
		}
	case "business:deploy:task:mark-result":
		return []PermissionAPIPolicy{
			{Path: "/api/v1/business/deploy/task-hosts/:id/result", Method: "POST"},
			{Path: "/api/v1/business/deploy/task-hosts/:id/report", Method: "POST"},
		}
	default:
		return nil
	}
}
