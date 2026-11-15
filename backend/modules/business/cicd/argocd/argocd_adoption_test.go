package argocd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/stretchr/testify/require"
)

func finalJSONResponse(r *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: r}
}

func TestArgoAdoptionCapabilityBlocksBeforeReservation(t *testing.T) {
	for _, fault := range []string{"stock", "malformed", "protocol", "revision", "marker", "conditional", "policy", "rollback", "null", "missing", "duplicate", "unknown"} {
		t.Run(fault, func(t *testing.T) {
			var mutations atomic.Int32
			s, id := actionService(t, func(http.ResponseWriter, *http.Request) { mutations.Add(1) })
			factory := s.clientFactory
			s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
				c, err := factory(endpoint, token, ca)
				if err != nil {
					return nil, err
				}
				base := c.httpClient.Transport
				c.httpClient.Transport = finalTransport(func(r *http.Request) (*http.Response, error) {
					if !strings.HasSuffix(r.URL.Path, "/reviewed-actions/capabilities") {
						return base.RoundTrip(r)
					}
					cap := requiredReviewedCapabilities
					switch fault {
					case "protocol":
						cap.Protocol = "future"
					case "revision":
						cap.RevisionScope = "any"
					case "marker":
						cap.MarkerName = "other"
					case "conditional":
						cap.ConditionalUpdate = false
					case "policy":
						cap.NativePolicy = false
					case "rollback":
						cap.CorrelatedRollback = false
					}
					body, _ := json.Marshal(cap)
					status := http.StatusOK
					switch fault {
					case "stock":
						status = http.StatusNotFound
					case "malformed":
						body = []byte("private-fixture")
					case "null":
						body = []byte("null")
					case "missing":
						body = []byte(`{"protocol":"pantheon-reviewed-actions/v1"}`)
					case "duplicate":
						body = []byte(strings.Replace(string(body), `"conditionalUpdate":true`, `"conditionalUpdate":false,"conditionalUpdate":true`, 1))
					case "unknown":
						body = []byte(strings.TrimSuffix(string(body), "}") + `,"unknown":true}`)
					}
					return finalJSONResponse(r, status, string(body)), nil
				})
				return c, nil
			}
			_, err := submitFixtureAction(s, context.Background(), id, "main", false)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private-fixture")
			require.Zero(t, mutations.Load(), "no native/PATCH/cluster fallback")
			var count int64
			require.NoError(t, s.db.Model(&ArgoCDSyncHistory{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestArgoAdoptionRequiredWireFields(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "sync", true: "rollback-id-zero"}[rollback], func(t *testing.T) {
			var calls atomic.Int32
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				require.Equal(t, http.MethodPost, r.Method)
				action := "sync"
				if rollback {
					action = "rollback"
				}
				require.Equal(t, "/api/v1/applications/test-app/reviewed-actions/"+action, r.URL.Path)
				require.Equal(t, "Bearer fixture-token", r.Header.Get("Authorization"))
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.JSONEq(t, `"test-app"`, string(body["name"]))
				require.JSONEq(t, `"argo-ns"`, string(body["appNamespace"]))
				require.JSONEq(t, `"default"`, string(body["project"]))
				require.JSONEq(t, `{"expectedUID":"provider-app","expectedResourceVersion":"1","resolvedRevision":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","actionMarker":"one-marker"}`, string(body["reviewed"]))
				require.NotContains(t, body, "infos")
				require.NotContains(t, body, "source")
				if rollback {
					require.Equal(t, "0", string(body["id"]))
					require.NotContains(t, body, "revision")
				} else {
					require.NotContains(t, body, "id")
				}
				w.WriteHeader(http.StatusConflict)
			}))
			defer provider.Close()
			client, err := newTestArgoCDClient(provider.URL, "fixture-token", "")
			require.NoError(t, err)
			app := &argoApplication{Spec: json.RawMessage(`{"project":"default"}`)}
			app.Metadata.UID, app.Metadata.ResourceVersion, app.Metadata.Namespace = "provider-app", "1", "argo-ns"
			op := &argoOperation{Info: []argoOperationInfo{{Name: actionInfoName, Value: "one-marker"}}}
			op.Sync.Revision = fixtureRevision
			var id *int64
			if rollback {
				zero := int64(0)
				id = &zero
			}
			err = client.submitReviewedOperation(context.Background(), "test-app", app, op, id)
			require.ErrorIs(t, err, common.ErrConflict)
			require.Equal(t, int32(1), calls.Load())
			for _, field := range []string{"uid", "version", "revision", "marker", "duplicate", "negative"} {
				invalidApp, invalidOp := *app, *op
				invalidOp.Info = append([]argoOperationInfo(nil), op.Info...)
				badID := id
				negative := int64(-1)
				switch field {
				case "uid":
					invalidApp.Metadata.UID = ""
				case "version":
					invalidApp.Metadata.ResourceVersion = strings.Repeat("x", 129)
				case "revision":
					invalidOp.Sync.Revision = strings.Repeat("A", 40)
				case "marker":
					invalidOp.Info[0].Value = "invalid marker"
				case "duplicate":
					invalidOp.Info = append(invalidOp.Info, invalidOp.Info[0])
				case "negative":
					badID = &negative
				}
				require.Error(t, client.submitReviewedOperation(context.Background(), "test-app", &invalidApp, &invalidOp, badID))
			}
			require.Equal(t, int32(1), calls.Load(), "invalid fields never reach provider")
		})
	}
}

