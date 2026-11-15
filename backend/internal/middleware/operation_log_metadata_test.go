package middleware

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
)

type auditReadSentinel struct {
	reads  int
	reader io.Reader
}

func (r *auditReadSentinel) Read(p []byte) (int, error) { r.reads++; return r.reader.Read(p) }
func (*auditReadSentinel) Close() error                 { return nil }

type auditDenyLimit struct{}

func (auditDenyLimit) Allow(context.Context, string, int, time.Duration) (bool, error) {
	return false, nil
}

func TestMetadataAuditBeforeAllCaptureAndDenials(t *testing.T) {
	db := setupOperationLogTestDB(t)
	core, observed := observer.New(zap.DebugLevel)
	previous := logging.Logger
	logging.Logger = zap.New(core)
	defer func() { logging.Logger = previous }()
	for _, kind := range []string{"csrf", "rate", "body", "auth", "bind", "provider", "storage", "business200", "success", "raw200", "panic"} {
		t.Run(kind, func(t *testing.T) {
			runMetadataAuditCase(t, db, observed, kind)
		})
	}
}

func runMetadataAuditCase(t *testing.T, db *gorm.DB, observed *observer.ObservedLogs, kind string) {
	t.Helper()
	route := auditpolicy.Route{Method: "POST", Template: "/api/v1/business/audit-" + kind + "/items/:id"}
	if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{route}); err != nil {
		t.Fatal(err)
	}
	marker := "synthetic-privacy-sentinel-" + kind
	requestID := strings.Repeat("a", 64)
	engine, id, handlerCalls := metadataAuditEngine(t, db, route, kind, marker, requestID)
	body := &auditReadSentinel{reader: strings.NewReader(`{"value":"` + marker + `"}`)}
	request := httptest.NewRequest("POST", strings.Replace(route.Template, ":id", marker, 1)+"?query="+marker, body)
	request.ContentLength = 1024
	request.Header.Set(common.HeaderRequestID, requestID)
	request.Header.Set(common.HeaderTraceID, marker)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	ShutdownOperationLog(closeCtx)
	cancel()
	metadataAuditAssertOutcome(t, db, observed, metadataAuditCaseRun{
		kind: kind, marker: marker, requestID: requestID, route: route,
		response: response, id: *id, handlerCalls: *handlerCalls, body: body,
	})
}

type metadataAuditCaseRun struct {
	kind         string
	marker       string
	requestID    string
	route        auditpolicy.Route
	response     *httptest.ResponseRecorder
	id           string
	handlerCalls int
	body         *auditReadSentinel
}

func metadataAuditEngine(t *testing.T, db *gorm.DB, route auditpolicy.Route, kind, marker, requestID string) (*gin.Engine, *string, *int) {
	t.Helper()
	engine := gin.New()
	engine.Use(MetadataAwareLogger(), MetadataAwareRecovery())
	engine.Use(RequestContextMiddleware(), OperationLogMiddleware(db))
	id := ""
	engine.Use(func(c *gin.Context) {
		id = common.GetRequestID(c)
		if _, err := uuid.Parse(id); err != nil || id == requestID {
			t.Error("incoming correlation was retained")
		}
		if c.GetString(common.ContextKeyTraceID) != id || c.Writer.Header().Get(common.HeaderTraceID) != id {
			t.Error("correlation fields disagree")
		}
		logging.LogFromContext(c.Request.Context()).Info("audit context checkpoint")
		if _, buffered := c.Writer.(operationLogWriter); buffered {
			t.Error("metadata strategy installed response buffer")
		}
		c.Next()
	})
	metadataAuditUseDenialMiddleware(engine, kind)
	handlerCalls := 0
	engine.POST(route.Template, func(c *gin.Context) {
		handlerCalls++
		common.SetAuditParam(c, marker)
		common.SetAuditResult(c, marker)
		common.SetAuditErrorMsg(c, marker)
		_ = c.Error(errors.New(marker))
		metadataAuditHandle(t, kind, marker, c)
	})
	return engine, &id, &handlerCalls
}

