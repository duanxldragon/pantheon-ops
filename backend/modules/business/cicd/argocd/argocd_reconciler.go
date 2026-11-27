package argocd

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const reconcileBatchSize = 100

// Shared error/i18n keys for reconciliation outcomes, hoisted to constants so
// every surface spells the same key.
const (
	appSyncFailedKey          = "cicd.argocd.app_sync_failed"
	syncObservationTimeoutKey = "business.cicd.application.syncObservationTimeout"
)

// Reconciler synchronizes registered ArgoCD applications from bounded observations.
type Reconciler struct {
	db           *gorm.DB
	interval     time.Duration
	stopCh       chan struct{}
	startOnce    sync.Once
	stopOnce     sync.Once
	cancelMu     sync.Mutex
	cancel       context.CancelFunc
	serverCursor uint64
	appCursors   map[uint64]uint64
	newClient    func(string, string, string) (*ArgoCDClient, error)
	modeReader   tenant.ModeReader
}

// NewReconciler creates a reconciler with a safe default interval.
func NewReconciler(db *gorm.DB, interval time.Duration) *Reconciler {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Reconciler{db: db, interval: interval, stopCh: make(chan struct{}), appCursors: make(map[uint64]uint64), newClient: NewArgoCDClient, modeReader: contracts.RegisteredBusinessTenantModeReader()}
}

// Start reconciles immediately and then periodically; repeated calls do not start another loop.
func (r *Reconciler) Start(ctx context.Context) {
	r.startOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		loopCtx, cancel := context.WithCancel(ctx)
		r.cancelMu.Lock()
		r.cancel = cancel
		r.cancelMu.Unlock()
		defer cancel()
		select {
		case <-loopCtx.Done():
			return
		case <-r.stopCh:
			return
		default:
		}
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			if err := r.reconcile(loopCtx); err != nil && loopCtx.Err() == nil {
				logging.Error("ArgoCD reconciliation failed", zap.String("subsystem", "registry_scan"), zap.String("error_class", reconciliationErrorClass(err)))
			}
			select {
			case <-loopCtx.Done():
				return
			case <-r.stopCh:
				return
			case <-ticker.C:
			}
		}
	})
}

