package namespace

// NamespaceItem summarizes a Kubernetes namespace.
//
//nolint:revive // DTO names retain the namespace domain prefix for generated API clarity.
type NamespaceItem struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Labels            map[string]string `json:"labels"`
	CreationTimestamp string            `json:"creationTimestamp"`
}

// NamespaceListResponse contains Kubernetes namespace summaries.
//
//nolint:revive // DTO names retain the namespace domain prefix for generated API clarity.
type NamespaceListResponse struct {
	Items []NamespaceItem `json:"items"`
	Total int             `json:"total"`
}

// CreateNamespaceRequest contains namespace creation fields.
type CreateNamespaceRequest struct {
	Name            string            `json:"name" binding:"required"`
	Labels          map[string]string `json:"labels"`
	BusinessScopeID uint64            `json:"businessScopeId" binding:"required"`
	Environment     string            `json:"environment" binding:"required"`
	AllowedActions  []string          `json:"allowedActions"`
	ResourceVersion string            `json:"resourceVersion"`
}
