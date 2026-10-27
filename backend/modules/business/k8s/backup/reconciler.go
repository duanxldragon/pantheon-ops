package backup

import (
	"context"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"gorm.io/gorm"
)

// Reconciler synchronizes Velero backup/restore state from the cluster CRDs.
type Reconciler struct {
	db             *gorm.DB
	clusterService *cluster.ClusterService
	interval       time.Duration
	stopCh         chan struct{}
	stopOnce       sync.Once
}

// NewReconciler creates a reconciler
func NewReconciler(db *gorm.DB, clusterService *cluster.ClusterService, interval time.Duration) *Reconciler {
	return &Reconciler{
		db:             db,
		clusterService: clusterService,
		interval:       interval,
		stopCh:         make(chan struct{}),
	}
}

// Start begins the reconciliation loop
func (r *Reconciler) Start(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	logging.Info("Velero reconciler started", zap.Duration("interval", r.interval))

	for {
		select {
		case <-ctx.Done():
			logging.Info("Velero reconciler stopped")
			return
		case <-r.stopCh:
			logging.Info("Velero reconciler stopped")
			return
		case <-ticker.C:
			if err := r.reconcile(ctx); err != nil {
				logging.Error("velero reconciliation failed", zap.Error(err))
			}
		}
	}
}

// Stop stops the reconciler
func (r *Reconciler) Stop() {
	r.stopOnce.Do(func() { close(r.stopCh) })
}

// veleroClient builds a Velero client for a cluster.
func (r *Reconciler) veleroClient(ctx context.Context, clusterID uint64) (*VeleroClient, error) {
	clientset, err := r.clusterService.GetClientsetWithContext(ctx, clusterID, nil)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := r.clusterService.GetDynamicClient(clusterID, nil)
	if err != nil {
		return nil, err
	}
	return NewVeleroClient(clientset, dynamicClient), nil
}

// reconcile synchronizes all Velero states
func (r *Reconciler) reconcile(ctx context.Context) error {
	if err := r.reconcileBackups(ctx); err != nil {
		logging.Error("backup reconciliation failed", zap.Error(err))
	}
	if err := r.reconcileRestores(ctx); err != nil {
		logging.Error("restore reconciliation failed", zap.Error(err))
	}
	return nil
}

// reconcileBackups synchronizes backup states
func (r *Reconciler) reconcileBackups(ctx context.Context) error {
	var backups []VeleroBackup
	if err := r.db.Where("status IN ?", []string{StatusNew, StatusInProgress}).Find(&backups).Error; err != nil {
		return err
	}
	for i := range backups {
		if err := r.reconcileBackup(ctx, &backups[i]); err != nil {
			logging.Error("backup reconciliation failed",
				zap.String("backup", backups[i].Name), zap.Error(err))
		}
	}
	return nil
}

// reconcileBackup synchronizes one backup state from its Velero Backup CRD.
func (r *Reconciler) reconcileBackup(ctx context.Context, backup *VeleroBackup) error {
	vc, err := r.veleroClient(ctx, backup.ClusterID)
	if err != nil {
		return err
	}
	obj, err := vc.GetBackup(ctx, backup.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// The CRD no longer exists; reflect that in the local record.
			return r.db.Model(backup).Update("status", StatusFailed).Error
		}
		return err
	}

	status, ok := nestedMap(obj.Object, "status")
	if !ok {
		return nil // status not populated yet; wait for the next tick
	}

	updates := map[string]interface{}{}
	if phase := nestedString(status, "phase"); phase != "" {
		updates["phase"] = phase
		updates["status"] = mapVeleroPhase(phase)
	}
	if v, ok := nestedInt64(status, "errors"); ok {
		updates["errors"] = v
	}
	if v, ok := nestedInt64(status, "warnings"); ok {
		updates["warnings"] = v
	}
	if v, ok := nestedInt64(status, "volumeSnapshotsAttempted"); ok {
		updates["volume_snapshots_count"] = v
	}
	if v, ok := nestedInt64(status, "backupItemOperationsAttempted"); ok {
		updates["backup_size"] = v
	}
	if ts := nestedString(status, "startTimestamp"); ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updates["start_time"] = parsed
		}
	}
	if ts := nestedString(status, "completionTimestamp"); ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updates["completion_time"] = parsed
		}
	}
	if ts := nestedString(status, "expiration"); ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updates["expiration"] = parsed
		}
	}

	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(backup).Updates(updates).Error
}

