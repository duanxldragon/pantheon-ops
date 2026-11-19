package argocd

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
)

func TestSyncActionErrorHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(oldLogger) })
	cases := []struct {
		name    string
		err     error
		status  int
		code    int
		message string
	}{
		{"in-progress", errors.New("business.cicd.application.syncInProgress"), 409, 409, "business.cicd.application.syncInProgress"},
		{"source-change", common.NewConflict("business.cicd.application.sourceChanged"), 409, 409, "business.cicd.application.sourceChanged"},
		{"authorization-first", common.NewForbidden("permission.denied"), 200, 403, "permission.denied"},
		{"typed-validation-first", common.NewBadRequest("param.invalid"), 200, 400, "param.invalid"},
		{"provider-key-preserved", errors.New(appSyncFailedKey), 200, 500, appSyncFailedKey},
		{"raw-secret-hidden", errors.New("provider token=synthetic-secret; SQL detail"), 200, 500, appSyncFailedKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/sync", func(c *gin.Context) { failSyncAction(c, tc.err, appSyncFailedKey) })
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sync", nil))
			var envelope common.Response
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || rec.Code != tc.status || envelope.Code != tc.code || envelope.Message != tc.message || envelope.Data != nil {
				t.Fatalf("wrong sync HTTP contract: HTTP=%d body=%s decode=%v", rec.Code, rec.Body.String(), err)
			}
		})
	}
	if strings.Contains(logs.String(), "synthetic-secret") || strings.Contains(logs.String(), "SQL detail") || !strings.Contains(logs.String(), "business.cicd.application") {
		t.Fatalf("request diagnostics exposed provider detail or lost subsystem: %s", logs.String())
	}
}
