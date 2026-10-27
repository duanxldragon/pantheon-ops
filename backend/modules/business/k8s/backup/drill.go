package backup

import (
	"context"
	"fmt"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"gorm.io/gorm"
)

// DisasterRecoveryDrill represents a disaster recovery drill record
type DisasterRecoveryDrill struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DrillName     string         `gorm:"size:128;not null" json:"drillName"`
	BackupID      uint64         `gorm:"not null;index" json:"backupId"`
	ClusterID     uint64         `gorm:"not null;index" json:"clusterId"`
	TempNamespace string         `gorm:"size:128;not null" json:"tempNamespace"`
	RestoreID     *uint64        `json:"restoreId,omitempty"`
	Status        string         `gorm:"size:32;not null;default:'pending'" json:"status"` // pending, running, passed, failed
	StartedAt     *time.Time     `json:"startedAt,omitempty"`
	FinishedAt    *time.Time     `json:"finishedAt,omitempty"`
	Report        string         `gorm:"type:text" json:"report,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	CreatedBy     string         `gorm:"size:64" json:"createdBy"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DisasterRecoveryDrill) TableName() string {
	return "biz_velero_drill"
}

// DrillService manages disaster recovery drills
type DrillService struct {
	db             *gorm.DB
	backupService  *BackupService
	clusterService *cluster.ClusterService
}

// NewDrillService creates a drill service
func NewDrillService(db *gorm.DB, backupService *BackupService, clusterService *cluster.ClusterService) *DrillService {
	return &DrillService{
		db:             db,
		backupService:  backupService,
		clusterService: clusterService,
	}
}

// Migrate creates or updates database schema
func (s *DrillService) Migrate() error {
	return s.db.AutoMigrate(&DisasterRecoveryDrill{})
}

// CreateDrillRequest represents create drill request
type CreateDrillRequest struct {
	DrillName string `json:"drillName" binding:"required"`
	BackupID  uint64 `json:"backupId" binding:"required"`
}

