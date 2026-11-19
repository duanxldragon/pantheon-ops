package filedist

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TransferEngine manages file transfer tasks.
type TransferEngine struct {
	db             *gorm.DB
	storageService *StorageService
	workers        int
	taskQueue      chan *TransferTask
	stopCh         chan struct{}
	wg             sync.WaitGroup
	stopOnce       sync.Once
	startOnce      sync.Once
	ctx            context.Context
	cancel         context.CancelFunc
	activeMu       sync.Mutex
	active         map[uint64]*activeTransfer
	upload         func(context.Context, SFTPConfig, string, string) error
}

type activeTransfer struct {
	taskID uint64
	cancel context.CancelFunc
}

// TransferTask represents a file transfer task.
type TransferTask struct {
	ID              uint64
	TaskID          uint64
	FileID          uint64
	TargetHost      string
	TargetPort      int
	TargetPath      string
	Username        string
	Password        string
	PrivateKey      string
	HostFingerprint string
	Status          string
	Progress        int
	Error           string
	StartTime       time.Time
	FinishTime      *time.Time
	RetryCount      int
	MaxRetries      int
	// Optional only for legacy all/admin tasks; stable-ID tasks require this
	// live check before every queued transfer attempt and local-file access.
	Authorize func(context.Context) error
}

// NewTransferEngine creates a new TransferEngine.
func NewTransferEngine(db *gorm.DB, storageService *StorageService, workers int) *TransferEngine {
	if workers <= 0 {
		workers = 5
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &TransferEngine{
		db:             db,
		storageService: storageService,
		workers:        workers,
		taskQueue:      make(chan *TransferTask, 100),
		stopCh:         make(chan struct{}),
		ctx:            ctx,
		cancel:         cancel,
		active:         make(map[uint64]*activeTransfer),
		upload:         uploadSFTP,
	}
}

// Start begins processing transfer tasks.
func (e *TransferEngine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		for i := 0; i < e.workers; i++ {
			e.wg.Add(1)
			go e.worker(e.ctx, i)
		}
		go func() {
			select {
			case <-ctx.Done():
				e.Stop()
			case <-e.stopCh:
			}
		}()
	})
}

// Stop stops the transfer engine.
func (e *TransferEngine) Stop() {
	e.stopOnce.Do(func() {
		e.cancel()
		close(e.stopCh)
	})
	e.wg.Wait()
}

// SubmitTask submits a new transfer task.
func (e *TransferEngine) SubmitTask(task *TransferTask) error {
	if task == nil {
		return fmt.Errorf("invalid_task")
	}
	if e.ctx.Err() != nil {
		return fmt.Errorf("engine_stopped")
	}
	select {
	case e.taskQueue <- task:
		return nil
	case <-e.stopCh:
		return fmt.Errorf("engine_stopped")
	default:
		return fmt.Errorf("queue_full")
	}
}

func (e *TransferEngine) worker(ctx context.Context, id int) {
	defer e.wg.Done()

	for {
		select {
		case task, ok := <-e.taskQueue:
			if !ok {
				return
			}
			e.processTask(ctx, task)
		case <-e.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (e *TransferEngine) processTask(ctx context.Context, task *TransferTask) {
	if task == nil {
		return
	}
	ctx, release, ok := e.beginTransfer(ctx, task)
	if !ok {
		return
	}
	defer release()
	if ctx.Err() != nil {
		e.finishTarget(task.ID, statusFailed, 0, executionInterruptedKey)
		return
	}
	if task.Authorize != nil {
		if err := task.Authorize(ctx); err != nil {
			e.finishTarget(task.ID, statusFailed, 0, safeDistributionFailure(err))
			return
		}
	}
	claimed, err := e.claimTarget(ctx, task)
	if err != nil {
		e.settleClaimFailure(ctx, task, err)
		return
	}
	if !claimed {
		return
	}

	// Get file path
	filePath, err := e.storageService.getFilePathContext(ctx, task.FileID)
	if err != nil {
		e.finishTarget(task.ID, statusFailed, 0, fmt.Sprintf("get_file_failed: %v", err))
		return
	}
	if ctx.Err() != nil {
		e.finishTarget(task.ID, statusFailed, 0, executionInterruptedKey)
		return
	}

	// Create SFTP client
	sftpConfig := SFTPConfig{
		Host:            task.TargetHost,
		Port:            task.TargetPort,
		Username:        task.Username,
		Password:        task.Password,
		PrivateKey:      task.PrivateKey,
		HostFingerprint: task.HostFingerprint,
	}

	if err := e.upload(ctx, sftpConfig, filePath, task.TargetPath); err != nil {
		e.handleTransferErrorContext(ctx, task, "business.filedist.task.transferFailed", release)
		return
	}
	if ctx.Err() == nil {
		e.finishTarget(task.ID, statusCompleted, 100, "")
	}
}

func uploadSFTP(ctx context.Context, config SFTPConfig, localPath, remotePath string) error {
	client := NewSFTPClient(config)
	defer func() { _ = client.Close() }()
	if err := client.ConnectContext(ctx); err != nil {
		return err
	}
	return client.UploadContext(ctx, localPath, remotePath)
}

// Register before claiming the target, so cancellation either sees this
// operation or wins the persistent parent/target claim before IO can start.
func (e *TransferEngine) beginTransfer(ctx context.Context, task *TransferTask) (context.Context, func(), bool) {
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(e.ctx, cancel)
	operation := &activeTransfer{taskID: task.TaskID, cancel: cancel}
	e.activeMu.Lock()
	if _, exists := e.active[task.ID]; exists {
		e.activeMu.Unlock()
		stop()
		cancel()
		return nil, nil, false
	}
	e.active[task.ID] = operation
	e.activeMu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			e.activeMu.Lock()
			delete(e.active, task.ID)
			e.activeMu.Unlock()
			stop()
			cancel()
		})
	}
	return ctx, release, true
}

