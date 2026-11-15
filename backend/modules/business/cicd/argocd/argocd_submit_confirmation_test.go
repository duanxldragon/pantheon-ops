package argocd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/stretchr/testify/require"
)

// V4's eight independent API/Kubernetes observations: one HTTP200 PATCH,
// unchanged UID/spec/resourceVersion, no operation/state, no Service. This
// regression reproduces that response shape, not a simulated terminal phase.
func TestArgoSubmitUnpersistedOperationRemainsUnknown(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "sync", true: "rollback"}[rollback], func(t *testing.T) {
			var calls atomic.Int32
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				app := submittedApplication(t, r)
				app.Operation = nil // Actual observed provider projection.
				app.Metadata.ResourceVersion = "1"
				_ = json.NewEncoder(w).Encode(app)
			})
			revision := fixtureRevision
			if rollback {
				revision = fixtureOldRevision
			}
			actor := productActor()
			preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: revision, Rollback: rollback}, actor)
			require.NoError(t, err)
			actor.IntentVersion = preview.IntentVersion
			result, err := s.submitApplicationAction(context.Background(), id, revision, rollback, actor)
			require.Error(t, err)
			require.Nil(t, result)
			var pending ArgoCDSyncHistory
			require.NoError(t, s.db.Where("application_id = ?", id).First(&pending).Error)
			require.Equal(t, AppStatusUnknown, pending.Status)
			require.Nil(t, pending.FinishedAt)
			var app ArgoCDApplication
			require.NoError(t, s.db.First(&app, id).Error)
			require.Equal(t, AppStatusUnknown, app.Status)
			require.Nil(t, app.LastSyncedAt)
			var server ArgoCDServer
			require.NoError(t, s.db.First(&server, app.ServerID).Error)
			client, err := s.clientFactory(server.Endpoint, "", "")
			require.NoError(t, err)
			require.NoError(t, NewReconciler(s.db, time.Second).reconcileApplication(context.Background(), client, &app))
			require.NoError(t, s.db.First(&pending, pending.ID).Error)
			require.Equal(t, AppStatusUnknown, pending.Status)
			require.Nil(t, pending.FinishedAt)
			recovered, err := s.submitApplicationAction(context.Background(), id, revision, rollback, actor)
			require.NoError(t, err)
			require.Equal(t, pending.ID, recovered.ID)
			require.Equal(t, AppStatusUnknown, recovered.Status)
			actor.IntentVersion = ""
			preview, err = s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: revision, Rollback: rollback}, actor)
			require.NoError(t, err)
			actor.IntentVersion = preview.IntentVersion
			_, err = s.submitApplicationAction(context.Background(), id, revision, rollback, actor)
			require.ErrorIs(t, err, common.ErrConflict)
			require.Equal(t, int32(1), calls.Load(), "uncertainty cannot redispatch, including under a new ticket")
		})
	}
}

