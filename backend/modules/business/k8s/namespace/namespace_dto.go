package namespace

type NamespaceItem struct {
	Name              string            `json:"name"`
	Status            string            `json:"status"`
	Labels            map[string]string `json:"labels"`
	CreationTimestamp string            `json:"creationTimestamp"`
}

type NamespaceListResponse struct {
	Items []NamespaceItem `json:"items"`
	Total int             `json:"total"`
}

type CreateNamespaceRequest struct {
	Name   string            `json:"name" binding:"required"`
	Labels map[string]string `json:"labels"`
}