// CancelDistribution stops only operations belonging to the cancelled parent.
func (e *TransferEngine) CancelDistribution(taskID uint64) {
	e.activeMu.Lock()
	defer e.activeMu.Unlock()
	for _, operation := range e.active {
		if operation.taskID == taskID {
			operation.cancel()
		}
	}
}

func (e *TransferEngine) claimTarget(ctx context.Context, task *TransferTask) (bool, error) {
	claimed := false
	err := e.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var parent DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&parent, task.TaskID).Error; err != nil {
			return err
		}
		if parent.Status != statusRunning {
			return nil
		}
		result := tx.Model(&DistributionTarget{}).Where("id = ? AND task_id = ? AND status IN ?", task.ID, task.TaskID, []string{statusPending, statusRetrying}).Updates(map[string]interface{}{
			"status": statusTransferring, "progress": 0, "error": "", "started_at": time.Now(),
		})
		claimed = result.RowsAffected == 1
		return result.Error
	})
	return claimed && err == nil, err
}

// No remote IO has begun when the claim fails, including an ambiguous COMMIT.
// Keep the consumed item owned until its terminal state is persisted or the
// lifecycle stops; a failed failure-write must not strand running work silently.
func (e *TransferEngine) settleClaimFailure(ctx context.Context, task *TransferTask, claimErr error) {
	slog.Warn("file distribution target claim failed", "task_id", task.TaskID, "target_id", task.ID, "error_type", fmt.Sprintf("%T", claimErr))
	for {
		if ctx.Err() != nil {
			return // Cancelled targets stay terminal; startup recovers stopped work.
		}
		err := e.updateTargetContext(ctx, task.ID, map[string]interface{}{
			"status": statusFailed, "progress": 0, "error": "business.filedist.task.transferFailed", "finished_at": time.Now(),
		})
		if err == nil {
			return
		}
		slog.Error("file distribution claim failure persistence will retry", "task_id", task.TaskID, "target_id", task.ID, "error_type", fmt.Sprintf("%T", err))
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (e *TransferEngine) handleTransferError(task *TransferTask, errorMsg string) {
	e.handleTransferErrorContext(context.Background(), task, errorMsg, nil)
}

func (e *TransferEngine) handleTransferErrorContext(ctx context.Context, task *TransferTask, errorMsg string, release func()) {
	if ctx.Err() != nil || e.ctx.Err() != nil {
		e.finishTarget(task.ID, statusFailed, 0, executionInterruptedKey)
		return
	}
	// Check if should retry
	if task.RetryCount < task.MaxRetries {
		task.RetryCount++
		e.updateTarget(task.ID, map[string]interface{}{
			"status":      statusRetrying,
			"progress":    0,
			"error":       errorMsg,
			"retry_count": task.RetryCount,
		})

		// Re-queue after delay
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			e.finishTarget(task.ID, statusFailed, 0, executionInterruptedKey)
			return
		case <-e.ctx.Done():
			e.finishTarget(task.ID, statusFailed, 0, executionInterruptedKey)
			return
		}
		if release != nil {
			release()
		}
		if err := e.SubmitTask(task); err != nil {
			e.finishTarget(task.ID, statusFailed, 0, err.Error())
		}
	} else {
		e.finishTarget(task.ID, statusFailed, 0, errorMsg)
	}
}

func (e *TransferEngine) updateTargetStatus(targetID uint64, status string, progress int, errorMsg string) {
	updates := map[string]interface{}{"status": status, "progress": progress}
	if errorMsg != "" {
		updates["error"] = errorMsg
	}
	e.updateTarget(targetID, updates)
}

func (e *TransferEngine) finishTarget(targetID uint64, status string, progress int, errorMsg string) {
	e.updateTarget(targetID, map[string]interface{}{
		"status":      status,
		"progress":    progress,
		"error":       errorMsg,
		"finished_at": time.Now(),
	})
}

func (e *TransferEngine) updateTarget(targetID uint64, updates map[string]interface{}) {
	_ = e.updateTargetContext(context.Background(), targetID, updates)
}

func (e *TransferEngine) updateTargetContext(ctx context.Context, targetID uint64, updates map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return e.db.WithContext(ctx).Model(&DistributionTarget{}).Where("id = ? AND status IN ?", targetID, unfinishedTargetStatuses()).Updates(updates).Error
}
