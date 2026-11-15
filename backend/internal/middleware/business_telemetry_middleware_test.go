package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	collectortrace "go.opentelemetry.io/proto/otlp/collector/trace/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const telemetrySentinel = "synthetic-telemetry-privacy-sentinel"

var telemetryCases = []string{
	"success", "raw200", "business200", "invalid-metadata", "provider", "storage", "error", "panic", "panic-error",
	"invalid-json", "csrf", "rate", "auth", "tenant", "body", "body-stream", "canceled",
}

// Take an SDK snapshot at Start, so a capture-then-sanitize regression cannot
// pass merely by removing attributes before the exporter sees the final span.
type telemetryStartRecorder struct {
	sync.Mutex
	spans []tracetest.SpanStub
}

func (r *telemetryStartRecorder) OnStart(_ context.Context, span sdktrace.ReadWriteSpan) {
	r.Lock()
	defer r.Unlock()
	r.spans = append(r.spans, tracetest.SpanStubFromReadOnlySpan(span))
}
func (*telemetryStartRecorder) OnEnd(sdktrace.ReadOnlySpan)      {}
func (*telemetryStartRecorder) Shutdown(context.Context) error   { return nil }
func (*telemetryStartRecorder) ForceFlush(context.Context) error { return nil }
func (r *telemetryStartRecorder) take() []tracetest.SpanStub {
	r.Lock()
	defer r.Unlock()
	spans := r.spans
	r.spans = nil
	return spans
}

// The registry is process-wide and startup-only. Exercise its actual Seal path
// in a subprocess, without exporting a test reset or sealing other suites.
func TestBusinessTelemetryPrivacy(t *testing.T) {
	if os.Getenv("PANTHEON_TEST_TELEMETRY_CHILD") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBusinessTelemetryPrivacy$", "-test.v")
		cmd.Env = append(os.Environ(), "PANTHEON_TEST_TELEMETRY_CHILD=1")
		out, err := cmd.CombinedOutput()
		t.Log(string(out))
		if err != nil {
			t.Fatalf("sealed telemetry suite: %v", err)
		}
		return
	}
	gin.SetMode(gin.TestMode)
	gin.DefaultWriter, gin.DefaultErrorWriter = io.Discard, io.Discard
	logging.Logger = zap.NewNop()
	propagators := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	otel.SetTextMapPropagator(propagators)
	exporter := tracetest.NewInMemoryExporter()
	started := &telemetryStartRecorder{}
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter), sdktrace.WithSpanProcessor(started), sdktrace.WithResource(resource.NewWithAttributes("",
		attribute.String("service.name", "pantheon-base"))))
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	// Register actual parameter routes, validate them against Gin and seal before
	// serving requests. Matching happens in global middleware before any denial.
	engines, callsByKind := telemetrySealRegistry(t)

	for _, kind := range telemetryCases {
		t.Run(kind, func(t *testing.T) {
			telemetryRunPrivacyCase(t, kind, engines[kind], callsByKind[kind], exporter, started)
		})
	}
	t.Run("unmatched-defaults", func(t *testing.T) { telemetryAssertLegacy(t, exporter) })
	t.Run("production-otlp", telemetryAssertOTLP)
}

func telemetrySealRegistry(t *testing.T) (map[string]*gin.Engine, map[string]*int) {
	t.Helper()
	engines := map[string]*gin.Engine{}
	callsByKind := map[string]*int{}
	var actualRoutes gin.RoutesInfo
	for _, kind := range telemetryCases {
		route := telemetryRoute(kind)
		if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{route}); err != nil {
			t.Fatal(err)
		}
		engine, calls := telemetrySetup(t, kind)
		engines[kind], callsByKind[kind] = engine, calls
		actualRoutes = append(actualRoutes, engine.Routes()...)
	}
	if err := auditpolicy.ValidateRoutes(actualRoutes); err != nil {
		t.Fatal(err)
	}
	if err := auditpolicy.Seal(); err != nil {
		t.Fatal(err)
	}
	if auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{telemetryRoute("late")}) == nil {
		t.Fatal("late strategy registration accepted")
	}
	return engines, callsByKind
}

var telemetryWantOutcomeByKind = map[string]string{
	"success": "success", "provider": "upstream", "storage": "storage",
	"business200": "denied", "csrf": "denied", "auth": "denied", "tenant": "denied",
	"invalid-json": "invalid", "body-stream": "invalid",
}

func telemetryWantOutcome(kind string) string {
	if outcome, ok := telemetryWantOutcomeByKind[kind]; ok {
		return outcome
	}
	return "unknown"
}

