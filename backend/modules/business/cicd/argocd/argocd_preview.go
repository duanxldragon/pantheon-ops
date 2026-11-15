package argocd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"sigs.k8s.io/yaml"
)

var gitRevision = regexp.MustCompile(`^[a-fA-F0-9]{40}([a-fA-F0-9]{24})?$`)

type ApplicationPreviewRequest struct {
	Revision string `json:"revision" binding:"max=128"`
	Rollback bool   `json:"rollback"`
}

// ApplicationPreviewChange contains only validated resource identity and change
// classification. All manifest values, labels, annotations and secrets remain
// private, including for copy/download/audit consumers.
type ApplicationPreviewChange struct {
	Group        string `json:"group"`
	Kind         string `json:"kind"`
	Namespace    string `json:"namespace"`
	Name         string `json:"name"`
	Change       string `json:"change"`
	ValuesHidden bool   `json:"valuesHidden"`
}

type ApplicationPreviewResponse struct {
	IntentID            string                       `json:"intentId"`
	IntentVersion       string                       `json:"intentVersion"`
	CurrentVersion      string                       `json:"currentVersion"`
	TargetVersion       string                       `json:"targetVersion"`
	Changes             []ApplicationPreviewChange   `json:"changes"`
	TargetCount         int                          `json:"targetCount"`
	RollbackAvailable   bool                         `json:"rollbackAvailable"`
	HealthCheckRequired bool                         `json:"healthCheckRequired"`
	ExpiresAt           time.Time                    `json:"expiresAt"`
	Namespace           string                       `json:"namespace"`
	CurrentRevision     string                       `json:"currentRevision"`
	History             []ApplicationPreviewRevision `json:"history"`
	HistoryTruncated    bool                         `json:"historyTruncated"`
	RiskKeys            []string                     `json:"riskKeys"`
}

// ApplicationPreviewRevision exposes provider history identity, never its source configuration.
type ApplicationPreviewRevision struct {
	ID                int64  `json:"id"`
	Revision          string `json:"revision"`
	Current           bool   `json:"current"`
	RollbackAvailable bool   `json:"rollbackAvailable"`
}

type applicationReview struct {
	ApplicationID                   uint64
	Actor                           uint64
	Revision                        string
	RequestedRevision               string
	Rollback                        bool
	Protocol                        string
	HistoryID                       *int64
	Fingerprint, UID, Version, Diff string
	Expires                         int64
}

type applicationActionPlan struct {
	app               ArgoCDApplication
	server            ArgoCDServer
	provider          *argoApplication
	client            *ArgoCDClient
	operation         *argoOperation
	historyID         *int64
	changes           []ApplicationPreviewChange
	fingerprint, diff string
}

