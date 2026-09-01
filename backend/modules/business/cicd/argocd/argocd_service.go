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
	// Encrypt auth token before storing
	encryptedToken, err := EncryptSecret(req.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("encryption_failed: %w", err)
	}

	record := &ArgoCDServer{
		Name:      req.Name,
		ClusterID: req.ClusterID,
		Endpoint:  req.Endpoint,
		AuthToken: encryptedToken,
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
		encryptedToken, err := EncryptSecret(req.AuthToken)
		if err != nil {
			return fmt.Errorf("encryption_failed: %w", err)
		}
		updates["auth_token"] = encryptedToken
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

	// Decrypt auth token
	authToken, err := DecryptSecret(record.AuthToken)
	if err != nil {
		return false, fmt.Errorf("decryption_failed: %w", err)
	}

	// Create ArgoCD client and test connection
	client := NewArgoCDClient(record.Endpoint, authToken, true)
	opCtx, cancel := database.WithCustomTimeout(nil, 10*time.Second)
	defer cancel()

	if err := client.TestConnection(opCtx); err != nil {
		// Update server status to disconnected
		_ = s.db.Model(&record).Updates(map[string]interface{}{
			"status":       ArgoCDStatusDisconnected,
			"last_checked": time.Now(),
		}).Error
		return false, err
	}

	// Update server status and version
	version, _ := client.GetVersion(opCtx)
	_ = s.db.Model(&record).Updates(map[string]interface{}{
		"status":       ArgoCDStatusConnected,
		"version":      version,
		"installed":    true,
		"last_checked": time.Now(),
	}).Error

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

	// Encrypt sensitive fields
	encryptedPassword, err := EncryptSecret(req.Password)
	if err != nil {
		return nil, fmt.Errorf("encryption_failed: %w", err)
	}
	encryptedSSHKey, err := EncryptSecret(req.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("encryption_failed: %w", err)
	}

	record := &ArgoCDRepository{
		ServerID:  req.ServerID,
		RepoURL:   req.RepoURL,
		Name:      req.Name,
		Branch:    req.Branch,
		Type:      req.Type,
		Username:  req.Username,
		Password:  encryptedPassword,
		SSHKey:    encryptedSSHKey,
		Status:    ArgoCDStatusUnknown,
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
		encrypted, err := EncryptSecret(req.Password)
		if err != nil {
			return fmt.Errorf("encryption_failed: %w", err)
		}
		updates["password"] = encrypted
	}
	if req.SSHKey != "" {
		encrypted, err := EncryptSecret(req.SSHKey)
		if err != nil {
			return fmt.Errorf("encryption_failed: %w", err)
		}
		updates["ssh_key"] = encrypted
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

	authToken, err := DecryptSecret(server.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("decryption_failed: %w", err)
	}

	if req.Project == "" {
		req.Project = "default"
	}
	if req.DestinationServer == "" {
		req.DestinationServer = "https://kubernetes.default.svc"
	}

	// Create the application on the remote ArgoCD server first; the local
	// record is only persisted once the remote create succeeds.
	payload := &CreateApplicationPayload{}
	payload.Metadata.Name = req.Name
	payload.Spec.Project = req.Project
	payload.Spec.Source.RepoURL = req.SourceRepoURL
	payload.Spec.Source.Path = req.SourcePath
	payload.Spec.Source.TargetRevision = req.SourceTargetRev
	payload.Spec.Destination.Server = req.DestinationServer
	payload.Spec.Destination.Namespace = req.DestinationNS

	client := NewArgoCDClient(server.Endpoint, authToken, true)
	opCtx, cancel := database.WithCustomTimeout(nil, 10*time.Second)
	defer cancel()
	if err := client.CreateApplication(opCtx, payload); err != nil {
		return nil, fmt.Errorf("argocd.create_failed: %w", err)
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
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("app.not_found")
		}
		return err
	}

	var server ArgoCDServer
	if err := s.db.First(&server, record.ServerID).Error; err != nil {
		return fmt.Errorf("server.not_found: %w", err)
	}
	authToken, err := DecryptSecret(server.AuthToken)
	if err != nil {
		return fmt.Errorf("decryption_failed: %w", err)
	}

	// Delete on the remote server first so local and remote state stay
	// consistent; the local record is only removed after the remote delete
	// succeeds.
	client := NewArgoCDClient(server.Endpoint, authToken, true)
	opCtx, cancel := database.WithCustomTimeout(nil, 10*time.Second)
	defer cancel()
	if err := client.DeleteApplication(opCtx, record.Name, false); err != nil {
		return fmt.Errorf("argocd.delete_failed: %w", err)
	}
	return s.db.Delete(&ArgoCDApplication{}, id).Error
}

