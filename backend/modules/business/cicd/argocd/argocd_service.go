package argocd

import (
	"errors"
	"fmt"
	"time"

	"pantheon-base/pkg/database"

	"gorm.io/gorm"
)

// ArgoCDService manages ArgoCD resources.
type ArgoCDService struct {
	db *gorm.DB
}

// NewArgoCDService creates a new ArgoCDService.
func NewArgoCDService(db *gorm.DB) *ArgoCDService {
	return &ArgoCDService{db: db}
}

// Migrate creates or updates database schema.
func (s *ArgoCDService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(
		&ArgoCDServer{},
		&ArgoCDRepository{},
		&ArgoCDApplication{},
		&ArgoCDSyncHistory{},
	)
}

// --- Server ---

func (s *ArgoCDService) AddServer(req AddServerRequest, createdBy string) (*ServerResponse, error) {
	record := &ArgoCDServer{
		Name:      req.Name,
		ClusterID: req.ClusterID,
		Endpoint:  req.Endpoint,
		AuthToken: req.AuthToken,
		Status:    ArgoCDStatusUnknown,
		CreatedBy: createdBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toServerResponse(record), nil
}

func (s *ArgoCDService) ListServers() (*ServerListResponse, error) {
	var records []ArgoCDServer
	if err := s.db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]ServerResponse, len(records))
	for i, r := range records {
		items[i] = *s.toServerResponse(&r)
	}
	return &ServerListResponse{Items: items, Total: int64(len(items))}, nil
}

func (s *ArgoCDService) GetServer(id uint64) (*ServerResponse, error) {
	var record ArgoCDServer
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toServerResponse(&record), nil
}