// reconcileRestores synchronizes restore states
func (r *Reconciler) reconcileRestores(ctx context.Context) error {
	var restores []VeleroRestore
	if err := r.db.Where("status IN ?", []string{StatusNew, StatusInProgress}).Find(&restores).Error; err != nil {
		return err
	}
	for i := range restores {
		if err := r.reconcileRestore(ctx, &restores[i]); err != nil {
			logging.Error("restore reconciliation failed",
				zap.String("restore", restores[i].Name), zap.Error(err))
		}
	}
	return nil
}

// reconcileRestore synchronizes one restore state from its Velero Restore CRD.
func (r *Reconciler) reconcileRestore(ctx context.Context, restore *VeleroRestore) error {
	vc, err := r.veleroClient(ctx, restore.ClusterID)
	if err != nil {
		return err
	}
	obj, err := vc.GetRestore(ctx, restore.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.db.Model(restore).Update("status", StatusFailed).Error
		}
		return err
	}

	status, ok := nestedMap(obj.Object, "status")
	if !ok {
		return nil
	}

	updates := map[string]interface{}{}
	if phase := nestedString(status, "phase"); phase != "" {
		updates["phase"] = phase
		updates["status"] = mapVeleroPhase(phase)
	}
	if v, ok := nestedInt64(status, "errors"); ok {
		updates["errors"] = v
	}
	if v, ok := nestedInt64(status, "warnings"); ok {
		updates["warnings"] = v
	}
	if ts := nestedString(status, "startTimestamp"); ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updates["start_time"] = parsed
		}
	}
	if ts := nestedString(status, "completionTimestamp"); ts != "" {
		if parsed, err := time.Parse(time.RFC3339, ts); err == nil {
			updates["completion_time"] = parsed
		}
	}

	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(restore).Updates(updates).Error
}

// CleanupExpiredBackups removes expired backup CRDs and local records.
func (r *Reconciler) CleanupExpiredBackups(ctx context.Context) error {
	var backups []VeleroBackup
	if err := r.db.Where("expiration IS NOT NULL AND expiration < ? AND status = ?",
		time.Now(), StatusCompleted).Find(&backups).Error; err != nil {
		return err
	}

	for i := range backups {
		backup := &backups[i]
		logging.Info("cleaning up expired backup",
			zap.String("backup", backup.Name),
			zap.Time("expiration", *backup.Expiration))

		vc, err := r.veleroClient(ctx, backup.ClusterID)
		if err == nil {
			_ = vc.DeleteBackup(ctx, backup.Name)
		}
		if err := r.db.Delete(backup).Error; err != nil {
			logging.Error("failed to delete expired backup",
				zap.String("backup", backup.Name), zap.Error(err))
		}
	}
	return nil
}

// StartAll starts all Velero reconcilers
func StartAll(ctx context.Context, db *gorm.DB, clusterService *cluster.ClusterService) {
	reconciler := NewReconciler(db, clusterService, 60*time.Second)
	go reconciler.Start(ctx)

	// 过期备份清理 (每小时)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := reconciler.CleanupExpiredBackups(ctx); err != nil {
					logging.Error("cleanup expired backups failed", zap.Error(err))
				}
			}
		}
	}()
}

// mapVeleroPhase maps a Velero CRD phase to the local status vocabulary.
func mapVeleroPhase(phase string) string {
	switch phase {
	case "New":
		return StatusNew
	case "InProgress", "Deleting":
		return StatusInProgress
	case StatusCompleted:
		return StatusCompleted
	default: // Failed, PartiallyFailed, anything else
		return StatusFailed
	}
}

func nestedMap(obj map[string]interface{}, key string) (map[string]interface{}, bool) {
	value, found, err := unstructured.NestedMap(obj, key)
	return value, found && err == nil
}

func nestedString(obj map[string]interface{}, key string) string {
	value, found, err := unstructured.NestedString(obj, key)
	if err != nil || !found {
		return ""
	}
	return value
}

func nestedInt64(obj map[string]interface{}, key string) (int64, bool) {
	value, found, err := unstructured.NestedInt64(obj, key)
	return value, found && err == nil
}
