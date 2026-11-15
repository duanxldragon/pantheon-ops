package observability

import (
	"context"
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Request cancellation must reach every datasource provider, not only Loki.
func TestDatasourceHTTPQueryCancellation(t *testing.T) {
	for _, kind := range []string{"prometheus", "victoriametrics", "tempo"} {
		t.Run(kind, func(t *testing.T) {
			db := testmysql.Open(t)
			if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
				t.Fatal(err)
			}
			entered := make(chan struct{})
			disconnected := make(chan struct{})
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// POST query bodies must be consumed so the server can observe the
				// client's cancellation while the fixture waits for a response.
				_, _ = io.Copy(io.Discard, r.Body)
				close(entered)
				select {
				case <-r.Context().Done():
					close(disconnected)
				case <-time.After(2 * time.Second):
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"status":"success","data":{}}`))
				}
			}))
			defer upstream.Close()
			row := PrometheusDatasource{TenantID: 11, Name: "cancel-" + kind, Type: kind, URL: upstream.URL, Timeout: 5}
			if err := db.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
			service := NewDatasourceService(db)
			service.ssrfValidator = testDatasourceValidator()
			router := newQueryTestRouter(t, db, service, QueryOwner{TenantID: 11, ActorID: 31}, &queryTestAuthority{mode: tenant.ModeMulti})
			path := observabilityAPIPrefix + "/datasources/1/query"
			body := `{"rawQuery":"up","time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}`
			if kind == "tempo" {
				path = observabilityAPIPrefix + "/datasources/1/traces"
				body = `{"recipe":{"version":1,"sourceKind":"tempo","rules":[{"field":"service","op":"exists"}],"time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table"}}`
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)).WithContext(ctx)
			done := make(chan struct{})
			go func() { router.ServeHTTP(httptest.NewRecorder(), req); close(done) }()
			select {
			case <-entered:
			case <-time.After(2 * time.Second):
				t.Fatal("provider not reached")
			}
			cancel()
			select {
			case <-disconnected:
			case <-time.After(700 * time.Millisecond):
				t.Error("request canceled but provider remained running")
			}
			select {
			case <-done:
			case <-time.After(3 * time.Second):
				t.Fatal("handler did not return")
			}
		})
	}
}

func TestDatasourceQueryContextPreCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &DatasourceService{}
	for _, query := range []func() error{
		func() error { _, err := s.QueryContext(ctx, 1, "up"); return err },
		func() error { _, err := s.SearchTracesContext(ctx, 1, "trace", "", "", 100); return err },
		func() error { _, err := s.GetTraceContext(ctx, 1, "abc123"); return err },
	} {
		if err := query(); !errors.Is(err, context.Canceled) {
			t.Fatalf("pre-canceled request touched provider: %v", err)
		}
	}
}

func TestTempoSearchNormalizesAbsoluteUTCToProviderSeconds(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start") != "1794186000" || r.URL.Query().Get("end") != "1794186900" {
			t.Errorf("Tempo requires Unix seconds, got %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"traces":[]}`))
	}))
	defer upstream.Close()
	row := PrometheusDatasource{Name: "tempo-time", Type: "tempo", URL: upstream.URL}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	service := NewDatasourceService(db)
	service.ssrfValidator = testDatasourceValidator()
	if _, err := service.SearchTracesContext(context.Background(), row.ID, "{}", "2026-11-09T01:00:00Z", "2026-11-09T01:15:00Z", 100); err != nil {
		t.Fatal(err)
	}
}
