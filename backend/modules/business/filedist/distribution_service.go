package filedist

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SQL fragments shared by the distribution target queries below.
const (
	whereTaskByID      = "task_id = ?"
	whereTaskAndStatus = "task_id = ? AND status = ?"
)

// DistributionService manages file distribution tasks.
type DistributionService struct {
	db               *gorm.DB
	storageService   *StorageService
	transferEngine   *TransferEngine
	targetSelection  cmdb.TargetSelectionCapability
	authorizer       FileDistributionAuthorizer
	runtimeMu        sync.Mutex
	runtimeCtx       context.Context
	runtimeCancel    context.CancelFunc
	runtimeDone      chan struct{}
	runtimeLeaseDone chan struct{}
	releaseLease     func()
	runtimeStopped   bool
}

// NewDistributionService creates a new DistributionService.
func NewDistributionService(db *gorm.DB, storageService *StorageService, transferEngine *TransferEngine) *DistributionService {
	return &DistributionService{
		db:             db,
		storageService: storageService,
		transferEngine: transferEngine,
	}
}

// Migrate creates or updates database schema.
func (s *DistributionService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&DistributionTask{}, &DistributionTarget{})
}

// CreateTask creates a new distribution task.
func (s *DistributionService) CreateTask(dto DistributionTaskDTO, createdBy string) (*DistributionTaskResponse, error) {
	// Verify file exists
	file, err := s.storageService.GetFileByID(dto.FileID)
	if err != nil {
		return nil, fmt.Errorf("file.not_found: %w", err)
	}

	// Keep execution credentials only in the encrypted target rows. Copy the
	// request slice so snapshot sanitization cannot mutate caller-owned input.
	snapshotTargets := append([]TargetConfig(nil), dto.Targets...)
	for i := range snapshotTargets {
		snapshotTargets[i].Password = ""
		snapshotTargets[i].PrivateKey = ""
	}
	targetsJSON, _ := json.Marshal(snapshotTargets)

	task := &DistributionTask{
		Name:         dto.Name,
		FileID:       dto.FileID,
		Targets:      datatypes.JSON(targetsJSON),
		Status:       statusPending,
		TotalTargets: len(dto.Targets),
		ScheduledAt:  dto.ScheduledAt,
		CreatedBy:    createdBy,
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		for _, target := range dto.Targets {
			encryptedPassword, err := secrets.Encrypt(target.Password)
			if err != nil {
				return fmt.Errorf("encrypt_password: %w", err)
			}
			encryptedKey, err := secrets.Encrypt(target.PrivateKey)
			if err != nil {
				return fmt.Errorf("encrypt_private_key: %w", err)
			}

			targetRecord := &DistributionTarget{
				TaskID:          task.ID,
				Host:            target.Host,
				Port:            target.Port,
				TargetPath:      target.TargetPath,
				Username:        target.Username,
				Password:        encryptedPassword,
				PrivateKey:      encryptedKey,
				HostFingerprint: target.HostFingerprint,
				Status:          statusPending,
				MaxRetries:      target.MaxRetries,
			}
			if targetRecord.Port == 0 {
				targetRecord.Port = 22
			}
			if err := tx.Create(targetRecord).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Start task if not scheduled
	if dto.ScheduledAt == nil || dto.ScheduledAt.Before(time.Now()) {
		go s.executeTask(task.ID)
	}

	return s.toTaskResponse(task, file), nil
}

// ListTasks returns distribution tasks with filtering.
func (s *DistributionService) ListTasks(query TaskListQuery) (*TaskListResponse, error) {
	db := s.db.Model(&DistributionTask{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.FileID > 0 {
		db = db.Where("file_id = ?", query.FileID)
	}
	if query.Keyword != "" {
		db = db.Where("name LIKE ?", "%"+query.Keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	var tasks []DistributionTask
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(query.PageSize).Find(&tasks).Error; err != nil {
		return nil, err
	}

	// Load file names
	fileMap := make(map[uint64]*FileRecordResponse)
	for _, t := range tasks {
		if _, exists := fileMap[t.FileID]; !exists {
			file, err := s.storageService.GetFileByID(t.FileID)
			if err == nil {
				fileMap[t.FileID] = file
			}
		}
	}

	items := make([]DistributionTaskResponse, len(tasks))
	for i, t := range tasks {
		items[i] = *s.toTaskResponse(&t, fileMap[t.FileID])
	}

	return &TaskListResponse{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

// GetTaskByID retrieves a task by ID.
func (s *DistributionService) GetTaskByID(id uint64) (*DistributionTaskResponse, error) {
	var task DistributionTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}

	file, _ := s.storageService.GetFileByID(task.FileID)

	return s.toTaskResponse(&task, file), nil
}

// StartTask starts a pending task. Tasks created without a schedule are
// started automatically; this method is for explicit UI retries/start actions.
func (s *DistributionService) StartTask(id uint64) error {
	var task DistributionTask
	if err := s.db.First(&task, id).Error; err != nil {
		return err
	}
	if task.Status != statusPending {
		return fmt.Errorf("task_not_pending")
	}
	if len(task.Selection) != 0 {
		if err := s.validateTaskAction(id, "start"); err != nil {
			return err
		}
	}
	if task.ScheduledAt != nil && task.ScheduledAt.After(time.Now()) {
		return errors.New("business.filedist.task.notDue")
	}
	go s.executeTask(id)
	return nil
}

// GetTaskTargets retrieves all targets for a task.
func (s *DistributionService) GetTaskTargets(taskID uint64) ([]DistributionTargetResponse, error) {
	var targets []DistributionTarget
	if err := s.db.Where(whereTaskByID, taskID).Order("created_at ASC").Find(&targets).Error; err != nil {
		return nil, err
	}

	items := make([]DistributionTargetResponse, len(targets))
	for i, t := range targets {
		items[i] = *s.toTargetResponse(&t)
	}

	return items, nil
}

// GetTaskProgress returns a compact progress view used by the task detail UI.
func (s *DistributionService) GetTaskProgress(taskID uint64) (map[string]interface{}, error) {
	var task DistributionTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, err
	}
	targets, err := s.GetTaskTargets(taskID)
	if err != nil {
		return nil, err
	}
	running := 0
	for _, target := range targets {
		if target.Status == statusTransferring || target.Status == statusRetrying {
			running++
		}
	}
	return map[string]interface{}{
		"taskId":          task.ID,
		"status":          task.Status,
		"overallProgress": task.Progress,
		"totalHosts":      task.TotalTargets,
		"successHosts":    task.SuccessCount,
		"failedHosts":     task.FailedCount,
		"runningHosts":    running,
		"hosts":           targets,
	}, nil
}

// CancelTask cancels a running task.
func (s *DistributionService) CancelTask(id uint64) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var task DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, id).Error; err != nil {
			return err
		}
		if task.Status == statusCancelled {
			return nil
		}
		if task.Status != statusPending && task.Status != statusRunning {
			return errors.New("business.filedist.task.notCancellable")
		}
		now := time.Now()
		if err := tx.Model(&task).Updates(map[string]interface{}{"status": statusCancelled, "finished_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&DistributionTarget{}).Where("task_id = ? AND status IN ?", id, unfinishedTargetStatuses()).Updates(map[string]interface{}{"status": statusCancelled, "finished_at": now}).Error; err != nil {
			return err
		}
		var completed, failed int64
		if err := tx.Model(&DistributionTarget{}).Where(whereTaskAndStatus, id, statusCompleted).Count(&completed).Error; err != nil {
			return err
		}
		if err := tx.Model(&DistributionTarget{}).Where(whereTaskAndStatus, id, statusFailed).Count(&failed).Error; err != nil {
			return err
		}
		return tx.Model(&task).Updates(map[string]interface{}{"success_count": completed, "failed_count": failed}).Error
	})
	if err == nil && s.transferEngine != nil {
		s.transferEngine.CancelDistribution(id)
	}
	return err
}

// RetryTask retries a failed task.
func (s *DistributionService) RetryTask(id uint64) error {
	var task DistributionTask
	if err := s.db.First(&task, id).Error; err != nil {
		return err
	}

	if task.Status != statusFailed {
		return fmt.Errorf("task_not_failed")
	}
	if len(task.Selection) != 0 {
		if err := s.validateTaskAction(id, "retry"); err != nil {
			return err
		}
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		var current DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
			return err
		}
		if current.Status != statusFailed {
			return fmt.Errorf("task_not_failed")
		}
		if err := tx.Model(&current).Updates(map[string]interface{}{
			"status": statusPending, "progress": 0, "success_count": 0, "failed_count": 0,
			"error": "", "started_at": nil, "finished_at": nil,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&DistributionTarget{}).Where(whereTaskAndStatus, id, statusFailed).Updates(map[string]interface{}{
			"status": statusPending, "progress": 0, "error": "", "retry_count": 0,
			"started_at": nil, "finished_at": nil,
		}).Error
	}); err != nil {
		return err
	}

	// Restart execution
	go s.executeTask(id)

	return nil
}

// DeleteTask removes a completed, failed, cancelled, or pending task. Running
// tasks are protected to avoid deleting records while workers are updating
// targets.
func (s *DistributionService) DeleteTask(id uint64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var task DistributionTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&task, id).Error; err != nil {
			return err
		}
		if task.Status == statusRunning {
			return fmt.Errorf("task_running")
		}
		if err := tx.Where(whereTaskByID, id).Delete(&DistributionTarget{}).Error; err != nil {
			return err
		}
		return tx.Delete(&task).Error
	})
}