func telemetryRunPrivacyCase(t *testing.T, kind string, engine *gin.Engine, calls *int, exporter *tracetest.InMemoryExporter, started *telemetryStartRecorder) {
	t.Helper()
	exporter.Reset()
	started.take()
	body, response := telemetryExercise(t, kind, engine)
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected one completed server span, got %d", len(spans))
	}
	span := spans[0]
	requestID := response.Header().Get(common.HeaderRequestID)
	telemetryAssertSafeSpan(t, span, telemetryRoute(kind), requestID)
	initial := started.take()
	if len(initial) != 1 {
		t.Fatal("expected exactly one server span at SDK Start")
	}
	telemetryAssertSafeSpan(t, initial[0], telemetryRoute(kind), requestID)
	attrs := telemetryAttributes(span.Attributes)
	if attrs["http.response.status_code"] != int64(response.Code) {
		t.Error("span lost HTTP status")
	}
	wantOutcome := telemetryWantOutcome(kind)
	if attrs["business.outcome"] != wantOutcome {
		t.Errorf("outcome = %v, want %s", attrs["business.outcome"], wantOutcome)
	}
	wantStatus := codes.Error
	if kind == "success" {
		wantStatus = codes.Ok
	}
	if span.Status.Code != wantStatus || span.Status.Description != "" {
		t.Error("span status hid failure or retained error text")
	}
	telemetryAssertCaseMetadata(t, kind, *calls, body, attrs)
	if !t.Failed() {
		t.Logf("safe receipt: case=%s spans=1 raw-sentinel=0 outcome=%s status=%d", kind, wantOutcome, response.Code)
	}
}

func telemetryAssertCaseMetadata(t *testing.T, kind string, calls int, body *auditReadSentinel, attrs map[string]any) {
	t.Helper()
	early := kind == "csrf" || kind == "rate" || kind == "auth" || kind == "tenant" || kind == "body"
	if early && (calls != 0 || body.reads != 0 || attrs["business.actor_id"] != int64(0) || attrs["business.tenant_id"] != int64(0)) {
		t.Error("early denial consumed input or inferred ownership")
	}
	if kind == "success" && (attrs["business.actor_id"] != int64(7) || attrs["business.tenant_id"] != int64(101) || attrs["business.datasource_id"] != int64(9) || attrs["business.history_id"] != int64(11) || attrs["business.view_id"] != int64(12) || attrs["business.count"] != int64(2) || attrs["business.catalog_version"] != int64(1)) {
		t.Error("source-authorized metadata was lost")
	}
	if kind == "invalid-metadata" && attrs["business.actor_id"] != int64(0) {
		t.Error("invalid typed metadata retained")
	}
}

func telemetryRoute(kind string) auditpolicy.Route {
	return auditpolicy.Route{Method: http.MethodPost, Template: "/api/v1/business/telemetry/" + kind + "/items/:id"}
}

func TestBusinessTelemetryProductionWiring(t *testing.T) {
	raw, err := os.ReadFile("../../cmd/server/main.go")
	if err != nil {
		t.Fatal(err)
	}
	last := -1
	for _, call := range []string{
		"middleware.MetadataAwareRecovery()", "middleware.RequestContextMiddleware()", "middleware.OperationLogMiddleware(database.DB)",
		"middleware.BusinessTelemetryMiddleware()", "middleware.BodySizeLimit(", "middleware.CORSMiddleware()",
		"middleware.LegacyTelemetryMiddleware(\"pantheon-base\")", "middleware.PrometheusMiddleware()", "middleware.CSRFMiddleware()",
		"registerAPIRoutes(r)", "contracts.SealBusinessTenantProviders()", "auditpolicy.ValidateRoutes(r.Routes())", "auditpolicy.Seal()",
	} {
		position := strings.Index(string(raw), call)
		if position < 0 || position <= last {
			t.Fatalf("production capture/startup order violated at %s", call)
		}
		last = position
	}
	if strings.Contains(string(raw), "otelgin.Middleware(") {
		t.Fatal("unfiltered production capture reintroduced")
	}
}

