package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
	contextValueKey      interface{}
	lastContextValue     interface{}
	lastPutObjectOptions minio.PutObjectOptions
}

func (f *fakeObjectStorageClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	f.lastBucket = bucketName
	return f.bucketExists, f.bucketExistsErr
}

func (f *fakeObjectStorageClient) MakeBucket(ctx context.Context, bucketName string, _ minio.MakeBucketOptions) error {
	f.makeBucketCalled = true
	f.lastBucket = bucketName
	return f.makeBucketErr
}

func (f *fakeObjectStorageClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, _ int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	if f.contextValueKey != nil {
		f.lastContextValue = ctx.Value(f.contextValueKey)
	}
	f.putObjectCalled = true
	f.lastBucket = bucketName
	f.lastObjectKey = objectName
	f.lastPutObjectOptions = opts
	buf := make([]byte, 32)
	_, _ = reader.Read(buf)
	return minio.UploadInfo{Bucket: bucketName, Key: objectName}, f.putObjectErr
}

// pngPayload 构造带真实 PNG magic bytes 的测试内容，配合 verifyImageContent 内容嗅探。
func pngPayload(extra []byte) []byte {
	head := []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
	return append(head, extra...)
}

// gifPayload 构造带真实 GIF magic bytes 的测试内容。
func gifPayload(extra []byte) []byte {
	head := []byte("GIF89a")
	return append(head, extra...)
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

	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload([]byte("avatar-demo")))
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
	// #nosec G304 -- the path is derived from the service's normalized object key.
	data, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if string(data) != string(pngPayload([]byte("avatar-demo"))) {
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

	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload([]byte("avatar-demo")))
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

func TestServiceStoreUsesRealS3WhenConfigured(t *testing.T) {
	endpoint, accessKey, secretKey, bucket, region := loadRealS3TestConfig(t)
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.storage_driver":       "s3",
			"upload.max_file_size":        "2",
			"upload.allowed_types":        `["png"]`,
			"upload.s3_endpoint":          endpoint,
			"upload.s3_bucket":            bucket,
			"upload.s3_region":            region,
			"upload.s3_access_key_id":     accessKey,
			"upload.s3_secret_access_key": secretKey,
		},
	})

	payload := pngPayload([]byte("pantheon-real-s3-upload"))
	fileHeader := buildFileHeader(t, "integration.png", "image/png", payload)
	storeCtx, storeCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer storeCancel()
	stored, err := service.StoreWithContext(
		storeCtx,
		fileHeader,
		"integration/upload",
		"http://localhost:8080",
	)
	if err != nil {
		t.Fatalf("store file in real S3-compatible service: %v", err)
	}

	client := newRealS3VerificationClient(t, endpoint, accessKey, secretKey, region)
	registerRealS3Cleanup(t, client, bucket, stored.ObjectKey)
	verifyRealS3Upload(t, client, endpoint, bucket, stored, payload)
}

func loadRealS3TestConfig(t *testing.T) (string, string, string, string, string) {
	t.Helper()
	endpoint := strings.TrimSpace(os.Getenv("PANTHEON_TEST_S3_ENDPOINT"))
	if endpoint == "" {
		t.Skip("PANTHEON_TEST_S3_ENDPOINT is not configured")
	}
	accessKey := strings.TrimSpace(os.Getenv("PANTHEON_TEST_S3_ACCESS_KEY"))
	secretKey := strings.TrimSpace(os.Getenv("PANTHEON_TEST_S3_SECRET_KEY"))
	if accessKey == "" || secretKey == "" {
		t.Fatal("PANTHEON_TEST_S3_ACCESS_KEY and PANTHEON_TEST_S3_SECRET_KEY are required")
	}
	region := strings.TrimSpace(os.Getenv("PANTHEON_TEST_S3_REGION"))
	if region == "" {
		region = "us-east-1"
	}
	return endpoint, accessKey, secretKey, fmt.Sprintf("pantheon-upload-it-%d", time.Now().UnixNano()), region
}

func newRealS3VerificationClient(t *testing.T, endpoint, accessKey, secretKey, region string) *minio.Client {
	t.Helper()
	host, secure, err := normalizeS3Endpoint(endpoint)
	if err != nil {
		t.Fatalf("normalize integration endpoint: %v", err)
	}
	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
		Region: region,
	})
	if err != nil {
		t.Fatalf("create integration verification client: %v", err)
	}
	return client
}

func registerRealS3Cleanup(t *testing.T, client *minio.Client, bucket, objectKey string) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
			t.Errorf("remove integration object: %v", err)
		}
		if err := client.RemoveBucket(ctx, bucket); err != nil {
			t.Errorf("remove integration bucket: %v", err)
		}
	})
}