// executeTask executes a distribution task.
func (s *DistributionService) executeTask(taskID uint64) {
	s.executeTaskContext(s.executionContext(), taskID)
}

func (s *DistributionService) executeTaskContext(ctx context.Context, taskID uint64) {
	if ctx.Err() != nil {
		return
	}
	db := s.db.WithContext(ctx)
	var subject DistributionTask
	if err := db.First(&subject, taskID).Error; err != nil {
		return
	}
	if subject.Status != statusPending || (subject.ScheduledAt != nil && subject.ScheduledAt.After(time.Now())) {
		return
	}
	if len(subject.Selection) != 0 {
		if err := s.validateTaskAction(taskID, "execute"); err != nil {
			s.markTaskFailed(taskID, safeDistributionFailure(err))
			return
		}
	}
	now := time.Now()
	claim := db.Model(&DistributionTask{}).Where("id = ? AND status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)", taskID, statusPending, now).Updates(map[string]interface{}{
		"status":     statusRunning,
		"started_at": now,
	})
	if claim.Error != nil || claim.RowsAffected != 1 {
		return
	}

	// Get all pending targets
	var targets []DistributionTarget
	if err := db.Where(whereTaskAndStatus, taskID, statusPending).Find(&targets).Error; err != nil {
		s.markTaskFailed(taskID, err.Error())
		return
	}

	// Submit transfer tasks
	for _, target := range targets {
		if ctx.Err() != nil {
			return
		}
		// Decrypt credentials
		password, err := secrets.Decrypt(target.Password)
		if err != nil {
			s.transferEngine.finishTarget(target.ID, statusFailed, 0, "business.filedist.task.transferFailed")
			continue
		}
		privateKey, err := secrets.Decrypt(target.PrivateKey)
		if err != nil {
			s.transferEngine.finishTarget(target.ID, statusFailed, 0, "business.filedist.task.transferFailed")
			continue
		}

		transferTask := &TransferTask{
			ID:              target.ID,
			TaskID:          taskID,
			FileID:          subject.FileID,
			TargetHost:      target.Host,
			TargetPort:      target.Port,
			TargetPath:      target.TargetPath,
			Username:        target.Username,
			Password:        password,
			PrivateKey:      privateKey,
			HostFingerprint: target.HostFingerprint,
			MaxRetries:      target.MaxRetries,
		}
		if len(subject.Selection) != 0 {
			transferTask.Authorize = func(ctx context.Context) error { return s.authorizeQueuedTarget(ctx, taskID) }
		}

		if err := s.transferEngine.SubmitTask(transferTask); err != nil {
			// Mark target as failed if can't submit
			s.transferEngine.finishTarget(target.ID, statusFailed, 0, "business.filedist.task.transferFailed")
		}
	}

	// Monitor task progress (in background)
	go s.monitorTaskProgressContext(ctx, taskID)
}

