package argocd

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"pantheon-base/pkg/logging"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

// WebhookEvent represents a webhook event
type WebhookEvent struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Source      string    `gorm:"size:32;not null" json:"source"` // github, gitlab, argocd
	EventType   string    `gorm:"size:64;not null" json:"eventType"`
	Repository  string    `gorm:"size:255" json:"repository"`
	Ref         string    `gorm:"size:255" json:"ref"`
	CommitSHA   string    `gorm:"size:128" json:"commitSha"`
	CommitMsg   string    `gorm:"type:text" json:"commitMessage"`
	Author      string    `gorm:"size:128" json:"author"`
	Payload     string    `gorm:"type:longtext" json:"payload"`
	Processed   bool      `gorm:"default:false" json:"processed"`
	ProcessedAt *time.Time `json:"processedAt,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (WebhookEvent) TableName() string {
	return "biz_cicd_webhook_event"
}

// WebhookHandler handles webhook events
type WebhookHandler struct {
	db *gorm.DB
}

// NewWebhookHandler creates a webhook handler
func NewWebhookHandler(db *gorm.DB) *WebhookHandler {
	return &WebhookHandler{db: db}
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
	defer r.Body.Close()

	// Determine source
	source := "github"
	if r.Header.Get("X-Gitlab-Event") != "" {
		source = "gitlab"
	} else if r.Header.Get("X-Gitea-Event") != "" {
		source = "gitea"
	}

	// Verify signature (optional, based on source)
	if err := h.verifySignature(r, body, source); err != nil {
		logging.Warn("webhook signature verification failed", zap.Error(err))
		// Continue anyway for now
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
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
	defer r.Body.Close()

	// Parse event
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create event record
	event := &WebhookEvent{
		Source:     "argocd",
		EventType:  "app-sync",
		Payload:    string(body),
		Processed:  false,
		CreatedAt:  time.Now(),
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
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// verifySignature verifies webhook signature
func (h *WebhookHandler) verifySignature(r *http.Request, body []byte, source string) error {
	var signature string

	switch source {
	case "github":
		signature = r.Header.Get("X-Hub-Signature-256")
		if signature == "" {
			return fmt.Errorf("missing signature")
		}
		// TODO: Get secret from config
		secret := "your-webhook-secret"
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedMAC := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(signature), []byte(expectedMAC)) {
			return fmt.Errorf("invalid signature")
		}
	case "gitlab":
		signature = r.Header.Get("X-Gitlab-Token")
		if signature == "" {
			return fmt.Errorf("missing token")
		}
		// TODO: Verify against configured token
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
	case "github":
		event.EventType = r.Header.Get("X-GitHub-Event")
		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			if url, ok := repo["clone_url"].(string); ok {
				event.Repository = url
			}
		}
		if ref, ok := payload["ref"].(string); ok {
			event.Ref = ref
		}
		if after, ok := payload["after"].(string); ok {
			event.CommitSHA = after
		}
		if sender, ok := payload["sender"].(map[string]interface{}); ok {
			if login, ok := sender["login"].(string); ok {
				event.Author = login
			}
		}

	case "gitlab":
		event.EventType = r.Header.Get("X-Gitlab-Event")
		if repo, ok := payload["project"].(map[string]interface{}); ok {
			if url, ok := repo["git_http_url"].(string); ok {
				event.Repository = url
			}
		}
		if ref, ok := payload["ref"].(string); ok {
			event.Ref = ref
		}
		if commits, ok := payload["commits"].([]interface{}); ok && len(commits) > 0 {
			if commit, ok := commits[0].(map[string]interface{}); ok {
				if id, ok := commit["id"].(string); ok {
					event.CommitSHA = id
				}
				if msg, ok := commit["message"].(string); ok {
					event.CommitMsg = msg
				}
				if author, ok := commit["author"].(map[string]interface{}); ok {
					if name, ok := author["name"].(string); ok {
						event.Author = name
					}
				}
			}
		}

	case "gitea":
		event.EventType = r.Header.Get("X-Gitea-Event")
		if repo, ok := payload["repository"].(map[string]interface{}); ok {
			if url, ok := repo["clone_url"].(string); ok {
				event.Repository = url
			}
		}
		if ref, ok := payload["ref"].(string); ok {
			event.Ref = ref
		}
	}

	return event
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
		// Trigger sync for auto-sync enabled apps
		if app.SyncPolicy != nil {
			// TODO: Parse sync policy and check if auto-sync is enabled
			logging.Info("triggering sync for application", zap.String("app", app.Name))
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
	json.Unmarshal([]byte(event.Payload), &payload)

	// Extract application name
	appName := event.Repository
	if appName == "" {
		if app, ok := payload["application"].(map[string]interface{}); ok {
			if metadata, ok := app["metadata"].(map[string]interface{}); ok {
				if name, ok := metadata["name"].(string); ok {
					appName = name
				}
			}
		}
	}

	// Find and update application
	var app ArgoCDApplication
	if err := h.db.Where("name = ?", appName).First(&app).Error; err != nil {
		logging.Warn("application not found for webhook", zap.String("app", appName))
		return
	}

	// Update application status from payload
	if status, ok := payload["status"].(map[string]interface{}); ok {
		if sync, ok := status["sync"].(map[string]interface{}); ok {
			if syncStatus, ok := sync["status"].(string); ok {
				h.db.Model(&app).Update("sync_status", syncStatus)
			}
		}
		if health, ok := status["health"].(map[string]interface{}); ok {
			if healthStatus, ok := health["status"].(string); ok {
				h.db.Model(&app).Update("health_status", healthStatus)
			}
		}
	}

	// Mark as processed
	now := time.Now()
	h.db.Model(event).Updates(map[string]interface{}{
		"processed":    true,
		"processed_at": now,
	})
}

// ListEvents lists webhook events
func (h *WebhookHandler) ListEvents(source string, limit int) ([]WebhookEvent, error) {
	var events []WebhookEvent
	query := h.db.Order("created_at DESC")
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
