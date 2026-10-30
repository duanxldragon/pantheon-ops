package argocd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

// Webhook sources stored in biz_cicd_webhook_event.source and matched by
// parseGitEvent, so the value is part of the persisted data contract.
const (
	sourceGitHub = "github"
	sourceGitLab = "gitlab"
	sourceGitea  = "gitea"
	sourceArgoCD = "argocd"
)

// Signature verification material is read from the environment. The names are
// environment variables, not credentials.
const (
	githubSignEnv = "PANTHEON_CICD_WEBHOOK_SIGN_GITHUB"
	gitlabSignEnv = "PANTHEON_CICD_WEBHOOK_SIGN_GITLAB"
)

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Source      string     `gorm:"size:32;not null" json:"source"` // github, gitlab, argocd
	EventType   string     `gorm:"size:64;not null" json:"eventType"`
	Repository  string     `gorm:"size:255" json:"repository"`
	Ref         string     `gorm:"size:255" json:"ref"`
	CommitSHA   string     `gorm:"size:128" json:"commitSha"`
	CommitMsg   string     `gorm:"type:text" json:"commitMessage"`
	Author      string     `gorm:"size:128" json:"author"`
	Payload     string     `gorm:"type:longtext" json:"payload"`
	Processed   bool       `gorm:"default:false" json:"processed"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// TableName returns the biz_cicd_webhook_event table name.
func (WebhookEvent) TableName() string {
	return "biz_cicd_webhook_event"
}

// WebhookHandler handles webhook events
type WebhookHandler struct {
	db     *gorm.DB
	syncer applicationSyncer
}

// applicationSyncer starts a sync for a stored application. *ArgoCDService satisfies it;
// the webhook depends on the behaviour instead of the concrete service so the trigger
// decision can be tested without an Argo CD server.
type applicationSyncer interface {
	SyncApplication(id uint64, revision string) (*SyncHistoryResponse, error)
}

// NewWebhookHandler creates a webhook handler
func NewWebhookHandler(db *gorm.DB) *WebhookHandler {
	return &WebhookHandler{db: db, syncer: NewArgoCDService(db)}
}

// HandleGitWebhook handles Git webhook events (GitHub/GitLab/Gitea)
func (h *WebhookHandler) HandleGitWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Determine source
	source := sourceGitHub
	if r.Header.Get("X-Gitlab-Event") != "" {
		source = sourceGitLab
	} else if r.Header.Get("X-Gitea-Event") != "" {
		source = sourceGitea
	}

	// Verify signature. An unverified delivery is refused: it would otherwise
	// let any caller that knows the endpoint write webhook rows and trigger
	// syncs for the applications named in them.
	if err := h.verifySignature(r, body, source); err != nil {
		logging.Warn("webhook signature verification failed", zap.Error(err))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse event
	event := h.parseGitEvent(r, body, source)
	if event == nil {
		http.Error(w, "Failed to parse event", http.StatusBadRequest)
		return
	}

	// Save event
	if err := h.db.Create(event).Error; err != nil {
		logging.Error("failed to save webhook event", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Process event asynchronously
	go h.processGitEvent(event)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleArgoCDWebhook handles ArgoCD webhook events
func (h *WebhookHandler) HandleArgoCDWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Parse event
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create event record
	event := &WebhookEvent{
		Source:    sourceArgoCD,
		EventType: "app-sync",
		Payload:   string(body),
		Processed: false,
		CreatedAt: time.Now(),
	}

	// Extract info from payload
	if app, ok := payload["application"].(map[string]interface{}); ok {
		if metadata, ok := app["metadata"].(map[string]interface{}); ok {
			if name, ok := metadata["name"].(string); ok {
				event.Repository = name
			}
		}
	}

	// Save event
	if err := h.db.Create(event).Error; err != nil {
		logging.Error("failed to save ArgoCD webhook event", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Process event asynchronously
	go h.processArgoCDEvent(event)

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// verifySignature checks the source-specific signature of a webhook delivery.
// A source whose verification material is not configured is rejected instead
// of falling back to a placeholder, so an unconfigured deployment fails closed.
func (h *WebhookHandler) verifySignature(r *http.Request, body []byte, source string) error {
	switch source {
	case sourceGitHub:
		signature := r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			return fmt.Errorf("missing signature")
		}
		want := strings.TrimSpace(os.Getenv(githubSignEnv))
		if want == "" {
			return fmt.Errorf("signature verification unavailable: %s is unset", githubSignEnv)
		}
		mac := hmac.New(sha256.New, []byte(want))
		mac.Write(body)
		expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
			return fmt.Errorf("invalid signature")
		}
	case sourceGitLab:
		token := r.Header.Get("X-Gitlab-Token")
		if token == "" {
			return fmt.Errorf("missing token")
		}
		want := strings.TrimSpace(os.Getenv(gitlabSignEnv))
		if want == "" {
			return fmt.Errorf("signature verification unavailable: %s is unset", gitlabSignEnv)
		}
		if !hmac.Equal([]byte(token), []byte(want)) {
			return fmt.Errorf("invalid token")
		}
	}

	return nil
}

// parseGitEvent parses Git webhook event
func (h *WebhookHandler) parseGitEvent(r *http.Request, body []byte, source string) *WebhookEvent {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	event := &WebhookEvent{
		Source:    source,
		Processed: false,
		CreatedAt: time.Now(),
	}

	switch source {
	case sourceGitHub:
		parseGitHubEventFields(event, payload, r.Header)
	case sourceGitLab:
		parseGitLabEventFields(event, payload, r.Header)
	case sourceGitea:
		parseGiteaEventFields(event, payload, r.Header)
	}

	return event
}

// parseGitHubEventFields copies the repository, ref, commit and sender of a GitHub delivery
// onto the event row.
func parseGitHubEventFields(event *WebhookEvent, payload map[string]interface{}, header http.Header) {
	event.EventType = header.Get("X-GitHub-Event")
	if url, ok := webhookString(payload, "repository", "clone_url"); ok {
		event.Repository = url
	}
	if ref, ok := webhookString(payload, "ref"); ok {
		event.Ref = ref
	}
	if after, ok := webhookString(payload, "after"); ok {
		event.CommitSHA = after
	}
	if login, ok := webhookString(payload, "sender", "login"); ok {
		event.Author = login
	}
}

// parseGitLabEventFields copies the project, ref and head commit of a GitLab delivery onto
// the event row.
func parseGitLabEventFields(event *WebhookEvent, payload map[string]interface{}, header http.Header) {
	event.EventType = header.Get("X-Gitlab-Event")
	if url, ok := webhookString(payload, "project", "git_http_url"); ok {
		event.Repository = url
	}
	if ref, ok := webhookString(payload, "ref"); ok {
		event.Ref = ref
	}
	commits, ok := payload["commits"].([]interface{})
	if !ok || len(commits) == 0 {
		return
	}
	applyGitLabHeadCommit(event, commits[0])
}

// applyGitLabHeadCommit reads the first commit of a GitLab push payload.
func applyGitLabHeadCommit(event *WebhookEvent, item interface{}) {
	commit, ok := item.(map[string]interface{})
	if !ok {
		return
	}
	if id, ok := webhookString(commit, "id"); ok {
		event.CommitSHA = id
	}
	if msg, ok := webhookString(commit, "message"); ok {
		event.CommitMsg = msg
	}
	if name, ok := webhookString(commit, "author", "name"); ok {
		event.Author = name
	}
}

// parseGiteaEventFields copies the repository and ref of a Gitea delivery onto the event row.
func parseGiteaEventFields(event *WebhookEvent, payload map[string]interface{}, header http.Header) {
	event.EventType = header.Get("X-Gitea-Event")
	if url, ok := webhookString(payload, "repository", "clone_url"); ok {
		event.Repository = url
	}
	if ref, ok := webhookString(payload, "ref"); ok {
		event.Ref = ref
	}
}

// webhookString walks an object path inside a decoded webhook payload and returns the string
// stored at the last key. A missing step, or a step that is not a JSON object, reports ok=false
// so callers keep the field untouched, exactly as the per-key type assertions they replace did.
func webhookString(payload map[string]interface{}, path ...string) (string, bool) {
	if len(path) == 0 {
		return "", false
	}
	current := payload
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]interface{})
		if !ok {
			return "", false
		}
		current = next
	}
	value, ok := current[path[len(path)-1]].(string)
	return value, ok
}

// gitSyncAutomated is Argo CD's automation marker object: its presence in the policy, not its
// fields, is what makes Argo CD reconcile on its own.
type gitSyncAutomated struct{}

// gitSyncPolicy is the subset of Argo CD's spec.syncPolicy that decides whether a Git push may
// trigger a sync.
type gitSyncPolicy struct {
	Automated *gitSyncAutomated `json:"automated"`
}

// hasAutomatedSyncPolicy reports whether a stored Argo CD syncPolicy object declares automated
// sync. An absent, null, unparseable or automated-less policy all mean "a push must not
// trigger a sync", so a malformed column fails closed instead of touching the deployment.
func hasAutomatedSyncPolicy(policy []byte) bool {
	trimmed := strings.TrimSpace(string(policy))
	if trimmed == "" || trimmed == "null" {
		return false
	}
	var parsed gitSyncPolicy
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return false
	}
	return parsed.Automated != nil
}

// processGitEvent processes a Git webhook event
func (h *WebhookHandler) processGitEvent(event *WebhookEvent) {
	logging.Info("processing git webhook event",
		zap.String("source", event.Source),
		zap.String("type", event.EventType),
		zap.String("repo", event.Repository))

	// Find applications that should be synced
	var apps []ArgoCDApplication
	if err := h.db.Where("source_repo_url = ? AND status != ?",
		event.Repository, "Error").Find(&apps).Error; err != nil {
		logging.Error("failed to find applications for webhook", zap.Error(err))
		return
	}

	for _, app := range apps {
		if !hasAutomatedSyncPolicy(app.SyncPolicy) {
			continue
		}
		logging.Info("triggering auto-sync for application", zap.String("app", app.Name))
		if _, err := h.syncer.SyncApplication(app.ID, app.SourceTargetRev); err != nil {
			logging.Error("auto-sync trigger failed",
				zap.String("app", app.Name), zap.Error(err))
		}
	}

	// Mark as processed
	now := time.Now()
	h.db.Model(event).Updates(map[string]interface{}{
		"processed":    true,
		"processed_at": now,
	})
}

// processArgoCDEvent processes an ArgoCD webhook event
func (h *WebhookHandler) processArgoCDEvent(event *WebhookEvent) {
	logging.Info("processing argocd webhook event",
		zap.String("type", event.EventType),
		zap.String("repo", event.Repository))

	// Parse payload
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
		logging.Warn("unparseable argocd webhook payload", zap.Error(err))
		return
	}

	// Extract application name
	appName := event.Repository
	if appName == "" {
		if name, ok := webhookString(payload, "application", "metadata", "name"); ok {
			appName = name
		}
	}

	// Find and update application
	var app ArgoCDApplication
	if err := h.db.Where("name = ?", appName).First(&app).Error; err != nil {
		logging.Warn("application not found for webhook", zap.String("app", appName))
		return
	}

	// Update application status from payload, sync status first and health status second,
	// exactly as the previous inline extraction did.
	syncStatus, healthStatus := argoCDStatusFields(payload)
	if syncStatus != nil {
		h.db.Model(&app).Update("sync_status", *syncStatus)
	}
	if healthStatus != nil {
		h.db.Model(&app).Update("health_status", *healthStatus)
	}

	// Mark as processed
	now := time.Now()
	h.db.Model(event).Updates(map[string]interface{}{
		"processed":    true,
		"processed_at": now,
	})
}

// argoCDStatusFields reads the sync and health statuses of an Argo CD notification payload.
// A nil value means the payload carried no string for that field, so the caller leaves the
// stored status untouched rather than writing an empty one.
func argoCDStatusFields(payload map[string]interface{}) (syncStatus *string, healthStatus *string) {
	status, ok := payload["status"].(map[string]interface{})
	if !ok {
		return nil, nil
	}
	if value, ok := webhookString(status, "sync", "status"); ok {
		syncStatus = &value
	}
	if value, ok := webhookString(status, "health", "status"); ok {
		healthStatus = &value
	}
	return syncStatus, healthStatus
}

// ListEvents lists webhook events
func (h *WebhookHandler) ListEvents(source string, limit int) ([]WebhookEvent, error) {
	var events []WebhookEvent
	query := h.db.Order(orderByCreatedAtDesc)
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