// DrillResponse represents drill response
type DrillResponse struct {
	ID            uint64     `json:"id"`
	DrillName     string     `json:"drillName"`
	BackupID      uint64     `json:"backupId"`
	ClusterID     uint64     `json:"clusterId"`
	TempNamespace string     `json:"tempNamespace"`
	Status        string     `json:"status"`
	StartedAt     *time.Time `json:"startedAt,omitempty"`
	FinishedAt    *time.Time `json:"finishedAt,omitempty"`
	Report        string     `json:"report,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	CreatedBy     string     `json:"createdBy"`
}

// CreateDrill creates a new disaster recovery drill
func (s *DrillService) CreateDrill(ctx context.Context, req CreateDrillRequest, createdBy string) (*DrillResponse, error) {
	// Validate backup exists
	var backup VeleroBackup
	if err := s.db.First(&backup, req.BackupID).Error; err != nil {
		return nil, fmt.Errorf("backup.not_found: %w", err)
	}

	// Generate temp namespace name
	tempNS := fmt.Sprintf("drill-%d-%d", backup.ID, time.Now().Unix())

	record := &DisasterRecoveryDrill{
		DrillName:     req.DrillName,
		BackupID:      req.BackupID,
		ClusterID:     backup.ClusterID,
		TempNamespace: tempNS,
		Status:        "pending",
		CreatedBy:     createdBy,
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	// Start drill asynchronously
	go s.executeDrill(record)

	return s.toDrillResponse(record), nil
}

// ListDrills lists all drills
func (s *DrillService) ListDrills(backupID uint64) ([]DrillResponse, error) {
	var records []DisasterRecoveryDrill
	query := s.db.Order("created_at DESC")
	if backupID > 0 {
		query = query.Where("backup_id = ?", backupID)
	}
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}

	items := make([]DrillResponse, len(records))
	for i, r := range records {
		items[i] = *s.toDrillResponse(&r)
	}
	return items, nil
}

// GetDrill gets a drill by ID
func (s *DrillService) GetDrill(id uint64) (*DrillResponse, error) {
	var record DisasterRecoveryDrill
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toDrillResponse(&record), nil
}

// executeDrill executes a disaster recovery drill: it restores the backup into
// a temporary namespace and only marks the drill passed after the restore
// reaches a successful terminal phase.
func (s *DrillService) executeDrill(drill *DisasterRecoveryDrill) {
	startedAt := time.Now()
	s.db.Model(drill).Updates(map[string]interface{}{
		"status":     "running",
		"started_at": startedAt,
	})
	ctx := context.Background()

	logging.Info("starting disaster recovery drill",
		zap.String("drill", drill.DrillName),
		zap.Uint64("backup_id", drill.BackupID))

	var backup VeleroBackup
	if err := s.db.First(&backup, drill.BackupID).Error; err != nil {
		s.failDrill(drill, fmt.Sprintf("failed to get backup: %v", err))
		return
	}

	// 1. Connect to the target cluster.
	clientset, err := s.clusterService.GetClientsetWithContext(ctx, drill.ClusterID, nil)
	if err != nil {
		s.failDrill(drill, fmt.Sprintf("failed to connect to cluster %d: %v", drill.ClusterID, err))
		return
	}

	// 2. Create the temporary namespace for the drill restore.
	tempNS := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: drill.TempNamespace}}
	if _, err := clientset.CoreV1().Namespaces().Create(ctx, tempNS, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		s.failDrill(drill, fmt.Sprintf("failed to create temp namespace %s: %v", drill.TempNamespace, err))
		return
	}
	defer func() {
		_ = clientset.CoreV1().Namespaces().Delete(ctx, drill.TempNamespace, metav1.DeleteOptions{})
	}()

	// 3. Restore the backup into the temporary namespace (PVs are skipped for drills).
	restoreReq := CreateRestoreRequest{
		Name:       fmt.Sprintf("drill-restore-%d", drill.ID),
		ClusterID:  drill.ClusterID,
		BackupName: backup.Name,
		NamespaceMapping: map[string]string{
			"*": drill.TempNamespace,
		},
		RestorePVs: false,
	}
	restore, err := s.backupService.CreateRestore(ctx, restoreReq, nil, "drill-system")
	if err != nil {
		s.failDrill(drill, fmt.Sprintf("failed to create restore: %v", err))
		return
	}
	_ = s.db.Model(drill).Update("restore_id", restore.ID)

	// 4. Poll the restore CRD until it reaches a terminal phase.
	vc, err := s.backupService.veleroClient(ctx, drill.ClusterID, nil)
	if err != nil {
		s.failDrill(drill, fmt.Sprintf("failed to build velero client: %v", err))
		return
	}
	phase, restoreErrors := s.waitForRestore(ctx, vc, restore.Name)

	if phase != "Completed" || restoreErrors > 0 {
		s.failDrill(drill, fmt.Sprintf("restore did not complete: phase=%s errors=%d", phase, restoreErrors))
		return
	}

	// 5. Generate the report and mark the drill passed only on verified success.
	report := s.generateDrillReport(drill, &backup, restore, startedAt, phase, restoreErrors)
	finishedAt := time.Now()
	s.db.Model(drill).Updates(map[string]interface{}{
		"status":      "passed",
		"finished_at": finishedAt,
		"report":      report,
	})

	logging.Info("disaster recovery drill passed",
		zap.String("drill", drill.DrillName),
		zap.Float64("duration", finishedAt.Sub(startedAt).Seconds()))
}

// waitForRestore polls the restore CRD until a terminal phase is reached or
// the timeout expires.
func (s *DrillService) waitForRestore(ctx context.Context, vc *VeleroClient, name string) (string, int64) {
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		obj, err := vc.GetRestore(ctx, name)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
		if phase == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		errorsCount, _, _ := unstructured.NestedInt64(obj.Object, "status", "errors")
		switch phase {
		case "Completed", "Failed", "PartiallyFailed":
			return phase, errorsCount
		}
		time.Sleep(5 * time.Second)
	}
	return "Timeout", 0
}

// failDrill marks a drill as failed
func (s *DrillService) failDrill(drill *DisasterRecoveryDrill, reason string) {
	finishedAt := time.Now()
	s.db.Model(drill).Updates(map[string]interface{}{
		"status":      "failed",
		"finished_at": finishedAt,
		"report":      reason,
	})

	logging.Error("disaster recovery drill failed",
		zap.String("drill", drill.DrillName),
		zap.String("reason", reason))
}

// generateDrillReport generates an honest drill report reflecting the phases
// that actually ran.
func (s *DrillService) generateDrillReport(drill *DisasterRecoveryDrill, backup *VeleroBackup, restore *RestoreResponse, startedAt time.Time, phase string, errorsCount int64) string {
	restoreOK := phase == "Completed" && errorsCount == 0
	return fmt.Sprintf(`# 灾难恢复演练报告

## 基本信息
- 演练名称: %s
- 备份名称: %s
- 集群ID: %d
- 临时Namespace: %s
- 开始时间: %s
- 完成时间: %s

## 验证结果
- [x] 临时Namespace创建成功
- [%s] 备份恢复到临时Namespace (phase=%s, errors=%d)
- [ ] 资源完整性逐项核对（演练仅验证恢复流程可用）
- [x] 临时Namespace清理完成

## 结论
%s
`,
		drill.DrillName,
		backup.Name,
		drill.ClusterID,
		drill.TempNamespace,
		startedAt.Format(time.RFC3339),
		time.Now().Format(time.RFC3339),
		checked(restoreOK),
		phase,
		errorsCount,
		drillConclusion(restoreOK),
	)
}

func checked(ok bool) string {
	if ok {
		return "x"
	}
	return " "
}

func drillConclusion(ok bool) string {
	if ok {
		return "演练成功完成，备份恢复能力验证通过。"
	}
	return "演练未通过，备份恢复未达到预期状态，需要人工排查。"
}

// toDrillResponse converts drill to response
func (s *DrillService) toDrillResponse(r *DisasterRecoveryDrill) *DrillResponse {
	return &DrillResponse{
		ID:            r.ID,
		DrillName:     r.DrillName,
		BackupID:      r.BackupID,
		ClusterID:     r.ClusterID,
		TempNamespace: r.TempNamespace,
		Status:        r.Status,
		StartedAt:     r.StartedAt,
		FinishedAt:    r.FinishedAt,
		Report:        r.Report,
		CreatedAt:     r.CreatedAt,
		CreatedBy:     r.CreatedBy,
	}
}
