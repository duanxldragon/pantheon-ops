package workload

// WorkloadListQuery filters Kubernetes workloads for a cluster.
//
//nolint:revive // retained as the public workload query contract.
type WorkloadListQuery struct {
	ClusterID     uint64 `form:"clusterId" json:"clusterId" binding:"required"`
	Namespace     string `form:"namespace" json:"namespace"`
	Kind          string `form:"kind" json:"kind"`
	Limit         int64  `form:"limit" json:"limit"`
	ContinueToken string `form:"continue" json:"continue"`
}

// WorkloadItem summarizes a Kubernetes workload.
//
//nolint:revive // retained as the public workload item contract.
type WorkloadItem struct {
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	Replicas        int32    `json:"replicas"`
	ReadyReplicas   int32    `json:"readyReplicas"`
	Images          []string `json:"images"`
	Status          string   `json:"status"`
	Age             string   `json:"age"`
	ResourceVersion string   `json:"resourceVersion"`
}

// WorkloadListResponse contains Kubernetes workload summaries.
//
//nolint:revive // retained as the public workload response contract.
type WorkloadListResponse struct {
	Items         []WorkloadItem `json:"items"`
	Total         int            `json:"total"`
	ContinueToken string         `json:"continueToken"`
}

// PodItem summarizes a Kubernetes pod.
type PodItem struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	NodeName  string `json:"nodeName"`
	Restarts  int32  `json:"restarts"`
	CreatedAt string `json:"createdAt"`
}

// PodListResponse contains Kubernetes pod summaries.
type PodListResponse struct {
	Items []PodItem `json:"items"`
	Total int       `json:"total"`
}

// ScaleRequest requests a new workload replica count.
type ScaleRequest struct {
	Replicas        int32  `json:"replicas" binding:"required,min=0"`
	ResourceVersion string `json:"resourceVersion"`
}
