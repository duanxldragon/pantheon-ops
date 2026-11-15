package argocd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProductUnrelatedTerminalCannotCompleteSubmittedAction(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"metadata":{"uid":"provider-app"},"status":{"sync":{"status":"Synced","revision":"previous"},"health":{"status":"Healthy"},"operationState":{"phase":"Succeeded","startedAt":"2026-11-09T00:00:00Z","finishedAt":"2026-11-09T00:00:12Z"}}}`))
	})
	pending := ArgoCDSyncHistory{ApplicationID: id, Revision: "next", Status: AppStatusUnknown, StartedAt: time.Now()}
	require.NoError(t, s.db.Create(&pending).Error)
	var app ArgoCDApplication
	require.NoError(t, s.db.First(&app, id).Error)
	var server ArgoCDServer
	require.NoError(t, s.db.First(&server, app.ServerID).Error)
	client, err := newTestArgoCDClient(server.Endpoint, "", "")
	require.NoError(t, err)
	require.NoError(t, (&Reconciler{db: s.db}).reconcileApplication(context.Background(), client, &app))
	require.NoError(t, s.db.First(&app, id).Error)
	require.Equal(t, AppStatusUnknown, app.Status, "previous/unrelated provider terminal must not masquerade as submitted success")
	var histories []ArgoCDSyncHistory
	require.NoError(t, s.db.Where("application_id = ?", id).Find(&histories).Error)
	require.Len(t, histories, 1, "reconciliation must resolve original actions, not invent submitted completions")
	require.Equal(t, AppStatusUnknown, histories[0].Status)
	require.Nil(t, histories[0].FinishedAt)
}

func TestProductProviderStatusLiteralsAreNotPersisted(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":{"sync":{"status":"repository-password=private-fixture","revision":"private-fixture"},"health":{"status":"private-fixture"},"operationState":{"phase":"private-fixture"}}}`))
	})
	var app ArgoCDApplication
	require.NoError(t, s.db.First(&app, id).Error)
	var server ArgoCDServer
	require.NoError(t, s.db.First(&server, app.ServerID).Error)
	client, err := newTestArgoCDClient(server.Endpoint, "", "")
	require.NoError(t, err)
	require.NoError(t, (&Reconciler{db: s.db}).reconcileApplication(context.Background(), client, &app))
	require.NoError(t, s.db.First(&app, id).Error)
	require.NotContains(t, app.SyncStatus+app.HealthStatus+app.Status+app.LastError, "private-fixture")
}

func TestProductUnknownActionCannotBeResentAfterRestart(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			mutations.Add(1)
		}
		w.WriteHeader(http.StatusInternalServerError)
	})
	pending := ArgoCDSyncHistory{ApplicationID: id, Revision: "main", Status: AppStatusUnknown, StartedAt: time.Now()}
	require.NoError(t, s.db.Create(&pending).Error)
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.Error(t, err)
	require.Zero(t, mutations.Load(), "restart must not resend an unresolved submission")
}

func productActor() ActionActor {
	return ActionActor{ID: 1, DataScope: &common.DataScopeReq{UserID: 1, IsAdmin: true, Mode: common.DataScopeModeAll}}
}

func TestProductReviewRequiresFreshAuthorityAndCredentialState(t *testing.T) {
	for _, change := range []string{"revoked-actor", "revoked-cluster", "missing-authority", "revoked-source", "source-credential", "server-credential", "other-actor", "tampered-review", "expired-review"} {
		t.Run(change, func(t *testing.T) {
			var mutations atomic.Int32
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
			actor := productActor()
			preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
			require.NoError(t, err)
			actor.IntentVersion = preview.IntentVersion
			switch change {
			case "revoked-actor", "revoked-cluster":
				// The request still says IsAdmin/all. Only the fresh authority
				// owns the revocation decision; its raw error must stay private.
				s.authorizer = fixtureAuthority(func(context.Context, ActionAuthorization) error { return errors.New("policy-password=private-fixture") })
			case "missing-authority":
				s.authorizer = nil
			case "revoked-source":
				require.NoError(t, s.db.Where("server_id = ?", 1).Delete(&ArgoCDRepository{}).Error)
			case "source-credential":
				require.NoError(t, s.db.Model(&ArgoCDRepository{}).Where("server_id = ?", 1).Update("password", "rotated-reference").Error)
			case "server-credential":
				value, err := EncryptSecret("rotated-fixture")
				require.NoError(t, err)
				require.NoError(t, s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Update("auth_token", value).Error)
			case "other-actor":
				actor.ID = 2
				actor.DataScope = &common.DataScopeReq{UserID: 2, IsAdmin: true}
			case "tampered-review":
				actor.IntentVersion += "x"
			case "expired-review":
				plain, err := DecryptSecret(actor.IntentVersion)
				require.NoError(t, err)
				var review applicationReview
				require.NoError(t, json.Unmarshal([]byte(plain), &review))
				review.Expires = time.Now().Add(-time.Second).Unix()
				data, _ := json.Marshal(review)
				actor.IntentVersion, err = EncryptSecret(string(data))
				require.NoError(t, err)
			}
			_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
			require.Error(t, err)
			require.NotContains(t, err.Error(), "private-fixture")
			require.Zero(t, mutations.Load())
		})
	}
}