func (s *DistributionService) monitorTaskProgress(taskID uint64) {
	s.monitorTaskProgressContext(s.executionContext(), taskID)
}

func (s *DistributionService) monitorTaskProgressContext(ctx context.Context, taskID uint64) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}

		// Check if task is still running
		var task DistributionTask
		if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
			return
		}

		if task.Status != statusRunning {
			return
		}

		// Count target statuses
		var stats struct {
			Total     int64
			Completed int64
			Failed    int64
		}

		db := s.db.WithContext(ctx)
		for _, query := range []*gorm.DB{
			db.Model(&DistributionTarget{}).Where(whereTaskByID, taskID).Count(&stats.Total),
			db.Model(&DistributionTarget{}).Where(whereTaskAndStatus, taskID, statusCompleted).Count(&stats.Completed),
			db.Model(&DistributionTarget{}).Where(whereTaskAndStatus, taskID, statusFailed).Count(&stats.Failed),
		} {
			if query.Error != nil {
				return
			}
		}

		// Calculate progress
		progress := 0
		if stats.Total > 0 {
			progress = int((stats.Completed + stats.Failed) * 100 / stats.Total)
		}

		// Update task
		updates := map[string]interface{}{
			"progress":      progress,
			"success_count": stats.Completed,
			"failed_count":  stats.Failed,
		}

		// Check if all targets are done
		if stats.Completed+stats.Failed >= stats.Total {
			now := time.Now()
			updates["finished_at"] = now

			if stats.Failed == 0 {
				updates["status"] = statusCompleted
			} else if stats.Completed == 0 {
				updates["status"] = statusFailed
			} else {
				updates["status"] = "partial"
			}

			_ = db.Model(&DistributionTask{}).Where("id = ? AND status = ?", taskID, statusRunning).Updates(updates).Error
			return
		}

		_ = db.Model(&DistributionTask{}).Where("id = ? AND status = ?", taskID, statusRunning).Updates(updates).Error
	}
}

