package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

func newObservationFixture(t *testing.T) (*gorm.DB, ArgoCDApplication) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ArgoCDServer{}, &ArgoCDApplication{}, &ArgoCDSyncHistory{}); err != nil {
		t.Fatal(err)
	}
	server := ArgoCDServer{Name: "test-server", Endpoint: "https://argocd.example.test", Status: ArgoCDStatusDisconnected}
	if err := db.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	app := ArgoCDApplication{Name: "test-app", ServerID: server.ID, SourceTargetRev: "main", Status: AppStatusUnknown, SyncStatus: AppStatusUnknown, HealthStatus: HealthUnknown}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	return db, app
}

func readObservationFixture(t *testing.T, db *gorm.DB, id uint64) (ArgoCDApplication, []ArgoCDSyncHistory) {
	t.Helper()
	var app ArgoCDApplication
	if err := db.First(&app, id).Error; err != nil {
		t.Fatal(err)
	}
	var histories []ArgoCDSyncHistory
	if err := db.Where("application_id = ?", id).Order("id ASC").Find(&histories).Error; err != nil {
		t.Fatal(err)
	}
	return app, histories
}

func TestSyncAcceptanceAndDeduplication(t *testing.T) {
	db, app := newObservationFixture(t)
	var requests atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/applications/test-app/sync" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer remote.Close()
	service := NewArgoCDService(db)
	service.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	first, err := service.SyncApplication(app.ID, "main")
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != syncHistoryRunning || first.FinishedAt != nil {
		t.Fatalf("acceptance completed history: %+v", first)
	}
	repeated, err := service.SyncApplication(app.ID, "main")
	if err != nil || repeated.ID != first.ID {
		t.Fatalf("duplicate result: %+v, %v", repeated, err)
	}
	if _, err := service.SyncApplication(app.ID, "other"); err == nil || err.Error() != "business.cicd.application.syncInProgress" {
		t.Fatalf("different revision admitted: %v", err)
	}
	stored, histories := readObservationFixture(t, db, app.ID)
	if requests.Load() != 1 || len(histories) != 1 || stored.Status != appStatusProgressing || stored.LastSyncedAt != nil || stored.SyncStatus != AppStatusUnknown || stored.HealthStatus != HealthUnknown {
		t.Fatalf("accepted sync state: requests=%d app=%+v histories=%+v", requests.Load(), stored, histories)
	}
}

func TestSyncRequestFailurePersistence(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			db, app := newObservationFixture(t)
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte("sensitive-provider-response"))
			}))
			defer remote.Close()
			service := NewArgoCDService(db)
			service.newClient = func(string, string, string) (*ArgoCDClient, error) {
				return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
			}
			response, err := service.SyncApplication(app.ID, "main")
			if response != nil || err == nil || err.Error() != appSyncFailedKey {
				t.Fatalf("request error hidden/leaked: %+v, %v", response, err)
			}
			stored, histories := readObservationFixture(t, db, app.ID)
			if len(histories) != 1 || histories[0].Message != appSyncFailedKey || stored.LastError != histories[0].Message || stored.LastSyncedAt != nil {
				t.Fatalf("failure not persisted safely: %+v %+v", stored, histories)
			}
			if code < 500 {
				if histories[0].Status != syncHistoryFailed || histories[0].FinishedAt == nil || stored.Status != AppStatusError {
					t.Fatalf("rejection unfinished: %+v %+v", stored, histories)
				}
			} else if histories[0].Status != syncHistoryRunning || histories[0].FinishedAt != nil || stored.Status != appStatusProgressing {
				t.Fatalf("uncertain response incorrectly finalized: %+v %+v", stored, histories)
			}
		})
	}
}

func TestReconcilePendingObservationLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name, phase, health, revision string
		age, remoteOffset             time.Duration
		code                          int
		wantStatus, wantHistory       string
		finished                      bool
	}{
		{"healthy", "Succeeded", HealthHealthy, "main", time.Minute, time.Second, 200, AppStatusSynced, syncHistorySucceeded, true},
		{"progressing", "Succeeded", HealthProgressing, "main", time.Minute, time.Second, 200, appStatusProgressing, syncHistoryRunning, false},
		{"old operation", "Succeeded", HealthHealthy, "main", time.Minute, -time.Second, 200, appStatusProgressing, syncHistoryRunning, false},
		{"different revision", "Succeeded", HealthHealthy, "other", time.Minute, time.Second, 200, appStatusProgressing, syncHistoryRunning, false},
		{"timeout", "Running", HealthProgressing, "main", syncObservationTimeout + time.Minute, time.Second, 200, AppStatusUnknown, syncHistoryUnknown, true},
		{"timeout on outage", "", "", "", syncObservationTimeout + time.Minute, time.Second, 503, AppStatusUnknown, syncHistoryUnknown, true},
		{"remote deletion", "", "", "", time.Minute, time.Second, 404, AppStatusError, syncHistoryFailed, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, app := newObservationFixture(t)
			started := time.Now().UTC().Add(-tt.age).Truncate(time.Second)
			history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "main", Status: syncHistoryRunning, StartedAt: started}
			if err := db.Create(&history).Error; err != nil {
				t.Fatal(err)
			}
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.code)
				_, _ = fmt.Fprintf(w, `{"status":{"sync":{"status":"Synced","revision":"sha"},"health":{"status":%q},"operationState":{"phase":%q,"startedAt":%q,"operation":{"sync":{"revision":%q}}}}}`, tt.health, tt.phase, started.Add(tt.remoteOffset).Format(time.RFC3339), tt.revision)
			}))
			defer remote.Close()
			client := &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}
			r := NewReconciler(db, time.Second)
			if err := r.reconcileApplication(context.Background(), client, &app); err != nil {
				t.Fatal(err)
			}
			stored, histories := readObservationFixture(t, db, app.ID)
			if stored.Status != tt.wantStatus || len(histories) != 1 || histories[0].ID != history.ID || histories[0].Status != tt.wantHistory || (histories[0].FinishedAt != nil) != tt.finished {
				t.Fatalf("unexpected convergence: %+v %+v", stored, histories)
			}
			if (stored.LastSyncedAt != nil) != (tt.wantStatus == AppStatusSynced) {
				t.Fatalf("successful timestamp: %+v", stored)
			}
			if tt.wantHistory == syncHistoryUnknown && histories[0].Message != syncObservationTimeoutKey {
				t.Fatalf("timeout cause missing: %+v", histories[0])
			}
			if tt.name == "healthy" {
				finished := *histories[0].FinishedAt
				if err := r.reconcileApplication(context.Background(), client, &app); err != nil {
					t.Fatal(err)
				}
				_, rows := readObservationFixture(t, db, app.ID)
				if len(rows) != 1 || !rows[0].FinishedAt.Equal(finished) {
					t.Fatalf("original history completed twice: %+v", rows)
				}
			}
		})
	}
}

