package argocd

import "time"

// --- Server ---

// AddServerRequest is the payload that registers an Argo CD server.
type AddServerRequest struct {
	Name      string `json:"name" binding:"required,max=128"`
	ClusterID uint64 `json:"clusterId"`
	Endpoint  string `json:"endpoint" binding:"required,max=256"`
	AuthToken string `json:"authToken" binding:"required"`
	CACert    string `json:"caCert,omitempty" binding:"max=8192"`
}

// UpdateServerRequest carries the editable fields of an Argo CD server.
type UpdateServerRequest struct {
	Name      string `json:"name" binding:"required,max=128"`
	Endpoint  string `json:"endpoint" binding:"required,max=256"`
	AuthToken string `json:"authToken,omitempty"`
	// CACert is tri-state: absent keeps the stored bundle, an empty string clears it back to
	// system trust, a non-empty value replaces it. Certificate verification is never optional.
	CACert *string `json:"caCert,omitempty" binding:"omitempty,max=8192"`
}

// ServerResponse is the Argo CD server shape returned to clients; it never carries the auth token.
type ServerResponse struct {
	ID          uint64     `json:"id"`
	Name        string     `json:"name"`
	ClusterID   uint64     `json:"clusterId"`
	Endpoint    string     `json:"endpoint"`
	HasCACert   bool       `json:"hasCaCert"`
	Status      string     `json:"status"`
	Version     string     `json:"version,omitempty"`
	Installed   bool       `json:"installed"`
	LastChecked *time.Time `json:"lastChecked,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy"`
}

// ServerListResponse is the paged list of Argo CD servers.
type ServerListResponse struct {
	Items []ServerResponse `json:"items"`
	Total int64            `json:"total"`
}

// --- Repository ---

// AddRepositoryRequest is the payload that registers a Git repository in Argo CD.
type AddRepositoryRequest struct {
	ServerID uint64 `json:"serverId" binding:"required"`
	RepoURL  string `json:"repoUrl" binding:"required,max=512"`
	Name     string `json:"name" binding:"max=128"`
	Branch   string `json:"branch" binding:"max=128"`
	Type     string `json:"type" binding:"required,oneof=git helm"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	SSHKey   string `json:"sshKey,omitempty"`
}

// RepositoryResponse is the Argo CD repository shape returned to clients.
type RepositoryResponse struct {
	ID        uint64    `json:"id"`
	ServerID  uint64    `json:"serverId"`
	RepoURL   string    `json:"repoUrl"`
	Name      string    `json:"name"`
	Branch    string    `json:"branch"`
	Type      string    `json:"type"`
	Username  string    `json:"username,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy"`
}

// RepositoryListResponse is the list of repositories known to one Argo CD server.
type RepositoryListResponse struct {
	Items []RepositoryResponse `json:"items"`
	Total int64                `json:"total"`
}

// --- Application ---

// CreateApplicationRequest is the payload that creates an Argo CD application.
type CreateApplicationRequest struct {
	ServerID          uint64 `json:"serverId" binding:"required"`
	Name              string `json:"name" binding:"required,max=128"`
	Project           string `json:"project" binding:"max=128"`
	SourceRepoURL     string `json:"sourceRepoUrl" binding:"required"`
	SourcePath        string `json:"sourcePath" binding:"max=256"`
	SourceTargetRev   string `json:"sourceTargetRev" binding:"max=128"`
	DestinationServer string `json:"destinationServer" binding:"max=256"`
	DestinationNS     string `json:"destinationNamespace" binding:"max=128"`
}

// UpdateApplicationRequest carries the editable fields of an Argo CD application.
type UpdateApplicationRequest struct {
	SourceRepoURL     string `json:"sourceRepoUrl"`
	SourcePath        string `json:"sourcePath"`
	SourceTargetRev   string `json:"sourceTargetRev"`
	DestinationServer string `json:"destinationServer"`
	DestinationNS     string `json:"destinationNamespace"`
	Project           string `json:"project"`
}

// ApplicationResponse is the Argo CD application shape returned to clients.
type ApplicationResponse struct {
	ID                uint64     `json:"id"`
	ServerID          uint64     `json:"serverId"`
	Name              string     `json:"name"`
	Project           string     `json:"project"`
	SourceRepoURL     string     `json:"sourceRepoUrl"`
	SourcePath        string     `json:"sourcePath"`
	SourceTargetRev   string     `json:"sourceTargetRev"`
	DestinationServer string     `json:"destinationServer"`
	DestinationNS     string     `json:"destinationNamespace"`
	Status            string     `json:"status"`
	HealthStatus      string     `json:"healthStatus"`
	SyncStatus        string     `json:"syncStatus"`
	LastError         string     `json:"lastError,omitempty"`
	LastSyncedAt      *time.Time `json:"lastSyncedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	CreatedBy         string     `json:"createdBy"`
}

// ApplicationListResponse is the list of applications deployed on one Argo CD server.
type ApplicationListResponse struct {
	Items []ApplicationResponse `json:"items"`
	Total int64                 `json:"total"`
}

// SyncApplicationRequest selects the revision an application syncs to.
type SyncApplicationRequest struct {
	Revision string `json:"revision,omitempty"`
}

// SyncHistoryResponse lists the recorded sync operations of an application.
type SyncHistoryResponse struct {
	ID         uint64     `json:"id"`
	Revision   string     `json:"revision"`
	Status     string     `json:"status"`
	Message    string     `json:"message,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}
