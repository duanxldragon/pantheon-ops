package argocd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestReconcilerStopIsIdempotent(t *testing.T) {
	reconciler := NewReconciler(nil, time.Second)
	reconciler.Stop()
	reconciler.Stop()
}

func TestObservedApplicationOutcome(t *testing.T) {
	for _, tt := range []struct {
		name, phase, sync, health string
		want                      applicationOutcome
	}{
		{"undated", "", AppStatusSynced, HealthHealthy, applicationOutcome{AppStatusUnknown, syncHistoryRunning, false}},
		{"running on old degraded health", "Running", AppStatusSynced, HealthDegraded, applicationOutcome{appStatusProgressing, syncHistoryRunning, false}},
		{"terminating", "Terminating", AppStatusSynced, HealthHealthy, applicationOutcome{appStatusProgressing, syncHistoryRunning, false}},
		{"operation success still progressing", "Succeeded", AppStatusSynced, HealthProgressing, applicationOutcome{appStatusProgressing, syncHistoryRunning, false}},
		{"operation success still out of sync", "Succeeded", AppStatusOutOfSync, HealthHealthy, applicationOutcome{appStatusProgressing, syncHistoryRunning, false}},
		{"degraded", "Succeeded", AppStatusSynced, HealthDegraded, applicationOutcome{AppStatusError, syncHistoryFailed, true}},
		{"missing", "Succeeded", AppStatusSynced, HealthMissing, applicationOutcome{AppStatusError, syncHistoryFailed, true}},
		{"operation failed", "Failed", AppStatusSynced, HealthHealthy, applicationOutcome{AppStatusError, syncHistoryFailed, true}},
		{"operation error", "Error", AppStatusSynced, HealthHealthy, applicationOutcome{AppStatusError, syncHistoryFailed, true}},
		{"healthy success", "Succeeded", AppStatusSynced, HealthHealthy, applicationOutcome{AppStatusSynced, syncHistorySucceeded, true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := &ArgoCDApplicationStatus{}
			if err := json.Unmarshal(fmt.Appendf(nil, `{"sync":{"status":%q},"health":{"status":%q},"operationState":{"phase":%q}}`, tt.sync, tt.health, tt.phase), status); err != nil {
				t.Fatal(err)
			}
			if got := observedApplicationOutcome(status); got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	if got := observedApplicationOutcome(nil); got.finished || got.status != AppStatusUnknown {
		t.Fatalf("nil observation: %+v", got)
	}
}

func TestHistoryMatchesObservation(t *testing.T) {
	started := time.Now().UTC().Truncate(time.Second)
	history := &ArgoCDSyncHistory{Revision: "main", StartedAt: started}
	for _, tt := range []struct {
		name, requested, observed string
		offset                    time.Duration
		dated, want               bool
	}{
		{"old operation", "main", "sha", -time.Second, true, false},
		{"undated operation", "main", "sha", 0, false, false},
		{"requested branch takes precedence", "main", "sha", time.Second, true, true},
		{"different requested branch", "other", "main", time.Second, true, false},
		{"revision fallback", "", "main", time.Second, true, true},
		{"revision mismatch", "", "sha", time.Second, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			status := &ArgoCDApplicationStatus{}
			if err := json.Unmarshal(fmt.Appendf(nil, `{"sync":{"revision":%q},"operationState":{"operation":{"sync":{"revision":%q}}}}`, tt.observed, tt.requested), status); err != nil {
				t.Fatal(err)
			}
			if tt.dated {
				remoteStarted := started.Add(tt.offset)
				status.OperationState.StartedAt = &remoteStarted
			}
			if got := historyMatchesObservation(history, status); got != tt.want {
				t.Fatalf("matched=%v, want=%v", got, tt.want)
			}
		})
	}
	if historyMatchesObservation(nil, nil) {
		t.Fatal("nil correlation accepted")
	}
}

func TestReconcilerStartAlreadyStoppedOrCanceled(t *testing.T) {
	for _, stopFirst := range []bool{false, true} {
		r := NewReconciler(nil, time.Hour)
		ctx, cancel := context.WithCancel(context.Background())
		if stopFirst {
			r.Stop()
		} else {
			cancel()
		}
		done := make(chan struct{})
		go func() { r.Start(ctx); r.Start(ctx); close(done) }()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Start did not return after cancellation/Stop")
		}
		cancel()
		r.Stop()
	}
}

func TestObservedRemoteMissingIsDistinctFromOutage(t *testing.T) {
	for _, code := range []int{http.StatusNotFound, http.StatusServiceUnavailable} {
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) }))
		client := &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}
		_, err := client.GetApplicationStatus(context.Background(), "test-app")
		remote.Close()
		if err == nil || errors.Is(err, ErrApplicationNotFound) != (code == http.StatusNotFound) {
			t.Fatalf("HTTP %d: %v", code, err)
		}
	}
}

