package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func actionService(t *testing.T, handler http.HandlerFunc) (*ArgoCDService, uint64) {
	t.Helper()
	db := testmysql.Open(t)
	require.NoError(t, db.AutoMigrate(&ArgoCDServer{}, &ArgoCDApplication{}, &ArgoCDSyncHistory{}, &ArgoCDRepository{}))
	provider := httptest.NewServer(handler)
	t.Cleanup(provider.Close)
	server := ArgoCDServer{Name: "action-fixture", Endpoint: provider.URL, ClusterID: 1}
	require.NoError(t, db.Create(&server).Error)
	app := ArgoCDApplication{ServerID: server.ID, Name: "test-app", SyncStatus: "OutOfSync", Status: AppStatusOutOfSync, Project: "default", SourceRepoURL: "https://fixture.example.invalid/repository.git", SourcePath: ".", SourceTargetRev: fixtureRevision, DestinationServer: "https://kubernetes.default.svc", DestinationNS: "test", CreatedBy: "1"}
	require.NoError(t, db.Create(&app).Error)
	service := NewArgoCDService(db)
	require.NoError(t, db.Create(&ArgoCDRepository{ServerID: server.ID, RepoURL: app.SourceRepoURL, Status: "connected"}).Error)
	t.Setenv(encryptionKeyEnv, "12345678901234567890123456789012")
	service.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
		if req.Actor.ID != 1 || req.Actor.DataScope == nil || req.Actor.DataScope.UserID != 1 {
			return common.NewForbidden("common.forbidden")
		}
		return nil
	})
	service.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
		c, err := newTestArgoCDClient(endpoint, token, ca)
		if err != nil {
			return nil, err
		}
		c.httpClient.Transport = &fixtureReadTransport{base: c.httpClient.Transport, app: app}
		return c, nil
	}
	return service, app.ID
}
func TestApplicationActionRejectedIsNotSuccessful(t *testing.T) {
	for _, code := range []int{http.StatusForbidden, http.StatusPreconditionFailed, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType, http.StatusTooManyRequests} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
				_, _ = w.Write([]byte("token=provider-private-fixture"))
			})
			_, err := submitFixtureAction(s, context.Background(), id, "main", false)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "provider-private-fixture")
			var history ArgoCDSyncHistory
			require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
			require.Equal(t, "Failed", history.Status)
			require.NotContains(t, history.Message, "provider-private-fixture")
			require.NotNil(t, history.FinishedAt)
		})
	}
}
func TestApplicationActionAcceptanceIsNotCompletion(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		writeFixtureAcceptedOperation(t, w, r)
	})
	history, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.NoError(t, err)
	require.Equal(t, "Running", history.Status)
	require.Nil(t, history.FinishedAt)
	var app ArgoCDApplication
	require.NoError(t, s.db.First(&app, id).Error)
	require.Equal(t, "OutOfSync", app.SyncStatus)
	require.Nil(t, app.LastSyncedAt)
}

func TestApplicationServerFailureAfterMutationRemainsUnknown(t *testing.T) {
	for _, code := range []int{http.StatusInternalServerError, http.StatusRequestTimeout, 499, 460} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var applied atomic.Bool
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
				applied.Store(true)
				w.WriteHeader(code)
				_, _ = w.Write([]byte("token=provider-private-fixture"))
			})
			_, err := submitFixtureAction(s, context.Background(), id, "main", false)
			require.Error(t, err)
			require.True(t, applied.Load())
			require.NotContains(t, err.Error(), "provider-private-fixture")
			var history ArgoCDSyncHistory
			require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
			require.Equal(t, AppStatusUnknown, history.Status)
			require.Nil(t, history.FinishedAt)
			var app ArgoCDApplication
			require.NoError(t, s.db.First(&app, id).Error)
			require.Equal(t, AppStatusUnknown, app.Status)
		})
	}
}

