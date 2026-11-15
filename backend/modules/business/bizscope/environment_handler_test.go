package bizscope

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestEnvironmentFailureMasksDatabaseErrorAndKeepsRequestDiagnostic(t *testing.T) {
	core, observed := observer.New(zap.ErrorLevel)
	previous := logging.Logger
	logging.Logger = zap.New(core)
	t.Cleanup(func() { logging.Logger = previous })
	cause := errors.New("sql: environment query failed")
	engine := gin.New()
	engine.GET("/environments", func(c *gin.Context) {
		c.Set("requestId", "environment-error-regression")
		failEnvironmentError(c, cause)
	})
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, httptest.NewRequest("GET", "/environments", nil))
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != common.CodeError || envelope.Message != "request.failed" || strings.Contains(response.Body.String(), cause.Error()) {
		t.Fatalf("database details leaked or generic error missing: %s", response.Body.String())
	}
	entries := observed.FilterMessage("request failed with unclassified error").All()
	if len(entries) != 1 {
		t.Fatalf("expected one underlying error diagnostic, got %d", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["requestId"] != "environment-error-regression" || fields["error"] != cause.Error() {
		t.Fatalf("missing correlated error evidence: %+v", fields)
	}
}
