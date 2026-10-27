package filedist

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
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
}

// TransferTask represents a file transfer task.
type TransferTask struct {
	ID              uint64
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
}

// NewTransferEngine creates a new TransferEngine.
func NewTransferEngine(db *gorm.DB, storageService *StorageService, workers int) *TransferEngine {
	if workers <= 0 {
		workers = 5
	}
	return &TransferEngine{
		db:             db,
		storageService: storageService,
		workers:        workers,
		taskQueue:      make(chan *TransferTask, 100),
		stopCh:         make(chan struct{}),
	}
}

// Start begins processing transfer tasks.
func (e *TransferEngine) Start(ctx context.Context) {
	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	// Watch for context cancellation
	go func() {
		<-ctx.Done()
		e.Stop()
	}()
}

// Stop stops the transfer engine.
func (e *TransferEngine) Stop() {
	e.stopOnce.Do(func() {
		close(e.stopCh)
	})
	e.wg.Wait()
}

// SubmitTask submits a new transfer task.
func (e *TransferEngine) SubmitTask(task *TransferTask) error {
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
	e.updateTarget(task.ID, map[string]interface{}{
		"status":     statusTransferring,
		"progress":   0,
		"error":      "",
		"started_at": time.Now(),
	})

	// Get file path
	filePath, err := e.storageService.GetFilePath(task.FileID)
	if err != nil {
		e.finishTarget(task.ID, statusFailed, 0, fmt.Sprintf("get_file_failed: %v", err))
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

	client := NewSFTPClient(sftpConfig)
	if err := client.Connect(); err != nil {
		e.handleTransferError(task, fmt.Sprintf("connect_failed: %v", err))
		return
	}
	defer func() { _ = client.Close() }()

	// Upload file
	e.updateTargetStatus(task.ID, statusTransferring, 50, "")

	if err := client.Upload(filePath, task.TargetPath); err != nil {
		e.handleTransferError(task, fmt.Sprintf("upload_failed: %v", err))
		return
	}

	// Success
	e.finishTarget(task.ID, statusCompleted, 100, "")
}

func (e *TransferEngine) handleTransferError(task *TransferTask, errorMsg string) {
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
		time.Sleep(5 * time.Second)
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
	_ = e.db.Model(&DistributionTarget{}).Where("id = ?", targetID).Updates(updates).Error
}
