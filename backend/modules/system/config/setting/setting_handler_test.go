package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	uploadpkg "github.com/duanxldragon/pantheon-base/backend/pkg/upload"
	"github.com/gin-gonic/gin"
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
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("avatar-demo"), 0o600); err != nil {
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

func TestServeUploadedFileRejectsAnotherTenantNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	target := filepath.Join(root, "t202", "general", "20260913", "secret.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("tenant-202-secret"), 0o600); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	handler := NewSettingHandler(nil, uploadpkg.NewService(stubUploadConfigReader{values: map[string]string{
		"upload.storage_driver": "local", "upload.local_path": root,
	}}))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/t202/general/20260913/secret.txt", nil)
	c.Params = gin.Params{{Key: "filepath", Value: "/t202/general/20260913/secret.txt"}}
	tenant.SetGin(c, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})

	handler.ServeUploadedFile(c)

	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "upload.file.not_found") {
		t.Fatalf("cross-tenant download must be denied, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestServeUploadedFileAllowsOwnTenantNamespace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	target := filepath.Join(root, "t101", "general", "20260913", "own.txt")
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(target, []byte("tenant-101-file"), 0o600); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	handler := NewSettingHandler(nil, uploadpkg.NewService(stubUploadConfigReader{values: map[string]string{
		"upload.storage_driver": "local", "upload.local_path": root,
	}}))
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/files/t101/general/20260913/own.txt", nil)
	c.Params = gin.Params{{Key: "filepath", Value: "/t101/general/20260913/own.txt"}}
	tenant.SetGin(c, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti})

	handler.ServeUploadedFile(c)

	if recorder.Code != http.StatusOK || recorder.Body.String() != "tenant-101-file" {
		t.Fatalf("own-tenant download should be served, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