func TestReconcileDiscardsSupersededObservation(t *testing.T) {
	for _, changeSource := range []bool{false, true} {
		t.Run(fmt.Sprint(changeSource), func(t *testing.T) {
			db, app := newObservationFixture(t)
			started := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
			history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "main", Status: syncHistoryRunning, StartedAt: started}
			if err := db.Create(&history).Error; err != nil {
				t.Fatal(err)
			}
			remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if changeSource {
					if err := db.Model(&ArgoCDApplication{}).Where("id = ?", app.ID).Update("source_path", "new-path").Error; err != nil {
						t.Error(err)
					}
				} else {
					newer := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "other", Status: syncHistoryRunning, StartedAt: started.Add(time.Second)}
					if err := db.Create(&newer).Error; err != nil {
						t.Error(err)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"status":{"sync":{"status":"Synced","revision":"main"},"health":{"status":"Healthy"},"operationState":{"phase":"Succeeded","startedAt":%q}}}`, started.Add(time.Second).Format(time.RFC3339))
			}))
			defer remote.Close()
			client := &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}
			if err := NewReconciler(db, time.Second).reconcileApplication(context.Background(), client, &app); err != nil {
				t.Fatal(err)
			}
			stored, rows := readObservationFixture(t, db, app.ID)
			if stored.Status != AppStatusUnknown || stored.LastSyncedAt != nil {
				t.Fatalf("stale observation overwrote app: %+v", stored)
			}
			for _, row := range rows {
				if row.Status != syncHistoryRunning || row.FinishedAt != nil {
					t.Fatalf("stale observation completed history: %+v", row)
				}
			}
		})
	}
}

func TestReconcileRetriesDisconnectedServer(t *testing.T) {
	db, app := newObservationFixture(t)
	var probes atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			probes.Add(1)
			w.WriteHeader(http.StatusOK)
			return
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"status": map[string]any{"sync": map[string]string{"status": "OutOfSync"}, "health": map[string]string{"status": "Progressing"}}}); err != nil {
			t.Error(err)
		}
	}))
	defer remote.Close()
	r := NewReconciler(db, time.Second)
	r.modeReader = argoCDTestMode{mode: tenant.ModeCompat}
	r.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	if err := r.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	var server ArgoCDServer
	if err := db.First(&server, app.ServerID).Error; err != nil {
		t.Fatal(err)
	}
	if probes.Load() != 1 || server.Status != ArgoCDStatusConnected || server.LastChecked == nil {
		t.Fatalf("disconnected server not retried: probes=%d %+v", probes.Load(), server)
	}
}

func TestReconcileExpiresPendingHistoryDuringServerOutage(t *testing.T) {
	db, app := newObservationFixture(t)
	app.Status = appStatusProgressing
	if err := db.Save(&app).Error; err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-syncObservationTimeout - time.Minute).Truncate(time.Second)
	history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "main", Status: syncHistoryRunning, StartedAt: started}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	var probes, applicationReads atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			probes.Add(1)
		} else {
			applicationReads.Add(1)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("sensitive-provider-response"))
	}))
	defer remote.Close()
	r := NewReconciler(db, time.Second)
	r.modeReader = argoCDTestMode{mode: tenant.ModeCompat}
	r.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	if err := r.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	stored, histories := readObservationFixture(t, db, app.ID)
	if stored.Status != AppStatusUnknown || stored.LastSyncedAt != nil || len(histories) != 1 || histories[0].Status != syncHistoryUnknown || histories[0].FinishedAt == nil || histories[0].Message != syncObservationTimeoutKey {
		t.Fatalf("server outage left pending history unresolved: app=%+v histories=%+v", stored, histories)
	}
	finished := *histories[0].FinishedAt
	var server ArgoCDServer
	if err := db.First(&server, app.ServerID).Error; err != nil {
		t.Fatal(err)
	}
	if server.Status != ArgoCDStatusDisconnected || server.LastChecked == nil || probes.Load() != 1 || applicationReads.Load() != 0 {
		t.Fatalf("outage state/request bounds: server=%+v probes=%d applicationReads=%d", server, probes.Load(), applicationReads.Load())
	}
	// Persistent outage keeps retrying the server but cannot complete history twice.
	if err := r.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := r.reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, histories = readObservationFixture(t, db, app.ID)
	if len(histories) != 1 || !histories[0].FinishedAt.Equal(finished) || probes.Load() != 2 || applicationReads.Load() != 0 {
		t.Fatalf("persistent outage changed terminal history: %+v probes=%d reads=%d", histories, probes.Load(), applicationReads.Load())
	}
}

func TestReconcileServerOutagePreservesRecentPendingHistory(t *testing.T) {
	db, app := newObservationFixture(t)
	app.Status = appStatusProgressing
	if err := db.Save(&app).Error; err != nil {
		t.Fatal(err)
	}
	history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "main", Status: syncHistoryRunning, StartedAt: time.Now().UTC().Add(-time.Minute)}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	before, _ := readObservationFixture(t, db, app.ID)
	var applicationReads atomic.Int32
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			applicationReads.Add(1)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer remote.Close()
	r := NewReconciler(db, time.Second)
	r.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	var server ArgoCDServer
	if err := db.First(&server, app.ServerID).Error; err != nil {
		t.Fatal(err)
	}
	if err := r.reconcileServer(context.Background(), &server); err == nil {
		t.Fatal("server failure evidence was lost")
	}
	stored, rows := readObservationFixture(t, db, app.ID)
	if applicationReads.Load() != 0 || stored.Status != appStatusProgressing || len(rows) != 1 || rows[0].Status != syncHistoryRunning || rows[0].FinishedAt != nil || !stored.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("outage prematurely finalized recent request: %+v %+v reads=%d", stored, rows, applicationReads.Load())
	}
}

func TestReconcilerStopCancelsInFlightServerProbe(t *testing.T) {
	db, app := newObservationFixture(t)
	history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "main", Status: syncHistoryRunning, StartedAt: time.Now().UTC().Add(-syncObservationTimeout - time.Minute)}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	started, requestCanceled := make(chan struct{}), make(chan struct{})
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/version" {
			t.Errorf("unexpected application request during cancellation: %s", r.URL.Path)
			return
		}
		close(started)
		<-r.Context().Done()
		close(requestCanceled)
	}))
	defer remote.Close()
	r := NewReconciler(db, time.Hour)
	defer r.Stop()
	r.modeReader = argoCDTestMode{mode: tenant.ModeCompat}
	r.newClient = func(string, string, string) (*ArgoCDClient, error) {
		return &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}, nil
	}
	done := make(chan struct{})
	go func() {
		r.Start(context.Background())
		close(done)
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("startup did not issue an immediate probe")
	}
	r.Stop()
	r.Stop()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not release the background loop")
	}
	select {
	case <-requestCanceled:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop did not cancel the in-flight HTTP request")
	}
	_, rows := readObservationFixture(t, db, app.ID)
	if len(rows) != 1 || rows[0].Status != syncHistoryRunning || rows[0].FinishedAt != nil {
		t.Fatalf("canceled loop persisted a partial expiry: %+v", rows)
	}
}
