package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/namespace"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const backupOpTimeout = 30 * time.Second

// BackupService manages Velero backup resources through DB + K8s API.
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

func (s *BackupService) CreateStorageLocation(req CreateStorageLocationRequest, createdBy string) (*StorageLocationResponse, error) {
	record := &VeleroStorageLocation{
		Name:           req.Name,
		Provider:       req.Provider,
		Bucket:         req.Bucket,
		Region:         req.Region,
		Endpoint:       req.Endpoint,
		AccessKeyID:    req.AccessKeyID,
		SecretAccessKey: req.SecretAccessKey,
		IsDefault:      req.IsDefault,
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

func (s *BackupService) GetStorageLocation(id uint64) (*StorageLocationResponse, error) {
	var record VeleroStorageLocation
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toStorageLocationResponse(&record), nil
}

func (s *BackupService) UpdateStorageLocation(id uint64, req CreateStorageLocationRequest) error {
	updates := map[string]interface{}{
		"name":     req.Name,
		"provider": req.Provider,
		"bucket":   req.Bucket,
		"region":   req.Region,
		"endpoint": req.Endpoint,
	}
	if req.AccessKeyID != "" {
		updates["access_key_id"] = req.AccessKeyID
	}
	if req.SecretAccessKey != "" {
		updates["secret_access_key"] = req.SecretAccessKey
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

func (s *BackupService) CreateBackup(ctx context.Context, req CreateBackupRequest, dataScope *common.DataScopeReq, createdBy string) (*BackupResponse, error) {
	// Validate cluster exists
	if _, err := s.clusterService.GetClientsetWithContext(ctx, req.ClusterID, dataScope); err != nil {
		return nil, fmt.Errorf("cluster.not_found: %w", err)
	}

	record := &VeleroBackup{
		Name:              req.Name,
		ClusterID:         req.ClusterID,
		BackupType:        req.BackupType,
		IncludePVs:        req.IncludePVs,
		TTL:               req.TTL,
		LabelSelector:     req.LabelSelector,
		StorageLocation:   req.StorageLocation,
		Status:            StatusNew,
		CreatedBy:         createdBy,
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

	// Create Velero Backup CRD via dynamic client (best-effort)
	go s.tryCreateVeleroBackupCRD(req)

	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toBackupResponse(record), nil
}

func (s *BackupService) ListBackups(query BackupListQuery, dataScope *common.DataScopeReq) (*BackupListResponse, error) {
	var records []VeleroBackup
	db := s.db.Where("cluster_id = ?", query.ClusterID)
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
	if err := db.Model(&VeleroBackup{}).Count(&total).Error; err != nil {
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

func (s *BackupService) GetBackup(id uint64) (*BackupResponse, error) {
	var record VeleroBackup
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toBackupResponse(&record), nil
}

func (s *BackupService) DeleteBackup(id uint64) error {
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

func (s *BackupService) CreateSchedule(ctx context.Context, req CreateScheduleRequest, dataScope *common.DataScopeReq, createdBy string) (*ScheduleResponse, error) {
	if _, err := s.clusterService.GetClientsetWithContext(ctx, req.ClusterID, dataScope); err != nil {
		return nil, fmt.Errorf("cluster.not_found: %w", err)
	}

	tplJSON, _ := json.Marshal(req.BackupTemplate)
	record := &VeleroSchedule{
		Name:             req.Name,
		ClusterID:        req.ClusterID,
		CronExpression:   req.CronExpression,
		Enabled:          req.Enabled,
		BackupTemplate:   datatypes.JSON(tplJSON),
		CreatedBy:        createdBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toScheduleResponse(record), nil
}

func (s *BackupService) ListSchedules(query ScheduleListQuery, dataScope *common.DataScopeReq) (*ScheduleListResponse, error) {
	var records []VeleroSchedule
	db := s.db.Where("cluster_id = ?", query.ClusterID)

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

func (s *BackupService) GetSchedule(id uint64) (*ScheduleResponse, error) {
	var record VeleroSchedule
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toScheduleResponse(&record), nil
}

func (s *BackupService) UpdateSchedule(id uint64, req CreateScheduleRequest) error {
	tplJSON, _ := json.Marshal(req.BackupTemplate)
	return s.db.Model(&VeleroSchedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":             req.Name,
		"cron_expression":  req.CronExpression,
		"enabled":          req.Enabled,
		"backup_template":  tplJSON,
	}).Error
}

func (s *BackupService) DeleteSchedule(id uint64) error {
	return s.db.Delete(&VeleroSchedule{}, id).Error
}

func (s *BackupService) PauseSchedule(id uint64) error {
	return s.db.Model(&VeleroSchedule{}).Where("id = ?", id).Update("enabled", false).Error
}

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

func (s *BackupService) CreateRestore(ctx context.Context, req CreateRestoreRequest, dataScope *common.DataScopeReq, createdBy string) (*RestoreResponse, error) {
	// Validate backup exists
	var backup VeleroBackup
	if err := s.db.Where("name = ? AND cluster_id = ?", req.BackupName, req.ClusterID).First(&backup).Error; err != nil {
		return nil, fmt.Errorf("backup.not_found: %w", err)
	}
	_ = backup

	record := &VeleroRestore{
		Name:         req.Name,
		ClusterID:    req.ClusterID,
		BackupName:   req.BackupName,
		RestorePVs:   req.RestorePVs,
		Status:       StatusNew,
		CreatedBy:    createdBy,
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
	return s.toRestoreResponse(record), nil
}

func (s *BackupService) ListRestores(query RestoreListQuery, dataScope *common.DataScopeReq) (*RestoreListResponse, error) {
	var records []VeleroRestore
	db := s.db.Where("cluster_id = ?", query.ClusterID)
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

func (s *BackupService) GetRestore(id uint64) (*RestoreResponse, error) {
	var record VeleroRestore
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toRestoreResponse(&record), nil
}

func (s *BackupService) DeleteRestore(id uint64) error {
	return s.db.Delete(&VeleroRestore{}, id).Error
}

func (s *BackupService) toRestoreResponse(r *VeleroRestore) *RestoreResponse {
	resp := &RestoreResponse{
		ID:               r.ID,
		Name:             r.Name,
		ClusterID:        r.ClusterID,
		BackupName:       r.BackupName,
		RestorePVs:       r.RestorePVs,
		Status:           r.Status,
		Phase:            r.Phase,
		StartTime:        r.StartTime,
		CompletionTime:   r.CompletionTime,
		Errors:           r.Errors,
		Warnings:         r.Warnings,
		CreatedAt:        r.CreatedAt,
		CreatedBy:        r.CreatedBy,
	}
	_ = json.Unmarshal(r.NamespaceMapping, &resp.NamespaceMapping)
	_ = json.Unmarshal(r.IncludedNamespaces, &resp.IncludedNamespaces)
	_ = json.Unmarshal(r.ExcludedNamespaces, &resp.ExcludedNamespaces)
	_ = json.Unmarshal(r.IncludedResources, &resp.IncludedResources)
	_ = json.Unmarshal(r.ExcludedResources, &resp.ExcludedResources)
	return resp
}

// --- Velero Status ---

func (s *BackupService) GetVeleroStatus(ctx context.Context, clusterID uint64, dataScope *common.DataScopeReq) (*VeleroStatusResponse, error) {
	clientset, err := s.clusterService.GetClientsetWithContext(ctx, clusterID, dataScope)
	if err != nil {
		return nil, err
	}

	opCtx, cancel := database.WithCustomTimeout(ctx, backupOpTimeout)
	defer cancel()

	vc := NewVeleroClient(clientset)
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

// tryCreateVeleroBackupCRD attempts to create a Velero Backup CRD via dynamic client.
// This is best-effort; failures are logged but don't block the DB record creation.
func (s *BackupService) tryCreateVeleroBackupCRD(req CreateBackupRequest) {
	// TODO: When cluster service exposes dynamic.Interface, create the actual Velero Backup CRD.
	// For now the DB record is the source of truth.
}
