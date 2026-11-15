package argocd

import (
	"context"
	"fmt"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Reconciler synchronizes ArgoCD Application state
type Reconciler struct {
	db            *gorm.DB
	interval      time.Duration
	stopCh        chan struct{}
	clientFactory func(string, string, string) (*ArgoCDClient, error)
}

// NewReconciler creates a reconciler
func NewReconciler(db *gorm.DB, interval time.Duration) *Reconciler {
	return &Reconciler{
		db:       db,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start begins the reconciliation loop
func (r *Reconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	logging.Info("ArgoCD reconciler started", zap.Duration("interval", r.interval))

	for {
		select {
		case <-ctx.Done():
			logging.Info("ArgoCD reconciler stopped")
			return
		case <-r.stopCh:
			logging.Info("ArgoCD reconciler stopped")
			return
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				logging.Error("reconciliation failed", zap.String("error_type", fmt.Sprintf("%T", err)))
			}
		}
	}
}

// Stop stops the reconciler
func (r *Reconciler) Stop() {
	close(r.stopCh)
}

// reconcile synchronizes all applications
func (r *Reconciler) reconcile(ctx context.Context) error {
	// 1. 查询所有 ArgoCD Server
	var servers []ArgoCDServer
	// A transient outage must not permanently remove a server from recovery.
	if err := r.db.WithContext(ctx).Where("status IN ?", []string{ArgoCDStatusUnknown, ArgoCDStatusConnected, ArgoCDStatusDisconnected}).Find(&servers).Error; err != nil {
		return err
	}

	for _, server := range servers {
		if err := r.reconcileServer(ctx, &server); err != nil {
			logging.Error("server reconciliation failed",
				zap.String("server", server.Name), zap.String("error_type", fmt.Sprintf("%T", err)))

			// 更新服务器状态
			r.db.Model(&server).Updates(map[string]interface{}{
				"status":       "disconnected",
				"last_checked": time.Now(),
			})
		} else {
			// 更新服务器状态为正常
			r.db.Model(&server).Updates(map[string]interface{}{
				"status":       "connected",
				"last_checked": time.Now(),
			})
		}
	}

	return nil
}

// reconcileServer synchronizes applications for one server
func (r *Reconciler) reconcileServer(ctx context.Context, server *ArgoCDServer) error {
	// Decrypt the stored token before authenticating against the remote.
	authToken, err := DecryptSecret(server.AuthToken)
	if err != nil {
		return err
	}

	// 1. 创建 ArgoCD client
	factory := r.clientFactory
	if factory == nil {
		factory = NewArgoCDClient
	}
	client, err := factory(server.Endpoint, authToken, server.CACert)
	if err != nil {
		return err
	}

	// 2. 健康检查
	if err := client.TestConnection(ctx); err != nil {
		return err
	}

	// 3. 查询该服务器的所有 Application
	var apps []ArgoCDApplication
	if err := r.db.Where("server_id = ?", server.ID).Find(&apps).Error; err != nil {
		return err
	}

	// 4. 同步每个 Application 状态
	for _, app := range apps {
		if err := r.reconcileApplication(ctx, client, &app); err != nil {
			logging.Error("application reconciliation failed",
				zap.String("app", app.Name), zap.String("error_type", fmt.Sprintf("%T", err)))
		}
	}

	return nil
}

// reconcileApplication synchronizes one application
func (r *Reconciler) reconcileApplication(ctx context.Context, client *ArgoCDClient, app *ArgoCDApplication) error {
	// 1. 从 ArgoCD 获取最新状态
	status, err := client.GetApplicationStatus(ctx, app.Name)
	if err != nil {
		return err
	}

	return applyApplicationStatus(ctx, r.db, app, status)
}

func applyApplicationStatus(ctx context.Context, db *gorm.DB, app *ArgoCDApplication, status *ArgoCDApplicationStatus) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current ArgoCDApplication
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, app.ID).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"sync_status": safeSyncStatus(status.Sync.Status), "health_status": safeHealthStatus(status.Health.Status)}
		var pending []ArgoCDSyncHistory
		if err := tx.Where("application_id = ? AND status IN ? AND finished_at IS NULL", app.ID, []string{AppStatusUnknown, "Running"}).Order("id DESC").Find(&pending).Error; err != nil {
			return err
		}
		if len(pending) > 0 {
			updates["status"] = AppStatusUnknown
			updates["last_error"] = ""
		}
		for _, history := range pending {
			metadata, ok := decodeActionMetadata(history.Message)
			operation := status.OperationState
			if !ok || metadata.UID != status.ApplicationUID || operation == nil || operation.Operation == nil || operation.Operation.Sync.Revision != metadata.Revision {
				continue
			}
			markers := 0
			matched := false
			for _, info := range operation.Operation.Info {
				if info.Name == actionInfoName {
					markers++
					matched = info.Value == metadata.Nonce
				}
			}
			if markers != 1 || !matched {
				continue
			}
			phase := operation.Phase
			if phase != "Running" && phase != "Succeeded" && phase != "Failed" && phase != "Error" {
				continue
			}
			terminal := phase != "Running"
			if terminal && (operation.StartedAt == nil || operation.FinishedAt == nil || operation.FinishedAt.Before(*operation.StartedAt) || (phase == "Succeeded" && operation.SyncResult.Revision != metadata.Revision)) {
				continue
			}
			metadata.Error = ""
			if phase == "Failed" || phase == "Error" {
				metadata.Error = "cicd.argocd.app_sync_failed"
				if metadata.Rollback {
					metadata.Error = "cicd.argocd.app_rollback_failed"
				}
			}
			values := map[string]interface{}{"status": phase, "message": encodeActionMetadata(metadata)}
			if terminal {
				values["finished_at"] = operation.FinishedAt
			}
			if err := tx.Model(&history).Where("finished_at IS NULL").Updates(values).Error; err != nil {
				return err
			}
			updates["status"] = phase
			updates["last_error"] = metadata.Error
			if phase == "Succeeded" {
				updates["last_synced_at"] = operation.FinishedAt
			}
		}
		return tx.Model(&current).Updates(updates).Error
	})
}

func safeSyncStatus(value string) string {
	switch value {
	case "Synced", "OutOfSync", "Unknown":
		return value
	}
	return "Unknown"
}
func safeHealthStatus(value string) string {
	switch value {
	case "Healthy", "Degraded", "Progressing", "Missing", "Unknown", "Suspended":
		return value
	}
	return "Unknown"
}

// StartAll starts all reconcilers
func StartAll(ctx context.Context, db *gorm.DB) {
	// ArgoCD Reconciler (每30秒)
	argocdReconciler := NewReconciler(db, 30*time.Second)
	go argocdReconciler.Start(ctx)
}
