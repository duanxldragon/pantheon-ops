package upload

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

type stubConfigReader struct {
	values map[string]string
}

func (s stubConfigReader) GetByKey(settingKey string) (string, error) {
	if value, ok := s.values[settingKey]; ok {
		return value, nil
	}
	return "", nil
}

type fakeObjectStorageClient struct {
	bucketExists         bool
	bucketExistsErr      error
	makeBucketErr        error
	putObjectErr         error
	makeBucketCalled     bool
	putObjectCalled      bool
	lastBucket           string
	lastObjectKey        string
	lastPutObjectOptions minio.PutObjectOptions
}

func (f *fakeObjectStorageClient) BucketExists(_ context.Context, bucketName string) (bool, error) {
	f.lastBucket = bucketName
	return f.bucketExists, f.bucketExistsErr
}

func (f *fakeObjectStorageClient) MakeBucket(_ context.Context, bucketName string, _ minio.MakeBucketOptions) error {
	f.makeBucketCalled = true
	f.lastBucket = bucketName
	return f.makeBucketErr
}

func (f *fakeObjectStorageClient) PutObject(_ context.Context, bucketName, objectName string, reader io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	f.putObjectCalled = true
	f.lastBucket = bucketName
	f.lastObjectKey = objectName
	f.lastPutObjectOptions = opts
	buf := make([]byte, 32)
	_, _ = reader.Read(buf)
	return minio.UploadInfo{Bucket: bucketName, Key: objectName}, f.putObjectErr
}

func buildFileHeader(t *testing.T, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)

	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content) + 1024))
	if err != nil {
		t.Fatalf("read form: %v", err)
	}
	files := form.File["file"]
	if len(files) == 0 {
		t.Fatal("missing file header")
	}
	return files[0]
}

func TestServiceStoreRespectsLocalConfigAndReturnsURL(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver":  "local",
			"upload.max_file_size":   "2",
			"upload.allowed_types":   `["png","jpg"]`,
			"upload.local_path":      tempDir,
			"upload.public_base_url": "/files",
		},
	})
	service.now = func() time.Time {
		return time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	}

	fileHeader := buildFileHeader(t, "avatar.png", "image/png", []byte("avatar-demo"))
	stored, err := service.Store(fileHeader, "profile/avatar", "http://localhost:8080")
	if err != nil {
		t.Fatalf("store file: %v", err)
	}

	if !strings.HasPrefix(stored.ObjectKey, "profile/avatar/20260424/") {
		t.Fatalf("unexpected object key: %s", stored.ObjectKey)
	}
	if stored.URL != "http://localhost:8080/files/"+stored.ObjectKey {
		t.Fatalf("unexpected file url: %s", stored.URL)
	}

	absolutePath, err := service.ResolveLocalPath(stored.ObjectKey)
	if err != nil {
		t.Fatalf("resolve local path: %v", err)
	}
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if string(data) != "avatar-demo" {
		t.Fatalf("unexpected stored content: %s", string(data))
	}
}

func TestServiceStoreUsesS3ClientWhenConfigured(t *testing.T) {
	fakeClient := &fakeObjectStorageClient{bucketExists: false}
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver":       "s3",
			"upload.max_file_size":        "2",
			"upload.allowed_types":        `["png","jpg"]`,
			"upload.s3_endpoint":          "https://minio.example.com",
			"upload.s3_bucket":            "pantheon",
			"upload.s3_region":            "cn-east-1",
			"upload.s3_access_key_id":     "demo-key",
			"upload.s3_secret_access_key": "demo-secret",
			"upload.public_base_url":      "https://cdn.example.com/files",
		},
	})
	service.now = func() time.Time {
		return time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	}
	service.s3ClientFactory = func(cfg *Config) (objectStorageClient, error) {
		if cfg.S3Bucket != "pantheon" {
			t.Fatalf("unexpected bucket in config: %s", cfg.S3Bucket)
		}
		return fakeClient, nil
	}

	fileHeader := buildFileHeader(t, "avatar.png", "image/png", []byte("avatar-demo"))
	stored, err := service.Store(fileHeader, "profile/avatar", "http://localhost:8080")
	if err != nil {
		t.Fatalf("store s3 file: %v", err)
	}

	if !fakeClient.makeBucketCalled {
		t.Fatal("expected missing bucket to trigger MakeBucket")
	}
	if !fakeClient.putObjectCalled {
		t.Fatal("expected PutObject to be called")
	}
	if fakeClient.lastPutObjectOptions.ContentType != "image/png" {
		t.Fatalf("unexpected content type: %s", fakeClient.lastPutObjectOptions.ContentType)
	}
	if stored.URL != "https://cdn.example.com/files/"+stored.ObjectKey {
		t.Fatalf("unexpected s3 url: %s", stored.URL)
	}
}