func TestProductDiffHidesEveryManifestValueAndBindsPrivateDrift(t *testing.T) {
	live := []string{`{"apiVersion":"v1","kind":"Secret","metadata":{"name":"safe-name","namespace":"test","annotations":{"private":"private-fixture"}},"data":{"password":"private-fixture"}}`}
	target := []string{strings.ReplaceAll(live[0], `"password":"private-fixture"`, `"password":"changed-fixture"`)}
	changes, digest, err := applicationDiff(live, target)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "modified", changes[0].Change)
	data, err := json.Marshal(changes)
	require.NoError(t, err)
	require.NotContains(t, string(data), "private-fixture")
	require.NotContains(t, string(data), "changed-fixture")
	require.NotContains(t, string(data), "password")
	_, other, err := applicationDiff(live, live)
	require.NoError(t, err)
	require.NotEqual(t, digest, other)
	_, _, err = applicationDiff([]string{strings.ReplaceAll(live[0], "safe-name", "password=private-fixture")}, nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "private-fixture")
	_, _, err = applicationDiff(make([]string, 201), nil)
	require.Error(t, err)
	_, _, err = applicationDiff(nil, []string{live[0], live[0]})
	require.Error(t, err)
}

func TestProductRestartResolvesOnlyExactDurableAction(t *testing.T) {
	for _, change := range []string{"exact", "other-marker", "old-uid", "other-revision", "missing-finish", "duplicate-marker", "invalid-phase"} {
		t.Run(change, func(t *testing.T) {
			s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { writeFixtureAcceptedOperation(t, w, r) })
			response, err := submitFixtureAction(s, context.Background(), id, "main", false)
			require.NoError(t, err)
			var history ArgoCDSyncHistory
			require.NoError(t, s.db.First(&history, response.ID).Error)
			metadata, ok := decodeActionMetadata(history.Message)
			require.True(t, ok)
			var status ArgoCDApplicationStatus
			require.NoError(t, json.Unmarshal([]byte(`{"sync":{"status":"Synced"},"health":{"status":"Healthy"},"operationState":{"phase":"Succeeded","startedAt":"2026-11-09T00:00:00Z","finishedAt":"2026-11-09T00:00:12Z","operation":{"sync":{}},"syncResult":{}}}`), &status))
			status.ApplicationUID = metadata.UID
			status.OperationState.Operation.Info = []argoOperationInfo{{Name: actionInfoName, Value: metadata.Nonce}}
			status.OperationState.Operation.Sync.Revision = metadata.Revision
			status.OperationState.SyncResult.Revision = metadata.Revision
			switch change {
			case "other-marker":
				status.OperationState.Operation.Info[0].Value = "unrelated"
			case "old-uid":
				status.ApplicationUID = "replacement"
			case "other-revision":
				status.OperationState.SyncResult.Revision = fixtureOldRevision
			case "missing-finish":
				status.OperationState.FinishedAt = nil
			case "duplicate-marker":
				status.OperationState.Operation.Info = append(status.OperationState.Operation.Info, status.OperationState.Operation.Info[0])
			case "invalid-phase":
				status.OperationState.Phase = "password=private-fixture"
			}
			var app ArgoCDApplication
			require.NoError(t, s.db.First(&app, id).Error)
			// A fresh reconciler has no in-memory submission state.
			require.NoError(t, applyApplicationStatus(context.Background(), NewArgoCDService(s.db).db, &app, &status))
			require.NoError(t, s.db.First(&history, response.ID).Error)
			if change == "exact" {
				require.Equal(t, "Succeeded", history.Status)
				require.NotNil(t, history.FinishedAt)
				require.NoError(t, applyApplicationStatus(context.Background(), s.db, &app, &status))
				var count int64
				require.NoError(t, s.db.Model(&ArgoCDSyncHistory{}).Where("application_id = ?", id).Count(&count).Error)
				require.Equal(t, int64(1), count)
			} else {
				require.Equal(t, "Running", history.Status)
				require.Nil(t, history.FinishedAt)
			}
		})
	}
}

