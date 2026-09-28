package iam

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// buildWorkbenchRouteProbeEngine registers the exact routes the workbench
// binding table claims to protect, mirroring the live registration shape
// (system/* and lowcode/* protected groups). Handlers are no-ops: this probe
// only inspects the routing table, never executes handlers or middleware.
//
// If a registration in the live modules changes shape (path or method), this
// probe must be updated in the same patch — the guard below then compares the
// binding table against the probe and fails on any drift.
func buildWorkbenchRouteProbeEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api/v1")

	systemProtected := api.Group("/system")
	{
		// user module (system_modules.go initIAMModules)
		systemProtected.GET("/user/import-template", noRouteHandler)
		systemProtected.GET("/user/:id", noRouteHandler)
		systemProtected.POST("/user", noRouteHandler)
		systemProtected.POST("/user/import", noRouteHandler)
		systemProtected.POST("/user/batch-status", noRouteHandler)
		systemProtected.POST("/user/batch-delete", noRouteHandler)
		systemProtected.PUT("/user/:id", noRouteHandler)
		systemProtected.PUT("/user/:id/reset-password", noRouteHandler)
		systemProtected.DELETE("/user/:id", noRouteHandler)
		systemProtected.GET("/user/list", noRouteHandler)
		systemProtected.POST("/user/export", noRouteHandler)
		// auth module security-event routes (auth/module.go)
		systemProtected.GET("/security-event/list", noRouteHandler)
		systemProtected.POST("/security-event/:id/acknowledge", noRouteHandler)
		systemProtected.POST("/security-event/batch-acknowledge", noRouteHandler)
		systemProtected.POST("/security-event/cleanup", noRouteHandler)
	}

	// lowcode dynamic-module routes (dynamicmodule/module.go)
	lowcodeRead := api.Group("/lowcode/dynamic-modules")
	{
		lowcodeRead.GET("", noRouteHandler)
		lowcodeRead.GET("/schema", noRouteHandler)
		lowcodeRead.GET("/:name", noRouteHandler)
	}
	lowcodeWrite := api.Group("/lowcode/dynamic-modules")
	{
		lowcodeWrite.POST("/generate", noRouteHandler)
		lowcodeWrite.POST("/repair", noRouteHandler)
		lowcodeWrite.POST("/activation-audit", noRouteHandler)
		lowcodeWrite.POST("", noRouteHandler)
		lowcodeWrite.DELETE("/:name", noRouteHandler)
		lowcodeWrite.DELETE("/:name/record", noRouteHandler)
		lowcodeWrite.DELETE("/:name/purge", noRouteHandler)
	}

	// lowcode generator datasources (generator/module.go writeAPI)
	generatorWrite := api.Group("/lowcode/generator")
	{
		generatorWrite.POST("/datasources", noRouteHandler)
		generatorWrite.PUT("/datasources/:id", noRouteHandler)
		generatorWrite.DELETE("/datasources/:id", noRouteHandler)
		generatorWrite.POST("/datasources/:id/test", noRouteHandler)
	}

	return r
}

func noRouteHandler(c *gin.Context) {
	c.Status(http.StatusOK)
}

// engineRouteSet flattens the gin routing table into a set of
// "METHOD path" keys with wildcard params normalized to gin's :param form,
// matching the binding table's notation.
func engineRouteSet(r *gin.Engine) map[string]struct{} {
	set := make(map[string]struct{})
	for _, info := range r.Routes() {
		set[normalizeProbeRouteKey(info.Method, info.Path)] = struct{}{}
	}
	return set
}

// normalizeProbeRouteKey aligns gin's route-tree path notation with the
// binding table's ":param" notation so both can be compared literally.
func normalizeProbeRouteKey(method, path string) string {
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

// TestRequiredAPIRoutesExistOnEngine is the drift guard for the workbench
// permission→route binding table: every entry must exist as a registered
// route on the probe engine. A failure means a route was renamed/removed or
// its method changed — fix the binding table (and any remediation policies
// already created from the stale entry) in the same patch.
//
// Context: audit finding D (PANTHEON_BASE_DESIGN_SUPPLEMENT_AUDIT_20260910.md
// §7.2) — the binding table previously shipped a stale
// "POST /api/v1/system/user/create" entry while the live route is
// "POST /api/v1/system/user", so remediation created a dead policy and the
// real route stayed forbidden.
func TestRequiredAPIRoutesExistOnEngine(t *testing.T) {
	routeSet := engineRouteSet(buildWorkbenchRouteProbeEngine())

	for permissionKey, routes := range RequiredAPIRoutesByPermission() {
		if len(routes) == 0 {
			t.Errorf("permission %q has an empty route binding", permissionKey)
			continue
		}
		for _, entry := range routes {
			key := normalizeProbeRouteKey(entry.Method, entry.Path)
			if _, ok := routeSet[key]; !ok {
				t.Errorf("binding table entry %q -> %q does not match any registered route (stale path or method)", permissionKey, key)
			}
		}
	}
}

// TestRequiredAPIRouteProbeCoversEngineRoutes keeps the probe honest: the
// routes the probe registers must be a superset of the binding table AND the
// probe itself must not silently drop registrations (a removed probe line
// would otherwise turn the guard above into a false pass for stale entries).
// It asserts a few sentinel routes exist in the probe engine regardless of
// the binding table's content.
func TestRequiredAPIRouteProbeCoversEngineRoutes(t *testing.T) {
	routeSet := engineRouteSet(buildWorkbenchRouteProbeEngine())

	sentinels := []string{
		"POST /api/v1/system/user",
		"GET /api/v1/system/user/list",
		"GET /api/v1/system/security-event/list",
		"POST /api/v1/lowcode/dynamic-modules/generate",
		"DELETE /api/v1/lowcode/dynamic-modules/:name/purge",
		"PUT /api/v1/lowcode/generator/datasources/:id",
	}
	for _, key := range sentinels {
		if _, ok := routeSet[key]; !ok {
			t.Errorf("probe engine lost sentinel route %q; update buildWorkbenchRouteProbeEngine to match the live registration", key)
		}
	}
}
