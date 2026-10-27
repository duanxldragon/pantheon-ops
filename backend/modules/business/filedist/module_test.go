package filedist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestFileDistRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	module := &FileDistModule{handler: NewFileDistHandler(nil, nil)}
	module.RegisterRoutes(api)

	for _, path := range []string{
		"/api/v1/business/filedist/files",
		"/api/v1/filedist/files",
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		router.ServeHTTP(recorder, request)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode %s response: %v", path, err)
		}
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s to require authentication, got envelope code %d: %s", path, response.Code, recorder.Body.String())
		}
	}
}