func metadataAuditUseDenialMiddleware(engine *gin.Engine, kind string) {
	switch kind {
	case "csrf":
		engine.Use(CSRFMiddleware())
	case "rate":
		engine.Use(RateLimiter(RateLimiterConfig{Store: auditDenyLimit{}}))
	case "body":
		engine.Use(BodySizeLimit(8))
	case "auth":
		engine.Use(BusinessTokenAuthMiddleware(nil))
	}
}

func metadataAuditHandle(t *testing.T, kind, marker string, c *gin.Context) {
	switch kind {
	case "panic":
		panic(marker)
	case "bind":
		metadataAuditBind(t, c, marker)
	case "provider":
		metadataAuditOutcomeFailure(c, auditpolicy.OutcomeUpstream, marker)
	case "storage":
		metadataAuditOutcomeFailure(c, auditpolicy.OutcomeStorage, marker)
	case "business200":
		auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{Outcome: auditpolicy.OutcomeSuccess})
		common.Fail(c, 500, marker)
	case "success":
		auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{ActorID: 7, TenantID: 101, DatasourceID: 9, Count: 2})
		common.Success(c, strings.Repeat(marker, 5000))
	case "raw200":
		c.JSON(200, gin.H{"result": marker})
	default:
		t.Error("denied handler ran")
	}
}

func metadataAuditBind(t *testing.T, c *gin.Context, marker string) {
	t.Helper()
	var input struct {
		Value int `json:"value"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.Fail(c, 400, marker)
		return
	}
	t.Error("hostile bind succeeded")
}

func metadataAuditOutcomeFailure(c *gin.Context, outcome auditpolicy.Outcome, marker string) {
	auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{Outcome: outcome})
	common.FailWithError(c, 500, errors.New(marker), marker)
}

func metadataAuditAssertOutcome(t *testing.T, db *gorm.DB, observed *observer.ObservedLogs, run metadataAuditCaseRun) {
	kind, marker, requestID, route := run.kind, run.marker, run.requestID, run.route
	response, id, handlerCalls, body := run.response, run.id, run.handlerCalls, run.body
	t.Helper()
	var row SystemLogOper
	if err := db.Where("request_id = ?", id).Take(&row).Error; err != nil {
		t.Fatal("audit missing after denial/success")
	}
	serialized := row.OperParam + row.JsonResult + row.ErrorMsg + row.OperURL + row.Title + row.OperName + row.OperIP + row.RequestID
	if strings.Contains(serialized, marker) || strings.Contains(serialized, requestID) {
		t.Fatal("raw sentinel retained in audit")
	}
	if row.OperURL != route.Template || row.Method != route.Method || row.RequestID != response.Header().Get(common.HeaderRequestID) {
		t.Fatal("audit route/correlation changed")
	}
	if metadataAuditEarlyDenial(kind) && (body.reads != 0 || handlerCalls != 0 || row.TenantID != 0) {
		t.Fatal("denial captured body or forged ownership")
	}
	if (row.Status == common.OperationStatusSuccess) != (kind == "success") {
		t.Fatal("HTTP200 or missing typed outcome hid failure")
	}
	metadataAuditAssertLogsClean(t, observed, marker, requestID)
}

func metadataAuditEarlyDenial(kind string) bool {
	return kind == "csrf" || kind == "rate" || kind == "body" || kind == "auth"
}

func metadataAuditAssertLogsClean(t *testing.T, observed *observer.ObservedLogs, marker, requestID string) {
	t.Helper()
	for _, entry := range observed.All() {
		for _, value := range entry.ContextMap() {
			if text, ok := value.(string); ok && (strings.Contains(text, marker) || strings.Contains(text, requestID)) {
				t.Fatal("raw sentinel retained in logging context")
			}
		}
	}
}
