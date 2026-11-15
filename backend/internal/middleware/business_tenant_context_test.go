package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestBusinessTenantContextNilFailsBeforeResolution(t *testing.T) {
	engine := gin.New()
	engine.Use(BusinessTenantContextMiddleware(nil, nil, nil, true))
	engine.POST("/business/:id", func(c *gin.Context) { t.Fatal("nil dependency reached handler") })
	request := httptest.NewRequest("POST", "/business/7", nil)
	request.Header.Set("X-Tenant-Id", "202")
	engine.ServeHTTP(httptest.NewRecorder(), request)
}
