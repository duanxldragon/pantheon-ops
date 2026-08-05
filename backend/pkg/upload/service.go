package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
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

const (
	schemeHTTP  = "http://"
	schemeHTTPS = "https://"
)

// ConfigReader reads upload settings by key.
type ConfigReader interface {
	GetByKey(settingKey string) (string, error)
}

// Config describes the active upload storage settings.
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

// StoredFile describes a file persisted through the upload service.
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

// Service loads upload configuration and stores files.
type Service struct {
	reader          ConfigReader
	now             func() time.Time
	s3ClientFactory func(cfg *Config) (objectStorageClient, error)
}

// NewService creates an upload service backed by the provided config reader.
func NewService(reader ConfigReader) *Service {
	return &Service{
		reader:          reader,
		now:             time.Now,
		s3ClientFactory: newS3Client,
	}
}

// LoadConfig resolves the current upload configuration from settings.
func (s *Service) LoadConfig() (*Config, error) {
	if s.reader == nil {
		return nil, errors.New("upload.config.unavailable")
	}

	cfg := &Config{
		StorageDriver: "local",
		MaxFileSizeMB: 20,
		AllowedTypes:  []string{"jpg", "jpeg", "png", "webp", "gif", "pdf", "doc", "docx", "xls", "xlsx", "zip", "gz", "tgz", "tar"},
		LocalPath:     "./uploads",
		S3Region:      "us-east-1",
	}

	for _, field := range loadConfigStringFields {
		raw, ok := readSetting(s.reader, field.key)
		if ok && (field.emptyAllowed || strings.TrimSpace(raw) != "") {
			field.apply(cfg, strings.TrimSpace(raw))
		}
	}

	if err := applyMaxFileSize(cfg, s.reader); err != nil {
		return nil, err
	}
	if err := applyAllowedTypes(cfg, s.reader); err != nil {
		return nil, err
	}
	return cfg, nil
}

// configStringField describes a scalar string setting and how it is applied to Config.
// emptyAllowed marks settings whose raw (trimmed) value is kept even when empty, mirroring the
// original per-key assignment semantics (settings read without the `&& trim != ""` guard).
type configStringField struct {
	key          string
	emptyAllowed bool
	apply        func(cfg *Config, value string)
}

// loadConfigStringFields enumerates the scalar string settings and maps them onto Config.
// The order is not significant for behaviour; only max_file_size/allowed_types can surface errors.
var loadConfigStringFields = []configStringField{
	{
		key: "upload.storage_driver",
		apply: func(cfg *Config, value string) {
			cfg.StorageDriver = strings.ToLower(value)
		},
	},
	{
		key: "upload.local_path",
		apply: func(cfg *Config, value string) {
			cfg.LocalPath = value
		},
	},
	{
		key:          "upload.public_base_url",
		emptyAllowed: true,
		apply: func(cfg *Config, value string) {
			cfg.PublicBaseURL = value
		},
	},
	{
		key:          "upload.s3_endpoint",
		emptyAllowed: true,
		apply: func(cfg *Config, value string) {
			cfg.S3Endpoint = value
		},
	},
	{
		key:          "upload.s3_bucket",
		emptyAllowed: true,
		apply: func(cfg *Config, value string) {
			cfg.S3Bucket = value
		},
	},
	{
		key: "upload.s3_region",
		apply: func(cfg *Config, value string) {
			cfg.S3Region = value
		},
	},
	{
		key:          "upload.s3_access_key_id",
		emptyAllowed: true,
		apply: func(cfg *Config, value string) {
			cfg.S3AccessKeyID = value
		},
	},
	{
		key:          "upload.s3_secret_access_key",
		emptyAllowed: true,
		apply: func(cfg *Config, value string) {
			cfg.S3SecretKey = value
		},
	},
}

// readSetting reads a raw setting value and reports whether the reader succeeded.
// The error guard (err == nil) is preserved by the ok result; callers apply their own trimming.
func readSetting(reader ConfigReader, key string) (string, bool) {
	value, err := reader.GetByKey(key)
	if err != nil {
		return "", false
	}
	return value, true
}

// applyMaxFileSize resolves upload.max_file_size, preserving the original error semantics:
// a parse failure or a non-positive value reports upload.config.invalid_max_file_size.
func applyMaxFileSize(cfg *Config, reader ConfigReader) error {
	raw, ok := readSetting(reader, "upload.max_file_size")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	size, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || size <= 0 {
		return errors.New("upload.config.invalid_max_file_size")
	}
	cfg.MaxFileSizeMB = size
	return nil
}

