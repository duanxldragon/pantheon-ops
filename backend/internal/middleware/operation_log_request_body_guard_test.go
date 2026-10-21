package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// multipartProbe builds a multipart/form-data request carrying a secret form
// field and a binary file part, returning the request and the exact file bytes.
func multipartProbe(t *testing.T) (*http.Request, []byte, string) {
	t.Helper()

	fileBytes := []byte("binary-file-payload-marker\x00\x01\x02PK\x03\x04rest-of-file")
	secretValue := "s3cr3t-probe-value"

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("note", secretValue); err != nil {
		t.Fatalf("write multipart field: %v", err)
	}
	filePart, err := writer.CreateFormFile("file", "probe-upload.bin")
	if err != nil {
		t.Fatalf("create multipart file part: %v", err)
	}
	if _, err := filePart.Write(fileBytes); err != nil {
		t.Fatalf("write multipart file bytes: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/system/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, fileBytes, secretValue
}

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

// F05 acceptance 1: multipart uploads persist only allowlisted metadata —
// neither the secret field value nor raw file bytes may reach the audit row,
// while the handler still receives the untouched multipart body.
func TestOperationLogMultipartStoresOnlyAllowlistedMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/system/upload", func(c *gin.Context) {
		c.Set(operationLogTitleKey, "upload.file.title")
		c.Set(operationLogBusinessTypeKey, 1)
		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.String(http.StatusBadRequest, "form file required")
			return
		}
		file, err := fileHeader.Open()
		if err != nil {
			c.String(http.StatusBadRequest, "file unreadable")
			return
		}
		defer file.Close()
		got, err := io.ReadAll(file)
		if err != nil {
			c.String(http.StatusBadRequest, "file read failed")
			return
		}
		c.String(http.StatusOK, "ok")
		_ = got
	})

	req, fileBytes, secretValue := multipartProbe(t)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("handler lost multipart body, status %d body %q", recorder.Code, recorder.Body.String())
	}

	log := waitOperationLog(t, db)
	if !strings.Contains(log.OperParam, `"file"`) || !strings.Contains(log.OperParam, "probe-upload.bin") {
		t.Fatalf("audit row must carry allowlisted upload metadata, got %q", log.OperParam)
	}
	if strings.Contains(log.OperParam, secretValue) {
		t.Fatalf("audit row must not persist secret multipart field values, got %q", log.OperParam)
	}
	if strings.Contains(log.OperParam, "binary-file-payload-marker") {
		t.Fatalf("audit row must not persist raw file bytes, got %q", log.OperParam)
	}
	if bytes.Contains([]byte(log.OperParam), fileBytes) {
		t.Fatalf("audit row must not persist raw file bytes verbatim")
	}
	if strings.Contains(log.OperParam, "Content-Disposition") || strings.Contains(log.OperParam, "boundary") {
		t.Fatalf("audit row must not persist raw multipart envelope, got %q", log.OperParam)
	}
}

// Non-JSON request bodies must never be persisted, including a capped prefix.
func TestOperationLogAuditBodyCapTruncatesOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/blob", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			c.String(http.StatusBadRequest, "body unreadable")
			return
		}
		c.String(http.StatusOK, "ok")
	})

	payload := strings.Repeat("A", 256*1024)
	req := httptest.NewRequest(http.MethodPost, "/blob", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	log := waitOperationLog(t, db)
	if strings.Contains(log.OperParam, "AAAA") || !strings.Contains(log.OperParam, "__body") {
		t.Fatalf("audit param must contain metadata only, got %q", log.OperParam)
	}
}

func TestOperationLogFormBodyDoesNotPersistSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)
	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/form", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/form", strings.NewReader("password=form-secret-marker"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(httptest.NewRecorder(), req)
	log := waitOperationLog(t, db)
	if strings.Contains(log.OperParam, "form-secret-marker") || !strings.Contains(log.OperParam, "__body") {
		t.Fatalf("form values must not enter audit, got %q", log.OperParam)
	}
}

func TestOperationLogInvalidJSONDoesNotPersistRawBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)
	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/json", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/json", strings.NewReader(`{"password":"secret-marker"`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(httptest.NewRecorder(), req)
	log := waitOperationLog(t, db)
	if strings.Contains(log.OperParam, "secret-marker") || !strings.Contains(log.OperParam, "__body") {
		t.Fatalf("invalid JSON must not enter audit, got %q", log.OperParam)
	}
}

