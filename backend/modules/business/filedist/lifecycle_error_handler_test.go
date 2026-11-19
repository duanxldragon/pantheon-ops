package filedist

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
)

func TestFileDistLifecycleErrorHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name    string
		err     error
		status  int
		message string
	}{
		{"interrupted", errors.New("business.filedist.task.executionInterrupted"), 500, "business.filedist.task.executionInterrupted"},
		{"transfer", errors.New("business.filedist.task.transferFailed"), 500, "business.filedist.task.transferFailed"},
		{"lease", errors.New("business.filedist.task.executionBusy"), 409, "business.filedist.task.executionBusy"},
		{"future", errors.New("business.filedist.task.notDue"), 409, "business.filedist.task.notDue"},
		{"terminal", errors.New("business.filedist.task.notCancellable"), 409, "business.filedist.task.notCancellable"},
		{"context", errors.New("business.filedist.task.invalidContext"), 400, "business.filedist.task.invalidContext"},
		{"wrapped-canonical", fmt.Errorf("%w", common.NewConflict("business.filedist.task.notDue")), 409, "business.filedist.task.notDue"},
		{"live-authorization", common.NewForbidden("permission.denied"), 403, "permission.denied"},
		{"typed-class-first", common.NewForbidden("business.filedist.task.notDue"), 403, "business.filedist.task.notDue"},
		{"raw-secret-hidden", errors.New("SQL password=synthetic-secret; remote response"), 500, "create_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/tasks", func(c *gin.Context) {
				failFileDistTaskError(c, tc.err, http.StatusInternalServerError, "create_failed")
			})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/tasks", nil))
			var envelope common.Response
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil || rec.Code != tc.status || envelope.Code != tc.status || envelope.Message != tc.message || envelope.Data != nil {
				t.Fatalf("wrong lifecycle HTTP contract: HTTP=%d body=%s decode=%v", rec.Code, rec.Body.String(), err)
			}
		})
	}
}
