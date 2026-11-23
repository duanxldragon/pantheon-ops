package workload

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

// TestScaleRequestReplicasValidation regresses the scale-to-zero defect: an
// explicit 0 must survive required validation (valid scale action) while
// missing, null, negative and non-numeric replicas stay rejected.
func TestScaleRequestReplicasValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"explicit zero accepted", `{"replicas":0}`, false},
		{"positive accepted", `{"replicas":3}`, false},
		{"missing rejected", `{}`, true},
		{"null rejected", `{"replicas":null}`, true},
		{"negative rejected", `{"replicas":-1}`, true},
		{"non-numeric rejected", `{"replicas":"abc"}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var req ScaleRequest
			err := bindScaleRequestJSON(t, tc.body, &req)
			if (err != nil) != tc.wantErr {
				t.Fatalf("bind %s: err = %v, wantErr = %v", tc.body, err, tc.wantErr)
			}
			if !tc.wantErr && (req.Replicas == nil || *req.Replicas < 0) {
				t.Fatalf("bind %s: Replicas = %v, want non-nil >= 0", tc.body, req.Replicas)
			}
		})
	}
}

// TestWorkloadErrorCodeClassifiesConflict regresses stale-resource-version CAS:
// ErrConflict must surface as code 409, every other provider failure stays 500.
func TestWorkloadErrorCodeClassifiesConflict(t *testing.T) {
	if got := workloadErrorCode(common.NewConflict("k8s.workload.resource_version_conflict")); got != codeConflict {
		t.Fatalf("conflict: got code %d, want %d", got, codeConflict)
	}
	if got := workloadErrorCode(errors.New("provider down")); got != common.CodeError {
		t.Fatalf("generic: got code %d, want %d", got, common.CodeError)
	}
}

// bindScaleRequestJSON binds a JSON body through Gin's ShouldBindJSON and the
// ScaleRequest binding tags, mirroring WorkloadHandler.Scale.
func bindScaleRequestJSON(t *testing.T, body string, dst interface{}) error {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/clusters/1/workloads/default/deployment/app/scale", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(dst)
}