// Stop is idempotent and cancels in-flight HTTP/DB work owned by this loop.
func (r *Reconciler) Stop() {
	r.stopOnce.Do(func() { close(r.stopCh) })
	r.cancelMu.Lock()
	cancel := r.cancel
	r.cancelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (r *Reconciler) reconcile(ctx context.Context) error {
	if r.db == nil {
		return errors.New("database.not_initialized")
	}
	passCtx, cancel := withCustomTimeout(ctx, 30*time.Second)
	defer cancel()
	if r.modeReader == nil {
		return tenant.ErrModeUnavailable
	}
	mode, err := r.modeReader.ReadMode(passCtx)
	if err != nil || (mode != tenant.ModeCompat && mode != tenant.ModeMulti) {
		return tenant.ErrModeUnavailable
	}
	scan := r.db.WithContext(passCtx).Where("id > ?", r.serverCursor)
	if mode == tenant.ModeCompat {
		scan = argoCDOwnerScope(tenant.PlatformGlobalTenantID).Scoped(scan)
	}
	var servers []ArgoCDServer
	// Disconnected servers are retried rather than permanently removed from the loop.
	if err := scan.Order("id ASC").Limit(reconcileBatchSize).Find(&servers).Error; err != nil {
		return err
	}
	if len(servers) == 0 {
		r.serverCursor = 0
		return nil
	}
	for _, server := range servers {
		if err := passCtx.Err(); err != nil {
			return err
		}
		r.serverCursor = server.ID
		if server.TenantID != tenant.PlatformGlobalTenantID && tenant.EnsureTenantLoginable(r.db.WithContext(passCtx), server.TenantID) != nil {
			continue
		}
		serverErr := r.reconcileServer(passCtx, &server)
		serverStatus := ArgoCDStatusConnected
		if serverErr != nil {
			serverStatus = ArgoCDStatusDisconnected
			logging.Error("ArgoCD server reconciliation failed", zap.Uint64("server_id", server.ID), zap.String("subsystem", "server_reconcile"), zap.String("error_class", reconciliationErrorClass(serverErr)))
		}
		if err := argoCDOwnerScope(server.TenantID).Scoped(r.db.WithContext(passCtx)).Model(&server).Updates(map[string]interface{}{"status": serverStatus, "last_checked": time.Now().UTC()}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Reconciler) reconcileServer(ctx context.Context, server *ArgoCDServer) error {
	authToken, serverErr := DecryptSecret(server.AuthToken)
	newClient := r.newClient
	if newClient == nil {
		newClient = NewArgoCDClient
	}
	var client *ArgoCDClient
	if serverErr == nil {
		client, serverErr = newClient(server.Endpoint, authToken, server.CACert)
	}
	if serverErr == nil {
		probeCtx, cancel := withCustomTimeout(ctx, 10*time.Second)
		serverErr = client.TestConnection(probeCtx)
		cancel()
	}
	if serverErr != nil {
		// Local pending expiry must continue even when setup or probing fails.
		// A nil client performs no remote application request.
		client = nil
	}
	var apps []ArgoCDApplication
	if err := argoCDOwnerScope(server.TenantID).Scoped(r.db.WithContext(ctx)).Where("server_id = ? AND id > ?", server.ID, r.appCursors[server.ID]).Order("id ASC").Limit(reconcileBatchSize).Find(&apps).Error; err != nil {
		return err
	}
	if len(apps) == 0 {
		r.appCursors[server.ID] = 0
		return serverErr
	}
	for _, app := range apps {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.appCursors[server.ID] = app.ID
		if err := r.reconcileApplication(ctx, client, &app); err != nil {
			logging.Error("ArgoCD application reconciliation failed", zap.Uint64("application_id", app.ID), zap.String("subsystem", "application_observation"), zap.String("error_class", reconciliationErrorClass(err)))
		}
	}
	return serverErr
}

// reconciliationErrorClass excludes arbitrary provider/driver error strings.
func reconciliationErrorClass(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, errEncryptionKeyMissing), errors.Is(err, errEncryptionKeyInvalid), errors.Is(err, errCiphertextMalformed), errors.Is(err, errCiphertextTampered):
		return "credential_configuration"
	case errors.Is(err, ErrInvalidCACert):
		return "certificate_configuration"
	case errors.Is(err, ErrApplicationNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		return "not_found"
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) {
		return "network"
	}
	return "dependency_failure"
}

func (r *Reconciler) latestHistory(db *gorm.DB, appID uint64) (ArgoCDSyncHistory, error) {
	var history ArgoCDSyncHistory
	err := db.Where("application_id = ?", appID).Order("id DESC").First(&history).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return history, nil
	}
	return history, err
}

func (r *Reconciler) reconcileApplication(ctx context.Context, client *ArgoCDClient, app *ArgoCDApplication) error {
	owner := argoCDOwnerScope(app.TenantID)
	var observed ArgoCDApplication
	if err := owner.Scoped(r.db.WithContext(ctx)).First(&observed, app.ID).Error; err != nil {
		return err
	}
	history, err := r.latestHistory(owner.Scoped(r.db.WithContext(ctx)), app.ID)
	if err != nil {
		return err
	}
	var status *ArgoCDApplicationStatus
	var observeErr error
	if client == nil {
		if history.Status != syncHistoryRunning || history.FinishedAt != nil || time.Since(history.StartedAt) < syncObservationTimeout {
			return nil
		}
	} else {
		opCtx, cancel := withCustomTimeout(ctx, 10*time.Second)
		status, observeErr = client.GetApplicationStatus(opCtx, observed.Name)
		cancel()
	}
	missing := errors.Is(observeErr, ErrApplicationNotFound)
	if observeErr != nil && !missing && (history.Status != syncHistoryRunning || time.Since(history.StartedAt) < syncObservationTimeout) {
		return observeErr
	}
	if missing {
		status = &ArgoCDApplicationStatus{}
		status.Health.Status, status.Sync.Status = HealthMissing, AppStatusUnknown
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current ArgoCDApplication
		if err := owner.Scoped(tx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, app.ID).Error; err != nil {
			return err
		}
		latest, err := r.latestHistory(owner.Scoped(tx), app.ID)
		if err != nil {
			return err
		}
		// An observation obtained for an older request/source/state cannot overwrite a newer one.
		if !sameApplicationSource(&current, &observed) || !current.UpdatedAt.Equal(observed.UpdatedAt) || latest.ID != history.ID {
			return nil
		}
		outcome := observedApplicationOutcome(status)
		now := time.Now().UTC()
		pending := latest.Status == syncHistoryRunning && latest.FinishedAt == nil
		message := ""
		if pending && !missing && !historyMatchesObservation(&latest, status) {
			outcome = applicationOutcome{status: appStatusProgressing, historyStatus: syncHistoryRunning}
			status = nil
		}
		if pending && !outcome.finished && now.Sub(latest.StartedAt) >= syncObservationTimeout {
			outcome = applicationOutcome{status: AppStatusUnknown, historyStatus: syncHistoryUnknown, finished: true}
			message = syncObservationTimeoutKey
		}
		if missing {
			message = "app.not_found"
		} else if outcome.status == AppStatusError {
			// Provider messages may contain arbitrary secret-bearing detail; persisted/public messages are keys.
			message = appSyncFailedKey
		}
		updates := map[string]interface{}{"status": outcome.status, "health_status": HealthUnknown, "sync_status": AppStatusUnknown, "last_error": message}
		if status != nil {
			if status.Health.Status != "" {
				updates["health_status"] = status.Health.Status
			}
			if status.Sync.Status != "" {
				updates["sync_status"] = status.Sync.Status
			}
		}
		if outcome.status == AppStatusSynced {
			updates["last_synced_at"] = now
		}
		if pending && outcome.finished {
			if err := owner.Scoped(tx).Model(&latest).Where("status = ?", syncHistoryRunning).Updates(map[string]interface{}{"status": outcome.historyStatus, "message": message, "finished_at": now}).Error; err != nil {
				return err
			}
		}
		return owner.Scoped(tx).Model(&current).Updates(updates).Error
	})
}

// StartAll starts the application-owned loop. Cancellation of ctx stops its work.
func StartAll(ctx context.Context, db *gorm.DB) {
	reconciler := NewReconciler(db, 30*time.Second)
	go reconciler.Start(ctx)
}
