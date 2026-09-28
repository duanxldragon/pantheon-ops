package config

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	uploadpkg "github.com/duanxldragon/pantheon-base/backend/pkg/upload"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// Download authorization slice (tenant-core-data-infrastructure): the S3
// driver must be servable through the authorized backend route with the same
// tenant-namespace isolation the local driver enforces. Cross-tenant reads
// must be denied before any object-store call is made (no existence leak),
// and compat must keep working end to end.

type fakeS3UploadClient struct {
	content  []byte
	isErr    bool
	statHit  bool
	getHit   bool
	lastStat string
}

func (f *fakeS3UploadClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	return true, nil
}

func (f *fakeS3UploadClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	return nil
}

func (f *fakeS3UploadClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	return minio.UploadInfo{Bucket: bucketName, Key: objectName}, nil
}

func (f *fakeS3UploadClient) GetScopedObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (io.ReadSeekCloser, error) {
	f.getHit = true
	f.lastStat = objectName
	if f.isErr {
		return nil, context.Canceled
	}
	return nopUploadReader{Reader: strings.NewReader(string(f.content))}, nil
}

func (f *fakeS3UploadClient) StatScopedObject(ctx context.Context, bucketName, objectName string, opts minio.StatObjectOptions) (minio.ObjectInfo, error) {
	f.statHit = true
	f.lastStat = objectName
	if f.isErr {
		return minio.ObjectInfo{}, context.Canceled
	}
	return minio.ObjectInfo{Key: objectName, Size: int64(len(f.content)), ContentType: "application/octet-stream"}, nil
}

type nopUploadReader struct{ *strings.Reader }

func (n nopUploadReader) Close() error { return nil }

func newS3ServeTestHandler(t *testing.T, client *fakeS3UploadClient) *SettingHandler {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := uploadpkg.NewService(stubUploadConfigReader{values: map[string]string{
		"upload.storage_driver": "s3",
		"upload.s3_endpoint":    "https://minio.example.com",
		"upload.s3_bucket":      "pantheon",
	}})
	svc.SetObjectStorageClientFactory(func(cfg *uploadpkg.Config) (uploadpkg.ObjectStorageClient, error) {
		return client, nil
	})
	return &SettingHandler{uploadService: svc}
}

func serveRequest(t *testing.T, handler *SettingHandler, tenantCtx *tenant.Context, key string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/system/upload/files/"+key, nil)
	c.Params = gin.Params{{Key: "filepath", Value: "/" + key}}
	if tenantCtx != nil {
		tenant.SetGin(c, tenantCtx)
	}
	handler.ServeUploadedFile(c)
	return recorder
}

func TestServeUploadedFileS3_CrossTenantDenied(t *testing.T) {
	client := &fakeS3UploadClient{content: []byte("tenant-202-secret")}
	handler := newS3ServeTestHandler(t, client)

	recorder := serveRequest(t, handler, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t202/20260915/x.png")

	if client.statHit || client.getHit {
		t.Fatalf("cross-tenant download must be denied before any object-store call (existence leak)")
	}
	if !strings.Contains(recorder.Body.String(), "upload.file.not_found") {
		t.Fatalf("expected not_found error, got %s", recorder.Body.String())
	}
}

func TestServeUploadedFileS3_OwnTenantServed(t *testing.T) {
	client := &fakeS3UploadClient{content: []byte("tenant-101-content")}
	handler := newS3ServeTestHandler(t, client)

	recorder := serveRequest(t, handler, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t101/general/20260915/a.png")

	if recorder.Code != http.StatusOK {
		t.Fatalf("own-tenant download must succeed, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !client.statHit || !client.getHit {
		t.Fatalf("object store must be consulted for own-tenant download")
	}
	if got := recorder.Body.String(); got != "tenant-101-content" {
		t.Fatalf("expected object content, got %q", got)
	}
	if got := recorder.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	if got := recorder.Header().Get("Content-Length"); got != "18" {
		t.Fatalf("expected Content-Length 18, got %q", got)
	}
}

func TestServeUploadedFileS3_CompatServesWithoutTenantPrefix(t *testing.T) {
	client := &fakeS3UploadClient{content: []byte("legacy-object")}
	handler := newS3ServeTestHandler(t, client)

	recorder := serveRequest(t, handler, &tenant.Context{TenantID: 0, Mode: tenant.ModeCompat}, "general/20260915/legacy.png")

	if recorder.Code != http.StatusOK {
		t.Fatalf("compat download must succeed, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Body.String(); got != "legacy-object" {
		t.Fatalf("expected legacy object content, got %q", got)
	}
}

func TestServeUploadedFileS3_ObjectStoreErrorIsNotFound(t *testing.T) {
	client := &fakeS3UploadClient{isErr: true}
	handler := newS3ServeTestHandler(t, client)

	recorder := serveRequest(t, handler, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t101/general/20260915/missing.png")

	if !strings.Contains(recorder.Body.String(), "upload.file.not_found") {
		t.Fatalf("object-store errors must not leak store internals, got %s", recorder.Body.String())
	}
}

// EnforceTenantObjectScope: the guard is enforced from the resolved context only.
func TestEnforceTenantObjectScope(t *testing.T) {
	if err := uploadpkg.EnforceTenantObjectScope(nil, "t101/x.png"); err != nil {
		t.Fatalf("missing context (public/compat path) must pass through, got %v", err)
	}
	if err := uploadpkg.EnforceTenantObjectScope(&tenant.Context{TenantID: 0, Mode: tenant.ModeCompat}, "t101/x.png"); err != nil {
		t.Fatalf("compat must keep legacy keyspace, got %v", err)
	}
	if err := uploadpkg.EnforceTenantObjectScope(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t202/x.png"); err == nil {
		t.Fatalf("cross-tenant key must be rejected")
	}
	if err := uploadpkg.EnforceTenantObjectScope(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t101/x.png"); err != nil {
		t.Fatalf("own-tenant key must pass, got %v", err)
	}
	// Tenant-prefixed segment must be the FIRST segment (t101x/ is not t101/).
	if err := uploadpkg.EnforceTenantObjectScope(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t101x/y.png"); err == nil {
		t.Fatalf("lookalike prefix t101x/ must be rejected")
	}
}

// Local-driver regression: the refactored ServeUploadedFile keeps local serving
// (guard now shared with the S3 path via EnforceTenantObjectScope).
func TestServeUploadedFile_LocalDriverStillServes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	root := t.TempDir()
	rel := filepath.Join("t101", "general", "a.png")
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte("local-bytes"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	handler := &SettingHandler{uploadService: uploadpkg.NewService(stubUploadConfigReader{values: map[string]string{
		"upload.storage_driver": "local",
		"upload.local_path":     root,
	}})}

	recorder := serveRequest(t, handler, &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}, "t101/general/a.png")

	if recorder.Code != http.StatusOK {
		t.Fatalf("local download must succeed, got status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Body.String(); got != "local-bytes" {
		t.Fatalf("expected local content, got %q", got)
	}
}
