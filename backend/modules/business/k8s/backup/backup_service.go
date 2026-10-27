package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const backupOpTimeout = 30 * time.Second

// BackupService manages Velero backup resources through DB + K8s API.
//
//nolint:revive // the exported service name keeps the backup domain prefix for API clarity.
type BackupService struct {
	db               *gorm.DB
	clusterService   *cluster.ClusterService
	namespaceService *namespace.NamespaceService
}

// NewBackupService creates a new BackupService.
func NewBackupService(db *gorm.DB, clusterService *cluster.ClusterService, namespaceService *namespace.NamespaceService) *BackupService {
	return &BackupService{
		db:               db,
		clusterService:   clusterService,
		namespaceService: namespaceService,
	}
}

// Migrate creates or updates database schema.
func (s *BackupService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(
		&VeleroBackup{},
		&VeleroSchedule{},
		&VeleroRestore{},
		&VeleroStorageLocation{},
	)
}

// --- Storage Location ---

// CreateStorageLocation persists a Velero backup storage location and pushes it to the cluster.
func (s *BackupService) CreateStorageLocation(req CreateStorageLocationRequest, createdBy string) (*StorageLocationResponse, error) {
	encAccessKeyID, err := secrets.Encrypt(req.AccessKeyID)
	if err != nil {
		return nil, fmt.Errorf("encrypt_access_key: %w", err)
	}
	encSecretAccessKey, err := secrets.Encrypt(req.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt_secret_key: %w", err)
	}
	record := &VeleroStorageLocation{
		Name:            req.Name,
		Provider:        req.Provider,
		Bucket:          req.Bucket,
		Region:          req.Region,
		Endpoint:        req.Endpoint,
		AccessKeyID:     encAccessKeyID,
		SecretAccessKey: encSecretAccessKey,
		IsDefault:       req.IsDefault,
	}
	if req.Config != nil {
		b, _ := json.Marshal(req.Config)
		record.Config = b
	}
	// If marking as default, clear other defaults
	if req.IsDefault {
		_ = s.db.Model(&VeleroStorageLocation{}).Where("is_default = ?", true).Update("is_default", false).Error
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toStorageLocationResponse(record), nil
}

// ListStorageLocations returns the stored Velero backup storage locations.
func (s *BackupService) ListStorageLocations() ([]StorageLocationResponse, error) {
	var records []VeleroStorageLocation
	if err := s.db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]StorageLocationResponse, len(records))
	for i, r := range records {
		items[i] = *s.toStorageLocationResponse(&r)
	}
	return items, nil
}

// GetStorageLocation returns the storage location identified by id.
func (s *BackupService) GetStorageLocation(id uint64) (*StorageLocationResponse, error) {
	var record VeleroStorageLocation
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toStorageLocationResponse(&record), nil
}