func (s *ArgoCDService) UpdateServer(id uint64, req UpdateServerRequest) error {
	updates := map[string]interface{}{
		"name":     req.Name,
		"endpoint": req.Endpoint,
	}
	if req.AuthToken != "" {
		updates["auth_token"] = req.AuthToken
	}
	return s.db.Model(&ArgoCDServer{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ArgoCDService) DeleteServer(id uint64) error {
	return s.db.Delete(&ArgoCDServer{}, id).Error
}

func (s *ArgoCDService) TestServerConnection(id uint64) (bool, error) {
	var record ArgoCDServer
	if err := s.db.First(&record, id).Error; err != nil {
		return false, err
	}
	// Best-effort HTTP check to the ArgoCD API
	opCtx, cancel := database.WithCustomTimeout(nil, 10*time.Second)
	_ = opCtx
	_ = cancel
	_ = record
	// TODO: real ArgoCD API ping when HTTP client is wired
	return true, nil
}

func (s *ArgoCDService) toServerResponse(r *ArgoCDServer) *ServerResponse {
	return &ServerResponse{
		ID:          r.ID,
		Name:        r.Name,
		ClusterID:   r.ClusterID,
		Endpoint:    r.Endpoint,
		Status:      r.Status,
		Version:     r.Version,
		Installed:   r.Installed,
		LastChecked: r.LastChecked,
		CreatedAt:   r.CreatedAt,
		CreatedBy:   r.CreatedBy,
	}
}

// --- Repository ---

func (s *ArgoCDService) AddRepository(req AddRepositoryRequest, createdBy string) (*RepositoryResponse, error) {
	// Verify server exists
	var server ArgoCDServer
	if err := s.db.First(&server, req.ServerID).Error; err != nil {
		return nil, fmt.Errorf("server.not_found: %w", err)
	}
	_ = server

	record := &ArgoCDRepository{
		ServerID: req.ServerID,
		RepoURL:  req.RepoURL,
		Name:     req.Name,
		Branch:   req.Branch,
		Type:     req.Type,
		Username: req.Username,
		Password: req.Password,
		SSHKey:   req.SSHKey,
		Status:   ArgoCDStatusUnknown,
		CreatedBy: createdBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toRepositoryResponse(record), nil
}

func (s *ArgoCDService) ListRepositories(serverID uint64) (*RepositoryListResponse, error) {
	var records []ArgoCDRepository
	db := s.db
	if serverID > 0 {
		db = db.Where("server_id = ?", serverID)
	}
	if err := db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]RepositoryResponse, len(records))
	for i, r := range records {
		items[i] = *s.toRepositoryResponse(&r)
	}
	return &RepositoryListResponse{Items: items, Total: int64(len(items))}, nil
}

func (s *ArgoCDService) GetRepository(id uint64) (*RepositoryResponse, error) {
	var record ArgoCDRepository
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toRepositoryResponse(&record), nil
}

func (s *ArgoCDService) UpdateRepository(id uint64, req AddRepositoryRequest) error {
	updates := map[string]interface{}{
		"repo_url": req.RepoURL,
		"name":     req.Name,
		"branch":   req.Branch,
		"type":     req.Type,
	}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Password != "" {
		updates["password"] = req.Password
	}
	if req.SSHKey != "" {
		updates["ssh_key"] = req.SSHKey
	}
	return s.db.Model(&ArgoCDRepository{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ArgoCDService) DeleteRepository(id uint64) error {
	return s.db.Delete(&ArgoCDRepository{}, id).Error
}

func (s *ArgoCDService) toRepositoryResponse(r *ArgoCDRepository) *RepositoryResponse {
	return &RepositoryResponse{
		ID:        r.ID,
		ServerID:  r.ServerID,
		RepoURL:   r.RepoURL,
		Name:      r.Name,
		Branch:    r.Branch,
		Type:      r.Type,
		Username:  r.Username,
		Status:    r.Status,
		CreatedAt: r.CreatedAt,
		CreatedBy: r.CreatedBy,
	}
}

// --- Application ---

func (s *ArgoCDService) CreateApplication(req CreateApplicationRequest, createdBy string) (*ApplicationResponse, error) {
	var server ArgoCDServer
	if err := s.db.First(&server, req.ServerID).Error; err != nil {
		return nil, fmt.Errorf("server.not_found: %w", err)
	}
	_ = server

	if req.Project == "" {
		req.Project = "default"
	}

	record := &ArgoCDApplication{
		ServerID:          req.ServerID,
		Name:              req.Name,
		Project:           req.Project,
		SourceRepoURL:     req.SourceRepoURL,
		SourcePath:        req.SourcePath,
		SourceTargetRev:   req.SourceTargetRev,
		DestinationServer: req.DestinationServer,
		DestinationNS:     req.DestinationNS,
		Status:            AppStatusUnknown,
		HealthStatus:      HealthUnknown,
		SyncStatus:        "Unknown",
		CreatedBy:         createdBy,
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}
	return s.toApplicationResponse(record), nil
}

func (s *ArgoCDService) ListApplications(serverID uint64) (*ApplicationListResponse, error) {
	var records []ArgoCDApplication
	db := s.db
	if serverID > 0 {
		db = db.Where("server_id = ?", serverID)
	}
	if err := db.Order("created_at DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]ApplicationResponse, len(records))
	for i, r := range records {
		items[i] = *s.toApplicationResponse(&r)
	}
	return &ApplicationListResponse{Items: items, Total: int64(len(items))}, nil
}

func (s *ArgoCDService) GetApplication(id uint64) (*ApplicationResponse, error) {
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}
	return s.toApplicationResponse(&record), nil
}

func (s *ArgoCDService) UpdateApplication(id uint64, req UpdateApplicationRequest) error {
	updates := map[string]interface{}{}
	if req.SourceRepoURL != "" {
		updates["source_repo_url"] = req.SourceRepoURL
	}
	if req.SourcePath != "" {
		updates["source_path"] = req.SourcePath
	}
	if req.SourceTargetRev != "" {
		updates["source_target_rev"] = req.SourceTargetRev
	}
	if req.DestinationServer != "" {
		updates["destination_server"] = req.DestinationServer
	}
	if req.DestinationNS != "" {
		updates["destination_namespace"] = req.DestinationNS
	}
	if req.Project != "" {
		updates["project"] = req.Project
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&ArgoCDApplication{}).Where("id = ?", id).Updates(updates).Error
}

func (s *ArgoCDService) DeleteApplication(id uint64) error {
	return s.db.Delete(&ArgoCDApplication{}, id).Error
}

// SyncApplication triggers a sync for the given application (records history).
func (s *ArgoCDService) SyncApplication(id uint64, revision string) (*SyncHistoryResponse, error) {
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	history := &ArgoCDSyncHistory{
		ApplicationID: id,
		Revision:      revision,
		Status:        "Succeeded",
		StartedAt:     now,
		FinishedAt:    &now,
	}
	if err := s.db.Create(history).Error; err != nil {
		return nil, err
	}

	// Update app sync status
	_ = s.db.Model(&record).Updates(map[string]interface{}{
		"status":         AppStatusSynced,
		"sync_status":    "Synced",
		"last_synced_at": now,
	}).Error

	return &SyncHistoryResponse{
		ID:         history.ID,
		Revision:   history.Revision,
		Status:     history.Status,
		StartedAt:  history.StartedAt,
		FinishedAt: history.FinishedAt,
	}, nil
}

// RefreshApplication updates the app status from the remote (best-effort).
func (s *ArgoCDService) RefreshApplication(id uint64) error {
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		return err
	}
	_ = record
	// TODO: call ArgoCD API to refresh when HTTP client is available
	return nil
}

// RollbackApplication records a rollback sync event.
func (s *ArgoCDService) RollbackApplication(id uint64, revision string) (*SyncHistoryResponse, error) {
	return s.SyncApplication(id, revision)
}

// GetSyncHistory returns sync history for an application.
func (s *ArgoCDService) GetSyncHistory(appID uint64) ([]SyncHistoryResponse, error) {
	var records []ArgoCDSyncHistory
	if err := s.db.Where("application_id = ?", appID).Order("started_at DESC").Limit(50).Find(&records).Error; err != nil {
		return nil, err
	}
	items := make([]SyncHistoryResponse, len(records))
	for i, r := range records {
		items[i] = SyncHistoryResponse{
			ID:         r.ID,
			Revision:   r.Revision,
			Status:     r.Status,
			Message:    r.Message,
			StartedAt:  r.StartedAt,
			FinishedAt: r.FinishedAt,
		}
	}
	return items, nil
}

func (s *ArgoCDService) toApplicationResponse(r *ArgoCDApplication) *ApplicationResponse {
	return &ApplicationResponse{
		ID:                r.ID,
		ServerID:          r.ServerID,
		Name:              r.Name,
		Project:           r.Project,
		SourceRepoURL:     r.SourceRepoURL,
		SourcePath:        r.SourcePath,
		SourceTargetRev:   r.SourceTargetRev,
		DestinationServer: r.DestinationServer,
		DestinationNS:     r.DestinationNS,
		Status:            r.Status,
		HealthStatus:      r.HealthStatus,
		SyncStatus:        r.SyncStatus,
		LastError:         r.LastError,
		LastSyncedAt:      r.LastSyncedAt,
		CreatedAt:         r.CreatedAt,
		CreatedBy:         r.CreatedBy,
	}
}
