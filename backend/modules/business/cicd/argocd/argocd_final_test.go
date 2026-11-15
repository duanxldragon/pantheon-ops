package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func finalRouter(s *ArgoCDService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userID", uint64(1))
		c.Set(common.DataScopeContextKey, productActor().DataScope)
	})
	NewArgoCDHandler(s).RegisterRoutes(r)
	return r
}

func finalCall(t *testing.T, r *gin.Engine, ctx context.Context, method, path, body string) (int, string) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, request)
	var envelope struct {
		Code int `json:"code"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	return envelope.Code, w.Body.String()
}

func TestArgoFinalPUTAuthorizesCurrentAndProposed(t *testing.T) {
	for _, denied := range []string{"current-source", "new-source", "current-destination", "new-destination", "current-namespace", "new-namespace", "cluster", "allowed", "empty"} {
		t.Run(denied, func(t *testing.T) {
			var provider atomic.Int32
			s, id := actionService(t, func(http.ResponseWriter, *http.Request) { provider.Add(1) })
			var before ArgoCDApplication
			require.NoError(t, s.db.First(&before, id).Error)
			seenCurrent, seenNew := false, false
			s.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
				require.Equal(t, before.ServerID, req.ServerID, "DTO cannot change owning server")
				require.Equal(t, uint64(1), req.ClusterID, "cluster comes from registered server, not request")
				seenCurrent = seenCurrent || (req.Namespace == before.DestinationNS && req.DestinationServer == before.DestinationServer && len(req.SourceURLs) == 1 && req.SourceURLs[0] == before.SourceRepoURL)
				seenNew = seenNew || (req.Namespace == "new-ns" && req.DestinationServer == "https://new.example.invalid" && len(req.SourceURLs) == 1 && req.SourceURLs[0] == "https://new.example.invalid/repo.git")
				for _, source := range req.SourceURLs {
					if (denied == "current-source" && source == before.SourceRepoURL) || (denied == "new-source" && source == "https://new.example.invalid/repo.git") {
						return common.NewForbidden("common.forbidden")
					}
				}
				if denied == "cluster" || (denied == "current-namespace" && req.Namespace == before.DestinationNS) || (denied == "new-namespace" && req.Namespace == "new-ns") || (denied == "current-destination" && req.DestinationServer == before.DestinationServer) || (denied == "new-destination" && req.DestinationServer == "https://new.example.invalid") {
					return common.NewForbidden("common.forbidden")
				}
				return nil
			})
			body := `{"serverId":999,"clusterId":999,"sourceRepoUrl":"https://new.example.invalid/repo.git","sourcePath":"next","sourceTargetRev":"next","destinationServer":"https://new.example.invalid","destinationNamespace":"new-ns","project":"next"}`
			if denied == "empty" {
				body = `{"sourceRepoUrl":"","sourcePath":"","destinationServer":"","destinationNamespace":""}`
			}
			code, _ := finalCall(t, finalRouter(s), context.Background(), http.MethodPut, fmt.Sprintf("/cicd/argocd/applications/%d", id), body)
			var after ArgoCDApplication
			require.NoError(t, s.db.First(&after, id).Error)
			if denied == "allowed" {
				require.Equal(t, common.CodeSuccess, code)
				require.True(t, seenCurrent)
				require.True(t, seenNew)
				require.Equal(t, "new-ns", after.DestinationNS)
				require.Equal(t, "next", after.SourcePath)
				require.Equal(t, "next", after.SourceTargetRev)
				require.Equal(t, "next", after.Project)
				require.Equal(t, "https://new.example.invalid/repo.git", after.SourceRepoURL)
				require.Equal(t, "https://new.example.invalid", after.DestinationServer)
			} else {
				if denied == "empty" {
					require.Equal(t, common.CodeSuccess, code)
				} else {
					require.Equal(t, common.CodeForbidden, code)
				}
				require.Equal(t, before, after, "denial/omitted or empty fields must not change stored configuration")
			}
			require.Zero(t, provider.Load())
		})
	}
}

type finalTransport func(*http.Request) (*http.Response, error)

func (f finalTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestArgoFinalGroupKindCollision(t *testing.T) {
	for _, kind := range []string{"Role", "Deployment", "Service"} {
		t.Run(kind, func(t *testing.T) {
			for _, ns := range []string{"", "test"} {
				manifest := fmt.Sprintf(`{"apiVersion":"hostile.example/v1","kind":%q,"metadata":{"name":"test","namespace":%q}}`, kind, ns)
				s, id := actionService(t, func(http.ResponseWriter, *http.Request) { t.Error("unsupported resource reached mutation") })
				factory := s.clientFactory
				s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
					c, err := factory(endpoint, token, ca)
					if err != nil {
						return nil, err
					}
					base := c.httpClient.Transport
					c.httpClient.Transport = finalTransport(func(r *http.Request) (*http.Response, error) {
						if strings.HasSuffix(r.URL.Path, "/manifests") {
							data, _ := json.Marshal(map[string]interface{}{"revision": fixtureRevision, "manifests": []string{manifest}})
							return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(data)), Header: make(http.Header), Request: r}, nil
						}
						return base.RoundTrip(r)
					})
					return c, nil
				}
				_, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, productActor())
				require.Error(t, err, "unknown group must not borrow a standard Kind's permissions")
				if ns == "" {
					_, _, err = applicationDiff(nil, []string{manifest}, "test")
					require.Error(t, err, "unknown group must not inherit destination namespace")
				}
			}
		})
	}
	for _, resource := range []struct{ api, kind string }{{"v1", "Service"}, {"apps/v1", "Deployment"}, {"rbac.authorization.k8s.io/v1", "Role"}, {"batch/v1", "Job"}, {"networking.k8s.io/v1", "Ingress"}} {
		manifest := fmt.Sprintf(`{"apiVersion":%q,"kind":%q,"metadata":{"name":"test"}}`, resource.api, resource.kind)
		changes, _, err := applicationDiff(nil, []string{manifest}, "test")
		require.NoError(t, err)
		require.Equal(t, "test", changes[0].Namespace)
	}
}

func TestArgoFinalPollingDoesNotInvalidateReview(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
	require.NoError(t, err)
	actor.IntentVersion = preview.IntentVersion
	now := time.Now()
	require.NoError(t, s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Updates(map[string]interface{}{"last_checked": now, "status": "connected", "updated_at": now}).Error)
	require.NoError(t, s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Updates(map[string]interface{}{"status": "Running", "sync_status": "Synced", "health_status": "Healthy", "last_synced_at": now, "last_error": "", "updated_at": now}).Error)
	require.NoError(t, s.db.Model(&ArgoCDRepository{}).Where("server_id = ?", 1).Updates(map[string]interface{}{"status": "unknown", "updated_at": now}).Error)
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.NoError(t, err)
	require.Equal(t, int32(1), mutations.Load())
}

func TestArgoFinalListFaultsNeverBecomeSuccessfulPartialLists(t *testing.T) {
	for _, collection := range []string{"servers", "repositories", "applications"} {
		for _, fault := range []string{"query", "authority", "cancel", "denial", "notfound"} {
			t.Run(collection+"/"+fault, func(t *testing.T) {
				s, _ := actionService(t, func(http.ResponseWriter, *http.Request) { t.Error("list reached provider") })
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				s.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
					if req.Action != "read" {
						return nil
					}
					switch fault {
					case "authority":
						return errors.New("private-fixture infrastructure failure")
					case "cancel":
						cancel()
						return context.Canceled
					case "denial":
						return common.NewForbidden("common.forbidden")
					case "notfound":
						return common.NewNotFound("common.forbidden")
					}
					return nil
				})
				require.NoError(t, s.db.Callback().Query().Before("gorm:query").Register("argo-final-list-fault", func(tx *gorm.DB) {
					if _, ok := tx.Statement.Dest.(*ArgoCDServer); ok && fault == "query" {
						tx.AddError(errors.New("private-fixture database failure"))
					}
				}))
				t.Cleanup(func() { _ = s.db.Callback().Query().Remove("argo-final-list-fault") })
				code, body := finalCall(t, finalRouter(s), ctx, http.MethodGet, "/cicd/argocd/"+collection, "")
				require.NotContains(t, body, "private-fixture")
				if fault == "denial" || fault == "notfound" {
					require.Equal(t, common.CodeSuccess, code)
					require.Contains(t, body, `"total":0`)
				} else {
					require.Equal(t, common.CodeError, code, "infrastructure/cancel is a failure, not filtered denial")
				}
			})
		}
	}
}

func TestArgoFinalExactResultRecoveryAfterExpiryAndProviderDrift(t *testing.T) {
	var mutations, reads atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, _ *http.Request) { mutations.Add(1); w.WriteHeader(500) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
	require.NoError(t, err)
	plain, err := DecryptSecret(preview.IntentVersion)
	require.NoError(t, err)
	var review applicationReview
	require.NoError(t, json.Unmarshal([]byte(plain), &review))
	review.Expires = time.Now().Add(2 * time.Second).Unix()
	data, _ := json.Marshal(review)
	actor.IntentVersion, err = EncryptSecret(string(data))
	require.NoError(t, err)
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.Error(t, err)
	var original ArgoCDSyncHistory
	require.NoError(t, s.db.Where("application_id = ?", id).First(&original).Error)
	// More than the public latest-50 window must not hide the owned exact result.
	for i := 0; i < 51; i++ {
		require.NoError(t, s.db.Create(&ArgoCDSyncHistory{ApplicationID: id, Revision: fixtureOldRevision, Status: "Succeeded", StartedAt: time.Now()}).Error)
	}
	s.clientFactory = func(string, string, string) (*ArgoCDClient, error) {
		reads.Add(1)
		return nil, errors.New("provider drift must never be read by recovery")
	}
	time.Sleep(time.Until(time.Unix(review.Expires+1, 0)))
	for _, status := range []string{AppStatusUnknown, "Running", "Succeeded", "Failed"} {
		updates := map[string]interface{}{"status": status, "finished_at": nil}
		if status == "Succeeded" || status == "Failed" {
			updates["finished_at"] = time.Now()
		}
		require.NoError(t, s.db.Model(&original).Updates(updates).Error)
		result, err := s.SyncApplicationContext(context.Background(), id, "main", actor)
		require.NoError(t, err, "same authorized actor must recover owned result despite expiry/provider drift")
		require.Equal(t, original.ID, result.ID)
		require.Equal(t, status, result.Status)
		require.NotContains(t, result.Message, actionMetadataPrefix)
	}
	_, err = s.SyncApplicationContext(context.Background(), id, fixtureOldRevision, actor)
	require.ErrorIs(t, err, common.ErrConflict, "ticket for another target cannot return a misleading result")
	foreign := actor
	foreign.ID = 2
	foreign.DataScope = &common.DataScopeReq{UserID: 2, IsAdmin: true}
	_, err = s.SyncApplicationContext(context.Background(), id, "main", foreign)
	require.ErrorIs(t, err, common.ErrForbidden)
	currentAuthority := s.authorizer
	s.authorizer = fixtureAuthority(func(context.Context, ActionAuthorization) error { return common.NewForbidden("common.forbidden") })
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.ErrorIs(t, err, common.ErrForbidden, "actor revocation denies prior result reads")
	s.authorizer = nil
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.ErrorIs(t, err, common.ErrForbidden, "missing current authority never releases prior results")
	s.authorizer = currentAuthority
	require.NoError(t, s.db.Where("server_id = ?", 1).Delete(&ArgoCDRepository{}).Error)
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.ErrorIs(t, err, common.ErrForbidden, "source revocation denies result recovery")
	require.Zero(t, reads.Load())
	require.Equal(t, int32(1), mutations.Load())
}

func TestArgoFinalExecutionConfigChangesInvalidateReview(t *testing.T) {
	for _, field := range []string{"destination", "namespace", "source", "path", "revision", "project", "server-version", "cluster", "credential", "repo-username", "repo-branch", "repo-identity"} {
		t.Run(field, func(t *testing.T) {
			var mutations atomic.Int32
			s, id := actionService(t, func(http.ResponseWriter, *http.Request) { mutations.Add(1) })
			actor := productActor()
			preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
			require.NoError(t, err)
			actor.IntentVersion = preview.IntentVersion
			var update *gorm.DB
			switch field {
			case "destination":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("destination_server", "https://changed.example.invalid")
			case "namespace":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("destination_ns", "changed")
			case "source":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("source_repo_url", "https://changed.example.invalid/repo.git")
			case "path":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("source_path", "changed")
			case "revision":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("source_target_rev", fixtureOldRevision)
			case "project":
				update = s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Update("project", "changed")
			case "server-version":
				update = s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Update("version", "changed")
			case "cluster":
				update = s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Update("cluster_id", 2)
			case "credential":
				value, err := EncryptSecret("rotated-fixture")
				require.NoError(t, err)
				update = s.db.Model(&ArgoCDServer{}).Where("id = ?", 1).Update("auth_token", value)
			case "repo-username":
				update = s.db.Model(&ArgoCDRepository{}).Where("id = ?", 1).Update("username", "changed")
			case "repo-branch":
				update = s.db.Model(&ArgoCDRepository{}).Where("id = ?", 1).Update("branch", "changed")
			case "repo-identity":
				var repo ArgoCDRepository
				require.NoError(t, s.db.First(&repo, 1).Error)
				require.NoError(t, s.db.Delete(&repo).Error)
				repo.ID = 0
				repo.DeletedAt = gorm.DeletedAt{}
				update = s.db.Create(&repo)
			}
			require.NoError(t, update.Error)
			_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
			require.ErrorIs(t, err, common.ErrConflict)
			require.Zero(t, mutations.Load())
			var count int64
			require.NoError(t, s.db.Model(&ArgoCDSyncHistory{}).Where("application_id = ?", id).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestArgoFinalPriorSourceAuthorizationSurvivesLocalConfigDrift(t *testing.T) {
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { writeFixtureAcceptedOperation(t, w, r) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
	require.NoError(t, err)
	actor.IntentVersion = preview.IntentVersion
	original, err := s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.NoError(t, err)
	newSource := "https://changed.example.invalid/repo.git"
	require.NoError(t, s.db.Create(&ArgoCDRepository{ServerID: 1, RepoURL: newSource, Status: "connected"}).Error)
	require.NoError(t, s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Updates(map[string]interface{}{"source_repo_url": newSource, "destination_ns": "changed"}).Error)
	seenCurrent, seenOriginal := false, false
	s.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
		for _, source := range req.SourceURLs {
			seenCurrent = seenCurrent || (source == newSource && req.Namespace == "changed")
			seenOriginal = seenOriginal || (source != newSource && req.Namespace == "test")
		}
		return nil
	})
	s.clientFactory = func(string, string, string) (*ArgoCDClient, error) {
		t.Error("recovery reached provider factory")
		return nil, errors.New("unreachable")
	}
	result, err := s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.NoError(t, err)
	require.Equal(t, original.ID, result.ID)
	require.True(t, seenCurrent)
	require.True(t, seenOriginal)
	s.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
		if req.Namespace == "test" {
			return common.NewForbidden("common.forbidden")
		}
		return nil
	})
	_, err = s.SyncApplicationContext(context.Background(), id, "main", actor)
	require.ErrorIs(t, err, common.ErrForbidden, "original reviewed target revocation must deny recovery even when new target is allowed")
}

func TestArgoFinalActorProvenanceNeverSerializes(t *testing.T) {
	actor := productActor()
	actor.SessionID = "private-fixture-session"
	actor.IntentVersion = "private-fixture-ticket"
	data, err := json.Marshal(actor)
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(data))
}

func TestArgoFinalLegacyWebhookWrapperFailsClosed(t *testing.T) {
	var calls atomic.Int32
	s, id := actionService(t, func(http.ResponseWriter, *http.Request) { calls.Add(1) })
	s.clientFactory = func(string, string, string) (*ArgoCDClient, error) {
		calls.Add(1)
		return nil, errors.New("unreachable")
	}
	_, err := s.SyncApplication(id, "main")
	require.Error(t, err)
	require.Zero(t, calls.Load(), "legacy no-actor/no-review webhook must not silently bypass the new protocol")
	var count int64
	require.NoError(t, s.db.Model(&ArgoCDSyncHistory{}).Where("application_id = ?", id).Count(&count).Error)
	require.Zero(t, count)
}

func TestArgoFinalConcurrentExactRetryNeverRedispatches(t *testing.T) {
	var mutations atomic.Int32
	s, id := actionService(t, func(w http.ResponseWriter, r *http.Request) { mutations.Add(1); writeFixtureAcceptedOperation(t, w, r) })
	actor := productActor()
	preview, err := s.PreviewApplicationContext(context.Background(), id, ApplicationPreviewRequest{Revision: "main"}, actor)
	require.NoError(t, err)
	actor.IntentVersion = preview.IntentVersion
	var wg sync.WaitGroup
	results := make(chan *SyncHistoryResponse, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := s.SyncApplicationContext(context.Background(), id, "main", actor)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var historyID uint64
	for result := range results {
		require.NotNil(t, result)
		if historyID == 0 {
			historyID = result.ID
		}
		require.Equal(t, historyID, result.ID)
	}
	require.Equal(t, int32(1), mutations.Load())
	var count int64
	require.NoError(t, s.db.Model(&ArgoCDSyncHistory{}).Where("application_id = ?", id).Count(&count).Error)
	require.Equal(t, int64(1), count)
}