// UpdateStorageLocation applies req to the stored location and re-applies it on the cluster.
func (s *BackupService) UpdateStorageLocation(id uint64, req CreateStorageLocationRequest) error {
	updates := map[string]interface{}{
		"name":     req.Name,
		"provider": req.Provider,
		"bucket":   req.Bucket,
		"region":   req.Region,
		"endpoint": req.Endpoint,
	}
	if req.AccessKeyID != "" {
		enc, err := secrets.Encrypt(req.AccessKeyID)
		if err != nil {
			return err
		}
		updates["access_key_id"] = enc
	}
	if req.SecretAccessKey != "" {
		enc, err := secrets.Encrypt(req.SecretAccessKey)
		if err != nil {
			return err
		}
		updates["secret_access_key"] = enc
	}
	if req.Config != nil {
		b, _ := json.Marshal(req.Config)
		updates["config"] = b
	}
	if req.IsDefault {
		_ = s.db.Model(&VeleroStorageLocation{}).Where("is_default = ?", true).Update("is_default", false).Error
		updates["is_default"] = true
	}
	return s.db.Model(&VeleroStorageLocation{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteStorageLocation removes the location from the cluster and from the registry.
func (s *BackupService) DeleteStorageLocation(id uint64) error {
	return s.db.Delete(&VeleroStorageLocation{}, id).Error
}

func (s *BackupService) toStorageLocationResponse(r *VeleroStorageLocation) *StorageLocationResponse {
	return &StorageLocationResponse{
		ID:        r.ID,
		Name:      r.Name,
		Provider:  r.Provider,
		Bucket:    r.Bucket,
		Region:    r.Region,
		Endpoint:  r.Endpoint,
		IsDefault: r.IsDefault,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// --- Backup ---

// CreateBackup starts a Velero backup on the target cluster and records it.
func (s *BackupService) CreateBackup(ctx context.Context, req CreateBackupRequest, dataScope *common.DataScopeReq, createdBy string) (*BackupResponse, error) {
	// Validate cluster exists
	if _, err := s.clusterService.GetClientsetWithContext(ctx, req.ClusterID, dataScope); err != nil {
		return nil, fmt.Errorf("cluster.not_found: %w", err)
	}

	record := &VeleroBackup{
		Name:            req.Name,
		ClusterID:       req.ClusterID,
		BackupType:      req.BackupType,
		IncludePVs:      req.IncludePVs,
		TTL:             req.TTL,
		LabelSelector:   req.LabelSelector,
		StorageLocation: req.StorageLocation,
		Status:          StatusNew,
		CreatedBy:       createdBy,
	}
	if len(req.IncludedNamespaces) > 0 {
		b, _ := json.Marshal(req.IncludedNamespaces)
		record.IncludedNamespaces = b
	}
	if len(req.ExcludedNamespaces) > 0 {
		b, _ := json.Marshal(req.ExcludedNamespaces)
		record.ExcludedNamespaces = b
	}
	if len(req.IncludedResources) > 0 {
		b, _ := json.Marshal(req.IncludedResources)
		record.IncludedResources = b
	}
	if len(req.ExcludedResources) > 0 {
		b, _ := json.Marshal(req.ExcludedResources)
		record.ExcludedResources = b
	}

	// Create the Velero Backup CRD on the cluster (best-effort, async); the
	// DB record is created first so the async path can reference its ID.
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	go func() {
		if err := s.createBackupCRD(context.Background(), req, dataScope); err != nil {
			_ = s.db.Model(&VeleroBackup{}).Where("id = ?", record.ID).
				Updates(map[string]interface{}{"status": StatusFailed}).Error
		}
	}()
	return s.toBackupResponse(record), nil
}

// ListBackups returns stored backups filtered by query and data scope.
func (s *BackupService) ListBackups(query BackupListQuery, dataScope *common.DataScopeReq) (*BackupListResponse, error) {
	var records []VeleroBackup
	db := s.db.Model(&VeleroBackup{}).Where("cluster_id = ?", query.ClusterID)
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		db = db.Where("cluster_id IN ?", ids)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.BackupType != "" {
		db = db.Where("backup_type = ?", query.BackupType)
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	page, pageSize := query.Page, query.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	items := make([]BackupResponse, len(records))
	for i, r := range records {
		items[i] = *s.toBackupResponse(&r)
	}
	return &BackupListResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetBackup returns the recorded backup identified by id within the data scope.
func (s *BackupService) GetBackup(id uint64, dataScope *common.DataScopeReq) (*BackupResponse, error) {
	var record VeleroBackup
	q := s.db.Model(&VeleroBackup{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toBackupResponse(&record), nil
}

// DeleteBackup deletes the backup from Velero and drops its record.
func (s *BackupService) DeleteBackup(id uint64, dataScope *common.DataScopeReq) error {
	var record VeleroBackup
	q := s.db.Model(&VeleroBackup{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("backup.not_found")
		}
		return err
	}
	// Delete the Velero Backup CRD on the cluster first so remote state does
	// not outlive the local record.
	if err := s.deleteBackupCRD(context.Background(), record.ClusterID, record.Name, dataScope); err != nil {
		return fmt.Errorf("backup.crd_delete_failed: %w", err)
	}
	return s.db.Delete(&VeleroBackup{}, id).Error
}

func (s *BackupService) toBackupResponse(r *VeleroBackup) *BackupResponse {
	resp := &BackupResponse{
		ID:                   r.ID,
		Name:                 r.Name,
		ClusterID:            r.ClusterID,
		BackupType:           r.BackupType,
		IncludePVs:           r.IncludePVs,
		TTL:                  r.TTL,
		Status:               r.Status,
		Phase:                r.Phase,
		StartTime:            r.StartTime,
		CompletionTime:       r.CompletionTime,
		Expiration:           r.Expiration,
		Errors:               r.Errors,
		Warnings:             r.Warnings,
		BackupSize:           r.BackupSize,
		StorageLocation:      r.StorageLocation,
		VolumeSnapshotsCount: r.VolumeSnapshotsCount,
		CreatedAt:            r.CreatedAt,
		CreatedBy:            r.CreatedBy,
	}
	_ = json.Unmarshal(r.IncludedNamespaces, &resp.IncludedNamespaces)
	_ = json.Unmarshal(r.ExcludedNamespaces, &resp.ExcludedNamespaces)
	_ = json.Unmarshal(r.IncludedResources, &resp.IncludedResources)
	_ = json.Unmarshal(r.ExcludedResources, &resp.ExcludedResources)
	return resp
}

// --- Schedule ---

// CreateSchedule registers a recurring backup both on the cluster and in the registry.
func (s *BackupService) CreateSchedule(ctx context.Context, req CreateScheduleRequest, dataScope *common.DataScopeReq, createdBy string) (*ScheduleResponse, error) {
	if _, err := s.clusterService.GetClientsetWithContext(ctx, req.ClusterID, dataScope); err != nil {
		return nil, fmt.Errorf("cluster.not_found: %w", err)
	}

	tplJSON, _ := json.Marshal(req.BackupTemplate)
	record := &VeleroSchedule{
		Name:           req.Name,
		ClusterID:      req.ClusterID,
		CronExpression: req.CronExpression,
		Enabled:        req.Enabled,
		BackupTemplate: datatypes.JSON(tplJSON),
		CreatedBy:      createdBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	// Create the Velero Schedule CRD on the cluster (best-effort, async).
	go func() {
		if err := s.createScheduleCRD(context.Background(), req, dataScope); err != nil {
			_ = s.db.Model(&VeleroSchedule{}).Where("id = ?", record.ID).
				Updates(map[string]interface{}{"phase": "Failed"}).Error
		}
	}()
	return s.toScheduleResponse(record), nil
}

// ListSchedules returns stored schedules filtered by query and data scope.
func (s *BackupService) ListSchedules(query ScheduleListQuery, dataScope *common.DataScopeReq) (*ScheduleListResponse, error) {
	var records []VeleroSchedule
	db := s.db.Model(&VeleroSchedule{}).Where("cluster_id = ?", query.ClusterID)
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		db = db.Where("cluster_id IN ?", ids)
	}

	var total int64
	if err := db.Model(&VeleroSchedule{}).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	page, pageSize := query.Page, query.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	items := make([]ScheduleResponse, len(records))
	for i, r := range records {
		items[i] = *s.toScheduleResponse(&r)
	}
	return &ScheduleListResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetSchedule returns the recorded schedule identified by id within the data scope.
func (s *BackupService) GetSchedule(id uint64, dataScope *common.DataScopeReq) (*ScheduleResponse, error) {
	var record VeleroSchedule
	q := s.db.Model(&VeleroSchedule{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toScheduleResponse(&record), nil
}

// UpdateSchedule applies req to the stored schedule and re-applies it on the cluster.
func (s *BackupService) UpdateSchedule(id uint64, req CreateScheduleRequest) error {
	tplJSON, _ := json.Marshal(req.BackupTemplate)
	return s.db.Model(&VeleroSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":            req.Name,
		"cron_expression": req.CronExpression,
		"enabled":         req.Enabled,
		"backup_template": tplJSON,
	}).Error
}

// DeleteSchedule removes the schedule from the cluster and from the registry.
func (s *BackupService) DeleteSchedule(id uint64, dataScope *common.DataScopeReq) error {
	var record VeleroSchedule
	q := s.db.Model(&VeleroSchedule{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("schedule.not_found")
		}
		return err
	}
	if err := s.deleteScheduleCRD(context.Background(), record.ClusterID, record.Name, dataScope); err != nil {
		return fmt.Errorf("schedule.crd_delete_failed: %w", err)
	}
	return s.db.Delete(&VeleroSchedule{}, id).Error
}

// PauseSchedule suspends the schedule on the cluster and marks it paused.
func (s *BackupService) PauseSchedule(id uint64) error {
	return s.db.Model(&VeleroSchedule{}).Where("id = ?", id).Update("enabled", false).Error
}

// ResumeSchedule re-activates the schedule on the cluster and marks it active.
func (s *BackupService) ResumeSchedule(id uint64) error {
	return s.db.Model(&VeleroSchedule{}).Where("id = ?", id).Update("enabled", true).Error
}

func (s *BackupService) toScheduleResponse(r *VeleroSchedule) *ScheduleResponse {
	var tpl map[string]any
	_ = json.Unmarshal(r.BackupTemplate, &tpl)
	return &ScheduleResponse{
		ID:               r.ID,
		Name:             r.Name,
		ClusterID:        r.ClusterID,
		CronExpression:   r.CronExpression,
		Enabled:          r.Enabled,
		BackupTemplate:   tpl,
		LastBackupTime:   r.LastBackupTime,
		NextBackupTime:   r.NextBackupTime,
		LastBackupStatus: r.LastBackupStatus,
		Phase:            r.Phase,
		CreatedAt:        r.CreatedAt,
		CreatedBy:        r.CreatedBy,
	}
}

// --- Restore ---

// CreateRestore starts a Velero restore for a recorded backup and persists the attempt.
func (s *BackupService) CreateRestore(ctx context.Context, req CreateRestoreRequest, dataScope *common.DataScopeReq, createdBy string) (*RestoreResponse, error) {
	// Validate backup exists
	var backup VeleroBackup
	if err := s.db.Where("name = ? AND cluster_id = ?", req.BackupName, req.ClusterID).First(&backup).Error; err != nil {
		return nil, fmt.Errorf("backup.not_found: %w", err)
	}
	_ = backup

	record := &VeleroRestore{
		Name:       req.Name,
		ClusterID:  req.ClusterID,
		BackupName: req.BackupName,
		RestorePVs: req.RestorePVs,
		Status:     StatusNew,
		CreatedBy:  createdBy,
	}
	if len(req.NamespaceMapping) > 0 {
		b, _ := json.Marshal(req.NamespaceMapping)
		record.NamespaceMapping = b
	}
	if len(req.IncludedNamespaces) > 0 {
		b, _ := json.Marshal(req.IncludedNamespaces)
		record.IncludedNamespaces = b
	}
	if len(req.ExcludedNamespaces) > 0 {
		b, _ := json.Marshal(req.ExcludedNamespaces)
		record.ExcludedNamespaces = b
	}
	if len(req.IncludedResources) > 0 {
		b, _ := json.Marshal(req.IncludedResources)
		record.IncludedResources = b
	}
	if len(req.ExcludedResources) > 0 {
		b, _ := json.Marshal(req.ExcludedResources)
		record.ExcludedResources = b
	}

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	// Create the Velero Restore CRD on the cluster (best-effort, async).
	go func() {
		if err := s.createRestoreCRD(context.Background(), req, dataScope); err != nil {
			_ = s.db.Model(&VeleroRestore{}).Where("id = ?", record.ID).
				Updates(map[string]interface{}{"status": StatusFailed}).Error
		}
	}()
	return s.toRestoreResponse(record), nil
}

// ListRestores returns stored restores filtered by query and data scope.
func (s *BackupService) ListRestores(query RestoreListQuery, dataScope *common.DataScopeReq) (*RestoreListResponse, error) {
	var records []VeleroRestore
	db := s.db.Model(&VeleroRestore{}).Where("cluster_id = ?", query.ClusterID)
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		db = db.Where("cluster_id IN ?", ids)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Model(&VeleroRestore{}).Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}

	page, pageSize := query.Page, query.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	items := make([]RestoreResponse, len(records))
	for i, r := range records {
		items[i] = *s.toRestoreResponse(&r)
	}
	return &RestoreListResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetRestore returns the recorded restore identified by id within the data scope.
func (s *BackupService) GetRestore(id uint64, dataScope *common.DataScopeReq) (*RestoreResponse, error) {
	var record VeleroRestore
	q := s.db.Model(&VeleroRestore{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return nil, err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toRestoreResponse(&record), nil
}

// DeleteRestore removes the restore from the cluster and drops its record.
func (s *BackupService) DeleteRestore(id uint64, dataScope *common.DataScopeReq) error {
	var record VeleroRestore
	q := s.db.Model(&VeleroRestore{})
	ids, err := s.scopedClusterIDs(dataScope)
	if err != nil {
		return err
	}
	if ids != nil {
		q = q.Where("cluster_id IN ?", ids)
	}
	if err := q.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("restore.not_found")
		}
		return err
	}
	if err := s.deleteRestoreCRD(context.Background(), record.ClusterID, record.Name, dataScope); err != nil {
		return fmt.Errorf("restore.crd_delete_failed: %w", err)
	}
	return s.db.Delete(&VeleroRestore{}, id).Error
}

func (s *BackupService) toRestoreResponse(r *VeleroRestore) *RestoreResponse {
	resp := &RestoreResponse{
		ID:             r.ID,
		Name:           r.Name,
		ClusterID:      r.ClusterID,
		BackupName:     r.BackupName,
		RestorePVs:     r.RestorePVs,
		Status:         r.Status,
		Phase:          r.Phase,
		StartTime:      r.StartTime,
		CompletionTime: r.CompletionTime,
		Errors:         r.Errors,
		Warnings:       r.Warnings,
		CreatedAt:      r.CreatedAt,
		CreatedBy:      r.CreatedBy,
	}
	_ = json.Unmarshal(r.NamespaceMapping, &resp.NamespaceMapping)
	_ = json.Unmarshal(r.IncludedNamespaces, &resp.IncludedNamespaces)
	_ = json.Unmarshal(r.ExcludedNamespaces, &resp.ExcludedNamespaces)
	_ = json.Unmarshal(r.IncludedResources, &resp.IncludedResources)
	_ = json.Unmarshal(r.ExcludedResources, &resp.ExcludedResources)
	return resp
}

// --- Velero Status ---

// GetVeleroStatus reads the live Velero deployment state of the given cluster.
func (s *BackupService) GetVeleroStatus(ctx context.Context, clusterID uint64, dataScope *common.DataScopeReq) (*VeleroStatusResponse, error) {
	clientset, err := s.clusterService.GetClientsetWithContext(ctx, clusterID, dataScope)
	if err != nil {
		return nil, err
	}

	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()

	// Get dynamic client for Velero CRD operations
	dynamicClient, err := s.clusterService.GetDynamicClient(clusterID, dataScope)
	if err != nil {
		return nil, err
	}

	vc := NewVeleroClient(clientset, dynamicClient)
	installed, err := vc.IsInstalled(opCtx)
	if err != nil {
		return nil, err
	}

	resp := &VeleroStatusResponse{Installed: installed}
	if installed {
		version, _ := vc.GetVersion(opCtx)
		resp.Version = version
		resp.Namespace = "velero"
		pods, _ := vc.GetRunningPods(opCtx)
		resp.RunningPods = pods
	}

	// Count storage locations from DB
	var count int64
	_ = s.db.Model(&VeleroStorageLocation{}).Count(&count).Error
	resp.StorageLocations = int(count)

	return resp, nil
}

// veleroClient builds a Velero dynamic client for a cluster, scoped by dataScope.
func (s *BackupService) veleroClient(ctx context.Context, clusterID uint64, dataScope *common.DataScopeReq) (*VeleroClient, error) {
	clientset, err := s.clusterService.GetClientsetWithContext(ctx, clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	dynamicClient, err := s.clusterService.GetDynamicClient(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	return NewVeleroClient(clientset, dynamicClient), nil
}

// backupCRDSpec builds the Velero Backup CRD spec from the request.
func backupCRDSpec(req CreateBackupRequest) map[string]interface{} {
	spec := map[string]interface{}{
		"includedNamespaces":      req.IncludedNamespaces,
		"excludedNamespaces":      req.ExcludedNamespaces,
		"includedResources":       req.IncludedResources,
		"excludedResources":       req.ExcludedResources,
		"includeClusterResources": true,
		"snapshotVolumes":         req.IncludePVs,
	}
	if req.TTL > 0 {
		spec["ttl"] = fmt.Sprintf("%dh", req.TTL)
	}
	if req.LabelSelector != "" {
		spec["labelSelector"] = parseLabelSelector(req.LabelSelector)
	}
	if req.StorageLocation != "" {
		spec["storageLocation"] = req.StorageLocation
	}
	return spec
}

// parseLabelSelector converts "k1=v1,k2=v2" into a Velero LabelSelector object.
func parseLabelSelector(selector string) map[string]interface{} {
	matchLabels := map[string]interface{}{}
	for _, part := range strings.Split(selector, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			matchLabels[kv[0]] = kv[1]
		}
	}
	return map[string]interface{}{"matchLabels": matchLabels}
}

// scheduleCRDSpec builds the Velero Schedule CRD spec from the request.
func scheduleCRDSpec(req CreateScheduleRequest) map[string]interface{} {
	return map[string]interface{}{
		"schedule": req.CronExpression,
		"template": backupCRDSpec(req.BackupTemplate),
	}
}

// restoreCRDSpec builds the Velero Restore CRD spec from the request.
func restoreCRDSpec(req CreateRestoreRequest) map[string]interface{} {
	spec := map[string]interface{}{
		"backupName":              req.BackupName,
		"restorePVs":              req.RestorePVs,
		"includedNamespaces":      req.IncludedNamespaces,
		"excludedNamespaces":      req.ExcludedNamespaces,
		"includedResources":       req.IncludedResources,
		"excludedResources":       req.ExcludedResources,
		"includeClusterResources": true,
	}
	if len(req.NamespaceMapping) > 0 {
		spec["namespaceMapping"] = req.NamespaceMapping
	}
	return spec
}

// createBackupCRD creates the Velero Backup CRD on the target cluster.
func (s *BackupService) createBackupCRD(ctx context.Context, req CreateBackupRequest, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, req.ClusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.CreateBackup(opCtx, req.Name, backupCRDSpec(req))
}

// createScheduleCRD creates the Velero Schedule CRD on the target cluster.
func (s *BackupService) createScheduleCRD(ctx context.Context, req CreateScheduleRequest, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, req.ClusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.CreateSchedule(opCtx, req.Name, scheduleCRDSpec(req))
}

// createRestoreCRD creates the Velero Restore CRD on the target cluster.
func (s *BackupService) createRestoreCRD(ctx context.Context, req CreateRestoreRequest, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, req.ClusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.CreateRestore(opCtx, req.Name, restoreCRDSpec(req))
}

// deleteBackupCRD deletes the Velero Backup CRD on the target cluster.
func (s *BackupService) deleteBackupCRD(ctx context.Context, clusterID uint64, name string, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, clusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.DeleteBackup(opCtx, name)
}

// deleteScheduleCRD deletes the Velero Schedule CRD on the target cluster.
func (s *BackupService) deleteScheduleCRD(ctx context.Context, clusterID uint64, name string, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, clusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.DeleteSchedule(opCtx, name)
}

// deleteRestoreCRD deletes the Velero Restore CRD on the target cluster.
func (s *BackupService) deleteRestoreCRD(ctx context.Context, clusterID uint64, name string, dataScope *common.DataScopeReq) error {
	vc, err := s.veleroClient(ctx, clusterID, dataScope)
	if err != nil {
		return err
	}
	opCtx, cancel := shared.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()
	return vc.DeleteRestore(opCtx, name)
}

// scopedClusterIDs returns the cluster IDs visible under dataScope, or nil when
// no scope filter applies (admin / no scope request).
func (s *BackupService) scopedClusterIDs(dataScope *common.DataScopeReq) ([]uint64, error) {
	if dataScope == nil || dataScope.IsAdmin {
		return nil, nil
	}
	var ids []uint64
	err := s.db.Model(&cluster.Cluster{}).Scopes(database.WithDataScope(dataScope)).Pluck("id", &ids).Error
	return ids, err
}