func TestServiceStoreRejectsDisallowedExtension(t *testing.T) {
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver": "local",
			"upload.max_file_size":  "2",
			"upload.allowed_types":  `["png"]`,
			"upload.local_path":     t.TempDir(),
		},
	})
	fileHeader := buildFileHeader(t, "avatar.gif", "image/gif", []byte("gif"))
	_, err := service.Store(fileHeader, "profile", "http://localhost:8080")
	if err == nil || err.Error() != "upload.file.type_not_allowed" {
		t.Fatalf("expected type_not_allowed, got %v", err)
	}
}

func TestServiceStoreRejectsOversizedFile(t *testing.T) {
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver": "local",
			"upload.max_file_size":  "1",
			"upload.allowed_types":  `["png"]`,
			"upload.local_path":     t.TempDir(),
		},
	})
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", bytes.Repeat([]byte("a"), 1024*1024+1))
	_, err := service.Store(fileHeader, "profile", "http://localhost:8080")
	if err == nil || err.Error() != "upload.file.too_large" {
		t.Fatalf("expected too_large, got %v", err)
	}
}

func TestServiceStoreRequiresS3Credentials(t *testing.T) {
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver": "s3",
			"upload.max_file_size":  "2",
			"upload.allowed_types":  `["png"]`,
			"upload.s3_endpoint":    "https://minio.example.com",
			"upload.s3_bucket":      "pantheon",
		},
	})
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", []byte("ok"))
	_, err := service.Store(fileHeader, "profile", "http://localhost:8080")
	if err == nil || err.Error() != "upload.s3.credentials.required" {
		t.Fatalf("expected s3 credentials required, got %v", err)
	}
}

func TestServiceStoreHandlesS3UploadFailure(t *testing.T) {
	fakeClient := &fakeObjectStorageClient{
		bucketExists: true,
		putObjectErr: errors.New("upload failed"),
	}
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver":       "s3",
			"upload.max_file_size":        "2",
			"upload.allowed_types":        `["png"]`,
			"upload.s3_endpoint":          "https://minio.example.com",
			"upload.s3_bucket":            "pantheon",
			"upload.s3_access_key_id":     "demo-key",
			"upload.s3_secret_access_key": "demo-secret",
		},
	})
	service.s3ClientFactory = func(cfg *Config) (objectStorageClient, error) {
		return fakeClient, nil
	}
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", []byte("ok"))
	_, err := service.Store(fileHeader, "profile", "http://localhost:8080")
	if err == nil || err.Error() != "upload.s3.upload.error" {
		t.Fatalf("expected s3 upload error, got %v", err)
	}
}

func TestResolveLocalPathBlocksTraversal(t *testing.T) {
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver": "local",
			"upload.local_path":     t.TempDir(),
		},
	})
	if _, err := service.ResolveLocalPath("../evil.txt"); err == nil {
		t.Fatal("expected traversal to be blocked")
	}
}

func TestBuildFileURLFallsBackToDefaultServePath(t *testing.T) {
	url := BuildFileURL("", "http://localhost:8080", filepath.ToSlash(filepath.Join("profile", "a.png")))
	if url != "http://localhost:8080/api/v1/system/upload/files/profile/a.png" {
		t.Fatalf("unexpected fallback url: %s", url)
	}
}

func TestBuildS3FileURLUsesEndpointWhenPublicBaseNotConfigured(t *testing.T) {
	url := buildS3FileURL(&Config{
		S3Endpoint: "https://minio.example.com",
		S3Bucket:   "pantheon",
	}, "profile/a.png")
	if url != "https://minio.example.com/pantheon/profile/a.png" {
		t.Fatalf("unexpected s3 fallback url: %s", url)
	}
}
