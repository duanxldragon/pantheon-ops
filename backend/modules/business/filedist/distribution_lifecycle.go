package filedist

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// executionInterruptedKey is the shared error/i18n key for tasks ended by an
// execution interruption; declared once to keep every surface spelling equal.
const executionInterruptedKey = "business.filedist.task.executionInterrupted"

func unfinishedTargetStatuses() []string {
	return []string{statusPending, statusTransferring, statusRetrying}
}

func (s *DistributionService) executionContext() context.Context {
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	if s.runtimeStopped {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}
	if s.runtimeCtx != nil {
		return s.runtimeCtx
	}
	return context.Background()
}

// Start acquires exclusive lifecycle ownership before reconciling interrupted
// work. Composition must call it after migration/policy wiring and before HTTP.
func (s *DistributionService) Start(ctx context.Context) error {
	if ctx == nil || s.db == nil || s.transferEngine == nil {
		return errors.New("database.not_initialized")
	}
	s.runtimeMu.Lock()
	defer s.runtimeMu.Unlock()
	if s.runtimeStopped {
		return errors.New(executionInterruptedKey)
	}
	if s.runtimeCtx != nil {
		return s.runtimeCtx.Err()
	}
	release, checkLease, err := acquireDistributionLease(ctx, s.db)
	if err != nil {
		return err
	}
	if err := s.recoverInterruptedTasks(ctx); err != nil {
		release()
		return err
	}
	s.runtimeCtx, s.runtimeCancel = context.WithCancel(ctx)
	s.runtimeDone = make(chan struct{})
	s.runtimeLeaseDone = make(chan struct{})
	s.releaseLease = release
	s.transferEngine.Start(s.runtimeCtx)
	go monitorDistributionLease(s.runtimeCtx, s.runtimeLeaseDone, checkLease, s.runtimeCancel)
	go s.runDispatcher(s.runtimeCtx, s.runtimeDone)
	return nil
}

// Stop interrupts the owned transport before releasing recovery ownership.
func (s *DistributionService) Stop() {
	s.runtimeMu.Lock()
	s.runtimeStopped = true
	cancel, done, leaseDone, release := s.runtimeCancel, s.runtimeDone, s.runtimeLeaseDone, s.releaseLease
	s.releaseLease = nil
	s.runtimeMu.Unlock()
	if cancel != nil {
		cancel()
		<-done
		<-leaseDone
	}
	if s.transferEngine != nil {
		s.transferEngine.Stop()
	}
	if release != nil {
		release()
	}
}

func acquireDistributionLease(ctx context.Context, db *gorm.DB) (func(), func(context.Context) error, error) {
	pool, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	connection, err := pool.Conn(ctx)
	if err != nil {
		return nil, nil, err
	}
	var databaseName string
	if err := connection.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&databaseName); err != nil {
		_ = connection.Close()
		return nil, nil, err
	}
	key := fmt.Sprintf("pantheon:filedist:%x", sha256.Sum256([]byte(databaseName)))[:63]
	var acquired sql.NullInt64
	if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", key).Scan(&acquired); err != nil || !acquired.Valid || acquired.Int64 != 1 {
		_ = connection.Close()
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, errors.New("business.filedist.task.executionBusy")
	}
	return func() {
			releaseCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, _ = connection.ExecContext(releaseCtx, "SELECT RELEASE_LOCK(?)", key)
			_ = connection.Close()
		}, func(ctx context.Context) error {
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			var owned sql.NullBool
			if err := connection.QueryRowContext(checkCtx, "SELECT IS_USED_LOCK(?) = CONNECTION_ID()", key).Scan(&owned); err != nil {
				return err
			}
			if !owned.Valid || !owned.Bool {
				return errors.New("business.filedist.task.executionBusy")
			}
			return nil
		}, nil
}

func (s *DistributionService) recoverInterruptedTasks(ctx context.Context) error {
	var after uint64
	for {
		var tasks []DistributionTask
		if err := s.db.WithContext(ctx).Where("id > ? AND status IN ?", after, []string{statusRunning, statusCancelled}).Order("id ASC").Limit(100).Find(&tasks).Error; err != nil {
			return err
		}
		if len(tasks) == 0 {
			return nil
		}
		for _, subject := range tasks {
			after = subject.ID
			if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				var task DistributionTask
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, subject.ID).Error; err != nil {
					return err
				}
				if task.Status != statusRunning && task.Status != statusCancelled {
					return nil
				}
				status, message := statusFailed, executionInterruptedKey
				if task.Status == statusCancelled {
					status, message = statusCancelled, ""
				}
				now := time.Now()
				if err := tx.Model(&DistributionTarget{}).Where("task_id = ? AND status IN ?", task.ID, unfinishedTargetStatuses()).Updates(map[string]interface{}{"status": status, "error": message, "finished_at": now}).Error; err != nil {
					return err
				}
				if task.Status == statusCancelled {
					return nil
				}
				var completed, failed int64
				if err := tx.Model(&DistributionTarget{}).Where(whereTaskAndStatus, task.ID, statusCompleted).Count(&completed).Error; err != nil {
					return err
				}
				if err := tx.Model(&DistributionTarget{}).Where(whereTaskAndStatus, task.ID, statusFailed).Count(&failed).Error; err != nil {
					return err
				}
				return tx.Model(&task).Updates(map[string]interface{}{
					"status": statusFailed, "error": message, "finished_at": now,
					"success_count": completed, "failed_count": failed, "progress": 100,
				}).Error
			}); err != nil {
				return err
			}
		}
	}
}

func monitorDistributionLease(ctx context.Context, done chan struct{}, checkLease func(context.Context) error, cancel context.CancelFunc) {
	defer close(done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		// A dropped lease connection must stop this owner's IO. This is bounded
		// failure detection, not remote-write fencing across network partitions.
		if err := checkLease(ctx); err != nil {
			cancel()
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *DistributionService) runDispatcher(ctx context.Context, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	discoveryFailed := false
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.dispatchDueTasks(ctx, time.Now()); err != nil {
			if ctx.Err() != nil {
				return
			}
			if !discoveryFailed {
				slog.Error("file distribution due-task discovery failed", "subsystem", "business.filedist.scheduler", "error_type", fmt.Sprintf("%T", err))
			}
			discoveryFailed = true
		} else if discoveryFailed {
			slog.Info("file distribution due-task discovery recovered", "subsystem", "business.filedist.scheduler")
			discoveryFailed = false
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *DistributionService) dispatchDueTasks(ctx context.Context, now time.Time) error {
	var tasks []DistributionTask
	if err := s.db.WithContext(ctx).Select("id").Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", statusPending, now).Order("id ASC").Limit(100).Find(&tasks).Error; err != nil {
		return err
	}
	for _, task := range tasks {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.executeTaskContext(ctx, task.ID)
	}
	return nil
}
