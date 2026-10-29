package role

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// F14 regression: a tenant-scoped role miss used to surface as request.failed
// (code 500) because every service error shared one handler branch. The
// envelope must now distinguish "the role does not exist in your tenant"
// (404 role.not_found) from a genuine server fault (500 request.failed).

func runRoleErrorResponse(t *testing.T, err error) (int, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/role/1/users", nil)

	roleErrorResponse(c, err)

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not a valid envelope: %v (%q)", err, w.Body.String())
	}
	return body.Code, body.Message
}

func TestRoleErrorResponseMapsRecordNotFoundTo404(t *testing.T) {
	for name, err := range map[string]error{
		"gorm record not found": gorm.ErrRecordNotFound,
		"wrapped gorm error":    errors.Join(errors.New("getRole"), gorm.ErrRecordNotFound),
		"common sentinel":       common.NewNotFound("role.not_found"),
	} {
		t.Run(name, func(t *testing.T) {
			code, message := runRoleErrorResponse(t, err)
			if code != common.CodeNotFound {
				t.Fatalf("expected code %d for a not-found error, got %d", common.CodeNotFound, code)
			}
			if message != errRoleNotFound {
				t.Fatalf("expected message %q, got %q", errRoleNotFound, message)
			}
		})
	}
}

func TestRoleErrorResponseKeepsGeneric500ForOtherErrors(t *testing.T) {
	code, message := runRoleErrorResponse(t, errors.New("driver: bad connection"))
	if code != common.CodeError {
		t.Fatalf("expected code %d for an unrelated error, got %d", common.CodeError, code)
	}
	if message != common.MessageRequestFailed {
		t.Fatalf("expected message %q, got %q", common.MessageRequestFailed, message)
	}
}