// applyAllowedTypes resolves upload.allowed_types, preserving the original error semantics:
// a JSON decode failure reports upload.config.invalid_allowed_types.
func applyAllowedTypes(cfg *Config, reader ConfigReader) error {
	raw, ok := readSetting(reader, "upload.allowed_types")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return errors.New("upload.config.invalid_allowed_types")
	}
	cfg.AllowedTypes = normalizeAllowedTypes(items)
	return nil
}

// MaxBytes returns the configured maximum upload size in bytes.
func (s *Service) MaxBytes() (int64, error) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return 0, err
	}
	return cfg.MaxFileSizeMB * 1024 * 1024, nil
}

// Store writes an uploaded file using the background context.
func (s *Service) Store(fileHeader *multipart.FileHeader, scope, requestBaseURL string) (*StoredFile, error) {
	return s.StoreWithContext(context.Background(), fileHeader, scope, requestBaseURL)
}

// StoreWithContext writes an uploaded file using the provided context.
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
	if err := verifyImageContent(fileHeader, extension); err != nil {
		return nil, err
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
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return nil, errors.New("upload.file.save.error")
	}

	source, err := fileHeader.Open()
	if err != nil {
		return nil, errors.New("upload.file.open.error")
	}
	defer func() {
		_ = source.Close()
	}()

	// #nosec G304 -- absolutePath has been normalized by secureJoin.
	target, err := os.Create(absolutePath)
	if err != nil {
		return nil, errors.New("upload.file.save.error")
	}
	defer func() {
		_ = target.Close()
	}()

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
	defer func() {
		_ = source.Close()
	}()

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

// ResolveLocalPath resolves a stored object key to a filesystem path.
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

// BuildFileURL builds the public URL for a stored file.
func BuildFileURL(publicBaseURL, requestBaseURL, objectKey string) string {
	normalizedKey := strings.TrimLeft(filepath.ToSlash(objectKey), "/")
	base := strings.TrimSpace(publicBaseURL)
	if base == "" {
		return strings.TrimRight(requestBaseURL, "/") + defaultServePath + "/" + normalizedKey
	}
	base = strings.TrimRight(base, "/")
	if strings.HasPrefix(base, schemeHTTP) || strings.HasPrefix(base, schemeHTTPS) {
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
	if !strings.HasPrefix(rawEndpoint, schemeHTTP) && !strings.HasPrefix(rawEndpoint, schemeHTTPS) {
		rawEndpoint = schemeHTTPS + rawEndpoint
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
	if !strings.HasPrefix(raw, schemeHTTP) && !strings.HasPrefix(raw, schemeHTTPS) {
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

// NormalizeObjectKey validates and normalizes an upload object key.
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

// sanitizeScopeSegment 对单个路径段执行白名单过滤：仅保留小写字母、大写字母（转小写）、
// 数字、连字符与下划线，其余 rune 直接丢弃。返回的字符串可能为空（当整段均被过滤时）。
// 该逻辑从 normalizeScope 提炼而来，语义须与原始内联实现逐字节等价。
func sanitizeScopeSegment(part string) string {
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
	return builder.String()
}

// normalizeScope 将任意 scope 字符串规范化为对象存储路径片段。
// 语义保持：按 '/' 与 '\' 切分 -> 逐段 trim -> 白名单过滤 -> 空结果回退 "general" -> 以 '/' 连接。
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
		if segment := sanitizeScopeSegment(part); segment != "" {
			result = append(result, segment)
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

// imageSniffPrefixes 图片扩展名与内容嗅探（http.DetectContentType）结果的对应关系。
var imageSniffPrefixes = map[string][]string{
	"jpg":  {"image/jpeg"},
	"jpeg": {"image/jpeg"},
	"png":  {"image/png"},
	"gif":  {"image/gif"},
	"webp": {"image/webp"},
}

// verifyImageContent 对图片类扩展做 magic-bytes 内容嗅探，拒绝“伪装成图片”的任意内容。
// 非图片扩展（pdf/zip 等）不做嗅探——这些类型内容多样，靠 serve 端 nosniff + attachment 兜底。
func verifyImageContent(fileHeader *multipart.FileHeader, extension string) error {
	expected, isImage := imageSniffPrefixes[extension]
	if !isImage {
		return nil
	}
	source, err := fileHeader.Open()
	if err != nil {
		return errors.New("upload.file.open.error")
	}
	defer func() {
		_ = source.Close()
	}()
	head := make([]byte, 512)
	n, err := io.ReadFull(source, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return errors.New("upload.file.open.error")
	}
	detected := http.DetectContentType(head[:n])
	for _, prefix := range expected {
		if strings.HasPrefix(detected, prefix) {
			return nil
		}
	}
	return errors.New("upload.file.type_not_allowed")
}
