package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupOperationLogTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemLogOper{}); err != nil {
		t.Fatalf("migrate operation log: %v", err)
	}
	return db
}

func waitOperationLog(t *testing.T, db *gorm.DB) SystemLogOper {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var log SystemLogOper
		err := db.Order("id desc").First(&log).Error
		if err == nil {
			return log
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("operation log not written in time")
	return SystemLogOper{}
}

func TestOperationLogStore_DropsWhenQueueFullAndDrainsOnClose(t *testing.T) {
	db := setupOperationLogTestDB(t)

	store := &operationLogAsyncStore{
		db:    db,
		queue: make(chan SystemLogOper, 1),
		done:  make(chan struct{}),
	}
	// 不启动 run()：队列容量 1，第二条必然走 drop 分支且不得阻塞。
	first := SystemLogOper{Title: "keep", OperTime: time.Now()}
	second := SystemLogOper{Title: "dropped", OperTime: time.Now()}

	enqueueDone := make(chan struct{})
	go func() {
		store.enqueue(first)
		store.enqueue(second)
		close(enqueueDone)
	}()
	select {
	case <-enqueueDone:
	case <-time.After(2 * time.Second):
		t.Fatal("enqueue blocked on full queue; sync fallback still present")
	}

	// Close 排空:启动 run 后关闭，队列中的 first 必须落库
	go store.run()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	store.Close(ctx)

	var count int64
	if err := db.Model(&SystemLogOper{}).Where("title = ?", "keep").Count(&count).Error; err != nil {
		t.Fatalf("count drained logs: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected queued log drained on close, got %d rows", count)
	}
	var droppedCount int64
	if err := db.Model(&SystemLogOper{}).Where("title = ?", "dropped").Count(&droppedCount).Error; err != nil {
		t.Fatalf("count dropped logs: %v", err)
	}
	if droppedCount != 0 {
		t.Fatalf("expected overflow log dropped, got %d rows", droppedCount)
	}

	// 关闭后 enqueue 是 no-op，不 panic
	store.enqueue(SystemLogOper{Title: "after-close"})
}

func TestOperationLogMiddleware_UsesAuditOverrides(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/import", func(c *gin.Context) {
		common.SetAuditMetadata(c, "导入测试", common.BusinessImport)
		common.SetAuditParam(c, `{"fileName":"demo.csv","fileSize":128}`)
		common.SetAuditResult(c, `{"applied":false,"failed":2}`)
		common.SetAuditStatus(c, 2)
		common.SetAuditErrorMsg(c, "import.result.has_errors")
		common.Success(c, gin.H{
			"applied": false,
			"failed":  2,
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/import", strings.NewReader("ignored"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=demo")
	req.Header.Set("X-Request-ID", "req-import-001")
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	log := waitOperationLog(t, db)
	if log.Method != http.MethodPost {
		t.Fatalf("expected method POST, got %s", log.Method)
	}
	if log.RequestID != "req-import-001" {
		t.Fatalf("expected request id req-import-001, got %s", log.RequestID)
	}
	if log.Status != 2 {
		t.Fatalf("expected overridden status=2, got %d", log.Status)
	}
	if log.SourceDomain != "other" {
		t.Fatalf("expected source domain other, got %s", log.SourceDomain)
	}
	if log.SourcePage != "other" {
		t.Fatalf("expected source page other, got %s", log.SourcePage)
	}
	if log.FailureCategory != "business" {
		t.Fatalf("expected failure category business, got %s", log.FailureCategory)
	}
	if log.ErrorMsg != "import.result.has_errors" {
		t.Fatalf("expected overridden error msg, got %s", log.ErrorMsg)
	}
	if log.OperParam != `{"fileName":"demo.csv","fileSize":128}` {
		t.Fatalf("unexpected audit param: %s", log.OperParam)
	}
	if log.JsonResult != `{"applied":false,"failed":2}` {
		t.Fatalf("unexpected audit result: %s", log.JsonResult)
	}
}

func TestOperationLogMiddleware_SanitizesResponseResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := setupOperationLogTestDB(t)

	engine := gin.New()
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	engine.POST("/login", func(c *gin.Context) {
		common.Success(c, gin.H{
			"accessToken":  "access-secret",
			"refreshToken": "refresh-secret",
			"tokenType":    "Bearer",
		})
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"username":"admin"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	log := waitOperationLog(t, db)
	if log.Method != http.MethodPost {
		t.Fatalf("expected method POST, got %s", log.Method)
	}
	if log.SourceDomain != "other" {
		t.Fatalf("expected source domain other, got %s", log.SourceDomain)
	}
	if log.SourcePage != "other" {
		t.Fatalf("expected source page other, got %s", log.SourcePage)
	}
	if log.FailureCategory != "" {
		t.Fatalf("expected empty failure category for success, got %s", log.FailureCategory)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(log.JsonResult), &payload); err != nil {
		t.Fatalf("unmarshal sanitized result: %v", err)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected response data object, got %#v", payload["data"])
	}
	if data["accessToken"] != "***" {
		t.Fatalf("expected masked accessToken, got %#v", data["accessToken"])
	}
	if data["refreshToken"] != "***" {
		t.Fatalf("expected masked refreshToken, got %#v", data["refreshToken"])
	}
	if data["tokenType"] != "***" {
		t.Fatalf("expected masked tokenType, got %#v", data["tokenType"])
	}
}

func TestOperationLogBufferCapsCapturedBody(t *testing.T) {
	buffer := newOperationLogBuffer()
	payload := strings.Repeat("a", defaultOperationLogBodyLimit+1024)

	written, err := buffer.Write([]byte(payload))
	if err != nil {
		t.Fatalf("write operation log buffer: %v", err)
	}
	if written != len(payload) {
		t.Fatalf("expected write to report full response length, got %d", written)
	}
	if buffer.Len() != defaultOperationLogBodyLimit {
		t.Fatalf("expected captured response to be capped at %d, got %d", defaultOperationLogBodyLimit, buffer.Len())
	}
}

func TestRequestContextMiddleware_PropagatesHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	engine.Use(RequestContextMiddleware())
	engine.GET("/ping", func(c *gin.Context) {
		common.Success(c, gin.H{
			"requestId": common.GetRequestID(c),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("X-Request-ID", "req-ping-001")
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get("X-Request-ID") != "req-ping-001" {
		t.Fatalf("expected response X-Request-ID header to be preserved")
	}
	if recorder.Header().Get("X-Trace-ID") != "req-ping-001" {
		t.Fatalf("expected response X-Trace-ID header to be preserved")
	}
}