// This is the production capture/denial order, using the actual middlewares.
// OperationLog(nil) avoids a credentialed DB; audit persistence has its own suite.
func telemetryPipeline() *gin.Engine {
	r := gin.New()
	r.Use(MetadataAwareLogger(), MetadataAwareRecovery())
	r.Use(SecurityHeadersMiddleware(), CSPNonceMiddleware(), CSPMiddleware())
	r.Use(RequestContextMiddleware(), OperationLogMiddleware(nil))
	r.Use(BusinessTelemetryMiddleware())
	r.Use(BodySizeLimit(DefaultMaxBodyBytes), CORSMiddleware())
	r.Use(LegacyTelemetryMiddleware("pantheon-base"))
	r.Use(PrometheusMiddleware(), CSRFMiddleware())
	return r
}

func telemetrySetup(t *testing.T, kind string) (*gin.Engine, *int) {
	t.Helper()
	r := telemetryPipeline()
	if kind == "rate" {
		r.Use(RateLimiter(RateLimiterConfig{Store: auditDenyLimit{}}))
	}
	if kind == "auth" {
		r.Use(BusinessTokenAuthMiddleware(nil))
	}
	if kind == "tenant" {
		r.Use(BusinessTenantContextMiddleware(nil, nil, nil, false))
	}
	calls := 0
	r.POST(telemetryRoute(kind).Template, func(c *gin.Context) {
		calls++
		telemetryAssertAdoptedContext(t, kind, c)
		telemetryRunHandler(t, kind, c)
	})
	return r, &calls
}

func telemetryAssertAdoptedContext(t *testing.T, kind string, c *gin.Context) {
	t.Helper()
	ctx := c.Request.Context()
	if _, ok := ctx.Deadline(); !ok || (kind == "canceled" && !errors.Is(ctx.Err(), context.Canceled)) {
		t.Error("tracing lost request deadline or cancellation")
	}
	if baggage.FromContext(ctx).Len() != 0 || trace.SpanContextFromContext(ctx).TraceState().Len() != 0 {
		t.Error("client baggage or tracestate reached adopted handler")
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if strings.Contains(carrier.Get("traceparent"), strings.Repeat("a", 32)) || carrier.Get("baggage") != "" || carrier.Get("tracestate") != "" || carrier.Get("traceparent") == "" {
		t.Error("server-only trace propagation lost or client metadata retained")
	}
}

func telemetryRunHandler(t *testing.T, kind string, c *gin.Context) {
	switch kind {
	case "success":
		telemetryHandleSuccess(t, c)
	case "raw200", "canceled":
		c.JSON(200, gin.H{"data": telemetrySentinel})
	case "business200":
		auditpolicy.SetOutcome(c, auditpolicy.OutcomeSuccess)
		common.Fail(c, 500, telemetrySentinel)
	case "invalid-metadata":
		auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{ActorID: 77, Outcome: auditpolicy.Outcome(255), Count: 10001})
		c.Status(200)
	case "provider":
		telemetryFailWithOutcome(c, auditpolicy.OutcomeUpstream)
	case "storage":
		telemetryFailWithOutcome(c, auditpolicy.OutcomeStorage)
	case "error":
		_ = c.Error(errors.New(telemetrySentinel))
		common.Success(c, telemetrySentinel)
	case "panic":
		panic(telemetrySentinel)
	case "panic-error":
		panic(errors.New(telemetrySentinel))
	case "invalid-json", "body-stream":
		telemetryHandleBind(t, c)
	default:
		t.Error("denied handler ran")
	}
}

func telemetryHandleSuccess(t *testing.T, c *gin.Context) {
	t.Helper()
	data, err := io.ReadAll(c.Request.Body)
	if err != nil || !strings.Contains(string(data), telemetrySentinel) {
		t.Error("telemetry changed the handler's request body")
	}
	auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{ActorID: 7, TenantID: 101, DatasourceID: 9, HistoryID: 11, ViewID: 12, Count: 2, CatalogVersion: 1})
	common.Success(c, telemetrySentinel)
}

func telemetryFailWithOutcome(c *gin.Context, outcome auditpolicy.Outcome) {
	auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{Outcome: outcome})
	_ = c.Error(errors.New(telemetrySentinel))
	common.FailWithError(c, 500, errors.New(telemetrySentinel), telemetrySentinel)
}

