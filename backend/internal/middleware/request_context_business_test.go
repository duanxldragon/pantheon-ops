package middleware

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http/httptest"
	"testing"
)

func TestBusinessCorrelationExactCanonicalAliasAndLegacy(t *testing.T) {
	for _, template := range []string{"/api/v1/business/correlation/datasources/:id", "/api/v1/correlation/datasources/:id"} {
		if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{{Method: "POST", Template: template}}); err != nil {
			t.Fatal(err)
		}
	}
	engine := gin.New()
	engine.Use(RequestContextMiddleware())
	for _, template := range []string{"/api/v1/business/correlation/datasources/:id", "/api/v1/correlation/datasources/:id"} {
		engine.POST(template, func(c *gin.Context) { c.AbortWithStatus(403) })
	}
	engine.POST("/api/v1/legacy-correlation", func(c *gin.Context) { c.Status(204) })
	for _, path := range []string{"/api/v1/business/correlation/datasources/7", "/api/v1/correlation/datasources/7", "/api/v1/legacy-correlation"} {
		req := httptest.NewRequest("POST", path, nil)
		req.Header.Set(common.HeaderRequestID, "allowed-client-sentinel")
		req.Header.Set(common.HeaderTraceID, "other-client-sentinel")
		res := httptest.NewRecorder()
		engine.ServeHTTP(res, req)
		id := res.Header().Get(common.HeaderRequestID)
		if path == "/api/v1/legacy-correlation" {
			if id != "allowed-client-sentinel" {
				t.Fatal("legacy correlation changed")
			}
		} else if _, err := uuid.Parse(id); err != nil {
			t.Fatal("opted ID not server UUID")
		}
	}
}
