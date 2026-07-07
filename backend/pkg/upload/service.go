package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const defaultServePath = "/api/v1/system/upload/files"

type ConfigReader interface {
	GetByKey(settingKey string) (string, error)
}

type Config struct {
	StorageDriver string
	MaxFileSizeMB int64
	AllowedTypes  []string
	LocalPath     string
	PublicBaseURL string
	S3Endpoint    string
	S3Bucket      string
	S3Region      string
	S3AccessKeyID string
	S3SecretKey   string
}

type StoredFile struct {
	FileName     string `json:"fileName"`
	OriginalName string `json:"originalName"`
	ObjectKey    string `json:"objectKey"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	ContentType  string `json:"contentType"`
}

type objectStorageClient interface {
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
}

type Service struct {
	reader          ConfigReader
	now             func() time.Time
	s3ClientFactory func(cfg *Config) (objectStorageClient, error)
}

func NewService(reader ConfigReader) *Service {
	return &Service{
		reader:          reader,
		now:             time.Now,
		s3ClientFactory: newS3Client,
	}
}

func (s *Service) LoadConfig() (*Config, error) {
	if s.reader == nil {
		return nil, errors.New("upload.config.unavailable")
	}

	cfg := &Config{
		StorageDriver: "local",
		MaxFileSizeMB: 20,
		AllowedTypes:  []string{"jpg", "jpeg", "png", "pdf", "doc", "docx", "xls", "xlsx", "zip", "gz", "tgz", "tar"},
		LocalPath:     "./uploads",
		S3Region:      "us-east-1",
	}

	if value, err := s.reader.GetByKey("upload.storage_driver"); err == nil && strings.TrimSpace(value) != "" {
		cfg.StorageDriver = strings.ToLower(strings.TrimSpace(value))
	}
	if value, err := s.reader.GetByKey("upload.max_file_size"); err == nil && strings.TrimSpace(value) != "" {
		size, parseErr := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if parseErr != nil || size <= 0 {
			return nil, errors.New("upload.config.invalid_max_file_size")
		}
		cfg.MaxFileSizeMB = size
	}
	if value, err := s.reader.GetByKey("upload.allowed_types"); err == nil && strings.TrimSpace(value) != "" {
		var items []string
		if parseErr := json.Unmarshal([]byte(value), &items); parseErr != nil {
			return nil, errors.New("upload.config.invalid_allowed_types")
		}
		cfg.AllowedTypes = normalizeAllowedTypes(items)
	}
	if value, err := s.reader.GetByKey("upload.local_path"); err == nil && strings.TrimSpace(value) != "" {
		cfg.LocalPath = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.public_base_url"); err == nil {
		cfg.PublicBaseURL = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.s3_endpoint"); err == nil {
		cfg.S3Endpoint = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.s3_bucket"); err == nil {
		cfg.S3Bucket = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.s3_region"); err == nil && strings.TrimSpace(value) != "" {
		cfg.S3Region = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.s3_access_key_id"); err == nil {
		cfg.S3AccessKeyID = strings.TrimSpace(value)
	}
	if value, err := s.reader.GetByKey("upload.s3_secret_access_key"); err == nil {
		cfg.S3SecretKey = strings.TrimSpace(value)
	}
	return cfg, nil
}

func (s *Service) MaxBytes() (int64, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return 0, err
	}
	return cfg.MaxFileSizeMB * 1024 * 1024, nil
}

func (s *Service) Store(fileHeader *multipart.FileHeader, scope, requestBaseURL string) (*StoredFile, error) {
	return s.StoreWithContext(context.Background(), fileHeader, scope, requestBaseURL)
}

func (s *Service) StoreWithContext(ctx context.Context, fileHeader *multipart.FileHeader, scope, requestBaseURL string) (*StoredFile, error) {
	if fileHeader == nil {
		return nil, errors.New("upload.file.required")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return nil, err
	}
	if fileHeader.Size > cfg.MaxFileSizeMB*1024*1024 {
		return nil, errors.New("upload.file.too_large")
	}

	extension := normalizeExtension(filepath.Ext(fileHeader.Filename))
	if extension == "" {
		return nil, errors.New("upload.file.type_not_allowed")
	}
	if len(cfg.AllowedTypes) > 0 && !containsString(cfg.AllowedTypes, extension) {
		return nil, errors.New("upload.file.type_not_allowed")
	}

	objectKey := filepath.ToSlash(filepath.Join(
		normalizeScope(scope),
		s.now().Format("20060102"),
		fmt.Sprintf("%s.%s", uuid.NewString(), extension),
	))
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = mime.TypeByExtension("." + extension)
	}

	switch cfg.StorageDriver {
	case "local":
		return s.storeLocal(cfg, fileHeader, objectKey, contentType, requestBaseURL)
	case "s3":
		return s.storeS3(ctx, cfg, fileHeader, objectKey, contentType)
	default:
		return nil, errors.New("upload.storage_driver.unsupported")
	}
}

func (s *Service) storeLocal(cfg *Config, fileHeader *multipart.FileHeader, objectKey, contentType, requestBaseURL string) (*StoredFile, error) {
	localPath, err := filepath.Abs(strings.TrimSpace(cfg.LocalPath))
	if err != nil {
		return nil, errors.New("upload.path.invalid")
	}

	absolutePath, err := secureJoin(localPath, objectKey)
	if err != nil {
		return nil, errors.New("upload.path.invalid")
	}
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		return nil, errors.New("upload.file.save.error")
	}

	source, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("upload.file.open.error")
	}
	defer source.Close()

	target, err := os.Create(absolutePath)
	if err != nil {
		return nil, errors.New("upload.file.save.error")
	}
	defer target.Close()

	if _, err := target.ReadFrom(source); err != nil {
		return nil, errors.New("upload.file.save.error")
	}

	return &StoredFile{
		FileName:     filepath.Base(objectKey),
		OriginalName: fileHeader.Filename,
		ObjectKey:    objectKey,
		URL:          BuildFileURL(cfg.PublicBaseURL, requestBaseURL, objectKey),
		Size:         fileHeader.Size,
		ContentType:  contentType,
	}, nil
}

func (s *Service) storeS3(ctx context.Context, cfg *Config, fileHeader *multipart.FileHeader, objectKey, contentType string) (*StoredFile, error) {
	if strings.TrimSpace(cfg.S3Endpoint) == "" {
		return nil, errors.New("upload.s3.endpoint.required")
	}
	if strings.TrimSpace(cfg.S3Bucket) == "" {
		return nil, errors.New("upload.s3.bucket.required")
	}
	if strings.TrimSpace(cfg.S3AccessKeyID) == "" || strings.TrimSpace(cfg.S3SecretKey) == "" {
		return nil, errors.New("upload.s3.credentials.required")
	}

	client, err := s.s3ClientFactory(cfg)
	if err != nil {
		return nil, err
	}

	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, errors.New("upload.s3.bucket.ensure.error")
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.S3Bucket, minio.MakeBucketOptions{Region: cfg.S3Region}); err != nil {
			return nil, errors.New("upload.s3.bucket.ensure.error")
		}
	}

	source, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("upload.file.open.error")
	}
	defer source.Close()

	if _, err := client.PutObject(ctx, cfg.S3Bucket, objectKey, source, fileHeader.Size, minio.PutObjectOptions{
		ContentType: contentType,
	}); err != nil {
		return nil, errors.New("upload.s3.upload.error")
	}

	return &StoredFile{
		FileName:     filepath.Base(objectKey),
		OriginalName: fileHeader.Filename,
		ObjectKey:    objectKey,
		URL:          buildS3FileURL(cfg, objectKey),
		Size:         fileHeader.Size,
		ContentType:  contentType,
	}, nil
}

func (s *Service) ResolveLocalPath(objectKey string) (string, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return "", err
	}
	if cfg.StorageDriver != "local" {
		return "", errors.New("upload.storage_driver.unsupported")
	}
	rootPath, err := filepath.Abs(strings.TrimSpace(cfg.LocalPath))
	if err != nil {
		return "", errors.New("upload.path.invalid")
	}
	normalizedKey, err := NormalizeObjectKey(objectKey)
	if err != nil {
		return "", err
	}
	return secureJoin(rootPath, normalizedKey)
}

func BuildFileURL(publicBaseURL, requestBaseURL, objectKey string) string {
	normalizedKey := strings.TrimLeft(filepath.ToSlash(objectKey), "/")
	base := strings.TrimSpace(publicBaseURL)
	if base == "" {
		return strings.TrimRight(requestBaseURL, "/") + defaultServePath + "/" + normalizedKey
	}
	base = strings.TrimRight(base, "/")
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		return base + "/" + normalizedKey
	}
	if requestBaseURL == "" {
		return "/" + strings.TrimLeft(base+"/"+normalizedKey, "/")
	}
	if strings.HasPrefix(base, "/") {
		return strings.TrimRight(requestBaseURL, "/") + base + "/" + normalizedKey
	}
	return strings.TrimRight(requestBaseURL, "/") + "/" + base + "/" + normalizedKey
}

func buildS3FileURL(cfg *Config, objectKey string) string {
	if strings.TrimSpace(cfg.PublicBaseURL) != "" {
		return BuildFileURL(cfg.PublicBaseURL, "", objectKey)
	}

	rawEndpoint := strings.TrimSpace(cfg.S3Endpoint)
	if rawEndpoint == "" {
		return "/" + strings.TrimLeft(filepath.ToSlash(filepath.Join(cfg.S3Bucket, objectKey)), "/")
	}
	if !strings.HasPrefix(rawEndpoint, "http://") && !strings.HasPrefix(rawEndpoint, "https://") {
		rawEndpoint = "https://" + rawEndpoint
	}
	return strings.TrimRight(rawEndpoint, "/") + "/" + strings.TrimLeft(filepath.ToSlash(filepath.Join(cfg.S3Bucket, objectKey)), "/")
}

func newS3Client(cfg *Config) (objectStorageClient, error) {
	endpoint, secure, err := normalizeS3Endpoint(strings.TrimSpace(cfg.S3Endpoint))
	if err != nil {
		return nil, err
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKeyID, cfg.S3SecretKey, ""),
		Secure: secure,
		Region: strings.TrimSpace(cfg.S3Region),
	})
	if err != nil {
		return nil, errors.New("upload.s3.endpoint.invalid")
	}
	return client, nil
}

func normalizeS3Endpoint(raw string) (string, bool, error) {
	if raw == "" {
		return "", false, errors.New("upload.s3.endpoint.required")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		return raw, true, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", false, errors.New("upload.s3.endpoint.invalid")
	}
	if parsed.Host == "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", false, errors.New("upload.s3.endpoint.invalid")
	}
	return parsed.Host, parsed.Scheme == "https", nil
}

func secureJoin(rootPath, relativePath string) (string, error) {
	cleanRelative, err := NormalizeObjectKey(relativePath)
	if err != nil {
		return "", err
	}
	targetPath := filepath.Join(rootPath, filepath.FromSlash(cleanRelative))
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return "", err
	}
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}
	prefix := absRoot + string(os.PathSeparator)
	if absTarget != absRoot && !strings.HasPrefix(absTarget, prefix) {
		return "", errors.New("upload.path.invalid")
	}
	return absTarget, nil
}

func NormalizeObjectKey(objectKey string) (string, error) {
	normalized := strings.TrimLeft(filepath.ToSlash(strings.TrimSpace(objectKey)), "/")
	if normalized == "" || !filepath.IsLocal(filepath.FromSlash(normalized)) {
		return "", errors.New("upload.path.invalid")
	}
	segments := strings.Split(normalized, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, "..") || strings.ContainsAny(segment, `<>:"|?*`) {
			return "", errors.New("upload.path.invalid")
		}
		for _, char := range segment {
			if char < 32 || char == 127 {
				return "", errors.New("upload.path.invalid")
			}
		}
	}
	return strings.Join(segments, "/"), nil
}

func normalizeAllowedTypes(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		normalized := normalizeExtension(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizeExtension(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
}

func normalizeScope(scope string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(scope), func(r rune) bool {
		return r == '/' || r == '\\'
	})
	if len(parts) == 0 {
		return "general"
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		builder := strings.Builder{}
		for _, r := range part {
			switch {
			case r >= 'a' && r <= 'z':
				builder.WriteRune(r)
			case r >= 'A' && r <= 'Z':
				builder.WriteRune(r + 32)
			case r >= '0' && r <= '9':
				builder.WriteRune(r)
			case r == '-' || r == '_':
				builder.WriteRune(r)
			}
		}
		if builder.Len() > 0 {
			result = append(result, builder.String())
		}
	}
	if len(result) == 0 {
		return "general"
	}
	return strings.Join(result, "/")
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
