package observability

import (
	"sync"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

const observabilityAPIPrefix = "/api/v1/business/observability"

var auditRegistrationOnce sync.Once
var auditRegistrationErr error

// ProtectedQueryRoutes is the exact, code-owned adopted slice. GETs also
// protect result/correlation privacy, including deprecated expression URLs.
func ProtectedQueryRoutes() []auditpolicy.Route {
	var routes []auditpolicy.Route
	for _, prefix := range []string{observabilityAPIPrefix, "/api/v1/observability"} {
		for _, item := range []auditpolicy.Route{
			{Method: "POST", Template: "/datasources"}, {Method: "GET", Template: "/datasources"},
			{Method: "GET", Template: "/datasources/:id"}, {Method: "PUT", Template: "/datasources/:id"},
			{Method: "DELETE", Template: "/datasources/:id"}, {Method: "POST", Template: "/datasources/:id/test"},
			{Method: "POST", Template: "/datasources/:id/query"}, {Method: "POST", Template: "/datasources/:id/logs"},
			{Method: "POST", Template: "/datasources/:id/traces"}, {Method: "GET", Template: "/datasources/:id/query"},
			{Method: "GET", Template: "/datasources/:id/logs"}, {Method: "GET", Template: "/datasources/:id/traces"},
			{Method: "GET", Template: "/datasources/:id/traces/:traceId"}, {Method: "GET", Template: "/datasources/:id/query-fields"},
		} {
			item.Template = prefix + item.Template
			routes = append(routes, item)
		}
	}
	for _, item := range []auditpolicy.Route{
		{Method: "GET", Template: "/query-history"}, {Method: "GET", Template: "/query-history/:id"},
		{Method: "DELETE", Template: "/query-history"}, {Method: "DELETE", Template: "/query-history/:id"},
		{Method: "POST", Template: "/query-history/:id/restore"},
		{Method: "GET", Template: "/saved-views"}, {Method: "GET", Template: "/saved-views/:id"},
		{Method: "POST", Template: "/saved-views"}, {Method: "PUT", Template: "/saved-views/:id"},
		{Method: "DELETE", Template: "/saved-views/:id"}, {Method: "POST", Template: "/saved-views/:id/restore"},
	} {
		item.Template = observabilityAPIPrefix + item.Template
		routes = append(routes, item)
	}
	return routes
}
func IsProtectedQueryRoute(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	for _, route := range ProtectedQueryRoutes() {
		if c.Request.Method == route.Method && c.FullPath() == route.Template {
			return true
		}
	}
	return false
}
func RegisterQueryPolicies() error {
	auditRegistrationOnce.Do(func() {
		auditRegistrationErr = auditpolicy.RegisterMetadataOnly(ProtectedQueryRoutes())
		if auditRegistrationErr != nil {
			return
		}
		var bindings []permissionroutes.Binding
		for _, item := range []struct {
			key    string
			routes []permissionroutes.Route
		}{
			{"metric_source:view", []permissionroutes.Route{{Method: "GET", Path: "/datasources"}, {Method: "GET", Path: "/datasources/:id"}, {Method: "GET", Path: "/datasources/:id/query-fields"}, {Method: "GET", Path: "/datasources/:id/query"}, {Method: "POST", Path: "/datasources/:id/query"}, {Method: "POST", Path: "/datasources/:id/test"}}},
			{"metric_source:create", []permissionroutes.Route{{Method: "POST", Path: "/datasources"}}},
			{"metric_source:update", []permissionroutes.Route{{Method: "PUT", Path: "/datasources/:id"}}},
			{"metric_source:delete", []permissionroutes.Route{{Method: "DELETE", Path: "/datasources/:id"}}},
			{"logs:view", []permissionroutes.Route{{Method: "POST", Path: "/datasources/:id/logs"}, {Method: "GET", Path: "/datasources/:id/logs"}, {Method: "GET", Path: "/datasources/:id/query-fields"}}},
			{"traces:view", []permissionroutes.Route{{Method: "POST", Path: "/datasources/:id/traces"}, {Method: "GET", Path: "/datasources/:id/traces"}, {Method: "GET", Path: "/datasources/:id/traces/:traceId"}, {Method: "GET", Path: "/datasources/:id/query-fields"}}},
			{"query-history:view", []permissionroutes.Route{{Method: "GET", Path: "/query-history"}, {Method: "GET", Path: "/query-history/:id"}}},
			{"query-history:delete", []permissionroutes.Route{{Method: "DELETE", Path: "/query-history"}, {Method: "DELETE", Path: "/query-history/:id"}}},
			{"query-history:restore", []permissionroutes.Route{{Method: "POST", Path: "/query-history/:id/restore"}}},
			{"saved-view:view", []permissionroutes.Route{{Method: "GET", Path: "/saved-views"}, {Method: "GET", Path: "/saved-views/:id"}}},
			{"saved-view:create", []permissionroutes.Route{{Method: "POST", Path: "/saved-views"}}},
			{"saved-view:update", []permissionroutes.Route{{Method: "PUT", Path: "/saved-views/:id"}}},
			{"saved-view:delete", []permissionroutes.Route{{Method: "DELETE", Path: "/saved-views/:id"}}},
			{"saved-view:restore", []permissionroutes.Route{{Method: "POST", Path: "/saved-views/:id/restore"}}},
		} {
			for i := range item.routes {
				item.routes[i].Path = observabilityAPIPrefix + item.routes[i].Path
			}
			bindings = append(bindings, permissionroutes.Binding{Permission: "business:observability:" + item.key, Routes: item.routes})
		}
		auditRegistrationErr = permissionroutes.Register("business:observability", bindings)
	})
	return auditRegistrationErr
}