func (s *DistributionService) markTaskFailed(taskID uint64, errorMsg string) {
	now := time.Now()
	_ = s.db.Model(&DistributionTask{}).Where("id = ? AND status IN ?", taskID, []string{statusPending, statusRunning}).Updates(map[string]interface{}{
		"status":      statusFailed,
		"error":       errorMsg,
		"finished_at": now,
	}).Error
}

func (s *DistributionService) toTaskResponse(t *DistributionTask, file *FileRecordResponse) *DistributionTaskResponse {
	resp := &DistributionTaskResponse{
		ID:           t.ID,
		Name:         t.Name,
		FileID:       t.FileID,
		Status:       t.Status,
		Progress:     t.Progress,
		TotalTargets: t.TotalTargets,
		SuccessCount: t.SuccessCount,
		FailedCount:  t.FailedCount,
		Error:        t.Error,
		ScheduledAt:  t.ScheduledAt,
		StartedAt:    t.StartedAt,
		FinishedAt:   t.FinishedAt,
		CreatedAt:    t.CreatedAt,
		CreatedBy:    t.CreatedBy,
	}

	if file != nil {
		resp.FileName = file.Name
	}

	return resp
}

func (s *DistributionService) toTargetResponse(t *DistributionTarget) *DistributionTargetResponse {
	return &DistributionTargetResponse{
		ID:         t.ID,
		TaskID:     t.TaskID,
		Host:       t.Host,
		Port:       t.Port,
		TargetPath: t.TargetPath,
		Username:   t.Username,
		Status:     t.Status,
		Progress:   t.Progress,
		Error:      t.Error,
		RetryCount: t.RetryCount,
		MaxRetries: t.MaxRetries,
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