// SyncApplication triggers a real sync on the remote ArgoCD server and
// records the outcome in the sync history.
func (s *ArgoCDService) SyncApplication(id uint64, revision string) (*SyncHistoryResponse, error) {
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		return nil, err
	}

	var server ArgoCDServer
	if err := s.db.First(&server, record.ServerID).Error; err != nil {
		return nil, fmt.Errorf("server.not_found: %w", err)
	}
	authToken, err := DecryptSecret(server.AuthToken)
	if err != nil {
		return nil, fmt.Errorf("decryption_failed: %w", err)
	}

	now := time.Now()
	history := &ArgoCDSyncHistory{
		ApplicationID: id,
		Revision:      revision,
		Status:        "Running",
		StartedAt:     now,
	}
	if err := s.db.Create(history).Error; err != nil {
		return nil, err
	}

	client := NewArgoCDClient(server.Endpoint, authToken, true)
	opCtx, cancel := database.WithCustomTimeout(nil, 30*time.Second)
	defer cancel()

	finished := time.Now()
	status := "Succeeded"
	var message string
	if err := client.SyncApplication(opCtx, record.Name, revision); err != nil {
		status = "Failed"
		message = err.Error()
	}
	history.Status = status
	history.Message = message
	history.FinishedAt = &finished
	_ = s.db.Model(history).Updates(map[string]interface{}{
		"status": status, "message": message, "finished_at": finished,
	}).Error

	updates := map[string]interface{}{"last_synced_at": finished}
	if status == "Succeeded" {
		updates["status"] = AppStatusSynced
		updates["sync_status"] = "Synced"
	} else {
		updates["status"] = AppStatusError
		updates["sync_status"] = "OutOfSync"
		updates["last_error"] = message
	}
	_ = s.db.Model(&record).Updates(updates).Error

	if err != nil {
		return nil, err
	}
	return &SyncHistoryResponse{
		ID:         history.ID,
		Revision:   history.Revision,
		Status:     history.Status,
		Message:    history.Message,
		StartedAt:  history.StartedAt,
		FinishedAt: history.FinishedAt,
	}, nil
}

// RefreshApplication updates the app status from the remote.
func (s *ArgoCDService) RefreshApplication(id uint64) error {
	var record ArgoCDApplication
	if err := s.db.First(&record, id).Error; err != nil {
		return err
	}

	// Get server info
	var server ArgoCDServer
	if err := s.db.First(&server, record.ServerID).Error; err != nil {
		return fmt.Errorf("server.not_found: %w", err)
	}

	// Decrypt auth token
	authToken, err := DecryptSecret(server.AuthToken)
	if err != nil {
		return fmt.Errorf("decryption_failed: %w", err)
	}

	// Create client and refresh
	client := NewArgoCDClient(server.Endpoint, authToken, true)
	opCtx, cancel := database.WithCustomTimeout(nil, 10*time.Second)
	defer cancel()

	if err := client.RefreshApplication(opCtx, record.Name); err != nil {
		return err
	}

	// Get updated status
	status, err := client.GetApplicationStatus(opCtx, record.Name)
	if err != nil {
		return err
	}

	// Update database
	updates := map[string]interface{}{
		"health_status": status.Health.Status,
		"sync_status":   status.Sync.Status,
	}

	if status.OperationState != nil {
		updates["status"] = status.OperationState.Phase
		if status.OperationState.Message != "" {
			updates["last_error"] = status.OperationState.Message
		}
	}

	return s.db.Model(&record).Updates(updates).Error
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