func TestApplicationOutcomeWriteFailureKeepsDurableUncertainty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rollback bool
		status   int
		key      string
		failApp  bool
	}{
		{"accepted-sync", false, http.StatusOK, "cicd.argocd.app_sync_failed", false},
		{"uncertain-sync", false, http.StatusInternalServerError, "cicd.argocd.app_sync_failed", false},
		{"accepted-rollback", true, http.StatusOK, "cicd.argocd.app_rollback_failed", false},
		{"application-update-failure", false, http.StatusOK, "cicd.argocd.app_sync_failed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mutations atomic.Int32
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`{"status":{"history":[{"id":1,"revision":"main"}]}}`))
					return
				}
				mutations.Add(1)
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte("provider-private-fixture"))
			})
			require.NoError(t, s.db.Callback().Update().Before("gorm:update").Register("argocd-outcome-fault", func(tx *gorm.DB) {
				table := (ArgoCDSyncHistory{}).TableName()
				if tc.failApp {
					table = (ArgoCDApplication{}).TableName()
				}
				if tx.Statement.Table == table {
					tx.AddError(errors.New("private-record-fixture"))
				}
			}))
			t.Cleanup(func() { _ = s.db.Callback().Update().Remove("argocd-outcome-fault") })
			var err error
			if tc.rollback {
				_, err = submitFixtureAction(s, context.Background(), id, fixtureRevision, true)
			} else {
				_, err = submitFixtureAction(s, context.Background(), id, "main", false)
			}
			require.EqualError(t, err, tc.key)
			require.Equal(t, int32(1), mutations.Load(), "bookkeeping failure must never replay a provider mutation")
			var history ArgoCDSyncHistory
			require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
			require.Equal(t, AppStatusUnknown, history.Status)
			require.Nil(t, history.FinishedAt)
			metadata, ok := decodeActionMetadata(history.Message)
			require.True(t, ok)
			require.Empty(t, metadata.Error)
		})
	}
}
func TestApplicationRollbackUsesProviderHistory(t *testing.T) {
	var calls atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"status":{"history":[{"id":4,"revision":"old-revision"},{"id":6,"revision":"new-revision"}]}}`))
		case r.Method == http.MethodPost:
			require.True(t, strings.HasSuffix(r.URL.Path, "/reviewed-actions/rollback"))
			accepted := submittedApplication(t, r)
			require.Equal(t, fixtureOldRevision, accepted.Operation.Sync.Revision)
			require.NotEmpty(t, accepted.Operation.Sync.Source)
			calls.Add(1)
			require.NoError(t, json.NewEncoder(w).Encode(accepted))
		default:
			w.WriteHeader(400)
		}
	})
	history, err := submitFixtureAction(s, context.Background(), id, fixtureOldRevision, true)
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
	require.Equal(t, "Running", history.Status)
	require.Nil(t, history.FinishedAt)
	_, err = submitFixtureAction(s, context.Background(), id, "unknown-revision", true)
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}

func TestApplicationActionContextCancellation(t *testing.T) {
	started := make(chan struct{})
	released := make(chan struct{})
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-released:
		}
	})
	defer close(released)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := submitFixtureAction(s, ctx, id, "main", false); result <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("provider request did not start")
	}
	cancel()
	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("cancelled request still blocked")
	}
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
	require.Equal(t, AppStatusUnknown, history.Status)
	require.Nil(t, history.FinishedAt)
}

func TestApplicationReconcileUsesProviderFinishAndRedactsMessage(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":{"uid":"provider-app"},"status":{"sync":{"status":"OutOfSync","revision":"main"},"health":{"status":"Degraded"},"operationState":{"operation":{"info":[{"name":"pantheon-action-id","value":"failed-action"}],"sync":{"revision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}},"phase":"Failed","message":"token=provider-private-fixture","startedAt":"2026-11-09T00:00:00Z","finishedAt":"2026-11-09T00:00:12Z"}}}`))
	})
	pending := ArgoCDSyncHistory{ApplicationID: id, Revision: fixtureRevision, Status: AppStatusUnknown, StartedAt: time.Now(), Message: encodeActionMetadata(actionMetadata{Nonce: "failed-action", UID: "provider-app", Revision: fixtureRevision})}
	require.NoError(t, s.db.Create(&pending).Error)
	var app ArgoCDApplication
	require.NoError(t, s.db.First(&app, id).Error)
	var server ArgoCDServer
	require.NoError(t, s.db.First(&server, app.ServerID).Error)
	client, err := newTestArgoCDClient(server.Endpoint, "", "")
	require.NoError(t, err)
	require.NoError(t, (&Reconciler{db: s.db}).reconcileApplication(context.Background(), client, &app))
	require.NoError(t, s.db.First(&app, id).Error)
	require.NotContains(t, app.LastError, "provider-private-fixture")
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
	require.Equal(t, "Failed", history.Status)
	require.Equal(t, "cicd.argocd.app_sync_failed", publicActionMessage(history.Message))
	require.NotNil(t, history.FinishedAt)
	require.Equal(t, "2026-11-09T00:00:12Z", history.FinishedAt.UTC().Format(time.RFC3339))
}

const fixtureRevision = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const fixtureOldRevision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

type fixtureAuthority func(context.Context, ActionAuthorization) error

func (f fixtureAuthority) AuthorizeArgoAction(ctx context.Context, req ActionAuthorization) error {
	return f(ctx, req)
}

type fixtureReadTransport struct {
	base http.RoundTripper
	app  ArgoCDApplication
}

func (f *fixtureReadTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodGet {
		return f.base.RoundTrip(req)
	}
	source := map[string]interface{}{"repoURL": f.app.SourceRepoURL, "path": f.app.SourcePath, "targetRevision": f.app.SourceTargetRev}
	var result interface{}
	switch {
	case strings.HasSuffix(req.URL.Path, "/reviewed-actions/capabilities"):
		result = requiredReviewedCapabilities
	case strings.HasSuffix(req.URL.Path, "/manifests"):
		revision := fixtureRevision
		if req.URL.Query().Get("revision") == fixtureOldRevision {
			revision = fixtureOldRevision
		}
		result = map[string]interface{}{"revision": revision, "manifests": []string{}}
	case strings.HasSuffix(req.URL.Path, "/managed-resources"):
		result = map[string]interface{}{"items": []interface{}{}}
	default:
		result = map[string]interface{}{"metadata": map[string]interface{}{"uid": "provider-app", "resourceVersion": "1"}, "spec": map[string]interface{}{"project": f.app.Project, "source": source, "destination": map[string]interface{}{"server": f.app.DestinationServer, "namespace": f.app.DestinationNS}}, "status": map[string]interface{}{"history": []interface{}{map[string]interface{}{"id": 4, "revision": fixtureOldRevision, "source": source}, map[string]interface{}{"id": 1, "revision": fixtureRevision, "source": source}}}}
	}
	data, _ := json.Marshal(result)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header), Request: req}, nil
}
func submitFixtureAction(s *ArgoCDService, ctx context.Context, id uint64, revision string, rollback bool) (*SyncHistoryResponse, error) {
	actor := ActionActor{ID: 1, DataScope: &common.DataScopeReq{UserID: 1, IsAdmin: true, Mode: common.DataScopeModeAll}}
	preview, err := s.PreviewApplicationContext(ctx, id, ApplicationPreviewRequest{Revision: revision, Rollback: rollback}, actor)
	if err != nil {
		return nil, err
	}
	actor.IntentVersion = preview.IntentVersion
	return s.submitApplicationAction(ctx, id, revision, rollback, actor)
}
