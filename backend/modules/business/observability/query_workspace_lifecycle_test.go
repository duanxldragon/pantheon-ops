package observability

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

func TestTenantDatasourceCRUDDoesNotDetachProbe(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	detached := make(chan struct{}, 2)
	if err := db.Callback().Query().Before("gorm:query").Register("test:reject-detached-probe", func(tx *gorm.DB) {
		if tx.Statement.Table == "obs_prometheus_datasource" {
			if _, bounded := tx.Statement.Context.Deadline(); !bounded {
				detached <- struct{}{}
				tx.AddError(errors.New("detached probe rejected by fixture"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	service := NewDatasourceService(db.WithContext(ctx))
	service.ssrfValidator = testDatasourceValidator()
	service, err := service.WithTenantContext(&tenant.Context{TenantID: 11, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatal(err)
	}
	dto := DatasourceDTO{Name: "not-tested", Type: "prometheus", URL: "http://127.0.0.1:1", Timeout: 1}
	created, err := service.Create(dto, "31", &common.DataScopeReq{IsAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Update(created.ID, dto, &common.DataScopeReq{IsAdmin: true}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-detached:
		t.Fatal("adopted CRUD launched an unauthenticated detached probe")
	case <-time.After(250 * time.Millisecond):
	}
}

func TestWorkspaceResponseRechecksDatasourceAction(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "safe-source", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	authority := &queryTestAuthority{mode: tenant.ModeMulti, denySourcesAtOwnCheck: 3}
	router := newQueryTestRouter(t, db, NewDatasourceService(db), QueryOwner{TenantID: 11, ActorID: 31}, authority)
	workspace, _ := NewQueryWorkspaceService(db, QueryOwner{TenantID: 11, ActorID: 31}, &common.DataScopeReq{IsAdmin: true})
	view, err := workspace.Save(context.Background(), source.ID, 0, 0, safeMetricRecipe())
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, observabilityAPIPrefix+"/saved-views/"+strconv.FormatUint(view.ID, 10)+"/restore", nil))
	if recorder.Code != http.StatusForbidden || strings.Contains(recorder.Body.String(), "availability") {
		t.Fatal("workspace returned a source recipe after source-only action revocation")
	}
}

func TestTenantDatasourceExplicitProbeAndCanceledStatus(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(strconv.FormatBool(canceled), func(t *testing.T) {
			db := testmysql.Open(t)
			if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryOwnerBudget{}); err != nil {
				t.Fatal(err)
			}
			var calls atomic.Int64
			entered, stopped := make(chan struct{}), make(chan struct{})
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if canceled {
					close(entered)
					<-r.Context().Done()
					close(stopped)
					return
				}
				_, _ = w.Write([]byte(`{"status":"success"}`))
			}))
			defer provider.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			service := NewDatasourceService(db.WithContext(ctx))
			service.ssrfValidator = testDatasourceValidator()
			service, _ = service.WithTenantContext(&tenant.Context{TenantID: 11, Mode: tenant.ModeMulti})
			scope := &common.DataScopeReq{IsAdmin: true}
			dto := DatasourceDTO{Name: "explicit-test", Type: "prometheus", URL: provider.URL}
			row, err := service.Create(dto, "31", scope)
			if err != nil {
				t.Fatal(err)
			}
			if err := service.Update(row.ID, dto, scope); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 0 {
				t.Fatal("CRUD consumed provider credentials")
			}
			var checkpoints atomic.Int64
			service.checkpoint = func(*PrometheusDatasource) error { checkpoints.Add(1); return nil }
			result := make(chan error, 1)
			go func() {
				healthy, err := service.TestConnection(row.ID, scope)
				if !canceled && !healthy && err == nil {
					err = errors.New("probe false")
				}
				result <- err
			}()
			if canceled {
				select {
				case <-entered:
				case <-time.After(time.Second):
					t.Fatal("explicit provider not entered")
				}
				cancel()
				select {
				case <-stopped:
				case <-time.After(time.Second):
					t.Fatal("provider not canceled")
				}
			}
			select {
			case err := <-result:
				if canceled && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
				if !canceled && err != nil {
					t.Fatal(err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("probe outlived bounded request")
			}
			var current PrometheusDatasource
			if err := db.First(&current, row.ID).Error; err != nil {
				t.Fatal(err)
			}
			if canceled && (current.LastChecked != nil || current.Status != "unknown") {
				t.Fatal("canceled probe changed status")
			}
			if !canceled && (checkpoints.Load() < 3 || current.Status != "healthy") {
				t.Fatal("probe omitted a source checkpoint")
			}
		})
	}
}

func TestExplicitProbeResponseRechecksRevokedActionAfterStatusWrite(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer provider.Close()
	source := PrometheusDatasource{TenantID: 11, Name: "release-check", Type: "prometheus", URL: provider.URL}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	service := NewDatasourceService(db)
	service.ssrfValidator = testDatasourceValidator()
	authority := &queryTestAuthority{mode: tenant.ModeMulti}
	router := newQueryTestRouter(t, db, service, QueryOwner{TenantID: 11, ActorID: 31}, authority)
	if err := db.Callback().Update().After("gorm:update").Register("test:late-probe-revocation", func(tx *gorm.DB) {
		if tx.Statement.Table == "obs_prometheus_datasource" {
			authority.revoke()
		}
	}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, observabilityAPIPrefix+"/datasources/"+strconv.FormatUint(source.ID, 10)+"/test", nil))
	if recorder.Code == http.StatusOK || strings.Contains(recorder.Body.String(), "healthy") {
		t.Fatal("probe returned outcome after late revocation")
	}
}

func TestCoalescedFollowHistoryOwnsOneSummary(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}); err != nil {
		t.Fatal(err)
	}
	source := PrometheusDatasource{TenantID: 11, Name: "coalesced", Type: "prometheus", URL: "http://127.0.0.1:1"}
	if err := db.Create(&source).Error; err != nil {
		t.Fatal(err)
	}
	scope := &common.DataScopeReq{IsAdmin: true}
	owner := QueryOwner{TenantID: 11, ActorID: 31}
	service, _ := NewQueryWorkspaceService(db, owner, scope)
	recipe := safeMetricRecipe()
	input := HistoryInput{HistoryMode: "coalesced", DatasourceID: source.ID, SourceRevision: source.Revision, SourceKind: source.Type, Time: recipe.Time, Limit: recipe.Limit, ResultMode: recipe.ResultMode, Outcome: "success", Recipe: &recipe}
	first, err := service.AppendHistory(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	input.HistoryID = first.ID
	for i := 0; i < 10; i++ {
		if _, err := service.AppendHistory(context.Background(), input); err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	db.Model(&QueryHistory{}).Count(&count)
	if count != 1 {
		t.Fatalf("follow persisted%d rows", count)
	}
	for _, foreign := range []QueryOwner{{TenantID: 11, ActorID: 32}, {TenantID: 22, ActorID: 31}} {
		other, _ := NewQueryWorkspaceService(db, foreign, scope)
		if _, err := other.AppendHistory(context.Background(), input); err == nil {
			t.Fatal("foreign summary rewritten")
		}
	}
	input.Limit++
	if _, err := service.AppendHistory(context.Background(), input); err == nil {
		t.Fatal("changed intent joined old session")
	}
}

func TestCoalescedHistoryEnvelopeAcceptsOnlyOwnedMetadataMode(t *testing.T) {
	valid := `{"rawQuery":"private editor text","time":{"mode":"relative","duration":"15m"},"limit":100,"resultMode":"table","historyMode":"coalesced","historyId":7}`
	if _, _, _, _, _, err := decodeRunQuery(strings.NewReader(valid), "prometheus", time.Now().UTC()); err != nil {
		t.Fatalf("approved coalesced metadata contract rejected: %v", err)
	}
	for _, body := range []string{strings.Replace(valid, "coalesced", "arbitrary", 1), strings.Replace(valid, "coalesced", "record", 1)} {
		if _, _, _, _, _, err := decodeRunQuery(strings.NewReader(body), "prometheus", time.Now().UTC()); err == nil {
			t.Fatal("unsupported mode or record-mode history rewrite accepted")
		}
	}
}