func submittedApplication(t *testing.T, r *http.Request) argoApplication {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	var body struct {
		Name, AppNamespace, Project, Revision string
		ID                                    *int64 `json:"id"`
		Prune                                 bool
		Reviewed                              reviewedActionFields
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	require.Equal(t, "test-app", body.Name)
	require.True(t, strings.Contains(r.URL.Path, "/reviewed-actions/"))
	require.Equal(t, "provider-app", body.Reviewed.ExpectedUID)
	require.Equal(t, "1", body.Reviewed.ExpectedResourceVersion)
	require.NotEmpty(t, body.Reviewed.ActionMarker)
	var app argoApplication
	app.Metadata.UID, app.Metadata.ResourceVersion = body.Reviewed.ExpectedUID, "2"
	app.Spec = json.RawMessage(`{"project":"default","source":{"repoURL":"https://fixture.example.invalid/repository.git","path":".","targetRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"destination":{"server":"https://kubernetes.default.svc","namespace":"test"}}`)
	app.Operation = &argoOperation{Info: []argoOperationInfo{{Name: actionInfoName, Value: body.Reviewed.ActionMarker}}}
	app.Operation.Sync.Revision = body.Reviewed.ResolvedRevision
	app.Operation.Sync.Prune = body.Prune
	app.Operation.Sync.Source = json.RawMessage(`{"repoURL":"https://fixture.example.invalid/repository.git","path":".","targetRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
	if strings.HasSuffix(r.URL.Path, "/rollback") {
		require.NotNil(t, body.ID)
		require.Equal(t, int64(4), *body.ID)
	} else {
		require.Nil(t, body.ID)
		require.Equal(t, body.Reviewed.ResolvedRevision, body.Revision)
	}
	return app
}

// Successful unit fixtures return the requested operation explicitly. They do
// not stand in for official server behavior; the v4-shaped test omits it.
func writeFixtureAcceptedOperation(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	if r.Method != http.MethodPost {
		_, _ = w.Write([]byte(`{}`))
		return
	}
	app := submittedApplication(t, r)
	require.NoError(t, json.NewEncoder(w).Encode(app))
}

func TestArgoSubmitJSONComparisonPreservesSpecNumbers(t *testing.T) {
	require.False(t, sameProviderJSON(json.RawMessage(`{"n":9007199254740992}`), json.RawMessage(`{"n":9007199254740993}`)), "spec numbers must not collapse through float64 rounding")
	require.True(t, sameProviderJSON(json.RawMessage(`{"n":9007199254740993,"project":"default"}`), json.RawMessage(`{"project":"default","n":9007199254740993}`)), "object key order is not spec drift")
}

func TestArgoSubmitRequiresExactProviderConfirmation(t *testing.T) {
	for _, change := range []string{"unchanged-version", "different-uid", "different-spec", "missing-operation", "different-marker", "duplicate-marker", "different-revision", "different-prune", "different-source", "empty-response", "malformed-response", "oversized-response", "matching-operation", "matching-recorded-operation", "unrecognized-recorded-phase"} {
		t.Run(change, func(t *testing.T) {
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				app := submittedApplication(t, r)
				switch change {
				case "unchanged-version":
					app.Metadata.ResourceVersion = "1"
				case "different-uid":
					app.Metadata.UID = "replacement"
				case "different-spec":
					app.Spec = json.RawMessage(`{"project":"changed"}`)
				case "missing-operation":
					app.Operation = nil
				case "different-marker":
					app.Operation.Info[0].Value = "different"
				case "duplicate-marker":
					app.Operation.Info = append(app.Operation.Info, app.Operation.Info[0])
				case "different-revision":
					app.Operation.Sync.Revision = fixtureOldRevision
				case "different-prune":
					app.Operation.Sync.Prune = true
				case "different-source":
					app.Operation.Sync.Source = json.RawMessage(`{"path":"changed"}`)
				case "empty-response":
					return
				case "malformed-response":
					_, _ = w.Write([]byte("private-provider-payload"))
					return
				case "oversized-response":
					_, _ = w.Write([]byte(strings.Repeat("x", providerReadLimit+1)))
					return
				case "matching-recorded-operation", "unrecognized-recorded-phase":
					var state ArgoCDApplicationStatus
					data, err := json.Marshal(map[string]interface{}{"operationState": map[string]interface{}{"operation": app.Operation, "phase": "Running"}})
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(data, &state))
					if change == "unrecognized-recorded-phase" {
						state.OperationState.Phase = "private-provider-payload"
					}
					app.Status = state
					app.Operation = nil
				}
				_ = json.NewEncoder(w).Encode(app)
			}))
			defer provider.Close()
			client, err := newTestArgoCDClient(provider.URL, "", "")
			require.NoError(t, err)
			app := &argoApplication{Spec: json.RawMessage(`{"project":"default","source":{"repoURL":"https://fixture.example.invalid/repository.git","path":".","targetRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"destination":{"server":"https://kubernetes.default.svc","namespace":"test"}}`)}
			app.Metadata.UID, app.Metadata.ResourceVersion = "provider-app", "1"
			op := &argoOperation{Info: []argoOperationInfo{{Name: actionInfoName, Value: "unique"}}}
			op.Sync.Revision = fixtureRevision
			op.Sync.Source = json.RawMessage(`{"repoURL":"https://fixture.example.invalid/repository.git","path":".","targetRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`)
			err = client.submitReviewedOperation(context.Background(), "test-app", app, op, nil)
			if strings.HasPrefix(change, "matching-") {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "private-provider-payload")
			}
		})
	}
}