func TestProductHTTPDenialIsSanitizedAndAudited(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { t.Error("denied HTTP request reached provider") })
	s.authorizer = fixtureAuthority(func(context.Context, ActionAuthorization) error {
		return fmt.Errorf("%w: policy-password=private-fixture", common.ErrForbidden)
	})
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var auditParam, auditResult, auditError interface{}
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint64(1))
		c.Set(common.DataScopeContextKey, productActor().DataScope)
		c.Next()
		auditParam, _ = c.Get(common.OperationLogParamKey)
		auditResult, _ = c.Get(common.OperationLogResultKey)
		auditError, _ = c.Get(common.OperationLogErrorMsgKey)
	})
	NewArgoCDHandler(s).RegisterRoutes(router)
	request := httptest.NewRequest(http.MethodPost, "/cicd/argocd/applications/"+fmt.Sprint(id)+"/sync", strings.NewReader(`{"revision":"main","intentVersion":"private-fixture","password":"private-fixture"}`))
	request.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, request)
	require.NotContains(t, w.Body.String(), "private-fixture")
	require.Contains(t, w.Body.String(), "common.forbidden")
	require.Equal(t, "{}", auditParam)
	require.Equal(t, "{}", auditResult)
	require.Equal(t, "common.forbidden", auditError)
}

func TestProductConfigRevokedAfterReservationCannotDispatch(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
	require.NoError(t, s.db.Callback().Create().After("gorm:create").Register("argo-product-revoke-source", func(tx *gorm.DB) {
		if tx.Statement.Table == (ArgoCDSyncHistory{}).TableName() {
			err := tx.Session(&gorm.Session{NewDB: true}).Model(&ArgoCDRepository{}).Where("server_id = ?", 1).Update("status", "disconnected").Error
			if err != nil {
				tx.AddError(err)
			}
		}
	}))
	t.Cleanup(func() { _ = s.db.Callback().Create().Remove("argo-product-revoke-source") })
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.Error(t, err)
	require.Zero(t, mutations.Load(), "credentials/config revoked after recording intent must not reach provider")
}

func TestProductDisconnectedServerIsReprobedAfterRestart(t *testing.T) {
	var reads atomic.Int32
	s, _ := actionService(t, func(w http.ResponseWriter, r *http.Request) { reads.Add(1); writeFixtureAcceptedOperation(t, w, r) })
	require.NoError(t, s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Update("status", ArgoCDStatusDisconnected).Error)
	r := NewReconciler(s.db, time.Second)
	r.clientFactory = newTestArgoCDClient
	require.NoError(t, r.reconcile(context.Background()))
	var server ArgoCDServer
	require.NoError(t, s.db.First(&server, 1).Error)
	require.Positive(t, reads.Load(), "disconnected servers must be retried so uncertain actions can recover")
	require.Equal(t, ArgoCDStatusConnected, server.Status)
	require.NotNil(t, server.LastChecked)
}

func TestProductProviderPatchConflictRequiresNewReview(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("provider-password=private-fixture"))
	})
	_, err := submitFixtureAction(s, context.Background(), id, "main", false)
	require.ErrorIs(t, err, common.ErrConflict)
	require.NotContains(t, err.Error(), "private-fixture")
	var history ArgoCDSyncHistory
	require.NoError(t, s.db.Where("application_id = ?", id).First(&history).Error)
	require.Equal(t, "Failed", history.Status)
	require.NotNil(t, history.FinishedAt)
}

func TestProductHTTPPreviewSubmitHistoryAndMalformedIntent(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint64(1))
		c.Set(common.DataScopeContextKey, productActor().DataScope)
	})
	NewArgoCDHandler(s).RegisterRoutes(router)
	call := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	path := "/cicd/argocd/applications/" + fmt.Sprint(id)
	for _, body := range []string{`{"revision":"main"}`, `{"revision":`, ``} {
		response := call(http.MethodPost, path+"/sync", body)
		require.Contains(t, response.Body.String(), msgParamInvalid)
	}
	require.Zero(t, mutations.Load(), "missing/malformed intent must not dispatch")
	response := call(http.MethodPost, path+"/preview", `{"revision":"main"}`)
	var preview struct {
		Code int                        `json:"code"`
		Data ApplicationPreviewResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &preview))
	require.Equal(t, 200, preview.Code)
	require.NotEmpty(t, preview.Data.IntentVersion)
	body, _ := json.Marshal(SyncApplicationRequest{Revision: "main", IntentVersion: preview.Data.IntentVersion})
	response = call(http.MethodPost, path+"/sync", string(body))
	var submitted struct {
		Code int                 `json:"code"`
		Data SyncHistoryResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &submitted))
	require.Equal(t, 200, submitted.Code)
	require.Equal(t, "Running", submitted.Data.Status)
	require.Nil(t, submitted.Data.FinishedAt)
	require.Equal(t, int32(1), mutations.Load())
	response = call(http.MethodGet, path+"/history", ``)
	require.NotContains(t, response.Body.String(), actionMetadataPrefix)
	require.NotContains(t, response.Body.String(), actionInfoName)
	require.Contains(t, response.Body.String(), `"status":"Running"`)
}
