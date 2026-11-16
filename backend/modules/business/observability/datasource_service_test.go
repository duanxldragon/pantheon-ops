package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

// testDatasourceValidator allows loopback test endpoints without widening the
// production SSRF port/private-IP policy enforced by NewDatasourceService.
func testDatasourceValidator() *ssrf.Validator {
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = nil
	validator.BlockPrivateIPs = false
	validator.DNSRebindProtection = false
	return validator
}

func TestDatasourceDefaultCanMoveAndClear(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &AlertRule{}); err != nil {
		t.Fatalf("migrate observability tables: %v", err)
	}
	first := PrometheusDatasource{Name: "first", Type: "prometheus", URL: "http://127.0.0.1:1", AccessMode: "proxy", Timeout: 1, IsDefault: true}
	second := PrometheusDatasource{Name: "second", Type: "prometheus", URL: "http://127.0.0.1:1", AccessMode: "proxy", Timeout: 1}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first datasource: %v", err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("create second datasource: %v", err)
	}

	service := NewDatasourceService(db)
	service.ssrfValidator = testDatasourceValidator()
	if err := service.Update(second.ID, DatasourceDTO{Name: second.Name, Type: second.Type, URL: second.URL, AccessMode: "proxy", Timeout: 1, IsDefault: true}); err != nil {
		t.Fatalf("move default datasource: %v", err)
	}
	if err := db.First(&first, first.ID).Error; err != nil {
		t.Fatalf("reload first datasource: %v", err)
	}
	if first.IsDefault {
		t.Fatal("expected previous default datasource to be cleared")
	}

	if err := service.Update(second.ID, DatasourceDTO{Name: second.Name, Type: second.Type, URL: second.URL, AccessMode: "proxy", Timeout: 1, IsDefault: false}); err != nil {
		t.Fatalf("clear default datasource: %v", err)
	}
	if err := db.First(&second, second.ID).Error; err != nil {
		t.Fatalf("reload second datasource: %v", err)
	}
	if second.IsDefault {
		t.Fatal("expected default datasource to be reversible")
	}
}

func TestDatasourceAccessHonorsDepartmentScope(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatalf("migrate datasource table: %v", err)
	}
	owned := PrometheusDatasource{Name: "owned", Type: "loki", URL: "http://127.0.0.1:1", DeptID: 7}
	foreign := PrometheusDatasource{Name: "foreign", Type: "tempo", URL: "http://127.0.0.1:1", DeptID: 8}
	if err := db.Create(&owned).Error; err != nil {
		t.Fatalf("create owned datasource: %v", err)
	}
	if err := db.Create(&foreign).Error; err != nil {
		t.Fatalf("create foreign datasource: %v", err)
	}

	service := NewDatasourceService(db)
	scope := &common.DataScopeReq{Mode: common.DataScopeModeDept, DeptID: 7}
	list, err := service.List(scope)
	if err != nil {
		t.Fatalf("list scoped datasources: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != owned.ID {
		t.Fatalf("unexpected scoped datasource list: %+v", list.Items)
	}
	if _, err := service.GetTrace(foreign.ID, "abc123", scope); err == nil {
		t.Fatal("expected cross-department trace datasource access to fail")
	}
}

func TestDatasourceDeleteProtectsAlertRuleReference(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &AlertRule{}); err != nil {
		t.Fatalf("migrate observability tables: %v", err)
	}
	datasource := PrometheusDatasource{Name: "metrics", Type: "prometheus", URL: "http://127.0.0.1:1", AccessMode: "proxy"}
	if err := db.Create(&datasource).Error; err != nil {
		t.Fatalf("create datasource: %v", err)
	}
	if err := db.Create(&AlertRule{Name: "cpu-high", DatasourceID: datasource.ID, Query: "up", Interval: 60, Severity: "warning"}).Error; err != nil {
		t.Fatalf("create alert rule: %v", err)
	}

	err := NewDatasourceService(db).Delete(datasource.ID)
	if !errors.Is(err, errors.New("observability.datasource_in_use")) && (err == nil || err.Error() != "observability.datasource_in_use") {
		t.Fatalf("expected datasource_in_use, got %v", err)
	}
}