func TestArgoAdoptionHistoryIDPrivateAndImmutable(t *testing.T) {
	for _, drift := range []bool{false, true} {
		t.Run(map[bool]string{false: "same-id", true: "same-revision-new-id"}[drift], func(t *testing.T) {
			var mutations atomic.Int32
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
			actor := productActor()
			preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: fixtureOldRevision, Rollback: true}, actor)
			require.NoError(t, err)
			plain, err := DecryptSecret(preview.IntentVersion)
			require.NoError(t, err)
			var review applicationReview
			require.NoError(t, json.Unmarshal([]byte(plain), &review))
			require.NotNil(t, review.HistoryID)
			require.Equal(t, int64(4), *review.HistoryID)
			require.Equal(t, reviewedActionsProtocol, review.Protocol)
			public, _ := json.Marshal(preview)
			require.NotContains(t, string(public), "HistoryID")
			require.NotContains(t, string(public), "historyId")
			actor.IntentVersion = preview.IntentVersion
			if drift {
				factory := s.clientFactory
				s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
					c, err := factory(endpoint, token, ca)
					if err != nil {
						return nil, err
					}
					base := c.httpClient.Transport
					c.httpClient.Transport = finalTransport(func(r *http.Request) (*http.Response, error) {
						resp, err := base.RoundTrip(r)
						if err != nil || !strings.HasSuffix(r.URL.Path, "/test-app") {
							return resp, err
						}
						defer resp.Body.Close()
						var app argoApplication
						require.NoError(t, json.NewDecoder(resp.Body).Decode(&app))
						app.Status.History[0].ID = 5
						data, _ := json.Marshal(app)
						return finalJSONResponse(r, 200, string(data)), nil
					})
					return c, nil
				}
			}
			result, err := s.RollbackApplicationContext(context.Background(), id, fixtureOldRevision, actor)
			if drift {
				require.ErrorIs(t, err, common.ErrConflict)
				require.Zero(t, mutations.Load())
				return
			}
			require.NoError(t, err)
			var history ArgoCDSyncHistory
			require.NoError(t, s.db.First(&history, result.ID).Error)
			metadata, ok := decodeActionMetadata(history.Message)
			require.True(t, ok)
			require.True(t, sameHistoryID(review.HistoryID, metadata.HistoryID))
			require.NotContains(t, result.Message, "HistoryID")
		})
	}
}

func TestArgoAdoptionLostReplyColdRecovery(t *testing.T) {
	var calls atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: fixtureRevision}, actor)
	require.NoError(t, err)
	actor.IntentVersion = preview.IntentVersion
	_, err = s.SyncApplicationContext(context.Background(), id, fixtureRevision, actor)
	require.Error(t, err)
	var original ArgoCDSyncHistory
	require.NoError(t, s.db.First(&original).Error)
	metadata, ok := decodeActionMetadata(original.Message)
	require.True(t, ok)
	cold := NewArgoCDService(s.db)
	cold.authorizer = s.authorizer
	cold.clientFactory = func(string, string, string) (*ArgoCDClient, error) {
		t.Error("recovery reached provider")
		return nil, errors.New("unreachable")
	}
	result, err := cold.SyncApplicationContext(context.Background(), id, fixtureRevision, actor)
	require.NoError(t, err)
	require.Equal(t, original.ID, result.ID)
	require.Equal(t, AppStatusUnknown, result.Status)
	require.Nil(t, result.FinishedAt)
	var status ArgoCDApplicationStatus
	require.NoError(t, json.Unmarshal([]byte(`{"operationState":{"phase":"Succeeded","operation":{"sync":{}},"syncResult":{}}}`), &status))
	status.ApplicationUID = metadata.UID
	status.OperationState.Operation.Sync.Revision = metadata.Revision
	status.OperationState.Operation.Info = []argoOperationInfo{{Name: actionInfoName, Value: metadata.Nonce}}
	status.OperationState.SyncResult.Revision = metadata.Revision
	start, finish := time.Now().Add(-time.Second), time.Now()
	status.OperationState.StartedAt, status.OperationState.FinishedAt = &start, &finish
	var app ArgoCDApplication
	require.NoError(t, s.db.First(&app, id).Error)
	require.NoError(t, applyApplicationStatus(context.Background(), cold.db, &app, &status))
	result, err = cold.SyncApplicationContext(context.Background(), id, fixtureRevision, actor)
	require.NoError(t, err)
	require.Equal(t, original.ID, result.ID)
	require.Equal(t, "Succeeded", result.Status)
	require.NotNil(t, result.FinishedAt)
	require.Equal(t, int32(1), calls.Load())
}

