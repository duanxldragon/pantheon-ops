package argocd

import (
	"context"
	"time"

	"pantheon-base/pkg/logging"
	"go.uber.org/zap"

	"gorm.io/gorm"
)

// Reconciler synchronizes ArgoCD Application state
type Reconciler struct {
	db       *gorm.DB
	interval time.Duration
	stopCh   chan struct{}
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
				logging.Error("reconciliation failed", zap.Error(err))
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
	if err := r.db.Where("status != ?", "disconnected").Find(&servers).Error; err != nil {
		return err
	}

	for _, server := range servers {
		if err := r.reconcileServer(ctx, &server); err != nil {
			logging.Error("server reconciliation failed",
				zap.String("server", server.Name), zap.Error(err))

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
	client := NewArgoCDClient(server.Endpoint, authToken, false)

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
				zap.String("app", app.Name), zap.Error(err))
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

	// 2. 检测状态变化
	stateChanged := app.SyncStatus != status.Sync.Status ||
		app.HealthStatus != status.Health.Status

	// 3. 更新本地 Application 状态
	updates := map[string]interface{}{
		"sync_status":    status.Sync.Status,
		"health_status":  status.Health.Status,
		"last_synced_at": time.Now(),
	}

	if status.OperationState != nil {
		updates["last_error"] = status.OperationState.Message
	}

	if err := r.db.Model(app).Updates(updates).Error; err != nil {
		return err
	}

	// 4. 如果状态变化，记录同步历史
	if stateChanged {
		history := ArgoCDSyncHistory{
			ApplicationID: app.ID,
			Revision:      status.Sync.Revision,
			Status:        status.Sync.Status,
			StartedAt:     time.Now(),
			FinishedAt:    func() *time.Time { t := time.Now(); return &t }(),
		}

		if status.OperationState != nil {
			history.Status = status.OperationState.Phase
			history.Message = status.OperationState.Message
			if status.OperationState.StartedAt != nil {
				history.StartedAt = *status.OperationState.StartedAt
			}
		}

		if err := r.db.Create(&history).Error; err != nil {
			logging.Error("create sync history failed", zap.Error(err))
		}
	}

	// 5. 更新关联的服务实例状态 (如果有关联)
	if app.Status != status.Sync.Status {
		r.updateServiceInstanceState(app)
	}

	return nil
}

// updateServiceInstanceState updates associated service instance state
func (r *Reconciler) updateServiceInstanceState(app *ArgoCDApplication) {
	// TODO: 关联服务实例状态更新
	// 需要通过 capability 接口调用
	logging.Info("service instance state should be updated",
		zap.String("application", app.Name),
		zap.String("sync_status", app.SyncStatus))
}

// StartAll starts all reconcilers
func StartAll(ctx context.Context, db *gorm.DB) {
	// ArgoCD Reconciler (每30秒)
	argocdReconciler := NewReconciler(db, 30*time.Second)
	go argocdReconciler.Start(ctx)
}