// PreviewApplicationContext obtains real provider manifests/live state and
// signs a bounded review. An unconfigured authority cannot produce a ticket.
func (s *ArgoCDService) PreviewApplicationContext(ctx context.Context, id uint64, req ApplicationPreviewRequest, actor ActionActor) (*ApplicationPreviewResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	plan, err := s.prepareApplicationAction(ctx, id, req, actor)
	if err != nil {
		return nil, err
	}
	expires := time.Now().Add(5 * time.Minute)
	review := applicationReview{ApplicationID: id, Actor: actor.ID, Revision: plan.operation.Sync.Revision, RequestedRevision: req.Revision, Rollback: req.Rollback, Protocol: reviewedActionsProtocol, HistoryID: plan.historyID, Fingerprint: plan.fingerprint, UID: plan.provider.Metadata.UID, Version: plan.provider.Metadata.ResourceVersion, Diff: plan.diff, Expires: expires.Unix()}
	data, _ := json.Marshal(review)
	ticket, err := EncryptSecret(string(data))
	if err != nil {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	history, truncated := previewProviderHistory(plan.provider)
	currentRevision := plan.provider.Status.Sync.Revision
	if !reviewedRevision.MatchString(currentRevision) {
		currentRevision = ""
	}
	return &ApplicationPreviewResponse{IntentID: "argocd:" + plan.provider.Metadata.UID, IntentVersion: ticket, CurrentVersion: review.Version, TargetVersion: review.Revision, Changes: plan.changes, TargetCount: len(plan.changes), RollbackAvailable: req.Rollback, HealthCheckRequired: true, ExpiresAt: expires, Namespace: plan.app.DestinationNS, CurrentRevision: currentRevision, History: history, HistoryTruncated: truncated, RiskKeys: []string{"business.cicd.argocd.preview.valuesHidden", "business.cicd.argocd.preview.pruneDisabled", "business.cicd.argocd.preview.healthRequired", "business.cicd.argocd.preview.rollbackLimits"}}, nil
}

func previewProviderHistory(provider *argoApplication) ([]ApplicationPreviewRevision, bool) {
	var spec struct {
		Source json.RawMessage `json:"source"`
	}
	_ = json.Unmarshal(provider.Spec, &spec)
	history := make([]ApplicationPreviewRevision, 0, len(provider.Status.History))
	for _, entry := range provider.Status.History {
		if entry.ID < 0 || !reviewedRevision.MatchString(entry.Revision) {
			continue
		}
		history = append(history, ApplicationPreviewRevision{ID: entry.ID, Revision: entry.Revision, Current: entry.Revision == provider.Status.Sync.Revision, RollbackAvailable: len(entry.Sources) == 0 && sameSource(spec.Source, entry.Source)})
	}
	sort.Slice(history, func(i, j int) bool { return history[i].ID > history[j].ID })
	// Execute selects the newest compatible provider entry for a revision. Project
	// the same choice, including legitimate history ID zero, so selection is honest.
	seen := map[string]bool{}
	available := make([]ApplicationPreviewRevision, 0, len(history))
	for _, entry := range history {
		if !seen[entry.Revision] {
			available = append(available, entry)
			seen[entry.Revision] = true
		} else if entry.RollbackAvailable {
			for i := range available {
				if available[i].Revision == entry.Revision && !available[i].RollbackAvailable {
					available[i] = entry
					break
				}
			}
		}
	}
	truncated := len(available) > 50
	if truncated {
		available = available[:50]
	}
	return available, truncated
}

func (s *ArgoCDService) prepareApplicationAction(ctx context.Context, id uint64, req ApplicationPreviewRequest, actor ActionActor) (*applicationActionPlan, error) {
	plan := &applicationActionPlan{}
	if len(req.Revision) > 128 || strings.ContainsAny(req.Revision, "\r\n\x00") {
		return nil, common.NewBadRequest(msgParamInvalid)
	}
	if err := s.db.WithContext(ctx).First(&plan.app, id).Error; err != nil {
		return nil, common.NewNotFound("cicd.argocd.app_get_failed")
	}
	if err := s.db.WithContext(ctx).First(&plan.server, plan.app.ServerID).Error; err != nil {
		return nil, common.NewNotFound("cicd.argocd.server_get_failed")
	}
	if plan.server.Status != ArgoCDStatusUnknown && plan.server.Status != ArgoCDStatusConnected {
		return nil, common.NewForbidden("common.forbidden")
	}
	action := "sync"
	if req.Rollback {
		action = "rollback"
	}
	if err := s.authorizeAction(ctx, actor, action, &plan.app, &plan.server, []string{plan.app.SourceRepoURL}); err != nil {
		return nil, err
	}
	token, err := DecryptSecret(plan.server.AuthToken)
	if err != nil {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	factory := s.clientFactory
	if factory == nil {
		factory = NewArgoCDClient
	}
	plan.client, err = factory(plan.server.Endpoint, token, plan.server.CACert)
	if err != nil {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	plan.provider, err = plan.client.readApplication(ctx, plan.app.Name)
	if err != nil {
		return nil, safeApplicationActionError(ctx, req.Rollback)
	}
	if !boundedProviderIdentity(plan.provider.Metadata.UID) || !boundedProviderIdentity(plan.provider.Metadata.ResourceVersion) {
		return nil, common.NewConflict(msgParamInvalid)
	}
	if err := plan.client.requireReviewedCapabilities(ctx, plan.app.Name, plan.provider); err != nil {
		return nil, safeApplicationActionError(ctx, req.Rollback)
	}
	if plan.provider.Operation != nil || (plan.provider.Status.OperationState != nil && (plan.provider.Status.OperationState.Phase == "Running" || plan.provider.Status.OperationState.Phase == "Terminating")) {
		return nil, common.NewConflict(msgParamInvalid)
	}
	var spec struct {
		Project     string                 `json:"project"`
		Source      json.RawMessage        `json:"source"`
		Sources     []json.RawMessage      `json:"sources"`
		Destination applicationDestination `json:"destination"`
		SyncPolicy  struct {
			Automated json.RawMessage `json:"automated"`
		} `json:"syncPolicy"`
	}
	if json.Unmarshal(plan.provider.Spec, &spec) != nil || len(spec.Sources) > 0 || len(spec.Source) == 0 || (len(spec.SyncPolicy.Automated) > 0 && string(spec.SyncPolicy.Automated) != "null") {
		return nil, common.NewBadRequest(msgParamInvalid)
	}
	var source applicationSource
	var sourceKind struct {
		Chart string `json:"chart"`
	}
	if json.Unmarshal(spec.Source, &sourceKind) != nil || sourceKind.Chart != "" {
		return nil, common.NewBadRequest(msgParamInvalid)
	}
	if json.Unmarshal(spec.Source, &source) != nil || source.RepoURL != plan.app.SourceRepoURL || source.Path != plan.app.SourcePath || spec.Project != plan.app.Project || spec.Destination.Server != plan.app.DestinationServer || spec.Destination.Namespace != plan.app.DestinationNS {
		return nil, common.NewConflict(msgParamInvalid)
	}
	plan.operation = &argoOperation{}
	plan.operation.Sync.Source = spec.Source
	revision := req.Revision
	if revision == "" {
		revision = source.TargetRevision
	}
	if req.Rollback {
		var selected int64 = -1
		for _, entry := range plan.provider.Status.History {
			if entry.Revision == revision && entry.ID > selected && len(entry.Sources) == 0 && sameSource(spec.Source, entry.Source) {
				selected = entry.ID
				plan.operation.Sync.Source = entry.Source
			}
		}
		// The provider manifest API renders the current source. Historical
		// source/config drift is deliberately unsupported until it can be
		// previewed through a provider-owned capability, never approximated.
		if selected < 0 {
			return nil, errors.New("cicd.argocd.app_rollback_failed")
		}
		plan.historyID = &selected
	}
	if err := s.authorizeAction(ctx, actor, action, &plan.app, &plan.server, []string{source.RepoURL}); err != nil {
		return nil, err
	}
	var repositories []ArgoCDRepository
	if err := s.db.WithContext(ctx).Where("server_id = ? AND repo_url = ?", plan.server.ID, source.RepoURL).Order("id ASC").Find(&repositories).Error; err != nil {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	if !actionConfigurationAvailable(&plan.server, repositories) {
		return nil, common.NewForbidden("common.forbidden")
	}
	var target struct {
		Manifests []string `json:"manifests"`
		Revision  string   `json:"revision"`
	}
	if err := plan.client.readProvider(ctx, "/api/v1/applications/"+url.PathEscape(plan.app.Name)+"/manifests?revision="+url.QueryEscape(revision), &target); err != nil {
		return nil, safeApplicationActionError(ctx, req.Rollback)
	}
	// Argo may omit the resolved revision in its manifests response. A full
	// Git hash already pins the render request; a mutable ref still requires
	// an explicit provider-resolved hash and must never be guessed locally.
	if target.Revision == "" && reviewedRevision.MatchString(revision) {
		target.Revision = revision
	}
	if !reviewedRevision.MatchString(target.Revision) || ((req.Rollback || gitRevision.MatchString(revision)) && target.Revision != revision) {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	plan.operation.Sync.Revision = target.Revision
	var managed struct {
		Items []struct {
			NormalizedLiveState string `json:"normalizedLiveState"`
		} `json:"items"`
	}
	if err := plan.client.readProvider(ctx, "/api/v1/applications/"+url.PathEscape(plan.app.Name)+"/managed-resources", &managed); err != nil {
		return nil, safeApplicationActionError(ctx, req.Rollback)
	}
	live := make([]string, 0, len(managed.Items))
	for _, item := range managed.Items {
		live = append(live, item.NormalizedLiveState)
	}
	plan.changes, plan.diff, err = applicationDiff(live, target.Manifests, plan.app.DestinationNS)
	if err != nil {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	for _, change := range plan.changes {
		if change.Namespace != plan.app.DestinationNS || !supportedNamespacedResource(change.Group, change.Kind) {
			return nil, common.NewForbidden("common.forbidden")
		}
	}
	plan.fingerprint = actionFingerprint(&plan.app, &plan.server, repositories)
	if err := s.authorizeAction(ctx, actor, action, &plan.app, &plan.server, []string{source.RepoURL}); err != nil {
		return nil, err
	}
	return plan, nil
}

func sameSource(a, b json.RawMessage) bool {
	var x, y map[string]interface{}
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil || x == nil || y == nil {
		return false
	}
	delete(x, "targetRevision")
	delete(y, "targetRevision")
	return reflect.DeepEqual(x, y)
}

var resourceIdentity = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/-]{0,252}$`)

func applicationDiff(live, target []string, defaultNamespaces ...string) ([]ApplicationPreviewChange, string, error) {
	if len(live) > 200 || len(target) > 200 {
		return nil, "", errors.New(msgParamInvalid)
	}
	type resource struct {
		identity ApplicationPreviewChange
		value    map[string]interface{}
	}
	parse := func(values []string) (map[string]resource, error) {
		result := map[string]resource{}
		for _, value := range values {
			if value == "" || value == "null" {
				continue
			}
			var object map[string]interface{}
			if yaml.UnmarshalStrict([]byte(value), &object) != nil || object == nil {
				return nil, errors.New(msgParamInvalid)
			}
			api, _ := object["apiVersion"].(string)
			kind, _ := object["kind"].(string)
			group, _, ok := strings.Cut(api, "/")
			if !ok {
				group = ""
			}
			metadata, _ := object["metadata"].(map[string]interface{})
			name, _ := metadata["name"].(string)
			namespace, _ := metadata["namespace"].(string)
			if namespace == "" && len(defaultNamespaces) == 1 {
				if supportedNamespacedResource(group, kind) {
					namespace = defaultNamespaces[0]
					metadata["namespace"] = namespace
				} else {
					return nil, errors.New(msgParamInvalid)
				}
			}
			if !resourceIdentity.MatchString(api) || !resourceIdentity.MatchString(kind) || !resourceIdentity.MatchString(name) || (namespace != "" && !resourceIdentity.MatchString(namespace)) {
				return nil, errors.New(msgParamInvalid)
			}
			identity := ApplicationPreviewChange{Group: group, Kind: kind, Namespace: namespace, Name: name, ValuesHidden: true}
			key := group + "/" + kind + "/" + namespace + "/" + name
			if _, exists := result[key]; exists {
				return nil, errors.New(msgParamInvalid)
			}
			// Provider bookkeeping is not a proposed configuration value.
			delete(object, "status")
			for _, field := range []string{"resourceVersion", "uid", "creationTimestamp", "managedFields", "generation"} {
				delete(metadata, field)
			}
			result[key] = resource{identity: identity, value: object}
		}
		return result, nil
	}
	before, err := parse(live)
	if err != nil {
		return nil, "", err
	}
	after, err := parse(target)
	if err != nil {
		return nil, "", err
	}
	changes := make([]ApplicationPreviewChange, 0, len(before)+len(after))
	for key, current := range before {
		change := current.identity
		if desired, ok := after[key]; ok {
			change.Change = "unchanged"
			if !reflect.DeepEqual(current.value, desired.value) {
				change.Change = "modified"
			}
		} else {
			change.Change = "retained"
		} // prune=false
		changes = append(changes, change)
	}
	for key, desired := range after {
		if _, ok := before[key]; !ok {
			change := desired.identity
			change.Change = "added"
			changes = append(changes, change)
		}
	}
	sortPreviewChanges(changes)
	if len(changes) > 200 {
		return nil, "", errors.New(msgParamInvalid)
	}
	// Private manifest values must participate in drift detection even though
	// they are never projected. Hash the maps explicitly (resource is private).
	values := map[string]interface{}{}
	for k, v := range before {
		values["before:"+k] = v.value
	}
	for k, v := range after {
		values["after:"+k] = v.value
	}
	data, _ := json.Marshal(values)
	hash := sha256.Sum256(data)
	return changes, hex.EncodeToString(hash[:]), nil
}

func actionNonce() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(data[:]), nil
}

func sortPreviewChanges(changes []ApplicationPreviewChange) {
	sort.Slice(changes, func(i, j int) bool {
		a, _ := json.Marshal(changes[i])
		b, _ := json.Marshal(changes[j])
		return string(a) < string(b)
	})
}

func supportedNamespacedResource(group, kind string) bool {
	switch group + "/" + kind {
	case "/Service", "/ConfigMap", "/Secret", "/ServiceAccount", "/PersistentVolumeClaim",
		"apps/Deployment", "apps/StatefulSet", "apps/DaemonSet",
		"rbac.authorization.k8s.io/Role", "rbac.authorization.k8s.io/RoleBinding",
		"batch/Job", "batch/CronJob", "networking.k8s.io/Ingress", "networking.k8s.io/NetworkPolicy":
		return true
	}
	return false
}