func TestDatasourceQueriesLokiAndTempo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/loki/api/v1/query_range":
			if request.URL.Query().Get("query") != `{app="api"}` {
				t.Fatalf("unexpected LogQL query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
		case "/api/search":
			if request.URL.Query().Get("q") != `{ resource.service.name = "api" }` {
				t.Fatalf("unexpected TraceQL query: %s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{"traces":[]}`))
		case "/api/traces/abc123":
			_, _ = response.Write([]byte(`{"batches":[]}`))
		default:
			http.NotFound(response, request)
		}
	}))
	defer server.Close()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatalf("migrate datasource table: %v", err)
	}
	loki := PrometheusDatasource{Name: "logs", Type: "loki", URL: server.URL, AccessMode: "proxy", Timeout: 2}
	tempo := PrometheusDatasource{Name: "traces", Type: "tempo", URL: server.URL, AccessMode: "proxy", Timeout: 2}
	if err := db.Create(&loki).Error; err != nil {
		t.Fatalf("create Loki datasource: %v", err)
	}
	if err := db.Create(&tempo).Error; err != nil {
		t.Fatalf("create Tempo datasource: %v", err)
	}

	service := NewDatasourceService(db)
	service.ssrfValidator = testDatasourceValidator()
	if _, err := service.QueryLogs(loki.ID, `{app="api"}`, "", "", 100); err != nil {
		t.Fatalf("query Loki: %v", err)
	}
	if _, err := service.SearchTraces(tempo.ID, `{ resource.service.name = "api" }`, "", "", 20); err != nil {
		t.Fatalf("search Tempo: %v", err)
	}
	if _, err := service.GetTrace(tempo.ID, "abc123"); err != nil {
		t.Fatalf("get Tempo trace: %v", err)
	}
}

func TestLogQueryRangeBudget(t *testing.T) {
	for _, tc := range []struct {
		start, end string
		valid      bool
	}{
		{"", "", true},
		{"2026-11-07T00:00:00Z", "2026-11-07T01:00:00Z", true},
		{"1794009600000000000", "1794013200000000000", true},
		{"1794009600", "1794013200", true},
		{"", "2026-11-07T01:00:00Z", false},
		{"2026-11-07T01:00:00Z", "2026-11-07T00:00:00Z", false},
		{"2026-11-07T00:00:00Z", "2026-11-09T00:00:00Z", false},
		{"bad", "also bad", false},
	} {
		if got := validLogQueryRange(tc.start, tc.end); got != tc.valid {
			t.Errorf("range %q/%q: want %v got %v", tc.start, tc.end, tc.valid, got)
		}
	}
}

func TestLogQueryCancellationAndResponseBudget(t *testing.T) {
	started, canceled := make(chan struct{}, 1), make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == "oversize" {
			_, _ = w.Write([]byte(`{"status":"success","padding":"` + strings.Repeat("x", 8<<20) + `"}`))
			return
		}
		started <- struct{}{}
		<-r.Context().Done()
		canceled <- struct{}{}
	}))
	defer server.Close()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	record := PrometheusDatasource{Name: "cancellation", Type: "loki", URL: server.URL, Timeout: 2}
	if err := db.Create(&record).Error; err != nil {
		t.Fatal(err)
	}
	service := NewDatasourceService(db)
	service.ssrfValidator = testDatasourceValidator()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := service.QueryLogsContext(ctx, record.ID, "slow", "", "", 500); result <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("upstream not started")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("caller cancellation not propagated")
	}
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("upstream cancellation not propagated")
	}
	if _, err := service.QueryLogsContext(context.Background(), record.ID, "oversize", "", "", 500); err == nil || err.Error() != "observability.datasource_response_too_large" {
		t.Fatalf("want bounded response rejection, got %v", err)
	}
	if _, err := service.QueryLogsContext(context.Background(), record.ID, strings.Repeat("x", 4097), "", "", 500); !errors.Is(err, errLogQueryInvalid) {
		t.Fatalf("want invalid query budget, got %v", err)
	}
}
