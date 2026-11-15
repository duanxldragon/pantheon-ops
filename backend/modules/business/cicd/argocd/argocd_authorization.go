package argocd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"
)

// ActionActor carries the authenticated subject and the request's scope. A
// caller-supplied scope is never itself an authorization decision.
type ActionActor struct {
	ID            uint64               `json:"-"`
	DataScope     *common.DataScopeReq `json:"-"`
	IntentVersion string               `json:"-"`
	// SessionID must come from Base's verified originating session, never a
	// request body/header or a fabricated machine identity. Wiring is pending.
	SessionID string `json:"-"`
}

// ActionAuthorization names the exact current resources and proposed source.
// The composition root must supply an authority backed by current identity,
// permissions, cluster/namespace scope and source access, including revocations.
type ActionAuthorization struct {
	Actor                              ActionActor
	Action                             string
	ServerID, ClusterID, ApplicationID uint64
	DestinationServer, Namespace       string
	SourceURLs                         []string
}

// ActionAuthorizer is an integration port, not a system-domain implementation.
// It must re-resolve current identity/DataScope; trusting Actor.DataScope or
// cached roles is insufficient. Missing wiring fails closed.
type ActionAuthorizer interface {
	AuthorizeArgoAction(context.Context, ActionAuthorization) error
}

// SetActionAuthorizer supplies the composition-root authority before serving.
func (s *ArgoCDService) SetActionAuthorizer(authorizer ActionAuthorizer) {
	s.authorizer = authorizer
}

func (s *ArgoCDService) authorizeAction(ctx context.Context, actor ActionActor, action string, app *ArgoCDApplication, server *ArgoCDServer, sourceURLs []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if actor.ID == 0 || actor.DataScope == nil || actor.DataScope.UserID != actor.ID || s.authorizer == nil {
		return common.NewForbidden("common.forbidden")
	}
	if err := s.authorizer.AuthorizeArgoAction(ctx, ActionAuthorization{
		Actor: actor, Action: action, ServerID: server.ID, ClusterID: server.ClusterID,
		ApplicationID: app.ID, DestinationServer: app.DestinationServer, Namespace: app.DestinationNS, SourceURLs: sourceURLs,
	}); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		// Only a typed policy decision proves denial. Infrastructure errors
		// must fail the request, rather than quietly filtering collections.
		if errors.Is(err, common.ErrForbidden) {
			return common.NewForbidden("common.forbidden")
		}
		if errors.Is(err, common.ErrNotFound) {
			return common.NewNotFound(common.MessageRequestFailed)
		}
		logging.Error("Argo CD authority unavailable", zap.String("error_type", fmt.Sprintf("%T", err)))
		return common.NewInternal(common.MessageRequestFailed)
	}
	return ctx.Err()
}

func actionActor(actors []ActionActor) ActionActor {
	if len(actors) != 1 {
		return ActionActor{}
	}
	return actors[0]
}

func actionFingerprint(app *ArgoCDApplication, server *ArgoCDServer, repositories []ArgoCDRepository) string {
	// Hash execution configuration/identity/credentials explicitly. Polling
	// status and timestamps are observations, not a changed execution contract.
	type appConfig struct {
		ID, ServerID                                                       uint64
		Name, Project, Repo, Path, Revision, Destination, Namespace, Owner string
		SyncPolicy                                                         json.RawMessage
	}
	type serverConfig struct {
		ID, ClusterID                       uint64
		Endpoint, Token, CA, Version, Owner string
	}
	type repositoryConfig struct {
		ID, ServerID                                         uint64
		URL, Branch, Type, Username, Password, SSHKey, Owner string
	}
	refs := make([]repositoryConfig, 0, len(repositories))
	for _, r := range repositories {
		refs = append(refs, repositoryConfig{r.ID, r.ServerID, r.RepoURL, r.Branch, r.Type, r.Username, r.Password, r.SSHKey, r.CreatedBy})
	}
	value, _ := json.Marshal(struct {
		App          appConfig
		Server       serverConfig
		Repositories []repositoryConfig
	}{
		App:          appConfig{app.ID, app.ServerID, app.Name, app.Project, app.SourceRepoURL, app.SourcePath, app.SourceTargetRev, app.DestinationServer, app.DestinationNS, app.CreatedBy, json.RawMessage(app.SyncPolicy)},
		Server:       serverConfig{server.ID, server.ClusterID, server.Endpoint, server.AuthToken, server.CACert, server.Version, server.CreatedBy},
		Repositories: refs,
	})
	hash := sha256.Sum256(value)
	return hex.EncodeToString(hash[:])
}

func actionConfigurationAvailable(server *ArgoCDServer, repositories []ArgoCDRepository) bool {
	return (server.Status == ArgoCDStatusUnknown || server.Status == ArgoCDStatusConnected) &&
		len(repositories) == 1 && (repositories[0].Status == "connected" || repositories[0].Status == "unknown")
}
