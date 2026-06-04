package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	uploadpkg "pantheon-platform/backend/pkg/upload"
)

type stubUploadConfigReader struct {
	values map[string]string
}

func (s stubUploadConfigReader) GetByKey(settingKey string) (string, error) {
	if value, ok := s.values[settingKey]; ok {
		return value, nil
	}
	return "", nil
}

func TestServeUploadedFileServesLocalFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	root := t.TempDir()
	target := filepath.Join(root, "profile", "avatar.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("avatar-demo"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	handler := NewSettingHandler(nil, uploadpkg.NewService(stubUploadConfigReader{
		values: map[string]string{
			"upload.storage_driver": "local",
			"upload.local_path":     root,
		},
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/files/profile/avatar.txt", nil)
	context.Params = gin.Params{{Key: "filepath", Value: "/profile/avatar.txt"}}

	handler.ServeUploadedFile(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "avatar-demo" {
		t.Fatalf("expected served file body, got %q", body)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", contentType)
	}
}

func TestServeUploadedFileRejectsTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewSettingHandler(nil, uploadpkg.NewService(stubUploadConfigReader{
		values: map[string]string{
			"upload.storage_driver": "local",
			"upload.local_path":     t.TempDir(),
		},
	}))

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/files/../secret.txt", nil)
	context.Params = gin.Params{{Key: "filepath", Value: "/../secret.txt"}}

	handler.ServeUploadedFile(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "\"upload.file.not_found\"") {
		t.Fatalf("expected upload.file.not_found response, got %s", body)
	}
}