func TestSyncRejectionIsDistinctFromUncertainResponse(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusConflict, http.StatusServiceUnavailable} {
		remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte("provider-secret"))
		}))
		client := &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}
		err := client.SyncApplication(context.Background(), "test-app", "main")
		remote.Close()
		if err == nil || errors.Is(err, errSyncRequestRejected) != (code < 500) {
			t.Fatalf("HTTP %d: %v", code, err)
		}
		if strings.Contains(err.Error(), "provider-secret") {
			t.Fatal("provider response leaked")
		}
	}
}

func TestReconciliationErrorClassOmitsProviderDetails(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want string
	}{
		{context.Canceled, "canceled"},
		{fmt.Errorf("sensitive-provider-response: %w", context.DeadlineExceeded), "deadline_exceeded"},
		{errCiphertextTampered, "credential_configuration"},
		{ErrInvalidCACert, "certificate_configuration"},
		{ErrApplicationNotFound, "not_found"},
		{&net.DNSError{Err: "sensitive-provider-response", Name: "sensitive-provider-response"}, "network"},
		{errors.New("sensitive-provider-response"), "dependency_failure"},
	} {
		if got := reconciliationErrorClass(tt.err); got != tt.want {
			t.Fatalf("error classification got=%q want=%q", got, tt.want)
		}
	}
}

func TestReconcileDoesNotPromoteUnhealthyAcceptedSync(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ArgoCDApplication{}, &ArgoCDSyncHistory{}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().UTC().Add(-time.Minute).Truncate(time.Second)
	app := ArgoCDApplication{Name: "image-pull-failure", ServerID: 1, Status: AppStatusUnknown, SyncStatus: "Unknown", HealthStatus: HealthUnknown}
	if err := db.Create(&app).Error; err != nil {
		t.Fatal(err)
	}
	history := ArgoCDSyncHistory{ApplicationID: app.ID, Revision: "new-revision", Status: "Running", StartedAt: started}
	if err := db.Create(&history).Error; err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/image-pull-failure" {
			t.Errorf("unexpected remote path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":{"sync":{"status":"Synced","revision":"new-revision"},"health":{"status":"Degraded"},"operationState":{"phase":"Succeeded","message":"ImagePullBackOff","startedAt":%q}}}`, started.Add(time.Second).Format(time.RFC3339))
	}))
	defer remote.Close()
	client := &ArgoCDClient{endpoint: remote.URL, httpClient: remote.Client()}
	if err := NewReconciler(db, time.Second).reconcileApplication(context.Background(), client, &app); err != nil {
		t.Fatal(err)
	}
	var stored ArgoCDApplication
	if err := db.First(&stored, app.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != AppStatusError || stored.HealthStatus != HealthDegraded {
		t.Fatalf("unhealthy accepted sync must be aggregate Error/Degraded, got %+v", stored)
	}
	var histories []ArgoCDSyncHistory
	if err := db.Where("application_id = ?", app.ID).Find(&histories).Error; err != nil {
		t.Fatal(err)
	}
	for _, row := range histories {
		if row.Status == "Succeeded" {
			t.Fatalf("image-pull failure promoted to successful history: %+v", row)
		}
	}
	if len(histories) != 1 || histories[0].Status != "Failed" || histories[0].FinishedAt == nil {
		t.Fatalf("the original pending history must converge to Failed once, got %+v", histories)
	}
}