func TestOperationLogAuditParamOverrideIsCappedAfterMasking(t *testing.T) {
	t.Setenv(operationLogAuditBodyLimitEnv, "128")
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/override", func(c *gin.Context) {
		c.Set(operationLogParamKey, `{"password":"`+strings.Repeat("secret", 64)+`","note":"`+strings.Repeat("detail", 32)+`"}`)
		c.Status(http.StatusOK)
	})
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/override", nil))

	log := waitOperationLog(t, db)
	if log.OperParam != operationLogParamOverLimit {
		t.Fatalf("oversized explicit override must persist the over-limit marker, got %q", log.OperParam)
	}
	if len(log.OperParam) > operationLogAuditBodyLimit() || strings.Contains(log.OperParam, "secret") {
		t.Fatalf("override marker must fit the audit cap and contain no input values: %q", log.OperParam)
	}
}

func TestAuditParamFallbackRejectsUnsupportedAndMalformedBodies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		contentType string
		body        string
	}{
		{"application/octet-stream", "binary-secret-marker"},
		{"application/x-www-form-urlencoded", "password=form-secret-marker"},
		{"application/json", `{"password":"json-secret-marker"`},
	} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(tc.body))
		c.Request.Header.Set("Content-Type", tc.contentType)
		got := readOperationLogParam(c, buildAuditParamFallback(c, tc.body))
		if strings.Contains(got, "secret-marker") || !strings.Contains(got, "__body") {
			t.Errorf("%s persisted unsupported body: %q", tc.contentType, got)
		}
	}
}

func TestMultipartAuditDoesNotParseRejectedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request, _, secret := multipartProbe(t)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = request
	got := multipartAuditMetadata(c)
	if strings.Contains(got, secret) || !strings.Contains(got, "unparsed") {
		t.Fatalf("unparsed multipart request leaked values: %q", got)
	}
	if c.Request.MultipartForm != nil {
		t.Fatal("audit must not parse a rejected multipart body")
	}
	if _, err := c.Request.MultipartReader(); err != nil {
		t.Fatalf("audit must leave the multipart body available: %v", err)
	}
}

func TestOperationLogMultipartLeavesUnparsedBodyUntouched(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)
	req, _, secret := multipartProbe(t)
	originalBody := req.Body
	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/system/upload", func(c *gin.Context) {
		if c.Request.Body != originalBody {
			t.Error("audit must not read or replace multipart request body")
		}
		body, err := io.ReadAll(c.Request.Body)
		if err != nil || !bytes.Contains(body, []byte("binary-file-payload-marker")) {
			t.Errorf("handler did not receive original multipart body: %v", err)
		}
		c.Status(http.StatusBadRequest)
	})
	engine.ServeHTTP(httptest.NewRecorder(), req)
	log := waitOperationLog(t, db)
	if log.OperParam != `{"__multipart":"unparsed"}` || strings.Contains(log.OperParam, secret) {
		t.Fatalf("unparsed multipart metadata must not expose the body: %q", log.OperParam)
	}
}

func TestMultipartAuditBoundsFileMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/upload", nil)
	c.Request.MultipartForm = &multipart.Form{File: map[string][]*multipart.FileHeader{
		strings.Repeat("f", 20000): {{Filename: strings.Repeat("x", 20000)}},
	}}
	got := multipartAuditMetadata(c)
	if !strings.Contains(got, `"__truncated":true`) || len(got) > operationLogAuditBodyLimit() {
		t.Fatalf("oversized multipart names were not bounded: %q", got)
	}
	c.Request.MultipartForm.File["many"] = make([]*multipart.FileHeader, maxMultipartAuditFiles)
	for i := range c.Request.MultipartForm.File["many"] {
		c.Request.MultipartForm.File["many"][i] = &multipart.FileHeader{Filename: "other.bin"}
	}
	got = multipartAuditMetadata(c)
	if len(got) > operationLogAuditBodyLimit() {
		t.Fatalf("oversized multipart metadata was not bounded: %q", got)
	}
	var metadata struct {
		Truncated bool `json:"__truncated"`
		Files     map[string][]struct {
			Filename string `json:"filename"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(got), &metadata); err != nil {
		t.Fatalf("invalid audit JSON: %v", err)
	}
	if !metadata.Truncated {
		t.Fatalf("expected truncation marker: %q", got)
	}
	count := 0
	for field, parts := range metadata.Files {
		if len(field) > maxMultipartAuditFieldName {
			t.Fatalf("field name exceeded limit: %d", len(field))
		}
		count += len(parts)
		for _, part := range parts {
			if len(part.Filename) > maxMultipartAuditFileName {
				t.Fatalf("filename exceeded limit: %d", len(part.Filename))
			}
		}
	}
	if count > maxMultipartAuditFiles {
		t.Fatalf("file count exceeded limit: %d", count)
	}
}

func TestMultipartAuditBoundsFinalJSON(t *testing.T) {
	t.Setenv(operationLogAuditBodyLimitEnv, "512")
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/upload", nil)
	c.Request.MultipartForm = &multipart.Form{File: map[string][]*multipart.FileHeader{
		"file": {{Filename: strings.Repeat("\"", maxMultipartAuditFileName)}},
	}}
	got := multipartAuditMetadata(c)
	if len(got) > operationLogAuditBodyLimit() || got != multipartAuditOverLimit {
		t.Fatalf("final JSON must be bounded with a truncation marker: %q", got)
	}
}

// F05 acceptance 3 (correlation half): readAndRestoreBody must keep the full
// body available to the handler even when the audit copy is truncated.
func TestReadAndRestoreBodyKeepsFullBodyForHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	payload := strings.Repeat("B", defaultOperationLogBodyLimit*2)
	c.Request = httptest.NewRequest(http.MethodPost, "/probe", strings.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/octet-stream")

	auditCopy := readAndRestoreBody(c)
	if len(auditCopy) > defaultOperationLogBodyLimit {
		t.Fatalf("audit copy must be capped at %d bytes, got %d", defaultOperationLogBodyLimit, len(auditCopy))
	}

	restored, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if string(restored) != payload {
		t.Fatalf("handler body must stay intact: got %d bytes, want %d", len(restored), len(payload))
	}
}

func TestReadAndRestoreBodyForwardsClose(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	originalBody := &closeTrackingBody{Reader: strings.NewReader("request body")}
	c.Request = httptest.NewRequest(http.MethodPost, "/probe", nil)
	c.Request.Body = originalBody

	readAndRestoreBody(c)
	if err := c.Request.Body.Close(); err != nil {
		t.Fatalf("close wrapped body: %v", err)
	}
	if !originalBody.closed {
		t.Fatal("closing restored request body must close the original body")
	}
}

// F05 acceptance 2 (override): the audit cap is configurable for tighter
// deployments via PANTHEON_OPERATION_LOG_AUDIT_BODY_LIMIT.
func TestOperationLogAuditBodyLimitEnvOverride(t *testing.T) {
	t.Setenv("PANTHEON_OPERATION_LOG_AUDIT_BODY_LIMIT", "1024")
	if got := operationLogAuditBodyLimit(); got != 1024 {
		t.Fatalf("expected env override 1024, got %d", got)
	}
	t.Setenv(operationLogAuditBodyLimitEnv, strconv.Itoa(maxOperationLogAuditBodyLimit+1))
	if got := operationLogAuditBodyLimit(); got != maxOperationLogAuditBodyLimit {
		t.Fatalf("audit body limit must clamp to %d, got %d", maxOperationLogAuditBodyLimit, got)
	}

	t.Setenv("PANTHEON_OPERATION_LOG_AUDIT_BODY_LIMIT", "bogus")
	if got := operationLogAuditBodyLimit(); got != defaultOperationLogAuditBodyLimit {
		t.Fatalf("invalid env value must fall back to default, got %d", got)
	}
	t.Setenv("PANTHEON_OPERATION_LOG_AUDIT_BODY_LIMIT", "1")
	if got := operationLogAuditBodyLimit(); got != defaultOperationLogAuditBodyLimit {
		t.Fatalf("limit smaller than the audit marker must fall back to default, got %d", got)
	}
}
