package http

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestMetadataFailureDoesNotClassifyOrLogRawError(t *testing.T) {
	route := auditpolicy.Route{Method: "POST", Template: "/api/v1/business/response/:id"}
	if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{route}); err != nil {
		t.Fatal(err)
	}
	core, observed := observer.New(zap.DebugLevel)
	previous := logging.Logger
	logging.Logger = zap.New(core)
	defer func() { logging.Logger = previous }()
	engine := gin.New()
	engine.POST(route.Template, func(c *gin.Context) {
		FailWithError(c, 500, errors.New("provider.synthetic_secret"), "client.synthetic_secret")
	})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest("POST", "/api/v1/business/response/7", nil))
	if strings.Contains(response.Body.String(), "synthetic_secret") {
		t.Fatal("provider-owned i18n-looking text returned")
	}
	for _, entry := range observed.All() {
		for _, value := range entry.ContextMap() {
			if text, ok := value.(string); ok && strings.Contains(text, "synthetic_secret") {
				t.Fatal("raw provider error logged")
			}
		}
	}
}