func verifyRealS3Upload(t *testing.T, client *minio.Client, endpoint, bucket string, stored *StoredFile, payload []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	info, err := client.StatObject(ctx, bucket, stored.ObjectKey, minio.StatObjectOptions{})
	if err != nil {
		t.Fatalf("stat uploaded integration object: %v", err)
	}
	if info.Size != int64(len(payload)) || info.ContentType != "image/png" {
		t.Fatalf("unexpected uploaded object metadata: size=%d contentType=%q", info.Size, info.ContentType)
	}
	object, err := client.GetObject(ctx, bucket, stored.ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("open uploaded integration object: %v", err)
	}
	defer func() {
		_ = object.Close()
	}()
	storedPayload, err := io.ReadAll(object)
	if err != nil {
		t.Fatalf("read uploaded integration object: %v", err)
	}
	if !bytes.Equal(storedPayload, payload) {
		t.Fatalf("unexpected uploaded object content: %q", storedPayload)
	}
	wantURL := strings.TrimRight(endpoint, "/") + "/" + bucket + "/" + stored.ObjectKey
	if stored.URL != wantURL {
		t.Fatalf("unexpected real S3 object URL: got %q want %q", stored.URL, wantURL)
	}
}

func TestServiceStoreWithContextPassesContextToS3Client(t *testing.T) {
	fakeClient := &fakeObjectStorageClient{bucketExists: true}
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

	type requestIDKey string
	ctx := context.WithValue(context.Background(), requestIDKey("request-id"), "req-upload-001")
	fakeClient.contextValueKey = requestIDKey("request-id")
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload([]byte("avatar-demo")))
	if _, err := service.StoreWithContext(ctx, fileHeader, "profile/avatar", "http://localhost:8080"); err != nil {
		t.Fatalf("store s3 file with context: %v", err)
	}

	if fakeClient.lastContextValue != "req-upload-001" {
		t.Fatalf("expected request context to reach s3 client, got %#v", fakeClient.lastContextValue)
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
	fileHeader := buildFileHeader(t, "avatar.gif", "image/gif", gifPayload([]byte("gif")))
	_, err := service.Store(fileHeader, "profile", "http://localhost:8080")
	if err == nil || err.Error() != "upload.file.type_not_allowed" {
		t.Fatalf("expected type_not_allowed, got %v", err)
	}
}

func TestServiceLoadConfigIncludesWebpAndGifByDefault(t *testing.T) {
	service := NewService(stubConfigReader{})
	cfg, err := service.LoadConfig()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	expected := map[string]struct{}{
		"jpg":  {},
		"jpeg": {},
		"png":  {},
		"webp": {},
		"gif":  {},
		"pdf":  {},
		"doc":  {},
		"docx": {},
		"xls":  {},
		"xlsx": {},
		"zip":  {},
		"gz":   {},
		"tgz":  {},
		"tar":  {},
	}
	if len(cfg.AllowedTypes) != len(expected) {
		t.Fatalf("unexpected allowed types length: %d", len(cfg.AllowedTypes))
	}
	for _, allowedType := range cfg.AllowedTypes {
		if _, ok := expected[allowedType]; !ok {
			t.Fatalf("unexpected allowed type: %s", allowedType)
		}
		delete(expected, allowedType)
	}
	if len(expected) != 0 {
		t.Fatalf("missing allowed types: %+v", expected)
	}
}

func TestServiceStoreAllowsWebpAndGifByDefault(t *testing.T) {
	tempDir := t.TempDir()
	service := NewService(stubConfigReader{
		values: map[string]string{
			"upload.local_path": tempDir,
		},
	})
	service.now = func() time.Time {
		return time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	}

	imagePayloads := map[string][]byte{
		"avatar.webp": append([]byte("RIFF\x24\x00\x00\x00WEBPVP8 "), []byte("avatar-demo")...),
		"avatar.gif":  gifPayload([]byte("avatar-demo")),
	}
	for _, filename := range []string{"avatar.webp", "avatar.gif"} {
		fileHeader := buildFileHeader(t, filename, "image/"+strings.TrimPrefix(filepath.Ext(filename), "."), imagePayloads[filename])
		stored, err := service.Store(fileHeader, "profile/avatar", "http://localhost:8080")
		if err != nil {
			t.Fatalf("store %s: %v", filename, err)
		}
		if !strings.HasSuffix(stored.ObjectKey, filepath.Ext(filename)) {
			t.Fatalf("unexpected object key for %s: %s", filename, stored.ObjectKey)
		}
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
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload(bytes.Repeat([]byte("a"), 1024*1024+1)))
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
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload([]byte("ok")))
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
	fileHeader := buildFileHeader(t, "avatar.png", "image/png", pngPayload([]byte("ok")))
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
	cases := []string{
		"../evil.txt",
		"profile/../../evil.txt",
		"profile/..hidden/evil.txt",
		`C:\windows\system.ini`,
	}
	for _, objectKey := range cases {
		if _, err := service.ResolveLocalPath(objectKey); err == nil {
			t.Fatalf("expected traversal to be blocked for %q", objectKey)
		}
	}
}

func TestNormalizeObjectKeyAllowsGeneratedUploadKeys(t *testing.T) {
	key, err := NormalizeObjectKey("/profile/avatar/20260424/0b4a5f46-7c3d-4f9e-9a55-9533e4da3d60.png")
	if err != nil {
		t.Fatalf("normalize generated key: %v", err)
	}
	if key != "profile/avatar/20260424/0b4a5f46-7c3d-4f9e-9a55-9533e4da3d60.png" {
		t.Fatalf("unexpected normalized key: %s", key)
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