func telemetryHandleBind(t *testing.T, c *gin.Context) {
	t.Helper()
	var input struct {
		Number int `json:"number"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		_ = c.Error(err)
		auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{Outcome: auditpolicy.OutcomeInvalid})
		common.Fail(c, 400, telemetrySentinel)
		c.Abort()
		return
	}
	t.Error("hostile JSON accepted")
}

func telemetryExercise(t *testing.T, kind string, r *gin.Engine) (*auditReadSentinel, *httptest.ResponseRecorder) {
	t.Helper()
	body := &auditReadSentinel{reader: strings.NewReader(`{"number":"` + telemetrySentinel + `"}`)}
	req := telemetryRequest(http.MethodPost, strings.Replace(telemetryRoute(kind).Template, ":id", telemetrySentinel, 1), body)
	if kind != "csrf" {
		req.AddCookie(&http.Cookie{Name: commonhttp.CookieCSRFToken, Value: telemetrySentinel})
		req.Header.Set("X-CSRF-Token", telemetrySentinel)
	}
	if kind == "body" {
		req.ContentLength = DefaultMaxBodyBytes + 1
	}
	if kind == "body-stream" {
		req.ContentLength = -1
		body.reader = strings.NewReader(strings.Repeat(" ", int(DefaultMaxBodyBytes)+1) + telemetrySentinel)
	}
	if kind == "canceled" {
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
	}
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	return body, response
}

func telemetryRequest(method, path string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, "http://"+telemetrySentinel+".invalid"+path+"?query="+telemetrySentinel, body)
	req.Header.Set("Content-Type", "application/json")
	for _, key := range []string{"User-Agent", "X-Forwarded-For", common.HeaderRequestID, common.HeaderTraceID, "X-Tenant-ID", "Authorization", "X-Custom-Secret"} {
		req.Header.Set(key, telemetrySentinel)
	}
	req.Header.Set("X-Audit-Strategy", "legacy")
	req.Header.Set("traceparent", "00-"+strings.Repeat("a", 32)+"-"+strings.Repeat("b", 16)+"-01")
	req.Header.Set("tracestate", "synthetic="+telemetrySentinel)
	req.Header.Set("baggage", "synthetic="+telemetrySentinel)
	member, _ := baggage.NewMember("inherited", telemetrySentinel)
	inherited, _ := baggage.New(member)
	state, _ := trace.ParseTraceState("inherited=" + telemetrySentinel)
	traceID, _ := trace.TraceIDFromHex(strings.Repeat("a", 32))
	spanID, _ := trace.SpanIDFromHex(strings.Repeat("b", 16))
	ctx := trace.ContextWithRemoteSpanContext(req.Context(), trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, TraceState: state, Remote: true,
	}))
	return req.WithContext(baggage.ContextWithBaggage(ctx, inherited))
}

func telemetryAttributes(attrs []attribute.KeyValue) map[string]any {
	out := make(map[string]any, len(attrs))
	for _, attr := range attrs {
		out[string(attr.Key)] = attr.Value.AsInterface()
	}
	return out
}

func telemetryAssertSafeSpan(t *testing.T, span tracetest.SpanStub, route auditpolicy.Route, requestID string) {
	t.Helper()
	encoded, err := json.Marshal(struct {
		Span     tracetest.SpanStub
		Resource []attribute.KeyValue
	}{span, span.Resource.Attributes()})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), telemetrySentinel) {
		t.Error("raw sentinel in exported span/event/resource/status")
	}
	if span.Name != auditpolicy.ActionName(route) || span.SpanKind != trace.SpanKindServer || span.Parent.IsValid() || len(span.Links) != 0 || span.SpanContext.TraceState().Len() != 0 {
		t.Error("registered action/server-only trace boundary lost")
	}
	attrs := telemetryAttributes(span.Attributes)
	if attrs["http.request.method"] != route.Method || attrs["http.route"] != route.Template || attrs["business.action"] != auditpolicy.ActionName(route) || attrs["request.id"] != requestID {
		t.Error("route/method/action/server correlation missing")
	}
	if _, err := uuid.Parse(requestID); err != nil {
		t.Error("request ID was not generated by server")
	}
	allowed := map[string]bool{"http.request.method": true, "http.route": true, "http.response.status_code": true,
		"business.action": true, "business.outcome": true, "request.id": true,
		"business.actor_id": true, "business.tenant_id": true, "business.datasource_id": true,
		"business.history_id": true, "business.view_id": true, "business.count": true, "business.catalog_version": true}
	for key := range attrs {
		if !allowed[key] {
			t.Errorf("unexpected captured attribute: %s", key)
		}
	}
	for _, event := range span.Events {
		if (event.Name != "business.request.panic" && event.Name != "business.request.error") || len(event.Attributes) != 0 {
			t.Error("unbounded error event")
		}
	}
}

func telemetryAssertLegacy(t *testing.T, exporter *tracetest.InMemoryExporter) {
	for _, target := range []struct{ method, path string }{
		{http.MethodPost, "/legacy/" + telemetrySentinel},
		{http.MethodGet, strings.Replace(telemetryRoute("success").Template, ":id", telemetrySentinel, 1)},
		{http.MethodGet, "/unknown/" + telemetrySentinel},
	} {
		exporter.Reset()
		r := telemetryPipeline()
		r.POST("/legacy/:id", func(c *gin.Context) { _ = c.Error(errors.New(telemetrySentinel)); c.Status(500) })
		r.GET(telemetryRoute("success").Template, func(c *gin.Context) { c.Status(200) })
		req := telemetryRequest(target.method, target.path, nil)
		req.Header.Set("X-Audit-Strategy", "metadata-only")
		req.AddCookie(&http.Cookie{Name: commonhttp.CookieCSRFToken, Value: telemetrySentinel})
		req.Header.Set("X-CSRF-Token", telemetrySentinel)
		r.ServeHTTP(httptest.NewRecorder(), req)
		spans := exporter.GetSpans()
		if len(spans) != 1 {
			t.Fatal("unmatched telemetry disabled")
		}
		encoded, _ := json.Marshal(spans[0])
		if !strings.Contains(string(encoded), telemetrySentinel) || !spans[0].Parent.IsRemote() {
			t.Error("legacy capture or propagation defaults changed")
		}
	}
	// Default body denial was before otelgin and must remain untraced on legacy.
	exporter.Reset()
	r := telemetryPipeline()
	r.POST("/legacy/:id", func(*gin.Context) { t.Error("oversized legacy handler ran") })
	req := telemetryRequest(http.MethodPost, "/legacy/"+telemetrySentinel, nil)
	req.ContentLength = DefaultMaxBodyBytes + 1
	r.ServeHTTP(httptest.NewRecorder(), req)
	if len(exporter.GetSpans()) != 0 {
		t.Error("legacy body-denial telemetry order changed")
	}
}

func telemetryOTLPWire(t *testing.T) (chan []byte, string, func()) {
	t.Helper()
	wire := make(chan []byte, len(telemetryCases))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		wire <- data
		w.Header().Set("Content-Type", "application/x-protobuf")
		w.WriteHeader(200)
	}))
	return wire, server.URL, server.Close
}

func telemetryCheckOTLPBatch(t *testing.T, data []byte) (spans int, sentinelPresent bool) {
	t.Helper()
	var exported collectortrace.ExportTraceServiceRequest
	if err := proto.Unmarshal(data, &exported); err != nil {
		t.Fatal(err)
	}
	encoded, err := protojson.Marshal(&exported)
	if err != nil {
		t.Fatal(err)
	}
	sentinelPresent = strings.Contains(string(encoded), telemetrySentinel)
	if sentinelPresent || strings.Contains(string(data), telemetrySentinel) {
		t.Error("raw sentinel in actual production OTLP wire output")
	}
	for _, res := range exported.ResourceSpans {
		for _, scope := range res.ScopeSpans {
			spans += len(scope.Spans)
			for _, span := range scope.Spans {
				if len(span.ParentSpanId) != 0 || span.TraceState != "" || len(span.Links) != 0 || span.Status.Message != "" {
					t.Error("client correlation or error text in OTLP")
				}
			}
		}
	}
	return spans, sentinelPresent
}

func telemetryAssertOTLP(t *testing.T) {
	wire, endpoint, closeWire := telemetryOTLPWire(t)
	defer closeWire()
	previous := otel.GetTracerProvider()
	tp, err := telemetry.InitTracer("pantheon-base", strings.TrimPrefix(endpoint, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer otel.SetTracerProvider(previous)
	defer func() { _ = tp.Shutdown(context.Background()) }()
	for _, kind := range telemetryCases {
		r, _ := telemetrySetup(t, kind)
		telemetryExercise(t, kind, r)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := tp.ForceFlush(ctx); err != nil {
		t.Fatal(err)
	}
	count, bytes := 0, 0
	for count < len(telemetryCases) {
		select {
		case data := <-wire:
			spans, sentinelPresent := telemetryCheckOTLPBatch(t, data)
			count += spans
			bytes += len(data)
			t.Logf("OTLP batch: spans=%d bytes=%d sha256=%x sentinel-present=%t", count, len(data), sha256.Sum256(data), sentinelPresent)
		case <-ctx.Done():
			t.Fatalf("OTLP missing spans: got %d, want %d", count, len(telemetryCases))
		}
	}
	if !t.Failed() {
		t.Logf("production OTLP receipt: spans=%d bytes=%d raw-sentinel=0", count, bytes)
	}
}
