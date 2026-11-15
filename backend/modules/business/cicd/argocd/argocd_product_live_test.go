package argocd

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/logger"
)

// TestArgoProductLive is opt-in. Its HTTP transport is a fixture-only relay to
// the authorized isolated Argo namespace via pinned SSH and verified provider
// TLS. NewArgoCDClient's public SSRF/TLS contract is exercised separately and
// never relaxed by product source. No token/CA/private key enters test output.
func TestArgoProductLive(t *testing.T) {
	value := os.Getenv("PANTHEON_ARGO_PRODUCT_FIXTURE")
	if value == "" {
		t.Skip("requires root-run argo-product-live.py finite provider fixture")
	}
	var fixture struct{ Endpoint, Repo, Namespace, Project, Application, OldRevision, NewRevision, AuthToken string }
	require.NoError(t, json.Unmarshal([]byte(value), &fixture))
	require.True(t, strings.HasPrefix(fixture.Endpoint, "http://127.0.0.1:"))
	db := testmysql.Open(t)
	db.Logger = logger.Default.LogMode(logger.Silent)
	s := NewArgoCDService(db)
	require.NoError(t, s.Migrate())
	t.Setenv(encryptionKeyEnv, "12345678901234567890123456789012")
	encryptedToken, err := EncryptSecret(fixture.AuthToken)
	require.NoError(t, err)
	server := ArgoCDServer{Name: "owned-product-fixture", Endpoint: fixture.Endpoint, ClusterID: 1, AuthToken: encryptedToken}
	require.NoError(t, db.Create(&server).Error)
	app := ArgoCDApplication{ServerID: server.ID, Name: fixture.Application, Project: fixture.Project, SourceRepoURL: fixture.Repo, SourcePath: ".", SourceTargetRev: fixture.OldRevision, DestinationServer: "https://kubernetes.default.svc", DestinationNS: fixture.Namespace, CreatedBy: "1"}
	require.NoError(t, db.Create(&app).Error)
	require.NoError(t, db.Create(&ArgoCDRepository{ServerID: server.ID, RepoURL: fixture.Repo, Status: "connected"}).Error)
	var actorRevoked atomic.Bool
	s.authorizer = fixtureAuthority(func(_ context.Context, req ActionAuthorization) error {
		if actorRevoked.Load() || req.Actor.ID != 1 || req.Actor.DataScope == nil || req.Actor.DataScope.UserID != 1 || req.ServerID != server.ID || req.ClusterID != 1 || req.ApplicationID != app.ID || req.Namespace != fixture.Namespace || len(req.SourceURLs) != 1 || req.SourceURLs[0] != fixture.Repo {
			return common.NewForbidden("common.forbidden")
		}
		return nil
	})
	s.clientFactory = func(endpoint, token, ca string) (*ArgoCDClient, error) {
		return newTestArgoCDClient(endpoint, token, ca)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	check := func(name string, ok bool) { t.Helper(); t.Log("PRODUCT_CHECK " + name); require.True(t, ok) }
	preview := func(revision string, rollback bool) (*ApplicationPreviewResponse, ActionActor) {
		t.Helper()
		actor := productActor()
		response, err := s.PreviewApplicationContext(ctx, app.ID, ApplicationPreviewRequest{Revision: revision, Rollback: rollback}, actor)
		require.NoError(t, err)
		actor.IntentVersion = response.IntentVersion
		return response, actor
	}
	wait := func(historyID uint64, revision string) {
		t.Helper()
		deadline := time.Now().Add(75 * time.Second)
		for time.Now().Before(deadline) {
			token, err := DecryptSecret(server.AuthToken)
			require.NoError(t, err)
			client, err := s.clientFactory(server.Endpoint, token, "")
			require.NoError(t, err)
			var current ArgoCDApplication
			require.NoError(t, db.First(&current, app.ID).Error)
			// A new reconciler instance proves there is no in-memory action map.
			require.NoError(t, NewReconciler(db, time.Second).reconcileApplication(ctx, client, &current))
			var history ArgoCDSyncHistory
			require.NoError(t, db.First(&history, historyID).Error)
			if history.FinishedAt != nil {
				require.Equal(t, "Succeeded", history.Status)
				require.Equal(t, revision, history.Revision)
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("bounded provider wait cancelled")
			case <-time.After(time.Second):
			}
		}
		t.Fatal("exact product action did not reach terminal within bounded window")
	}
	state := func() string {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fixture.Endpoint+"/fixture/service", nil)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer response.Body.Close()
		var result struct {
			Revision string `json:"revision"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
		return result.Revision
	}
	first, actor := preview(fixture.OldRevision, false)
	check("real provider preview contains added Service and no literal values", len(first.Changes) == 1 && first.Changes[0].Kind == "Service" && first.Changes[0].Change == "added" && first.Changes[0].ValuesHidden)
	result, err := s.SyncApplicationContext(ctx, app.ID, fixture.OldRevision, actor)
	require.NoError(t, err)
	check("product persists acceptance separately from terminal", result.Status == "Running" && result.FinishedAt == nil)
	wait(result.ID, fixture.OldRevision)
	check("product exact terminal first revision matches actual Service", state() == "v1")
	second, actor := preview(fixture.NewRevision, false)
	check("real provider diff identifies routing change without selector literal", len(second.Changes) == 1 && second.Changes[0].Change == "modified" && second.Changes[0].ValuesHidden)
	_, err = s.SyncApplicationContext(ctx, app.ID, fixture.NewRevision, ActionActor{ID: 1, DataScope: actor.DataScope, IntentVersion: first.IntentVersion})
	check("previous review cannot execute another target", errors.Is(err, common.ErrConflict))
	result, err = s.SyncApplicationContext(ctx, app.ID, fixture.NewRevision, actor)
	require.NoError(t, err)
	wait(result.ID, fixture.NewRevision)
	check("product second terminal matches actual Service", state() == "v2")
	rollback, actor := preview(fixture.OldRevision, true)
	check("real history rollback preview is reviewed", rollback.RollbackAvailable && rollback.TargetVersion == fixture.OldRevision)
	result, err = s.RollbackApplicationContext(ctx, app.ID, fixture.OldRevision, actor)
	require.NoError(t, err)
	wait(result.ID, fixture.OldRevision)
	check("product historical source rollback restores actual Service", state() == "v1")
	_, actor = preview(fixture.NewRevision, false)
	actorRevoked.Store(true)
	_, err = s.SyncApplicationContext(ctx, app.ID, fixture.NewRevision, actor)
	check("current authority revocation overrides stale caller IsAdmin/all", errors.Is(err, common.ErrForbidden) && state() == "v1")
	actorRevoked.Store(false)
	_, actor = preview(fixture.NewRevision, false)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fixture.Endpoint+"/fixture/uncertain-next", nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	_, err = s.SyncApplicationContext(ctx, app.ID, fixture.NewRevision, actor)
	require.Error(t, err)
	var pending ArgoCDSyncHistory
	require.NoError(t, db.Where("application_id = ?", app.ID).Order("id DESC").First(&pending).Error)
	check("actual provider acceptance plus lost response remains durable Unknown", pending.Status == AppStatusUnknown && pending.FinishedAt == nil)
	// Retry/restart may read, but cannot send another native operation.
	_, err = s.SyncApplicationContext(ctx, app.ID, fixture.NewRevision, actor)
	require.Error(t, err)
	wait(pending.ID, fixture.NewRevision)
	check("new reconciler recovers original unknown action to exact terminal", state() == "v2")
	history, err := s.GetSyncHistory(app.ID)
	require.NoError(t, err)
	data, err := json.Marshal(history)
	require.NoError(t, err)
	check("history has one row per submitted action and no private marker", len(history) == 4 && !strings.Contains(string(data), actionMetadataPrefix) && !strings.Contains(string(data), actionInfoName))
	// The relay returns sanitized counters for independent no-resend evidence.
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fixture.Endpoint+"/fixture/counters", nil)
	require.NoError(t, err)
	response, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	var counters struct {
		Mutations int `json:"mutations"`
	}
	require.NoError(t, json.Unmarshal(raw, &counters))
	check("provider received exactly four mutations including uncertain submission", counters.Mutations == 4)
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, fixture.Endpoint+"/fixture/revoke-token", nil)
	require.NoError(t, err)
	response, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	_ = response.Body.Close()
	require.Equal(t, 200, response.StatusCode)
	_, err = s.PreviewApplicationContext(ctx, app.ID, ApplicationPreviewRequest{Revision: fixture.OldRevision}, productActor())
	check("real provider credential revocation prevents product preview and mutation", err != nil)
	t.Log("PRODUCT_SCOPE real ArgoCDService/preview/action/reconciler + real provider; current identity authority is an explicit test fixture; full Base auth/HTTP/audit/public origin are not covered")
}