func TestArgoAdoptionCapabilityDisappearsBeforeDispatch(t *testing.T) {
	var mutations, capabilities atomic.Int32
	s, id := actionService(t, func(http.ResponseWriter, *http.Request) { mutations.Add(1) })
	factory := s.clientFactory
	s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
		c, err := factory(endpoint, token, ca)
		if err != nil {
			return nil, err
		}
		base := c.httpClient.Transport
		c.httpClient.Transport = finalTransport(func(r *http.Request) (*http.Response, error) {
			if strings.HasSuffix(r.URL.Path, "/reviewed-actions/capabilities") && capabilities.Add(1) == 3 {
				return finalJSONResponse(r, http.StatusNotFound, `{}`), nil
			}
			return base.RoundTrip(r)
		})
		return c, nil
	}
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.Error(t, err)
	require.Zero(t, mutations.Load())
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.First(&history).Error)
	require.Equal(t, "Failed", history.Status, "capability rejection before dispatch is definitive")
	require.NotNil(t, history.FinishedAt)
}

func TestArgoAdoptionMissingReviewedRouteHasNoFallback(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		mutations.Add(1)
		require.True(t, strings.HasSuffix(r.URL.Path, "/reviewed-actions/sync"))
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.Error(t, err)
	require.Equal(t, int32(1), mutations.Load())
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.First(&history).Error)
	require.Equal(t, "Failed", history.Status)
	require.NotNil(t, history.FinishedAt)
}

func TestArgoAdoptionObsoleteUndispatchedTicketConflicts(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(http.ResponseWriter, *http.Request) { mutations.Add(1) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: fixtureRevision}, actor)
	require.NoError(t, err)
	plain, err := DecryptSecret(preview.IntentVersion)
	require.NoError(t, err)
	var review applicationReview
	require.NoError(t, json.Unmarshal([]byte(plain), &review))
	review.Protocol = ""
	data, _ := json.Marshal(review)
	actor.IntentVersion, err = EncryptSecret(string(data))
	require.NoError(t, err)
	_, err = s.SyncApplicationContext(context.Background(), id, fixtureRevision, actor)
	require.ErrorIs(t, err, common.ErrConflict)
	require.Zero(t, mutations.Load())
}

func TestArgoAdoptionRevocationDuringCapabilityReadBlocksDispatch(t *testing.T) {
	var mutations, capabilities atomic.Int32
	var revoked atomic.Bool
	s, id := actionService(t, func(http.ResponseWriter, *http.Request) { mutations.Add(1) })
	factory := s.clientFactory
	s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
		c, err := factory(endpoint, token, ca)
		if err != nil {
			return nil, err
		}
		base := c.httpClient.Transport
		c.httpClient.Transport = finalTransport(func(r *http.Request) (*http.Response, error) {
			if strings.HasSuffix(r.URL.Path, "/reviewed-actions/capabilities") && capabilities.Add(1) == 3 {
				revoked.Store(true)
			}
			return base.RoundTrip(r)
		})
		return c, nil
	}
	authority := s.authorizer
	s.authorizer = fixtureAuthority(func(ctx context.Context, req ActionAuthorization) error {
		if revoked.Load() {
			return common.NewForbidden("common.forbidden")
		}
		return authority.AuthorizeArgoAction(ctx, req)
	})
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.ErrorIs(t, err, common.ErrForbidden)
	require.Zero(t, mutations.Load())
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.First(&history).Error)
	require.Equal(t, "Failed", history.Status)
	require.NotNil(t, history.FinishedAt)
}
